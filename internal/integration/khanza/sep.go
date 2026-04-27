package khanza

import (
	"context"
	"fmt"

	"github.com/arunika/apm-go/internal/domain"
)

type sepBody struct {
	NoSEP    string `json:"no_sep"`
	NoKartu  string `json:"no_kartu"`
	TglSEP   string `json:"tgl_sep"`
	KdPoli   string `json:"kd_poli"`
	NmPoli   string `json:"nm_poli,omitempty"`
	KdDokter string `json:"kd_dokter"`
	NmDokter string `json:"nm_dokter,omitempty"`
}

// SimpanSEP — POST /sep. Audit trail SEP yang sudah di-issue VClaim.
// Endpoint hanya butuh status sukses/gagal — tidak ada response body
// yang dipakai caller.
func (c *Client) SimpanSEP(ctx context.Context, sep domain.SEP) error {
	if sep.NoSEP == "" || sep.NoKartu == "" {
		return fmt.Errorf("simpan sep: no_sep dan no_kartu wajib diisi")
	}
	body := sepBody{
		NoSEP: sep.NoSEP, NoKartu: sep.NoKartu, TglSEP: sep.TglSEP,
		KdPoli: sep.KdPoli, NmPoli: sep.NmPoli,
		KdDokter: sep.KdDokter, NmDokter: sep.NmDokter,
	}
	if err := c.doPost(ctx, "/sep", body, nil); err != nil {
		return fmt.Errorf("simpan sep %s: %w", sep.NoSEP, err)
	}
	return nil
}

// SimpanRujukMasuk — REST stub. Khanza Laravel umumnya tidak punya
// endpoint POST /rujuk-masuk dedicated; rujukan disimpan via SEP body
// langsung. Return nil supaya caller bisa lanjut.
func (c *Client) SimpanRujukMasuk(ctx context.Context, r domain.RujukMasuk) error {
	if r.NoRawat == "" {
		return fmt.Errorf("simpan rujuk masuk: no_rawat wajib")
	}
	// REST mode: rujukan biasanya di-handle via SimpanSEP body. Kita
	// no-op di sini supaya signature interface konsisten.
	return nil
}

type satusehatBody struct {
	IhsNumber string `json:"ihs_number"`
}

// UpdateSatuSehatID — PUT /pasien/{noRM}/satusehat-id
func (c *Client) UpdateSatuSehatID(ctx context.Context, noRM, ihsNumber string) error {
	if noRM == "" || ihsNumber == "" {
		return fmt.Errorf("update satusehat: no_rm dan ihs_number wajib diisi")
	}
	path := fmt.Sprintf("/pasien/%s/satusehat-id", noRM)
	if err := c.doPut(ctx, path, satusehatBody{IhsNumber: ihsNumber}, nil); err != nil {
		return fmt.Errorf("update satusehat id %s: %w", noRM, err)
	}
	return nil
}
