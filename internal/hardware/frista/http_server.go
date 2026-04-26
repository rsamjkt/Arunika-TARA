package frista

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"
)

// startHTTPLocked memulai HTTP server di 127.0.0.1:serverPort dengan
// 3 route untuk simulasi tap kartu. Caller WAJIB sudah memegang m.mu.
//
// Bind ke 127.0.0.1 saja (BUKAN :port) supaya endpoint mock tidak
// reachable dari LAN — hanya dari mesin development.
func (m *MockReader) startHTTPLocked(ctx context.Context) error {
	addr := "127.0.0.1:" + strconv.Itoa(m.serverPort)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("frista mock: bind %s: %w", addr, err)
	}
	m.serverAddr = listener.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/mock/card-read", m.handleCardRead)
	mux.HandleFunc("/mock/card-read-delay", m.handleCardReadDelay)
	mux.HandleFunc("/mock/fp-fail", m.handleFPFail)
	mux.HandleFunc("/", m.handleInfoPage)

	m.server = &http.Server{
		Handler:           mux,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	// log dulu sebelum Serve goroutine — supaya pasti tampil dengan
	// addr yang benar (bukan dari listener yang sudah di-Close oleh
	// race kalau Stop dipanggil cepat sekali).
	m.logger.Info("frista mock: HTTP server aktif",
		"url", "http://"+m.serverAddr)

	go func() {
		if err := m.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			m.logger.Error("frista mock: HTTP server error", "err", err.Error())
		}
	}()
	return nil
}

// cardReadBody adalah JSON body untuk POST /mock/card-read.
// Pakai snake_case sesuai konvensi REST API yang umum di Indonesia.
type cardReadBody struct {
	NIK      string `json:"nik"`
	Nama     string `json:"nama"`
	TglLahir string `json:"tgl_lahir"`
	Alamat   string `json:"alamat"`
	NoKartu  string `json:"no_kartu"`
}

func (b cardReadBody) toCardData() CardData {
	return CardData{
		NIK:      b.NIK,
		Nama:     b.Nama,
		TglLahir: b.TglLahir,
		Alamat:   b.Alamat,
		NoKartu:  b.NoKartu,
	}
}

// handleCardRead — POST /mock/card-read
//
//	Body JSON: { nik, nama, tgl_lahir, alamat, no_kartu }
//	Response 202: { "ok": true, "queued": true }
//	Response 400: { "error": "..." }
//	Response 503: { "error": "channel penuh" }
func (m *MockReader) handleCardRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": "method not allowed, gunakan POST",
		})
		return
	}

	var body cardReadBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "JSON invalid: " + err.Error(),
		})
		return
	}
	if body.NIK == "" && body.NoKartu == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "minimal salah satu dari nik atau no_kartu harus diisi",
		})
		return
	}

	if err := m.EmitCard(body.toCardData()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": err.Error(),
		})
		return
	}
	m.logger.Info("frista mock: card-read injected",
		"nik_masked", maskID(body.NIK),
		"no_kartu_masked", maskID(body.NoKartu))
	writeJSON(w, http.StatusAccepted, map[string]any{
		"ok": true, "queued": true,
	})
}

// handleCardReadDelay — POST /mock/card-read-delay?seconds=N
//
//	Body JSON sama dengan /mock/card-read.
//	Response 202: { "ok": true, "scheduled_in_sec": N }
//
// Implementasi async — request return cepat, emit di goroutine
// terpisah setelah delay. Cocok untuk test scenario "kartu di-tap
// 3 detik dari sekarang".
func (m *MockReader) handleCardReadDelay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": "method not allowed, gunakan POST",
		})
		return
	}

	secStr := r.URL.Query().Get("seconds")
	if secStr == "" {
		secStr = "3"
	}
	sec, err := strconv.Atoi(secStr)
	if err != nil || sec < 0 || sec > 60 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "seconds harus integer 0-60",
		})
		return
	}

	var body cardReadBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "JSON invalid: " + err.Error(),
		})
		return
	}

	go func(d time.Duration, c CardData) {
		time.Sleep(d)
		if err := m.EmitCard(c); err != nil {
			m.logger.Warn("frista mock: delayed emit gagal",
				"err", err.Error())
		}
	}(time.Duration(sec)*time.Second, body.toCardData())

	writeJSON(w, http.StatusAccepted, map[string]any{
		"ok":               true,
		"scheduled_in_sec": sec,
	})
}

// handleFPFail — POST /mock/fp-fail
//
//	Trigger fingerprint mock untuk gagal sekali pada Verify() berikutnya.
//	Body tidak diperlukan. Idempotent — dipanggil 2x berturut tetap
//	hanya 1 fail (karena flag direset setelah 1 Verify).
//
//	Response 202: { "ok": true }
//	Response 200 + warning: { "ok": true, "warning": "..." } kalau callback
//	   belum di-wire (Wails app belum panggil SetOnFPFail).
func (m *MockReader) handleFPFail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"error": "method not allowed, gunakan POST",
		})
		return
	}

	m.mu.Lock()
	cb := m.onFPFailRequest
	m.mu.Unlock()

	if cb == nil {
		m.logger.Warn("frista mock: /mock/fp-fail dipanggil tapi callback belum di-wire")
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"warning": "callback belum terdaftar — panggil fristaMock.SetOnFPFail(fpMock.SetNextFail) saat init",
		})
		return
	}

	cb()
	m.logger.Info("frista mock: fingerprint fail-next triggered")
	writeJSON(w, http.StatusAccepted, map[string]any{"ok": true})
}

// handleInfoPage — GET /
// HTML sederhana dengan endpoint info + curl examples copy-paste.
func (m *MockReader) handleInfoPage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, infoPageHTML, m.serverAddr, m.serverAddr, m.serverAddr, m.serverAddr)
}

// infoPageHTML diformat dengan 4x %s — semuanya isi serverAddr.
const infoPageHTML = `<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="utf-8">
<title>Frista Mock - APM (T.A.R.A)</title>
<style>
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
         max-width: 720px; margin: 40px auto; padding: 0 20px; color: #0E1117;
         line-height: 1.6; }
  h1 { color: #1B4FD8; margin-bottom: 4px; }
  .subtitle { color: #4B5563; margin-top: 0; }
  h2 { color: #1E40AF; margin-top: 32px; border-bottom: 1px solid #E4E6EA;
       padding-bottom: 8px; }
  pre { background: #F5F6F8; border: 1px solid #E4E6EA; border-radius: 8px;
        padding: 12px 16px; overflow-x: auto; font-size: 13px; line-height: 1.5; }
  code { background: #EEF2FF; color: #1B4FD8; padding: 2px 6px; border-radius: 4px;
         font-size: 13px; }
  .endpoint { display: inline-block; background: #1B4FD8; color: white;
              padding: 2px 8px; border-radius: 4px; font-family: monospace;
              font-size: 12px; }
  .ok { background: #ECFDF5; color: #065F46; padding: 8px 12px; border-radius: 6px;
        border-left: 3px solid #6EE7B7; margin: 16px 0; }
</style>
</head>
<body>
<h1>Frista Mock — APM</h1>
<p class="subtitle">Simulasi card reader untuk development di Mac/Linux.</p>

<div class="ok">
  Server aktif di <code>http://%s</code> (terikat localhost only).
</div>

<h2>POST /mock/card-read</h2>
<p>Inject CardData ke channel langsung. Wails frontend akan menerima event
<code>frista:card_read</code> seketika.</p>
<pre>curl -X POST http://%s/mock/card-read \
  -H "Content-Type: application/json" \
  -d '{
    "nik":       "3271234567890001",
    "nama":      "Budi Santoso",
    "tgl_lahir": "1990-01-15",
    "alamat":    "Jl. Merdeka No.1",
    "no_kartu":  "0001234567890012"
  }'</pre>

<h2>POST /mock/card-read-delay?seconds=N</h2>
<p>Inject setelah delay N detik (default 3, max 60). Async — request
return seketika, emit terjadi di background.</p>
<pre>curl -X POST 'http://%s/mock/card-read-delay?seconds=5' \
  -H "Content-Type: application/json" \
  -d '{"nik":"3271234567890001","nama":"Budi"}'</pre>

<h2>POST /mock/fp-fail</h2>
<p>Trigger fingerprint mock untuk gagal sekali pada Verify() berikutnya.
Body tidak diperlukan.</p>
<pre>curl -X POST http://%s/mock/fp-fail</pre>

<h2>Tip</h2>
<p>Pakai shortcut Makefile: <code>make mock-card-default</code> atau
<code>make mock-card-read NIK=... NAMA=... KARTU=...</code></p>

</body>
</html>
`

// ============================================================
// Helpers
// ============================================================

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// maskID — mask 16-digit identifier menjadi 4 digit terakhir saja.
// Duplikasi kecil dari detector.maskID supaya frista tetap self-contained
// (tidak depend ke service layer untuk PHI helper).
func maskID(id string) string {
	if len(id) < 8 {
		return "***"
	}
	out := make([]byte, len(id))
	for i := range out {
		if i >= len(id)-4 {
			out[i] = id[i]
		} else {
			out[i] = '*'
		}
	}
	return string(out)
}
