package khanza

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

// Coverage booster — exercises endpoint methods + mock branches yang
// belum di-touch oleh client_test.go.

func TestKhanza_GetRiwayatRANAP_Smoke(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/pasien/RM-001/ranap") {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.Write(envelopeJSON(t, true, []map[string]any{
			{"no_rm": "RM-001", "no_rawat": "R-001", "tgl_keluar": "2026-04-25", "status_pulang": "PULANG"},
		}, "OK"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	got, err := c.GetRiwayatRANAP(context.Background(), "RM-001")
	if err != nil {
		t.Fatalf("GetRiwayatRANAP: %v", err)
	}
	if len(got) != 1 || got[0].NoRawat != "R-001" || got[0].StatusPulang != "PULANG" {
		t.Errorf("hasil salah: %+v", got)
	}

	// noRM kosong → return nil tanpa HTTP call
	empty, err := c.GetRiwayatRANAP(context.Background(), "")
	if err != nil || empty != nil {
		t.Errorf("noRM kosong harus return (nil, nil), got (%v, %v)", empty, err)
	}
}

func TestKhanza_GetKunjunganAktif_Smoke(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(envelopeJSON(t, true, []map[string]any{
			{
				"no_rm": "RM-001", "no_rawat": "R-002",
				"kd_poli": "INT", "jns_pelayanan": "1",
				"no_skdp": "SKDP-1", "kd_poli_skdp": "JTG",
			},
		}, "OK"))
	}))
	defer server.Close()

	c := newTestClient(t, server)
	got, err := c.GetKunjunganAktif(context.Background(), "RM-001")
	if err != nil {
		t.Fatalf("GetKunjunganAktif: %v", err)
	}
	if len(got) != 1 || got[0].NoSKDP != "SKDP-1" {
		t.Errorf("hasil salah: %+v", got)
	}
	if !got[0].PunyaSKDPBedaPoli() {
		t.Errorf("kunjungan harus terdeteksi punya SKDP beda poli")
	}
}

// ============================================================
// Coverage booster: SetResponse untuk semua method
// ============================================================

func TestMockKhanza_SetResponse_AllMethods(t *testing.T) {
	m := NewMock()
	ctx := context.Background()

	// GetSuratKontrol
	m.SetResponse("GetSuratKontrol", []domain.SuratKontrol{{NoSurat: "SK"}}, nil)
	if got, _ := m.GetSuratKontrol(ctx, "X"); len(got) != 1 {
		t.Errorf("GetSuratKontrol via SetResponse")
	}

	// GetRiwayatRANAP
	m.SetResponse("GetRiwayatRANAP", []domain.RiwayatRANAP{{NoRawat: "R"}}, nil)
	if got, _ := m.GetRiwayatRANAP(ctx, "X"); len(got) != 1 {
		t.Errorf("GetRiwayatRANAP via SetResponse")
	}

	// GetKunjunganAktif
	m.SetResponse("GetKunjunganAktif", []domain.Kunjungan{{NoRawat: "K"}}, nil)
	if got, _ := m.GetKunjunganAktif(ctx, "X"); len(got) != 1 {
		t.Errorf("GetKunjunganAktif via SetResponse")
	}

	// GetJadwalDokter
	m.SetResponse("GetJadwalDokter", []domain.JadwalDokter{{KdDokter: "D"}}, nil)
	if got, _ := m.GetJadwalDokter(ctx, "INT", time.Now()); len(got) != 1 {
		t.Errorf("GetJadwalDokter via SetResponse")
	}

	// BuatPendaftaran
	m.SetResponse("BuatPendaftaran", &domain.Pendaftaran{NoRawat: "R-001"}, nil)
	if got, _ := m.BuatPendaftaran(ctx, domain.PendaftaranRequest{}); got == nil || got.NoRawat != "R-001" {
		t.Errorf("BuatPendaftaran via SetResponse")
	}

	// BuatAntrian
	m.SetResponse("BuatAntrian", &domain.Ticket{Nomor: "A-001"}, nil)
	if got, _ := m.BuatAntrian(ctx, domain.AntrianRequest{}); got == nil || got.Nomor != "A-001" {
		t.Errorf("BuatAntrian via SetResponse")
	}

	// SimpanSEP — error path
	m.SetResponse("SimpanSEP", nil, errors.New("simulasi"))
	if err := m.SimpanSEP(ctx, domain.SEP{}); err == nil {
		t.Errorf("SimpanSEP error via SetResponse")
	}

	// UpdateSatuSehatID — sukses path
	m.SetResponse("UpdateSatuSehatID", nil, nil)
	if err := m.UpdateSatuSehatID(ctx, "X", "Y"); err != nil {
		t.Errorf("UpdateSatuSehatID via SetResponse")
	}
}
