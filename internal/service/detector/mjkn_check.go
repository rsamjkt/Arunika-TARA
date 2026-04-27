package detector

import (
	"context"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

// checkMJKN memeriksa apakah pasien punya booking aktif di Mobile JKN
// untuk hari ini, dengan fallback chain:
//
//  1. Antrol HTTP API (BPJS official source)
//  2. Khanza direct-DB `booking_registrasi` (kalau Antrol error/empty)
//
// Fallback ke Khanza penting karena:
//   - Antrol API endpoint development BPJS sering down (rate limit / IP whitelist)
//   - RS yang pakai sistem booking lokal kadang sync ke booking_registrasi
//     bukan ke Antrol — kita tetap mau detect "ada booking hari ini"
//
// Kedua sumber dianggap setara untuk klasifikasi MJKN — pakai whichever hits dulu.
func (d *Detector) checkMJKN(ctx context.Context, noKartu, noRM string, today time.Time, ch chan<- checkResult) {
	d.safeRun(domain.PatientTypeMJKN, "MJKN", func() (any, bool, error) {
		// Source 1: Antrol API
		if noKartu != "" {
			booking, err := d.antrol.GetBookingHariIni(ctx, noKartu, today)
			if err == nil && booking != nil && booking.IsValidUntuk(today.Format("2006-01-02")) {
				return booking, true, nil
			}
			if err != nil {
				d.logger.Debug("detector: antrol gagal, fallback ke khanza booking_registrasi",
					"err", err.Error())
			}
		}

		// Source 2: Khanza direct-DB fallback
		if noRM == "" {
			return nil, false, nil
		}
		booking, err := d.khanza.GetBookingMJKN(ctx, noRM, today)
		if err != nil {
			return nil, false, err
		}
		if booking == nil {
			return nil, false, nil
		}
		if !booking.IsValidUntuk(today.Format("2006-01-02")) {
			return nil, false, nil
		}
		return booking, true, nil
	}, ch)
}
