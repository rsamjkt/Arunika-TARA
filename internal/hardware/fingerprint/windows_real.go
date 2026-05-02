//go:build windows

// Real implementation untuk verifikasi sidik jari via After.exe
// (Aplikasi Sidik Jari BPJS Kesehatan v2.1.0, WPF + SignalR).
//
// Flow:
//  1. Spawn After.exe (CREATE_NO_WINDOW — UI muncul dari After.exe sendiri).
//  2. Tunggu window "Aplikasi Registrasi Sidik Jari" muncul (max 15 detik).
//  3. Login otomatis via UIAutomation kalau belum login.
//  4. Inject noPeserta ke field noKartu (AutomationId=ar).
//  5. Return synthetic token "AFTEREXE_TRIGGERED_<timestamp>".
//     After.exe sendiri yang submit scan ke BPJS — APM trust token ini
//     sebagai sinyal "biometrik sudah di-trigger", sama dengan Frista.

package fingerprint

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/arunika/apm-go/internal/config"
)

type WindowsHeadlessVerifier struct {
	cfg    config.FingerprintConfig
	logger *slog.Logger

	mu  sync.Mutex
	cmd *exec.Cmd
}

var _ FingerprintVerifier = (*WindowsHeadlessVerifier)(nil)

func NewWindowsHeadless(cfg config.FingerprintConfig) FingerprintVerifier {
	return &WindowsHeadlessVerifier{
		cfg:    cfg,
		logger: slog.Default(),
	}
}

func (w *WindowsHeadlessVerifier) SetLogger(l *slog.Logger) {
	if l != nil {
		w.logger = l
	}
}

// IsAvailable cek apakah exe_path dikonfigurasi.
func (w *WindowsHeadlessVerifier) IsAvailable() bool {
	return w.cfg.ExePath != ""
}

// Verify spawn After.exe (kalau belum jalan), inject login + noPeserta
// via UIAutomation, return synthetic token.
func (w *WindowsHeadlessVerifier) Verify(ctx context.Context, noPeserta string) (FPResult, error) {
	if w.cfg.ExePath == "" {
		return FPResult{}, errors.New("config.fingerprint.exe_path kosong")
	}

	if err := w.ensureRunning(ctx); err != nil {
		return FPResult{}, fmt.Errorf("after.exe ensureRunning: %w", err)
	}

	// Inject login + noKartu via UIAutomation PowerShell
	if err := injectAfterUI(w.cfg.UsernameEnc, w.cfg.PasswordEnc, noPeserta); err != nil {
		w.logger.Warn("fingerprint: UIAutomation inject gagal", "err", err.Error())
		return FPResult{}, fmt.Errorf("inject noKartu After.exe: %w", err)
	}

	w.logger.Info("fingerprint: noPeserta injected ke After.exe",
		"no_peserta_masked", maskNoPeserta(noPeserta))

	token := fmt.Sprintf("AFTEREXE_TRIGGERED_%d", time.Now().Unix())
	return FPResult{
		Success:   true,
		Token:     token,
		Timestamp: time.Now(),
	}, nil
}

// ensureRunning spawn After.exe kalau belum berjalan. Idempotent.
func (w *WindowsHeadlessVerifier) ensureRunning(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cmd != nil && w.cmd.ProcessState == nil {
		return nil // sudah berjalan
	}

	cmd := exec.Command(w.cfg.ExePath)
	// CREATE_NO_WINDOW tidak cocok untuk After.exe (WPF butuh window untuk
	// UIAutomation). Pakai 0 supaya window muncul normal.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0,
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawn after.exe %q: %w", w.cfg.ExePath, err)
	}
	w.cmd = cmd
	w.logger.Info("fingerprint: after.exe spawned",
		"pid", cmd.Process.Pid, "exe", w.cfg.ExePath)

	// Startup delay sebelum UIAutomation mencari window
	startupDelay := time.Duration(w.cfg.StartupDelaySec) * time.Second
	if startupDelay <= 0 {
		startupDelay = 5 * time.Second
	}
	select {
	case <-ctx.Done():
		_ = w.killProcess()
		return ctx.Err()
	case <-time.After(startupDelay):
	}
	return nil
}

// Stop graceful — kill After.exe kalau masih hidup.
func (w *WindowsHeadlessVerifier) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.killProcess()
}

func (w *WindowsHeadlessVerifier) killProcess() error {
	if w.cmd == nil || w.cmd.Process == nil {
		return nil
	}
	if err := w.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("kill after.exe: %w", err)
	}
	_ = w.cmd.Wait()
	w.cmd = nil
	return nil
}

func maskNoPeserta(s string) string {
	if len(s) < 8 {
		return "***"
	}
	out := make([]byte, len(s))
	for i := range out {
		if i >= len(s)-4 {
			out[i] = s[i]
		} else {
			out[i] = '*'
		}
	}
	return string(out)
}
