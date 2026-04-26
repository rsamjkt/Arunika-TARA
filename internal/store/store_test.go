package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// loadSchema membaca migrations/001_initial.sql dari root project.
// Path dihitung relatif ke file test ini supaya bekerja apapun
// working directory test runner.
func loadSchema(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("tidak bisa resolve path test file")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	b, err := os.ReadFile(filepath.Join(root, "migrations", "001_initial.sql"))
	if err != nil {
		t.Fatalf("baca schema: %v", err)
	}
	return string(b)
}

// newTestDB membuka in-memory SQLite dengan schema sudah ter-apply.
func newTestDB(t *testing.T) (*sql.DB, *Queries) {
	t.Helper()
	ctx := context.Background()
	db, q, err := Open(ctx, ":memory:", loadSchema(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, q
}

// ============================================================
// Lifecycle: Open + schema apply
// ============================================================

func TestOpen_InMemory_SchemaApplied(t *testing.T) {
	db, _ := newTestDB(t)

	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' ORDER BY name`)
	if err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			t.Fatalf("scan: %v", err)
		}
		tables = append(tables, n)
	}

	expected := []string{"antrian_lokal", "config_cache", "pending_sep", "print_history", "reconcile_log"}
	for _, want := range expected {
		found := false
		for _, got := range tables {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tabel %q tidak ada di schema. Tables: %v", want, tables)
		}
	}
}

func TestOpen_SchemaKosong_TidakApply(t *testing.T) {
	ctx := context.Background()
	db, _, err := Open(ctx, ":memory:", "")
	if err != nil {
		t.Fatalf("Open dengan schema kosong harus tetap berhasil: %v", err)
	}
	defer db.Close()

	row := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='antrian_lokal'`)
	var n int
	if err := row.Scan(&n); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if n != 0 {
		t.Errorf("dengan schema kosong tidak boleh ada tabel — got count=%d", n)
	}
}

// ============================================================
// antrian_lokal: Insert -> GetPending -> MarkSynced
// ============================================================

func TestAntrian_FullLifecycle(t *testing.T) {
	_, q := newTestDB(t)
	ctx := context.Background()

	a, err := q.InsertAntrian(ctx, InsertAntrianParams{
		Jenis:    "POLI",
		SubJenis: sql.NullString{String: "WALKIN", Valid: true},
		Nomor:    "B-DALAM-015",
		Prefix:   "B",
		NoUrut:   15,
		NoRm:     sql.NullString{String: "RM-001", Valid: true},
		NoPoli:   sql.NullString{String: "DALAM", Valid: true},
	})
	if err != nil {
		t.Fatalf("InsertAntrian: %v", err)
	}
	if a.ID == 0 {
		t.Errorf("ID auto-increment harus > 0")
	}
	if a.SyncStatus.String != "pending" {
		t.Errorf("default sync_status harus 'pending', got %q", a.SyncStatus.String)
	}

	pending, err := q.GetPendingAntrian(ctx, 100)
	if err != nil {
		t.Fatalf("GetPendingAntrian: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}
	if pending[0].ID != a.ID {
		t.Errorf("pending ID mismatch: got %d, want %d", pending[0].ID, a.ID)
	}

	if err := q.MarkAntrianSynced(ctx, a.ID); err != nil {
		t.Fatalf("MarkAntrianSynced: %v", err)
	}

	after, err := q.GetPendingAntrian(ctx, 100)
	if err != nil {
		t.Fatalf("GetPendingAntrian after sync: %v", err)
	}
	if len(after) != 0 {
		t.Errorf("setelah MarkSynced harus tidak ada pending, got %d", len(after))
	}
}

func TestAntrian_MarkFailed(t *testing.T) {
	_, q := newTestDB(t)
	ctx := context.Background()

	a, err := q.InsertAntrian(ctx, InsertAntrianParams{
		Jenis: "LOKET", Nomor: "A-001", Prefix: "A", NoUrut: 1,
	})
	if err != nil {
		t.Fatalf("InsertAntrian: %v", err)
	}
	if err := q.MarkAntrianFailed(ctx, a.ID); err != nil {
		t.Fatalf("MarkAntrianFailed: %v", err)
	}
	pending, _ := q.GetPendingAntrian(ctx, 100)
	if len(pending) != 0 {
		t.Errorf("MarkFailed juga harus keluarkan dari pending, got %d", len(pending))
	}
}

// ============================================================
// pending_sep: Insert -> Confirm -> MarkSynced + retry tracking
// ============================================================

func TestPendingSEP_ConfirmAndSync(t *testing.T) {
	_, q := newTestDB(t)
	ctx := context.Background()

	sep, err := q.InsertPendingSEP(ctx, InsertPendingSEPParams{
		NoKartu:        "0001234567890012",
		Kategori:       "RUJUKAN",
		PayloadJson:    `{"noKartu":"0001234567890012","kdPoli":"INT"}`,
		VclaimResponse: sql.NullString{String: `{"noSep":"1234567890"}`, Valid: true},
	})
	if err != nil {
		t.Fatalf("InsertPendingSEP: %v", err)
	}
	if sep.Status.String != "pending" {
		t.Errorf("default status harus 'pending', got %q", sep.Status.String)
	}

	pending, err := q.GetPendingSEPs(ctx, GetPendingSEPsParams{
		Status: sql.NullString{String: "pending", Valid: true},
		Limit:  100,
	})
	if err != nil {
		t.Fatalf("GetPendingSEPs: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending SEP, got %d", len(pending))
	}

	// Operator konfirmasi
	if err := q.ConfirmSEP(ctx, ConfirmSEPParams{
		ConfirmedBy: sql.NullString{String: "operator-01", Valid: true},
		ID:          sep.ID,
	}); err != nil {
		t.Fatalf("ConfirmSEP: %v", err)
	}

	// Setelah confirm, status pindah ke 'awaiting_sync'
	awaiting, err := q.GetPendingSEPs(ctx, GetPendingSEPsParams{
		Status: sql.NullString{String: "awaiting_sync", Valid: true},
		Limit:  100,
	})
	if err != nil {
		t.Fatalf("GetPendingSEPs awaiting: %v", err)
	}
	if len(awaiting) != 1 {
		t.Fatalf("expected 1 awaiting_sync, got %d", len(awaiting))
	}
	if awaiting[0].ConfirmedBy.String != "operator-01" {
		t.Errorf("ConfirmedBy salah: %q", awaiting[0].ConfirmedBy.String)
	}

	// Reconcile worker tandai synced
	if err := q.MarkSEPSynced(ctx, sep.ID); err != nil {
		t.Fatalf("MarkSEPSynced: %v", err)
	}

	synced, _ := q.GetPendingSEPs(ctx, GetPendingSEPsParams{
		Status: sql.NullString{String: "synced", Valid: true},
		Limit:  100,
	})
	if len(synced) != 1 {
		t.Errorf("expected 1 synced SEP, got %d", len(synced))
	}
}

func TestPendingSEP_RetryCounter(t *testing.T) {
	_, q := newTestDB(t)
	ctx := context.Background()

	sep, err := q.InsertPendingSEP(ctx, InsertPendingSEPParams{
		NoKartu:     "0001",
		Kategori:    "KONTROL",
		PayloadJson: `{}`,
	})
	if err != nil {
		t.Fatalf("InsertPendingSEP: %v", err)
	}

	// 3x increment retry
	for i := 0; i < 3; i++ {
		if err := q.IncrementSEPRetry(ctx, IncrementSEPRetryParams{
			LastError: sql.NullString{String: "timeout", Valid: true},
			ID:        sep.ID,
		}); err != nil {
			t.Fatalf("IncrementSEPRetry attempt %d: %v", i, err)
		}
	}

	rows, _ := q.GetPendingSEPs(ctx, GetPendingSEPsParams{
		Status: sql.NullString{String: "pending", Valid: true},
		Limit:  10,
	})
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	if rows[0].RetryCount.Int64 != 3 {
		t.Errorf("retry_count = %d, want 3", rows[0].RetryCount.Int64)
	}
	if rows[0].LastError.String != "timeout" {
		t.Errorf("last_error = %q, want 'timeout'", rows[0].LastError.String)
	}
}

// ============================================================
// print_history: Insert -> Get -> IncrementReprint
// ============================================================

func TestPrintHistory_LifecycleAndReprint(t *testing.T) {
	_, q := newTestDB(t)
	ctx := context.Background()

	bytes := []byte{0x1B, 0x40, 0x42, 0x55, 0x44, 0x49} // dummy ESC/POS
	ph, err := q.InsertPrintHistory(ctx, InsertPrintHistoryParams{
		DocType:     "TIKET",
		RefID:       sql.NullString{String: "ticket-123", Valid: true},
		EscposBytes: bytes,
	})
	if err != nil {
		t.Fatalf("InsertPrintHistory: %v", err)
	}
	if ph.ReprintCount.Int64 != 0 {
		t.Errorf("default reprint_count harus 0, got %d", ph.ReprintCount.Int64)
	}

	got, err := q.GetPrintHistory(ctx, ph.ID)
	if err != nil {
		t.Fatalf("GetPrintHistory: %v", err)
	}
	if string(got.EscposBytes) != string(bytes) {
		t.Errorf("escpos_bytes round-trip gagal")
	}

	byRef, err := q.GetPrintHistoryByRefID(ctx, GetPrintHistoryByRefIDParams{
		DocType: "TIKET",
		RefID:   sql.NullString{String: "ticket-123", Valid: true},
	})
	if err != nil {
		t.Fatalf("GetPrintHistoryByRefID: %v", err)
	}
	if byRef.ID != ph.ID {
		t.Errorf("lookup by ref_id mismatch")
	}

	// 2x reprint
	for i := 0; i < 2; i++ {
		if err := q.IncrementReprintCount(ctx, ph.ID); err != nil {
			t.Fatalf("IncrementReprintCount: %v", err)
		}
	}
	after, _ := q.GetPrintHistory(ctx, ph.ID)
	if after.ReprintCount.Int64 != 2 {
		t.Errorf("reprint_count = %d, want 2", after.ReprintCount.Int64)
	}
}

// ============================================================
// reconcile_log: Insert -> GetRecent
// ============================================================

func TestReconcileLog_InsertAndQuery(t *testing.T) {
	_, q := newTestDB(t)
	ctx := context.Background()

	// Insert 3 log entry
	for i := 0; i < 3; i++ {
		_, err := q.InsertReconcileLog(ctx, InsertReconcileLogParams{
			TableName:  "antrian_lokal",
			RecordID:   int64(i + 1),
			Action:     "SYNC_SUCCESS",
			OperatorID: sql.NullString{String: "system", Valid: true},
			Result:     sql.NullString{String: "OK", Valid: true},
		})
		if err != nil {
			t.Fatalf("InsertReconcileLog #%d: %v", i, err)
		}
	}

	logs, err := q.GetRecentLogs(ctx, 10)
	if err != nil {
		t.Fatalf("GetRecentLogs: %v", err)
	}
	if len(logs) != 3 {
		t.Errorf("expected 3 logs, got %d", len(logs))
	}
	// Ordering: paling baru dulu (timestamp DESC) — record_id 3 di awal
	if logs[0].RecordID != 3 {
		t.Errorf("urutan salah — log pertama harus record_id=3, got %d", logs[0].RecordID)
	}

	// Limit harus dihormati
	limited, _ := q.GetRecentLogs(ctx, 2)
	if len(limited) != 2 {
		t.Errorf("limit=2 harus return 2 row, got %d", len(limited))
	}
}
