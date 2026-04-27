//go:build windows

// File ini hanya di-compile saat GOOS=windows.
// Real implementation untuk verifikasi sidik jari headless via After.exe
// (Aplikasi Sidik Jari BPJS Kesehatan) + REST polling.

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

	"github.com/go-resty/resty/v2"

	"github.com/arunika/apm-go/internal/config"
)

// WindowsHeadlessVerifier menjalankan After.exe sebagai background process
// (CREATE_NO_WINDOW), inject login lewat user32.dll UI Automation, lalu
// pakai REST API lokal After.exe untuk start scan + poll status.
//
// Lifecycle:
//
//	v := NewWindowsHeadless(cfg)
//	res, err := v.Verify(ctx, noPeserta)   // ensureRunning di-call internal
//	v.Stop()                                // graceful shutdown saat app exit
//
// Single-flight: hanya 1 Verify boleh berjalan pada satu waktu (After.exe
// sendiri tidak handle concurrent scan dengan baik). Caller serialize
// kalau perlu — atau wrap dengan mutex di service layer.
type WindowsHeadlessVerifier struct {
	cfg    config.FingerprintConfig
	client *resty.Client
	logger *slog.Logger

	mu  sync.Mutex
	cmd *exec.Cmd
}

// NewWindowsHeadless membangun verifier nyata. After.exe BELUM di-spawn
// sampai Verify() dipanggil pertama kali (lazy init untuk hemat resource).
func NewWindowsHeadless(cfg config.FingerprintConfig) FingerprintVerifier {
	timeout := time.Duration(cfg.ScanTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &WindowsHeadlessVerifier{
		cfg: cfg,
		client: resty.New().
			SetBaseURL(cfg.RestURL).
			SetTimeout(5 * time.Second).
			SetHeader("Content-Type", "application/json").
			SetHeader("User-Agent", "APM-TARA/1.0"),
		logger: slog.Default(),
	}
}

// IsAvailable cek apakah process After.exe masih hidup.
func (w *WindowsHeadlessVerifier) IsAvailable() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cmd == nil || w.cmd.Process == nil {
		return false
	}
	// Process.Signal(0) di Windows tidak begitu reliable — pakai
	// ProcessState check (nil = masih berjalan).
	return w.cmd.ProcessState == nil
}

// Verify menjalankan satu round verifikasi sidik jari.
//
//  1. ensureRunning() — spawn After.exe + auto-login kalau belum
//  2. POST /api/fingerprint dengan credential + noPeserta
//  3. Poll GET /api/fingerprint/status setiap PollIntervalMs
//     (default 500ms) sampai sukses, gagal, atau ScanTimeoutSec
//  4. Return FPResult dengan token kalau sukses
func (w *WindowsHeadlessVerifier) Verify(ctx context.Context, noPeserta string) (FPResult, error) {
	if err := w.ensureRunning(); err != nil {
		return FPResult{}, fmt.Errorf("after.exe ensureRunning: %w", err)
	}

	// Step 1: Start scan request
	startReq := map[string]any{
		"userId":       w.cfg.UsernameEnc, // sudah di-decrypt di config layer
		"userPassword": w.cfg.PasswordEnc,
		"noPeserta":    noPeserta,
	}
	resp, err := w.client.R().
		SetContext(ctx).
		SetBody(startReq).
		Post("/api/fingerprint")
	if err != nil {
		return FPResult{}, fmt.Errorf("POST /api/fingerprint: %w", err)
	}
	if resp.IsError() {
		return FPResult{}, fmt.Errorf("POST /api/fingerprint status=%d body=%s",
			resp.StatusCode(), resp.String())
	}

	// Step 2: Poll status
	pollInterval := time.Duration(w.cfg.PollIntervalMs) * time.Millisecond
	if pollInterval <= 0 {
		pollInterval = 500 * time.Millisecond
	}
	scanTimeout := time.Duration(w.cfg.ScanTimeoutSec) * time.Second
	if scanTimeout <= 0 {
		scanTimeout = 30 * time.Second
	}
	deadline := time.Now().Add(scanTimeout)

	type statusResp struct {
		Status  string `json:"status"`  // PENDING | SUCCESS | FAILED
		Token   string `json:"token"`
		Message string `json:"message"`
	}

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return FPResult{}, ctx.Err()
		case <-time.After(pollInterval):
		}

		var st statusResp
		statusResp, err := w.client.R().
			SetContext(ctx).
			SetResult(&st).
			Get("/api/fingerprint/status")
		if err != nil {
			w.logger.Warn("fingerprint: poll status error, retry",
				"err", err.Error())
			continue
		}
		if statusResp.IsError() {
			continue
		}

		switch st.Status {
		case "SUCCESS":
			return FPResult{
				Success:   true,
				Token:     st.Token,
				Timestamp: time.Now(),
			}, nil
		case "FAILED":
			return FPResult{}, fmt.Errorf("fingerprint failed: %s", st.Message)
		}
		// PENDING / unknown → continue polling
	}

	return FPResult{}, errors.New("fingerprint scan timeout")
}

// ensureRunning spawn After.exe kalau belum, lalu inject login.
// Idempotent — safe dipanggil multiple kali.
func (w *WindowsHeadlessVerifier) ensureRunning() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cmd != nil && w.cmd.ProcessState == nil {
		return nil // sudah berjalan
	}

	if w.cfg.ExePath == "" {
		return errors.New("config.fingerprint.exe_path kosong")
	}

	// Spawn dengan CREATE_NO_WINDOW (0x08000000) supaya After.exe
	// tidak muncul UI window di kiosk.
	cmd := exec.Command(w.cfg.ExePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawn after.exe %q: %w", w.cfg.ExePath, err)
	}
	w.cmd = cmd
	w.logger.Info("fingerprint: after.exe spawned",
		"pid", cmd.Process.Pid, "exe", w.cfg.ExePath)

	// Wait beberapa detik untuk app load sebelum inject login.
	startupDelay := time.Duration(w.cfg.StartupDelaySec) * time.Second
	if startupDelay <= 0 {
		startupDelay = 3 * time.Second
	}
	time.Sleep(startupDelay)

	// Inject login lewat user32.dll UI Automation. Class names dari
	// config — fallback ke Delphi VCL default kalau kosong.
	if err := injectAfterLogin(
		w.cfg.UsernameEnc, w.cfg.PasswordEnc,
		w.cfg.WindowClassLogin, w.cfg.WindowClassEdit, w.cfg.WindowClassButton,
	); err != nil {
		w.logger.Warn("fingerprint: gagal inject login", "err", err.Error())
		// Tidak fatal — operator mungkin sudah login manual sebelumnya.
		// Verify call berikutnya akan fail kalau login memang belum done.
	}
	return nil
}

// Stop graceful — kill After.exe process kalau masih hidup.
// Dipanggil saat APM shutdown.
func (w *WindowsHeadlessVerifier) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()
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

