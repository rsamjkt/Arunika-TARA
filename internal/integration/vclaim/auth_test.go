package vclaim

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"testing"
)

func TestVClaim_Sign_TestVector(t *testing.T) {
	c := &Client{
		consID:    "12345",
		secretKey: "rahasia-bpjs",
	}
	const ts int64 = 1700000000

	got := c.sign(ts)

	// Hitung expected secara manual dengan stdlib — kalau implementasi
	// pakai algoritma yang sama, hasilnya match.
	mac := hmac.New(sha256.New, []byte("rahasia-bpjs"))
	mac.Write([]byte("12345&" + strconv.FormatInt(ts, 10)))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if got != want {
		t.Errorf("sign() = %q, want %q", got, want)
	}
}

func TestVClaim_Sign_TimestampMengubahHasil(t *testing.T) {
	c := &Client{consID: "C", secretKey: "S"}
	if c.sign(1) == c.sign(2) {
		t.Errorf("signature seharusnya berbeda untuk timestamp berbeda")
	}
}

func TestVClaim_Sign_SecretSalahMengubahHasil(t *testing.T) {
	a := (&Client{consID: "C", secretKey: "secret-A"}).sign(123)
	b := (&Client{consID: "C", secretKey: "secret-B"}).sign(123)
	if a == b {
		t.Errorf("signature seharusnya berbeda untuk secret berbeda")
	}
}
