package frista

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
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

	// serverPort menentukan apakah HTTP mock server akan di-spawn:
	//   port == 0  → tidak ada HTTP server (channel-only mock,
	//                pattern P-030 untuk unit test ringan)
	//   port  > 0  → bind HTTP server di port tsb saat Start()
	serverPort int
	server     *http.Server
	serverAddr string // diisi setelah Start kalau ada server

	// onFPFailRequest dipanggil oleh handler POST /mock/fp-fail.
	// Hook callback design — frista mock TIDAK import fingerprint
	// package; Wails app yang wire callback (mis. fpMock.SetNextFail)
	// supaya separation of concerns terjaga.
	onFPFailRequest func()

	logger *slog.Logger
}

var _ CardReader = (*MockReader)(nil)

// NewMock membuat reader baru.
//
//	port == 0 : tidak ada HTTP server (channel-only — pattern P-030)
//	port  > 0 : Start() akan bind HTTP server di "127.0.0.1:port"
//	            dengan endpoint mock POST /mock/card-read & GET /
func NewMock(serverPort int) *MockReader {
	return &MockReader{
		ch:         make(chan CardData, 5),
		available:  true,
		serverPort: serverPort,
		logger:     slog.Default(),
	}
}

// SetLogger mengganti logger (dipakai test atau caller dengan PHIMaskingHandler).
func (m *MockReader) SetLogger(l *slog.Logger) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l != nil {
		m.logger = l
	}
}

// SetOnFPFail mendaftarkan callback yang dipanggil saat HTTP request
// POST /mock/fp-fail diterima. Wails app saat init pasok:
//
//	fristaMock.SetOnFPFail(fpMock.SetNextFail)
//
// supaya developer bisa pakai `make mock-fp-fail` untuk simulasi
// fingerprint gagal sekali. Boleh nil — handler akan return 200 dengan
// warning kalau belum ada hook terdaftar.
func (m *MockReader) SetOnFPFail(cb func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onFPFailRequest = cb
}

// Start menandai reader aktif. Channel sudah ter-buffer dari
// constructor. Jika serverPort > 0, juga bind HTTP server di
// 127.0.0.1:port — terikat-localhost untuk safety (kiosk dev,
// JANGAN expose ke network supaya tidak bisa di-trigger orang lain).
func (m *MockReader) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.started {
		return nil
	}
	m.started = true

	if m.serverPort > 0 {
		if err := m.startHTTPLocked(ctx); err != nil {
			m.started = false
			return err
		}
	}
	return nil
}

// Stop menutup channel + graceful shutdown HTTP server (jika ada).
// Idempotent — boleh dipanggil berkali-kali.
func (m *MockReader) Stop() error {
	m.mu.Lock()
	srv := m.server
	m.server = nil
	if !m.started {
		m.mu.Unlock()
		return nil
	}
	close(m.ch)
	m.started = false
	m.available = false
	m.mu.Unlock()

	if srv != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}
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

// ServerPort accessor — port yang diminta saat NewMock (configured port).
func (m *MockReader) ServerPort() int {
	return m.serverPort
}

// HTTPAddr mengembalikan address HTTP server yang aktif (mis.
// "127.0.0.1:9090") atau "" jika server tidak running. Dipakai test
// & log info untuk tahu URL yang bisa di-curl.
func (m *MockReader) HTTPAddr() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.serverAddr
}
