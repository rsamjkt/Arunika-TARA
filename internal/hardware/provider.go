// Package hardware adalah dispatch layer untuk semua peripheral kiosk APM.
//
// SATU-SATUNYA tempat di codebase di mana runtime.GOOS check dilakukan.
// Service layer dan UI layer cukup pakai interface — implementasi dipilih
// otomatis oleh Provider berdasarkan platform.
//
// Pattern:
//
//	Mac/Linux (development): mock implementations
//	Windows (production):    real hardware implementations
//
// Selama P-031..P-033, real Windows implementations belum ada — stub di
// frista/windows.go, fingerprint/windows.go, printer/escpos.go sementara
// delegasi ke mock supaya Provider on Windows tidak crash. UI/service
// tests dapat dijalankan di Mac dengan mock identik.
package hardware

import (
	"database/sql"
	"runtime"

	"github.com/arunika/apm-go/internal/config"
	"github.com/arunika/apm-go/internal/hardware/fingerprint"
	"github.com/arunika/apm-go/internal/hardware/frista"
	"github.com/arunika/apm-go/internal/hardware/printer"
)

// Provider menyatukan 3 hardware layer + lifecycle management.
//
// Frista & Fingerprint sama-sama biometric verifier (call-based),
// frontend pilih salah satu (atau dua-duanya) sesuai preferensi pasien.
type Provider struct {
	Frista      frista.FaceVerifier
	Fingerprint fingerprint.FingerprintVerifier
	Printer     printer.ThermalPrinter
}

// NewProvider memilih implementasi sesuai platform host:
//
//	windows  → real (saat ini stub yang delegate ke mock — diganti P-031..P-033)
//	darwin   → mock + console printer
//	linux    → mock + console printer (CI / dev VM)
//
// db dipakai oleh ConsolePrinter untuk Reprint (perlu print_history).
// Boleh nil — Reprint akan return error kalau dipanggil.
func NewProvider(cfg config.Config, db *sql.DB) *Provider {
	// Printer dispatch:
	//  - Windows + mode=escpos_*  → real ESC/POS (USB / serial / network)
	//  - Windows + mode=console   → ConsolePrinter (admin override untuk
	//                               test tanpa printer fisik)
	//  - Mac/Linux                → ConsolePrinter (config mode di-abaikan,
	//                               kiosk tetap jalan tanpa printer)
	var printerImpl printer.ThermalPrinter
	if runtime.GOOS == "windows" && cfg.Printer.Mode != "console" {
		printerImpl = printer.NewESCPOS(cfg.Printer, db)
	} else {
		printerImpl = printer.NewConsolePrinter(db)
	}

	switch runtime.GOOS {
	case "windows":
		return &Provider{
			Frista:      frista.NewWindowsHeadless(cfg.Frista),
			Fingerprint: fingerprint.NewWindowsHeadless(cfg.Fingerprint),
			Printer:     printerImpl,
		}
	default: // darwin (Mac), linux
		return &Provider{
			Frista:      frista.NewMock(),
			Fingerprint: fingerprint.NewMock(),
			Printer:     printerImpl,
		}
	}
}

// Platform mengembalikan label platform yang sedang aktif.
// Dipakai admin panel + log header.
func (p *Provider) Platform() string {
	return runtime.GOOS
}

// IsRealHardware mengembalikan true jika running di Windows (mode
// production dengan hardware nyata), false jika di Mac/Linux dev.
func (p *Provider) IsRealHardware() bool {
	return runtime.GOOS == "windows"
}
