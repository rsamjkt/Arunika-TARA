// APM (T.A.R.A) — entry point Wails application.
//
// main.go bootstrap Wails + setup global slog dengan PHIMaskingHandler.
// Plus CLI flags untuk operational task (encrypt-config) yang di-handle
// sebelum window dibuka.
package main

import (
	"embed"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/arunika/apm-go/internal/config"
	apmlog "github.com/arunika/apm-go/internal/log"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	encryptConfigPath := flag.String("encrypt-config", "",
		"Path config.toml untuk dienkripsi (interactive prompt). "+
			"Contoh: apm.exe --encrypt-config config.toml")
	flag.Parse()

	// CLI mode: encrypt-config — interactive, exit setelah selesai.
	if *encryptConfigPath != "" {
		if err := config.EncryptConfig(*encryptConfigPath); err != nil {
			fmt.Fprintf(os.Stderr, "❌ EncryptConfig: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Setup global slog dengan PHI masking sebelum apapun yang bisa log.
	setupSlog()

	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "Anjungan Pasien Mandiri",
		Width:     1280,
		Height:    800,
		MinWidth:  1024,
		MinHeight: 768,

		// Production kiosk: Frameless + Fullscreen biasanya di-set
		// via config setelah startup (DOM-ready event). Default dev
		// mode normal supaya developer bisa resize/test.
		Frameless:  false,
		Fullscreen: false,

		// #F5F6F8 — bg kiosk dari design system
		BackgroundColour: &options.RGBA{R: 245, G: 246, B: 248, A: 255},

		AssetServer: &assetserver.Options{Assets: assets},

		OnStartup:  app.startup,
		OnShutdown: app.shutdown,

		Bind: []interface{}{app},

		// Wails internal logger pass nil — kita pakai slog yang sudah
		// PHI-mask. Wails default logger ke stdout untuk dev.
		Logger: nil,
	})
	if err != nil {
		slog.Error("wails run gagal", "err", err.Error())
	}
}

// setupSlog inisialisasi slog.Default dengan PHIMaskingHandler.
//
// Output:
//   - APM_APP_LOG_DIR set + writable → JSON file di logs/apm.log
//   - Fallback → stdout text format (dev visibility)
//
// PHI masking ALWAYS aktif — JANGAN pernah bypass handler ini di
// production. Production deployment WAJIB cek log file tidak punya
// PHI raw.
func setupSlog() {
	logDir := os.Getenv("APM_APP_LOG_DIR")
	if logDir == "" {
		logDir = "logs"
	}

	var inner slog.Handler
	if err := os.MkdirAll(logDir, 0o755); err == nil {
		path := filepath.Join(logDir, "apm.log")
		f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err == nil {
			inner = slog.NewJSONHandler(f, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			})
		}
	}
	if inner == nil {
		inner = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	logger := slog.New(apmlog.NewPHIMaskingHandler(inner))
	slog.SetDefault(logger)
}
