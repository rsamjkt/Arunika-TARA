package antrian

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

	"github.com/arunika/apm-go/internal/config"
	"github.com/arunika/apm-go/internal/domain"
	"github.com/arunika/apm-go/internal/integration/antrol"
	"github.com/arunika/apm-go/internal/integration/khanza"
	"github.com/arunika/apm-go/internal/store"
)

// loadSchema membaca migrations/001_initial.sql relatif ke project root.
func loadSchema(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("tidak bisa resolve test file path")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	b, err := os.ReadFile(filepath.Join(root, "migrations", "001_initial.sql"))
	if err != nil {
		t.Fatalf("baca schema: %v", err)
	}
	return string(b)
}

// setupSvc membangun AntrianService dengan mock khanza/antrol + SQLite
// in-memory yang sudah ter-apply schema.
func setupSvc(t *testing.T) (*AntrianService, *khanza.MockKhanzaClient, *antrol.MockAntrolClient, *sql.DB) {
	t.Helper()
	ctx := context.Background()
	db, _, err := store.Open(ctx, ":memory:", loadSchema(t))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	k := khanza.NewMock()
	a := antrol.NewMock()
	cfg := config.AntrianConfig{
		LoketPrefix: "A",
		PoliPrefix:  "B",
		UmumPrefix:  "C",
		ResetTime:   "00:01",
	}
	svc := New(k, db, a, cfg)
	return svc, k, a, db
}

// ============================================================
// Skenario 1 — Online: Khanza tersedia
// ============================================================

func TestAntrianService_Create_Online_NomorDariKhanza(t *testing.T) {
	svc, k, a, _ := setupSvc(t)

	want := &domain.Ticket{
		ID: "tk-001", Nomor: "B-INT-005",
		Jenis: "POLI", Prefix: "B", NoUrut: 5, NoPoli: "INT",
	}
	k.SetResponse("BuatAntrian", want, nil)

	got, err := svc.Create(context.Background(), domain.AntrianRequest{
		Jenis: "POLI", KdPoli: "INT", NoSEP: "SEP-001",
	})
	if err != nil {
		t.Fatalf("Create online: %v", err)
	}
	if got.Nomor != "B-INT-005" || got.NoUrut != 5 {
		t.Errorf("ticket dari Khanza salah: %+v", got)
	}
	if got.IsOffline {
		t.Errorf("online ticket tidak boleh IsOffline=true")
	}
	if k.CallCount("BuatAntrian") != 1 {
		t.Errorf("Khanza harus dipanggil exactly 1x")
	}

	// Antrol push fire-and-forget — beri grace untuk goroutine
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) && a.CallCount("PushAntrian") == 0 {
		time.Sleep(10 * time.Millisecond)
	}
	if a.CallCount("PushAntrian") != 1 {
		t.Errorf("PushAntrian harus dipanggil 1x setelah Khanza sukses, got %d",
			a.CallCount("PushAntrian"))
	}
}

func TestAntrianService_Create_AntrolPushError_TidakBlokirReturn(t *testing.T) {
	svc, k, a, _ := setupSvc(t)

	k.SetResponse("BuatAntrian", &domain.Ticket{ID: "1", Nomor: "A-001", Jenis: "LOKET"}, nil)
	a.PushAntrianFunc = func(ctx context.Context, req domain.AntrianRequest, t *domain.Ticket) error {
		return errors.New("antrol API down")
	}

	start := time.Now()
	got, err := svc.Create(context.Background(), domain.AntrianRequest{Jenis: "LOKET"})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Create error karena antrol gagal: %v", err)
	}
	if got == nil {
		t.Fatal("ticket nil")
	}
	if elapsed > 200*time.Millisecond {
		t.Errorf("Create blocked menunggu antrol push: %v", elapsed)
	}
}

// ============================================================
// Skenario 2 — Offline: Khanza down
// ============================================================

func TestAntrianService_Create_Offline_NomorDariSQLite(t *testing.T) {
	svc, k, a, _ := setupSvc(t)

	k.SetResponse("BuatAntrian", nil, domain.ErrOffline)

	got, err := svc.Create(context.Background(), domain.AntrianRequest{Jenis: "LOKET"})
	if err != nil {
		t.Fatalf("Create offline: %v", err)
	}
	if !got.IsOffline {
		t.Errorf("offline ticket harus IsOffline=true")
	}
	if got.Nomor != "A-001" {
		t.Errorf("offline ticket pertama harus A-001, got %q", got.Nomor)
	}
	if got.NoUrut != 1 {
		t.Errorf("NoUrut = %d, want 1", got.NoUrut)
	}

	// Antrol TIDAK boleh dipush untuk offline ticket
	time.Sleep(100 * time.Millisecond)
	if a.CallCount("PushAntrian") != 0 {
		t.Errorf("offline tidak boleh push antrol, got %d call", a.CallCount("PushAntrian"))
	}

	// Counter berikutnya harus naik
	got2, _ := svc.Create(context.Background(), domain.AntrianRequest{Jenis: "LOKET"})
	if got2.Nomor != "A-002" {
		t.Errorf("counter offline kedua = %q, want A-002", got2.Nomor)
	}
}

func TestAntrianService_Create_Offline_PoliFormat(t *testing.T) {
	svc, k, _, _ := setupSvc(t)
	k.SetResponse("BuatAntrian", nil, domain.ErrOffline)

	got, err := svc.Create(context.Background(), domain.AntrianRequest{
		Jenis: "POLI", KdPoli: "INT",
	})
	if err != nil {
		t.Fatalf("Create offline POLI: %v", err)
	}
	if got.Nomor != "B-INT-001" {
		t.Errorf("format POLI offline salah: %q, want B-INT-001", got.Nomor)
	}
}

func TestAntrianService_Create_Offline_KhanzaOtherError_TidakFallback(t *testing.T) {
	svc, k, _, _ := setupSvc(t)
	k.SetResponse("BuatAntrian", nil, errors.New("validation: kd_poli invalid"))

	_, err := svc.Create(context.Background(), domain.AntrianRequest{Jenis: "LOKET"})
	if err == nil {
		t.Fatal("error non-offline harus diteruskan, got nil")
	}
	if errors.Is(err, domain.ErrOffline) {
		t.Errorf("error harus bukan ErrOffline")
	}
}

// ============================================================
// Skenario 3 — Concurrent: 5 goroutine, no duplikat
// ============================================================

func TestAntrianService_Create_5Concurrent_TanpaDuplikat(t *testing.T) {
	svc, k, _, _ := setupSvc(t)
	k.SetResponse("BuatAntrian", nil, domain.ErrOffline) // semua offline

	const N = 5
	var wg sync.WaitGroup
	wg.Add(N)
	tickets := make([]*domain.Ticket, N)

	for i := 0; i < N; i++ {
		i := i
		go func() {
			defer wg.Done()
			t, err := svc.Create(context.Background(), domain.AntrianRequest{Jenis: "LOKET"})
			if err != nil {
				return
			}
			tickets[i] = t
		}()
	}
	wg.Wait()

	// Verifikasi semua sukses + nomor unik 1..N
	seen := make(map[int]bool, N)
	for i, t := range tickets {
		if t == nil {
			t2 := tickets[i]
			_ = t2
			continue
		}
		if seen[t.NoUrut] {
			tt := t
			_ = tt
			continue
		}
		seen[t.NoUrut] = true
	}
	if len(seen) != N {
		var nos []int
		for _, t := range tickets {
			if t != nil {
				nos = append(nos, t.NoUrut)
			}
		}
		t.Errorf("expected %d nomor unik, got %d. Nomor: %v", N, len(seen), nos)
	}
}

func TestAntrianService_Create_30Concurrent_StressTest(t *testing.T) {
	svc, k, _, _ := setupSvc(t)
	k.SetResponse("BuatAntrian", nil, domain.ErrOffline)

	const N = 30
	var wg sync.WaitGroup
	wg.Add(N)
	var ok atomic.Int32
	results := make([]int, N)

	for i := 0; i < N; i++ {
		i := i
		go func() {
			defer wg.Done()
			tk, err := svc.Create(context.Background(), domain.AntrianRequest{Jenis: "POLI", KdPoli: "INT"})
			if err == nil && tk != nil {
				results[i] = tk.NoUrut
				ok.Add(1)
			}
		}()
	}
	wg.Wait()

	if int(ok.Load()) != N {
		t.Fatalf("hanya %d/%d sukses", ok.Load(), N)
	}

	seen := make(map[int]bool)
	for _, n := range results {
		if seen[n] {
			t.Errorf("duplikat no_urut: %d", n)
		}
		seen[n] = true
	}
	if len(seen) != N {
		t.Errorf("expected %d unik, got %d", N, len(seen))
	}
}

// ============================================================
// GetCounter
// ============================================================

func TestAntrianService_GetCounter_SetelahBeberapaCreate(t *testing.T) {
	svc, k, _, _ := setupSvc(t)
	k.SetResponse("BuatAntrian", nil, domain.ErrOffline)

	for i := 0; i < 5; i++ {
		_, _ = svc.Create(context.Background(), domain.AntrianRequest{Jenis: "LOKET"})
	}
	for i := 0; i < 3; i++ {
		_, _ = svc.Create(context.Background(), domain.AntrianRequest{Jenis: "UMUM"})
	}

	if c, _ := svc.GetCounter(context.Background(), "LOKET"); c != 5 {
		t.Errorf("Counter LOKET = %d, want 5", c)
	}
	if c, _ := svc.GetCounter(context.Background(), "UMUM"); c != 3 {
		t.Errorf("Counter UMUM = %d, want 3", c)
	}
	if c, _ := svc.GetCounter(context.Background(), "POLI"); c != 0 {
		t.Errorf("Counter POLI (kosong) = %d, want 0", c)
	}
}

func TestAntrianService_GetCounter_JenisKosong_Error(t *testing.T) {
	svc, _, _, _ := setupSvc(t)
	_, err := svc.GetCounter(context.Background(), "")
	if err == nil {
		t.Fatal("jenis kosong harus error")
	}
}

// ============================================================
// ResetAll
// ============================================================

func TestAntrianService_ResetAll_HapusEntryHariIni(t *testing.T) {
	svc, k, _, db := setupSvc(t)
	k.SetResponse("BuatAntrian", nil, domain.ErrOffline)

	for i := 0; i < 3; i++ {
		_, _ = svc.Create(context.Background(), domain.AntrianRequest{Jenis: "LOKET"})
	}

	// Reset
	if err := svc.ResetAll(context.Background()); err != nil {
		t.Fatalf("ResetAll: %v", err)
	}

	// Counter harus 0 setelah reset
	if c, _ := svc.GetCounter(context.Background(), "LOKET"); c != 0 {
		t.Errorf("counter setelah reset = %d, want 0", c)
	}

	// Audit log harus tercatat
	q := store.New(db)
	logs, _ := q.GetRecentLogs(context.Background(), 10)
	found := false
	for _, l := range logs {
		if l.Action == "RESET_COUNTER" && l.TableName == "antrian_lokal" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("audit log RESET_COUNTER tidak tertulis")
	}

	// Create setelah reset → mulai dari 1 lagi
	tk, _ := svc.Create(context.Background(), domain.AntrianRequest{Jenis: "LOKET"})
	if tk.NoUrut != 1 {
		t.Errorf("setelah reset, ticket pertama = %d, want 1", tk.NoUrut)
	}
}

// ============================================================
// Edge case: jenis kosong
// ============================================================

func TestAntrianService_Create_JenisKosong_Error(t *testing.T) {
	svc, _, _, _ := setupSvc(t)
	_, err := svc.Create(context.Background(), domain.AntrianRequest{})
	if err == nil {
		t.Fatal("jenis kosong harus error")
	}
}

// ============================================================
// Cron: AddFunc invalid schedule, dan callback fires
// ============================================================

func TestStartDailyReset_InvalidSchedule(t *testing.T) {
	svc, _, _, _ := setupSvc(t)
	c, err := StartDailyReset(svc, "this-is-not-a-cron")
	if err == nil {
		c.Stop()
		t.Fatal("schedule invalid harus error")
	}
}

func TestStartDailyReset_NilService(t *testing.T) {
	_, err := StartDailyReset(nil, "")
	if err == nil {
		t.Fatal("nil svc harus error")
	}
}

func TestStartDailyReset_CallbackBenarBenarFires(t *testing.T) {
	svc, k, _, _ := setupSvc(t)
	k.SetResponse("BuatAntrian", nil, domain.ErrOffline)

	// Buat beberapa entry
	for i := 0; i < 3; i++ {
		_, _ = svc.Create(context.Background(), domain.AntrianRequest{Jenis: "LOKET"})
	}
	// Pakai schedule "@every 1s" agar test cepat
	c, err := StartDailyReset(svc, "@every 1s")
	if err != nil {
		t.Fatalf("StartDailyReset: %v", err)
	}
	defer c.Stop()

	// Tunggu hingga reset terjadi (max 3 detik)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if c, _ := svc.GetCounter(context.Background(), "LOKET"); c == 0 {
			return // sukses, reset terjadi
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Errorf("cron callback tidak fire dalam 3 detik")
}
