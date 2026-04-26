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
//
// Method:
//   - GetBookingHariIni: dipakai detector untuk checkMJKN.
//   - PushAntrian: dipakai antrian service untuk fire-and-forget
//     mempublikasikan nomor antrian ke Mobile JKN.
type AntrolClient interface {
	// GetBookingHariIni mengembalikan booking aktif untuk noKartu
	// pada tanggal tgl. Return (nil, nil) jika tidak ada booking
	// (bukan error). Error hanya untuk masalah teknis (network, dll).
	GetBookingHariIni(ctx context.Context, noKartu string, tgl time.Time) (*domain.BookingMJKN, error)

	// PushAntrian mempublikasikan nomor antrian poli ke Antrol agar
	// muncul di Mobile JKN pasien. Caller pakai pola fire-and-forget:
	// error di-log tapi TIDAK menggagalkan flow utama (cetak tiket
	// tetap berjalan).
	PushAntrian(ctx context.Context, req domain.AntrianRequest, ticket *domain.Ticket) error
}
