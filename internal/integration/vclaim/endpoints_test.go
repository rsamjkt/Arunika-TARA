package vclaim

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

func TestVClaim_GetRencanaKontrol_ParsesList(t *testing.T) {
	plaintext := []byte(`{
        "list": [
            {
                "noSuratKontrol": "SK-001",
                "noRekamMedis": "RM-12345",
                "tglRencanaKontrol": "2026-04-26",
                "poliKontrol": {"kode":"INT","nama":"Penyakit Dalam"},
                "dokter": {"kode":"D-001"}
            },
            {
                "noSuratKontrol": "SK-002",
                "tglRencanaKontrol": "2026-04-27",
                "poliKontrol": {"kode":"BDH","nama":"Bedah"},
                "dokter": {"kode":"D-002"}
            }
        ]
    }`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/RencanaKontrol/List/") {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write(makeEnvelope(t, "200", "OK", plaintext, "rahasia-bpjs", "12345"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	list, err := c.GetRencanaKontrol(context.Background(), "0001234567890012",
		time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GetRencanaKontrol: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 surat kontrol, got %d", len(list))
	}
	if list[0].NoSurat != "SK-001" || list[0].NmPoli != "Penyakit Dalam" {
		t.Errorf("item 0 mismatch: %+v", list[0])
	}
}

func TestVClaim_GetRiwayatPelayanan_ParsesArray(t *testing.T) {
	plaintext := []byte(`{
        "noKartu": "0001234567890012",
        "riwayat": [
            {
                "noSep": "SEP-001",
                "tglSep": "2026-04-20",
                "jnsPelayanan": "1",
                "poli": {"kode":"INT","nama":"Penyakit Dalam"},
                "dpjp": {"kode":"D-001","nama":"dr. A"},
                "diagnosa": {"kode":"K30","nama":"Dyspepsia"}
            }
        ]
    }`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/RiwayatPelayanan/") {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write(makeEnvelope(t, "200", "OK", plaintext, "rahasia-bpjs", "12345"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	awal := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	akhir := time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC)
	list, err := c.GetRiwayatPelayanan(context.Background(), "0001234567890012", awal, akhir)
	if err != nil {
		t.Fatalf("GetRiwayatPelayanan: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 riwayat, got %d", len(list))
	}
	if list[0].NoSEP != "SEP-001" {
		t.Errorf("NoSEP = %q", list[0].NoSEP)
	}
	if list[0].JnsPelayanan != "1" {
		t.Errorf("JnsPelayanan = %q", list[0].JnsPelayanan)
	}
	if list[0].NmDiagnosa != "Dyspepsia" {
		t.Errorf("NmDiagnosa = %q", list[0].NmDiagnosa)
	}
}

func TestVClaim_ValidasiRujukan_HitungTglBerlaku90Hari(t *testing.T) {
	plaintext := []byte(`{
        "rujukan": {
            "noKunjungan": "RUJ-001",
            "tglKunjungan": "2026-04-01",
            "provPerujuk": {"nama":"PKM Sukamaju"},
            "poliRujukan": {"kode":"INT"},
            "diagnosa": {"kode":"K30"}
        }
    }`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/Rujukan/noSurat/RUJ-001") {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write(makeEnvelope(t, "200", "OK", plaintext, "rahasia-bpjs", "12345"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	r, err := c.ValidasiRujukan(context.Background(), "RUJ-001",
		time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("ValidasiRujukan: %v", err)
	}
	if r.NoSurat != "RUJ-001" {
		t.Errorf("NoSurat = %q", r.NoSurat)
	}
	// TglRujukan + 90 hari = 2026-06-30
	if r.TglBerlaku != "2026-06-30" {
		t.Errorf("TglBerlaku = %q, expected 2026-06-30", r.TglBerlaku)
	}
	if r.NmFaskes != "PKM Sukamaju" {
		t.Errorf("NmFaskes = %q", r.NmFaskes)
	}
}

func TestVClaim_CreateSEPKontrol_PostKeEndpointKontrol(t *testing.T) {
	plaintext := []byte(`{"sep":{"noSep":"SEP-K-001","tglSep":"2026-04-26","peserta":{"noKartu":"0001234567890012"},"poli":{"kdPoli":"INT","nmPoli":"Penyakit Dalam"},"dpjp":{"kdDokter":"D-001","nmDokter":"dr. B"}}}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/SEP/2.0/kontrol/insert") {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write(makeEnvelope(t, "200", "OK", plaintext, "rahasia-bpjs", "12345"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	sep, err := c.CreateSEPKontrol(context.Background(), domain.SEPKontrolRequest{
		NoSuratKontrol: "SK-001",
		NoKartu:        "0001234567890012",
		TglSEP:         "2026-04-26",
		KdDokter:       "D-001",
		KelasRawat:     "2",
		JnsPelayanan:   "1",
		FPToken:        "MOCK_FP",
	})
	if err != nil {
		t.Fatalf("CreateSEPKontrol: %v", err)
	}
	if sep.NoSEP != "SEP-K-001" {
		t.Errorf("NoSEP = %q", sep.NoSEP)
	}
}

// ============================================================
// Mock client — verifikasi compile-time interface compliance + counter
// ============================================================

func TestVClaim_Mock_CallCountAndStub(t *testing.T) {
	mock := NewMock()

	// Stub GetPeserta
	mock.GetPesertaFunc = func(ctx context.Context, identifier string, tgl time.Time) (*domain.Peserta, error) {
		return &domain.Peserta{NoKartu: identifier, StatusAktif: "1"}, nil
	}

	// Panggil 3 kali
	for i := 0; i < 3; i++ {
		p, err := mock.GetPeserta(context.Background(), "X", time.Now())
		if err != nil || p == nil || p.NoKartu != "X" {
			t.Fatalf("iter %d: stub return salah", i)
		}
	}
	if got := mock.CallCount("GetPeserta"); got != 3 {
		t.Errorf("CallCount(GetPeserta) = %d, want 3", got)
	}

	// Method tanpa stub return zero-value
	list, err := mock.GetRencanaKontrol(context.Background(), "X", time.Now())
	if err != nil {
		t.Errorf("unstubbed method seharusnya return nil error")
	}
	if len(list) != 0 {
		t.Errorf("unstubbed method seharusnya return empty slice")
	}
	if mock.CallCount("GetRencanaKontrol") != 1 {
		t.Errorf("CallCount(GetRencanaKontrol) salah")
	}
}
