package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validTOML = `
[app]
idle_timeout_sec = 60
log_level        = "info"
log_dir          = "./logs"
timezone         = "Asia/Jakarta"
version          = "1.0.0"

[server]
khanza_url      = "http://192.168.1.10:8080"
khanza_api_key  = "test-token"
timeout_ms      = 10000
retry           = 2

[bpjs]
vclaim_url           = "https://dvlp.bpjs-kesehatan.go.id/vclaim-rest/"
cons_id              = "12345"
consumer_secret      = "secret-xyz"
antrol_url           = "https://apijkn.bpjs-kesehatan.go.id/antrean-rest/"
detector_timeout_ms  = 5000

[fingerprint]
exe_path           = "C:\\fp\\After.exe"
rest_url           = "https://fp.bpjs-kesehatan.go.id/finger-rest/"
username_enc       = "fp-user"
password_enc       = "fp-pass"
scan_timeout_sec   = 30
poll_interval_ms   = 500

[frista]
exe_path          = "C:\\Frista\\frista.exe"
rest_url          = "https://frista.bpjs-kesehatan.go.id/face-rest/"
username_enc      = "frista-user"
password_enc      = "frista-pass"
scan_timeout_sec  = 30
poll_interval_ms  = 500

[printer]
mode      = "console"
port      = "POS-58"
width_mm  = 58

[antrian]
loket_prefix = "A"
poli_prefix  = "B"
umum_prefix  = "C"
reset_time   = "00:01"

[admin]
pin = "1234"

[dev]
mock_hardware    = true
mock_server_port = 9090
`

// writeTempTOML menulis konten ke file sementara dan return path.
func writeTempTOML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

func TestLoad_ValidTOML(t *testing.T) {
	path := writeTempTOML(t, validTOML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Spot-check beberapa field dari setiap section
	if cfg.App.IdleTimeoutSec != 60 {
		t.Errorf("app.idle_timeout_sec = %d, want 60", cfg.App.IdleTimeoutSec)
	}
	if cfg.App.Timezone != "Asia/Jakarta" {
		t.Errorf("app.timezone = %q, want Asia/Jakarta", cfg.App.Timezone)
	}
	if cfg.Server.KhanzaURL != "http://192.168.1.10:8080" {
		t.Errorf("server.khanza_url salah: %q", cfg.Server.KhanzaURL)
	}
	if cfg.Server.TimeoutMs != 10000 {
		t.Errorf("server.timeout_ms = %d, want 10000", cfg.Server.TimeoutMs)
	}
	if cfg.BPJS.ConsID != "12345" {
		t.Errorf("bpjs.cons_id = %q", cfg.BPJS.ConsID)
	}
	if cfg.BPJS.DetectorTimeoutMs != 5000 {
		t.Errorf("bpjs.detector_timeout_ms = %d, want 5000", cfg.BPJS.DetectorTimeoutMs)
	}
	if cfg.Frista.ScanTimeoutSec != 30 {
		t.Errorf("frista.scan_timeout_sec = %d, want 30", cfg.Frista.ScanTimeoutSec)
	}
	if cfg.Frista.RestURL != "https://frista.bpjs-kesehatan.go.id/face-rest/" {
		t.Errorf("frista.rest_url = %q", cfg.Frista.RestURL)
	}
	if cfg.Printer.WidthMm != 58 {
		t.Errorf("printer.width_mm = %d, want 58", cfg.Printer.WidthMm)
	}
	if cfg.Antrian.LoketPrefix != "A" {
		t.Errorf("antrian.loket_prefix = %q, want A", cfg.Antrian.LoketPrefix)
	}
	if !cfg.Dev.MockHardware {
		t.Errorf("dev.mock_hardware harus true")
	}
}

func TestLoad_FileTidakAda(t *testing.T) {
	_, err := Load("/nonexistent/config.toml")
	if err == nil {
		t.Fatal("Load() expected error untuk file tidak ada, got nil")
	}
}

func TestLoad_TOMLInvalid(t *testing.T) {
	path := writeTempTOML(t, "this is = not valid TOML [[")
	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() expected error untuk TOML invalid, got nil")
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("APM_BPJS_CONS_ID", "from-env-12345")
	t.Setenv("APM_DEV_MOCK_HARDWARE", "false")

	path := writeTempTOML(t, validTOML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.BPJS.ConsID != "from-env-12345" {
		t.Errorf("env override gagal: bpjs.cons_id = %q, want from-env-12345", cfg.BPJS.ConsID)
	}
	if cfg.Dev.MockHardware != false {
		t.Errorf("env override gagal: dev.mock_hardware = %v, want false", cfg.Dev.MockHardware)
	}
}

func TestValidate_Valid(t *testing.T) {
	path := writeTempTOML(t, validTOML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestValidate_NilConfig(t *testing.T) {
	var cfg *Config
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() pada nil config expected error, got nil")
	}
}

func TestValidate_MissingFields(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(c *Config)
		mustHave string // substring yang harus muncul di pesan error
	}{
		{
			name:     "khanza_url_kosong",
			mutate:   func(c *Config) { c.Server.KhanzaURL = "" },
			mustHave: "server.khanza_url",
		},
		{
			name:     "cons_id_kosong",
			mutate:   func(c *Config) { c.BPJS.ConsID = "" },
			mustHave: "bpjs.cons_id",
		},
		{
			name:     "detector_timeout_nol",
			mutate:   func(c *Config) { c.BPJS.DetectorTimeoutMs = 0 },
			mustHave: "bpjs.detector_timeout_ms",
		},
		{
			name:     "loket_prefix_kosong",
			mutate:   func(c *Config) { c.Antrian.LoketPrefix = "" },
			mustHave: "antrian.loket_prefix",
		},
		{
			name:     "printer_mode_kosong",
			mutate:   func(c *Config) { c.Printer.Mode = "" },
			mustHave: "printer.mode",
		},
		{
			name:     "idle_timeout_negatif",
			mutate:   func(c *Config) { c.App.IdleTimeoutSec = -1 },
			mustHave: "app.idle_timeout_sec",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempTOML(t, validTOML)
			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			tt.mutate(cfg)
			err = cfg.Validate()
			if err == nil {
				t.Fatalf("Validate() expected error karena %s, got nil", tt.name)
			}
			if !errors.Is(err, ErrConfigInvalid) {
				t.Errorf("error harus wrap ErrConfigInvalid, got %v", err)
			}
			if !strings.Contains(err.Error(), tt.mustHave) {
				t.Errorf("pesan error %q tidak mengandung %q", err.Error(), tt.mustHave)
			}
		})
	}
}

func TestValidate_AggregateMultipleErrors(t *testing.T) {
	path := writeTempTOML(t, validTOML)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	cfg.Server.KhanzaURL = ""
	cfg.BPJS.ConsID = ""
	cfg.Antrian.LoketPrefix = ""

	err = cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error")
	}
	for _, frag := range []string{"server.khanza_url", "bpjs.cons_id", "antrian.loket_prefix"} {
		if !strings.Contains(err.Error(), frag) {
			t.Errorf("error message harus melaporkan semua field yang missing — tidak ada %q dalam: %s",
				frag, err.Error())
		}
	}
}
