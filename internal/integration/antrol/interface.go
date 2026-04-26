// Package antrol adalah client untuk BPJS Antrean Online API.
//
// Auth dan signing identik dengan VClaim (HMAC-SHA256), TAPI memakai
// credential terpisah (cons_id & secret berbeda di config.toml).
//
// Status implementasi: interface + mock saja. Real HTTP client akan
// di-implement nanti — auth/encrypt logic bisa di-share dari vclaim
// package via helper kalau perlu.
package antrol

import (
	"context"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

// AntrolClient adalah surface API untuk Antrol Online.
// Detector (P-011) butuh GetBookingHariIni untuk checkMJKN.
//
// Method lain (CreateAntrian, Checkin, dll) akan ditambahkan saat
// Antrian Service & MJKN flow diimplementasikan.
type AntrolClient interface {
	// GetBookingHariIni mengembalikan booking aktif untuk noKartu
	// pada tanggal tgl. Return (nil, nil) jika tidak ada booking
	// (bukan error). Error hanya untuk masalah teknis (network, dll).
	GetBookingHariIni(ctx context.Context, noKartu string, tgl time.Time) (*domain.BookingMJKN, error)
}
