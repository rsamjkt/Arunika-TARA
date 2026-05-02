package vclaim

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
)

var errInvalidCiphertext = errors.New("ciphertext VClaim invalid")

// decrypt mendekripsi response body VClaim v2.0.
//
// Skema (dari SIMRS-Khanza / ApiBPJS.java):
//  1. base64 decode → raw ciphertext
//  2. AES-256-CBC decrypt:
//     key = SHA256(consID + consumerSecret + timestamp)
//     IV  = key[:16]
//  3. PKCS7 unpad
//  4. LZString decompressFromEncodedURIComponent → plaintext JSON
//
// timestamp HARUS sama dengan X-Timestamp yang dikirim di request header,
// karena BPJS server menggunakan nilai itu saat mengenkripsi response.
func (c *Client) decrypt(ciphertext string, timestamp int64) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}

	// Key = SHA256(consID + consumerSecret + timestamp)
	keyInput := c.consID + c.secretKey + strconv.FormatInt(timestamp, 10)
	keyHash := sha256.Sum256([]byte(keyInput))
	key := keyHash[:]
	iv := key[:aes.BlockSize]

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

	// Setelah AES decrypt, data biasanya ter-kompres dengan LZString.
	// Jika dekompresi gagal tapi byte mentah sudah valid JSON (mulai {/[),
	// gunakan langsung — mendukung mock/test dan endpoint legacy.
	decompressed, lzErr := lzDecompressURIComponent(string(unpadded))
	if lzErr != nil {
		first := unpadded[0]
		if first == '{' || first == '[' {
			return unpadded, nil
		}
		return nil, fmt.Errorf("lzstring decompress: %w", lzErr)
	}
	return []byte(decompressed), nil
}

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
