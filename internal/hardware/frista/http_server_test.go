package frista

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// findFreePort cari TCP port bebas untuk test bind. TOCTOU race
// di antara Listen-Close di sini dan Listen di MockReader.startHTTP
// secara teori mungkin, tapi negligible untuk test. Retry kalau perlu.
func findFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("findFreePort: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port
}

// newMockHTTP membuat MockReader dengan HTTP server di port random.
// Cleanup otomatis via t.Cleanup.
func newMockHTTP(t *testing.T) (*MockReader, string) {
	t.Helper()
	port := findFreePort(t)
	m := NewMock(port)

	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = m.Stop() })

	// Tunggu sampai server benar-benar listening (max 500ms)
	addr := m.HTTPAddr()
	if addr == "" {
		t.Fatal("HTTPAddr kosong setelah Start")
	}
	url := "http://" + addr
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond); err == nil {
			conn.Close()
			return m, url
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("server tidak ready dalam 500ms")
	return nil, ""
}

// ============================================================
// POST /mock/card-read
// ============================================================

func TestHTTP_CardRead_Success(t *testing.T) {
	m, url := newMockHTTP(t)

	body := bytes.NewBufferString(`{
        "nik":       "3271234567890001",
        "nama":      "Budi Santoso",
        "tgl_lahir": "1990-01-15",
        "alamat":    "Jl. Merdeka No.1",
        "no_kartu":  "0001234567890012"
    }`)
	resp, err := http.Post(url+"/mock/card-read", "application/json", body)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		dump, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d, want 202. body=%s", resp.StatusCode, dump)
	}

	select {
	case got := <-m.CardRead():
		if got.NIK != "3271234567890001" {
			t.Errorf("NIK = %q", got.NIK)
		}
		if got.Nama != "Budi Santoso" {
			t.Errorf("Nama = %q", got.Nama)
		}
		if got.NoKartu != "0001234567890012" {
			t.Errorf("NoKartu = %q", got.NoKartu)
		}
		if got.Timestamp.IsZero() {
			t.Error("Timestamp seharusnya auto-fill")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout menunggu CardRead")
	}
}

func TestHTTP_CardRead_HanyaNoKartuJugaOK(t *testing.T) {
	m, url := newMockHTTP(t)

	resp, err := http.Post(url+"/mock/card-read", "application/json",
		bytes.NewBufferString(`{"no_kartu":"0001234"}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("status = %d, want 202", resp.StatusCode)
	}
	select {
	case <-m.CardRead():
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout")
	}
}

func TestHTTP_CardRead_NIKDanNoKartuKosong_400(t *testing.T) {
	_, url := newMockHTTP(t)

	resp, err := http.Post(url+"/mock/card-read", "application/json",
		bytes.NewBufferString(`{"nama":"X"}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestHTTP_CardRead_JSONInvalid_400(t *testing.T) {
	_, url := newMockHTTP(t)

	resp, err := http.Post(url+"/mock/card-read", "application/json",
		bytes.NewBufferString(`{this is not json}`))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestHTTP_CardRead_GETNotAllowed_405(t *testing.T) {
	_, url := newMockHTTP(t)

	resp, err := http.Get(url + "/mock/card-read")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", resp.StatusCode)
	}
}

// ============================================================
// POST /mock/card-read-delay
// ============================================================

func TestHTTP_CardReadDelay_AsyncEmit(t *testing.T) {
	m, url := newMockHTTP(t)

	const delaySec = 1
	body := bytes.NewBufferString(`{"nik":"3271234567890001","nama":"Delayed"}`)
	start := time.Now()
	resp, err := http.Post(
		fmt.Sprintf("%s/mock/card-read-delay?seconds=%d", url, delaySec),
		"application/json", body)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	postElapsed := time.Since(start)

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("status = %d, want 202", resp.StatusCode)
	}
	// Request harus return cepat (<200ms), bukan tunggu delaySec
	if postElapsed > 200*time.Millisecond {
		t.Errorf("POST seharusnya async, return cepat. elapsed=%v", postElapsed)
	}

	// CardRead harus muncul kira-kira setelah delaySec detik
	select {
	case got := <-m.CardRead():
		emitElapsed := time.Since(start)
		if got.Nama != "Delayed" {
			t.Errorf("Nama = %q", got.Nama)
		}
		if emitElapsed < time.Duration(delaySec)*time.Second {
			t.Errorf("emit terlalu cepat: elapsed=%v, want >= %ds", emitElapsed, delaySec)
		}
	case <-time.After(time.Duration(delaySec+1) * time.Second):
		t.Fatal("timeout menunggu delayed CardRead")
	}
}

func TestHTTP_CardReadDelay_DefaultSeconds(t *testing.T) {
	_, url := newMockHTTP(t)

	body := bytes.NewBufferString(`{"nik":"X","nama":"X"}`)
	resp, err := http.Post(url+"/mock/card-read-delay", "application/json", body)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		t.Errorf("status = %d, want 202", resp.StatusCode)
	}

	var body2 map[string]any
	json.NewDecoder(resp.Body).Decode(&body2)
	if body2["scheduled_in_sec"].(float64) != 3 {
		t.Errorf("default seconds = %v, want 3", body2["scheduled_in_sec"])
	}
}

func TestHTTP_CardReadDelay_SecondsOutOfRange_400(t *testing.T) {
	_, url := newMockHTTP(t)
	for _, sec := range []string{"-1", "100", "abc"} {
		resp, _ := http.Post(url+"/mock/card-read-delay?seconds="+sec,
			"application/json", strings.NewReader(`{"nik":"X"}`))
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("seconds=%s: status = %d, want 400", sec, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

// ============================================================
// GET /
// ============================================================

func TestHTTP_InfoPage_HTMLRendered(t *testing.T) {
	_, url := newMockHTTP(t)

	resp, err := http.Get(url + "/")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	html := string(body)
	for _, sub := range []string{"Frista Mock", "/mock/card-read", "POST", "curl"} {
		if !strings.Contains(html, sub) {
			t.Errorf("HTML tidak mengandung %q", sub)
		}
	}
}

func TestHTTP_InfoPage_PathTidakAda_404(t *testing.T) {
	_, url := newMockHTTP(t)

	resp, err := http.Get(url + "/random-path")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

// ============================================================
// Lifecycle: Stop benar-benar shutdown server (port reusable)
// ============================================================

func TestHTTP_StopShutdownServer_PortReusable(t *testing.T) {
	port := findFreePort(t)

	// Start mock #1
	m1 := NewMock(port)
	if err := m1.Start(context.Background()); err != nil {
		t.Fatalf("Start m1: %v", err)
	}

	// Verifikasi server #1 listening
	if _, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", port)); err != nil {
		_ = m1.Stop()
		t.Fatalf("server m1 tidak listening: %v", err)
	}

	// Stop m1 — port harus released
	if err := m1.Stop(); err != nil {
		t.Fatalf("Stop m1: %v", err)
	}

	// Tunggu OS release port (kadang ada TIME_WAIT delay singkat)
	time.Sleep(50 * time.Millisecond)

	// Start mock #2 di port yang sama — harus sukses (port reusable)
	m2 := NewMock(port)
	if err := m2.Start(context.Background()); err != nil {
		t.Fatalf("Start m2 di port yang sama gagal — Stop m1 tidak release port: %v", err)
	}
	defer m2.Stop()

	if _, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/", port)); err != nil {
		t.Fatalf("server m2 tidak listening: %v", err)
	}
}

func TestHTTP_NewMockPort0_TidakSpawnServer(t *testing.T) {
	// Backward-compat dengan P-030: port=0 → no HTTP server
	m := NewMock(0)
	if err := m.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer m.Stop()

	if m.HTTPAddr() != "" {
		t.Errorf("port=0 seharusnya tidak spawn server, got addr=%q", m.HTTPAddr())
	}
}

func TestHTTP_PortBentrok_StartError(t *testing.T) {
	port := findFreePort(t)

	// Bind manual ke port — simulasi port sudah dipakai
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()

	m := NewMock(port)
	err = m.Start(context.Background())
	if err == nil {
		_ = m.Stop()
		t.Fatal("Start ke port yang sudah dipakai harus error")
	}
}
