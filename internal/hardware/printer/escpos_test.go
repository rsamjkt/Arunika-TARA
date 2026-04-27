package printer

import (
	"bytes"
	"context"
	"database/sql"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arunika/apm-go/internal/config"
	"github.com/arunika/apm-go/internal/store"
)

// ============================================================
// encodeESCPOS — bytes structure correctness
// ============================================================

func TestEncodeESCPOS_DimulaiResetDanDiakhiriCut(t *testing.T) {
	bytes := encodeESCPOS("TIKET", "Hello\n")

	if len(bytes) < 6 {
		t.Fatalf("output terlalu pendek: %d", len(bytes))
	}
	// Awal: ESC @ (0x1B 0x40)
	if bytes[0] != 0x1B || bytes[1] != 0x40 {
		t.Errorf("output tidak dimulai dengan ESC @: %v", bytes[:4])
	}
	// Akhir: GS V 0 (0x1D 0x56 0x00)
	last3 := bytes[len(bytes)-3:]
	if last3[0] != 0x1D || last3[1] != 0x56 || last3[2] != 0x00 {
		t.Errorf("output tidak diakhiri GS V 0 (cut): %v", last3)
	}
}

func TestEncodeESCPOS_ContainsBodyText(t *testing.T) {
	body := "No. SEP: 123456789"
	bytes := encodeESCPOS("SEP", body)
	if !strings.Contains(string(bytes), body) {
		t.Errorf("body text tidak ada di output")
	}
}

func TestEncodeESCPOS_MarkerCenter_DiSubstitusiKeESCByte(t *testing.T) {
	bytes := encodeESCPOS("X", "[C]CENTERED[/C]\nbody")
	// Marker [C] harus di-substitusi ke ESC a 1, [/C] ke ESC a 0
	if !strings.Contains(string(bytes), string([]byte{0x1B, 0x61, 0x01})) {
		t.Errorf("[C] tidak di-substitusi ke ESC a 1 (center align)")
	}
	if !strings.Contains(string(bytes), string([]byte{0x1B, 0x61, 0x00})) {
		t.Errorf("[/C] tidak di-substitusi ke ESC a 0 (left align)")
	}
	// Marker text tidak boleh muncul di output
	if strings.Contains(string(bytes), "[C]") || strings.Contains(string(bytes), "[/C]") {
		t.Errorf("marker [C]/[/C] tidak boleh tersisa di output")
	}
}

func TestEncodeESCPOS_MarkerBold_DiSubstitusi(t *testing.T) {
	bytes := encodeESCPOS("X", "[B]BOLD[/B]")
	if !strings.Contains(string(bytes), string([]byte{0x1B, 0x45, 0x01})) {
		t.Errorf("[B] tidak di-substitusi ke ESC E 1 (bold on)")
	}
	if !strings.Contains(string(bytes), string([]byte{0x1B, 0x45, 0x00})) {
		t.Errorf("[/B] tidak di-substitusi ke ESC E 0 (bold off)")
	}
}

func TestEncodeESCPOS_MarkerXL_DiSubstitusi(t *testing.T) {
	bytes := encodeESCPOS("X", "[XL]A001[/XL]")
	if !strings.Contains(string(bytes), string([]byte{0x1D, 0x21, 0x33})) {
		t.Errorf("[XL] tidak di-substitusi ke GS ! 0x33 (quad size)")
	}
}

func TestEncodeESCPOS_FooterAdaCutPaper(t *testing.T) {
	bytes := encodeESCPOS("X", "body")
	if !strings.Contains(string(bytes), string([]byte{0x1D, 0x56, 0x00})) {
		t.Errorf("cut paper command (GS V 0) tidak ada di footer")
	}
}


// ============================================================
// ESCPOSPrinter — Print + connection (pakai mode="console" untuk test)
// ============================================================

func TestESCPOSPrinter_Print_TulisKeStdout_BytesValid(t *testing.T) {
	cfg := config.PrinterConfig{Mode: "console"}
	p := NewESCPOSPrinter(cfg, nil)

	// Capture stdout via pipe — kita TIDAK bisa setWriter langsung
	// karena ESCPOSPrinter pakai stdoutConn{} yang hardcoded ke os.Stdout.
	// Untuk test bytes, kita verify Print tidak error + IsAvailable
	err := p.Print(context.Background(), "TIKET", map[string]any{
		"RSName": "X", "Tanggal": "X", "JenisAntrian": "X", "Nomor": "A-001",
	})
	if err != nil {
		t.Errorf("Print: %v", err)
	}
}

func TestESCPOSPrinter_Print_TidakAvailable_Error(t *testing.T) {
	p := NewESCPOSPrinter(config.PrinterConfig{Mode: "console"}, nil)
	p.SetAvailable(false)

	err := p.Print(context.Background(), "TIKET", nil)
	if err == nil {
		t.Fatal("printer tidak available harus error")
	}
}

func TestESCPOSPrinter_Print_DocTypeUnknown_Error(t *testing.T) {
	p := NewESCPOSPrinter(config.PrinterConfig{Mode: "console"}, nil)

	err := p.Print(context.Background(), "TIDAK_ADA", nil)
	if err == nil {
		t.Fatal("docType unknown harus error (ESCPOS strict, beda dari Console fallback JSON)")
	}
}

// ============================================================
// ESCPOSPrinter Network mode dengan TCP server fake
// ============================================================

func TestESCPOSPrinter_Network_KirimByteKeFakeTCP(t *testing.T) {
	// Spawn TCP listener yang collect bytes incoming
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()

	var (
		mu       sync.Mutex
		received []byte
		done     = make(chan struct{})
	)
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 4096)
		n, _ := conn.Read(buf)
		mu.Lock()
		received = append(received, buf[:n]...)
		mu.Unlock()
		close(done)
	}()

	cfg := config.PrinterConfig{
		Mode: "escpos_network",
		Port: l.Addr().String(),
	}
	p := NewESCPOSPrinter(cfg, nil)
	defer p.Stop()

	if err := p.Print(context.Background(), "TIKET", map[string]any{
		"RSName": "RS Test", "Tanggal": "2026-04-26",
		"JenisAntrian": "POLI", "Nomor": "B-001",
	}); err != nil {
		t.Fatalf("Print: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("TCP listener tidak menerima bytes dalam 2s")
	}

	mu.Lock()
	got := received
	mu.Unlock()

	if len(got) < 6 {
		t.Fatalf("received bytes terlalu pendek: %d", len(got))
	}
	// Verify ESC @ di awal
	if got[0] != 0x1B || got[1] != 0x40 {
		t.Errorf("bytes tidak dimulai ESC @")
	}
	// Verify cut command di akhir
	if !bytes.Contains(got, []byte{0x1D, 0x56, 0x00}) {
		t.Errorf("bytes tidak mengandung GS V 0 (cut)")
	}
	// Verify content
	if !bytes.Contains(got, []byte("RS Test")) {
		t.Errorf("bytes tidak mengandung RS Test")
	}
}

func TestESCPOSPrinter_Network_PortKosong_Error(t *testing.T) {
	cfg := config.PrinterConfig{Mode: "escpos_network", Port: ""}
	p := NewESCPOSPrinter(cfg, nil)
	err := p.Print(context.Background(), "TIKET", map[string]any{
		"RSName": "X", "Tanggal": "X", "JenisAntrian": "X", "Nomor": "X",
	})
	if err == nil {
		t.Fatal("port kosong harus error")
	}
}

func TestESCPOSPrinter_Network_PortTidakReachable_Error(t *testing.T) {
	cfg := config.PrinterConfig{Mode: "escpos_network", Port: "127.0.0.1:1"} // port 1 = reserved
	p := NewESCPOSPrinter(cfg, nil)
	err := p.Print(context.Background(), "TIKET", map[string]any{
		"RSName": "X", "Tanggal": "X", "JenisAntrian": "X", "Nomor": "X",
	})
	if err == nil {
		t.Fatal("port unreachable harus error")
	}
}

// ============================================================
// Mode unknown
// ============================================================

func TestOpenESCPOSConn_ModeUnknown_Error(t *testing.T) {
	_, err := openESCPOSConn(config.PrinterConfig{Mode: "unknown_mode"})
	if err == nil {
		t.Fatal("mode unknown harus error")
	}
	if !strings.Contains(err.Error(), "tidak dikenal") {
		t.Errorf("error message harus jelas: %v", err)
	}
}

// ============================================================
// Reprint
// ============================================================

func TestESCPOSPrinter_Reprint_LoadDanWriteByteSamaPersis(t *testing.T) {
	// Setup TCP listener
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()

	var (
		mu       sync.Mutex
		received []byte
	)
	done := make(chan struct{})
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf := make([]byte, 4096)
		n, _ := conn.Read(buf)
		mu.Lock()
		received = append(received, buf[:n]...)
		mu.Unlock()
		close(done)
	}()

	// Setup DB + seed print_history
	db := newTestDB(t)
	q := store.New(db)
	originalBytes := []byte{0x1B, 0x40, 'H', 'e', 'l', 'l', 'o', 0x0A, 0x1D, 0x56, 0x00}
	h, err := q.InsertPrintHistory(context.Background(), store.InsertPrintHistoryParams{
		DocType:     "TIKET",
		RefID:       sql.NullString{String: "ticket-1", Valid: true},
		EscposBytes: originalBytes,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	cfg := config.PrinterConfig{Mode: "escpos_network", Port: l.Addr().String()}
	p := NewESCPOSPrinter(cfg, db)
	defer p.Stop()

	if err := p.Reprint(context.Background(), h.ID); err != nil {
		t.Fatalf("Reprint: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("TCP listener tidak menerima bytes")
	}

	mu.Lock()
	got := received
	mu.Unlock()

	// Reprint harus kirim bytes-perfect copy (TIDAK re-encode)
	if !bytes.Equal(got, originalBytes) {
		t.Errorf("reprint bytes tidak match original\n  got=%v\n  want=%v",
			got, originalBytes)
	}

	// Counter harus increment
	after, _ := q.GetPrintHistory(context.Background(), h.ID)
	if after.ReprintCount.Int64 != 1 {
		t.Errorf("reprint_count = %d, want 1", after.ReprintCount.Int64)
	}
}

func TestESCPOSPrinter_Reprint_TanpaStore_Error(t *testing.T) {
	p := NewESCPOSPrinter(config.PrinterConfig{Mode: "console"}, nil)
	if err := p.Reprint(context.Background(), 1); err == nil {
		t.Fatal("Reprint tanpa store harus error")
	}
}

func TestESCPOSPrinter_Stop_Idempotent(t *testing.T) {
	p := NewESCPOSPrinter(config.PrinterConfig{Mode: "console"}, nil)
	if err := p.Stop(); err != nil {
		t.Errorf("Stop pertama: %v", err)
	}
	if err := p.Stop(); err != nil {
		t.Errorf("Stop kedua harus no-op: %v", err)
	}
}

// ============================================================
// NewESCPOS factory — return ThermalPrinter interface
// ============================================================

func TestNewESCPOS_ReturnThermalPrinterInterface(t *testing.T) {
	// Verify NewESCPOS satisfies ThermalPrinter interface at compile time
	// + runtime non-nil. Compile-time check via package-level _ var di
	// escpos.go (sudah ada). Di sini cuma runtime check.
	p := NewESCPOS(config.PrinterConfig{Mode: "console"}, nil)
	if p == nil {
		t.Fatal("NewESCPOS return nil")
	}
	if !p.IsAvailable() {
		t.Error("default available")
	}
}
