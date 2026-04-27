//go:build windows

// Real Windows implementation: spawn frista.exe (CREATE_NO_WINDOW),
// inject login via Win32 UI Automation (mirror DlgRegistrasiSEPPertama.java
// pattern), lalu polling Windows clipboard untuk capture card data
// yang Frista letakkan ke clipboard setiap kartu di-tap.
//
// Format clipboard yang diharapkan dari Frista (JSON):
//
//	{"nik":"3271...","nama":"BUDI ...","tgl_lahir":"1959-12-03",
//	 "alamat":"JL JAMBU...","no_kartu":"0001..."}
//
// Atau (legacy/varian RS):
//
//	NIK#NAMA#TGL_LAHIR#ALAMAT#NO_KARTU
//
// Auto-detect format per parse attempt.

package frista

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/arunika/apm-go/internal/config"
)

// WindowsReader adalah real implementation Frista card reader untuk
// Windows production. Lifecycle:
//
//	r := NewWindowsReader(cfg)
//	r.Start(ctx)         // spawn + login + start clipboard poller
//	for c := range r.CardRead() { ... }
//	r.Stop()             // kill process + stop poller
//
// Thread-safe.
type WindowsReader struct {
	cfg    config.FristaConfig
	logger *slog.Logger

	mu        sync.Mutex
	cmd       *exec.Cmd
	available bool
	cancel    context.CancelFunc

	ch chan CardData
}

var _ CardReader = (*WindowsReader)(nil)

// NewWindowsReader pada Windows = real implementation. provider.go
// memanggil ini, dan signature CardReader interface di-satisfy oleh
// *WindowsReader. Pada non-Windows, ada di windows.go (return mock).
func NewWindowsReader(cfg config.FristaConfig) CardReader {
	return &WindowsReader{
		cfg:    cfg,
		logger: slog.Default(),
		ch:     make(chan CardData, 4),
	}
}

// SetLogger menukar logger (untuk PHIMaskingHandler).
func (r *WindowsReader) SetLogger(l *slog.Logger) {
	if l != nil {
		r.logger = l
	}
}

func (r *WindowsReader) IsAvailable() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.available && r.cmd != nil && r.cmd.ProcessState == nil
}

func (r *WindowsReader) CardRead() <-chan CardData {
	return r.ch
}

// Start spawn frista.exe + auto-login + start clipboard polling goroutine.
func (r *WindowsReader) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.cmd != nil && r.cmd.ProcessState == nil {
		r.mu.Unlock()
		return nil // sudah jalan
	}
	if r.cfg.ExePath == "" {
		r.mu.Unlock()
		return errors.New("config.frista.exe_path kosong")
	}

	cmd := exec.Command(r.cfg.ExePath)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	if err := cmd.Start(); err != nil {
		r.mu.Unlock()
		return fmt.Errorf("spawn frista.exe %q: %w", r.cfg.ExePath, err)
	}
	r.cmd = cmd
	r.available = true
	r.logger.Info("frista: process spawned",
		"pid", cmd.Process.Pid, "exe", r.cfg.ExePath)

	pollCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.mu.Unlock()

	// Wait + inject login
	startupDelay := time.Duration(r.cfg.StartupDelaySec) * time.Second
	if startupDelay <= 0 {
		startupDelay = 5 * time.Second // Frista lebih lambat dari After.exe
	}
	time.Sleep(startupDelay)

	if err := injectFristaLogin(
		r.cfg.UsernameEnc, r.cfg.PasswordEnc,
		r.cfg.WindowClassLogin, r.cfg.WindowClassEdit, r.cfg.WindowClassButton,
	); err != nil {
		r.logger.Warn("frista: gagal inject login (operator mungkin perlu manual)",
			"err", err.Error())
		// Bukan fatal — operator bisa login manual di pop-up Frista
	} else {
		r.logger.Info("frista: auto-login sukses")
	}

	// Start clipboard poller
	go r.pollClipboard(pollCtx)
	return nil
}

// Stop graceful shutdown.
func (r *WindowsReader) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	if r.cmd != nil && r.cmd.Process != nil {
		_ = r.cmd.Process.Kill()
		_ = r.cmd.Wait()
		r.cmd = nil
	}
	r.available = false
	// JANGAN close r.ch — caller (Wails app) mungkin masih iterate.
	// GC akan handle saat WindowsReader di-collect.
	return nil
}

// pollClipboard periodic check Windows clipboard. Kalau content berubah
// dan parse-able sebagai card data, emit ke r.ch.
func (r *WindowsReader) pollClipboard(ctx context.Context) {
	interval := time.Duration(r.cfg.PollIntervalMs) * time.Millisecond
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	t := time.NewTicker(interval)
	defer t.Stop()

	var lastSeen string
	for {
		select {
		case <-ctx.Done():
			r.logger.Info("frista: clipboard poller stopped")
			return
		case <-t.C:
		}

		text, err := readClipboardText()
		if err != nil {
			// Clipboard sering busy (race dengan apps lain) — log debug saja.
			continue
		}
		if text == "" || text == lastSeen {
			continue
		}
		// Coba parse sebagai card data
		card, ok := parseCardClipboard(text)
		if !ok {
			// Bukan format Frista — biarkan untuk sekarang. lastSeen
			// tidak di-update supaya kalau kemudian jadi format yg valid
			// (mis. user copy text lain dulu) tetap bisa di-detect.
			continue
		}
		lastSeen = text
		card.Timestamp = time.Now()
		r.logger.Info("frista: card read terdeteksi",
			"nik_masked", maskCardField(card.NIK),
			"kartu_masked", maskCardField(card.NoKartu))
		select {
		case r.ch <- card:
		default:
			// Channel penuh — caller belum consume. Drop event lama.
			r.logger.Warn("frista: CardRead channel penuh, drop event")
		}
	}
}

// parseCardClipboard coba dua format yang Frista pakai:
//
//  1. JSON: {"nik":"...","nama":"...","tgl_lahir":"...","alamat":"...","no_kartu":"..."}
//  2. Pipe-delimited: NIK#NAMA#TGL_LAHIR#ALAMAT#NO_KARTU
//
// Return ok=false kalau tidak match keduanya.
func parseCardClipboard(text string) (CardData, bool) {
	text = strings.TrimSpace(text)
	if text == "" {
		return CardData{}, false
	}

	// Coba JSON
	if strings.HasPrefix(text, "{") {
		var raw struct {
			NIK      string `json:"nik"`
			Nama     string `json:"nama"`
			TglLahir string `json:"tgl_lahir"`
			Alamat   string `json:"alamat"`
			NoKartu  string `json:"no_kartu"`
		}
		if err := json.Unmarshal([]byte(text), &raw); err == nil && raw.NIK != "" {
			return CardData{
				NIK: raw.NIK, Nama: raw.Nama, TglLahir: raw.TglLahir,
				Alamat: raw.Alamat, NoKartu: raw.NoKartu,
			}, true
		}
	}

	// Coba pipe-delimited "#"
	parts := strings.Split(text, "#")
	if len(parts) >= 2 && len(parts[0]) == 16 && allDigits(parts[0]) {
		c := CardData{NIK: parts[0]}
		if len(parts) > 1 {
			c.Nama = parts[1]
		}
		if len(parts) > 2 {
			c.TglLahir = parts[2]
		}
		if len(parts) > 3 {
			c.Alamat = parts[3]
		}
		if len(parts) > 4 {
			c.NoKartu = parts[4]
		}
		return c, true
	}

	// Bisa juga single-line NIK (16 digit) — tapi kita butuh nama minimal.
	// Skip — tidak cukup info.
	return CardData{}, false
}

func allDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// maskCardField mengembalikan field yang di-mask kecuali 4 char terakhir.
// Disebut maskCardField (bukan maskID) untuk hindari clash dengan helper
// yang sama di http_server.go (yang dipakai mock).
func maskCardField(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return strings.Repeat("*", len(s)-4) + s[len(s)-4:]
}
