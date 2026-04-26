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
type SEP struct {
	NoSEP     string
	NoKartu   string
	TglSEP    string
	KdPoli    string
	NmPoli    string
	KdDokter  string
	NmDokter  string
	CreatedAt time.Time
}
