package reconcile

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arunika/apm-go/internal/domain"
	"github.com/arunika/apm-go/internal/integration/khanza"
	"github.com/arunika/apm-go/internal/store"
)

func loadSchema(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test path")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
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
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// seedAntrian helper: insert N pending antrian rows
func seedAntrian(t *testing.T, db *sql.DB, n int, jenis string) []int64 {
	t.Helper()
	q := store.New(db)
	ids := make([]int64, 0, n)
	for i := 0; i < n; i++ {
		a, err := q.InsertAntrian(context.Background(), store.InsertAntrianParams{
			Jenis: jenis, Nomor: "A-001", Prefix: "A", NoUrut: int64(i + 1),
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		ids = append(ids, a.ID)
	}
	return ids
}

// ============================================================
// offlineDetector unit tests
// ============================================================

func TestDetector_FirstOnlineProbe_TidakFireCallback(t *testing.T) {
	var fired atomic.Int32
	d := newOfflineDetector(func(_ bool) { fired.Add(1) })
	d.update(nil) // online
	if fired.Load() != 0 {
		t.Errorf("first online probe seharusnya tidak fire (default sudah online)")
	}
	if !d.IsOnline() {
		t.Errorf("state harus online")
	}
}

func TestDetector_FirstOfflineProbe_FireCallback(t *testing.T) {
	var fired atomic.Int32
	var lastState atomic.Bool
	d := newOfflineDetector(func(online bool) {
		fired.Add(1)
		lastState.Store(online)
	})
	d.update(errors.New("network down"))
	if fired.Load() != 1 {
		t.Errorf("first offline probe harus fire, got %d", fired.Load())
	}
	if lastState.Load() {
		t.Errorf("state harus offline")
	}
}

func TestDetector_TidakSpamSaatStateSama(t *testing.T) {
	var fired atomic.Int32
	d := newOfflineDetector(func(_ bool) { fired.Add(1) })
	d.update(nil) // online (no fire)
	d.update(nil) // online (no fire)
	d.update(nil) // online (no fire)
	if fired.Load() != 0 {
		t.Errorf("3 online probes harusnya 0 fire, got %d", fired.Load())
	}
}

func TestDetector_OfflineKeOnline_FireCallback(t *testing.T) {
	var fired atomic.Int32
	var states []bool
	var mu sync.Mutex
	d := newOfflineDetector(func(online bool) {
		fired.Add(1)
		mu.Lock()
		states = append(states, online)
		mu.Unlock()
	})

	d.update(errors.New("offline")) // 1: false
	d.update(errors.New("offline")) // no fire
	d.update(nil)                    // 2: true
	d.update(nil)                    // no fire
	d.update(errors.New("offline")) // 3: false

	if fired.Load() != 3 {
		t.Errorf("expected 3 state changes, got %d", fired.Load())
	}
	mu.Lock()
	got := append([]bool{}, states...)
	mu.Unlock()
	want := []bool{false, true, false}
	for i, v := range want {
		if got[i] != v {
			t.Errorf("state[%d] = %v, want %v", i, got[i], v)
		}
	}
}

// ============================================================
// Worker SyncPendingAntrian
// ============================================================

func TestWorker_SyncPendingAntrian_Success(t *testing.T) {
	db := newTestDB(t)
	ids := seedAntrian(t, db, 3, "LOKET")

	k := khanza.NewMock()
	k.BuatAntrianFunc = func(ctx context.Context, req domain.AntrianRequest) (*domain.Ticket, error) {
		return &domain.Ticket{Nomor: "A-001"}, nil
	}

	w := NewWithOptions(db, k, Options{Interval: time.Hour, MaxRetry: 5})
	if err := w.SyncPendingAntrian(context.Background()); err != nil {
		t.Fatalf("SyncPendingAntrian: %v", err)
	}

	// Verifikasi semua synced
	q := store.New(db)
	pending, _ := q.GetPendingAntrian(context.Background(), 10)
	if len(pending) != 0 {
		t.Errorf("expected 0 pending after sync, got %d", len(pending))
	}
	// Verifikasi reconcile_log SYNC_SUCCESS
	logs, _ := q.GetRecentLogs(context.Background(), 10)
	successCount := 0
	for _, l := range logs {
		if l.Action == "SYNC_SUCCESS" {
			successCount++
		}
	}
	if successCount != 3 {
		t.Errorf("expected 3 SYNC_SUCCESS logs, got %d", successCount)
	}
	_ = ids
}

func TestWorker_SyncPendingAntrian_FailRetry_KemudianFailed(t *testing.T) {
	db := newTestDB(t)
	seedAntrian(t, db, 1, "LOKET")

	k := khanza.NewMock()
	k.BuatAntrianFunc = func(ctx context.Context, req domain.AntrianRequest) (*domain.Ticket, error) {
		return nil, errors.New("khanza 500")
	}

	w := NewWithOptions(db, k, Options{Interval: time.Hour, MaxRetry: 3})

	// 3 attempt → harus mark failed
	for i := 0; i < 3; i++ {
		if err := w.SyncPendingAntrian(context.Background()); err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
	}

	q := store.New(db)
	pending, _ := q.GetPendingAntrian(context.Background(), 10)
	if len(pending) != 0 {
		t.Errorf("setelah retry exhausted, harus tidak ada pending (sudah failed). got %d", len(pending))
	}

	// Verifikasi log SYNC_FAILED ada
	logs, _ := q.GetRecentLogs(context.Background(), 20)
	hasFailed := false
	for _, l := range logs {
		if l.Action == "SYNC_FAILED" {
			hasFailed = true
			break
		}
	}
	if !hasFailed {
		t.Errorf("SYNC_FAILED log seharusnya ada setelah retry exhausted")
	}
}

func TestWorker_SyncPendingAntrian_PartialFailure(t *testing.T) {
	db := newTestDB(t)
	seedAntrian(t, db, 5, "LOKET")

	k := khanza.NewMock()
	var hits atomic.Int32
	k.BuatAntrianFunc = func(ctx context.Context, req domain.AntrianRequest) (*domain.Ticket, error) {
		n := hits.Add(1)
		if n%2 == 0 {
			return nil, errors.New("transient")
		}
		return &domain.Ticket{Nomor: "X"}, nil
	}

	w := NewWithOptions(db, k, Options{MaxRetry: 5})
	if err := w.SyncPendingAntrian(context.Background()); err != nil {
		t.Fatalf("SyncPendingAntrian: %v", err)
	}

	q := store.New(db)
	pending, _ := q.GetPendingAntrian(context.Background(), 10)
	// 3 sukses (ke-1, 3, 5), 2 masih pending dengan retry_count=1
	if len(pending) != 2 {
		t.Errorf("expected 2 pending (gagal yg masih retry-able), got %d", len(pending))
	}
}

// ============================================================
// Worker SyncConfirmedSEP
// ============================================================

func TestWorker_SyncConfirmedSEP_Success(t *testing.T) {
	db := newTestDB(t)
	q := store.New(db)

	// Seed pending_sep dengan status awaiting_sync (sudah dikonfirmasi)
	sep, err := q.InsertPendingSEP(context.Background(), store.InsertPendingSEPParams{
		NoKartu: "0001", Kategori: "RUJUKAN", PayloadJson: `{}`,
		VclaimResponse: sql.NullString{String: `{"NoSEP":"SEP-X","NoKartu":"0001"}`, Valid: true},
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	_ = q.ConfirmSEP(context.Background(), store.ConfirmSEPParams{
		ConfirmedBy: sql.NullString{String: "operator", Valid: true},
		ID:          sep.ID,
	})

	k := khanza.NewMock()
	k.SimpanSEPFunc = func(ctx context.Context, s domain.SEP) error { return nil }

	w := NewWithOptions(db, k, Options{})
	if err := w.SyncConfirmedSEP(context.Background()); err != nil {
		t.Fatalf("SyncConfirmedSEP: %v", err)
	}

	awaiting, _ := q.GetPendingSEPs(context.Background(), store.GetPendingSEPsParams{
		Status: sql.NullString{String: "awaiting_sync", Valid: true}, Limit: 10,
	})
	if len(awaiting) != 0 {
		t.Errorf("setelah sync, awaiting_sync harus 0, got %d", len(awaiting))
	}
	synced, _ := q.GetPendingSEPs(context.Background(), store.GetPendingSEPsParams{
		Status: sql.NullString{String: "synced", Valid: true}, Limit: 10,
	})
	if len(synced) != 1 {
		t.Errorf("expected 1 synced, got %d", len(synced))
	}
}

func TestWorker_SyncConfirmedSEP_TidakProsesPending(t *testing.T) {
	db := newTestDB(t)
	q := store.New(db)

	// Seed pending_sep status='pending' (belum dikonfirmasi)
	_, _ = q.InsertPendingSEP(context.Background(), store.InsertPendingSEPParams{
		NoKartu: "0001", Kategori: "RUJUKAN", PayloadJson: `{}`,
	})

	k := khanza.NewMock()
	var simpanCalled atomic.Int32
	k.SimpanSEPFunc = func(ctx context.Context, s domain.SEP) error {
		simpanCalled.Add(1)
		return nil
	}

	w := NewWithOptions(db, k, Options{})
	if err := w.SyncConfirmedSEP(context.Background()); err != nil {
		t.Fatalf("SyncConfirmedSEP: %v", err)
	}

	if simpanCalled.Load() != 0 {
		t.Errorf("SimpanSEP TIDAK boleh dipanggil untuk status='pending', got %d", simpanCalled.Load())
	}
}

// ============================================================
// Worker lifecycle (Start/Stop, ticker)
// ============================================================

func TestWorker_StartStop_Idempotent(t *testing.T) {
	db := newTestDB(t)
	k := khanza.NewMock()
	w := NewWithOptions(db, k, Options{Interval: time.Hour})

	w.Start(context.Background())
	w.Start(context.Background()) // idempotent
	w.Stop()
	w.Stop() // idempotent
}

func TestWorker_Tick_OnlineCallbackFiresSekali(t *testing.T) {
	db := newTestDB(t)
	k := khanza.NewMock()
	// Simulate offline → online transition
	var hits atomic.Int32
	k.HealthCheckFunc = func(ctx context.Context) error {
		n := hits.Add(1)
		if n == 1 {
			return errors.New("offline")
		}
		return nil
	}

	var stateChanges atomic.Int32
	w := NewWithOptions(db, k, Options{
		Interval: 50 * time.Millisecond,
		OnStateChange: func(_ bool) {
			stateChanges.Add(1)
		},
	})
	w.Start(context.Background())
	// Tunggu 200ms — cukup untuk 3-4 tick
	time.Sleep(200 * time.Millisecond)
	w.Stop()

	// Expected: tick #1 = offline (fire), tick #2 = online (fire),
	// tick #3,4 = online (no fire). Total = 2.
	got := stateChanges.Load()
	if got < 1 || got > 3 {
		t.Errorf("expected 1-3 state changes, got %d", got)
	}
}

func TestWorker_Tick_OfflineSkipSync(t *testing.T) {
	db := newTestDB(t)
	seedAntrian(t, db, 1, "LOKET")

	k := khanza.NewMock()
	k.HealthCheckFunc = func(ctx context.Context) error {
		return errors.New("offline")
	}
	var antrianCalls atomic.Int32
	k.BuatAntrianFunc = func(ctx context.Context, req domain.AntrianRequest) (*domain.Ticket, error) {
		antrianCalls.Add(1)
		return nil, nil
	}

	w := NewWithOptions(db, k, Options{Interval: 50 * time.Millisecond})
	w.Start(context.Background())
	time.Sleep(150 * time.Millisecond)
	w.Stop()

	if antrianCalls.Load() != 0 {
		t.Errorf("offline state seharusnya tidak panggil BuatAntrian, got %d", antrianCalls.Load())
	}
}

func TestWorker_IsOnline(t *testing.T) {
	db := newTestDB(t)
	k := khanza.NewMock()
	w := NewWithOptions(db, k, Options{})
	if !w.IsOnline() {
		t.Error("default state harus online (optimistic)")
	}
}
