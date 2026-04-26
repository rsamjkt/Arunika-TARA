//go:build windows

package config

// keyFromPlatform di Windows: TODO P-052 implement DPAPI via
// crypt32.dll CryptProtectData/CryptUnprotectData. Untuk sekarang
// return ErrMasterKeyMissing supaya caller wajib set env var via
// Service Properties → Environment.
//
// Future Windows DPAPI plan:
//   1. Generate 32-byte random key
//   2. ProtectData(key, machine-bound flag) → ciphertext
//   3. Simpan ciphertext di registry HKLM\Software\APM\MasterKey
//   4. Load: read registry → UnprotectData → key
func keyFromPlatform() ([]byte, error) {
	return nil, ErrMasterKeyMissing
}
