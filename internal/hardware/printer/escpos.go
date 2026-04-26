package printer

import (
	"database/sql"

	"github.com/arunika/apm-go/internal/config"
)

// NewESCPOS sementara delegasi ke ConsolePrinter sampai real impl
// ESC/POS USB/Serial/Network di-implement di P-033.
//
// Saat real impl masuk, function ini akan:
//   - Open koneksi sesuai cfg.Mode (escpos_usb / escpos_serial /
//     escpos_network)
//   - Render template Go text/template (templates/*.tmpl)
//   - Convert ke ESC/POS bytes (ESC @, ESC E, GS V 0, dll)
//   - Write ke koneksi
//
// File ini SENGAJA tidak diberi build tag — function harus exist
// di semua platform supaya provider.go (yang non-tagged) bisa
// merefer. Real impl akan dipindah ke escpos_actual.go (build tag
// kalau perlu OS-specific port handling).
func NewESCPOS(cfg config.PrinterConfig, db *sql.DB) ThermalPrinter {
	// TODO P-033: ganti dengan ESCPOSPrinter sebenarnya
	return NewConsolePrinter(db)
}
