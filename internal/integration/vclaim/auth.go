package vclaim

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
)

// sign menghitung X-signature header untuk satu request.
//
// Formula resmi BPJS:
//
//	signature = base64( HMAC-SHA256( consID + "&" + timestamp, secretKey ) )
//
// timestamp adalah Unix epoch detik (UTC).
func (c *Client) sign(timestamp int64) string {
	msg := c.consID + "&" + strconv.FormatInt(timestamp, 10)
	mac := hmac.New(sha256.New, []byte(c.secretKey))
	mac.Write([]byte(msg))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// headers mengembalikan empat header BPJS yang wajib ada di tiap request:
// X-cons-id, X-timestamp, X-signature, user_key.
//
// user_key dipakai BPJS server untuk derive AES-256-CBC key buat decrypt
// response payload. Kalau kosong, response akan blank atau encryption
// header akan hilang.
//
// Timestamp diambil dari c.now() supaya bisa di-stub di test.
func (c *Client) headers() map[string]string {
	ts := c.now().Unix()
	h := map[string]string{
		"X-cons-id":   c.consID,
		"X-timestamp": strconv.FormatInt(ts, 10),
		"X-signature": c.sign(ts),
	}
	if c.userKey != "" {
		h["user_key"] = c.userKey
	}
	return h
}
