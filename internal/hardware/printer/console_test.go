package printer

import (
	"bytes"
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/arunika/apm-go/internal/store"
)

// loadSchema baca migration dari project root.
func loadSchema(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("tidak bisa resolve test file")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	b, err := os.ReadFile(filepath.Join(root, "migrations", "001_initial.sql"))
	if err != nil {
		t.Fatalf("baca schema: %v", err)
	}
	return string(b)
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, _, err := store.Open(context.Background(), ":memory:", loadSchema(t))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// ============================================================
// Print
// ============================================================

func TestConsolePrinter_Print_RendersDocTypeAndData(t *testing.T) {
	p := NewConsolePrinter(nil)
	var buf bytes.Buffer
	p.SetWriter(&buf)

	type tiketData struct {
		Nomor string
		Poli  string
	}
	err := p.Print(context.Background(), "TIKET", tiketData{Nomor: "B-INT-005", Poli: "INT"})
	if err != nil {
		t.Fatalf("Print: %v", err)
	}

	out := buf.String()
	for _, sub := range []string{"TIKET", "B-INT-005", "INT", "[CETAK]"} {
		if !strings.Contains(out, sub) {
			t.Errorf("output tidak mengandung %q:\n%s", sub, out)
		}
	}
}

func TestConsolePrinter_Print_TidakAvailable_Error(t *testing.T) {
	p := NewConsolePrinter(nil)
	p.SetAvailable(false)

	err := p.Print(context.Background(), "TIKET", nil)
	if err == nil {
		t.Fatal("printer tidak available harus error")
	}
}

// ============================================================
// Reprint
// ============================================================

func TestConsolePrinter_Reprint_LoadFromStoreAndIncrement(t *testing.T) {
	db := newTestDB(t)
	q := store.New(db)
	ctx := context.Background()

	// Seed print_history
	payload := []byte(`{"nomor":"B-INT-005"}`)
	h, err := q.InsertPrintHistory(ctx, store.InsertPrintHistoryParams{
		DocType:     "TIKET",
		RefID:       sql.NullString{String: "ticket-1", Valid: true},
		EscposBytes: payload,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	p := NewConsolePrinter(db)
	var buf bytes.Buffer
	p.SetWriter(&buf)

	if err := p.Reprint(ctx, h.ID); err != nil {
		t.Fatalf("Reprint: %v", err)
	}

	out := buf.String()
	for _, sub := range []string{"REPRINT", "TIKET", `"nomor":"B-INT-005"`} {
		if !strings.Contains(out, sub) {
			t.Errorf("output tidak mengandung %q:\n%s", sub, out)
		}
	}

	// Increment ter-persist
	got, _ := q.GetPrintHistory(ctx, h.ID)
	if got.ReprintCount.Int64 != 1 {
		t.Errorf("reprint_count = %d, want 1", got.ReprintCount.Int64)
	}
}

func TestConsolePrinter_Reprint_TanpaStore_Error(t *testing.T) {
	p := NewConsolePrinter(nil) // tanpa db
	if err := p.Reprint(context.Background(), 1); err == nil {
		t.Fatal("Reprint tanpa store harus error")
	}
}

func TestConsolePrinter_Reprint_IDTidakAda_Error(t *testing.T) {
	db := newTestDB(t)
	p := NewConsolePrinter(db)
	if err := p.Reprint(context.Background(), 999999); err == nil {
		t.Fatal("Reprint id tidak ada harus error")
	}
}

