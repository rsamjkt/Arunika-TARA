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
	Server      ServerConfig      `mapstructure:"server"`
	BPJS        BPJSConfig        `mapstructure:"bpjs"`
	Fingerprint FingerprintConfig `mapstructure:"fingerprint"`
	Frista      FristaConfig      `mapstructure:"frista"`
	Printer     PrinterConfig     `mapstructure:"printer"`
	Antrian     AntrianConfig     `mapstructure:"antrian"`
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

// ServerConfig — koneksi ke SIMRS Khanza (Laravel REST).
type ServerConfig struct {
	KhanzaURL    string `mapstructure:"khanza_url"`
	KhanzaAPIKey string `mapstructure:"khanza_api_key"`
	TimeoutMs    int    `mapstructure:"timeout_ms"`
	Retry        int    `mapstructure:"retry"`
}

// BPJSConfig — kredensial dan endpoint BPJS (VClaim + Antrol).
// Cons ID & secret dipakai untuk HMAC-SHA256 signing.
type BPJSConfig struct {
	VClaimURL         string `mapstructure:"vclaim_url"`
	ConsID            string `mapstructure:"cons_id"`
	ConsumerSecret    string `mapstructure:"consumer_secret"`
	AntrolURL         string `mapstructure:"antrol_url"`
	DetectorTimeoutMs int    `mapstructure:"detector_timeout_ms"`
}

// FingerprintConfig — After.exe (BPJS Sidik Jari) integrasi.
// UsernameEnc & PasswordEnc adalah cipher AES-256-GCM (lihat P-051).
type FingerprintConfig struct {
	ExePath        string `mapstructure:"exe_path"`
	RestURL        string `mapstructure:"rest_url"`
	UsernameEnc    string `mapstructure:"username_enc"`
	PasswordEnc    string `mapstructure:"password_enc"`
	ScanTimeoutSec int    `mapstructure:"scan_timeout_sec"`
	PollIntervalMs int    `mapstructure:"poll_interval_ms"`
}

// FristaConfig — card reader Frista.
type FristaConfig struct {
	ExePath        string `mapstructure:"exe_path"`
	UsernameEnc    string `mapstructure:"username_enc"`
	PasswordEnc    string `mapstructure:"password_enc"`
	ReadTimeoutMs  int    `mapstructure:"read_timeout_ms"`
	RestartOnCrash bool   `mapstructure:"restart_on_crash"`
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
	return &cfg, nil
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

	// Server
	check("server.khanza_url", c.Server.KhanzaURL)
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
