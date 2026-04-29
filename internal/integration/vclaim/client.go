package vclaim

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/arunika/apm-go/internal/config"
)

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

