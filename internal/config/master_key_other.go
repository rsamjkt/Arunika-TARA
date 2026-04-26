//go:build !darwin && !windows

package config

// keyFromPlatform di Linux/dll: env var only. Operator wajib set
// APM_MASTER_KEY di systemd service file atau ~/.profile.
func keyFromPlatform() ([]byte, error) {
	return nil, ErrMasterKeyMissing
}
