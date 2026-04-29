// Package updater handle auto-update APM dari GitHub Releases.
//
// Flow:
//
//  1. CheckLatest() — GET /repos/{repo}/releases/latest dengan PAT
//     (repo private), parse tag_name, compare dengan current version.
//  2. Download() — fetch asset matching cfg.AssetPattern ke temp file.
//  3. Apply() — backup current .exe ke ./backups/, atomic replace pakai
//     go-update, return path baru.
//  4. Restart() — spawn detached process baru lalu os.Exit(0). Kiosk
//     auto-restart dengan binary baru.
//
// Rollback: backup di-keep selama 7 hari (cleanup di startup berikutnya).
// Kalau update broken, admin manual swap via AdminScreen → "Rollback".
package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/inconshreveable/go-update"
)

// UpdateInfo adalah hasil cek release terbaru.
type UpdateInfo struct {
	Available      bool      // true kalau versi GitHub > current
	CurrentVersion string    // mis. "v2.1.2-mahatma"
	LatestVersion  string    // mis. "v2.2.0-mahatma"
	ReleaseNotes   string    // body markdown dari GitHub release
	AssetURL       string    // download URL asset (private — butuh token saat fetch)
	AssetName      string    // mis. "apm-windows-amd64.exe"
	AssetSize      int64     // bytes
	PublishedAt    time.Time // waktu release di-publish
}

// Updater — client untuk auto-update.
type Updater struct {
	repo           string
	token          string
	assetPattern   string
	currentVersion string

	httpClient *http.Client
}

// New membangun Updater dari config.
func New(repo, token, assetPattern, currentVersion string) *Updater {
	return &Updater{
		repo:           repo,
		token:          token,
		assetPattern:   assetPattern,
		currentVersion: currentVersion,
		httpClient:     &http.Client{Timeout: 30 * time.Second},
	}
}

// ============================================================
// CheckLatest — GitHub API call
// ============================================================

type ghRelease struct {
	TagName     string    `json:"tag_name"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name string `json:"name"`
	URL  string `json:"url"`  // API URL (butuh Authorization: token)
	Size int64  `json:"size"`
}

// CheckLatest hit /repos/{repo}/releases/latest.
//
// Return UpdateInfo dengan Available=false kalau tag GitHub == current
// version, atau Available=true + asset URL kalau ada update.
func (u *Updater) CheckLatest(ctx context.Context) (*UpdateInfo, error) {
	if u.repo == "" {
		return nil, fmt.Errorf("updater: repo kosong di config")
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", u.repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if u.token != "" {
		req.Header.Set("Authorization", "Bearer "+u.token)
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github API call: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("repo %s tidak ditemukan atau token tidak punya akses", u.repo)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("github token invalid / expired (401)")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("github API status %d: %s", resp.StatusCode, string(body))
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}

	info := &UpdateInfo{
		CurrentVersion: u.currentVersion,
		LatestVersion:  rel.TagName,
		ReleaseNotes:   rel.Body,
		PublishedAt:    rel.PublishedAt,
	}

	// Find asset yang match pattern
	for _, asset := range rel.Assets {
		if matchAsset(asset.Name, u.assetPattern) {
			info.AssetURL = asset.URL
			info.AssetName = asset.Name
			info.AssetSize = asset.Size
			break
		}
	}

	// Compare versions — kalau LatestVersion > CurrentVersion → Available
	info.Available = isNewer(rel.TagName, u.currentVersion) && info.AssetURL != ""
	return info, nil
}

// matchAsset — simple glob: kalau pattern punya "*" treat as wildcard,
// otherwise exact match (case-insensitive).
func matchAsset(name, pattern string) bool {
	if pattern == "" {
		return false
	}
	n := strings.ToLower(name)
	p := strings.ToLower(pattern)
	if strings.Contains(p, "*") {
		// Simple wildcard: split by * and check substring sequence
		parts := strings.Split(p, "*")
		idx := 0
		for _, part := range parts {
			if part == "" {
				continue
			}
			pos := strings.Index(n[idx:], part)
			if pos < 0 {
				return false
			}
			idx += pos + len(part)
		}
		return true
	}
	return n == p
}

// isNewer compare two semver-ish version strings.
// Format: vX.Y.Z[-suffix]. Suffix di-ignore untuk compare.
//
// Return true kalau latest > current.
func isNewer(latest, current string) bool {
	lv := parseVersion(latest)
	cv := parseVersion(current)
	if lv == nil || cv == nil {
		// Kalau parse gagal, treat as not newer (safe default)
		return false
	}
	for i := 0; i < 3; i++ {
		if lv[i] != cv[i] {
			return lv[i] > cv[i]
		}
	}
	return false
}

// parseVersion — strip "v" prefix + suffix setelah "-", split by ".".
// "v2.1.2-mahatma" → [2, 1, 2]
func parseVersion(s string) []int {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if dash := strings.Index(s, "-"); dash >= 0 {
		s = s[:dash]
	}
	parts := strings.Split(s, ".")
	if len(parts) < 3 {
		return nil
	}
	out := make([]int, 3)
	for i := 0; i < 3; i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return nil
		}
		out[i] = n
	}
	return out
}

// ============================================================
// Download + Apply
// ============================================================

// Download fetch asset dari GitHub API URL ke temp file.
// Return path file lokal.
func (u *Updater) Download(ctx context.Context, info *UpdateInfo, progress func(downloaded, total int64)) (string, error) {
	if info == nil || info.AssetURL == "" {
		return "", fmt.Errorf("updater: asset URL kosong")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.AssetURL, nil)
	if err != nil {
		return "", fmt.Errorf("build download request: %w", err)
	}
	req.Header.Set("Accept", "application/octet-stream")
	if u.token != "" {
		req.Header.Set("Authorization", "Bearer "+u.token)
	}

	// Longer timeout untuk download (file bisa puluhan MB)
	dlClient := &http.Client{Timeout: 5 * time.Minute}
	resp, err := dlClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download asset: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download status %d", resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "apm-update-*"+filepath.Ext(info.AssetName))
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Stream copy dengan progress callback
	written, err := copyWithProgress(tmpFile, resp.Body, info.AssetSize, progress)
	tmpFile.Close()
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("write asset: %w", err)
	}
	if info.AssetSize > 0 && written != info.AssetSize {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("download size mismatch: got %d, expected %d", written, info.AssetSize)
	}
	return tmpPath, nil
}

// copyWithProgress — io.Copy dengan periodic callback.
func copyWithProgress(dst io.Writer, src io.Reader, total int64, progress func(downloaded, total int64)) (int64, error) {
	var written int64
	buf := make([]byte, 32*1024)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			nw, werr := dst.Write(buf[:n])
			if werr != nil {
				return written, werr
			}
			written += int64(nw)
			if progress != nil {
				progress(written, total)
			}
		}
		if err != nil {
			if err == io.EOF {
				return written, nil
			}
			return written, err
		}
	}
}

// Apply ganti current executable dengan file di tmpPath, backup yang
// lama ke ./backups/. Pakai go-update yang atomic (rename ke .old +
// move temp ke target — Windows handle file lock).
//
// Return: backup path (untuk rollback) + error.
func (u *Updater) Apply(tmpPath string) (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve exe path: %w", err)
	}

	// Buat backup dulu
	backupDir := filepath.Join(filepath.Dir(exePath), "backups")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return "", fmt.Errorf("buat backup dir: %w", err)
	}
	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("apm-%s-%s%s", u.currentVersion, timestamp, filepath.Ext(exePath))
	backupPath := filepath.Join(backupDir, backupName)

	if err := copyFile(exePath, backupPath); err != nil {
		return "", fmt.Errorf("copy backup: %w", err)
	}

	// Apply atomic update
	f, err := os.Open(tmpPath)
	if err != nil {
		return backupPath, fmt.Errorf("open temp file: %w", err)
	}
	defer f.Close()

	opts := update.Options{}
	if err := update.Apply(f, opts); err != nil {
		// Rollback otomatis dari go-update kalau OldFile masih ada
		return backupPath, fmt.Errorf("apply update: %w", err)
	}

	// Cleanup temp
	_ = os.Remove(tmpPath)

	return backupPath, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

// Restart spawn binary baru sebagai detached process lalu return —
// caller harus os.Exit(0) setelah panggil ini.
//
// Windows: spawn pakai cmd.exe /C start untuk detach.
// Mac/Linux: spawn dengan SysProcAttr Setsid.
func Restart() error {
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve exe path: %w", err)
	}
	args := os.Args[1:]

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// "start" trick supaya parent bisa exit tanpa kill child
		quotedArgs := make([]string, 0, len(args)+2)
		quotedArgs = append(quotedArgs, "/C", "start", "", exePath)
		quotedArgs = append(quotedArgs, args...)
		cmd = exec.Command("cmd.exe", quotedArgs...)
	} else {
		cmd = exec.Command(exePath, args...)
	}
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawn baru: %w", err)
	}
	return nil
}

// ============================================================
// Backup management
// ============================================================

// CleanupOldBackups — hapus backup yang lebih dari maxAgeDays.
// Dipanggil di startup supaya tidak membengkak.
func CleanupOldBackups(exePath string, maxAgeDays int) error {
	if maxAgeDays <= 0 {
		return nil
	}
	backupDir := filepath.Join(filepath.Dir(exePath), "backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cutoff := time.Now().Add(-time.Duration(maxAgeDays) * 24 * time.Hour)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(backupDir, entry.Name()))
		}
	}
	return nil
}
