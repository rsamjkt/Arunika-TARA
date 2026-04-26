package khanza

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

type jadwalDokterWire struct {
	KdDokter   string `json:"kd_dokter"`
	NmDokter   string `json:"nm_dokter"`
	KdPoli     string `json:"kd_poli"`
	NmPoli     string `json:"nm_poli"`
	Hari       string `json:"hari"`
	JamMulai   string `json:"jam_mulai"`
	JamSelesai string `json:"jam_selesai"`
	Kuota      int    `json:"kuota"`
	Sisa       int    `json:"sisa"`
	Aktif      bool   `json:"aktif"`
}

// GetJadwalDokter — GET /poli/{kdPoli}/jadwal?tgl=YYYY-MM-DD
func (c *Client) GetJadwalDokter(ctx context.Context, kdPoli string, tgl time.Time) ([]domain.JadwalDokter, error) {
	if kdPoli == "" {
		return nil, fmt.Errorf("kd_poli kosong")
	}
	var list []jadwalDokterWire
	path := fmt.Sprintf("/poli/%s/jadwal?tgl=%s",
		url.PathEscape(kdPoli), tgl.Format("2006-01-02"))
	if err := c.doGet(ctx, path, &list); err != nil {
		return nil, fmt.Errorf("get jadwal dokter %s: %w", kdPoli, err)
	}
	out := make([]domain.JadwalDokter, 0, len(list))
	for _, j := range list {
		out = append(out, domain.JadwalDokter{
			KdDokter: j.KdDokter, NmDokter: j.NmDokter,
			KdPoli: j.KdPoli, NmPoli: j.NmPoli,
			Hari:     j.Hari,
			JamMulai: j.JamMulai, JamSelesai: j.JamSelesai,
			Kuota: j.Kuota, Sisa: j.Sisa, Aktif: j.Aktif,
		})
	}
	return out, nil
}
