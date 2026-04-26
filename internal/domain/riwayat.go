package domain

// RiwayatPelayanan adalah satu entry kunjungan/pelayanan pasien dari
// VClaim (GET /RiwayatPelayanan). Dipakai detector untuk checkPostRAJAL —
// cari kunjungan rawat jalan aktif di poli berbeda.
type RiwayatPelayanan struct {
	NoSEP        string
	NoKartu      string
	TglPelayanan string // "2006-01-02"
	JnsPelayanan string // "1" = Rawat Jalan, "2" = Rawat Inap
	KdPoli       string
	NmPoli       string
	KdDokter     string
	NmDokter     string
	Diagnosa     string // ICD-10 code(s), comma-separated
	NmDiagnosa   string
}
