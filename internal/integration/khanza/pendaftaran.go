package khanza

import (
	"context"
	"fmt"

	"github.com/arunika/apm-go/internal/domain"
)

type pendaftaranBody struct {
	NoRM       string `json:"no_rm"`
	KdPoli     string `json:"kd_poli"`
	KdDokter   string `json:"kd_dokter"`
	TglPeriksa string `json:"tgl_periksa"`
	JamPeriksa string `json:"jam_periksa,omitempty"`
	Penjamin   string `json:"penjamin"`
	NoSEP      string `json:"no_sep,omitempty"`
	Catatan    string `json:"catatan,omitempty"`
}

type pendaftaranWire struct {
	NoRawat    string `json:"no_rawat"`
	NoRM       string `json:"no_rm"`
	KdPoli     string `json:"kd_poli"`
	NmPoli     string `json:"nm_poli"`
	KdDokter   string `json:"kd_dokter"`
	NmDokter   string `json:"nm_dokter"`
	TglPeriksa string `json:"tgl_periksa"`
	NoUrut     int    `json:"no_urut"`
}

// CheckDuplicateRegistration — REST stub. Khanza Laravel umumnya tidak
// expose endpoint ini; flow direct-DB MySQL pakai SQL count langsung.
// Return false supaya caller flow REST tidak block (BPJS server akan
// reject duplikasi sendiri).
func (c *Client) CheckDuplicateRegistration(ctx context.Context, noRM, kdPoli, kdDokter, tglRegistrasi, kdPj string) (bool, error) {
	return false, nil
}

// CheckDoctorOnLeave — REST stub.
func (c *Client) CheckDoctorOnLeave(ctx context.Context, kdDokter, tglRegistrasi string) (bool, error) {
	return false, nil
}

// BuatPendaftaran — POST /pendaftaran
func (c *Client) BuatPendaftaran(ctx context.Context, req domain.PendaftaranRequest) (*domain.Pendaftaran, error) {
	if req.NoRM == "" || req.KdPoli == "" || req.KdDokter == "" {
		return nil, fmt.Errorf("pendaftaran request: no_rm/kd_poli/kd_dokter wajib diisi")
	}
	if req.Penjamin == "BPJS" && req.NoSEP == "" {
		return nil, fmt.Errorf("pendaftaran BPJS wajib menyertakan no_sep")
	}

	body := pendaftaranBody{
		NoRM: req.NoRM, KdPoli: req.KdPoli, KdDokter: req.KdDokter,
		TglPeriksa: req.TglPeriksa, JamPeriksa: req.JamPeriksa,
		Penjamin: req.Penjamin, NoSEP: req.NoSEP, Catatan: req.Catatan,
	}

	var resp pendaftaranWire
	if err := c.doPost(ctx, "/pendaftaran", body, &resp); err != nil {
		return nil, fmt.Errorf("buat pendaftaran: %w", err)
	}
	return &domain.Pendaftaran{
		NoRawat: resp.NoRawat, NoRM: resp.NoRM,
		KdPoli: resp.KdPoli, NmPoli: resp.NmPoli,
		KdDokter: resp.KdDokter, NmDokter: resp.NmDokter,
		TglPeriksa: resp.TglPeriksa, NoUrut: resp.NoUrut,
	}, nil
}
