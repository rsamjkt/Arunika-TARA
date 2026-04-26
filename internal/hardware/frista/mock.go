package frista

import (
	"context"
	"errors"
	"sync"
	"time"
)

// MockReader adalah implementasi development di Mac/Linux. Channel
// CardRead bersifat reactive — test (atau HTTP mock server di P-031)
// bisa panggil EmitCard(...) untuk menyimulasi tap kartu.
//
// Lifecycle:
//
//	r := NewMock()
//	r.Start(ctx)            // siapkan channel
//	go func() {
//	    for c := range r.CardRead() { ... }
//	}()
//	r.EmitCard(CardData{...}) // simulasi tap
//	r.Stop()                  // tutup channel
//
// Thread-safe — beberapa goroutine boleh EmitCard bersamaan.
type MockReader struct {
	mu        sync.Mutex
	ch        chan CardData
	started   bool
	available bool

	// serverPort menyimpan port HTTP mock yang dipakai P-031 (saat
	// MockReader di-wrap dengan endpoint /mock/card-read). Belum
	// dipakai di P-030 — disiapkan supaya signature New stabil.
	serverPort int
}

var _ CardReader = (*MockReader)(nil)

// NewMock membuat reader baru. serverPort di-store untuk dipakai
// HTTP wrapper P-031 (saat ini tidak start server apapun).
func NewMock(serverPort int) *MockReader {
	return &MockReader{
		ch:         make(chan CardData, 5),
		available:  true,
		serverPort: serverPort,
	}
}

// Start menandai reader aktif. Channel sudah ter-buffer dari
// constructor sehingga EmitCard sebelum Start juga aman (event akan
// di-buffer sampai 5 item).
func (m *MockReader) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		return nil
	}
	m.started = true
	return nil
}

// Stop menutup channel — pembaca via CardRead akan keluar dari
// for-range loop. Idempotent.
func (m *MockReader) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.started {
		return nil
	}
	close(m.ch)
	m.started = false
	m.available = false
	return nil
}

func (m *MockReader) IsAvailable() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.available
}

func (m *MockReader) CardRead() <-chan CardData {
	return m.ch
}

// EmitCard mendorong satu CardData ke channel. Thread-safe; jika
// channel penuh (5 item belum dikonsumsi), kembalikan error agar
// caller bisa decide back-pressure (mis. drop atau retry).
func (m *MockReader) EmitCard(c CardData) error {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return errors.New("frista mock: belum di-Start")
	}
	m.mu.Unlock()

	if c.Timestamp.IsZero() {
		c.Timestamp = time.Now()
	}
	select {
	case m.ch <- c:
		return nil
	default:
		return errors.New("frista mock: channel penuh (consumer lambat)")
	}
}

// SetAvailable di-pakai test untuk simulasi reader tercabut.
func (m *MockReader) SetAvailable(v bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.available = v
}

// ServerPort accessor — dipakai HTTP wrapper P-031.
func (m *MockReader) ServerPort() int {
	return m.serverPort
}
