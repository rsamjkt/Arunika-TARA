package vclaim

import (
	"context"
	"fmt"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

type kontrolItem struct {
	NoSurat    string `json:"noSuratKontrol"`
	NoRM       string `json:"noRekamMedis"`
	TglRencana string `json:"tglRencanaKontrol"`
	Poli       struct {
		Kode string `json:"kode"`
		Nama string `json:"nama"`
	} `json:"poliKontrol"`
	Dokter struct {
		Kode string `json:"kode"`
	} `json:"dokter"`
}

type kontrolListResponse struct {
	List []kontrolItem `json:"list"`
}

// GetRencanaKontrol mengembalikan list surat kontrol antara today
// dan tgl yang diberikan. Endpoint VClaim:
//
//	GET /RencanaKontrol/List/{noKartu}/{tglAwal}/{tglAkhir}
func (c *Client) GetRencanaKontrol(ctx context.Context, noKartu string, tgl time.Time) ([]domain.SuratKontrol, error) {
	today := time.Now()
	tglAwal := urlDate(today)
	tglAkhir := urlDate(tgl)
	if today.After(tgl) {
		// caller pass tgl di masa lalu — swap supaya rentang valid
		tglAwal, tglAkhir = tglAkhir, tglAwal
	}

	var resp kontrolListResponse
	path := fmt.Sprintf("/RencanaKontrol/List/%s/%s/%s", noKartu, tglAwal, tglAkhir)
	if err := c.doGet(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("rencana kontrol %s: %w", noKartu, err)
	}

	out := make([]domain.SuratKontrol, 0, len(resp.List))
	for _, k := range resp.List {
		out = append(out, domain.SuratKontrol{
			NoSurat:    k.NoSurat,
			NoRM:       k.NoRM,
			TglRencana: k.TglRencana,
			KdPoli:     k.Poli.Kode,
			NmPoli:     k.Poli.Nama,
			KdDokter:   k.Dokter.Kode,
		})
	}
	return out, nil
}
