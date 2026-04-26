// Package printer adalah abstraksi printer thermal kiosk APM.
//
// Status implementasi:
//   - interface.go (file ini) — kontrak yang dipakai service layer + Wails app
//   - console.go               — Mac/Linux dev mock — render ke stdout
//   - escpos.go                — stub yang sementara delegasi ke ConsolePrinter;
//                                real ESC/POS USB/Serial impl di P-033
package printer

import "context"

// ThermalPrinter adalah surface API yang dipakai service layer.
//
// Print: render dokumen + kirim ke output (stdout / printer fisik).
// Reprint: ambil bytes lama dari print_history + render ulang +
//          increment reprint_count (audit).
type ThermalPrinter interface {
	// Print render dokumen sesuai docType + data, lalu kirim ke
	// output. docType: "TIKET" | "SEP" | "REGISTRASI" | "TEST".
	// data: struct sesuai docType (mis. *domain.Ticket untuk TIKET).
	// Insert ke print_history TIDAK dilakukan di sini — caller
	// service layer yang handle (supaya audit trail eksplisit).
	Print(ctx context.Context, docType string, data any) error

	// IsAvailable false saat printer belum siap (kertas habis,
	// disconnected, mock di-toggle off, dll). Status panel admin
	// pakai ini untuk indicator.
	IsAvailable() bool

	// Reprint baca bytes dari print_history.id, render ulang ke
	// printer, lalu increment reprint_count.
	Reprint(ctx context.Context, printHistoryID int64) error
}
