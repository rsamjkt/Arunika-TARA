package vclaim

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
)

// encrypt adalah inverse dari decrypt — dipakai oleh test helper untuk
// membuat ciphertext yang valid dari plaintext known. Tidak di-export
// karena production tidak butuh enkripsi (BPJS server yang enkripsi).
//
// Walaupun fungsi ini hanya dipanggil dari test, kita taruh di file
// non-_test.go supaya bisa dipakai oleh test di package eksternal kalau
// suatu hari diperlukan (mis. integration test di service layer).
func encrypt(secretKey, consID string, plaintext []byte) (string, error) {
	keyHash := sha256.Sum256([]byte(secretKey + consID))
	key := keyHash[:]
	iv := key[:aes.BlockSize]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	padded := pkcs7Pad(plaintext, aes.BlockSize)
	cipherBytes := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(cipherBytes, padded)

	return base64.StdEncoding.EncodeToString(cipherBytes), nil
}

// pkcs7Pad menambahkan padding PKCS#7 — counterpart dari pkcs7Unpad
// di decrypt.go. Selalu menambah minimal 1 byte (jika len kelipatan
// blockSize, tambah 1 blok penuh berisi blockSize).
func pkcs7Pad(b []byte, blockSize int) []byte {
	pad := blockSize - (len(b) % blockSize)
	out := make([]byte, len(b)+pad)
	copy(out, b)
	for i := len(b); i < len(out); i++ {
		out[i] = byte(pad)
	}
	return out
}
