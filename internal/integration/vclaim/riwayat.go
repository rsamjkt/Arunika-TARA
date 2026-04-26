package vclaim

import (
	"context"
	"fmt"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

type riwayatItem struct {
	NoSEP        string `json:"noSep"`
	TglPelayanan string `json:"tglSep"`
	JnsPelayanan string `json:"jnsPelayanan"`
	Poli         struct {
		Kode string `json:"kode"`
		Nama string `json:"nama"`
	} `json:"poli"`
	Dokter struct {
		Kode string `json:"kode"`
		Nama string `json:"nama"`
	} `json:"dpjp"`
	Diagnosa struct {
		Kode string `json:"kode"`
		Nama string `json:"nama"`
	} `json:"diagnosa"`
}

type riwayatResponse struct {
	NoKartu  string        `json:"noKartu"`
	Riwayat  []riwayatItem `json:"riwayat"`
	Histori  []riwayatItem `json:"histori"` // alias yang kadang dipakai
}

// GetRiwayatPelayanan mengembalikan riwayat kunjungan antara
// tglAwal dan tglAkhir (inklusif). Endpoint:
//
//	GET /RiwayatPelayanan/{noKartu}/{tglAwal}/{tglAkhir}
func (c *Client) GetRiwayatPelayanan(ctx context.Context, noKartu string, tglAwal, tglAkhir time.Time) ([]domain.RiwayatPelayanan, error) {
	if tglAwal.After(tglAkhir) {
		tglAwal, tglAkhir = tglAkhir, tglAwal
	}

	var resp riwayatResponse
	path := fmt.Sprintf("/RiwayatPelayanan/%s/%s/%s",
		noKartu, urlDate(tglAwal), urlDate(tglAkhir))
	if err := c.doGet(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("riwayat pelayanan %s: %w", noKartu, err)
	}

	src := resp.Riwayat
	if len(src) == 0 && len(resp.Histori) > 0 {
		src = resp.Histori
	}

	out := make([]domain.RiwayatPelayanan, 0, len(src))
	for _, r := range src {
		out = append(out, domain.RiwayatPelayanan{
			NoSEP:        r.NoSEP,
			NoKartu:      noKartu,
			TglPelayanan: r.TglPelayanan,
			JnsPelayanan: r.JnsPelayanan,
			KdPoli:       r.Poli.Kode,
			NmPoli:       r.Poli.Nama,
			KdDokter:     r.Dokter.Kode,
			NmDokter:     r.Dokter.Nama,
			Diagnosa:     r.Diagnosa.Kode,
			NmDiagnosa:   r.Diagnosa.Nama,
		})
	}
	return out, nil
}
