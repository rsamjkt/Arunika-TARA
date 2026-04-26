package detector

import (
	"context"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

// checkKontrol memeriksa apakah ada surat kontrol (SKDP) yang
// TglRencana == today atau sudah lewat (tidak futuro).
//
// Domain.SuratKontrol.IsTodayOrPast() sudah handle WIB timezone.
func (d *Detector) checkKontrol(ctx context.Context, noRM string, today time.Time, ch chan<- checkResult) {
	d.safeRun(domain.PatientTypeKontrol, "Kontrol", func() (any, bool, error) {
		if noRM == "" {
			return nil, false, nil
		}
		list, err := d.khanza.GetSuratKontrol(ctx, noRM)
		if err != nil {
			return nil, false, err
		}
		var hits []domain.SuratKontrol
		for i := range list {
			if list[i].IsTodayOrPast() {
				hits = append(hits, list[i])
			}
		}
		if len(hits) == 0 {
			return nil, false, nil
		}
		// Return slice supaya UI bisa tampilkan multiple kalau ada,
		// tapi prioritas pakai entry pertama.
		return hits, true, nil
	}, ch)
	_ = today // diabaikan — IsTodayOrPast pakai time.Now() WIB internally
}
