package frista

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestMockVerifier_VerifySuccess(t *testing.T) {
	m := NewMock()
	m.SetScanDelay(0) // test cepat

	res, err := m.Verify(context.Background(), "0001234567890012")
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !res.Success {
		t.Errorf("Success = false, want true")
	}
	if res.Token == "" {
		t.Errorf("Token kosong")
	}
	if !strings.Contains(res.Token, "MOCK_FACE") {
		t.Errorf("Token harus mengandung MOCK_FACE prefix, got %q", res.Token)
	}
	if !strings.Contains(res.Token, "0001234567890012") {
		t.Errorf("Token harus include noPeserta, got %q", res.Token)
	}
	if res.Timestamp.IsZero() {
		t.Errorf("Timestamp seharusnya auto-fill")
	}
}

func TestMockVerifier_SetNextFail(t *testing.T) {
	m := NewMock()
	m.SetScanDelay(0)
	m.SetNextFail()

	// Verify pertama harus gagal
	_, err := m.Verify(context.Background(), "X")
	if err == nil {
		t.Fatal("expected error setelah SetNextFail")
	}
	if !strings.Contains(err.Error(), "simulasi") {
		t.Errorf("error message harus mention simulasi, got: %v", err)
	}

	// Verify kedua harus sukses lagi (flag direset setelah 1 fail)
	res, err := m.Verify(context.Background(), "X")
	if err != nil {
		t.Errorf("Verify kedua harus sukses, got: %v", err)
	}
	if !res.Success {
		t.Errorf("Success setelah fail-once seharusnya kembali true")
	}
}

func TestMockVerifier_ContextCancel(t *testing.T) {
	m := NewMock()
	m.SetScanDelay(2 * time.Second) // delay panjang

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := m.Verify(ctx, "X")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error karena ctx cancel")
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("Verify tidak respect ctx.Done() cepat, elapsed=%v", elapsed)
	}
}

func TestMockVerifier_SetAvailable(t *testing.T) {
	m := NewMock()
	if !m.IsAvailable() {
		t.Error("default available")
	}
	m.SetAvailable(false)
	if m.IsAvailable() {
		t.Error("setelah SetAvailable(false) harus false")
	}
}

func TestMockVerifier_TokenUnik(t *testing.T) {
	m := NewMock()
	m.SetScanDelay(0)

	// Token harus include timestamp HHMMSS — 2 verify dalam waktu
	// berbeda harus hasilkan token berbeda. Pakai sleep 1.1s untuk
	// memastikan format HHMMSS berubah (resolusi detik).
	r1, _ := m.Verify(context.Background(), "X")
	time.Sleep(1100 * time.Millisecond)
	r2, _ := m.Verify(context.Background(), "X")

	if r1.Token == r2.Token {
		t.Errorf("token seharusnya unik antar Verify (timestamp HHMMSS)")
	}
}

func TestMockVerifier_InterfaceCompliance(t *testing.T) {
	// Compile-time assertion juga ada di mock.go (`var _ FaceVerifier = ...`).
	// Test ini double-check supaya kalau interface berubah signature, test
	// gagal di sini lebih dulu (dan eror message-nya jelas).
	var v FaceVerifier = NewMock()
	if v == nil {
		t.Fatal("NewMock seharusnya return non-nil")
	}
	if !v.IsAvailable() {
		t.Error("default IsAvailable harus true")
	}
}
