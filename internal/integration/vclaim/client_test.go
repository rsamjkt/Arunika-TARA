package vclaim

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arunika/apm-go/internal/config"
	"github.com/arunika/apm-go/internal/domain"
)

// newTestClient membangun Client yang baseURL-nya diarahkan ke server
// dan timeout/retry ringan supaya test cepat.
// Clock di-freeze ke testFixedTS supaya key AES encrypt (makeEnvelope)
// dan decrypt (parseEnvelope) menggunakan timestamp yang sama.
func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	c := New(config.BPJSConfig{
		VClaimURL:      server.URL,
		ConsID:         "12345",
		ConsumerSecret: "rahasia-bpjs",
	})
	c.now = func() time.Time { return time.Unix(testFixedTS, 0) }
	c.SetTimeout(2 * time.Second)
	c.httpClient.SetRetryCount(2)
	c.httpClient.SetRetryWaitTime(50 * time.Millisecond)
	c.httpClient.SetRetryMaxWaitTime(150 * time.Millisecond)
	return c
}

// makeEnvelope membangun JSON envelope BPJS dengan response yang ter-enkripsi
// (sesuai protocol VClaim v2.0).
func makeEnvelope(t *testing.T, code, message string, plaintext []byte, secret, consID string) []byte {
	t.Helper()
	cipher, err := encrypt(consID, secret, testFixedTS, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	body, _ := json.Marshal(map[string]any{
		"metaData": map[string]string{"code": code, "message": message},
		"response": cipher,
	})
	return body
}

// makeErrorEnvelope membangun envelope tanpa response (error case).
func makeErrorEnvelope(t *testing.T, code, message string) []byte {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"metaData": map[string]string{"code": code, "message": message},
		"response": nil,
	})
	return body
}

// ============================================================
// GetPeserta — happy path
// ============================================================

func TestVClaim_GetPeserta_Success(t *testing.T) {
	plaintext := []byte(`{
        "peserta": {
            "noKartu": "0001234567890012",
            "nik": "3271234567890001",
            "nama": "Budi Santoso",
            "tglLahir": "1980-05-15",
            "statusPeserta": {"kode":"1","keterangan":"AKTIF"},
            "hakKelas": {"kode":"2","keterangan":"Kelas 2"},
            "jenisPeserta": {"kode":"1","keterangan":"PBI"},
            "mr": {"noMR":"RM-12345"}
        }
    }`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verifikasi auth headers ada
		if r.Header.Get("X-cons-id") != "12345" {
			t.Errorf("X-cons-id header missing/wrong: %q", r.Header.Get("X-cons-id"))
		}
		if r.Header.Get("X-signature") == "" {
			t.Errorf("X-signature header missing")
		}
		// Path harus mengandung /Peserta/noKartu/
		if !strings.Contains(r.URL.Path, "/Peserta/noKartu/0001234567890012") {
			t.Errorf("path salah: %s", r.URL.Path)
		}

		w.Write(makeEnvelope(t, "200", "OK", plaintext, "rahasia-bpjs", "12345"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	p, err := c.GetPeserta(context.Background(), "0001234567890012", time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GetPeserta: %v", err)
	}
	if p.NoKartu != "0001234567890012" {
		t.Errorf("NoKartu = %q", p.NoKartu)
	}
	if p.Nama != "Budi Santoso" {
		t.Errorf("Nama = %q", p.Nama)
	}
	if !p.IsAktif() {
		t.Errorf("seharusnya aktif (status kode 1)")
	}
	if p.NoRM != "RM-12345" {
		t.Errorf("NoRM = %q", p.NoRM)
	}
}

// ============================================================
// GetPeserta — peserta tidak aktif → ErrPesertaTidakAktif
// ============================================================

func TestVClaim_GetPeserta_TidakAktif_MengembalikanErrPesertaTidakAktif(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(makeErrorEnvelope(t, "-3", "Status kepesertaan tidak aktif"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	_, err := c.GetPeserta(context.Background(), "0001234567890012", time.Now())
	if err == nil {
		t.Fatal("expected error untuk peserta tidak aktif, got nil")
	}
	if !errors.Is(err, domain.ErrPesertaTidakAktif) {
		t.Errorf("error harus wrap domain.ErrPesertaTidakAktif, got: %v", err)
	}
}

// ============================================================
// GetPeserta — cascade noKartu → NIK
// ============================================================

func TestVClaim_GetPeserta_CascadeNoKartuKeNIK(t *testing.T) {
	plaintext := []byte(`{"peserta":{"noKartu":"0001234567890012","nik":"3271234567890001","statusPeserta":{"kode":"1"}}}`)
	var hits atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		switch {
		case strings.Contains(r.URL.Path, "/Peserta/noKartu/"):
			// noKartu lookup gagal — peserta tidak ditemukan
			w.Write(makeErrorEnvelope(t, "-2", "Peserta tidak ditemukan"))
		case strings.Contains(r.URL.Path, "/Peserta/nik/"):
			// NIK lookup berhasil
			w.Write(makeEnvelope(t, "200", "OK", plaintext, "rahasia-bpjs", "12345"))
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	c := newTestClient(t, server)
	p, err := c.GetPeserta(context.Background(), "3271234567890001", time.Now())
	if err != nil {
		t.Fatalf("GetPeserta cascade: %v", err)
	}
	if p.NIK != "3271234567890001" {
		t.Errorf("NIK = %q", p.NIK)
	}
	if hits.Load() != 2 {
		t.Errorf("expected 2 hit (noKartu lalu NIK), got %d", hits.Load())
	}
}

func TestVClaim_GetPeserta_TidakAktif_TidakCascadeKeNIK(t *testing.T) {
	// Status tidak aktif adalah error informatif — TIDAK fallback ke NIK lookup.
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Write(makeErrorEnvelope(t, "-3", "Status kepesertaan tidak aktif"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	_, err := c.GetPeserta(context.Background(), "0001234567890012", time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
	if hits.Load() != 1 {
		t.Errorf("expected 1 hit saja (tidak cascade), got %d", hits.Load())
	}
}

// ============================================================
// Retry behavior pada 500
// ============================================================

func TestVClaim_RetryPada500_LaluSukses(t *testing.T) {
	plaintext := []byte(`{"peserta":{"noKartu":"X","statusPeserta":{"kode":"1"}}}`)
	var hits atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n < 3 {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Write(makeEnvelope(t, "200", "OK", plaintext, "rahasia-bpjs", "12345"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	c.httpClient.SetRetryCount(3) // butuh 3 retry kalau gagal 2x dulu

	_, err := c.GetPeserta(context.Background(), "X", time.Now())
	if err != nil {
		t.Fatalf("GetPeserta dengan retry: %v", err)
	}
	if hits.Load() < 3 {
		t.Errorf("expected setidaknya 3 hit (2 fail + 1 sukses), got %d", hits.Load())
	}
}

func TestVClaim_400TidakDiRetry(t *testing.T) {
	// 4xx error tidak boleh di-retry (validation/auth tidak akan berubah).
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer server.Close()

	c := newTestClient(t, server)
	c.httpClient.SetRetryCount(3)

	_, err := c.GetPeserta(context.Background(), "X", time.Now())
	if err == nil {
		t.Fatal("expected error")
	}
	if hits.Load() != 1 {
		t.Errorf("4xx tidak boleh di-retry — expected 1 hit, got %d", hits.Load())
	}
}

// ============================================================
// Timeout handling
// ============================================================

func TestVClaim_TimeoutContextDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
			fmt.Fprintln(w, "{}")
		case <-r.Context().Done():
			return
		}
	}))
	defer server.Close()

	c := newTestClient(t, server)
	c.SetTimeout(10 * time.Second) // client timeout panjang
	c.httpClient.SetRetryCount(0)

	// Context yang dibatalkan oleh caller harus dihormati dengan cepat
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := c.GetPeserta(ctx, "X", time.Now())
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if elapsed > 1*time.Second {
		t.Errorf("seharusnya cancel cepat (<1s), elapsed=%v", elapsed)
	}
}

// ============================================================
// CreateSEP — happy path verifikasi POST + body
// ============================================================

func TestVClaim_CreateSEP_PostBodyDanResponse(t *testing.T) {
	plaintext := []byte(`{"sep":{"noSep":"0123456789012345678","tglSep":"2026-04-26","peserta":{"noKartu":"0001234567890012"},"poli":{"kdPoli":"INT","nmPoli":"Penyakit Dalam"},"dpjp":{"kdDokter":"D-001","nmDokter":"dr. Smith"}}}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/SEP/2.0/insert") {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write(makeEnvelope(t, "200", "OK", plaintext, "rahasia-bpjs", "12345"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	sep, err := c.CreateSEP(context.Background(), domain.SEPRequest{
		NoKartu:      "0001234567890012",
		TglSEP:       "2026-04-26",
		KdPoli:       "INT",
		KdDokter:     "D-001",
		JnsPelayanan: "1",
		KelasRawat:   "2",
		NoRujukan:    "RUJ-001",
		FPToken:      "MOCK_FP_TOKEN",
	})
	if err != nil {
		t.Fatalf("CreateSEP: %v", err)
	}
	if sep.NoSEP != "0123456789012345678" {
		t.Errorf("NoSEP = %q", sep.NoSEP)
	}
	if sep.NmPoli != "Penyakit Dalam" {
		t.Errorf("NmPoli = %q", sep.NmPoli)
	}
}

// ============================================================
// Mapping error code spesifik
// ============================================================

func TestVClaim_ErrorCodes_DiPetakanKeDomainErrors(t *testing.T) {
	cases := []struct {
		code     string
		message  string
		wantSentinel error
	}{
		{"-3", "Status tidak aktif", domain.ErrPesertaTidakAktif},
		{"-5", "Rujukan expired", domain.ErrRujukanExpired},
		{"-6", "SEP sudah dibuat", domain.ErrDuplikasiSEP},
		{"-10", "Fingerprint diperlukan", domain.ErrBiometrikDiperlukan},
	}

	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write(makeErrorEnvelope(t, tc.code, tc.message))
			}))
			defer server.Close()

			c := newTestClient(t, server)
			_, err := c.GetPeserta(context.Background(), "X", time.Now())
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, tc.wantSentinel) {
				t.Errorf("code %s: error harus wrap %v, got: %v",
					tc.code, tc.wantSentinel, err)
			}
		})
	}
}
