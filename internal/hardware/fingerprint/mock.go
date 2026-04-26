package fingerprint

import (
	"context"
	"errors"
	"sync"
	"time"
)

// MockVerifier adalah implementasi FingerprintVerifier untuk
// development di Mac/Linux. Default behaviour: setelah delay 2 detik
// return success dengan token dummy.
//
// Kontrol untuk test failure:
//
//	mock.SetNextFail()  // satu kali — Verify() berikutnya gagal
//
// Mock juga listen ke ctx.Done() supaya cancel cepat.
type MockVerifier struct {
	mu            sync.Mutex
	forceFailNext bool

	// scanDelay default 2 detik — bisa di-set ke 0 di test untuk
	// menghindari sleep panjang.
	scanDelay time.Duration

	// available di-set ke true; bisa di-toggle test untuk simulasi
	// hardware tidak ada.
	available bool
}

var _ FingerprintVerifier = (*MockVerifier)(nil)

// NewMock membangun MockVerifier dengan default setting development.
func NewMock() *MockVerifier {
	return &MockVerifier{
		scanDelay: 2 * time.Second,
		available: true,
	}
}

// SetNextFail menandai Verify() berikutnya untuk return error.
// Setelah satu pemanggilan Verify(), flag di-reset.
// Dipakai dari mock HTTP server endpoint /mock/fp-fail.
func (m *MockVerifier) SetNextFail() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forceFailNext = true
}

// SetScanDelay mengganti delay simulasi scan. Dipakai test untuk
// scanDelay = 0 supaya test cepat.
func (m *MockVerifier) SetScanDelay(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scanDelay = d
}

// SetAvailable mengganti hasil IsAvailable() — simulasi hardware
// tercabut atau service mati.
func (m *MockVerifier) SetAvailable(v bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.available = v
}

func (m *MockVerifier) IsAvailable() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.available
}

// Verify simulasi scan: tunggu scanDelay lalu return success
// (atau error kalau forceFailNext di-set).
func (m *MockVerifier) Verify(ctx context.Context, noPeserta string) (FPResult, error) {
	m.mu.Lock()
	fail := m.forceFailNext
	m.forceFailNext = false
	delay := m.scanDelay
	m.mu.Unlock()

	if fail {
		return FPResult{}, errors.New("simulasi: verifikasi sidik jari gagal")
	}

	select {
	case <-time.After(delay):
		return FPResult{
			Success:   true,
			Token:     "MOCK_FP_" + noPeserta + "_" + time.Now().Format("150405"),
			Timestamp: time.Now(),
		}, nil
	case <-ctx.Done():
		return FPResult{}, ctx.Err()
	}
}
