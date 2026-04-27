// Package config memuat dan memvalidasi konfigurasi APM dari file TOML.
//
// Loader pakai Viper sehingga env var bisa override field individual:
//
//	APM_BPJS_CONS_ID=xxxx → mengganti [bpjs] cons_id
//	APM_DEV_MOCK_HARDWARE=true → mengganti [dev] mock_hardware
package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config adalah top-level config struct. Setiap section di config.toml
// dipetakan ke satu sub-struct.
type Config struct {
	App         AppConfig         `mapstructure:"app"`
	Branding    BrandingConfig    `mapstructure:"branding"`
	Server      ServerConfig      `mapstructure:"server"`
	BPJS        BPJSConfig        `mapstructure:"bpjs"`
	Fingerprint FingerprintConfig `mapstructure:"fingerprint"`
	Frista      FristaConfig      `mapstructure:"frista"`
	Printer     PrinterConfig     `mapstructure:"printer"`
	Antrian     AntrianConfig     `mapstructure:"antrian"`
	Admin       AdminConfig       `mapstructure:"admin"`
	Audio       AudioConfig       `mapstructure:"audio"`
	Dev         DevConfig         `mapstructure:"dev"`
}

// AppConfig — pengaturan global aplikasi.
type AppConfig struct {
	IdleTimeoutSec int    `mapstructure:"idle_timeout_sec"`
	LogLevel       string `mapstructure:"log_level"`
	LogDir         string `mapstructure:"log_dir"`
	Timezone       string `mapstructure:"timezone"`
	Version        string `mapstructure:"version"`
}

// ServerConfig — koneksi ke SIMRS Khanza.
//
// Dua mode dukungan:
//   - REST (default): KhanzaURL + KhanzaAPIKey → Laravel API Khanza.
//   - Direct MySQL: KhanzaDSN diisi → tembak langsung ke DB Khanza
//     (mengikuti pola repo referensi RS-INDRIATI/anjunganmandiriSEP).
//
// Kalau KhanzaDSN non-kosong, App.initialize akan pakai khanza.NewMySQL().
type ServerConfig struct {
	KhanzaURL    string `mapstructure:"khanza_url"`
	KhanzaAPIKey string `mapstructure:"khanza_api_key"`
	TimeoutMs    int    `mapstructure:"timeout_ms"`
	Retry        int    `mapstructure:"retry"`

	// KhanzaDSN — Go MySQL DSN, contoh:
	//   "user:pass@tcp(10.0.2.121:3306)/sikrsam260312?parseTime=true&loc=Local&timeout=5s"
	// Kosong = pakai REST. Non-kosong = mode direct-DB.
	// Kalau dimulai dengan "ENC:", akan didekripsi via master key (P-051).
	KhanzaDSN string `mapstructure:"khanza_dsn"`

	// KhanzaKdPjUmum — kode penjamin "Umum" di tabel penjab Khanza RS ini.
	// Default kosong → MySQLClient pakai "A03" (umum di sikrsam260312).
	KhanzaKdPjUmum string `mapstructure:"khanza_kd_pj_umum"`

	// KhanzaKdPjBPJS — kode penjamin "BPJS" di tabel penjab Khanza RS ini.
	// Default kosong → MySQLClient pakai "BPJ".
	KhanzaKdPjBPJS string `mapstructure:"khanza_kd_pj_bpjs"`
}

// BPJSConfig — kredensial dan endpoint BPJS (VClaim + Antrol).
//
// Cons ID & ConsumerSecret dipakai untuk HMAC-SHA256 signing
// (header X-cons-id + X-signature).
//
// UserKey wajib di header `user_key` untuk decrypt response BPJS
// (AES-256-CBC). Kalau kosong, endpoint production akan reject
// atau return blank — selalu isi sesuai issued credential dari BPJS.
type BPJSConfig struct {
	VClaimURL         string `mapstructure:"vclaim_url"`
	ConsID            string `mapstructure:"cons_id"`
	ConsumerSecret    string `mapstructure:"consumer_secret"`
	UserKey           string `mapstructure:"user_key"`
	AntrolURL         string `mapstructure:"antrol_url"`
	DetectorTimeoutMs int    `mapstructure:"detector_timeout_ms"`
}

// FingerprintConfig — After.exe (BPJS Sidik Jari) integrasi.
// UsernameEnc & PasswordEnc adalah cipher AES-256-GCM (lihat P-051).
//
// Field WindowClass* opsional — pakai default Delphi VCL kalau kosong.
// Override hanya kalau RS pakai After.exe versi non-standar dengan
// class name berbeda (verify via Spy++ di Windows kiosk).
type FingerprintConfig struct {
	ExePath        string `mapstructure:"exe_path"`
	RestURL        string `mapstructure:"rest_url"`
	UsernameEnc    string `mapstructure:"username_enc"`
	PasswordEnc    string `mapstructure:"password_enc"`
	ScanTimeoutSec int    `mapstructure:"scan_timeout_sec"`
	PollIntervalMs int    `mapstructure:"poll_interval_ms"`

	// Class names dialog login After.exe untuk auto-login injection.
	// Default Delphi VCL: TfrmLogin / TEdit / TButton.
	WindowClassLogin    string `mapstructure:"window_class_login"`
	WindowClassEdit     string `mapstructure:"window_class_edit"`
	WindowClassButton   string `mapstructure:"window_class_button"`
	StartupDelaySec     int    `mapstructure:"startup_delay_sec"` // wait setelah spawn sebelum inject (default 3s)
}

// FristaConfig — card reader Frista (BPJS desktop app).
//
// Pola interaksi (mirror DlgRegistrasiSEPPertama.java reference repo):
//  1. Spawn frista.exe (CREATE_NO_WINDOW kalau supported)
//  2. Auto-login: inject UsernameEnc + PasswordEnc via Win32 UI Automation
//  3. Pasien tap KTP/kartu BPJS → Frista taruh JSON di clipboard
//  4. APM polling clipboard → parse → emit ke CardRead channel
//
// Field WindowClass* opsional — Frista pakai class name Delphi VCL
// standar (TfrmLogin/TEdit/TButton). Override hanya kalau versi beda.
//
// PollIntervalMs: berapa sering polling clipboard (default 500ms).
type FristaConfig struct {
	ExePath        string `mapstructure:"exe_path"`
	UsernameEnc    string `mapstructure:"username_enc"`
	PasswordEnc    string `mapstructure:"password_enc"`
	ReadTimeoutMs  int    `mapstructure:"read_timeout_ms"`
	RestartOnCrash bool   `mapstructure:"restart_on_crash"`

	// UI Automation — class names dialog login Frista.
	WindowClassLogin  string `mapstructure:"window_class_login"`
	WindowClassEdit   string `mapstructure:"window_class_edit"`
	WindowClassButton string `mapstructure:"window_class_button"`
	StartupDelaySec   int    `mapstructure:"startup_delay_sec"` // wait setelah spawn (default 5s — Frista lebih lambat dari After.exe)

	// Clipboard polling untuk capture card data.
	PollIntervalMs int `mapstructure:"poll_interval_ms"` // default 500ms
}

// PrinterConfig — thermal printer.
// Mode: "console" (Mac dev) | "escpos_usb" | "escpos_serial" | "escpos_network".
type PrinterConfig struct {
	Mode    string `mapstructure:"mode"`
	Port    string `mapstructure:"port"`
	WidthMm int    `mapstructure:"width_mm"`
}

// AntrianConfig — prefix per jenis dan jam reset harian.
type AntrianConfig struct {
	LoketPrefix string `mapstructure:"loket_prefix"`
	PoliPrefix  string `mapstructure:"poli_prefix"`
	UmumPrefix  string `mapstructure:"umum_prefix"`
	ResetTime   string `mapstructure:"reset_time"` // "HH:MM" WIB
}

// BrandingConfig — identitas visual RS yang ditampilkan di kiosk.
//
// LogoPath: path absolut ke file logo (PNG/SVG/JPG). Kalau kosong,
// pakai default mark "T" generik di header.
//
// HospitalName + HospitalTagline: di-render di header HomeScreen.
//
// PrimaryColor: warna utama tombol/aksen (CSS hex). Default biru
// korporat #1B4FD8 — ganti ke teal RS Anggrek Mas (#00897B) atau
// warna RS-mu sendiri.
//
// PrimaryColorDark: hover/active state (default = darken 12% dari Primary).
//
// AccentColor: warna sekunder untuk badges, info banners (default
// turunan dari Primary).
type BrandingConfig struct {
	LogoPath          string `mapstructure:"logo_path"`
	HospitalName      string `mapstructure:"hospital_name"`
	HospitalTagline   string `mapstructure:"hospital_tagline"`
	PrimaryColor      string `mapstructure:"primary_color"`
	PrimaryColorDark  string `mapstructure:"primary_color_dark"`
	AccentColor       string `mapstructure:"accent_color"`

	// BpjsLogoPath — opsional override logo BPJS Kesehatan dengan file resmi
	// dari brosur RS. Kalau kosong, pakai SVG default di BpjsLogo.vue.
	BpjsLogoPath string `mapstructure:"bpjs_logo_path"`
}

// AudioConfig — kontrol audio cue (tap/success/error sound).
//
// Enabled: master switch. False = silent kiosk.
// Volume: 0.0–1.0 (default 0.6).
type AudioConfig struct {
	Enabled bool    `mapstructure:"enabled"`
	Volume  float64 `mapstructure:"volume"`
}

// AdminConfig — pengaturan akses admin panel.
type AdminConfig struct {
	// PIN 4-6 digit untuk akses admin panel. Plaintext saat ini —
	// production-grade hashing (bcrypt) dapat ditambah saat P-051
	// security hardening. Default kosong = admin panel terbuka
	// tanpa PIN (untuk dev convenience).
	PIN string `mapstructure:"pin"`
}

// DevConfig — flag pengembangan (hanya berlaku di non-Windows).
type DevConfig struct {
	MockHardware   bool `mapstructure:"mock_hardware"`
	MockServerPort int  `mapstructure:"mock_server_port"`
}

// Load membaca file TOML di path lalu mengembalikan Config yang sudah ter-unmarshal.
// Env var dengan prefix APM_ otomatis di-bind (titik diganti underscore):
// APM_BPJS_CONS_ID akan menimpa bpjs.cons_id.
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("toml")

	v.SetEnvPrefix("APM")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("baca config %q: %w", path, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Auto-decrypt field bertanda "ENC:..." kalau master key tersedia.
	// Kalau key tidak ada, biarkan field tetap "ENC:..." — caller akan
	// fail saat coba pakai (mis. spawn Frista dengan password yang
	// masih encrypted) dengan error message yang jelas.
	if err := decryptInPlace(&cfg); err != nil {
		// Log warning tapi continue — bisa jadi field tidak ada yang
		// encrypted (config plaintext mode untuk dev).
		// Caller bisa cek via IsEncrypted() di field individual.
		_ = err
	}
	return &cfg, nil
}

// decryptInPlace ganti semua field "ENC:..." dengan plaintext-nya.
// Field yang BUKAN ENC: dibiarkan apa adanya. Master key resolved
// satu kali — kalau gagal resolve, return error tapi field-field
// ENC: tetap tidak di-decrypt (caller akan terima string raw "ENC:..."
// yang akan trigger error saat dipakai).
func decryptInPlace(cfg *Config) error {
	// Lazy resolve master key — hanya kalau ada field ENC:
	hasEncrypted := IsEncrypted(cfg.Fingerprint.UsernameEnc) ||
		IsEncrypted(cfg.Fingerprint.PasswordEnc) ||
		IsEncrypted(cfg.Frista.UsernameEnc) ||
		IsEncrypted(cfg.Frista.PasswordEnc) ||
		IsEncrypted(cfg.Server.KhanzaDSN)
	if !hasEncrypted {
		return nil
	}

	masterKey, err := GetMasterKey()
	if err != nil {
		return fmt.Errorf("master key untuk decrypt: %w", err)
	}

	mustDecrypt := func(s string) string {
		if !IsEncrypted(s) {
			return s
		}
		plain, derr := DecryptValue(s, masterKey)
		if derr != nil {
			// Biarkan tetap "ENC:..." — caller bisa cek
			return s
		}
		return plain
	}
	cfg.Fingerprint.UsernameEnc = mustDecrypt(cfg.Fingerprint.UsernameEnc)
	cfg.Fingerprint.PasswordEnc = mustDecrypt(cfg.Fingerprint.PasswordEnc)
	cfg.Frista.UsernameEnc = mustDecrypt(cfg.Frista.UsernameEnc)
	cfg.Frista.PasswordEnc = mustDecrypt(cfg.Frista.PasswordEnc)
	cfg.Server.KhanzaDSN = mustDecrypt(cfg.Server.KhanzaDSN)
	return nil
}

// ErrConfigInvalid dikembalikan oleh Validate jika ada field wajib yang kosong.
var ErrConfigInvalid = errors.New("konfigurasi tidak valid")

// Validate memeriksa field wajib non-kosong. Field yang khusus Windows
// (frista.exe_path, fingerprint.exe_path) tidak dipaksakan karena di Mac
// hardware di-mock — provider akan mengabaikannya.
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("%w: config nil", ErrConfigInvalid)
	}

	var missing []string
	check := func(name, val string) {
		if strings.TrimSpace(val) == "" {
			missing = append(missing, name)
		}
	}

	// App
	if c.App.IdleTimeoutSec <= 0 {
		missing = append(missing, "app.idle_timeout_sec (>0)")
	}
	check("app.log_level", c.App.LogLevel)
	check("app.timezone", c.App.Timezone)

	// Server — wajib salah satu: KhanzaURL (REST) ATAU KhanzaDSN (direct MySQL).
	if strings.TrimSpace(c.Server.KhanzaURL) == "" &&
		strings.TrimSpace(c.Server.KhanzaDSN) == "" {
		missing = append(missing, "server.khanza_url ATAU server.khanza_dsn")
	}
	if c.Server.TimeoutMs <= 0 {
		missing = append(missing, "server.timeout_ms (>0)")
	}

	// BPJS
	check("bpjs.vclaim_url", c.BPJS.VClaimURL)
	check("bpjs.cons_id", c.BPJS.ConsID)
	check("bpjs.consumer_secret", c.BPJS.ConsumerSecret)
	if c.BPJS.DetectorTimeoutMs <= 0 {
		missing = append(missing, "bpjs.detector_timeout_ms (>0)")
	}

	// Antrian
	check("antrian.loket_prefix", c.Antrian.LoketPrefix)
	check("antrian.poli_prefix", c.Antrian.PoliPrefix)
	check("antrian.umum_prefix", c.Antrian.UmumPrefix)
	check("antrian.reset_time", c.Antrian.ResetTime)

	// Printer
	check("printer.mode", c.Printer.Mode)
	if c.Printer.WidthMm <= 0 {
		missing = append(missing, "printer.width_mm (>0)")
	}

	if len(missing) > 0 {
		return fmt.Errorf("%w: field wajib kosong/invalid: %s",
			ErrConfigInvalid, strings.Join(missing, ", "))
	}
	return nil
}
