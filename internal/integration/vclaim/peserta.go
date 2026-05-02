package vclaim

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

// pesertaWire adalah representasi peserta sesuai field VClaim plaintext.
// Dipisahkan dari domain.Peserta supaya wire format tidak bocor ke service.
type pesertaWire struct {
	NoKartu      string `json:"noKartu"`
	NIK          string `json:"nik"`
	Nama         string `json:"nama"`
	TglLahir     string `json:"tglLahir"`
	Sex          string `json:"sex"` // plain string: "L" / "P"
	StatusPeserta struct {
		Kode       string `json:"kode"`
		Keterangan string `json:"keterangan"`
	} `json:"statusPeserta"`
	Hak struct {
		Kode       string `json:"kode"`
		Keterangan string `json:"keterangan"`
	} `json:"hakKelas"`
	JenisPeserta struct {
		Kode       string `json:"kode"`
		Keterangan string `json:"keterangan"`
	} `json:"jenisPeserta"`
	MR struct {
		NoMR string `json:"noMR"`
	} `json:"mr"`
}

func (p *pesertaWire) toDomain() *domain.Peserta {
	return &domain.Peserta{
		NoKartu:      p.NoKartu,
		NoRM:         p.MR.NoMR,
		NIK:          p.NIK,
		Nama:         p.Nama,
		TglLahir:     p.TglLahir,
		StatusAktif:  p.StatusPeserta.Kode,
		KelasHak:     p.Hak.Kode,
		JenisPeserta: p.JenisPeserta.Keterangan,
	}
}

// pesertaResponse mengakomodasi 2 bentuk response BPJS:
//   - response.peserta = {...}    (typical untuk noKartu lookup)
//   - response.peserta = {...}    (NIK lookup juga sama)
type pesertaResponse struct {
	Peserta pesertaWire `json:"peserta"`
}

// GetPeserta — lookup cascade noKartu → NIK.
//
// 1. Coba sebagai noKartu: GET /Peserta/noKartu/{id}/tglSEP/{tgl}
// 2. Jika error "tidak ditemukan", coba sebagai NIK: GET /Peserta/nik/{id}/tglSEP/{tgl}
// 3. Sukses → return *domain.Peserta
//
// Error lainnya dari step 1 (mis. status non-aktif, network) tidak
// fallback ke NIK — itu adalah error informatif yang harus diteruskan.
func (c *Client) GetPeserta(ctx context.Context, identifier string, tgl time.Time) (*domain.Peserta, error) {
	tglStr := urlDate(tgl)

	// Cascade #1: noKartu
	p, err := c.lookupPesertaByNoKartu(ctx, identifier, tglStr)
	if err == nil {
		return p, nil
	}

	// Hanya cascade jika error spesifik "tidak ditemukan".
	// Status non-aktif (-3) atau error lain → jangan retry sebagai NIK.
	if !errors.Is(err, errPesertaTidakDitemukan) {
		return nil, fmt.Errorf("lookup noKartu: %w", err)
	}

	// Cascade #2: NIK
	p, err = c.lookupPesertaByNIK(ctx, identifier, tglStr)
	if err != nil {
		return nil, fmt.Errorf("lookup NIK setelah noKartu miss: %w", err)
	}
	return p, nil
}

func (c *Client) lookupPesertaByNoKartu(ctx context.Context, noKartu, tglStr string) (*domain.Peserta, error) {
	var resp pesertaResponse
	path := fmt.Sprintf("/Peserta/noKartu/%s/tglSEP/%s", noKartu, tglStr)
	if err := c.doGet(ctx, path, &resp); err != nil {
		return nil, err
	}
	return resp.Peserta.toDomain(), nil
}

func (c *Client) lookupPesertaByNIK(ctx context.Context, nik, tglStr string) (*domain.Peserta, error) {
	var resp pesertaResponse
	path := fmt.Sprintf("/Peserta/nik/%s/tglSEP/%s", nik, tglStr)
	if err := c.doGet(ctx, path, &resp); err != nil {
		return nil, err
	}
	return resp.Peserta.toDomain(), nil
}
