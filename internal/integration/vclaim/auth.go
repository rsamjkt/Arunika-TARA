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

// headers mengembalikan header BPJS untuk request dengan timestamp yang diberikan.
// Timestamp di-pass eksplisit supaya nilai yang sama bisa dipakai untuk
// derive AES decrypt key di parseEnvelope.
func (c *Client) headers(ts int64) map[string]string {
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
