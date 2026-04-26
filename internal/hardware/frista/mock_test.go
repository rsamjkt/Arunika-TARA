package frista

import (
	"context"
	"testing"
	"time"
)

func TestMockReader_StartEmitConsume(t *testing.T) {
	r := NewMock(0)
	if err := r.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer r.Stop()

	want := CardData{
		NIK:      "3271234567890001",
		Nama:     "Budi Santoso",
		TglLahir: "1980-05-15",
		NoKartu:  "0001234567890012",
	}
	if err := r.EmitCard(want); err != nil {
		t.Fatalf("EmitCard: %v", err)
	}

	select {
	case got := <-r.CardRead():
		if got.NIK != want.NIK || got.NoKartu != want.NoKartu {
			t.Errorf("CardData mismatch: got %+v", got)
		}
		if got.Timestamp.IsZero() {
			t.Error("Timestamp seharusnya auto-fill")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout menunggu CardRead")
	}
}

func TestMockReader_EmitSebelumStart_Error(t *testing.T) {
	r := NewMock(0)
	if err := r.EmitCard(CardData{NIK: "X"}); err == nil {
		t.Fatal("EmitCard sebelum Start harus error")
	}
}

func TestMockReader_StopMenutupChannel(t *testing.T) {
	r := NewMock(0)
	r.Start(context.Background())
	r.Stop()

	// Channel harus closed — read return zero value + ok=false
	_, ok := <-r.CardRead()
	if ok {
		t.Error("channel seharusnya closed setelah Stop")
	}
	if r.IsAvailable() {
		t.Error("setelah Stop seharusnya tidak available")
	}
}

func TestMockReader_StopIdempotent(t *testing.T) {
	r := NewMock(0)
	r.Start(context.Background())
	if err := r.Stop(); err != nil {
		t.Fatalf("Stop pertama: %v", err)
	}
	if err := r.Stop(); err != nil {
		t.Fatalf("Stop kedua harus no-op, got: %v", err)
	}
}

func TestMockReader_ChannelPenuh_ErrorBackpressure(t *testing.T) {
	r := NewMock(0)
	r.Start(context.Background())
	defer r.Stop()

	// Buffer = 5, fill it up
	for i := 0; i < 5; i++ {
		if err := r.EmitCard(CardData{NIK: "X"}); err != nil {
			t.Fatalf("emit ke-%d: %v", i, err)
		}
	}
	// 6th emit should error (channel full)
	if err := r.EmitCard(CardData{NIK: "X"}); err == nil {
		t.Error("emit ke-6 ke channel penuh harus error")
	}
}

func TestMockReader_SetAvailable(t *testing.T) {
	r := NewMock(0)
	if !r.IsAvailable() {
		t.Error("default available")
	}
	r.SetAvailable(false)
	if r.IsAvailable() {
		t.Error("setelah SetAvailable(false) harus false")
	}
}

func TestMockReader_ServerPortAccessor(t *testing.T) {
	r := NewMock(9090)
	if r.ServerPort() != 9090 {
		t.Errorf("ServerPort = %d, want 9090", r.ServerPort())
	}
}
