package frista

import (
	"context"
	"errors"
	"sync"
	"time"
)

// MockVerifier adalah implementasi FaceVerifier untuk development di
// Mac/Linux. Default behaviour: setelah delay 1.5 detik return success
// dengan token dummy "MOCK_FACE_<noPeserta>_<HHMMSS>".
//
// Kontrol untuk test failure:
//
//	mock.SetNextFail()  // satu kali — Verify() berikutnya gagal
//
// Mock juga listen ke ctx.Done() supaya cancel cepat (pattern sama
// persis dengan fingerprint.MockVerifier — supaya wails mock UX
// konsisten antar biometrik).
type MockVerifier struct {
	mu            sync.Mutex
	forceFailNext bool

	// scanDelay default 1.5 detik — bisa di-set ke 0 di test untuk
	// menghindari sleep panjang. Frista face di production juga relatif
	// cepat (~1-2 detik) jadi 1.5s terasa realistic untuk dev.
	scanDelay time.Duration

	// available di-set ke true; bisa di-toggle test untuk simulasi
	// hardware tidak ada / Frista belum login.
	available bool
}

var _ FaceVerifier = (*MockVerifier)(nil)

// NewMock membangun MockVerifier dengan default setting development.
func NewMock() *MockVerifier {
	return &MockVerifier{
		scanDelay: 1500 * time.Millisecond,
		available: true,
	}
}

// SetNextFail menandai Verify() berikutnya untuk return error.
// Setelah satu pemanggilan Verify(), flag di-reset.
//
// Dipakai test atau dev tooling (mis. shortcut Makefile di P-031+
// "make mock-face-fail") supaya developer bisa exercise error path
// tanpa harus utak-atik kamera / hardware.
func (m *MockVerifier) SetNextFail() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.forceFailNext = true
}

// SetScanDelay mengganti delay simulasi scan. Test biasanya pakai 0
// supaya tidak block.
func (m *MockVerifier) SetScanDelay(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scanDelay = d
}

// SetAvailable mengganti hasil IsAvailable() — simulasi Frista crash
// atau service mati.
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

// Verify simulasi scan wajah: tunggu scanDelay lalu return success
// (atau error kalau forceFailNext di-set).
//
// Token format: "MOCK_FACE_<noPeserta>_<HHMMSS>" — include timestamp
// supaya 2 panggilan Verify dalam waktu berbeda hasilkan token unik
// (memudahkan debugging via log).
func (m *MockVerifier) Verify(ctx context.Context, noPeserta string) (FRResult, error) {
	m.mu.Lock()
	fail := m.forceFailNext
	m.forceFailNext = false
	delay := m.scanDelay
	m.mu.Unlock()

	if fail {
		return FRResult{}, errors.New("simulasi: verifikasi sidik wajah gagal")
	}

	select {
	case <-time.After(delay):
		return FRResult{
			Success:   true,
			Token:     "MOCK_FACE_" + noPeserta + "_" + time.Now().Format("150405"),
			Timestamp: time.Now(),
		}, nil
	case <-ctx.Done():
		return FRResult{}, ctx.Err()
	}
}
