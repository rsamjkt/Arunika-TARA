package config

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

// genKey buat random 32-byte key untuk test.
func genKey(t *testing.T) []byte {
	t.Helper()
	enc, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}
	k, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	return k
}

func TestEncryptValue_RoundTrip(t *testing.T) {
	key := genKey(t)
	plaintext := "frista_password_123!@#"

	encrypted, err := EncryptValue(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptValue: %v", err)
	}
	if !strings.HasPrefix(encrypted, ENCPrefix) {
		t.Errorf("output harus prefix ENC:, got %q", encrypted)
	}

	got, err := DecryptValue(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptValue: %v", err)
	}
	if got != plaintext {
		t.Errorf("round-trip mismatch: got %q want %q", got, plaintext)
	}
}

func TestEncryptValue_NonceUnik(t *testing.T) {
	key := genKey(t)
	plaintext := "same input"

	a, _ := EncryptValue(plaintext, key)
	b, _ := EncryptValue(plaintext, key)

	if a == b {
		t.Errorf("output identik untuk plaintext sama — nonce harus random per-encrypt")
	}
}

func TestDecryptValue_PlaintextPassthrough(t *testing.T) {
	// Input tanpa prefix ENC: dianggap plaintext, langsung return
	got, err := DecryptValue("plain-text-not-encrypted", nil)
	if err != nil {
		t.Errorf("plaintext passthrough seharusnya tidak error, got: %v", err)
	}
	if got != "plain-text-not-encrypted" {
		t.Errorf("plaintext berubah: %q", got)
	}
}

func TestDecryptValue_KeySalah_Error(t *testing.T) {
	keyA := genKey(t)
	keyB := genKey(t)
	encrypted, _ := EncryptValue("secret", keyA)

	_, err := DecryptValue(encrypted, keyB)
	if err == nil {
		t.Fatal("decrypt dengan key salah harus error (GCM auth fail)")
	}
}

func TestDecryptValue_TamperedCiphertext_Error(t *testing.T) {
	key := genKey(t)
	encrypted, _ := EncryptValue("secret", key)

	// Tamper: flip karakter ke-15 (somewhere in ciphertext)
	tampered := encrypted[:15] + "X" + encrypted[16:]
	_, err := DecryptValue(tampered, key)
	if err == nil {
		t.Fatal("decrypt ciphertext yang ter-modifikasi harus error (GCM auth fail)")
	}
}

func TestEncryptValue_KeySalahSize_Error(t *testing.T) {
	wrongKey := []byte("only-16-bytes-here")
	_, err := EncryptValue("x", wrongKey)
	if !errors.Is(err, ErrMasterKeyInvalid) {
		t.Errorf("err harus wrap ErrMasterKeyInvalid, got: %v", err)
	}
}

func TestIsEncrypted(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"ENC:base64stuff", true},
		{"plain text", false},
		{"", false},
		{"ENC:", true},
	}
	for _, c := range cases {
		if got := IsEncrypted(c.s); got != c.want {
			t.Errorf("IsEncrypted(%q) = %v, want %v", c.s, got, c.want)
		}
	}
}

func TestGenerateMasterKey_Length(t *testing.T) {
	enc, err := GenerateMasterKey()
	if err != nil {
		t.Fatalf("GenerateMasterKey: %v", err)
	}
	k, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(k) != MasterKeySize {
		t.Errorf("key panjang %d, want %d", len(k), MasterKeySize)
	}
}

func TestGenerateMasterKey_Unik(t *testing.T) {
	a, _ := GenerateMasterKey()
	b, _ := GenerateMasterKey()
	if a == b {
		t.Errorf("2 generate harus return key berbeda")
	}
}

func TestGetMasterKey_DariEnvVar(t *testing.T) {
	enc, _ := GenerateMasterKey()
	t.Setenv(MasterKeyEnvVar, enc)

	k, err := GetMasterKey()
	if err != nil {
		t.Fatalf("GetMasterKey: %v", err)
	}
	if len(k) != MasterKeySize {
		t.Errorf("key length salah: %d", len(k))
	}
}

func TestGetMasterKey_EnvKosong_BukanError_PadaPlatformTanpaStore(t *testing.T) {
	t.Setenv(MasterKeyEnvVar, "")
	_, err := GetMasterKey()
	// Di mac dev tanpa keychain item, harus return ErrMasterKeyMissing.
	// Di mac dengan keychain item, akan return key — test ini tetap pass
	// karena err nil-nya juga acceptable.
	if err != nil && !errors.Is(err, ErrMasterKeyMissing) {
		t.Errorf("expected ErrMasterKeyMissing atau nil, got: %v", err)
	}
}
