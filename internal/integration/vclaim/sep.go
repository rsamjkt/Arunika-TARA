package vclaim

import (
	"context"
	"fmt"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

type sepInsertResponse struct {
	SEP struct {
		NoSEP    string `json:"noSep"`
		TglSEP   string `json:"tglSep"`
		Peserta  struct {
			NoKartu string `json:"noKartu"`
			Nama    string `json:"nama"`
		} `json:"peserta"`
		Poli struct {
			Kode string `json:"kdPoli"`
			Nama string `json:"nmPoli"`
		} `json:"poli"`
		Dokter struct {
			Kode string `json:"kdDokter"`
			Nama string `json:"nmDokter"`
		} `json:"dpjp"`
	} `json:"sep"`
}

func (s *sepInsertResponse) toDomain(req domain.SEPRequest) *domain.SEP {
	return &domain.SEP{
		NoSEP:     s.SEP.NoSEP,
		NoKartu:   coalesce(s.SEP.Peserta.NoKartu, req.NoKartu),
		TglSEP:    coalesce(s.SEP.TglSEP, req.TglSEP),
		KdPoli:    coalesce(s.SEP.Poli.Kode, req.KdPoli),
		NmPoli:    s.SEP.Poli.Nama,
		KdDokter:  coalesce(s.SEP.Dokter.Kode, req.KdDokter),
		NmDokter:  s.SEP.Dokter.Nama,
		CreatedAt: time.Now(),
	}
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// CreateSEP membuat SEP baru dari rujukan FKTP.
// Endpoint: POST /SEP/2.0/insert
func (c *Client) CreateSEP(ctx context.Context, req domain.SEPRequest) (*domain.SEP, error) {
	body := buildSEPInsertBody(req)

	var resp sepInsertResponse
	if err := c.doPost(ctx, "/SEP/2.0/insert", body, &resp); err != nil {
		return nil, fmt.Errorf("create SEP %s: %w", req.NoKartu, err)
	}
	return resp.toDomain(req), nil
}

// CreateSEPKontrol membuat SEP dari surat kontrol (SKDP).
// Endpoint: POST /SEP/2.0/kontrol/insert
func (c *Client) CreateSEPKontrol(ctx context.Context, req domain.SEPKontrolRequest) (*domain.SEP, error) {
	body := buildSEPKontrolBody(req)

	var resp sepInsertResponse
	if err := c.doPost(ctx, "/SEP/2.0/kontrol/insert", body, &resp); err != nil {
		return nil, fmt.Errorf("create SEP kontrol %s: %w", req.NoSuratKontrol, err)
	}
	out := resp.toDomain(domain.SEPRequest{
		NoKartu:    req.NoKartu,
		TglSEP:     req.TglSEP,
		KdDokter:   req.KdDokter,
		KelasRawat: req.KelasRawat,
	})
	return out, nil
}

// buildSEPInsertBody membangun body insert SEP dari domain request.
// Hanya field essensial yang di-set; sisanya ikut default BPJS.
func buildSEPInsertBody(req domain.SEPRequest) map[string]any {
	return map[string]any{
		"request": map[string]any{
			"noKartu":      req.NoKartu,
			"tglSep":       req.TglSEP,
			"jnsPelayanan": req.JnsPelayanan,
			"klsRawat": map[string]any{
				"klsRawatHak": req.KelasRawat,
			},
			"rujukan": map[string]any{
				"noRujukan": req.NoRujukan,
			},
			"catatan": req.CatatanPelayanan,
			"poli": map[string]any{
				"tujuan":    req.KdPoli,
				"eksekutif": "0",
			},
			"dpjpLayan": map[string]any{
				"kdDPJP": req.KdDokter,
			},
			"finger": req.FPToken,
		},
	}
}

func buildSEPKontrolBody(req domain.SEPKontrolRequest) map[string]any {
	return map[string]any{
		"request": map[string]any{
			"noSuratKontrol": req.NoSuratKontrol,
			"noKartu":        req.NoKartu,
			"tglSep":         req.TglSEP,
			"jnsPelayanan":   req.JnsPelayanan,
			"klsRawat": map[string]any{
				"klsRawatHak": req.KelasRawat,
			},
			"dpjpLayan": map[string]any{
				"kdDPJP": req.KdDokter,
			},
			"finger": req.FPToken,
		},
	}
}
