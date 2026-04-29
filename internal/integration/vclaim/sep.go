package vclaim

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

// Payload SEP/2.0/insert mirror vendor DlgRegistrasiSEPPertama.java:2610.
// Wrapper: { "request": { "t_sep": { ... } } }.
//
// Field finger di-DROP dari payload — vendor tidak pernah kirim token
// biometrik di payload insert. Validasi biometrik dilakukan server-side
// via /SEP/FingerPrint/Peserta/{noka}/TglPelayanan/{tgl} (lihat
// CekFingerprintStatus). App eksternal Frista/After.exe push hasil
// verifikasi ke BPJS server independent.

type sepInsertResponse struct {
	SEP struct {
		NoSEP   string `json:"noSep"`
		TglSEP  string `json:"tglSep"`
		Peserta struct {
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
	body := c.buildSEPInsertBody(req)

	var resp sepInsertResponse
	if err := c.doPost(ctx, "/SEP/2.0/insert", body, &resp); err != nil {
		return nil, fmt.Errorf("create SEP %s: %w", req.NoKartu, err)
	}
	return resp.toDomain(req), nil
}

// CreateSEPKontrol membuat SEP dari surat kontrol (SKDP).
// Endpoint: POST /SEP/2.0/kontrol/insert
func (c *Client) CreateSEPKontrol(ctx context.Context, req domain.SEPKontrolRequest) (*domain.SEP, error) {
	body := c.buildSEPKontrolBody(req)

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

// buildSEPInsertBody membangun body insert SEP sesuai vendor
// DlgRegistrasiSEPPertama.java:2610-2671. Wrapper "t_sep" mandatory.
//
// Field default ikut vendor:
//   - jnsPelayanan: "1" (Rawat Jalan) jika kosong
//   - eksekutif: "0"
//   - cob: "0"
//   - katarak: "0"
//   - lakaLantas: "0"
//   - tujuanKunj: "0"
//   - asalRujukan: "1" (FKTP) jika ada NoRujukan dan kosong
func (c *Client) buildSEPInsertBody(req domain.SEPRequest) map[string]any {
	jnsPelayanan := defaultStr(req.JnsPelayanan, "1")
	tujuanKunj := defaultStr(req.TujuanKunjungan, "0")
	asalRujukan := req.AsalRujukan
	if asalRujukan == "" && req.NoRujukan != "" {
		asalRujukan = "1"
	}

	tglKejadian := req.TglKejadian
	if req.LakaLantas == "" || req.LakaLantas == "0" {
		tglKejadian = "" // vendor strip "0000-00-00" jadi ""
	}

	user := req.User
	if user == "" {
		user = req.NoKartu
	}

	return map[string]any{
		"request": map[string]any{
			"t_sep": map[string]any{
				"noKartu":      req.NoKartu,
				"tglSep":       req.TglSEP,
				"ppkPelayanan": c.ppkPelayanan,
				"jnsPelayanan": jnsPelayanan,
				"klsRawat": map[string]any{
					"klsRawatHak":     req.KelasRawat,
					"klsRawatNaik":    "",
					"pembiayaan":      "",
					"penanggungJawab": "",
				},
				"noMR": req.NoMR,
				"rujukan": map[string]any{
					"asalRujukan": asalRujukan,
					"tglRujukan":  req.TglRujukan,
					"noRujukan":   req.NoRujukan,
					"ppkRujukan":  req.KdPPKRujukan,
				},
				"catatan":  req.CatatanPelayanan,
				"diagAwal": req.DiagnosaAwal,
				"poli": map[string]any{
					"tujuan":    req.KdPoli,
					"eksekutif": defaultStr(req.Eksekutif, "0"),
				},
				"cob": map[string]any{
					"cob": defaultStr(req.COB, "0"),
				},
				"katarak": map[string]any{
					"katarak": defaultStr(req.Katarak, "0"),
				},
				"jaminan": map[string]any{
					"lakaLantas": defaultStr(req.LakaLantas, "0"),
					"penjamin": map[string]any{
						"tglKejadian": tglKejadian,
						"keterangan":  req.KetKecelakaan,
						"suplesi": map[string]any{
							"suplesi":      defaultStr(req.Suplesi, "0"),
							"noSepSuplesi": req.NoSepSuplesi,
							"lokasiLaka": map[string]any{
								"kdPropinsi":  req.KdPropinsi,
								"kdKabupaten": req.KdKabupaten,
								"kdKecamatan": req.KdKecamatan,
							},
						},
					},
				},
				"tujuanKunj":    tujuanKunj,
				"flagProcedure": req.FlagProcedure,
				"kdPenunjang":   req.KdPenunjang,
				"assesmentPel":  req.AsesmenPelayanan,
				"skdp": map[string]any{
					"noSurat":  req.NoSKDP,
					"kodeDPJP": req.KdDPJP,
				},
				// vendor (line 2666): dpjpLayan adalah KdDPJPLayanan (DPJP
				// terapis layanan kalau BERBEDA dari KdDPJP utama). Field
				// terpisah dari skdp.kodeDPJP. Biasanya kosong — diisi
				// hanya kalau pasien butuh terapis berbeda dari DPJP utama.
				"dpjpLayan": req.KdDPJPLayanan,
				"noTelp":    req.NoTelp,
				"user":      user,
			},
		},
	}
}

// buildSEPKontrolBody — payload untuk SEP dari SKDP existing.
// Endpoint /SEP/2.0/kontrol/insert juga butuh t_sep wrapper.
//
// Pada SEP kontrol, dpjpLayan = KdDokter karena DPJP utama sudah
// implicit dari NoSuratKontrol (BPJS server lookup SKDP record),
// sehingga field dpjpLayan dipakai untuk override dokter layanan
// hari ini (mis. kalau DPJP utama cuti, ditangani dokter pengganti).
func (c *Client) buildSEPKontrolBody(req domain.SEPKontrolRequest) map[string]any {
	jnsPelayanan := defaultStr(req.JnsPelayanan, "1")
	return map[string]any{
		"request": map[string]any{
			"t_sep": map[string]any{
				"noSuratKontrol": req.NoSuratKontrol,
				"noKartu":        req.NoKartu,
				"tglSep":         req.TglSEP,
				"ppkPelayanan":   c.ppkPelayanan,
				"jnsPelayanan":   jnsPelayanan,
				"klsRawat": map[string]any{
					"klsRawatHak": req.KelasRawat,
				},
				"dpjpLayan": req.KdDokter,
				"user":      req.NoKartu,
			},
		},
	}
}

func defaultStr(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}
