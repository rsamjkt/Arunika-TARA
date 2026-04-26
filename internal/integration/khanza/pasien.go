package khanza

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/arunika/apm-go/internal/domain"
)

// pasienWire adalah representasi wire dari Khanza pasien resource.
// Field tag mengikuti konvensi snake_case Laravel.
type pasienWire struct {
	NoRM      string `json:"no_rm"`
	Nama      string `json:"nama"`
	NIK       string `json:"nik"`
	NoKartu   string `json:"no_kartu"`
	TglLahir  string `json:"tgl_lahir"`
	JK        string `json:"jk"`
	Alamat    string `json:"alamat"`
	NoTelp    string `json:"no_telp"`
	IhsNumber string `json:"ihs_number"`
}

func (p *pasienWire) toDomain() *domain.Pasien {
	return &domain.Pasien{
		NoRM: p.NoRM, Nama: p.Nama, NIK: p.NIK, NoKartu: p.NoKartu,
		TglLahir: p.TglLahir, JK: p.JK, Alamat: p.Alamat,
		NoTelp: p.NoTelp, IhsNumber: p.IhsNumber,
	}
}

// CariPasien — GET /pasien?q={q}
//
// Khanza match strategy: exact NoRM > exact NIK > exact NoKartu >
// fuzzy nama. Return (nil, nil) jika tidak match (dipetakan dari 404).
func (c *Client) CariPasien(ctx context.Context, q string) (*domain.Pasien, error) {
	if strings.TrimSpace(q) == "" {
		return nil, errors.New("query pencarian pasien kosong")
	}

	var p pasienWire
	path := fmt.Sprintf("/pasien?q=%s", url.QueryEscape(q))
	err := c.doGet(ctx, path, &p)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return nil, nil
		}
		return nil, fmt.Errorf("cari pasien: %w", err)
	}
	if p.NoRM == "" {
		return nil, nil
	}
	return p.toDomain(), nil
}

// suratKontrolWire — Khanza response untuk surat kontrol.
type suratKontrolWire struct {
	NoSurat    string `json:"no_surat"`
	NoRM       string `json:"no_rm"`
	TglRencana string `json:"tgl_rencana"`
	KdPoli     string `json:"kd_poli"`
	NmPoli     string `json:"nm_poli"`
	KdDokter   string `json:"kd_dokter"`
}

// GetSuratKontrol — GET /pasien/{noRM}/surat-kontrol
func (c *Client) GetSuratKontrol(ctx context.Context, noRM string) ([]domain.SuratKontrol, error) {
	if noRM == "" {
		return nil, nil
	}
	var list []suratKontrolWire
	path := fmt.Sprintf("/pasien/%s/surat-kontrol", url.PathEscape(noRM))
	if err := c.doGet(ctx, path, &list); err != nil {
		return nil, fmt.Errorf("get surat kontrol: %w", err)
	}
	out := make([]domain.SuratKontrol, 0, len(list))
	for _, s := range list {
		out = append(out, domain.SuratKontrol{
			NoSurat: s.NoSurat, NoRM: s.NoRM,
			TglRencana: s.TglRencana,
			KdPoli:     s.KdPoli, NmPoli: s.NmPoli,
			KdDokter: s.KdDokter,
		})
	}
	return out, nil
}

// riwayatRANAPWire
type riwayatRANAPWire struct {
	NoRM         string `json:"no_rm"`
	NoRawat      string `json:"no_rawat"`
	KdKamar      string `json:"kd_kamar"`
	NmKamar      string `json:"nm_kamar"`
	TglMasuk     string `json:"tgl_masuk"`
	TglKeluar    string `json:"tgl_keluar"`
	StatusPulang string `json:"status_pulang"`
}

// GetRiwayatRANAP — GET /pasien/{noRM}/ranap
func (c *Client) GetRiwayatRANAP(ctx context.Context, noRM string) ([]domain.RiwayatRANAP, error) {
	if noRM == "" {
		return nil, nil
	}
	var list []riwayatRANAPWire
	path := fmt.Sprintf("/pasien/%s/ranap", url.PathEscape(noRM))
	if err := c.doGet(ctx, path, &list); err != nil {
		return nil, fmt.Errorf("get riwayat ranap: %w", err)
	}
	out := make([]domain.RiwayatRANAP, 0, len(list))
	for _, r := range list {
		out = append(out, domain.RiwayatRANAP{
			NoRM: r.NoRM, NoRawat: r.NoRawat,
			KdKamar: r.KdKamar, NmKamar: r.NmKamar,
			TglMasuk: r.TglMasuk, TglKeluar: r.TglKeluar,
			StatusPulang: r.StatusPulang,
		})
	}
	return out, nil
}

// kunjunganWire
type kunjunganWire struct {
	NoRM           string `json:"no_rm"`
	NoRawat        string `json:"no_rawat"`
	TglKunjungan   string `json:"tgl_kunjungan"`
	KdPoli         string `json:"kd_poli"`
	NmPoli         string `json:"nm_poli"`
	JnsPelayanan   string `json:"jns_pelayanan"`
	Status         string `json:"status"`
	NoSKDP         string `json:"no_skdp"`
	KdPoliSKDP     string `json:"kd_poli_skdp"`
	TglRencanaSKDP string `json:"tgl_rencana_skdp"`
}

// GetKunjunganAktif — GET /pasien/{noRM}/kunjungan-aktif
func (c *Client) GetKunjunganAktif(ctx context.Context, noRM string) ([]domain.Kunjungan, error) {
	if noRM == "" {
		return nil, nil
	}
	var list []kunjunganWire
	path := fmt.Sprintf("/pasien/%s/kunjungan-aktif", url.PathEscape(noRM))
	if err := c.doGet(ctx, path, &list); err != nil {
		return nil, fmt.Errorf("get kunjungan aktif: %w", err)
	}
	out := make([]domain.Kunjungan, 0, len(list))
	for _, k := range list {
		out = append(out, domain.Kunjungan{
			NoRM: k.NoRM, NoRawat: k.NoRawat,
			TglKunjungan: k.TglKunjungan,
			KdPoli:       k.KdPoli, NmPoli: k.NmPoli,
			JnsPelayanan: k.JnsPelayanan, Status: k.Status,
			NoSKDP:         k.NoSKDP,
			KdPoliSKDP:     k.KdPoliSKDP,
			TglRencanaSKDP: k.TglRencanaSKDP,
		})
	}
	return out, nil
}
