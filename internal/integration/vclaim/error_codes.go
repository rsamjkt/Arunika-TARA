package vclaim

import (
	"errors"
	"fmt"
	"strings"

	"github.com/arunika/apm-go/internal/domain"
)

// VClaimError adalah error yang dihasilkan saat VClaim metaData.code != "200".
// Selalu wrap sentinel error dari domain (jika ada mapping yang cocok)
// supaya service layer bisa errors.Is(err, domain.ErrXxx).
type VClaimError struct {
	Code    string // raw code dari metaData
	Message string // raw message dari metaData (Bahasa Inggris dari BPJS)
	domain  error  // sentinel domain error (nil kalau tidak ada mapping)
}

func (e *VClaimError) Error() string {
	if e.domain != nil {
		return fmt.Sprintf("VClaim %s: %s (BPJS: %q)", e.Code, e.domain.Error(), e.Message)
	}
	return fmt.Sprintf("VClaim %s: %s", e.Code, e.Message)
}

// Unwrap memungkinkan errors.Is(err, domain.ErrXxx).
func (e *VClaimError) Unwrap() error { return e.domain }

// mapErrorCode mengkonversi (code, message) dari VClaim metaData
// menjadi *VClaimError yang sudah ter-wrap sentinel domain (kalau ada).
//
// Mengembalikan nil jika code menandakan sukses ("200" atau "1").
func mapErrorCode(code, message string) error {
	switch code {
	case "200", "1":
		return nil
	}

	var de error
	switch code {
	case "-2":
		de = errPesertaTidakDitemukan
	case "-3":
		de = domain.ErrPesertaTidakAktif
	case "-5":
		de = domain.ErrRujukanExpired
	case "-6":
		de = domain.ErrDuplikasiSEP
	case "-10":
		de = domain.ErrBiometrikDiperlukan
	}

	// Heuristik fallback berbasis substring di message — VClaim
	// kadang-kadang kembalikan code numerik berbeda dengan message
	// yang sama untuk kondisi yang sama (mis. peserta tidak aktif
	// kadang dilaporkan via code 201 vs -3).
	if de == nil {
		lower := strings.ToLower(message)
		switch {
		case strings.Contains(lower, "tidak aktif"):
			de = domain.ErrPesertaTidakAktif
		case strings.Contains(lower, "tidak ditemukan") && strings.Contains(lower, "rujukan"):
			de = domain.ErrRujukanExpired
		case strings.Contains(lower, "sudah dibuat"), strings.Contains(lower, "duplikasi"):
			de = domain.ErrDuplikasiSEP
		case strings.Contains(lower, "fingerprint"), strings.Contains(lower, "sidik jari"):
			de = domain.ErrBiometrikDiperlukan
		}
	}

	return &VClaimError{Code: code, Message: message, domain: de}
}

// errPesertaTidakDitemukan adalah sentinel internal — peserta dengan
// identifier tersebut tidak ada di database BPJS. Berbeda dengan
// ErrPesertaTidakAktif (peserta ada tapi non-aktif).
var errPesertaTidakDitemukan = errors.New("peserta tidak ditemukan di database BPJS")
