package vclaim

import (
	"context"
	"fmt"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

type rujukanResponse struct {
	Rujukan struct {
		NoKunjungan  string `json:"noKunjungan"`  // = no rujukan
		TglKunjungan string `json:"tglKunjungan"` // = tgl rujukan
		PPKAsal      struct {
			Nama string `json:"nama"`
		} `json:"provPerujuk"`
		Poli struct {
			Kode string `json:"kode"`
		} `json:"poliRujukan"`
		Diagnosa struct {
			Kode string `json:"kode"`
		} `json:"diagnosa"`
	} `json:"rujukan"`
}

// ValidasiRujukan memeriksa rujukan FKTP. tgl adalah tanggal SEP yang
// akan dibuat — dipakai BPJS untuk validasi masa berlaku 3 bulan.
//
// Endpoint:
//
//	GET /Rujukan/noSurat/{noSurat}/{tglSEP}
//
// Domain Rujukan.IsValid() di sisi caller akan cek TglBerlaku.
// Karena BPJS tidak return TglBerlaku langsung, kita hitung +90 hari
// dari TglRujukan (aturan standar BPJS untuk rujukan FKTP).
func (c *Client) ValidasiRujukan(ctx context.Context, noSurat string, tgl time.Time) (*domain.Rujukan, error) {
	var resp rujukanResponse
	path := fmt.Sprintf("/Rujukan/noSurat/%s/%s", noSurat, urlDate(tgl))
	if err := c.doGet(ctx, path, &resp); err != nil {
		return nil, fmt.Errorf("validasi rujukan %s: %w", noSurat, err)
	}

	tglBerlaku := ""
	if resp.Rujukan.TglKunjungan != "" {
		if t, err := time.Parse("2006-01-02", resp.Rujukan.TglKunjungan); err == nil {
			tglBerlaku = t.AddDate(0, 0, 90).Format("2006-01-02")
		}
	}

	return &domain.Rujukan{
		NoSurat:    resp.Rujukan.NoKunjungan,
		TglRujukan: resp.Rujukan.TglKunjungan,
		TglBerlaku: tglBerlaku,
		KdPoli:     resp.Rujukan.Poli.Kode,
		NmFaskes:   resp.Rujukan.PPKAsal.Nama,
	}, nil
}
