// Package log menyediakan PHI-masking layer di atas slog.
//
// Production setup:
//
//	base := slog.NewJSONHandler(logFile, &slog.HandlerOptions{Level: slog.LevelInfo})
//	logger := slog.New(log.NewPHIMaskingHandler(base))
//	slog.SetDefault(logger)
//
// Setelah ini, slog.Info("foo", "nik", "3271234567890001") akan otomatis
// menghasilkan output dengan field nik="***" — JANGAN ada PHI mentah
// pernah masuk ke disk/stdout.
package log

import (
	"context"
	"log/slog"
	"regexp"
	"strings"
)

// sensitiveKeys: nama field yang SELALU di-mask jadi "***" tanpa peduli
// nilainya. Case-insensitive match. Ini fail-safe pertama — meskipun
// developer accidentally pass plaintext PHI, key-name yang sensitif
// akan tetap mask.
var sensitiveKeys = map[string]struct{}{
	"nik":          {},
	"no_kartu":     {},
	"nokartu":      {},
	"no_rm":        {},
	"norm":         {},
	"username":     {},
	"password":     {},
	"token":        {},
	"secret":       {},
	"consumer_secret": {},
	"api_key":      {},
	"finger":       {}, // FP token
	"fp_token":     {},
}

// digit16 = pattern PHI: 16 digit angka berurutan (NIK / No Kartu BPJS).
// Match exact 16-digit untuk hindari false-positive nomor lain.
var digit16 = regexp.MustCompile(`^\d{16}$`)

// digit10plus = relaxed pattern: 10+ digit angka kemungkinan PHI
// (sebagai fallback). Threshold 10 supaya tidak match short ID seperti
// IP / port / counter angka kecil.
var digit10plus = regexp.MustCompile(`^\d{10,}$`)

// PHIMaskingHandler wraps slog.Handler dan mask PHI sebelum forward
// ke inner handler. Implements slog.Handler interface.
//
// Jika inner nil, default ke slog.Default().Handler() saat Handle.
type PHIMaskingHandler struct {
	inner slog.Handler
}

// NewPHIMaskingHandler bungkus inner. Nil inner OK (default ke
// slog.Default fallback saat Handle).
func NewPHIMaskingHandler(inner slog.Handler) *PHIMaskingHandler {
	return &PHIMaskingHandler{inner: inner}
}

// Enabled forward ke inner.
func (h *PHIMaskingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if h.inner == nil {
		return true
	}
	return h.inner.Enabled(ctx, level)
}

// Handle build new record dengan masked attrs, lalu forward.
func (h *PHIMaskingHandler) Handle(ctx context.Context, r slog.Record) error {
	inner := h.inner
	if inner == nil {
		inner = slog.Default().Handler()
	}

	// Construct new record dengan attrs yang sudah ter-mask
	newR := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	r.Attrs(func(a slog.Attr) bool {
		newR.AddAttrs(maskAttr(a))
		return true
	})
	return inner.Handle(ctx, newR)
}

// WithAttrs return new handler dengan attrs di-mask juga.
func (h *PHIMaskingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if h.inner == nil {
		return h
	}
	masked := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		masked[i] = maskAttr(a)
	}
	return &PHIMaskingHandler{inner: h.inner.WithAttrs(masked)}
}

// WithGroup forward ke inner.
func (h *PHIMaskingHandler) WithGroup(name string) slog.Handler {
	if h.inner == nil {
		return h
	}
	return &PHIMaskingHandler{inner: h.inner.WithGroup(name)}
}

// ============================================================
// Masking core
// ============================================================

// maskAttr return attr yang sudah ter-mask kalau qualified.
//   - Nama key di sensitiveKeys → "***"
//   - Nilai 16-digit → "****<last4>"
//   - Nilai 10+ digit → "****<last4>" (defensive)
//   - Lainnya → unchanged
//
// Untuk nested attr (slog.Group), recursively mask.
func maskAttr(a slog.Attr) slog.Attr {
	// Group: recurse ke nested attrs
	if a.Value.Kind() == slog.KindGroup {
		nested := a.Value.Group()
		out := make([]slog.Attr, len(nested))
		for i, n := range nested {
			out[i] = maskAttr(n)
		}
		return slog.Group(a.Key, anyAttrSlice(out)...)
	}

	keyLower := strings.ToLower(a.Key)
	if _, ok := sensitiveKeys[keyLower]; ok {
		return slog.String(a.Key, "***")
	}

	// Pattern match value (kalau string)
	if a.Value.Kind() == slog.KindString {
		s := a.Value.String()
		if masked := maskValue(s); masked != s {
			return slog.String(a.Key, masked)
		}
	}

	return a
}

// maskValue heuristik: 16-digit exact → mask, 10+ digit → mask too.
// Return original kalau tidak match pattern PHI.
func maskValue(s string) string {
	if digit16.MatchString(s) {
		return maskDigits(s)
	}
	if digit10plus.MatchString(s) {
		return maskDigits(s)
	}
	return s
}

// maskDigits keep last 4 digits, replace rest dengan '*'.
func maskDigits(s string) string {
	if len(s) <= 4 {
		return strings.Repeat("*", len(s))
	}
	return strings.Repeat("*", len(s)-4) + s[len(s)-4:]
}

// anyAttrSlice convert []slog.Attr ke []any untuk slog.Group var-args.
func anyAttrSlice(attrs []slog.Attr) []any {
	out := make([]any, len(attrs))
	for i, a := range attrs {
		out[i] = a
	}
	return out
}
