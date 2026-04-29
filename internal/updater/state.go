package updater

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// UpdateState — file last-update.json yang menyimpan info update terakhir.
// Dipakai untuk health check post-startup + rollback.
type UpdateState struct {
	PreviousVersion string    `json:"previous_version"`
	NewVersion      string    `json:"new_version"`
	BackupPath      string    `json:"backup_path"`
	AppliedAt       time.Time `json:"applied_at"`
	HealthChecked   bool      `json:"health_checked"`
}

func stateFilePath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exePath), "last-update.json"), nil
}

// SaveState write state.json ke folder eksekutable.
func SaveState(state *UpdateState) error {
	if state == nil {
		return fmt.Errorf("state nil")
	}
	path, err := stateFilePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// LoadState baca state.json. Return nil + nil error kalau file tidak ada.
func LoadState() (*UpdateState, error) {
	path, err := stateFilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var st UpdateState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("decode state: %w", err)
	}
	return &st, nil
}

// MarkHealthy update state.json untuk mark bahwa post-update health
// check sudah pass. Dipanggil setelah ping VClaim + Khanza sukses.
func MarkHealthy() error {
	st, err := LoadState()
	if err != nil || st == nil {
		return err
	}
	st.HealthChecked = true
	return SaveState(st)
}

// IsRecentUpdate return true kalau state.AppliedAt dalam window terakhir
// (default 10 menit). Dipakai startup untuk decide apakah perlu run
// health check + auto-rollback logic.
func (s *UpdateState) IsRecentUpdate(window time.Duration) bool {
	if s == nil || s.AppliedAt.IsZero() {
		return false
	}
	if window <= 0 {
		window = 10 * time.Minute
	}
	return time.Since(s.AppliedAt) < window
}

// Rollback swap current .exe dengan backup. Caller harus Restart()
// + os.Exit(0) setelah ini.
//
// Strategi: copy backup ke temp, lalu Apply (atomic replace).
// Backup file di-keep — admin bisa lihat history di backups/ folder.
func Rollback() error {
	st, err := LoadState()
	if err != nil {
		return fmt.Errorf("load state untuk rollback: %w", err)
	}
	if st == nil || st.BackupPath == "" {
		return fmt.Errorf("tidak ada state update sebelumnya — tidak bisa rollback")
	}
	if _, err := os.Stat(st.BackupPath); err != nil {
		return fmt.Errorf("backup file tidak ditemukan: %s", st.BackupPath)
	}

	// Copy backup ke temp dulu — Apply() destructive (delete tmpPath
	// setelah swap), jangan biarkan backup ke-hapus.
	tmp, err := os.CreateTemp("", "apm-rollback-*"+filepath.Ext(st.BackupPath))
	if err != nil {
		return fmt.Errorf("buat temp rollback: %w", err)
	}
	tmpPath := tmp.Name()
	tmp.Close()
	if err := copyFile(st.BackupPath, tmpPath); err != nil {
		return fmt.Errorf("copy backup ke temp: %w", err)
	}

	u := &Updater{currentVersion: st.NewVersion}
	if _, err := u.Apply(tmpPath); err != nil {
		return fmt.Errorf("rollback apply: %w", err)
	}

	// Hapus state file — current binary sekarang adalah versi PreviousVersion.
	path, _ := stateFilePath()
	_ = os.Remove(path)
	return nil
}
