package detector

import (
	"context"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

// checkMJKN memeriksa apakah pasien punya booking aktif di Mobile JKN
// untuk hari ini.
func (d *Detector) checkMJKN(ctx context.Context, noKartu string, today time.Time, ch chan<- checkResult) {
	d.safeRun(domain.PatientTypeMJKN, "MJKN", func() (any, bool, error) {
		if noKartu == "" {
			return nil, false, nil
		}
		booking, err := d.antrol.GetBookingHariIni(ctx, noKartu, today)
		if err != nil {
			return nil, false, err
		}
		if booking == nil {
			return nil, false, nil
		}
		// Defensive: BPJS Antrol kadang return booking di luar tanggal —
		// validasi tanggalnya sebelum dianggap hit.
		if !booking.IsValidUntuk(today.Format("2006-01-02")) {
			return nil, false, nil
		}
		return booking, true, nil
	}, ch)
}
