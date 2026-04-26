package sep

import (
	"strings"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

// perluBiometrik menentukan apakah verifikasi sidik jari WAJIB
// untuk pasien & poli yang dipilih.
//
// Aturan BPJS:
//   - Pasien usia < 17 tahun → tidak perlu (anak-anak dikecualikan)
//   - Poli IGD / Gawat Darurat → tidak perlu (situasi emergency)
//   - Selain itu (pasien dewasa non-emergency) → wajib biometrik
//
// Reference: BPJS_INTEGRATION.md → "Fingerprint BPJS".
func perluBiometrik(peserta domain.Peserta, kdPoli string) bool {
	if computeAgeYears(peserta.TglLahir, time.Now()) < 17 {
		return false
	}
	if isIGD(kdPoli) {
		return false
	}
	return true
}

// computeAgeYears menghitung umur dalam tahun dari TglLahir
// format "2006-01-02" terhadap referenceTime.
//
// Return 0 jika TglLahir invalid — caller akan treat sebagai
// "tidak bisa kalkulasi → asumsi anak-anak → tidak perlu biometrik"
// (default aman, lebih baik miss FP daripada blokir pasien yang
// data tgl lahirnya rusak).
func computeAgeYears(tglLahir string, ref time.Time) int {
	if tglLahir == "" {
		return 0
	}
	t, err := time.Parse("2006-01-02", tglLahir)
	if err != nil {
		return 0
	}
	years := ref.Year() - t.Year()
	// Adjust kalau ulang tahun belum lewat tahun ini
	if ref.Month() < t.Month() ||
		(ref.Month() == t.Month() && ref.Day() < t.Day()) {
		years--
	}
	if years < 0 {
		return 0
	}
	return years
}

// isIGD memeriksa apakah kdPoli mengindikasikan IGD/UGD.
// Konvensi RS bervariasi — kita cover yang paling umum:
//
//	IGD, UGD, IGDK, IGD24, EMR (emergency room)
//
// Match case-insensitive dan accept prefix "IGD" / "UGD".
// RS dengan kode unik bisa di-konfigurasi via cfg.SEP.IGDPoliCodes
// di iterasi berikutnya — saat ini hardcoded prefix.
func isIGD(kdPoli string) bool {
	upper := strings.ToUpper(strings.TrimSpace(kdPoli))
	if upper == "" {
		return false
	}
	if strings.HasPrefix(upper, "IGD") || strings.HasPrefix(upper, "UGD") {
		return true
	}
	if upper == "EMR" {
		return true
	}
	return false
}
