package vclaim

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
)

// testFixedTS adalah timestamp tetap untuk semua unit test — harus sama
// dengan nilai yang di-inject ke c.now di newTestClient supaya key derivasi
// AES encrypt (test helper) dan decrypt (Client) menghasilkan key yang identik.
const testFixedTS int64 = 1700000000

// encrypt adalah inverse dari decrypt — dipakai oleh test helper untuk
// membuat ciphertext yang valid dari plaintext known. Tidak di-export
// karena production tidak butuh enkripsi (BPJS server yang enkripsi).
func encrypt(consID, secretKey string, ts int64, plaintext []byte) (string, error) {
	keyHash := sha256.Sum256([]byte(consID + secretKey + strconv.FormatInt(ts, 10)))
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
