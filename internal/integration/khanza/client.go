package khanza

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"

	"github.com/arunika/apm-go/internal/config"
)

// Client adalah implementasi HTTP nyata KhanzaClient.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *resty.Client
	logger     *slog.Logger
}

var _ KhanzaClient = (*Client)(nil)

// New membangun Client dari config.ServerConfig (P-003 — KhanzaURL +
// KhanzaAPIKey + TimeoutMs + Retry).
func New(cfg config.ServerConfig) *Client {
	timeout := time.Duration(cfg.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	retry := cfg.Retry
	if retry <= 0 {
		retry = 2
	}

	baseURL := strings.TrimRight(cfg.KhanzaURL, "/")

	hc := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(timeout).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("X-API-Key", cfg.KhanzaAPIKey).
		SetHeader("User-Agent", "APM-TARA/1.0 (Go khanza client)").
		SetRetryCount(retry).
		SetRetryWaitTime(500*time.Millisecond).
		SetRetryMaxWaitTime(2*time.Second).
		AddRetryCondition(func(r *resty.Response, err error) bool {
			// Retry hanya untuk network error & 5xx. JANGAN retry 4xx.
			if err != nil {
				return true
			}
			return r.StatusCode() >= http.StatusInternalServerError
		})

	return &Client{
		baseURL:    baseURL,
		apiKey:     cfg.KhanzaAPIKey,
		httpClient: hc,
		logger:     slog.Default(),
	}
}

// SetLogger mengganti logger (dipakai test atau caller dengan PHIMaskingHandler).
func (c *Client) SetLogger(l *slog.Logger) {
	if l != nil {
		c.logger = l
	}
}

// SetBaseURL mengganti baseURL — dipakai test untuk arahkan ke httptest server.
func (c *Client) SetBaseURL(u string) {
	c.baseURL = strings.TrimRight(u, "/")
	c.httpClient.SetBaseURL(c.baseURL)
}

// HealthCheck ping endpoint /health (atau base "/" sebagai fallback).
// nil = reachable, ErrOffline = jaringan mati, error lain = HTTP error.
//
// Khanza tidak punya endpoint /health standar, tapi GET / biasanya
// return 200 (Laravel welcome page) atau 401 (auth required) — yang
// penting koneksi TCP berhasil. ErrOffline hanya saat connection
// refused / DNS fail / network unreachable.
func (c *Client) HealthCheck(ctx context.Context) error {
	resp, err := c.httpClient.R().
		SetContext(ctx).
		Get("/health")
	if err != nil {
		return wrapOffline(err)
	}
	// 5xx → backend up tapi sakit
	if resp.StatusCode() >= 500 {
		return fmt.Errorf("khanza unhealthy: HTTP %d", resp.StatusCode())
	}
	// 2xx/3xx/4xx semua OK untuk health check (server alive)
	return nil
}

// envelope adalah response wrapper standard Khanza Laravel:
//
//	{ "success": true, "message": "OK", "data": <payload> }
//
// Field data sengaja json.RawMessage supaya bisa di-unmarshal ke
// target yang berbeda per endpoint.
type envelope struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// doGet menjalankan GET request ke path (relatif ke baseURL), handle
// status code transformasi standar, dan unmarshal envelope.data ke target.
func (c *Client) doGet(ctx context.Context, path string, target any) error {
	resp, err := c.httpClient.R().
		SetContext(ctx).
		Get(path)
	return c.handleResponse(resp, err, "GET "+path, target)
}

// doPost menjalankan POST JSON.
func (c *Client) doPost(ctx context.Context, path string, body any, target any) error {
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetBody(body).
		Post(path)
	return c.handleResponse(resp, err, "POST "+path, target)
}

// doPut menjalankan PUT JSON.
func (c *Client) doPut(ctx context.Context, path string, body any, target any) error {
	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetBody(body).
		Put(path)
	return c.handleResponse(resp, err, "PUT "+path, target)
}

// handleResponse memetakan error transport (offline detection) dan
// status code ke error yang sesuai. target nil OK untuk endpoint tanpa
// body penting (misal SimpanSEP yang cuma butuh sukses/gagal).
func (c *Client) handleResponse(resp *resty.Response, err error, label string, target any) error {
	if err != nil {
		if offline := wrapOffline(err); offline != err {
			c.logger.Warn("khanza: offline detected", "label", label)
			return offline
		}
		return fmt.Errorf("%s: %w", label, err)
	}

	switch resp.StatusCode() {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent:
		// success path — fall through
	case http.StatusUnauthorized:
		c.logger.Warn("khanza: API key expired atau salah",
			"label", label, "status", resp.StatusCode())
		return fmt.Errorf("%s: 401 unauthorized — periksa khanza_api_key", label)
	case http.StatusNotFound:
		// 404 di Khanza biasanya berarti pasien/resource tidak ada —
		// caller akan interpret berbeda per method (CariPasien null,
		// dll). Return error generic, biarkan caller decide.
		return fmt.Errorf("%s: 404 not found", label)
	default:
		if resp.StatusCode() >= 400 {
			return fmt.Errorf("%s: status %d body=%s",
				label, resp.StatusCode(), truncate(string(resp.Body()), 200))
		}
	}

	if target == nil {
		return nil
	}

	var env envelope
	if err := json.Unmarshal(resp.Body(), &env); err != nil {
		return fmt.Errorf("%s: unmarshal envelope: %w", label, err)
	}
	if !env.Success {
		return fmt.Errorf("%s: khanza error: %s", label, env.Message)
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(env.Data, target); err != nil {
		return fmt.Errorf("%s: unmarshal data: %w", label, err)
	}
	return nil
}

// truncate memotong string pada n karakter untuk pesan error supaya
// log tidak meledak kalau body response besar.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "...(truncated)"
}
