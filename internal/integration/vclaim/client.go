package vclaim

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/arunika/apm-go/internal/config"
)

// bpjsTransport mengganti nama header yang sudah dinormalisasi oleh Go
// net/http kembali ke bentuk aslinya sebelum dikirim ke wire.
//
// Go canonicalizes HTTP headers (e.g., "X-cons-id" → "X-Cons-Id") per
// RFC 7230, tapi BPJS server melakukan matching case-sensitive pada nama
// header. Tanpa fix ini, server BPJS mengembalikan "Authentication
// parameters missing" karena tidak mengenali X-Cons-Id / User_key.
type bpjsTransport struct {
	inner http.RoundTripper
}

// bpjsHeaderMap memetakan bentuk canonical Go → bentuk exact yang diharapkan BPJS.
var bpjsHeaderMap = map[string]string{
	"X-Cons-Id":   "X-cons-id",
	"X-Timestamp": "X-timestamp",
	"X-Signature": "X-signature",
	"User_key":    "user_key",
}

func (t *bpjsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	for canonical, original := range bpjsHeaderMap {
		if vals, ok := req2.Header[canonical]; ok {
			delete(req2.Header, canonical)
			req2.Header[original] = vals
		}
	}
	return t.inner.RoundTrip(req2)
}

// Client adalah implementasi HTTP nyata untuk VClaimClient.
// Thread-safe: resty.Client thread-safe, dan field lain immutable
// setelah New().
type Client struct {
	consID       string
	secretKey    string
	userKey      string // BPJS user_key header (untuk decrypt AES-256-CBC response)
	baseURL      string
	ppkPelayanan string // kode PPK RS sendiri (cfg.BPJS.PPKPelayanan)
	httpClient   *resty.Client
	now          func() time.Time // injected untuk test
}

// New membangun Client dari config.BPJSConfig (P-003).
//
// Default behavior:
//   - timeout per-request: 15 detik (jika cfg tidak menyediakan,
//     pakai default safe untuk environment BPJS)
//   - retry: 2x untuk 5xx dan network error, backoff 500ms..2s
//   - User-Agent: "APM-TARA/1.0 (Go vclaim client)"
func New(cfg config.BPJSConfig) *Client {
	baseURL := strings.TrimRight(cfg.VClaimURL, "/")

	hc := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(15*time.Second).
		SetHeader("User-Agent", "APM-TARA/1.0 (Go vclaim client)").
		SetHeader("Content-Type", "application/json").
		SetTransport(&bpjsTransport{inner: http.DefaultTransport}).
		SetRetryCount(2).
		SetRetryWaitTime(500*time.Millisecond).
		SetRetryMaxWaitTime(2*time.Second).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			// Retry pada network error atau 5xx — jangan retry 4xx
			// (validation/auth error tidak akan berubah dengan retry).
			if err != nil {
				return true
			}
			return r.StatusCode() >= http.StatusInternalServerError
		})

	return &Client{
		consID:       cfg.ConsID,
		secretKey:    cfg.ConsumerSecret,
		userKey:      cfg.UserKey,
		baseURL:      baseURL,
		ppkPelayanan: cfg.PPKPelayanan,
		httpClient:   hc,
		now:          time.Now,
	}
}

// SetBaseURL mengganti baseURL — dipakai test untuk arahkan ke httptest server.
func (c *Client) SetBaseURL(u string) {
	c.baseURL = strings.TrimRight(u, "/")
	c.httpClient.SetBaseURL(c.baseURL)
}

// SetTimeout mengganti timeout per-request.
func (c *Client) SetTimeout(d time.Duration) {
	c.httpClient.SetTimeout(d)
}

