package khanza

import (
	"context"
	"sync"

	"github.com/arunika/apm-go/internal/domain"
)

// MockKhanzaClient adalah implementasi KhanzaClient untuk testing.
type MockKhanzaClient struct {
	GetSuratKontrolFunc   func(ctx context.Context, noRM string) ([]domain.SuratKontrol, error)
	GetRiwayatRANAPFunc   func(ctx context.Context, noRM string) ([]domain.RiwayatRANAP, error)
	GetKunjunganAktifFunc func(ctx context.Context, noRM string) ([]domain.Kunjungan, error)

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
