// Package antrian adalah service layer untuk pengelolaan antrian pasien
// 3 jalur (Loket / Poli / Umum). Service ini menggabungkan:
//
//   - Khanza (atomic counter di server SIMRS — single source of truth saat online)
//   - SQLite store lokal (buffer offline + counter fallback)
//   - Antrol API BPJS (push fire-and-forget supaya antrian muncul di Mobile JKN)
//
// Strategi offline:
//
//	Saat Khanza unreachable (domain.ErrOffline), service generate nomor
//	dari MAX(no_urut) hari ini di SQLite + 1 (di-serialize via mutex
//	supaya 5 kiosk sekalipun tidak menghasilkan duplikat).
//	Reconcile worker (P-050) akan mem-flush entry pending ke Khanza
//	begitu koneksi pulih.
//
//	CAVEAT diketahui: counter offline TIDAK aware Khanza counter
//	terakhir. Jika sebelum offline Khanza sudah issue A-050, counter
//	offline mulai dari A-001 → tampilan duplikat. Risiko diterima
//	karena offline mode jarang & singkat. Mitigasi level UX: badge
//	"OFFLINE" pada tiket via Ticket.IsOffline.
package antrian

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/arunika/apm-go/internal/config"
	"github.com/arunika/apm-go/internal/domain"
	"github.com/arunika/apm-go/internal/integration/antrol"
	"github.com/arunika/apm-go/internal/integration/khanza"
	"github.com/arunika/apm-go/internal/store"
)

// AntrianService — sesuai spec P-021. db ditambahkan supaya bisa membuat
// store.Queries on-demand (untuk operasi yang butuh handle DB lain).
// Untuk path standar dipakai field store langsung.
type AntrianService struct {
	khanza khanza.KhanzaClient
	store  *store.Queries
	antrol antrol.AntrolClient

	cfg    config.AntrianConfig
	now    func() time.Time
	logger *slog.Logger

	// offlineMu meng-serialize generation counter offline supaya
	// concurrent Create() di mode offline tidak menghasilkan
	// duplikat no_urut. Single-writer SQLite handle persistence,
	// mutex ini handle race read-then-write.
	offlineMu sync.Mutex

	// pushTimeout cap untuk Antrol push fire-and-forget.
	pushTimeout time.Duration
}

// New membangun AntrianService. db perlu — sql.DB karena akan dipakai
// store.New(db) saat init plus dipakai future-proof untuk transaction.
func New(
	khanzaCli khanza.KhanzaClient,
	db *sql.DB,
	antrolCli antrol.AntrolClient,
	cfg config.AntrianConfig,
) *AntrianService {
	return &AntrianService{
		khanza:      khanzaCli,
		store:       store.New(db),
		antrol:      antrolCli,
		cfg:         cfg,
		now:         time.Now,
		logger:      slog.Default(),
		pushTimeout: 5 * time.Second,
	}
}

// SetLogger mengganti logger (test atau caller dengan PHIMaskingHandler).
func (s *AntrianService) SetLogger(l *slog.Logger) {
	if l != nil {
		s.logger = l
	}
}

// Create mengeluarkan satu nomor antrian sesuai req.
//
// Flow:
//  1. Khanza online → pakai nomor dari Khanza (atomic, anti-duplikat
//     antar-kiosk).
//  2. Khanza offline → generate dari counter lokal SQLite dengan mutex
//     serialize, simpan ke antrian_lokal status='pending', set
//     ticket.IsOffline=true.
//  3. Push ke Antrol fire-and-forget (online ticket only — offline
//     tidak perlu push, akan di-push saat reconcile).
func (s *AntrianService) Create(ctx context.Context, req domain.AntrianRequest) (*domain.Ticket, error) {
	if req.Jenis == "" {
		return nil, fmt.Errorf("antrian: jenis wajib diisi")
	}

	ticket, err := s.khanza.BuatAntrian(ctx, req)
	switch {
	case err == nil:
		s.pushToAntrolAsync(req, ticket)
		s.logger.Info("antrian: ticket dari khanza",
			"jenis", req.Jenis, "nomor", ticket.Nomor)
		return ticket, nil

	case errors.Is(err, domain.ErrOffline):
		s.logger.Warn("antrian: khanza offline, fallback ke counter lokal",
			"jenis", req.Jenis)
		return s.createOffline(ctx, req)

	default:
		return nil, fmt.Errorf("buat antrian khanza: %w", err)
	}
}

// createOffline generate nomor lokal + persist ke antrian_lokal.
func (s *AntrianService) createOffline(ctx context.Context, req domain.AntrianRequest) (*domain.Ticket, error) {
	s.offlineMu.Lock()
	defer s.offlineMu.Unlock()

	maxUrut, err := s.store.GetMaxNoUrutToday(ctx, req.Jenis)
	if err != nil {
		return nil, fmt.Errorf("get max no_urut offline: %w", err)
	}
	next := int(maxUrut) + 1
	prefix := s.prefixFor(req.Jenis)
	nomor := buildNomor(prefix, req, next)

	a, err := s.store.InsertAntrian(ctx, store.InsertAntrianParams{
		Jenis:    req.Jenis,
		SubJenis: nullableStr(req.SubJenis),
		Nomor:    nomor,
		Prefix:   prefix,
		NoUrut:   int64(next),
		NoRm:     nullableStr(req.NoRM),
		NoPoli:   nullableStr(req.KdPoli),
	})
	if err != nil {
		return nil, fmt.Errorf("insert antrian lokal: %w", err)
	}

	t := &domain.Ticket{
		ID:        strconv.FormatInt(a.ID, 10),
		Nomor:     a.Nomor,
		Jenis:     a.Jenis,
		SubJenis:  a.SubJenis.String,
		Prefix:    a.Prefix,
		NoUrut:    int(a.NoUrut),
		NoRM:      a.NoRm.String,
		NoPoli:    a.NoPoli.String,
		IsOffline: true,
	}
	if a.CreatedAt.Valid {
		t.CreatedAt = a.CreatedAt.Time
	} else {
		t.CreatedAt = s.now()
	}
	s.logger.Info("antrian: ticket offline dari counter lokal",
		"jenis", req.Jenis, "nomor", t.Nomor)
	return t, nil
}

// GetCounter mengembalikan no_urut terbesar hari ini untuk jenis yang
// diminta. Best-effort — hanya membaca dari store lokal (counter Khanza
// tidak tersedia via API). Dipakai oleh AntrianScreen untuk display
// "Sekarang: A-035".
func (s *AntrianService) GetCounter(ctx context.Context, jenis string) (int, error) {
	if jenis == "" {
		return 0, fmt.Errorf("jenis wajib diisi")
	}
	max, err := s.store.GetMaxNoUrutToday(ctx, jenis)
	if err != nil {
		return 0, fmt.Errorf("get counter %s: %w", jenis, err)
	}
	return int(max), nil
}

// ResetAll menghapus semua entry antrian_lokal hari ini — efektif
// reset counter ke 0. Dipakai oleh:
//
//   - cron daily reset (00:01 WIB)
//   - admin panel "Reset counter antrian"
//
// Audit trail di-log via reconcile_log.
func (s *AntrianService) ResetAll(ctx context.Context) error {
	deleted, err := s.store.DeleteAntrianToday(ctx)
	if err != nil {
		return fmt.Errorf("reset counter: %w", err)
	}

	if _, err := s.store.InsertReconcileLog(ctx, store.InsertReconcileLogParams{
		TableName:  "antrian_lokal",
		RecordID:   0,
		Action:     "RESET_COUNTER",
		OperatorID: nullableStr("system"),
		Result:     nullableStr(fmt.Sprintf("deleted=%d", deleted)),
	}); err != nil {
		s.logger.Warn("antrian: gagal write audit reset", "err", err)
	}
	s.logger.Info("antrian: reset counter selesai", "rows_deleted", deleted)
	return nil
}

// prefixFor map jenis → prefix dari config.
func (s *AntrianService) prefixFor(jenis string) string {
	switch jenis {
	case string(domain.AntrianJenisLoket):
		return s.cfg.LoketPrefix
	case string(domain.AntrianJenisPoli):
		return s.cfg.PoliPrefix
	case string(domain.AntrianJenisUmum):
		return s.cfg.UmumPrefix
	default:
		return "?"
	}
}

// pushToAntrolAsync menjalankan PushAntrian di goroutine terpisah.
// Fire-and-forget: error tidak diteruskan ke caller. Goroutine
// memiliki context dengan timeout sendiri supaya tidak menggantung.
func (s *AntrianService) pushToAntrolAsync(req domain.AntrianRequest, ticket *domain.Ticket) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("antrian: panic di push antrol",
					"panic", r, "nomor", ticket.Nomor)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), s.pushTimeout)
		defer cancel()
		if err := s.antrol.PushAntrian(ctx, req, ticket); err != nil {
			s.logger.Warn("antrian: push antrol gagal (fire-and-forget)",
				"err", err, "nomor", ticket.Nomor)
		}
	}()
}

// buildNomor membentuk display nomor: dipakai prefix dari jenis +
// optional kdPoli (untuk POLI) + zero-padded urut.
func buildNomor(prefix string, req domain.AntrianRequest, urut int) string {
	if req.Jenis == string(domain.AntrianJenisPoli) && req.KdPoli != "" {
		return fmt.Sprintf("%s-%s-%03d", prefix, req.KdPoli, urut)
	}
	return fmt.Sprintf("%s-%03d", prefix, urut)
}

// nullableStr returns sql.NullString — Valid kalau s non-empty.
func nullableStr(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
