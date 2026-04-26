package domain

import "time"

// RiwayatRANAP adalah satu episode rawat inap pasien dari Khanza.
// Detector pakai TglKeluar untuk identifikasi pasien post-RANAP
// (kemarin atau hari ini → kategori PatientTypePostRANAP).
type RiwayatRANAP struct {
	NoRM        string
	NoRawat     string
	KdKamar     string
	NmKamar     string
	TglMasuk    string // "2006-01-02"
	TglKeluar   string // "2006-01-02" (kosong jika masih dirawat)
	StatusPulang string // "PULANG", "MENINGGAL", "PULANG_PAKSA"
}

// IsKeluarSejak mengembalikan true jika TglKeluar berada dalam
// rentang [since, today] inklusif. Dipakai detector untuk filter
// "baru keluar kemarin/hari ini".
func (r *RiwayatRANAP) IsKeluarSejak(since time.Time) bool {
	if r == nil || r.TglKeluar == "" {
		return false
	}
	tgl, err := time.Parse("2006-01-02", r.TglKeluar)
	if err != nil {
		return false
	}
	cutoff := time.Date(since.Year(), since.Month(), since.Day(), 0, 0, 0, 0, since.Location())
	return !tgl.Before(cutoff)
}

// Kunjungan adalah satu kunjungan pasien (RJ atau RI) dari Khanza
// reg_periksa. Detector pakai untuk checkPostRAJAL — cari kunjungan
// RJ aktif yang punya SKDP ke poli berbeda.
type Kunjungan struct {
	NoRM         string
	NoRawat      string
	TglKunjungan string // "2006-01-02"
	KdPoli       string
	NmPoli       string
	JnsPelayanan string // "1"=RJ, "2"=RI
	Status       string // "Belum", "Sudah", "Batal"

	// SKDP lama → poli baru (jika ada surat kontrol antar-poli)
	NoSKDP        string
	KdPoliSKDP    string
	TglRencanaSKDP string
}

// PunyaSKDPBedaPoli mengembalikan true jika kunjungan ini punya
// surat kontrol (SKDP) ke poli berbeda — sinyal pasien post-RAJAL
// yang akan kontrol di poli lain.
func (k *Kunjungan) PunyaSKDPBedaPoli() bool {
	return k != nil && k.NoSKDP != "" && k.KdPoliSKDP != "" && k.KdPoliSKDP != k.KdPoli
}
