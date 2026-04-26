package config

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
)

// MasterKeyEnvVar adalah env var yang dibaca PERTAMA — base64-encoded
// 32 bytes (AES-256). Production deployment set lewat:
//   - Linux/Mac: ~/.profile, systemd service file, launchd plist
//   - Windows: Service Properties → Environment Variables, atau
//     Registry-stored env (Computer\HKEY_LOCAL_MACHINE\SYSTEM\
//     CurrentControlSet\Control\Session Manager\Environment)
const MasterKeyEnvVar = "APM_MASTER_KEY"

// MasterKeySize adalah ukuran AES-256 key dalam bytes.
const MasterKeySize = 32

// ErrMasterKeyMissing dikembalikan saat tidak ada master key tersedia
// dari sumber manapun.
var ErrMasterKeyMissing = errors.New("master key tidak tersedia (set APM_MASTER_KEY env var)")

// ErrMasterKeyInvalid dikembalikan saat key ada tapi format salah
// (bukan base64 valid atau ukuran bukan 32 bytes).
var ErrMasterKeyInvalid = errors.New("master key format invalid (harus base64 32 bytes)")

// GetMasterKey resolve master key dari sumber prioritas:
//
//  1. Env var APM_MASTER_KEY (base64 32 bytes)
//  2. Platform-specific store (macOS Keychain — lihat master_key_darwin.go)
//
// Production wajib set env var via service config. Keychain hook
// hanya untuk dev convenience di mesin development.
func GetMasterKey() ([]byte, error) {
	if k, ok := keyFromEnv(); ok {
		return k, nil
	}
	if k, err := keyFromPlatform(); err == nil {
		return k, nil
	} else if !errors.Is(err, ErrMasterKeyMissing) {
		return nil, fmt.Errorf("platform key store: %w", err)
	}
	return nil, ErrMasterKeyMissing
}

// keyFromEnv parse APM_MASTER_KEY env var.
func keyFromEnv() ([]byte, bool) {
	raw := os.Getenv(MasterKeyEnvVar)
	if raw == "" {
		return nil, false
	}
	k, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(k) != MasterKeySize {
		// Invalid env var — bukan miss, ini error setup
		return nil, false
	}
	return k, true
}

// GenerateMasterKey buat key acak 32 bytes + return sebagai base64.
// Caller pakai untuk first-time setup: simpan ke env/keychain.
func GenerateMasterKey() (string, error) {
	k := make([]byte, MasterKeySize)
	if _, err := rand.Read(k); err != nil {
		return "", fmt.Errorf("rand read: %w", err)
	}
	return base64.StdEncoding.EncodeToString(k), nil
}
