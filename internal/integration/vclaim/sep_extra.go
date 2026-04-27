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

// BuatRencanaKontrol — POST /RencanaKontrol/insert
//
// Endpoint ini schedule SKDP baru untuk pasien post-discharge atau
// pasien yang butuh kontrol berikutnya. Request body sesuai spec
// VClaim 2.0: noSEP, kodeDokter, poliKontrol, tglRencanaKontrol, user.
//
// Response (sukses 200): noSuratKontrol baru yang caller wajib simpan
// ke bridging_surat_kontrol_bpjs supaya:
//  1. Smart Detector di-call berikutnya bisa detect kontrol
//  2. CreateSEPKontrol berikutnya bisa pakai noSurat ini
func (c *Client) BuatRencanaKontrol(
	ctx context.Context,
	req domain.RencanaKontrolRequest,
) (*domain.RencanaKontrol, error) {
	if strings.TrimSpace(req.NoSEP) == "" ||
		strings.TrimSpace(req.KodeDokter) == "" ||
		strings.TrimSpace(req.PoliKontrol) == "" ||
		strings.TrimSpace(req.TglRencanaKontrol) == "" {
		return nil, fmt.Errorf("buat rencana kontrol: field wajib (noSEP, kodeDokter, poliKontrol, tglRencanaKontrol) tidak boleh kosong")
	}
	user := req.User
	if user == "" {
		user = "kiosk-tara"
	}

	body := map[string]any{
		"request": map[string]any{
			"noSEP":             req.NoSEP,
			"kodeDokter":        req.KodeDokter,
			"poliKontrol":       req.PoliKontrol,
			"tglRencanaKontrol": req.TglRencanaKontrol,
			"user":              user,
		},
	}

	type respWire struct {
		NoSuratKontrol string `json:"noSuratKontrol"`
		NoSEP          string `json:"noSEP"`
		TglRencana     string `json:"tglRencanaKontrol"`
		PoliKontrol    string `json:"poliKontrol"`
		NmPoli         string `json:"namaPoli"`
		KdDokter       string `json:"kodeDokter"`
		NmDokter       string `json:"namaDokter"`
	}
	var resp respWire
	if err := c.doPost(ctx, "/RencanaKontrol/insert", body, &resp); err != nil {
		return nil, fmt.Errorf("buat rencana kontrol VClaim: %w", err)
	}
	if resp.NoSuratKontrol == "" {
		return nil, fmt.Errorf("buat rencana kontrol: response BPJS tidak menyertakan noSuratKontrol")
	}
	return &domain.RencanaKontrol{
		NoSuratKontrol: resp.NoSuratKontrol,
		NoSEP:          resp.NoSEP,
		TglRencana:     resp.TglRencana,
		KdPoli:         resp.PoliKontrol,
		NmPoli:         resp.NmPoli,
		KdDokter:       resp.KdDokter,
		NmDokter:       resp.NmDokter,
	}, nil
}
