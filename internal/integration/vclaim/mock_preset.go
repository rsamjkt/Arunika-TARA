package vclaim

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

// NewMockPreset return MockVClaimClient dengan canned response varied
// untuk dev environment. Hit identifier yang berbeda akan trigger
// kategori Smart Detector berbeda — useful untuk test full flow.
//
// Scenario mapping (identifier → kategori detector):
//
//	"000001"        → Peserta aktif default (RujukanBaru flow)
//	"000002"        → Peserta aktif default (RujukanBaru flow)
//	"MJKN" prefix   → Peserta aktif + booking MJKN (untuk test MJKN flow)
//	"KONTROL" prefix → Peserta aktif + ada SKDP (untuk test Kontrol flow)
//	"INACTIVE"      → Peserta tidak aktif
//	"INVALID"       → GetPeserta error (test error path)
//	default         → Peserta aktif minimal data
//
// MockVClaimClient field di-set langsung supaya bisa di-tweak per-method
// kalau test butuh override.
func NewMockPreset() *MockVClaimClient {
	m := NewMock()

	// GetPeserta — lookup berdasar identifier
	m.GetPesertaFunc = func(ctx context.Context, identifier string, tgl time.Time) (*domain.Peserta, error) {
		id := strings.TrimSpace(identifier)
		if strings.HasPrefix(id, "INVALID") {
			return nil, fmt.Errorf("mock: peserta tidak ditemukan (test error path)")
		}
		if strings.HasPrefix(id, "INACTIVE") {
			return &domain.Peserta{
				NoKartu:      "0001234567890003",
				NoRM:         "078763",
				NIK:          "3271234567890003",
				Nama:         "PASIEN INACTIVE (DEMO)",
				TglLahir:     "1970-01-01",
				StatusAktif:  "0", // tidak aktif
				KelasHak:     "3",
				JenisPeserta: "PBI",
			}, nil
		}
		// Default: peserta aktif
		nama := "BUDI SUPRAYOGI (DEMO)"
		if strings.HasPrefix(id, "MJKN") {
			nama = "PASIEN MJKN (DEMO)"
		} else if strings.HasPrefix(id, "KONTROL") {
			nama = "PASIEN KONTROL (DEMO)"
		}
		noKartu := strings.TrimSpace(id)
		if noKartu == "" {
			noKartu = "0001234567890012"
		}
		return &domain.Peserta{
			NoKartu:      noKartu,
			NoRM:         "078763",
			NIK:          "3271234567890001",
			Nama:         nama,
			TglLahir:     "1959-12-03",
			StatusAktif:  "1", // aktif
			KelasHak:     "3",
			JenisPeserta: "PBPU",
		}, nil
	}

	// ValidasiRujukan — semua nomor rujukan valid (untuk test happy path)
	m.ValidasiRujukanFunc = func(ctx context.Context, noSurat string, tgl time.Time) (*domain.Rujukan, error) {
		return &domain.Rujukan{
			NoSurat:    noSurat,
			TglRujukan: tgl.AddDate(0, 0, -3).Format("2006-01-02"),
			TglBerlaku: tgl.AddDate(0, 3, 0).Format("2006-01-02"),
			KdPoli:     "OBG", // BPJS code, akan di-translate ke kd_poli RS
			KdDokter:   "61110",
			NmFaskes:   "PUSKESMAS DEMO FKTP",
		}, nil
	}

	// GetRencanaKontrol — kalau identifier mengandung "KONTROL", return 1 SKDP
	m.GetRencanaKontrolFunc = func(ctx context.Context, noKartu string, tgl time.Time) ([]domain.SuratKontrol, error) {
		// Mock heuristic: kalau noKartu mengandung "K" → kontrol
		if !strings.Contains(strings.ToUpper(noKartu), "K") {
			return nil, nil
		}
		return []domain.SuratKontrol{
			{
				NoSurat:    "0115R0500226K000449",
				NoRM:       "078763",
				TglRencana: tgl.Format("2006-01-02"), // hari ini
				KdPoli:     "U0077",
				NmPoli:     "POLI MATA",
				KdDokter:   "D0000089",
			},
		}, nil
	}

	// CreateSEP — return SEP dummy dengan no_sep generated
	m.CreateSEPFunc = func(ctx context.Context, req domain.SEPRequest) (*domain.SEP, error) {
		return &domain.SEP{
			NoSEP:     fmt.Sprintf("0115R%sV%06d", time.Now().Format("0501"), randSepSuffix()),
			NoKartu:   req.NoKartu,
			TglSEP:    req.TglSEP,
			KdPoli:    req.KdPoli,
			NmPoli:    "Poli (mock)",
			KdDokter:  req.KdDokter,
			NmDokter:  "dr. Mock, Sp.X",
			CreatedAt: time.Now(),
		}, nil
	}

	// CreateSEPKontrol — sama dengan CreateSEP tapi format K (Kontrol)
	m.CreateSEPKontrolFunc = func(ctx context.Context, req domain.SEPKontrolRequest) (*domain.SEP, error) {
		return &domain.SEP{
			NoSEP:     fmt.Sprintf("0115R%sK%06d", time.Now().Format("0501"), randSepSuffix()),
			NoKartu:   req.NoKartu,
			TglSEP:    req.TglSEP,
			NmPoli:    "Poli (mock kontrol)",
			KdDokter:  req.KdDokter,
			NmDokter:  "dr. Mock, Sp.X",
			CreatedAt: time.Now(),
		}, nil
	}

	// CekSEPDuplikasi — selalu nil (gak ada dup)
	m.CekSEPDuplikasiFunc = func(ctx context.Context, noKartu, tglSEP string) (*domain.SEP, error) {
		return nil, nil
	}

	// BuatRencanaKontrol — generate SKDP baru
	m.BuatRencanaKontrolFunc = func(ctx context.Context, req domain.RencanaKontrolRequest) (*domain.RencanaKontrol, error) {
		return &domain.RencanaKontrol{
			NoSuratKontrol: fmt.Sprintf("0115R%sK%06d", time.Now().Format("0501"), randSepSuffix()),
			NoSEP:          req.NoSEP,
			TglRencana:     req.TglRencanaKontrol,
			KdPoli:         req.PoliKontrol,
			KdDokter:       req.KodeDokter,
		}, nil
	}

	return m
}

// randSepSuffix simple counter (real-world: pakai counter atomic, tapi
// untuk dev cukup pseudo-random dari time nanos)
func randSepSuffix() int {
	return int(time.Now().UnixNano() % 1_000_000)
}
