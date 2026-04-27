package khanza

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

// MockKhanzaClient adalah implementasi KhanzaClient untuk testing.
//
// Dua API setting:
//
//  1. Per-method Func field — kontrol penuh per method, dipakai
//     detector test (P-011/P-012).
//  2. SetResponse(method, response, err) — sesuai spec P-020,
//     single-call setter dengan type-assert otomatis.
//
// Keduanya bisa dipakai bareng — Func selalu menang kalau di-set
// langsung. SetResponse internal-nya overwrite Func sesuai method.
type MockKhanzaClient struct {
	HealthCheckFunc       func(ctx context.Context) error
	CariPasienFunc        func(ctx context.Context, q string) (*domain.Pasien, error)
	GetSuratKontrolFunc   func(ctx context.Context, noRM string) ([]domain.SuratKontrol, error)
	GetRiwayatRANAPFunc   func(ctx context.Context, noRM string) ([]domain.RiwayatRANAP, error)
	GetKunjunganAktifFunc func(ctx context.Context, noRM string) ([]domain.Kunjungan, error)
	GetJadwalDokterFunc   func(ctx context.Context, kdPoli string, tgl time.Time) ([]domain.JadwalDokter, error)
	GetPoliklinikAktifFunc func(ctx context.Context) ([]domain.Poliklinik, error)
	GetBookingMJKNFunc    func(ctx context.Context, noRM string, tgl time.Time) (*domain.BookingMJKN, error)
	GetRujukanInternalAntarPoliFunc func(ctx context.Context, noRM string, daysBack int) ([]domain.RujukanInternalPoli, error)
	CheckDuplicateRegistrationFunc func(ctx context.Context, noRM, kdPoli, kdDokter, tglRegistrasi, kdPj string) (bool, error)
	CheckDoctorOnLeaveFunc         func(ctx context.Context, kdDokter, tglRegistrasi string) (bool, error)
	BuatPendaftaranFunc            func(ctx context.Context, req domain.PendaftaranRequest) (*domain.Pendaftaran, error)
	BuatAntrianFunc       func(ctx context.Context, req domain.AntrianRequest) (*domain.Ticket, error)
	SimpanSEPFunc            func(ctx context.Context, sep domain.SEP) error
	SimpanRujukMasukFunc     func(ctx context.Context, r domain.RujukMasuk) error
	SimpanRujukanBPJSFunc    func(ctx context.Context, r domain.RujukanBPJS) error
	SimpanSuratKontrolBPJSFunc func(ctx context.Context, sk domain.RencanaKontrol) error
	UpdateSatuSehatIDFunc    func(ctx context.Context, noRM, ihsNumber string) error

	mu        sync.Mutex
	callCount map[string]int
}

var _ KhanzaClient = (*MockKhanzaClient)(nil)

func NewMock() *MockKhanzaClient {
	return &MockKhanzaClient{callCount: make(map[string]int)}
}

func (m *MockKhanzaClient) CallCount(method string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount[method]
}

func (m *MockKhanzaClient) recordCall(name string) {
	m.mu.Lock()
	m.callCount[name]++
	m.mu.Unlock()
}

// SetResponse mengkonfigurasi response (atau error) untuk method
// dengan nama tertentu. Type-assert otomatis berdasarkan signature
// method. Panic jika tipe response tidak cocok — kesalahan test setup
// harus loud.
//
// Method names yang didukung (case-sensitive, sesuai nama method
// di interface KhanzaClient):
//
//	"CariPasien"        → *domain.Pasien
//	"GetSuratKontrol"   → []domain.SuratKontrol
//	"GetRiwayatRANAP"   → []domain.RiwayatRANAP
//	"GetKunjunganAktif" → []domain.Kunjungan
//	"GetJadwalDokter"   → []domain.JadwalDokter
//	"BuatPendaftaran"   → *domain.Pendaftaran
//	"BuatAntrian"       → *domain.Ticket
//	"SimpanSEP"         → nil (response diabaikan)
//	"UpdateSatuSehatID" → nil (response diabaikan)
func (m *MockKhanzaClient) SetResponse(method string, response any, err error) {
	switch method {
	case "HealthCheck":
		m.HealthCheckFunc = func(ctx context.Context) error { return err }
	case "CariPasien":
		p, _ := response.(*domain.Pasien)
		m.CariPasienFunc = func(ctx context.Context, q string) (*domain.Pasien, error) {
			return p, err
		}
	case "GetSuratKontrol":
		list, _ := response.([]domain.SuratKontrol)
		m.GetSuratKontrolFunc = func(ctx context.Context, noRM string) ([]domain.SuratKontrol, error) {
			return list, err
		}
	case "GetRiwayatRANAP":
		list, _ := response.([]domain.RiwayatRANAP)
		m.GetRiwayatRANAPFunc = func(ctx context.Context, noRM string) ([]domain.RiwayatRANAP, error) {
			return list, err
		}
	case "GetKunjunganAktif":
		list, _ := response.([]domain.Kunjungan)
		m.GetKunjunganAktifFunc = func(ctx context.Context, noRM string) ([]domain.Kunjungan, error) {
			return list, err
		}
	case "GetJadwalDokter":
		list, _ := response.([]domain.JadwalDokter)
		m.GetJadwalDokterFunc = func(ctx context.Context, kdPoli string, tgl time.Time) ([]domain.JadwalDokter, error) {
			return list, err
		}
	case "GetPoliklinikAktif":
		list, _ := response.([]domain.Poliklinik)
		m.GetPoliklinikAktifFunc = func(ctx context.Context) ([]domain.Poliklinik, error) {
			return list, err
		}
	case "GetBookingMJKN":
		b, _ := response.(*domain.BookingMJKN)
		m.GetBookingMJKNFunc = func(ctx context.Context, noRM string, tgl time.Time) (*domain.BookingMJKN, error) {
			return b, err
		}
	case "GetRujukanInternalAntarPoli":
		list, _ := response.([]domain.RujukanInternalPoli)
		m.GetRujukanInternalAntarPoliFunc = func(ctx context.Context, noRM string, daysBack int) ([]domain.RujukanInternalPoli, error) {
			return list, err
		}
	case "BuatPendaftaran":
		p, _ := response.(*domain.Pendaftaran)
		m.BuatPendaftaranFunc = func(ctx context.Context, req domain.PendaftaranRequest) (*domain.Pendaftaran, error) {
			return p, err
		}
	case "BuatAntrian":
		t, _ := response.(*domain.Ticket)
		m.BuatAntrianFunc = func(ctx context.Context, req domain.AntrianRequest) (*domain.Ticket, error) {
			return t, err
		}
	case "SimpanSEP":
		m.SimpanSEPFunc = func(ctx context.Context, sep domain.SEP) error {
			return err
		}
	case "SimpanRujukMasuk":
		m.SimpanRujukMasukFunc = func(ctx context.Context, r domain.RujukMasuk) error {
			return err
		}
	case "UpdateSatuSehatID":
		m.UpdateSatuSehatIDFunc = func(ctx context.Context, noRM, ihsNumber string) error {
			return err
		}
	default:
		panic(fmt.Sprintf("MockKhanzaClient.SetResponse: unknown method %q", method))
	}
}

func (m *MockKhanzaClient) HealthCheck(ctx context.Context) error {
	m.recordCall("HealthCheck")
	if m.HealthCheckFunc != nil {
		return m.HealthCheckFunc(ctx)
	}
	return nil
}

func (m *MockKhanzaClient) CariPasien(ctx context.Context, q string) (*domain.Pasien, error) {
	m.recordCall("CariPasien")
	if m.CariPasienFunc != nil {
		return m.CariPasienFunc(ctx, q)
	}
	return nil, nil
}

func (m *MockKhanzaClient) GetSuratKontrol(ctx context.Context, noRM string) ([]domain.SuratKontrol, error) {
	m.recordCall("GetSuratKontrol")
	if m.GetSuratKontrolFunc != nil {
		return m.GetSuratKontrolFunc(ctx, noRM)
	}
	return nil, nil
}

func (m *MockKhanzaClient) GetRiwayatRANAP(ctx context.Context, noRM string) ([]domain.RiwayatRANAP, error) {
	m.recordCall("GetRiwayatRANAP")
	if m.GetRiwayatRANAPFunc != nil {
		return m.GetRiwayatRANAPFunc(ctx, noRM)
	}
	return nil, nil
}

func (m *MockKhanzaClient) GetKunjunganAktif(ctx context.Context, noRM string) ([]domain.Kunjungan, error) {
	m.recordCall("GetKunjunganAktif")
	if m.GetKunjunganAktifFunc != nil {
		return m.GetKunjunganAktifFunc(ctx, noRM)
	}
	return nil, nil
}

func (m *MockKhanzaClient) GetJadwalDokter(ctx context.Context, kdPoli string, tgl time.Time) ([]domain.JadwalDokter, error) {
	m.recordCall("GetJadwalDokter")
	if m.GetJadwalDokterFunc != nil {
		return m.GetJadwalDokterFunc(ctx, kdPoli, tgl)
	}
	return nil, nil
}

func (m *MockKhanzaClient) GetPoliklinikAktif(ctx context.Context) ([]domain.Poliklinik, error) {
	m.recordCall("GetPoliklinikAktif")
	if m.GetPoliklinikAktifFunc != nil {
		return m.GetPoliklinikAktifFunc(ctx)
	}
	return nil, nil
}

func (m *MockKhanzaClient) GetBookingMJKN(ctx context.Context, noRM string, tgl time.Time) (*domain.BookingMJKN, error) {
	m.recordCall("GetBookingMJKN")
	if m.GetBookingMJKNFunc != nil {
		return m.GetBookingMJKNFunc(ctx, noRM, tgl)
	}
	return nil, nil
}

func (m *MockKhanzaClient) GetRujukanInternalAntarPoli(ctx context.Context, noRM string, daysBack int) ([]domain.RujukanInternalPoli, error) {
	m.recordCall("GetRujukanInternalAntarPoli")
	if m.GetRujukanInternalAntarPoliFunc != nil {
		return m.GetRujukanInternalAntarPoliFunc(ctx, noRM, daysBack)
	}
	return nil, nil
}

func (m *MockKhanzaClient) CheckDuplicateRegistration(ctx context.Context, noRM, kdPoli, kdDokter, tglRegistrasi, kdPj string) (bool, error) {
	m.recordCall("CheckDuplicateRegistration")
	if m.CheckDuplicateRegistrationFunc != nil {
		return m.CheckDuplicateRegistrationFunc(ctx, noRM, kdPoli, kdDokter, tglRegistrasi, kdPj)
	}
	return false, nil
}

func (m *MockKhanzaClient) CheckDoctorOnLeave(ctx context.Context, kdDokter, tglRegistrasi string) (bool, error) {
	m.recordCall("CheckDoctorOnLeave")
	if m.CheckDoctorOnLeaveFunc != nil {
		return m.CheckDoctorOnLeaveFunc(ctx, kdDokter, tglRegistrasi)
	}
	return false, nil
}

func (m *MockKhanzaClient) BuatPendaftaran(ctx context.Context, req domain.PendaftaranRequest) (*domain.Pendaftaran, error) {
	m.recordCall("BuatPendaftaran")
	if m.BuatPendaftaranFunc != nil {
		return m.BuatPendaftaranFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockKhanzaClient) BuatAntrian(ctx context.Context, req domain.AntrianRequest) (*domain.Ticket, error) {
	m.recordCall("BuatAntrian")
	if m.BuatAntrianFunc != nil {
		return m.BuatAntrianFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockKhanzaClient) SimpanSEP(ctx context.Context, sep domain.SEP) error {
	m.recordCall("SimpanSEP")
	if m.SimpanSEPFunc != nil {
		return m.SimpanSEPFunc(ctx, sep)
	}
	return nil
}

func (m *MockKhanzaClient) SimpanRujukMasuk(ctx context.Context, r domain.RujukMasuk) error {
	m.recordCall("SimpanRujukMasuk")
	if m.SimpanRujukMasukFunc != nil {
		return m.SimpanRujukMasukFunc(ctx, r)
	}
	return nil
}

func (m *MockKhanzaClient) SimpanRujukanBPJS(ctx context.Context, r domain.RujukanBPJS) error {
	m.recordCall("SimpanRujukanBPJS")
	if m.SimpanRujukanBPJSFunc != nil {
		return m.SimpanRujukanBPJSFunc(ctx, r)
	}
	return nil
}

func (m *MockKhanzaClient) SimpanSuratKontrolBPJS(ctx context.Context, sk domain.RencanaKontrol) error {
	m.recordCall("SimpanSuratKontrolBPJS")
	if m.SimpanSuratKontrolBPJSFunc != nil {
		return m.SimpanSuratKontrolBPJSFunc(ctx, sk)
	}
	return nil
}

func (m *MockKhanzaClient) UpdateSatuSehatID(ctx context.Context, noRM, ihsNumber string) error {
	m.recordCall("UpdateSatuSehatID")
	if m.UpdateSatuSehatIDFunc != nil {
		return m.UpdateSatuSehatIDFunc(ctx, noRM, ihsNumber)
	}
	return nil
}
