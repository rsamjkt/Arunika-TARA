package vclaim

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// FingerprintStatus adalah hasil cek status biometrik di server BPJS.
// Mirror vendor cekFinger() di DlgRegistrasiSEPPertama.java:2785.
//
// Verified=true berarti pasien sudah melakukan verifikasi biometrik
// (sidik jari After.exe ATAU sidik wajah Frista) dan record-nya sudah
// tersimpan di server BPJS untuk tanggal pelayanan tersebut. SEP boleh
// di-issue tanpa prompt biometrik tambahan.
type FingerprintStatus struct {
	Verified bool
	Message  string // raw "status" string dari BPJS — dipakai untuk debug log
}

// CekFingerprintStatus — GET /SEP/FingerPrint/Peserta/{noKartu}/TglPelayanan/{tgl}
//
// Endpoint vendor (DlgRegistrasiSEPPertama.java line 2798):
//
//	URL = link + "/SEP/FingerPrint/Peserta/" + noka + "/TglPelayanan/" + tgl
//
// Response decrypted:
//
//	{
//	  "kode": "1",                             // 1 = ada record FP
//	  "status": "Sudah FP tgl 2025-04-29 ..."  // mengandung tanggal verifikasi
//	}
//
// Vendor cek: response.kode=="1" AND response.status mengandung CURRENT_DATE
// → statusfinger=true (skip biometrik). Kita replikasi logic itu disini.
//
// Return:
//   - Verified=true  → pasien sudah verifikasi biometrik untuk tgl tsb
//   - Verified=false → belum verifikasi, frontend harus prompt modal biometrik
//   - error          → network / decrypt / API error (caller decide degradasi)
func (c *Client) CekFingerprintStatus(ctx context.Context, noKartu string, tgl time.Time) (*FingerprintStatus, error) {
	if strings.TrimSpace(noKartu) == "" {
		return nil, fmt.Errorf("cek fingerprint: noKartu wajib diisi")
	}
	tglStr := tgl.Format("2006-01-02")

	type respWire struct {
		Kode   string `json:"kode"`
		Status string `json:"status"`
	}

	path := fmt.Sprintf("/SEP/FingerPrint/Peserta/%s/TglPelayanan/%s",
		url.PathEscape(noKartu), url.PathEscape(tglStr))

	var resp respWire
	if err := c.doGet(ctx, path, &resp); err != nil {
		// 404 / "tidak ditemukan" = pasien belum pernah FP, bukan error fatal
		if strings.Contains(strings.ToLower(err.Error()), "tidak ditemukan") ||
			strings.Contains(err.Error(), "404") {
			return &FingerprintStatus{Verified: false, Message: ""}, nil
		}
		return nil, fmt.Errorf("cek fingerprint VClaim: %w", err)
	}

	verified := resp.Kode == "1" && strings.Contains(resp.Status, tglStr)
	return &FingerprintStatus{Verified: verified, Message: resp.Status}, nil
}
