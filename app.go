// Package main — Wails app bindings.
//
// File ini adalah jembatan tunggal antara Vue frontend dan Go service
// layer. Setiap method exported di App di-bind oleh Wails dan bisa
// dipanggil langsung dari Vue:
//
//	import { DetectPatient } from '../wailsjs/go/main/App'
//	const result = await DetectPatient("3271234567890001")
//
// Aturan bind:
//   - Method yang exported (huruf kapital) → otomatis di-bind oleh Wails
//   - Return signature wajib (data, error) — error otomatis di-throw di JS
//   - Argumen & return value harus JSON-serializable (struct/primitive)
//
// Catatan: file ini tidak dipindah ke cmd/apm/ supaya wails dev/build
// CLI bisa pickup main package dari root (Wails default behavior).
// CLAUDE.md menyebut cmd/apm/main.go sebagai sketch — yang penting
// kontrak App bindings sesuai spec.
package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/robfig/cron/v3"

	"github.com/arunika/apm-go/internal/config"
	"github.com/arunika/apm-go/internal/domain"
	"github.com/arunika/apm-go/internal/hardware"
	"github.com/arunika/apm-go/internal/integration/antrol"
	"github.com/arunika/apm-go/internal/integration/khanza"
	"github.com/arunika/apm-go/internal/integration/vclaim"
	"github.com/arunika/apm-go/internal/reconcile"
	"github.com/arunika/apm-go/internal/service/antrian"
	"github.com/arunika/apm-go/internal/service/detector"
	"github.com/arunika/apm-go/internal/service/sep"
	"github.com/arunika/apm-go/internal/store"
)

// ============================================================
// App struct — semua state Wails app
// ============================================================

// App adalah Wails-bound struct. Field unexported supaya tidak bocor
// ke JS (Wails hanya bind method).
type App struct {
	ctx       context.Context
	startedAt time.Time
	logger    *slog.Logger

	cfg *config.Config
	db  *sql.DB

	vclaim vclaim.VClaimClient
	antrol antrol.AntrolClient
	khanza khanza.KhanzaClient

	hw *hardware.Provider

	detectorSvc *detector.Detector
	antrianSvc  *antrian.AntrianService
	sepSvc      *sep.SEPService

	cron       *cron.Cron
	reconciler *reconcile.ReconcileWorker

	// Session cache — PHI-sensitive. Diset saat DetectPatient sukses,
	// dipakai BuatSEPxxx supaya UI tidak harus carry Peserta di payload.
	// Cleared di ResetSession() (idle timeout) atau session berakhir.
	mu          sync.Mutex
	lastPeserta *domain.Peserta
}

// NewApp membuat App kosong. Wiring sebenarnya di startup() — supaya
// Wails CLI bisa panggil NewApp() dari main.go tanpa pra-syarat
// (config file dll baru di-load saat startup).
func NewApp() *App {
	return &App{
		startedAt: time.Now(),
		logger:    slog.Default(),
	}
}

// startup di-call Wails saat window open. Inisialisasi semua dependency.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	if err := a.initialize(ctx); err != nil {
		a.logger.Error("app startup failed", "err", err)
		// Untuk kiosk: lanjut dengan dependency yang berhasil di-init.
		// UI akan show error state via GetSystemStatus.
	}
}

// shutdown di-call Wails saat window close. Graceful cleanup.
// Defensif terhadap nil — startup mungkin partial gagal tapi shutdown
// tetap dipanggil Wails.
func (a *App) shutdown(ctx context.Context) {
	if a.reconciler != nil {
		a.reconciler.Stop()
	}
	if a.cron != nil {
		a.cron.Stop()
	}
	if a.hw != nil {
		// Frista (face verifier) call-based — tidak ada Stop lifecycle.
		// Real Windows impl punya Stop() untuk kill frista.exe; di-call
		// via interface assertion supaya tidak butuh Stop di interface.
		if a.hw.Frista != nil {
			if stopper, ok := a.hw.Frista.(interface{ Stop() error }); ok {
				_ = stopper.Stop()
			}
		}
		if a.hw.Printer != nil {
			if stopper, ok := a.hw.Printer.(interface{ Stop() error }); ok {
				_ = stopper.Stop()
			}
		}
	}
	if a.db != nil {
		_ = a.db.Close()
	}
	a.logger.Info("app shutdown complete")
}

// initialize memuat config + spawn semua dependency. Dipisah dari
// startup supaya bisa dipanggil dari test (newAppForTest).
func (a *App) initialize(ctx context.Context) error {
	cfgPath := os.Getenv("APM_CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "config.toml"
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config %s: %w", cfgPath, err)
	}
	if err := cfg.Validate(); err != nil {
		a.logger.Warn("config validation warning (continue dengan default)",
			"err", err.Error())
	}
	a.cfg = cfg

	// Open DB + apply schema
	schemaPath := filepath.Join("migrations", "001_initial.sql")
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("baca schema %s: %w", schemaPath, err)
	}
	dbPath := "data/apm.db"
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	db, _, err := store.Open(ctx, dbPath, string(schemaBytes))
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	a.db = db

	// Clients
	if cfg.BPJS.Mock {
		a.vclaim = vclaim.NewMockPreset()
		a.logger.Warn("vclaim: MOCK mode aktif — semua call BPJS pakai canned response (jangan dipakai di production!)")
	} else {
		a.vclaim = vclaim.New(cfg.BPJS)
	}
	a.antrol = antrol.NewMock() // TODO P-021+: real Antrol HTTP client

	// Khanza: pilih implementasi berdasarkan config.
	//   khanza_dsn diisi → direct MySQL (pola anjunganmandiriSEP).
	//   else → Laravel REST (default).
	if cfg.Server.KhanzaDSN != "" {
		mysqlClient, mysqlErr := khanza.NewMySQL(cfg.Server)
		if mysqlErr != nil {
			a.logger.Error("khanza: gagal connect MySQL — fallback ke REST",
				"err", mysqlErr.Error())
			a.khanza = khanza.New(cfg.Server)
		} else {
			mysqlClient.SetLogger(a.logger)
			a.khanza = mysqlClient
			a.logger.Info("khanza: mode direct MySQL aktif")
		}
	} else {
		a.khanza = khanza.New(cfg.Server)
	}

	// Hardware provider — Mac dev: mocks; Windows: real impl.
	// Frista (face) & Fingerprint sama-sama call-based biometric verifier
	// — tidak ada Start/Stop atau channel forward; frontend panggil
	// VerifikasiWajah / VerifikasiSidikJari saat dibutuhkan.
	a.hw = hardware.NewProvider(*cfg, db)

	// Services
	a.detectorSvc = detector.New(a.vclaim, a.antrol, a.khanza)
	a.antrianSvc = antrian.New(a.khanza, db, a.antrol, cfg.Antrian)
	a.sepSvc = sep.New(a.vclaim, a.khanza, a.hw.Fingerprint, db)

	// Cron daily reset
	c, err := antrian.StartDailyReset(a.antrianSvc, "")
	if err != nil {
		a.logger.Warn("daily reset cron failed", "err", err.Error())
	}
	a.cron = c

	// Reconcile worker — track online/offline + sync backlog
	a.reconciler = reconcile.NewWithOptions(db, a.khanza, reconcile.Options{
		OnStateChange: func(online bool) {
			// Emit ke Vue: false = sedang offline, true = pulih
			a.emitEvent("system:offline", !online)
			a.logger.Info("reconcile state change", "online", online)
		},
	})
	a.reconciler.Start(ctx)

	a.logger.Info("app initialized",
		"platform", a.hw.Platform(),
		"real_hardware", a.hw.IsRealHardware())
	return nil
}

// emitEvent helper — emit Wails event dengan safe-guard ctx nil
// (test environment tidak punya Wails runtime).
func (a *App) emitEvent(name string, data ...any) {
	if a.ctx == nil {
		return
	}
	wailsruntime.EventsEmit(a.ctx, name, data...)
}

// ============================================================
// Detection
// ============================================================

// DetectPatient — Smart BPJS Detector entry point.
// Side effect: cache hasil Peserta untuk dipakai BuatSEPxxx berikutnya.
func (a *App) DetectPatient(identifier string) (*domain.DetectionResult, error) {
	if a.detectorSvc == nil {
		return nil, errors.New("detector belum diinisialisasi")
	}
	r := a.detectorSvc.Detect(a.ctx, domain.PatientInput{Identifier: identifier})

	if r.Peserta != nil {
		a.mu.Lock()
		a.lastPeserta = r.Peserta
		a.mu.Unlock()
	}
	return &r, nil
}

// ResetSession clear cached Peserta — dipanggil Vue saat idle timeout
// atau user klik "Mulai dari awal".
func (a *App) ResetSession() error {
	a.mu.Lock()
	a.lastPeserta = nil
	a.mu.Unlock()
	return nil
}

// ============================================================
// Antrian
// ============================================================

// CreateAntrian membuat satu nomor antrian + auto-print tiket +
// simpan ke print_history supaya bisa di-Reprint dari TicketScreen.
//
// Print error TIDAK menggagalkan return — ticket tetap diberikan ke
// pasien (mereka sudah dapat nomor). Error di-emit lewat event
// "printer:error" supaya UI bisa show toast.
func (a *App) CreateAntrian(jenis, subJenis string) (*domain.Ticket, error) {
	if a.antrianSvc == nil {
		return nil, errors.New("antrian service belum diinisialisasi")
	}
	ticket, err := a.antrianSvc.Create(a.ctx, domain.AntrianRequest{
		Jenis: jenis, SubJenis: subJenis,
	})
	if err != nil {
		return nil, err
	}
	a.printAndPersistTicket(ticket)
	return ticket, nil
}

// printAndPersistTicket non-blocking flow:
//   - call hw.Printer.Print untuk physical/console output
//   - InsertPrintHistory dengan ticket.ID sebagai ref_id supaya
//     ticket.PrintHistoryID dapat dipopulate untuk Reprint
//
// Untuk error apapun: log + emit "printer:error" event, TIDAK
// throw ke caller (pasien sudah dapat nomor — tidak boleh dibatalkan).
func (a *App) printAndPersistTicket(ticket *domain.Ticket) {
	if ticket == nil || a.hw == nil {
		return
	}

	// Step 1: physical/console print
	if a.hw.Printer != nil {
		if err := a.hw.Printer.Print(a.ctx, "TIKET", ticket); err != nil {
			a.logger.Warn("printer print tiket gagal", "err", err.Error())
			a.emitEvent("printer:error", err.Error())
		}
	}

	// Step 2: insert print_history untuk Reprint capability
	if a.db != nil {
		// Render bytes (placeholder JSON sampai P-050+ capture
		// raw ESC/POS bytes via printer.Print return value).
		bodyBytes, _ := json.Marshal(ticket)
		ph, err := store.New(a.db).InsertPrintHistory(a.ctx, store.InsertPrintHistoryParams{
			DocType:     "TIKET",
			RefID:       sql.NullString{String: ticket.ID, Valid: ticket.ID != ""},
			EscposBytes: bodyBytes,
		})
		if err != nil {
			a.logger.Warn("insert print_history tiket gagal",
				"err", err.Error(), "ticket_id", ticket.ID)
			return
		}
		ticket.PrintHistoryID = ph.ID
	}
}

// GetCounters return counter per jenis untuk display "Sekarang: A-035".
func (a *App) GetCounters() (map[string]int, error) {
	if a.antrianSvc == nil {
		return nil, errors.New("antrian service belum diinisialisasi")
	}
	out := make(map[string]int, 3)
	for _, jenis := range []string{
		string(domain.AntrianJenisLoket),
		string(domain.AntrianJenisPoli),
		string(domain.AntrianJenisUmum),
	} {
		c, err := a.antrianSvc.GetCounter(a.ctx, jenis)
		if err != nil {
			a.logger.Warn("GetCounter error", "jenis", jenis, "err", err.Error())
			continue
		}
		out[jenis] = c
	}
	return out, nil
}

// ============================================================
// SEP — semua method ambil Peserta dari cache (set by DetectPatient)
// ============================================================

// BuatSEPRujukan untuk pasien dengan rujukan FKTP (kunjungan baru).
func (a *App) BuatSEPRujukan(req domain.SEPRequest) (*domain.SEP, error) {
	p, err := a.requirePeserta()
	if err != nil {
		return nil, err
	}
	return a.sepSvc.BuatSEPRujukan(a.ctx, p, req)
}

// BuatSEPKontrol — pasien dengan SKDP. UI pilih dokter dari jadwal
// (kdDokter di payload), atau pakai default dari surat kontrol.
func (a *App) BuatSEPKontrol(noSuratKontrol, kdDokter string) (*domain.SEP, error) {
	p, err := a.requirePeserta()
	if err != nil {
		return nil, err
	}
	return a.sepSvc.BuatSEPKontrol(a.ctx, p, noSuratKontrol, kdDokter)
}

// BuatSEPPostRANAP — kontrol pasca rawat inap.
func (a *App) BuatSEPPostRANAP(kdPoliKontrol, kdDokter string) (*domain.SEP, error) {
	p, err := a.requirePeserta()
	if err != nil {
		return nil, err
	}
	return a.sepSvc.BuatSEPPostRANAP(a.ctx, p, kdPoliKontrol, kdDokter)
}

// BuatSEPPostRAJAL — lanjutan rawat jalan beda poli.
func (a *App) BuatSEPPostRAJAL(kdPoliTujuan, kdDokter string) (*domain.SEP, error) {
	p, err := a.requirePeserta()
	if err != nil {
		return nil, err
	}
	return a.sepSvc.BuatSEPPostRAJAL(a.ctx, p, kdPoliTujuan, kdDokter)
}

// requirePeserta ambil cached Peserta atau error kalau belum.
func (a *App) requirePeserta() (*domain.Peserta, error) {
	a.mu.Lock()
	p := a.lastPeserta
	a.mu.Unlock()
	if p == nil {
		return nil, errors.New("peserta belum di-detect — panggil DetectPatient dulu")
	}
	return p, nil
}

// ============================================================
// Biometrik (call-based) — frontend pilih wajah ATAU sidik jari
// ============================================================

// VerifikasiWajah memicu verifikasi sidik wajah via Frista (BPJS face
// recognition app). Blocking — frontend tampilkan modal "hadap kamera"
// sampai method return. Hasil token dilampirkan oleh frontend ke
// req.BiometrikToken di BuatSEP* berikutnya.
//
// Return:
//   - token (string non-kosong) + nil error → sukses
//   - "" + error → gagal (kamera tidak respon, pasien batal, dll)
//
// Frontend dapat menampilkan tombol "Coba lagi" atau switch ke
// VerifikasiSidikJari kalau wajah gagal.
func (a *App) VerifikasiWajah(noPeserta string) (string, error) {
	if a.hw == nil || a.hw.Frista == nil {
		return "", errors.New("frista (face verifier) belum diinisialisasi")
	}
	if !a.hw.Frista.IsAvailable() {
		return "", errors.New("frista (face verifier) tidak tersedia — coba VerifikasiSidikJari atau hubungi operator")
	}
	if strings.TrimSpace(noPeserta) == "" {
		return "", errors.New("noPeserta wajib diisi untuk VerifikasiWajah")
	}

	res, err := a.hw.Frista.Verify(a.ctx, noPeserta)
	if err != nil {
		a.logger.Warn("VerifikasiWajah gagal",
			"no_peserta_masked", maskCardForLog(noPeserta),
			"err", err.Error())
		return "", fmt.Errorf("verifikasi wajah: %w", err)
	}
	if !res.Success || res.Token == "" {
		return "", errors.New("verifikasi wajah tidak menghasilkan token")
	}
	a.logger.Info("VerifikasiWajah sukses",
		"no_peserta_masked", maskCardForLog(noPeserta),
		"token_len", len(res.Token))
	return res.Token, nil
}

// VerifikasiSidikJari memicu verifikasi sidik jari via After.exe (BPJS
// fingerprint app). Sama pattern dengan VerifikasiWajah. Return token
// untuk dilampirkan ke req.BiometrikToken di BuatSEP* berikutnya.
func (a *App) VerifikasiSidikJari(noPeserta string) (string, error) {
	if a.hw == nil || a.hw.Fingerprint == nil {
		return "", errors.New("fingerprint verifier belum diinisialisasi")
	}
	if !a.hw.Fingerprint.IsAvailable() {
		return "", errors.New("fingerprint verifier tidak tersedia — coba VerifikasiWajah atau hubungi operator")
	}
	if strings.TrimSpace(noPeserta) == "" {
		return "", errors.New("noPeserta wajib diisi untuk VerifikasiSidikJari")
	}

	res, err := a.hw.Fingerprint.Verify(a.ctx, noPeserta)
	if err != nil {
		a.logger.Warn("VerifikasiSidikJari gagal",
			"no_peserta_masked", maskCardForLog(noPeserta),
			"err", err.Error())
		return "", fmt.Errorf("verifikasi sidik jari: %w", err)
	}
	if !res.Success || res.Token == "" {
		return "", errors.New("verifikasi sidik jari tidak menghasilkan token")
	}
	a.logger.Info("VerifikasiSidikJari sukses",
		"no_peserta_masked", maskCardForLog(noPeserta),
		"token_len", len(res.Token))
	return res.Token, nil
}

// maskCardForLog memendekkan no_kartu jadi "************XXXX" untuk
// log (PHI safety). Duplikat kecil dari helper di paket lain — App
// tidak import dari service/ supaya menghindari cycle.
func maskCardForLog(s string) string {
	if len(s) < 8 {
		return "***"
	}
	out := make([]byte, len(s))
	for i := range out {
		if i >= len(s)-4 {
			out[i] = s[i]
		} else {
			out[i] = '*'
		}
	}
	return string(out)
}

// ============================================================
// Pendaftaran umum
// ============================================================

// CariPasien lookup pasien di Khanza berdasarkan q (nama/NIK/no RM).
func (a *App) CariPasien(q string) (*domain.Pasien, error) {
	if a.khanza == nil {
		return nil, errors.New("khanza client belum diinisialisasi")
	}
	return a.khanza.CariPasien(a.ctx, q)
}

// BuatPendaftaran daftar pasien ke poli (umum atau BPJS dengan SEP).
func (a *App) BuatPendaftaran(req domain.PendaftaranRequest) (*domain.Pendaftaran, error) {
	if a.khanza == nil {
		return nil, errors.New("khanza client belum diinisialisasi")
	}
	return a.khanza.BuatPendaftaran(a.ctx, req)
}

// GetJadwalDokter — list dokter praktik di poli pada tanggal ini.
// Dipakai DokterPickerScreen.
func (a *App) GetJadwalDokter(kdPoli string) ([]domain.JadwalDokter, error) {
	if a.khanza == nil {
		return nil, errors.New("khanza client belum diinisialisasi")
	}
	return a.khanza.GetJadwalDokter(a.ctx, kdPoli, time.Now())
}

// GetPoliklinikAktif — list poli aktif untuk picker pasien umum.
func (a *App) GetPoliklinikAktif() ([]domain.Poliklinik, error) {
	if a.khanza == nil {
		return nil, errors.New("khanza client belum diinisialisasi")
	}
	return a.khanza.GetPoliklinikAktif(a.ctx)
}

// ============================================================
// Branding (UI theming dari config.toml)
// ============================================================

// Branding adalah snapshot konfig branding RS yang dikonsumsi Vue
// untuk render header/logo + apply theme color via CSS variables.
type Branding struct {
	HospitalName    string `json:"hospital_name"`
	HospitalTagline string `json:"hospital_tagline"`
	LogoPath        string `json:"logo_path"`        // server-side absolute path
	LogoDataURL     string `json:"logo_data_url"`    // base64 data URL — Vue tampilkan langsung
	PrimaryColor    string `json:"primary_color"`
	PrimaryColorDark string `json:"primary_color_dark"`
	AccentColor     string `json:"accent_color"`
	BpjsLogoPath    string `json:"bpjs_logo_path"`
	BpjsLogoDataURL string `json:"bpjs_logo_data_url"`  // override default SVG kalau RS punya file
	AudioEnabled    bool   `json:"audio_enabled"`
	AudioVolume     float64 `json:"audio_volume"`
}

// GetBranding — return current branding config + logo as data URL kalau
// LogoPath di-set (Vue tinggal pakai di <img :src="branding.LogoDataURL">).
func (a *App) GetBranding() Branding {
	if a.cfg == nil {
		return defaultBranding()
	}
	b := a.cfg.Branding
	au := a.cfg.Audio

	out := Branding{
		HospitalName:    valueOrDefault(b.HospitalName, "Rumah Sakit"),
		HospitalTagline: valueOrDefault(b.HospitalTagline, "Anjungan Pasien Mandiri"),
		LogoPath:        b.LogoPath,
		PrimaryColor:    valueOrDefault(b.PrimaryColor, "#1B4FD8"),
		PrimaryColorDark: valueOrDefault(b.PrimaryColorDark, ""),
		AccentColor:     valueOrDefault(b.AccentColor, ""),
		AudioEnabled:    au.Enabled,
		AudioVolume:     au.Volume,
	}
	if out.AudioVolume <= 0 || out.AudioVolume > 1 {
		out.AudioVolume = 0.6
	}

	// Load logo file kalau ada
	if b.LogoPath != "" {
		if data, mime, err := readLogoAsDataURL(b.LogoPath); err == nil {
			out.LogoDataURL = "data:" + mime + ";base64," + data
		} else {
			a.logger.Warn("branding: gagal load logo", "path", b.LogoPath, "err", err.Error())
		}
	}
	// Load logo BPJS file kalau ada (override SVG default di BpjsLogo.vue)
	out.BpjsLogoPath = b.BpjsLogoPath
	if b.BpjsLogoPath != "" {
		if data, mime, err := readLogoAsDataURL(b.BpjsLogoPath); err == nil {
			out.BpjsLogoDataURL = "data:" + mime + ";base64," + data
		} else {
			a.logger.Warn("branding: gagal load logo BPJS", "path", b.BpjsLogoPath, "err", err.Error())
		}
	}
	return out
}

func defaultBranding() Branding {
	return Branding{
		HospitalName:    "Rumah Sakit",
		HospitalTagline: "Anjungan Pasien Mandiri",
		PrimaryColor:    "#1B4FD8",
		AudioEnabled:    true,
		AudioVolume:     0.6,
	}
}

func valueOrDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// readLogoAsDataURL baca file logo, return base64 + MIME type.
func readLogoAsDataURL(path string) (string, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	mime := "image/png"
	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case ".jpg", ".jpeg":
		mime = "image/jpeg"
	case ".svg":
		mime = "image/svg+xml"
	case ".webp":
		mime = "image/webp"
	}
	return base64.StdEncoding.EncodeToString(data), mime, nil
}

// ============================================================
// Hardware status
// ============================================================

// HardwareStatus per-device availability.
type HardwareStatus struct {
	Frista      bool `json:"frista"`
	Fingerprint bool `json:"fingerprint"`
	Printer     bool `json:"printer"`
}

// SystemStatus aggregate untuk admin panel & status panel kiosk.
type SystemStatus struct {
	Hardware  HardwareStatus `json:"hardware"`
	Online    bool           `json:"online"`     // overall (Khanza reachable)
	Platform  string         `json:"platform"`   // "darwin" / "windows"
	Version   string         `json:"version"`    // dari config.app.version
	UptimeSec int64          `json:"uptime_sec"` // detik sejak app start
	StartedAt string         `json:"started_at"` // ISO8601
}

// GetHardwareStatus — snapshot hardware availability.
func (a *App) GetHardwareStatus() HardwareStatus {
	if a.hw == nil {
		return HardwareStatus{}
	}
	return HardwareStatus{
		Frista:      a.hw.Frista != nil && a.hw.Frista.IsAvailable(),
		Fingerprint: a.hw.Fingerprint != nil && a.hw.Fingerprint.IsAvailable(),
		Printer:     a.hw.Printer != nil && a.hw.Printer.IsAvailable(),
	}
}

// GetSystemStatus — full snapshot untuk admin panel.
func (a *App) GetSystemStatus() SystemStatus {
	st := SystemStatus{
		Hardware:  a.GetHardwareStatus(),
		StartedAt: a.startedAt.Format(time.RFC3339),
		UptimeSec: int64(time.Since(a.startedAt).Seconds()),
	}
	if a.hw != nil {
		st.Platform = a.hw.Platform()
	}
	if a.cfg != nil {
		st.Version = a.cfg.App.Version
	}
	// Online: best-effort. Iterasi berikutnya P-050 reconcile worker
	// track real connectivity. Untuk sekarang asumsi true kalau khanza
	// client ada.
	st.Online = a.khanza != nil
	return st
}

// ============================================================
// Startup self-test
// ============================================================

// CheckResult — hasil satu probe yang ditampilkan di SplashScreen.
type CheckResult struct {
	Component  string `json:"component"`   // label user-facing ("Database lokal", "Khanza", ...)
	Status     string `json:"status"`      // "ok" | "warn" | "fail" | "skip"
	Message    string `json:"message"`     // detail pendek (1-line)
	Critical   bool   `json:"critical"`    // true = blokir lanjut ke HomeScreen kalau status=fail
	DurationMs int64  `json:"duration_ms"` // berapa lama probe selesai
}

// RunStartupChecks menjalankan probe semua dependency dan return
// list result. Dipanggil SplashScreen di Vue saat boot.
//
// Probes (urutan tampilan):
//  1. Database lokal (SQLite)         — critical
//  2. SIMRS Khanza                    — critical
//  3. BPJS VClaim                     — info-only (mock vs real)
//  4. Card reader (Frista)            — non-critical
//  5. Fingerprint (After.exe)         — non-critical
//  6. Thermal printer                 — non-critical
//
// Probes berjalan paralel — total wall-time umumnya <1 detik di Mac dev,
// <3 detik di Windows produksi (DB ping yang dominan).
func (a *App) RunStartupChecks() []CheckResult {
	ctx, cancel := context.WithTimeout(a.ctx, 5*time.Second)
	defer cancel()

	type slot struct {
		idx int
		res CheckResult
	}
	results := make([]CheckResult, 6)
	ch := make(chan slot, 6)

	probe := func(idx int, label string, critical bool, fn func() (string, string, string)) {
		start := time.Now()
		status, msg, _ := fn()
		ch <- slot{idx, CheckResult{
			Component:  label,
			Status:     status,
			Message:    msg,
			Critical:   critical,
			DurationMs: time.Since(start).Milliseconds(),
		}}
	}

	go probe(0, "Database lokal", true, func() (string, string, string) {
		if a.db == nil {
			return "fail", "DB belum diinisialisasi", ""
		}
		if err := a.db.PingContext(ctx); err != nil {
			return "fail", "ping gagal: " + err.Error(), ""
		}
		return "ok", "siap", ""
	})

	go probe(1, "SIMRS Khanza", true, func() (string, string, string) {
		if a.khanza == nil {
			return "fail", "client belum diinisialisasi", ""
		}
		if hc, ok := a.khanza.(interface {
			HealthCheck(context.Context) error
		}); ok {
			if err := hc.HealthCheck(ctx); err != nil {
				return "fail", "MySQL unreachable: " + err.Error(), ""
			}
			return "ok", "MySQL terhubung", ""
		}
		return "ok", "REST mode (no ping)", ""
	})

	go probe(2, "BPJS VClaim", false, func() (string, string, string) {
		if a.cfg != nil && a.cfg.BPJS.Mock {
			return "warn", "MOCK aktif (canned response)", ""
		}
		if a.cfg == nil || a.cfg.BPJS.ConsID == "" || a.cfg.BPJS.ConsumerSecret == "" {
			return "warn", "credential belum di-set", ""
		}
		return "ok", "credential ter-load", ""
	})

	go probe(3, "Card reader (Frista)", false, func() (string, string, string) {
		if a.hw == nil || a.hw.Frista == nil {
			return "skip", "tidak terpasang", ""
		}
		if a.hw.Frista.IsAvailable() {
			return "ok", "siap baca kartu", ""
		}
		return "warn", "offline — pasien input manual via NumPad", ""
	})

	go probe(4, "Fingerprint BPJS", false, func() (string, string, string) {
		if a.hw == nil || a.hw.Fingerprint == nil {
			return "skip", "tidak terpasang", ""
		}
		if a.hw.Fingerprint.IsAvailable() {
			return "ok", "siap verifikasi", ""
		}
		return "warn", "offline — biometrik akan di-skip", ""
	})

	go probe(5, "Thermal printer", false, func() (string, string, string) {
		if a.hw == nil || a.hw.Printer == nil {
			return "skip", "tidak terpasang", ""
		}
		if a.hw.Printer.IsAvailable() {
			return "ok", "siap cetak", ""
		}
		return "warn", "offline — periksa kertas/koneksi USB", ""
	})

	for i := 0; i < 6; i++ {
		s := <-ch
		results[s.idx] = s.res
	}
	return results
}

// ============================================================
// Reprint
// ============================================================

// Reprint dokumen dari print_history.id.
func (a *App) Reprint(printHistoryID int64) error {
	if a.hw == nil || a.hw.Printer == nil {
		return errors.New("printer belum diinisialisasi")
	}
	if err := a.hw.Printer.Reprint(a.ctx, printHistoryID); err != nil {
		a.emitEvent("printer:error", err.Error())
		return err
	}
	return nil
}

// ============================================================
// Admin
// ============================================================

// GetPendingSEPs — list SEP yang menunggu sync ke Khanza.
func (a *App) GetPendingSEPs() ([]store.PendingSep, error) {
	if a.db == nil {
		return nil, errors.New("db belum diinisialisasi")
	}
	q := store.New(a.db)
	return q.GetPendingSEPs(a.ctx, store.GetPendingSEPsParams{
		Status: sql.NullString{String: "pending", Valid: true},
		Limit:  100,
	})
}

// ConfirmSEPSync operator konfirmasi → status awaiting_sync.
// Reconcile worker (P-050) yang akan flush ke Khanza.
func (a *App) ConfirmSEPSync(id int64) error {
	if a.db == nil {
		return errors.New("db belum diinisialisasi")
	}
	q := store.New(a.db)
	return q.ConfirmSEP(a.ctx, store.ConfirmSEPParams{
		ConfirmedBy: sql.NullString{String: "operator-kiosk", Valid: true},
		ID:          id,
	})
}

// ResetCounters — admin trigger reset counter antrian.
func (a *App) ResetCounters() error {
	if a.antrianSvc == nil {
		return errors.New("antrian service belum diinisialisasi")
	}
	return a.antrianSvc.ResetAll(a.ctx)
}

// VerifyAdminPIN cocokkan input dengan cfg.Admin.PIN.
// Kalau cfg.Admin.PIN kosong, panel terbuka (return true) — untuk
// kemudahan dev. Production WAJIB set PIN di config.toml.
func (a *App) VerifyAdminPIN(pin string) bool {
	if a.cfg == nil {
		return false
	}
	configured := a.cfg.Admin.PIN
	if configured == "" {
		return true // panel tidak dilindungi
	}
	return pin == configured
}

// AdminLogEntry — wire format ReconcileLog untuk Vue.
type AdminLogEntry struct {
	ID         int64  `json:"id"`
	TableName  string `json:"table_name"`
	RecordID   int64  `json:"record_id"`
	Action     string `json:"action"`
	OperatorID string `json:"operator_id"`
	Result     string `json:"result"`
	Timestamp  string `json:"timestamp"`
}

// GetRecentLogs — 50 log rekonsiliasi terakhir untuk admin viewer.
func (a *App) GetRecentLogs(limit int64) ([]AdminLogEntry, error) {
	if a.db == nil {
		return nil, errors.New("db belum diinisialisasi")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := store.New(a.db).GetRecentLogs(a.ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("get recent logs: %w", err)
	}
	out := make([]AdminLogEntry, 0, len(rows))
	for _, r := range rows {
		ts := ""
		if r.Timestamp.Valid {
			ts = r.Timestamp.Time.Format(time.RFC3339)
		}
		out = append(out, AdminLogEntry{
			ID: r.ID, TableName: r.TableName, RecordID: r.RecordID,
			Action:     r.Action,
			OperatorID: r.OperatorID.String,
			Result:     r.Result.String,
			Timestamp:  ts,
		})
	}
	return out, nil
}

// AdminStats — angka untuk stat grid di admin panel.
type AdminStats struct {
	AntrianHariIni int    `json:"antrian_hari_ini"`     // total nomor antrian today
	SEPHariIni     int    `json:"sep_hari_ini"`         // jumlah SEP berhasil today
	PendingSync    int    `json:"pending_sync"`         // pending_sep status=pending
	UptimeSec      int64  `json:"uptime_sec"`
	StartedAt      string `json:"started_at"`
}

// GetAdminStats — agregat 4 angka untuk stat grid admin.
func (a *App) GetAdminStats() (*AdminStats, error) {
	st := &AdminStats{
		StartedAt: a.startedAt.Format(time.RFC3339),
		UptimeSec: int64(time.Since(a.startedAt).Seconds()),
	}

	// Antrian hari ini = sum counter per jenis (LOKET+POLI+UMUM)
	if a.antrianSvc != nil {
		for _, j := range []string{
			string(domain.AntrianJenisLoket),
			string(domain.AntrianJenisPoli),
			string(domain.AntrianJenisUmum),
		} {
			c, err := a.antrianSvc.GetCounter(a.ctx, j)
			if err == nil {
				st.AntrianHariIni += c
			}
		}
	}

	// Pending SEP = count yang status='pending'
	if a.db != nil {
		q := store.New(a.db)
		pending, err := q.GetPendingSEPs(a.ctx, store.GetPendingSEPsParams{
			Status: sql.NullString{String: "pending", Valid: true},
			Limit:  1000,
		})
		if err == nil {
			st.PendingSync = len(pending)
		}

		// SEP hari ini = count print_history doc_type='SEP'
		// Approximation - tidak ada query khusus, sementara skip dengan 0.
		// Iterasi P-050+ tambah query "count_sep_today".
	}
	return st, nil
}

// TestPrint cetak dokumen test untuk verifikasi printer fungsi.
// Dipanggil dari admin panel "Test cetak printer" button.
func (a *App) TestPrint() error {
	if a.hw == nil || a.hw.Printer == nil {
		return errors.New("printer belum diinisialisasi")
	}
	type testData struct {
		Title     string
		Timestamp string
		Message   string
	}
	data := testData{
		Title:     "TEST PRINT",
		Timestamp: time.Now().Format("2006-01-02 15:04:05 WIB"),
		Message:   "Jika Anda bisa membaca tulisan ini, printer berfungsi normal.",
	}
	if err := a.hw.Printer.Print(a.ctx, "TEST", data); err != nil {
		a.emitEvent("printer:error", err.Error())
		return fmt.Errorf("test print: %w", err)
	}
	a.logger.Info("admin: test print sukses")
	return nil
}

// ============================================================
// Diagnostik (test/dev only)
// ============================================================

// Greet legacy method dari scaffold Wails. Dipertahankan untuk
// smoke test "is App alive" dari Vue.
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s — APM (T.A.R.A) is ready.", name)
}
