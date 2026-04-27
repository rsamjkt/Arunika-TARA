package domain

// Poliklinik adalah master poli dari Khanza (tabel poliklinik).
// Status "1" = aktif, "0" = non-aktif.
type Poliklinik struct {
	KdPoli         string  `json:"kd_poli"`
	NmPoli         string  `json:"nm_poli"`
	Registrasi     float64 `json:"registrasi"`      // tarif registrasi pasien baru
	RegistrasiLama float64 `json:"registrasi_lama"` // tarif registrasi pasien lama
	Status         string  `json:"status"`          // "0" / "1"
}
