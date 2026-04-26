package reconcile

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/arunika/apm-go/internal/domain"
	"github.com/arunika/apm-go/internal/integration/khanza"
	"github.com/arunika/apm-go/internal/store"
)

// Defaults — bisa di-override via Options di NewWithOptions.
const (
	defaultInterval     = 30 * time.Second
	defaultMaxRetry     = 5
	defaultBatchSize    = 50
	defaultProbeTimeout = 5 * time.Second
)

// Options untuk konfigurasi worker (dipakai test atau caller yang
// butuh tuning per deployment).
type Options struct {
	// Interval antar probe. Default 30 detik.
	Interval time.Duration
	// MaxRetry sebelum mark sync_status='failed'. Default 5.
	MaxRetry int
	// BatchSize per cycle SyncPendingAntrian. Default 50.
	BatchSize int
	// ProbeTimeout untuk HealthCheck. Default 5 detik.
	ProbeTimeout time.Duration
	// OnStateChange dipanggil saat online↔offline berubah.
	// Boleh nil (no-op).
	OnStateChange stateChangeCallback
	// Logger custom. Boleh nil — pakai slog.Default().
	Logger *slog.Logger
}

// ReconcileWorker = background worker yang sync data offline ke Khanza
// saat koneksi pulih.
type ReconcileWorker struct {
	db     *sql.DB
	khanza khanza.KhanzaClient

	opts     Options
	detector *offlineDetector
	logger   *slog.Logger

	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex // protects cancel + start state
}

// New membangun worker dengan default settings.
func New(db *sql.DB, k khanza.KhanzaClient) *ReconcileWorker {
	return NewWithOptions(db, k, Options{})
}

// NewWithOptions membangun worker dengan custom options.
// Field yang nil/zero pakai default.
func NewWithOptions(db *sql.DB, k khanza.KhanzaClient, opts Options) *ReconcileWorker {
	if opts.Interval <= 0 {
		opts.Interval = defaultInterval
	}
	if opts.MaxRetry <= 0 {
		opts.MaxRetry = defaultMaxRetry
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = defaultBatchSize
	}
	if opts.ProbeTimeout <= 0 {
		opts.ProbeTimeout = defaultProbeTimeout
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &ReconcileWorker{
		db:       db,
		khanza:   k,
		opts:     opts,
		detector: newOfflineDetector(opts.OnStateChange),
		logger:   logger,
	}
}

// Start jalanin background goroutine. Non-blocking — return cepat.
// Idempotent: panggil 2x cuma start sekali.
func (w *ReconcileWorker) Start(ctx context.Context) {
	w.mu.Lock()
	if w.cancel != nil {
		w.mu.Unlock()
		return
	}
	wctx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	w.mu.Unlock()

	w.wg.Add(1)
	go w.run(wctx)
	w.logger.Info("reconcile worker started", "interval", w.opts.Interval)
}

// Stop sinyal cancel + tunggu goroutine selesai. Idempotent.
func (w *ReconcileWorker) Stop() {
	w.mu.Lock()
	cancel := w.cancel
	w.cancel = nil
	w.mu.Unlock()

	if cancel == nil {
		return
	}
	cancel()
	w.wg.Wait()
	w.logger.Info("reconcile worker stopped")
}

// IsOnline accessor untuk admin panel / status display.
func (w *ReconcileWorker) IsOnline() bool {
	return w.detector.IsOnline()
}

// run main loop. Probe + sync setiap interval.
func (w *ReconcileWorker) run(ctx context.Context) {
	defer w.wg.Done()

	// First tick: jalankan segera (jangan tunggu interval pertama)
	w.tick(ctx)

	ticker := time.NewTicker(w.opts.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

// tick = 1 cycle worker: probe → sync (kalau online).
func (w *ReconcileWorker) tick(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("reconcile tick panic recovered", "panic", r)
		}
	}()

	probeCtx, cancel := context.WithTimeout(ctx, w.opts.ProbeTimeout)
	err := w.khanza.HealthCheck(probeCtx)
	cancel()
	w.detector.update(err)

	if err != nil {
		// Offline — skip sync
		return
	}

	// Online: sync data backlog
	if syncErr := w.SyncPendingAntrian(ctx); syncErr != nil {
		w.logger.Warn("sync pending antrian gagal", "err", syncErr.Error())
	}
	if syncErr := w.SyncConfirmedSEP(ctx); syncErr != nil {
		w.logger.Warn("sync pending sep gagal", "err", syncErr.Error())
	}
}

// SyncPendingAntrian flush antrian_lokal pending ke Khanza.
//
// Strategi:
//
//	Untuk setiap record:
//	  1. POST ke khanza.BuatAntrian dengan jenis/sub_jenis/no_rm
//	  2. Sukses → MarkAntrianSynced (sync_status='synced', synced_at=now)
//	  3. Gagal → IncrementAntrianRetry (last_error)
//	     Setelah retry_count >= MaxRetry → MarkAntrianFailed
//	  4. Selalu insert ke reconcile_log untuk audit
func (w *ReconcileWorker) SyncPendingAntrian(ctx context.Context) error {
	q := store.New(w.db)
	rows, err := q.GetAntrianForSync(ctx, store.GetAntrianForSyncParams{
		RetryCount: sql.NullInt64{Int64: int64(w.opts.MaxRetry), Valid: true},
		Limit:      int64(w.opts.BatchSize),
	})
	if err != nil {
		return fmt.Errorf("get pending antrian: %w", err)
	}

	for _, row := range rows {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		req := domain.AntrianRequest{
			Jenis:    row.Jenis,
			SubJenis: row.SubJenis.String,
			KdPoli:   row.NoPoli.String,
			NoRM:     row.NoRm.String,
		}
		_, kerr := w.khanza.BuatAntrian(ctx, req)
		if kerr == nil {
			if mErr := q.MarkAntrianSynced(ctx, row.ID); mErr != nil {
				w.logger.Warn("mark antrian synced gagal",
					"id", row.ID, "err", mErr.Error())
			}
			w.logReconcile(ctx, q, "antrian_lokal", row.ID, "SYNC_SUCCESS", "OK")
			continue
		}

		// Gagal — increment retry, mark failed kalau exceed
		_ = q.IncrementAntrianRetry(ctx, store.IncrementAntrianRetryParams{
			LastError: sql.NullString{String: kerr.Error(), Valid: true},
			ID:        row.ID,
		})
		newRetry := row.RetryCount.Int64 + 1
		if int(newRetry) >= w.opts.MaxRetry {
			_ = q.MarkAntrianFailed(ctx, row.ID)
			w.logger.Warn("antrian sync exhausted retry",
				"id", row.ID, "retry_count", newRetry, "err", kerr.Error())
			w.logReconcile(ctx, q, "antrian_lokal", row.ID, "SYNC_FAILED",
				fmt.Sprintf("retry_exhausted: %s", kerr.Error()))
		} else {
			w.logReconcile(ctx, q, "antrian_lokal", row.ID, "SYNC_ATTEMPT",
				fmt.Sprintf("retry=%d err=%s", newRetry, kerr.Error()))
		}
	}
	return nil
}

// SyncConfirmedSEP flush pending_sep status='awaiting_sync' ke Khanza.
// CATATAN: hanya yang sudah dikonfirmasi operator (P-046 admin panel).
// SEP status='pending' TIDAK di-sync otomatis — operator audit dulu.
func (w *ReconcileWorker) SyncConfirmedSEP(ctx context.Context) error {
	q := store.New(w.db)
	rows, err := q.GetPendingSEPs(ctx, store.GetPendingSEPsParams{
		Status: sql.NullString{String: "awaiting_sync", Valid: true},
		Limit:  int64(w.opts.BatchSize),
	})
	if err != nil {
		return fmt.Errorf("get awaiting sep: %w", err)
	}

	for _, row := range rows {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Re-hydrate SEP dari vclaim_response (yang disimpan saat issue)
		var sep domain.SEP
		if row.VclaimResponse.Valid {
			_ = json.Unmarshal([]byte(row.VclaimResponse.String), &sep)
		}
		if sep.NoSEP == "" {
			// Data corrupt — mark synced (avoid stuck) dan log
			w.logger.Warn("pending_sep vclaim_response invalid, skip",
				"id", row.ID)
			_ = q.MarkSEPSynced(ctx, row.ID)
			w.logReconcile(ctx, q, "pending_sep", row.ID,
				"SYNC_SKIPPED", "vclaim_response invalid")
			continue
		}

		kerr := w.khanza.SimpanSEP(ctx, sep)
		if kerr == nil {
			_ = q.MarkSEPSynced(ctx, row.ID)
			w.logReconcile(ctx, q, "pending_sep", row.ID, "SYNC_SUCCESS", "OK")
			continue
		}
		_ = q.IncrementSEPRetry(ctx, store.IncrementSEPRetryParams{
			LastError: sql.NullString{String: kerr.Error(), Valid: true},
			ID:        row.ID,
		})
		w.logReconcile(ctx, q, "pending_sep", row.ID, "SYNC_ATTEMPT", kerr.Error())
	}
	return nil
}

// logReconcile insert audit ke reconcile_log. Error TIDAK throw —
// reconcile log adalah audit, tidak boleh menggagalkan main flow.
func (w *ReconcileWorker) logReconcile(ctx context.Context, q *store.Queries,
	table string, id int64, action, result string) {
	_, err := q.InsertReconcileLog(ctx, store.InsertReconcileLogParams{
		TableName:  table,
		RecordID:   id,
		Action:     action,
		OperatorID: sql.NullString{String: "system", Valid: true},
		Result:     sql.NullString{String: result, Valid: true},
	})
	if err != nil {
		w.logger.Warn("insert reconcile_log gagal", "err", err.Error())
	}
}
