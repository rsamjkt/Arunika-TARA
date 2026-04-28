// Package frista adalah abstraksi verifikasi WAJAH (face recognition)
// pasien BPJS via aplikasi Frista. Posisinya sejajar dengan paket
// `fingerprint` (After.exe untuk verifikasi sidik jari) — keduanya
// menghasilkan token biometrik yang dilampirkan ke payload SEP.
//
// Catatan historis: paket ini SEBELUMNYA berisi abstraksi card reader
// (Pasien tap KTP/kartu BPJS, channel CardRead). Itu salah konsep
// — Frista BUKAN reader kartu. Frista adalah aplikasi BPJS desktop
// untuk SIDIK WAJAH (mirror DlgRegistrasiSEPPertama.java:2213-2225,
// 3764: BukaFrista(NoKartu)). Pasien hadap kamera, Frista match ke
// face DB BPJS, hasil token biometrik dilampirkan ke SEP.
//
// Status implementasi:
//   - interface.go (file ini) — kontrak yang dipakai service layer + Wails app
//   - mock.go                  — implementasi development di Mac/Linux
//   - windows_stub.go          — placeholder NewWindowsHeadless di non-Windows
//   - windows_real.go          — TODO: real impl headless via frista.exe
//                                + Win32 UI Automation (mirror After.exe)
package frista

import (
	"context"
	"time"
)

// FaceVerifier adalah surface API yang dipakai SEP service. Mirror
// pattern fingerprint.FingerprintVerifier — keduanya call-based
// (bukan event-driven) supaya frontend bisa pilih biometrik mana
// yang akan dipanggil per kasus.
type FaceVerifier interface {
	// Verify memulai proses verifikasi sidik wajah untuk pasien dengan
	// noPeserta (no kartu BPJS). Blocking sampai sukses, gagal, atau
	// timeout (umumnya 30 detik — di-set lewat config.Frista.ScanTimeoutSec).
	//
	// ctx.Done() harus dihormati supaya pasien yang batal di tengah
	// proses tidak meninggalkan goroutine yang menggantung.
	Verify(ctx context.Context, noPeserta string) (FRResult, error)

	// IsAvailable mengembalikan false jika hardware atau dependency
	// (frista.exe di Windows, mock di Mac) tidak ready.
	// Service layer dapat skip biometrik & log warning bila false.
	IsAvailable() bool
}

// FRResult adalah hasil verifikasi sidik wajah yang sukses.
// Token dipakai sebagai field "finger" / biometrik proof di payload
// SEP request (BPJS treat fingerprint & face token sebagai biometric
// token di field yang sama).
type FRResult struct {
	Success   bool
	Token     string    // dipakai di SEP payload sebagai biometrik proof
	Timestamp time.Time
}
