package vclaim

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
)

// errInvalidCiphertext dikembalikan saat ciphertext tidak memenuhi syarat
// AES-CBC (panjang bukan kelipatan block size, atau padding rusak).
var errInvalidCiphertext = errors.New("ciphertext VClaim invalid")

// decrypt mendekripsi response body VClaim v2.0.
//
// Skema:
//  1. base64 decode → raw ciphertext
//  2. AES-256-CBC decrypt:
//     key = SHA256(userKey)           — jika user_key di-set (rekomendasi BPJS)
//         = SHA256(secretKey+consID)  — fallback jika user_key kosong
//     IV  = key[:16]                  — first 16 bytes of same hash
//  3. PKCS7 unpad
//
// BPJS server meng-enkripsi response menggunakan user_key yang dikirim
// di header request. Client harus decrypt dengan key yang sama.
func (c *Client) decrypt(ciphertext string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	// Pakai user_key kalau tersedia — BPJS server enkripsi response
	// menggunakan user_key yang dikirim di header. Fallback ke
	// secretKey+consID untuk kompatibilitas implementasi lama.
	keyInput := c.userKey
	if keyInput == "" {
		keyInput = c.secretKey + c.consID
	}
	keyHash := sha256.Sum256([]byte(keyInput))
	key := keyHash[:]
	iv := key[:aes.BlockSize] // 16 bytes

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes new cipher: %w", err)
	}

	if len(raw) == 0 || len(raw)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("%w: panjang %d bukan kelipatan %d",
			errInvalidCiphertext, len(raw), aes.BlockSize)
	}

	plain := make([]byte, len(raw))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plain, raw)

	unpadded, err := pkcs7Unpad(plain, aes.BlockSize)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errInvalidCiphertext, err)
	}
	return unpadded, nil
}

// pkcs7Unpad menghapus PKCS#7 padding dari blok terakhir.
// Padding valid: setiap byte padding bernilai n, dan ada n byte di akhir.
func pkcs7Unpad(b []byte, blockSize int) ([]byte, error) {
	n := len(b)
	if n == 0 {
		return nil, errors.New("pad: input kosong")
	}
	pad := int(b[n-1])
	if pad == 0 || pad > blockSize {
		return nil, fmt.Errorf("pad: byte padding invalid (%d)", pad)
	}
	if n < pad {
		return nil, fmt.Errorf("pad: input lebih pendek dari padding (%d < %d)", n, pad)
	}
	for i := n - pad; i < n; i++ {
		if int(b[i]) != pad {
			return nil, fmt.Errorf("pad: byte ke-%d harusnya %d, got %d", i, pad, b[i])
		}
	}
	return b[:n-pad], nil
}
