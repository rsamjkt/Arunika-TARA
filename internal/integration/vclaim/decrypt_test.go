package vclaim

import (
	"strings"
	"testing"
)

func TestVClaim_Decrypt_RoundTripPlaintextKnown(t *testing.T) {
	c := &Client{consID: "12345", secretKey: "rahasia-bpjs"}

	plaintext := []byte(`{"noKartu":"0001234567890012","nama":"Budi Santoso"}`)
	cipher, err := encrypt(c.consID, c.secretKey, testFixedTS, plaintext)
	if err != nil {
		t.Fatalf("encrypt helper: %v", err)
	}

	got, err := c.decrypt(cipher, testFixedTS)
	if err != nil {
		t.Fatalf("decrypt() error = %v", err)
	}
	if string(got) != string(plaintext) {
		t.Errorf("decrypt round-trip mismatch:\n  got  = %s\n  want = %s",
			string(got), string(plaintext))
	}
}

func TestVClaim_Decrypt_PlaintextPanjang_LebihDariSatuBlok(t *testing.T) {
	c := &Client{consID: "C", secretKey: "S"}
	// Plaintext panjang berupa JSON supaya fallback path (non-LZString) aktif.
	plain := []byte(`{"data":"` + strings.Repeat("x", 600) + `"}`)
	cipher, err := encrypt(c.consID, c.secretKey, testFixedTS, plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	got, err := c.decrypt(cipher, testFixedTS)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(got) != string(plain) {
		t.Errorf("round-trip panjang gagal")
	}
}

func TestVClaim_Decrypt_KeyBerbeda_HarusGagalAtauOutputBeda(t *testing.T) {
	a := &Client{consID: "C", secretKey: "secret-A"}
	b := &Client{consID: "C", secretKey: "secret-B"}

	plain := []byte("test ciphertext")
	cipher, _ := encrypt(a.consID, a.secretKey, testFixedTS, plain)

	got, err := b.decrypt(cipher, testFixedTS)
	if err == nil && string(got) == string(plain) {
		t.Errorf("decrypt dengan secret berbeda harus gagal — got plaintext yang sama")
	}
}

func TestVClaim_Decrypt_Base64Invalid(t *testing.T) {
	c := &Client{consID: "C", secretKey: "S"}
	_, err := c.decrypt("***this is not base64***", testFixedTS)
	if err == nil {
		t.Fatal("decrypt(invalid base64) expected error, got nil")
	}
}

func TestVClaim_Decrypt_PanjangBukanKelipatanBlok(t *testing.T) {
	c := &Client{consID: "C", secretKey: "S"}
	// 17 bytes random base64 — bukan kelipatan 16
	_, err := c.decrypt("AAECAwQFBgcICQoLDA0ODxA=", testFixedTS)
	if err == nil {
		t.Fatal("decrypt(non-blocksize) expected error, got nil")
	}
}

func TestVClaim_Decrypt_StringKosong(t *testing.T) {
	c := &Client{consID: "C", secretKey: "S"}
	_, err := c.decrypt("", testFixedTS)
	if err == nil {
		t.Fatal("decrypt(\"\") expected error")
	}
}
