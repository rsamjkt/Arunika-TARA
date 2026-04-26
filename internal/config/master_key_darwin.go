//go:build darwin

package config

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
)

// keychainService adalah label generic password di Keychain Mac.
// Generate dengan:
//
//	security add-generic-password -s apm-go -a $(whoami) -w <base64-key>
const keychainService = "apm-go"

// keyFromPlatform di macOS coba ambil dari Keychain via `security` CLI.
// Best-effort — kalau key tidak ada, return ErrMasterKeyMissing supaya
// caller fallback ke env var.
func keyFromPlatform() ([]byte, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-s", keychainService, "-w")
	out, err := cmd.Output()
	if err != nil {
		// security exit non-zero saat item tidak ditemukan
		return nil, ErrMasterKeyMissing
	}

	encoded := strings.TrimSpace(string(out))
	if encoded == "" {
		return nil, ErrMasterKeyMissing
	}
	k, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("keychain key bukan base64: %w", err)
	}
	if len(k) != MasterKeySize {
		return nil, fmt.Errorf("keychain key panjang %d, harus %d",
			len(k), MasterKeySize)
	}
	return k, nil
}
