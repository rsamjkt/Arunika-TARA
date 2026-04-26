package log

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

// captureLogger build slog.Logger dengan PHIMaskingHandler wrapping
// JSON handler ke buffer — dipakai test untuk inspeksi output.
func captureLogger(t *testing.T) (*slog.Logger, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	base := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger := slog.New(NewPHIMaskingHandler(base))
	return logger, &buf
}

// parseLog return last log line as map[string]any
func parseLog(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	out := strings.TrimSpace(buf.String())
	if out == "" {
		return nil
	}
	// Ambil baris terakhir kalau multiple
	lines := strings.Split(out, "\n")
	last := lines[len(lines)-1]

	var m map[string]any
	if err := json.Unmarshal([]byte(last), &m); err != nil {
		t.Fatalf("parse JSON: %v\nline: %s", err, last)
	}
	return m
}

// ============================================================
// Sensitive keys → "***"
// ============================================================

func TestPHI_SensitiveKey_NIK_Masked(t *testing.T) {
	logger, buf := captureLogger(t)
	logger.Info("test", "nik", "3271234567890001")

	out := parseLog(t, buf)
	if out["nik"] != "***" {
		t.Errorf("nik = %v, want ***", out["nik"])
	}
	if strings.Contains(buf.String(), "3271234567890001") {
		t.Errorf("output mengandung NIK asli — PHI bocor!")
	}
}

func TestPHI_SensitiveKey_AllVariants(t *testing.T) {
	cases := []string{
		"nik", "NIK", "no_kartu", "nokartu", "no_rm", "norm",
		"username", "password", "token", "secret",
		"consumer_secret", "api_key", "finger", "fp_token",
	}
	for _, key := range cases {
		t.Run(key, func(t *testing.T) {
			logger, buf := captureLogger(t)
			logger.Info("test", key, "should-be-masked-no-matter-what")

			out := parseLog(t, buf)
			if out[strings.ToLower(key)] == nil && out[key] == nil {
				t.Errorf("output tidak punya key %q", key)
			}
			val, _ := out[strings.ToLower(key)].(string)
			if val == "" {
				val, _ = out[key].(string)
			}
			if val != "***" {
				t.Errorf("key %q value = %q, want ***", key, val)
			}
			if strings.Contains(buf.String(), "should-be-masked-no-matter-what") {
				t.Errorf("plaintext nilai bocor untuk key %q", key)
			}
		})
	}
}

// ============================================================
// Pattern 16-digit → mask last 4
// ============================================================

func TestPHI_Pattern_16DigitMasked(t *testing.T) {
	logger, buf := captureLogger(t)
	logger.Info("test", "some_field", "3271234567890001")

	out := parseLog(t, buf)
	got := out["some_field"]
	want := "************0001"
	if got != want {
		t.Errorf("16-digit masked: got %v, want %v", got, want)
	}
	if strings.Contains(buf.String(), "3271234567890001") {
		t.Errorf("16-digit raw bocor")
	}
}

func TestPHI_Pattern_10PlusDigitMasked(t *testing.T) {
	logger, buf := captureLogger(t)
	logger.Info("test", "phone", "08123456789") // 11 digit

	out := parseLog(t, buf)
	got, _ := out["phone"].(string)
	if !strings.HasSuffix(got, "6789") {
		t.Errorf("10+ digit masked salah: %q", got)
	}
	if !strings.HasPrefix(got, "*") {
		t.Errorf("10+ digit harus prefix '*', got %q", got)
	}
}

func TestPHI_Pattern_DigitPendekTidakMasked(t *testing.T) {
	logger, buf := captureLogger(t)
	// 5 digit — bukan PHI, jangan di-mask
	logger.Info("test", "no_urut", "12345")

	out := parseLog(t, buf)
	if out["no_urut"] != "12345" {
		t.Errorf("digit pendek seharusnya tidak di-mask: got %v", out["no_urut"])
	}
}

// ============================================================
// Non-string values
// ============================================================

func TestPHI_NonStringValue_Passthrough(t *testing.T) {
	logger, buf := captureLogger(t)
	logger.Info("test", "count", 42, "active", true, "rate", 3.14)

	out := parseLog(t, buf)
	if int(out["count"].(float64)) != 42 {
		t.Errorf("int value berubah: %v", out["count"])
	}
	if out["active"] != true {
		t.Errorf("bool value berubah: %v", out["active"])
	}
}

// ============================================================
// Sensitive key dengan non-string value (mis. token=int)
// ============================================================

func TestPHI_SensitiveKey_NonStringValue_TetapMasked(t *testing.T) {
	logger, buf := captureLogger(t)
	logger.Info("test", "token", 123456789)

	out := parseLog(t, buf)
	if out["token"] != "***" {
		t.Errorf("sensitive key dengan int value harus tetap masked: %v", out["token"])
	}
}

// ============================================================
// Nested groups
// ============================================================

func TestPHI_Group_NestedFieldMasked(t *testing.T) {
	logger, buf := captureLogger(t)
	logger.Info("test",
		slog.Group("user",
			"nik", "3271234567890001",
			"nama", "Budi"),
	)

	output := buf.String()
	if strings.Contains(output, "3271234567890001") {
		t.Errorf("NIK di group bocor: %s", output)
	}
	if !strings.Contains(output, "Budi") {
		t.Errorf("non-PHI di group seharusnya tetap ada")
	}
}

// ============================================================
// WithAttrs propagation
// ============================================================

func TestPHI_WithAttrs_StillMasks(t *testing.T) {
	logger, buf := captureLogger(t)
	scoped := logger.With("nik", "3271234567890001", "request_id", "abc-123")
	scoped.Info("test")

	out := parseLog(t, buf)
	if out["nik"] != "***" {
		t.Errorf("nik via With() tidak di-mask: %v", out["nik"])
	}
	if out["request_id"] != "abc-123" {
		t.Errorf("non-sensitive via With() berubah: %v", out["request_id"])
	}
}

// ============================================================
// Enabled forward
// ============================================================

func TestPHI_Enabled_ForwardsToInner(t *testing.T) {
	var buf bytes.Buffer
	base := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	h := NewPHIMaskingHandler(base)

	if h.Enabled(context.Background(), slog.LevelWarn) != true {
		t.Errorf("Warn level harus enabled")
	}
	if h.Enabled(context.Background(), slog.LevelDebug) != false {
		t.Errorf("Debug level harus disabled (level=Warn)")
	}
}

// ============================================================
// maskDigits unit test
// ============================================================

func TestMaskDigits(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"3271234567890001", "************0001"},
		{"1234567890", "******7890"},
		{"abc", "***"},
		{"", ""},
		{"12", "**"},
		{"1234", "****"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := maskDigits(c.in); got != c.want {
				t.Errorf("maskDigits(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
