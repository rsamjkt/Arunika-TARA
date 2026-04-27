package vclaim

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/arunika/apm-go/internal/domain"
)

// CekSEPDuplikasi — GET /SEP/{noKartu}/{tglSEP}
//
// BPJS endpoint ini return list SEP existing untuk pasien di tgl
// tertentu. Kita pakai untuk pre-flight check sebelum CreateSEP supaya
// tidak ada duplikasi server-side (yang bisa cause billing reject /
// klaim ditolak).
//
// Return:
//   - *SEP non-nil → duplikasi ada (caller harus block atau confirm)
//   - nil, nil     → aman, tidak ada duplikasi
//   - nil, err     → error API atau network
func (c *Client) CekSEPDuplikasi(ctx context.Context, noKartu, tglSEP string) (*domain.SEP, error) {
	if strings.TrimSpace(noKartu) == "" || strings.TrimSpace(tglSEP) == "" {
		return nil, fmt.Errorf("cek sep duplikasi: noKartu & tglSEP wajib")
	}

	type sepWire struct {
		NoSEP    string `json:"noSep"`
		NoKartu  string `json:"noKartu"`
		TglSEP   string `json:"tglSep"`
		KdPoli   string `json:"poli"`
		KdDokter string `json:"kdDokter"`
	}
	type respWire struct {
		List []sepWire `json:"list"`
	}

	path := fmt.Sprintf("/SEP/%s/%s",
		url.PathEscape(noKartu), url.PathEscape(tglSEP))
	var resp respWire
	if err := c.doGet(ctx, path, &resp); err != nil {
		// 404 / "Data tidak ditemukan" = aman, gak duplikasi
		if strings.Contains(err.Error(), "tidak ditemukan") ||
			strings.Contains(err.Error(), "404") ||
			strings.Contains(err.Error(), "no data") {
			return nil, nil
		}
		return nil, fmt.Errorf("cek sep duplikasi VClaim: %w", err)
	}
	if len(resp.List) == 0 {
		return nil, nil
	}
	first := resp.List[0]
	return &domain.SEP{
		NoSEP:    first.NoSEP,
		NoKartu:  first.NoKartu,
		TglSEP:   first.TglSEP,
		KdPoli:   first.KdPoli,
		KdDokter: first.KdDokter,
	}, nil
}

