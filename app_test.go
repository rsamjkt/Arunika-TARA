package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/arunika/apm-go/internal/config"
	"github.com/arunika/apm-go/internal/domain"
	"github.com/arunika/apm-go/internal/hardware"
	"github.com/arunika/apm-go/internal/hardware/fingerprint"
	"github.com/arunika/apm-go/internal/hardware/frista"
	"github.com/arunika/apm-go/internal/integration/antrol"
	"github.com/arunika/apm-go/internal/integration/khanza"
	"github.com/arunika/apm-go/internal/integration/vclaim"
	"github.com/arunika/apm-go/internal/service/antrian"
	"github.com/arunika/apm-go/internal/service/detector"
	"github.com/arunika/apm-go/internal/service/sep"
	"github.com/arunika/apm-go/internal/store"
)

// loadSchema baca migration relatif ke project root.
func loadSchema(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("migrations", "001_initial.sql"))
	if err != nil {
		t.Fatalf("baca schema: %v", err)
	}
	return string(b)
}

// newTestApp membangun App dengan mock semua dependency + in-memory DB.
// Skip startup() — wire manual untuk control penuh di test.
func newTestApp(t *testing.T) (*App, *vclaim.MockVClaimClient, *khanza.MockKhanzaClient, *antrol.MockAntrolClient) {
	t.Helper()
	ctx := context.Background()
	db, _, err := store.Open(ctx, ":memory:", loadSchema(t))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	cfg := &config.Config{
		App: config.AppConfig{Version: "1.0.0-test"},
		Antrian: config.AntrianConfig{
			LoketPrefix: "A", PoliPrefix: "B", UmumPrefix: "C",
			ResetTime: "00:01",
		},
	}

	v := vclaim.NewMock()
	k := khanza.NewMock()
	a := antrol.NewMock()

	// Frista (face verifier) sekarang call-based — sama interface
	// dengan fingerprint, hanya beda hardware. Wire mock supaya
	// VerifikasiWajah test path bisa dieksekusi.
	hw := &hardware.Provider{
		Frista:      frista.NewMock(),
		Fingerprint: fingerprint.NewMock(),
		Printer:     nil,
	}

	app := NewApp()
	app.ctx = ctx
	app.cfg = cfg
	app.db = db
	app.vclaim = v
	app.khanza = k
	app.antrol = a
	app.hw = hw
	app.detectorSvc = detector.New(v, a, k)
	app.antrianSvc = antrian.New(k, db, a, cfg.Antrian)
	app.sepSvc = sep.New(v, k, hw.Fingerprint, db)

	if mfp, ok := hw.Fingerprint.(*fingerprint.MockVerifier); ok {
		mfp.SetScanDelay(0)
	}
	if mfr, ok := hw.Frista.(*frista.MockVerifier); ok {
		mfr.SetScanDelay(0)
	}
	return app, v, k, a
}

// ============================================================
// Lifecycle
// ============================================================

func TestApp_NewApp_DefaultsSet(t *testing.T) {
	app := NewApp()
	if app == nil {
		t.Fatal("NewApp return nil")
	}
	if app.startedAt.IsZero() {
		t.Error("startedAt belum diset")
	}
	if app.logger == nil {
		t.Error("logger belum diset")
	}
}

func TestApp_StartupTanpaConfig_TidakPanic(t *testing.T) {
	// Pastikan startup tidak panic kalau config hilang —
	// just log error dan lanjut.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("startup panic dengan config hilang: %v", r)
		}
	}()

	t.Setenv("APM_CONFIG_PATH", "/nonexistent/config.toml")
	app := NewApp()
	app.startup(context.Background())
	app.shutdown(context.Background())
}

// ============================================================
// Detection + session cache
// ============================================================

func TestApp_DetectPatient_CacheLastPeserta(t *testing.T) {
	app, v, _, _ := newTestApp(t)

	v.GetPesertaFunc = func(ctx context.Context, id string, tgl time.Time) (*domain.Peserta, error) {
		return &domain.Peserta{
			NoKartu: "0001234567890012",
			NoRM:    "RM-001",
			Nama:    "Budi Santoso",
			StatusAktif: "1",
			KelasHak:    "2",
		}, nil
	}

	r, err := app.DetectPatient("0001234567890012")
	if err != nil {
		t.Fatalf("DetectPatient: %v", err)
	}
	if r.Peserta == nil {
		t.Fatal("Peserta tidak terisi")
	}

	app.mu.Lock()
	cached := app.lastPeserta
	app.mu.Unlock()
	if cached == nil || cached.NoKartu != "0001234567890012" {
		t.Errorf("lastPeserta tidak ter-cache: %+v", cached)
	}
}

func TestApp_DetectPatient_VClaimError_CacheTidakDiUpdate(t *testing.T) {
	app, v, _, _ := newTestApp(t)
	v.GetPesertaFunc = func(ctx context.Context, id string, tgl time.Time) (*domain.Peserta, error) {
		return nil, errors.New("network down")
	}

	_, err := app.DetectPatient("X")
	if err != nil {
		t.Fatalf("DetectPatient: %v", err)
	}

	app.mu.Lock()
	cached := app.lastPeserta
	app.mu.Unlock()
	if cached != nil {
		t.Errorf("cache seharusnya tidak ter-update saat error, got: %+v", cached)
	}
}

func TestApp_ResetSession_ClearCache(t *testing.T) {
	app, _, _, _ := newTestApp(t)
	app.lastPeserta = &domain.Peserta{NoKartu: "X"}

	if err := app.ResetSession(); err != nil {
		t.Fatalf("ResetSession: %v", err)
	}
	if app.lastPeserta != nil {
		t.Errorf("cache harus cleared")
	}
}

// ============================================================
// SEP — ambil dari cached Peserta
// ============================================================

func TestApp_BuatSEPRujukan_TanpaCache_Error(t *testing.T) {
	app, _, _, _ := newTestApp(t)
	// Tidak set lastPeserta

	_, err := app.BuatSEPRujukan(domain.SEPRequest{NoRujukan: "X"})
	if err == nil {
		t.Fatal("expected error karena belum DetectPatient")
	}
}

func TestApp_BuatSEPRujukan_DenganCache_Sukses(t *testing.T) {
	app, v, k, _ := newTestApp(t)
	app.lastPeserta = &domain.Peserta{
		NoKartu: "0001", NoRM: "RM-001",
		TglLahir: time.Now().AddDate(-30, 0, 0).Format("2006-01-02"),
		StatusAktif: "1", KelasHak: "2",
	}

	v.ValidasiRujukanFunc = func(ctx context.Context, noSurat string, tgl time.Time) (*domain.Rujukan, error) {
		return &domain.Rujukan{NoSurat: noSurat}, nil
	}
	v.CreateSEPFunc = func(ctx context.Context, req domain.SEPRequest) (*domain.SEP, error) {
		return &domain.SEP{NoSEP: "SEP-X", NoKartu: req.NoKartu}, nil
	}
	// Vendor pattern: cekFinger return Verified=true (pasien sudah verifikasi
	// biometrik di server BPJS) → SEP boleh issued tanpa prompt modal.
	v.CekFingerprintStatusFunc = func(ctx context.Context, noKartu string, tgl time.Time) (*vclaim.FingerprintStatus, error) {
		return &vclaim.FingerprintStatus{Verified: true}, nil
	}
	k.SetResponse("SimpanSEP", nil, nil)

	got, err := app.BuatSEPRujukan(domain.SEPRequest{
		NoRujukan: "RUJ-001", KdPoli: "ANAK", KdDokter: "D-001",
	})
	if err != nil {
		t.Fatalf("BuatSEPRujukan: %v", err)
	}
	if got.NoSEP != "SEP-X" {
		t.Errorf("NoSEP = %q", got.NoSEP)
	}
}

// ============================================================
// Antrian
// ============================================================

func TestApp_CreateAntrian_DelegateKeService(t *testing.T) {
	app, _, k, _ := newTestApp(t)
	k.SetResponse("BuatAntrian", &domain.Ticket{
		ID: "1", Nomor: "B-INT-001", Jenis: "POLI", NoUrut: 1,
	}, nil)

	got, err := app.CreateAntrian("POLI", "WALKIN")
	if err != nil {
		t.Fatalf("CreateAntrian: %v", err)
	}
	if got.Nomor != "B-INT-001" {
		t.Errorf("Nomor = %q", got.Nomor)
	}
}

func TestApp_GetCounters_Return3Jenis(t *testing.T) {
	app, _, k, _ := newTestApp(t)
	k.SetResponse("BuatAntrian", nil, domain.ErrOffline) // semua offline → fallback lokal

	// Trigger beberapa create
	for i := 0; i < 3; i++ {
		_, _ = app.CreateAntrian("LOKET", "")
	}
	for i := 0; i < 2; i++ {
		_, _ = app.CreateAntrian("UMUM", "")
	}

	counters, err := app.GetCounters()
	if err != nil {
		t.Fatalf("GetCounters: %v", err)
	}
	if counters["LOKET"] != 3 {
		t.Errorf("LOKET = %d, want 3", counters["LOKET"])
	}
	if counters["UMUM"] != 2 {
		t.Errorf("UMUM = %d, want 2", counters["UMUM"])
	}
	if counters["POLI"] != 0 {
		t.Errorf("POLI (kosong) = %d, want 0", counters["POLI"])
	}
}

// ============================================================
// Pasien (Khanza)
// ============================================================

func TestApp_CariPasien_DelegateKeKhanza(t *testing.T) {
	app, _, k, _ := newTestApp(t)
	want := &domain.Pasien{NoRM: "RM-001", Nama: "Test"}
	k.SetResponse("CariPasien", want, nil)

	got, err := app.CariPasien("RM-001")
	if err != nil {
		t.Fatalf("CariPasien: %v", err)
	}
	if got != want {
		t.Errorf("CariPasien tidak forward")
	}
}

func TestApp_GetJadwalDokter_DelegateKeKhanza(t *testing.T) {
	app, _, k, _ := newTestApp(t)
	want := []domain.JadwalDokter{{KdDokter: "D-001"}}
	k.SetResponse("GetJadwalDokter", want, nil)

	got, err := app.GetJadwalDokter("INT")
	if err != nil {
		t.Fatalf("GetJadwalDokter: %v", err)
	}
	if len(got) != 1 || got[0].KdDokter != "D-001" {
		t.Errorf("got = %+v", got)
	}
}

// ============================================================
// Status
// ============================================================

func TestApp_GetHardwareStatus(t *testing.T) {
	app, _, _, _ := newTestApp(t)

	st := app.GetHardwareStatus()
	// Frista mock → true; Fingerprint mock → true; Printer nil → false
	if !st.Frista {
		t.Errorf("Frista mock seharusnya available")
	}
	if !st.Fingerprint {
		t.Errorf("Fingerprint mock seharusnya available")
	}
	if st.Printer {
		t.Errorf("Printer nil seharusnya false")
	}
}

func TestApp_GetSystemStatus(t *testing.T) {
	app, _, _, _ := newTestApp(t)

	st := app.GetSystemStatus()
	if st.Version != "1.0.0-test" {
		t.Errorf("Version = %q", st.Version)
	}
	if st.UptimeSec < 0 {
		t.Errorf("UptimeSec invalid: %d", st.UptimeSec)
	}
	if st.StartedAt == "" {
		t.Errorf("StartedAt kosong")
	}
	if !st.Online {
		t.Errorf("Online false padahal khanza ada")
	}
}

// ============================================================
// Admin
// ============================================================

func TestApp_GetPendingSEPs_BacaDariStore(t *testing.T) {
	app, _, _, _ := newTestApp(t)

	q := store.New(app.db)
	_, err := q.InsertPendingSEP(context.Background(), store.InsertPendingSEPParams{
		NoKartu: "0001", Kategori: "RUJUKAN", PayloadJson: `{}`,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	got, err := app.GetPendingSEPs()
	if err != nil {
		t.Fatalf("GetPendingSEPs: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("expected 1 pending, got %d", len(got))
	}
}

func TestApp_ConfirmSEPSync_UpdateStatus(t *testing.T) {
	app, _, _, _ := newTestApp(t)

	q := store.New(app.db)
	pending, _ := q.InsertPendingSEP(context.Background(), store.InsertPendingSEPParams{
		NoKartu: "0001", Kategori: "RUJUKAN", PayloadJson: `{}`,
	})

	if err := app.ConfirmSEPSync(pending.ID); err != nil {
		t.Fatalf("ConfirmSEPSync: %v", err)
	}

	awaiting, _ := q.GetPendingSEPs(context.Background(), store.GetPendingSEPsParams{
		Status: sql.NullString{String: "awaiting_sync", Valid: true},
		Limit:  10,
	})
	if len(awaiting) != 1 {
		t.Errorf("expected 1 awaiting_sync, got %d", len(awaiting))
	}
}

func TestApp_ResetCounters(t *testing.T) {
	app, _, k, _ := newTestApp(t)
	k.SetResponse("BuatAntrian", nil, domain.ErrOffline)

	for i := 0; i < 3; i++ {
		_, _ = app.CreateAntrian("LOKET", "")
	}

	if err := app.ResetCounters(); err != nil {
		t.Fatalf("ResetCounters: %v", err)
	}

	counters, _ := app.GetCounters()
	if counters["LOKET"] != 0 {
		t.Errorf("setelah ResetCounters, LOKET = %d, want 0", counters["LOKET"])
	}
}

// ============================================================
// Concurrency: cache lastPeserta thread-safe
// ============================================================

func TestApp_LastPesertaCache_ThreadSafe(t *testing.T) {
	app, v, _, _ := newTestApp(t)
	v.GetPesertaFunc = func(ctx context.Context, id string, tgl time.Time) (*domain.Peserta, error) {
		return &domain.Peserta{NoKartu: id, StatusAktif: "1"}, nil
	}

	const N = 30
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			_, _ = app.DetectPatient("X")
		}()
	}
	wg.Wait()

	app.mu.Lock()
	cached := app.lastPeserta
	app.mu.Unlock()
	if cached == nil {
		t.Errorf("expected cached peserta after concurrent calls")
	}
}

// ============================================================
// Diagnostik
// ============================================================

func TestApp_Greet(t *testing.T) {
	app, _, _, _ := newTestApp(t)
	got := app.Greet("APM")
	if got == "" {
		t.Errorf("Greet empty")
	}
}

func TestApp_emitEvent_NilCtx_NoOp(t *testing.T) {
	app := NewApp()
	app.logger = slog.Default()
	app.ctx = nil
	// Tidak panic
	app.emitEvent("test", "data")
}

// ============================================================
// Biometrik (call-based)
// ============================================================

func TestApp_VerifikasiWajah_Sukses(t *testing.T) {
	app, _, _, _ := newTestApp(t)

	token, err := app.VerifikasiWajah("0001234567890012")
	if err != nil {
		t.Fatalf("VerifikasiWajah: %v", err)
	}
	if token == "" {
		t.Error("token kosong")
	}
	if !contains(token, "MOCK_FACE") {
		t.Errorf("token harus mengandung MOCK_FACE prefix, got %q", token)
	}
}

func TestApp_VerifikasiWajah_NoPesertaKosong_Error(t *testing.T) {
	app, _, _, _ := newTestApp(t)

	_, err := app.VerifikasiWajah("")
	if err == nil {
		t.Error("noPeserta kosong harus error")
	}
}

func TestApp_VerifikasiWajah_FristaUnavail_Error(t *testing.T) {
	app, _, _, _ := newTestApp(t)
	if mfr, ok := app.hw.Frista.(*frista.MockVerifier); ok {
		mfr.SetAvailable(false)
	}

	_, err := app.VerifikasiWajah("X")
	if err == nil {
		t.Error("frista unavail harus error supaya frontend bisa fallback ke sidik jari")
	}
}

func TestApp_VerifikasiSidikJari_Sukses(t *testing.T) {
	app, _, _, _ := newTestApp(t)

	token, err := app.VerifikasiSidikJari("0001234567890012")
	if err != nil {
		t.Fatalf("VerifikasiSidikJari: %v", err)
	}
	if token == "" {
		t.Error("token kosong")
	}
	if !contains(token, "MOCK_FP") {
		t.Errorf("token harus mengandung MOCK_FP prefix, got %q", token)
	}
}

func TestApp_VerifikasiSidikJari_FpUnavail_Error(t *testing.T) {
	app, _, _, _ := newTestApp(t)
	if mfp, ok := app.hw.Fingerprint.(*fingerprint.MockVerifier); ok {
		mfp.SetAvailable(false)
	}

	_, err := app.VerifikasiSidikJari("X")
	if err == nil {
		t.Error("fingerprint unavail harus error supaya frontend bisa fallback ke wajah")
	}
}

// contains: helper string-search (test helpers package tidak tersedia
// di scope ini supaya tetap zero-import).
func contains(s, sub string) bool {
	if sub == "" {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
