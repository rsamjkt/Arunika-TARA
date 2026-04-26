// Package fingerprint adalah abstraksi verifikasi sidik jari pasien
// untuk syarat penerbitan SEP BPJS pasien dewasa non-IGD.
//
// Status implementasi:
//   - interface.go (file ini) — kontrak yang dipakai service layer
//   - mock.go                   — implementasi development di Mac/Linux
//   - windows.go (P-032)        — implementasi headless via After.exe
//                                 + Windows UI Automation
//
// Provider switching ditangani di internal/hardware/provider.go (P-030)
// berdasarkan runtime.GOOS.
package fingerprint

import (
	"context"
	"time"
)

// FingerprintVerifier adalah surface API yang dipakai SEP service.
type FingerprintVerifier interface {
	// Verify memulai proses verifikasi sidik jari untuk pasien dengan
	// noPeserta (no kartu BPJS). Blocking sampai sukses, gagal, atau
	// timeout (umumnya 30 detik — di-set lewat config.Fingerprint).
	//
	// ctx.Done() harus dihormati supaya pasien yang batal di tengah
	// proses tidak meninggalkan goroutine yang menggantung.
	Verify(ctx context.Context, noPeserta string) (FPResult, error)

	// IsAvailable mengembalikan false jika hardware atau dependency
	// (After.exe di Windows, mock server di Mac) tidak ready.
	// Service layer dapat skip biometrik & log warning bila false.
	IsAvailable() bool
}

// FPResult adalah hasil verifikasi sidik jari yang sukses.
// Token dipakai sebagai field "finger" di payload SEP request.
type FPResult struct {
	Success   bool
	Token     string    // dipakai di SEP payload
	Timestamp time.Time
}
