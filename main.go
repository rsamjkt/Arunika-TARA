// APM (T.A.R.A) — entry point Wails application.
//
// main.go hanya bootstrap Wails — semua logic ada di app.go (App struct
// dengan IPC bindings ke Vue) dan internal/* (service & integration layer).
package main

import (
	"embed"
	"log/slog"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// assets di-embed dari build frontend Vue (Vite output).
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	err := wails.Run(&options.App{
		Title:     "Anjungan Pasien Mandiri",
		Width:     1280,
		Height:    800,
		MinWidth:  1024,
		MinHeight: 768,

		// Production kiosk: Frameless + Fullscreen biasanya di-set
		// via config setelah startup (DOM-ready event). Untuk default
		// dev mode, window normal supaya developer bisa resize/test.
		Frameless:  false,
		Fullscreen: false,

		// #F5F6F8 — bg kiosk dari design system (DESIGN_SYSTEM.md)
		BackgroundColour: &options.RGBA{R: 245, G: 246, B: 248, A: 255},

		AssetServer: &assetserver.Options{Assets: assets},

		// Lifecycle hooks — App.startup wire semua dependency,
		// shutdown graceful close DB & hardware.
		OnStartup:  app.startup,
		OnShutdown: app.shutdown,

		// Bind expose semua method exported App ke Vue lewat
		// generated TypeScript di frontend/wailsjs/go/main/App.
		Bind: []interface{}{app},

		// Window logging via slog default (P-051 nanti pasok
		// PHIMaskingHandler global).
		Logger: nil,
	})
	if err != nil {
		slog.Error("wails run gagal", "err", err.Error())
	}
}
