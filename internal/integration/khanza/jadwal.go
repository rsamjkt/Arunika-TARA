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

// poliklinikWire — GET /poliklinik?status=1
type poliklinikWire struct {
	KdPoli         string  `json:"kd_poli"`
	NmPoli         string  `json:"nm_poli"`
	Registrasi     float64 `json:"registrasi"`
	RegistrasiLama float64 `json:"registrasi_lama"`
	Status         string  `json:"status"`
}

// GetPoliklinikAktif — GET /poliklinik?status=1
func (c *Client) GetPoliklinikAktif(ctx context.Context) ([]domain.Poliklinik, error) {
	var list []poliklinikWire
	if err := c.doGet(ctx, "/poliklinik?status=1", &list); err != nil {
		return nil, fmt.Errorf("get poliklinik aktif: %w", err)
	}
	out := make([]domain.Poliklinik, 0, len(list))
	for _, p := range list {
		out = append(out, domain.Poliklinik{
			KdPoli: p.KdPoli, NmPoli: p.NmPoli,
			Registrasi: p.Registrasi, RegistrasiLama: p.RegistrasiLama,
			Status: p.Status,
		})
	}
	return out, nil
}

// GetBookingMJKN — REST endpoint hipotetis. Khanza Laravel umumnya
// expose /pasien/{noRM}/booking?tgl=YYYY-MM-DD. Kalau RS tidak punya
// endpoint, return nil — detector akan fallback / treat as miss.
func (c *Client) GetBookingMJKN(ctx context.Context, noRM string, tgl time.Time) (*domain.BookingMJKN, error) {
	if noRM == "" {
		return nil, nil
	}
	// Tidak ada endpoint standar Khanza untuk ini di REST mode —
	// MJKN biasanya di-handle Antrol API yang dipanggil terpisah.
	// Return nil supaya detector treat sebagai miss & lanjut ke check lain.
	return nil, nil
}

// GetRujukanInternalAntarPoli — REST stub. Khanza Laravel tidak punya
// endpoint dedicated; return nil supaya detector treat sebagai miss.
func (c *Client) GetRujukanInternalAntarPoli(ctx context.Context, noRM string, daysBack int) ([]domain.RujukanInternalPoli, error) {
	if noRM == "" {
		return nil, nil
	}
	return nil, nil
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
