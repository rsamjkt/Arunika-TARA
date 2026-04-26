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

// headers mengembalikan tiga header BPJS yang wajib ada di tiap request:
// X-cons-id, X-timestamp, X-signature.
//
// Timestamp diambil dari c.now() supaya bisa di-stub di test
// (lihat client.setNow).
func (c *Client) headers() map[string]string {
	ts := c.now().Unix()
	return map[string]string{
		"X-cons-id":   c.consID,
		"X-timestamp": strconv.FormatInt(ts, 10),
		"X-signature": c.sign(ts),
	}
}
