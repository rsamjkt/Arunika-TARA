package khanza

import (
	"context"
	"fmt"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

type antrianBody struct {
	Jenis    string `json:"jenis"`
	SubJenis string `json:"sub_jenis,omitempty"`
	KdPoli   string `json:"kd_poli,omitempty"`
	NoRM     string `json:"no_rm,omitempty"`
	NoSEP    string `json:"no_sep,omitempty"`
}

type antrianWire struct {
	ID        string    `json:"id"`
	Nomor     string    `json:"nomor"`
	Jenis     string    `json:"jenis"`
	SubJenis  string    `json:"sub_jenis"`
	Prefix    string    `json:"prefix"`
	NoUrut    int       `json:"no_urut"`
	NoRM      string    `json:"no_rm"`
	NoPoli    string    `json:"no_poli"`
	CreatedAt time.Time `json:"created_at"`
}

// BuatAntrian — POST /antrian. Khanza atomic counter — tidak akan
// mengeluarkan nomor duplikat antar-kiosk.
func (c *Client) BuatAntrian(ctx context.Context, req domain.AntrianRequest) (*domain.Ticket, error) {
	if req.Jenis == "" {
		return nil, fmt.Errorf("antrian request: jenis wajib diisi")
	}
	body := antrianBody{
		Jenis: req.Jenis, SubJenis: req.SubJenis,
		KdPoli: req.KdPoli, NoRM: req.NoRM, NoSEP: req.NoSEP,
	}

	var resp antrianWire
	if err := c.doPost(ctx, "/antrian", body, &resp); err != nil {
		return nil, fmt.Errorf("buat antrian: %w", err)
	}
	return &domain.Ticket{
		ID: resp.ID, Nomor: resp.Nomor,
		Jenis: resp.Jenis, SubJenis: resp.SubJenis,
		Prefix: resp.Prefix, NoUrut: resp.NoUrut,
		NoRM: resp.NoRM, NoPoli: resp.NoPoli,
		CreatedAt: resp.CreatedAt,
	}, nil
}
