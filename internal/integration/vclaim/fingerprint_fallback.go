package vclaim

import (
	"context"
	"fmt"
	"strings"
)

// FPFallbackRequest adalah payload untuk endpoint approval/pengajuan SEP
// saat FP gagal — fallback supaya pasien tetap bisa SEP tanpa biometrik
// sukses.
//
// Mirror vendor DlgRegistrasiSEPPertama.java line 2122 (aprovalSEP) +
// line 2173 (pengajuanSEP). Body keduanya identik kecuali endpoint:
//
//	{ "request": { "t_sep": {
//	    "noKartu": "...",
//	    "tglSep": "yyyy-MM-dd",
//	    "jnsPelayanan": "1" | "2",
//	    "jnsPengajuan": "2",            // hardcoded di vendor
//	    "keterangan": "...",
//	    "user": "NoRM:..." | "..."
//	} } }
type FPFallbackRequest struct {
	NoKartu      string
	TglSEP       string // "2006-01-02"
	JnsPelayanan string // "1" Rawat Jalan / "2" Rawat Inap
	Keterangan   string // alasan pengajuan
	User         string // identifier petugas/sumber pengajuan
}

// FPFallbackResponse — kontrak ringan untuk hasil approval/pengajuan.
// BPJS hanya respond OK / error message via metaData (vendor cuma cek
// code=="200" + display message).
type FPFallbackResponse struct {
	Sukses  bool
	Message string
}

// AprovalSEP — POST /Sep/aprovalSEP
//
// Vendor (line 2122): "Aproval FingerPrint karena Gagal FP melalui
// Anjungan Pasien Mandiri". Dipakai saat FP gagal di kiosk dan operator
// mau approve override agar SEP bisa di-issue.
func (c *Client) AprovalSEP(ctx context.Context, req FPFallbackRequest) (*FPFallbackResponse, error) {
	return c.doFPFallback(ctx, "/Sep/aprovalSEP", req, "Aproval FingerPrint karena Gagal FP melalui Anjungan Pasien Mandiri")
}

// PengajuanSEP — POST /Sep/pengajuanSEP
//
// Vendor (line 2173): "Pengajuan SEP Finger oleh Anjungan Mandiri".
// Dipakai untuk pengajuan resmi ke BPJS supaya SEP bisa diterbitkan
// walau biometrik tidak match (mis. pasien lansia / disabilitas).
func (c *Client) PengajuanSEP(ctx context.Context, req FPFallbackRequest) (*FPFallbackResponse, error) {
	return c.doFPFallback(ctx, "/Sep/pengajuanSEP", req, "Pengajuan SEP Finger oleh Anjungan Mandiri")
}

func (c *Client) doFPFallback(ctx context.Context, path string, req FPFallbackRequest, defaultKeterangan string) (*FPFallbackResponse, error) {
	if strings.TrimSpace(req.NoKartu) == "" {
		return nil, fmt.Errorf("fp fallback %s: noKartu wajib", path)
	}
	jnsPelayanan := defaultStr(req.JnsPelayanan, "1")
	keterangan := defaultStr(req.Keterangan, defaultKeterangan)

	body := map[string]any{
		"request": map[string]any{
			"t_sep": map[string]any{
				"noKartu":      req.NoKartu,
				"tglSep":       req.TglSEP,
				"jnsPelayanan": jnsPelayanan,
				"jnsPengajuan": "2", // vendor hardcoded
				"keterangan":   keterangan,
				"user":         defaultStr(req.User, req.NoKartu),
			},
		},
	}

	// Endpoint ini cukup peduli sukses/gagal via metaData — tidak ada
	// response decrypt yang perlu di-parse. Pass nil target ke parseEnvelope
	// (handled di common.go) dan kalau metaData != 200 akan return error
	// via mapErrorCode.
	if err := c.doPost(ctx, path, body, nil); err != nil {
		return &FPFallbackResponse{Sukses: false, Message: err.Error()}, fmt.Errorf("%s: %w", path, err)
	}
	return &FPFallbackResponse{Sukses: true, Message: "OK"}, nil
}
