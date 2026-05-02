// Diagnostic tool: test koneksi BPJS VClaim production.
// Jalankan: go run ./cmd/bpjs-ping
// Atau:     go run ./cmd/bpjs-ping <path/to/config.toml>
//
// Output: full HTTP request + response transcript untuk diagnosa 403/4xx.
package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptrace"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/arunika/apm-go/internal/config"
)

func main() {
	cfgPath := "config.toml"
	testNoKartu := "0002076061241" // nomor kartu dev default
	if len(os.Args) > 1 {
		cfgPath = os.Args[1]
	}
	if len(os.Args) > 2 {
		testNoKartu = os.Args[2]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fatalf("Gagal load config %s: %v\n", cfgPath, err)
	}

	bpjs := cfg.BPJS
	if bpjs.Mock {
		fatalf("config.toml: bpjs.mock = true — ubah ke false dulu\n")
	}
	if bpjs.ConsID == "" || bpjs.ConsumerSecret == "" {
		fatalf("config.toml: cons_id / consumer_secret kosong\n")
	}

	baseURL := strings.TrimRight(bpjs.VClaimURL, "/")
	testDate := time.Now().Format("2006-01-02")
	path := fmt.Sprintf("/Peserta/noKartu/%s/tglSEP/%s", testNoKartu, testDate)
	fullURL := baseURL + path

	ts := time.Now().Unix()
	sig := sign(bpjs.ConsID, bpjs.ConsumerSecret, ts)

	headers := map[string]string{
		"X-cons-id":    bpjs.ConsID,
		"X-timestamp":  strconv.FormatInt(ts, 10),
		"X-signature":  sig,
		"Content-Type": "application/json",
		"User-Agent":   "APM-TARA/1.0 (bpjs-ping diagnostic)",
	}
	if bpjs.UserKey != "" {
		headers["user_key"] = bpjs.UserKey
	}

	fmt.Println("=== BPJS VClaim Diagnostic ===")
	fmt.Printf("URL       : %s\n", fullURL)
	fmt.Printf("cons_id   : %s\n", bpjs.ConsID)
	fmt.Printf("timestamp : %d\n", ts)
	fmt.Printf("signature : %s\n", sig)
	if bpjs.UserKey != "" {
		fmt.Printf("user_key  : %s\n", bpjs.UserKey)
	} else {
		fmt.Println("user_key  : (kosong — tidak dikirim)")
	}
	fmt.Println()

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		fatalf("Buat request gagal: %v\n", err)
	}
	// Bypass Go's header normalization — BPJS requires exact-case headers.
	// net/http.Header.Set() normalizes e.g. "X-cons-id" → "X-Cons-Id".
	for k, v := range headers {
		req.Header[k] = []string{v}
	}

	fmt.Println("--- Request Headers ---")
	for k, vs := range req.Header {
		fmt.Printf("  %s: %s\n", k, strings.Join(vs, ", "))
	}
	fmt.Println()

	// trace TLS handshake untuk deteksi TLS issue
	var tlsState *tls.ConnectionState
	trace := &httptrace.ClientTrace{
		TLSHandshakeDone: func(s tls.ConnectionState, err error) {
			if err != nil {
				fmt.Printf("[TLS] Handshake gagal: %v\n", err)
			} else {
				fmt.Printf("[TLS] Handshake OK — server: %s, proto: %s\n",
					s.ServerName, s.NegotiatedProtocol)
				tlsState = &s
			}
		},
		GotConn: func(info httptrace.GotConnInfo) {
			fmt.Printf("[TCP] Terhubung ke %s (reused=%v)\n",
				info.Conn.RemoteAddr(), info.Reused)
		},
	}
	_ = tlsState
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))

	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false, // strict TLS verification
			},
		},
	}

	fmt.Println("--- Mengirim Request... ---")
	resp, err := client.Do(req)
	if err != nil {
		fatalf("HTTP error: %v\n", err)
	}
	defer resp.Body.Close()

	fmt.Printf("\n--- Response ---\n")
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Println("Headers:")
	for k, vs := range resp.Header {
		fmt.Printf("  %s: %s\n", k, strings.Join(vs, ", "))
	}

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("\nBody (%d bytes):\n%s\n", len(body), string(body))

	fmt.Println()
	if resp.StatusCode == 200 {
		fmt.Println("[OK] Server merespons 200 — autentikasi berhasil!")
		fmt.Println("     (Data peserta mungkin tidak ditemukan karena nomor test, itu normal)")
	} else if resp.StatusCode == 403 {
		fmt.Println("[ERROR 403] Akses ditolak. Kemungkinan penyebab:")
		fmt.Println("  1. IP mesin ini belum didaftarkan ke BPJS untuk cons_id ini")
		fmt.Println("  2. cons_id tidak aktif di lingkungan production")
		fmt.Println("  3. Signature salah (periksa consumer_secret di config.toml)")
	} else if resp.StatusCode == 401 {
		fmt.Println("[ERROR 401] Autentikasi gagal — signature atau cons_id salah")
	} else {
		fmt.Printf("[ERROR %d] Respons tidak terduga\n", resp.StatusCode)
	}
}

func sign(consID, secretKey string, timestamp int64) string {
	msg := consID + "&" + strconv.FormatInt(timestamp, 10)
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(msg))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[FATAL] "+format, args...)
	os.Exit(1)
}
