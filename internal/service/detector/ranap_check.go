package detector

import (
	"context"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

// checkPostRANAP memeriksa apakah pasien baru saja keluar dari rawat
// inap (TglKeluar == kemarin atau hari ini).
func (d *Detector) checkPostRANAP(ctx context.Context, noRM string, today time.Time, ch chan<- checkResult) {
	d.safeRun(domain.PatientTypePostRANAP, "PostRANAP", func() (any, bool, error) {
		if noRM == "" {
			return nil, false, nil
		}
		list, err := d.khanza.GetRiwayatRANAP(ctx, noRM)
		if err != nil {
			return nil, false, err
		}

		yesterday := today.AddDate(0, 0, -1)
		var hits []domain.RiwayatRANAP
		for i := range list {
			if list[i].IsKeluarSejak(yesterday) {
				hits = append(hits, list[i])
			}
		}
		if len(hits) == 0 {
			return nil, false, nil
		}
		return hits, true, nil
	}, ch)
}
