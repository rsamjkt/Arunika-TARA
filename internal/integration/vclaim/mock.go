package vclaim

import (
	"context"
	"sync"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

// MockVClaimClient adalah implementasi VClaimClient untuk testing.
// Setiap method bisa di-stub via field Func — kalau Func nil,
// method return zero-value + nil error.
//
// Thread-safe untuk read & call counting; setter Func sebaiknya
// dipanggil sebelum test concurrent dimulai.
//
// Compile-time check: var _ VClaimClient = (*MockVClaimClient)(nil)
type MockVClaimClient struct {
	GetPesertaFunc           func(ctx context.Context, identifier string, tgl time.Time) (*domain.Peserta, error)
	GetRiwayatPelayananFunc  func(ctx context.Context, noKartu string, tglAwal, tglAkhir time.Time) ([]domain.RiwayatPelayanan, error)
	ValidasiRujukanFunc      func(ctx context.Context, noSurat string, tgl time.Time) (*domain.Rujukan, error)
	CreateSEPFunc            func(ctx context.Context, req domain.SEPRequest) (*domain.SEP, error)
	CreateSEPKontrolFunc     func(ctx context.Context, req domain.SEPKontrolRequest) (*domain.SEP, error)
	CekSEPDuplikasiFunc      func(ctx context.Context, noKartu, tglSEP string) (*domain.SEP, error)
	CekFingerprintStatusFunc func(ctx context.Context, noKartu string, tgl time.Time) (*FingerprintStatus, error)

	mu        sync.Mutex
	callCount map[string]int
}

var _ VClaimClient = (*MockVClaimClient)(nil)

// NewMock membuat MockVClaimClient kosong dengan call counter ter-init.
func NewMock() *MockVClaimClient {
	return &MockVClaimClient{callCount: make(map[string]int)}
}

// CallCount mengembalikan jumlah pemanggilan method dengan nama tersebut.
// Nama method case-sensitive (mis. "GetPeserta", "CreateSEP").
func (m *MockVClaimClient) CallCount(method string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.callCount[method]
}

func (m *MockVClaimClient) recordCall(method string) {
	m.mu.Lock()
	m.callCount[method]++
	m.mu.Unlock()
}

func (m *MockVClaimClient) GetPeserta(ctx context.Context, identifier string, tgl time.Time) (*domain.Peserta, error) {
	m.recordCall("GetPeserta")
	if m.GetPesertaFunc != nil {
		return m.GetPesertaFunc(ctx, identifier, tgl)
	}
	return nil, nil
}

func (m *MockVClaimClient) GetRiwayatPelayanan(ctx context.Context, noKartu string, tglAwal, tglAkhir time.Time) ([]domain.RiwayatPelayanan, error) {
	m.recordCall("GetRiwayatPelayanan")
	if m.GetRiwayatPelayananFunc != nil {
		return m.GetRiwayatPelayananFunc(ctx, noKartu, tglAwal, tglAkhir)
	}
	return nil, nil
}

func (m *MockVClaimClient) ValidasiRujukan(ctx context.Context, noSurat string, tgl time.Time) (*domain.Rujukan, error) {
	m.recordCall("ValidasiRujukan")
	if m.ValidasiRujukanFunc != nil {
		return m.ValidasiRujukanFunc(ctx, noSurat, tgl)
	}
	return nil, nil
}

func (m *MockVClaimClient) CreateSEP(ctx context.Context, req domain.SEPRequest) (*domain.SEP, error) {
	m.recordCall("CreateSEP")
	if m.CreateSEPFunc != nil {
		return m.CreateSEPFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockVClaimClient) CreateSEPKontrol(ctx context.Context, req domain.SEPKontrolRequest) (*domain.SEP, error) {
	m.recordCall("CreateSEPKontrol")
	if m.CreateSEPKontrolFunc != nil {
		return m.CreateSEPKontrolFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockVClaimClient) CekSEPDuplikasi(ctx context.Context, noKartu, tglSEP string) (*domain.SEP, error) {
	m.recordCall("CekSEPDuplikasi")
	if m.CekSEPDuplikasiFunc != nil {
		return m.CekSEPDuplikasiFunc(ctx, noKartu, tglSEP)
	}
	return nil, nil
}

func (m *MockVClaimClient) CekFingerprintStatus(ctx context.Context, noKartu string, tgl time.Time) (*FingerprintStatus, error) {
	m.recordCall("CekFingerprintStatus")
	if m.CekFingerprintStatusFunc != nil {
		return m.CekFingerprintStatusFunc(ctx, noKartu, tgl)
	}
	// Default: belum verifikasi (frontend akan prompt modal).
	return &FingerprintStatus{Verified: false}, nil
}

