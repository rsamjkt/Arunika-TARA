package detector

import (
	"context"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

// postRAJALWindowDays = window deteksi rujukan internal antar poli
// dalam berapa hari ke belakang. 14 hari biasanya cukup untuk lifecycle
// kontrol antar-poli dalam satu episode pelayanan.
const postRAJALWindowDays = 14

// checkPostRAJAL memeriksa apakah pasien sedang dalam rangkaian kontrol
// antar-poli, dengan dua sinyal:
//
//  1. Rujukan internal poli (`rujukan_internal_poli`) yang dikeluarkan
//     dokter di kunjungan recent ke poli lain dalam RS — pasien datang
//     untuk consultation lanjutan tanpa SEP baru. Sinyal PRIORITAS karena
//     spesifik dan ada arahan dokter sebelumnya.
//  2. SKDP / Surat Kontrol BPJS yang menunjuk ke poli berbeda dari
//     kunjungan terakhir (existing logic — pakai GetKunjunganAktif).
//
// Sumber 1 adalah enhancement smart detector vs Java repo referensi
// yang tidak punya auto-detect (user pilih manual).
func (d *Detector) checkPostRAJAL(ctx context.Context, noRM string, today time.Time, ch chan<- checkResult) {
	d.safeRun(domain.PatientTypePostRAJAL, "PostRAJAL", func() (any, bool, error) {
		if noRM == "" {
			return nil, false, nil
		}

		// Source 1 (preferred): rujukan internal antar-poli.
		rujukan, err := d.khanza.GetRujukanInternalAntarPoli(ctx, noRM, postRAJALWindowDays)
		if err == nil && len(rujukan) > 0 {
			return rujukan, true, nil
		}
		if err != nil {
			d.logger.Debug("detector: GetRujukanInternalAntarPoli error, fallback ke kunjungan_aktif",
				"err", err.Error())
		}

		// Source 2 (fallback): kunjungan aktif dengan SKDP beda poli.
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
	_ = today // sumber 1 pakai window const, sumber 2 pakai Khanza side filter
}
