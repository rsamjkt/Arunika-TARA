package config

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"golang.org/x/term"
)

// ENCPrefix adalah penanda field yang sudah ter-enkripsi di config.toml.
const ENCPrefix = "ENC:"

// IsEncrypted check apakah string punya prefix ENC:
func IsEncrypted(s string) bool {
	return strings.HasPrefix(s, ENCPrefix)
}

// EncryptValue enkripsi plaintext dengan AES-256-GCM dan return
// "ENC:<base64(nonce+ciphertext)>". Master key wajib 32 bytes.
//
// Format:
//
//	plaintext (UTF-8)
//	→ AES-256-GCM seal dengan random nonce 12 bytes
//	→ output = nonce || ciphertext (ciphertext sudah include auth tag)
//	→ base64 encode
//	→ prefix "ENC:"
func EncryptValue(plaintext string, masterKey []byte) (string, error) {
	if len(masterKey) != MasterKeySize {
		return "", ErrMasterKeyInvalid
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "", fmt.Errorf("aes new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm new: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("random nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	// Concat nonce + ciphertext supaya self-contained
	combined := make([]byte, 0, len(nonce)+len(ciphertext))
	combined = append(combined, nonce...)
	combined = append(combined, ciphertext...)

	return ENCPrefix + base64.StdEncoding.EncodeToString(combined), nil
}

// DecryptValue parse "ENC:..." string, AES-256-GCM open, return plaintext.
// Kalau s TIDAK punya prefix ENC:, dianggap plaintext dan langsung return.
func DecryptValue(s string, masterKey []byte) (string, error) {
	if !IsEncrypted(s) {
		return s, nil
	}
	if len(masterKey) != MasterKeySize {
		return "", ErrMasterKeyInvalid
	}

	encoded := strings.TrimPrefix(s, ENCPrefix)
	combined, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "", fmt.Errorf("aes new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm new: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(combined) < nonceSize+1 {
		return "", errors.New("ciphertext terlalu pendek")
	}
	nonce, ciphertext := combined[:nonceSize], combined[nonceSize:]

	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("gcm open: %w", err)
	}
	return string(plain), nil
}

// ============================================================
// EncryptConfig — CLI flow yang prompt user + tulis ulang config.toml
// ============================================================

// EncryptConfig prompt user untuk credential sensitif, encrypt, dan
// tulis ulang config.toml dengan field-field yang sudah ter-enkripsi.
//
// Field yang di-encrypt:
//   - frista.username_enc
//   - frista.password_enc
//   - fingerprint.username_enc
//   - fingerprint.password_enc
//
// Field yang sudah punya prefix "ENC:" SKIP (tidak prompt) — supaya
// bisa run berulang tanpa over-encrypt.
//
// Master key resolved via GetMasterKey() — env var atau platform store.
func EncryptConfig(configPath string) error {
	masterKey, err := GetMasterKey()
	if err != nil {
		return fmt.Errorf("master key: %w", err)
	}

	// Baca config.toml sebagai map (preserve unknown sections + comments
	// sebatas TOML parser support — go-toml/v2 strips comments saat
	// marshal, jadi user yang punya komentar custom akan kehilangan.
	// Untuk first-time setup ini acceptable.
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("baca %s: %w", configPath, err)
	}

	var cfgMap map[string]any
	if err := toml.Unmarshal(raw, &cfgMap); err != nil {
		return fmt.Errorf("unmarshal %s: %w", configPath, err)
	}

	reader := bufio.NewReader(os.Stdin)

	if err := promptAndEncryptInto(cfgMap, "frista", "username_enc",
		"Frista username", reader, masterKey, false); err != nil {
		return err
	}
	if err := promptAndEncryptInto(cfgMap, "frista", "password_enc",
		"Frista password", reader, masterKey, true); err != nil {
		return err
	}
	if err := promptAndEncryptInto(cfgMap, "fingerprint", "username_enc",
		"Fingerprint BPJS username", reader, masterKey, false); err != nil {
		return err
	}
	if err := promptAndEncryptInto(cfgMap, "fingerprint", "password_enc",
		"Fingerprint BPJS password", reader, masterKey, true); err != nil {
		return err
	}

	// Tulis ulang
	out, err := toml.Marshal(cfgMap)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(configPath, out, 0o600); err != nil {
		return fmt.Errorf("tulis %s: %w", configPath, err)
	}

	fmt.Println("\n✅ Credential berhasil dienkripsi di " + configPath)
	return nil
}

// promptAndEncryptInto prompt user untuk satu field, encrypt, set ke map.
// Skip kalau sudah ada nilai dengan prefix ENC:.
func promptAndEncryptInto(
	cfgMap map[string]any,
	section, field, label string,
	reader *bufio.Reader,
	masterKey []byte,
	hidden bool,
) error {
	sec, ok := cfgMap[section].(map[string]any)
	if !ok {
		// Section belum ada — skip (operator harus add manually)
		fmt.Printf("⚠ Section [%s] tidak ada di config, skip.\n", section)
		return nil
	}
	current, _ := sec[field].(string)
	if IsEncrypted(current) {
		fmt.Printf("• %s sudah ter-enkripsi, skip.\n", label)
		return nil
	}

	val, err := promptInput(label, reader, hidden)
	if err != nil {
		return err
	}
	if val == "" {
		fmt.Printf("• %s dilewati (input kosong).\n", label)
		return nil
	}
	encrypted, err := EncryptValue(val, masterKey)
	if err != nil {
		return fmt.Errorf("encrypt %s: %w", label, err)
	}
	sec[field] = encrypted
	cfgMap[section] = sec
	fmt.Printf("✓ %s ter-enkripsi.\n", label)
	return nil
}

// promptInput baca dari stdin. hidden=true pakai term.ReadPassword
// (no echo). Cocok untuk password.
func promptInput(label string, reader *bufio.Reader, hidden bool) (string, error) {
	fmt.Printf("%s: ", label)
	if hidden {
		// Pastikan stdin adalah terminal — kalau tidak (mis. CI),
		// fallback baca line normal.
		fd := int(os.Stdin.Fd())
		if term.IsTerminal(fd) {
			b, err := term.ReadPassword(fd)
			fmt.Println()
			if err != nil {
				return "", fmt.Errorf("read password: %w", err)
			}
			return string(b), nil
		}
	}
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read input: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}
