//go:build windows

// File ini hanya di-compile saat GOOS=windows.
//
// STATUS: STUB — belum ada real impl.
// Real implementation untuk verifikasi sidik wajah headless via frista.exe
// (Aplikasi Sidik Wajah BPJS Kesehatan) + REST polling akan menyusul
// di tiket P-031+ (sejajar dengan fingerprint/windows_real.go pattern).
//
// Skema target (TODO):
//
//   1. Spawn frista.exe (CREATE_NO_WINDOW) — mirror BukaFrista() di
//      KhanzaHMSAnjunganSEP_RSAMXIP/src/khanzahmsanjungan/
//      DlgRegistrasiSEPPertama.java line 3764.
//   2. Auto-login via Win32 UI Automation (FindWindowW + SendMessageW
//      WM_SETTEXT + BM_CLICK ke dialog login Frista). Class names di-set
//      lewat config.Frista.WindowClass{Login,Edit,Button} dengan default
//      Delphi VCL TfrmLogin/TEdit/TButton.
//   3. Frista expose REST endpoint lokal (mirip After.exe) untuk
//      start scan + poll status; alternatifnya pakai clipboard polling
//      kalau Frista tidak expose REST. Verify metode tergantung versi
//      Frista — perlu probe vendor lebih lanjut.
//   4. Token hasil scan dipakai sebagai biometrik proof di SEP payload
//      (BPJS treat fingerprint & face token sama — field "finger").

package frista

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/arunika/apm-go/internal/config"
)

// WindowsHeadlessVerifier adalah real implementation Frista face
// verifier untuk Windows production. SAAT INI MASIH STUB — Verify
// langsung return error supaya kalau dipanggil di production tanpa
// real impl, log jelas dan caller bisa fallback ke skip biometrik
// (sama kayak fingerprint behavior).
//
// Struct field & method signature sudah final supaya P-031+ tinggal
// isi body Verify dengan spawn frista.exe + REST polling.
type WindowsHeadlessVerifier struct {
	cfg    config.FristaConfig
	logger *slog.Logger

	mu sync.Mutex
}

var _ FaceVerifier = (*WindowsHeadlessVerifier)(nil)

// NewWindowsHeadless membangun verifier nyata. frista.exe BELUM di-spawn
// sampai Verify() dipanggil pertama kali (lazy init).
//
// CATATAN: implementation BELUM lengkap. IsAvailable() sengaja return
// false supaya service layer auto-skip biometrik tanpa fail. Hapus
// fallback false setelah real impl di-merge.
func NewWindowsHeadless(cfg config.FristaConfig) FaceVerifier {
	return &WindowsHeadlessVerifier{
		cfg:    cfg,
		logger: slog.Default(),
	}
}

// SetLogger menukar logger (untuk PHIMaskingHandler).
func (w *WindowsHeadlessVerifier) SetLogger(l *slog.Logger) {
	if l != nil {
		w.logger = l
	}
}

// IsAvailable — STUB: selalu return false sampai real impl tersedia.
// Service layer (sep.maybeBiometrik) akan log warning + skip biometrik
// kalau false, jadi pasien tetap bisa lanjut SEP — operator BPJS akan
// reject saat klaim kalau memang token wajib.
func (w *WindowsHeadlessVerifier) IsAvailable() bool {
	return false
}

// Verify — STUB: belum di-implement. Return error eksplisit supaya
// kalau caller bypass IsAvailable() check (jangan dilakukan), error
// message clear: "frista windows real impl belum di-implement (P-031)".
//
// Real impl: spawn frista.exe + auto-login + start scan + poll status,
// return token dari Frista REST response.
func (w *WindowsHeadlessVerifier) Verify(ctx context.Context, noPeserta string) (FRResult, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.logger.Warn("frista: Windows real impl belum di-implement (P-031)",
		"exe_path", w.cfg.ExePath)
	return FRResult{}, errors.New("frista windows real impl belum di-implement (P-031): " +
		"verifikasi sidik wajah hanya tersedia setelah real impl di-merge — sementara skip biometrik")
}

// Stop graceful — placeholder. Real impl akan kill frista.exe process
// kalau masih hidup (mirror fingerprint.WindowsHeadlessVerifier.Stop).
func (w *WindowsHeadlessVerifier) Stop() error {
	return nil
}
