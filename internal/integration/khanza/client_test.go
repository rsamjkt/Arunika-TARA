package khanza

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arunika/apm-go/internal/config"
	"github.com/arunika/apm-go/internal/domain"
)

// newTestClient membangun Client diarahkan ke httptest server dengan
// timeout/retry yang ringan supaya test cepat.
func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	c := New(config.ServerConfig{
		KhanzaURL:    server.URL,
		KhanzaAPIKey: "test-api-key",
		TimeoutMs:    2000,
		Retry:        2,
	})
	c.httpClient.SetRetryWaitTime(50 * time.Millisecond)
	c.httpClient.SetRetryMaxWaitTime(150 * time.Millisecond)
	return c
}

// envelopeJSON membangun response Khanza dengan {success, data}.
func envelopeJSON(t *testing.T, success bool, data any, message string) []byte {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"success": success,
		"data":    data,
		"message": message,
	})
	return body
}

// ============================================================
// Auth header — verifikasi X-API-Key terkirim
// ============================================================

func TestKhanza_XApiKeyHeader_Terkirim(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-API-Key"); got != "test-api-key" {
			t.Errorf("X-API-Key = %q, want test-api-key", got)
		}
		w.Write(envelopeJSON(t, true, nil, "OK"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	_, _ = c.CariPasien(context.Background(), "RM-001")
}

// ============================================================
// Happy path — CariPasien
// ============================================================

func TestKhanza_CariPasien_Success(t *testing.T) {
	pasien := map[string]any{
		"no_rm":     "RM-12345",
		"nama":      "Budi Santoso",
		"nik":       "3271234567890001",
		"no_kartu":  "0001234567890012",
		"tgl_lahir": "1980-05-15",
		"jk":        "L",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/pasien") {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("q") != "RM-12345" {
			t.Errorf("q = %q", r.URL.Query().Get("q"))
		}
		w.Write(envelopeJSON(t, true, pasien, "OK"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	p, err := c.CariPasien(context.Background(), "RM-12345")
	if err != nil {
		t.Fatalf("CariPasien: %v", err)
	}
	if p == nil || p.NoRM != "RM-12345" || p.Nama != "Budi Santoso" {
		t.Errorf("pasien tidak match: %+v", p)
	}
}

func TestKhanza_CariPasien_404_ReturnNilTanpaError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	c := newTestClient(t, server)
	c.httpClient.SetRetryCount(0)

	p, err := c.CariPasien(context.Background(), "RM-XXX")
	if err != nil {
		t.Fatalf("404 seharusnya jadi (nil, nil), got err: %v", err)
	}
	if p != nil {
		t.Errorf("404 seharusnya nil pasien, got %+v", p)
	}
}

func TestKhanza_CariPasien_QueryKosong_Error(t *testing.T) {
	c := New(config.ServerConfig{KhanzaURL: "http://x", KhanzaAPIKey: "k"})
	_, err := c.CariPasien(context.Background(), "  ")
	if err == nil {
		t.Fatal("query kosong harus error")
	}
}

// ============================================================
// 401 — log warning + error
// ============================================================

func TestKhanza_401Unauthorized_LogWarningDanError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	c := newTestClient(t, server)
	c.httpClient.SetRetryCount(0)

	var buf bytes.Buffer
	c.SetLogger(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	_, err := c.CariPasien(context.Background(), "RM-001")
	if err == nil {
		t.Fatal("401 harus error")
	}
	if !strings.Contains(err.Error(), "401") && !strings.Contains(err.Error(), "unauthorized") {
		t.Errorf("error message tidak mengandung 401/unauthorized: %v", err)
	}

	logs := buf.String()
	if !strings.Contains(strings.ToLower(logs), "api key") {
		t.Errorf("log seharusnya warning tentang API key: %s", logs)
	}
	if !strings.Contains(logs, "WARN") {
		t.Errorf("log level harus WARN: %s", logs)
	}
}

// ============================================================
// 5xx retry behavior
// ============================================================

func TestKhanza_RetryPada5xx_LaluSukses(t *testing.T) {
	pasien := map[string]any{"no_rm": "RM-001", "nama": "X"}
	var hits atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 3 {
			http.Error(w, "internal", http.StatusInternalServerError)
			return
		}
		w.Write(envelopeJSON(t, true, pasien, "OK"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	c.httpClient.SetRetryCount(3) // 2 fail + 1 sukses

	p, err := c.CariPasien(context.Background(), "RM-001")
	if err != nil {
		t.Fatalf("CariPasien dengan retry: %v", err)
	}
	if p == nil || p.NoRM != "RM-001" {
		t.Errorf("pasien salah: %+v", p)
	}
	if hits.Load() < 3 {
		t.Errorf("expected setidaknya 3 hit, got %d", hits.Load())
	}
}

func TestKhanza_4xxTidakDiRetry(t *testing.T) {
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	c := newTestClient(t, server)
	c.httpClient.SetRetryCount(3)

	_, _ = c.CariPasien(context.Background(), "RM-001")
	if hits.Load() != 1 {
		t.Errorf("4xx tidak boleh di-retry — got %d hit, want 1", hits.Load())
	}
}

// ============================================================
// Offline detection — server di-close → ErrOffline
// ============================================================

func TestKhanza_ServerMati_ReturnErrOffline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(envelopeJSON(t, true, nil, "OK"))
	}))
	url := server.URL
	server.Close() // matikan SEBELUM request

	c := New(config.ServerConfig{
		KhanzaURL:    url,
		KhanzaAPIKey: "test",
		TimeoutMs:    1000,
	})
	c.httpClient.SetRetryCount(0) // jangan retry connection refused

	_, err := c.CariPasien(context.Background(), "RM-001")
	if err == nil {
		t.Fatal("expected error untuk server mati")
	}
	if !errors.Is(err, domain.ErrOffline) {
		t.Errorf("expected domain.ErrOffline, got: %v", err)
	}
}

// ============================================================
// Endpoint smoke tests — verifikasi method + path + body shape
// ============================================================

func TestKhanza_GetSuratKontrol_ParsesArray(t *testing.T) {
	list := []map[string]any{
		{"no_surat": "SK-001", "no_rm": "RM-001", "tgl_rencana": "2026-04-26", "kd_poli": "INT", "nm_poli": "Penyakit Dalam"},
		{"no_surat": "SK-002", "no_rm": "RM-001", "tgl_rencana": "2026-04-30", "kd_poli": "BDH"},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/pasien/RM-001/surat-kontrol") {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write(envelopeJSON(t, true, list, "OK"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	got, err := c.GetSuratKontrol(context.Background(), "RM-001")
	if err != nil {
		t.Fatalf("GetSuratKontrol: %v", err)
	}
	if len(got) != 2 || got[0].NoSurat != "SK-001" {
		t.Errorf("hasil salah: %+v", got)
	}
}

func TestKhanza_GetJadwalDokter_TanggalDiQueryString(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/poli/INT/jadwal") {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("tgl") != "2026-04-26" {
			t.Errorf("tgl query = %q", r.URL.Query().Get("tgl"))
		}
		w.Write(envelopeJSON(t, true, []map[string]any{
			{"kd_dokter": "D-001", "nm_dokter": "dr. A", "kd_poli": "INT", "aktif": true, "kuota": 30, "sisa": 25},
		}, "OK"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	tgl := time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC)
	got, err := c.GetJadwalDokter(context.Background(), "INT", tgl)
	if err != nil {
		t.Fatalf("GetJadwalDokter: %v", err)
	}
	if len(got) != 1 || got[0].KdDokter != "D-001" || got[0].Sisa != 25 {
		t.Errorf("jadwal salah: %+v", got)
	}
}

func TestKhanza_BuatPendaftaran_Body(t *testing.T) {
	resp := map[string]any{
		"no_rawat": "2026/04/26/0001", "no_rm": "RM-001",
		"kd_poli": "INT", "nm_poli": "Penyakit Dalam",
		"kd_dokter": "D-001", "nm_dokter": "dr. A",
		"tgl_periksa": "2026-04-26", "no_urut": 5,
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		body, _ := jsonDecodeBody(r)
		if body["no_sep"] != "SEP-001" {
			t.Errorf("body.no_sep salah: %v", body["no_sep"])
		}
		w.Write(envelopeJSON(t, true, resp, "OK"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	got, err := c.BuatPendaftaran(context.Background(), domain.PendaftaranRequest{
		NoRM: "RM-001", KdPoli: "INT", KdDokter: "D-001",
		TglPeriksa: "2026-04-26", Penjamin: "BPJS", NoSEP: "SEP-001",
	})
	if err != nil {
		t.Fatalf("BuatPendaftaran: %v", err)
	}
	if got.NoRawat != "2026/04/26/0001" || got.NoUrut != 5 {
		t.Errorf("hasil salah: %+v", got)
	}
}

func TestKhanza_BuatPendaftaran_BPJSTanpaSEP_Error(t *testing.T) {
	c := New(config.ServerConfig{KhanzaURL: "http://x", KhanzaAPIKey: "k"})
	_, err := c.BuatPendaftaran(context.Background(), domain.PendaftaranRequest{
		NoRM: "RM-001", KdPoli: "INT", KdDokter: "D-001",
		Penjamin: "BPJS", // NoSEP kosong
	})
	if err == nil {
		t.Fatal("BPJS tanpa NoSEP harus error sebelum HTTP call")
	}
}

func TestKhanza_BuatAntrian_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := jsonDecodeBody(r)
		if body["jenis"] != "POLI" {
			t.Errorf("body.jenis = %v", body["jenis"])
		}
		w.Write(envelopeJSON(t, true, map[string]any{
			"id":       "tk-001",
			"nomor":    "B-INT-005",
			"jenis":    "POLI",
			"prefix":   "B",
			"no_urut":  5,
			"no_poli":  "INT",
		}, "OK"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	tk, err := c.BuatAntrian(context.Background(), domain.AntrianRequest{
		Jenis: "POLI", KdPoli: "INT", NoSEP: "SEP-001",
	})
	if err != nil {
		t.Fatalf("BuatAntrian: %v", err)
	}
	if tk.Nomor != "B-INT-005" || tk.NoUrut != 5 {
		t.Errorf("ticket salah: %+v", tk)
	}
}

func TestKhanza_SimpanSEP_OKTanpaResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := newTestClient(t, server)
	err := c.SimpanSEP(context.Background(), domain.SEP{
		NoSEP: "SEP-001", NoKartu: "0001", TglSEP: "2026-04-26", KdPoli: "INT",
	})
	if err != nil {
		t.Errorf("SimpanSEP: %v", err)
	}
}

func TestKhanza_UpdateSatuSehatID_PUTRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/pasien/RM-001/satusehat-id") {
			t.Errorf("path = %s", r.URL.Path)
		}
		body, _ := jsonDecodeBody(r)
		if body["ihs_number"] != "P02476543210" {
			t.Errorf("body.ihs_number = %v", body["ihs_number"])
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := newTestClient(t, server)
	err := c.UpdateSatuSehatID(context.Background(), "RM-001", "P02476543210")
	if err != nil {
		t.Errorf("UpdateSatuSehatID: %v", err)
	}
}

// ============================================================
// Mock SetResponse — sesuai spec
// ============================================================

func TestMockKhanza_SetResponse_TipeMatch(t *testing.T) {
	m := NewMock()
	want := &domain.Pasien{NoRM: "RM-001", Nama: "X"}
	m.SetResponse("CariPasien", want, nil)

	got, err := m.CariPasien(context.Background(), "X")
	if err != nil {
		t.Fatalf("CariPasien: %v", err)
	}
	if got != want {
		t.Errorf("response mismatch: %+v vs %+v", got, want)
	}
	if m.CallCount("CariPasien") != 1 {
		t.Errorf("CallCount salah")
	}
}

func TestMockKhanza_SetResponse_DenganError(t *testing.T) {
	m := NewMock()
	m.SetResponse("GetSuratKontrol", nil, errors.New("simulasi error"))

	_, err := m.GetSuratKontrol(context.Background(), "RM-001")
	if err == nil || err.Error() != "simulasi error" {
		t.Errorf("expected simulasi error, got: %v", err)
	}
}

func TestMockKhanza_SetResponse_UnknownMethod_Panic(t *testing.T) {
	m := NewMock()
	defer func() {
		if r := recover(); r == nil {
			t.Error("SetResponse dengan method unknown harus panic (test setup error)")
		}
	}()
	m.SetResponse("MethodTidakAda", nil, nil)
}

// ============================================================
// Helpers
// ============================================================

func jsonDecodeBody(r *http.Request) (map[string]any, error) {
	defer r.Body.Close()
	var m map[string]any
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}
