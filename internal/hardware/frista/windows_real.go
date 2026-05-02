//go:build windows

// File ini hanya di-compile saat GOOS=windows.
//
// Frista (Aplikasi Sidik Wajah BPJS Kesehatan) — vendor pattern impl.
// Mirror BukaFrista() di KhanzaHMSAnjunganSEP_RSAMXIP/src/khanzahmsanjungan/
// DlgRegistrasiSEPPertama.java line 3764-3829.
//
// Flow:
//
//  1. Spawn frista.exe (visible window — Frista butuh kamera UI).
//  2. Wait startup_delay_sec (vendor 5500ms via Thread.sleep).
//  3. BringToFront via SetForegroundWindow.
//  4. Inject creds + noPeserta via clipboard + SendInput Ctrl+V/Tab:
//       - paste username  → Tab
//       - paste password  → Tab → Space (klik tombol login Frista)
//       - paste noPeserta (BPJS card number)
//  5. Return synthetic token "FRISTA_TRIGGERED_<timestamp>" — Frista
//     sendiri push verifikasi ke server BPJS, jadi APM tidak punya
//     token nyata. Service.maybeBiometrik akan trust externalToken
//     ini sebagai sinyal kesuksesan.

package frista

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

// WindowsHeadlessVerifier — nama legacy (tidak benar-benar headless lagi
// karena Frista perlu kamera visible). Disimpan supaya import path dari
// hardware/provider.go tidak break. Real implementation below.
type WindowsHeadlessVerifier struct {
	cfg    config.FristaConfig
	logger *slog.Logger

	mu  sync.Mutex
	cmd *exec.Cmd
}

var _ FaceVerifier = (*WindowsHeadlessVerifier)(nil)

// NewWindowsHeadless membangun verifier nyata. frista.exe BELUM di-spawn
// sampai Verify() dipanggil pertama kali (lazy init untuk hemat resource).
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

// IsAvailable cek apakah frista.exe path valid + ExePath non-kosong.
// Process status tidak di-cek (Frista bisa di-launch on-demand).
func (w *WindowsHeadlessVerifier) IsAvailable() bool {
	return w.cfg.ExePath != ""
}

// Verify — replikasi vendor BukaFrista pattern.
//
// Sukses-criteria: SendInput sequence completed tanpa error. Token
// kembalian adalah synthetic marker — caller (service.maybeBiometrik)
// akan trust ini sebagai sinyal "biometrik sudah di-trigger frontend"
// dan skip cekFinger BPJS server verification (eventual consistency).
func (w *WindowsHeadlessVerifier) Verify(ctx context.Context, noPeserta string) (FRResult, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.cfg.ExePath == "" {
		return FRResult{}, errors.New("config.frista.exe_path kosong")
	}

	// Step 1: spawn frista.exe (visible — kamera Frista butuh UI render).
	cmd := exec.Command(w.cfg.ExePath)
	// CREATE_NEW_CONSOLE supaya Frista tidak inherit stdin/out kiosk
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x00000010, // CREATE_NEW_CONSOLE
	}
	if err := cmd.Start(); err != nil {
		return FRResult{}, fmt.Errorf("spawn frista.exe %q: %w", w.cfg.ExePath, err)
	}
	w.cmd = cmd
	w.logger.Info("frista: process spawned",
		"pid", cmd.Process.Pid, "exe", w.cfg.ExePath)

	// Step 2: wait startup delay supaya UI Frista siap menerima input.
	startupDelay := time.Duration(w.cfg.StartupDelaySec) * time.Second
	if startupDelay <= 0 {
		startupDelay = 5 * time.Second // vendor 5500ms
	}
	select {
	case <-ctx.Done():
		_ = w.killProcess()
		return FRResult{}, ctx.Err()
	case <-time.After(startupDelay):
	}

	// Step 3: bring window to front. Retry sampai window muncul atau
	// timeout 15 detik (Frista kadang butuh waktu ekstra untuk render UI).
	var hwnd uintptr
	windowDeadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(windowDeadline) {
		hwnd = FindWindowByTitleSubstring("Frista")
		if hwnd != 0 {
			break
		}
		select {
		case <-ctx.Done():
			_ = w.killProcess()
			return FRResult{}, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	if hwnd != 0 {
		if err := BringToFront(hwnd); err != nil {
			w.logger.Warn("frista: bring to front gagal (lanjut, mungkin already focused)",
				"err", err.Error())
		}
		time.Sleep(300 * time.Millisecond)
	} else {
		w.logger.Warn("frista: window 'Frista' tidak ditemukan setelah 15 detik — paste mungkin gagal target")
	}

	// Step 4: inject username + password + noPeserta via clipboard.
	// Vendor sequence (line 3785-3821):
	//   - paste user → Ctrl+V → Tab
	//   - paste pass → Ctrl+V → Tab → Space (klik login)
	//   - wait 2000ms (login process)
	//   - mouse click center (focus field nokartu) — di sini kita skip,
	//     mengandalkan focus pindah otomatis ke field nokartu setelah login
	//   - paste nokartu → Ctrl+V

	if err := PasteText(w.cfg.UsernameEnc); err != nil {
		return FRResult{}, fmt.Errorf("paste username: %w", err)
	}
	time.Sleep(200 * time.Millisecond)
	if err := PressTab(); err != nil {
		return FRResult{}, fmt.Errorf("tab after username: %w", err)
	}
	time.Sleep(150 * time.Millisecond)

	if err := PasteText(w.cfg.PasswordEnc); err != nil {
		return FRResult{}, fmt.Errorf("paste password: %w", err)
	}
	time.Sleep(200 * time.Millisecond)
	if err := PressTab(); err != nil {
		return FRResult{}, fmt.Errorf("tab after password: %w", err)
	}
	time.Sleep(150 * time.Millisecond)
	// Vendor pakai Space, biasanya mapping ke tombol "Login" yang focused
	if err := PressSpace(); err != nil {
		w.logger.Warn("frista: press space gagal", "err", err.Error())
	}

	// Wait Frista login process — vendor 2000ms
	time.Sleep(2 * time.Second)

	// Setelah login, Frista bisa buka window baru (HWND lama invalid).
	// Re-find window terbaru sebelum click center.
	mainHwnd := FindWindowByTitleSubstring("Frista")
	if mainHwnd == 0 {
		mainHwnd = hwnd // fallback ke hwnd lama kalau tidak ketemu
	}
	// Vendor: mouse click center untuk fokus field noKartu setelah login.
	if mainHwnd != 0 {
		_ = BringToFront(mainHwnd)
		time.Sleep(200 * time.Millisecond)
		if err := ClickWindowCenter(mainHwnd); err != nil {
			w.logger.Warn("frista: click center gagal", "err", err.Error())
		}
		time.Sleep(300 * time.Millisecond)
	}

	if err := PasteText(noPeserta); err != nil {
		return FRResult{}, fmt.Errorf("paste noPeserta: %w", err)
	}
	w.logger.Info("frista: creds + noPeserta injected",
		"no_peserta_masked", maskNoPeserta(noPeserta))

	// Step 5: return synthetic token. Frista sendiri yang submit ke
	// server BPJS — APM tidak punya cara ambil token. Service layer
	// akan trust externalToken non-empty.
	token := fmt.Sprintf("FRISTA_TRIGGERED_%d", time.Now().Unix())
	return FRResult{
		Success:   true,
		Token:     token,
		Timestamp: time.Now(),
	}, nil
}

// Stop graceful — kill frista.exe kalau masih hidup. Dipanggil saat
// APM shutdown atau session reset.
func (w *WindowsHeadlessVerifier) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.killProcess()
}

func (w *WindowsHeadlessVerifier) killProcess() error {
	if w.cmd == nil || w.cmd.Process == nil {
		return nil
	}
	// Try graceful close first
	hwnd := FindWindowByTitleSubstring("Frista")
	if hwnd != 0 {
		_ = CloseWindow(hwnd)
		time.Sleep(500 * time.Millisecond)
	}
	// Force kill kalau masih running
	if w.cmd.ProcessState == nil {
		_ = w.cmd.Process.Kill()
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
