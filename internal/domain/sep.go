package domain

import "time"

// SEPRequest adalah payload untuk membuat SEP baru lewat VClaim
// (POST /SEP/2.0/insert atau /SEP/2.0/kontrol/insert).
type SEPRequest struct {
	NoKartu          string
	TglSEP           string // format "2006-01-02"
	KdPoli           string
	KdDokter         string
	JnsPelayanan     string // "1" = Rawat Jalan, "2" = Rawat Inap
	KelasRawat       string // "1", "2", "3"
	NoRujukan        string
	CatatanPelayanan string
	FPToken          string // token sidik jari (opsional, dipakai jika perluBiometrik)
}

// SEPKontrolRequest adalah payload untuk membuat SEP dari surat kontrol
// (SKDP) yang sudah ada di VClaim. Berbeda dengan SEPRequest karena
// referensinya nomor surat kontrol, bukan nomor rujukan FKTP.
type SEPKontrolRequest struct {
	NoSuratKontrol string
	NoKartu        string
	TglSEP         string // format "2006-01-02"
	KdDokter       string
	KelasRawat     string // "1", "2", "3"
	JnsPelayanan   string // biasanya "1" untuk kontrol RJ
	FPToken        string
}

// SEP adalah Surat Eligibilitas Peserta yang berhasil di-issue oleh BPJS.
//
// Field rujukan/SKDP/DPJP optional — diisi kalau SEP berasal dari
// rujukan FKTP atau surat kontrol berikutnya (kontrol/post-RAJAL).
// Khanza bridging_sep punya ~52 kolom, tapi APM hanya populate yang
// critical untuk klaim. Sisanya default enum kosong.
type SEP struct {
	NoSEP     string
	NoKartu   string
	TglSEP    string
	KdPoli    string
	NmPoli    string
	KdDokter  string
	NmDokter  string
	CreatedAt time.Time

	// Rujukan FKTP — kalau SEP issued dari rujukan baru
	NoRujukan      string
	TglRujukan     string // "2006-01-02"
	KdPPKRujukan   string // kode faskes perujuk (PPK)
	NmPPKRujukan   string // nama faskes perujuk
	AsalRujukan    string // "1" = Faskes 1 (FKTP), "2" = Faskes 2 (RS)
	DiagnosaAwal   string // ICD-10 code (mis. "J06.9")
	NamaDiagnosa   string // deskripsi diagnosa awal
	JenisPelayanan string // "1" = Rawat Inap, "2" = Rawat Jalan (default 2)
	KelasRawat     string // "1", "2", "3" (default "3")

	// SKDP — kalau SEP issued dari surat kontrol berikutnya
	NoSKDP   string
	KdDPJP   string // kode dokter DPJP (penanggung jawab) — kode BPJS
	NmDPJP   string // nama DPJP
	NoMR     string // no rekam medis (untuk display di SEP)
	NamaPasien string // duplicate untuk display

	// PRB (Program Rujuk Balik) — kalau pasien punya program PRB
	// (mis. Diabetes Melitus, Hipertensi, Jantung). Kosong kalau tidak.
	PRBCode string
}
