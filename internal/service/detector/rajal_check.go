package detector

import (
	"context"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

// checkPostRAJAL memeriksa apakah pasien punya kunjungan rawat jalan
// aktif yang punya SKDP ke poli berbeda — sinyal pasien sedang dalam
// rangkaian kontrol antar-poli.
func (d *Detector) checkPostRAJAL(ctx context.Context, noRM string, today time.Time, ch chan<- checkResult) {
	d.safeRun(domain.PatientTypePostRAJAL, "PostRAJAL", func() (any, bool, error) {
		if noRM == "" {
			return nil, false, nil
		}
		list, err := d.khanza.GetKunjunganAktif(ctx, noRM)
		if err != nil {
			return nil, false, err
		}

		var hits []domain.Kunjungan
		for i := range list {
			if list[i].JnsPelayanan == "1" && list[i].PunyaSKDPBedaPoli() {
				hits = append(hits, list[i])
			}
		}
		if len(hits) == 0 {
			return nil, false, nil
		}
		return hits, true, nil
	}, ch)
	_ = today // tidak dipakai langsung — Khanza filter di server side
}
