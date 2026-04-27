package detector

import (
	"context"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

// postRANAPWindowDays adalah jumlah hari ke belakang untuk deteksi
// "baru keluar dari rawat inap". 7 hari sesuai window kontrol BPJS:
// SEP kontrol pasca-RI valid kalau kontrol terjadi ≤7 hari setelah
// pulang RI tanpa perlu rujukan baru. Bisa di-override per RS dari
// config nanti.
const postRANAPWindowDays = 7

// checkPostRANAP memeriksa apakah pasien baru saja keluar dari rawat
// inap dalam postRANAPWindowDays hari terakhir.
func (d *Detector) checkPostRANAP(ctx context.Context, noRM string, today time.Time, ch chan<- checkResult) {
	d.safeRun(domain.PatientTypePostRANAP, "PostRANAP", func() (any, bool, error) {
		if noRM == "" {
			return nil, false, nil
		}
		list, err := d.khanza.GetRiwayatRANAP(ctx, noRM)
		if err != nil {
			return nil, false, err
		}

		windowStart := today.AddDate(0, 0, -postRANAPWindowDays)
		var hits []domain.RiwayatRANAP
		for i := range list {
			if list[i].IsKeluarSejak(windowStart) {
				hits = append(hits, list[i])
			}
		}
		if len(hits) == 0 {
			return nil, false, nil
		}
		return hits, true, nil
	}, ch)
}
