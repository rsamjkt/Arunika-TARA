package domain

import "time"

// SEPRequest adalah payload untuk membuat SEP baru lewat VClaim
// (POST /SEP/2.0/insert atau /SEP/2.0/kontrol/insert).
// SEPRequest adalah payload untuk membuat SEP baru lewat VClaim.
// Field opsional ditambahkan untuk parity dengan vendor Java
// (DlgRegistrasiSEPPertama.java:2610-2671). Kalau caller tidak
// pasok (kosong), backend pakai default safe ("0", "0. Tidak", dll).
type SEPRequest struct {
	NoKartu          string
	TglSEP           string // format "2006-01-02"
	KdPoli           string
	KdDokter         string
	JnsPelayanan     string // "1" = Rawat Jalan, "2" = Rawat Inap (default "1")
	KelasRawat       string // "1", "2", "3"
	NoRujukan        string
	CatatanPelayanan string
	FPToken          string

	// Detail rujukan FKTP (untuk SEP dari rujukan baru)
	TglRujukan   string
	KdPPKRujukan string
	NmPPKRujukan string
	AsalRujukan  string // "1" Faskes 1 / "2" Faskes 2(RS)
	DiagnosaAwal string // ICD-10
	NamaDiagnosa string
	NoMR         string
	NoTelp       string

	// SKDP context (untuk SEP kontrol kalau panggil CreateSEP, bukan CreateSEPKontrol)
	NoSKDP string
	KdDPJP string

	// Conditional / additional flags untuk klaim BPJS
	Eksekutif        string // "0" / "1"
	COB              string // "0" / "1"
	Katarak          string // "0" / "1"
	LakaLantas       string // "0"/"1"/"2"/"3"
	TglKejadian      string // "2006-01-02"
	KetKecelakaan    string
	Suplesi          string // "0" / "1"
	NoSepSuplesi     string
	KdPropinsi       string
	NmPropinsi       string
	KdKabupaten      string
	NmKabupaten      string
	KdKecamatan      string
	NmKecamatan      string
	TujuanKunjungan  string
	FlagProcedure    string
	KdPenunjang      string
	AsesmenPelayanan string
	User             string
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

	// Laka Lantas — kalau pasien korban kecelakaan lalu lintas.
	// LakaLantas: "0" tidak, "1" laka, "2" KKL, "3" PAK
	// Kalau LakaLantas != "0", wajib isi tglKejadian + keterangan + lokasi.
	LakaLantas    string
	TglKejadian   string // "2006-01-02" — tgl kecelakaan
	KetKecelakaan string // deskripsi singkat
	KdPropinsi    string // kode propinsi lokasi kejadian (BPJS reference)
	NmPropinsi    string
	KdKabupaten   string
	NmKabupaten   string
	KdKecamatan   string
	NmKecamatan   string

	// COB (Coordination of Benefit) — pasien dengan asuransi tambahan
	// "0" tidak, "1" ada COB. Kalau "1", peserta harus klaim dual.
	COB string

	// Eksekutif (kelas naik) — "0" reguler, "1" eksekutif
	Eksekutif string

	// Tujuan kunjungan — "0" Konsultasi, "1" Tindakan, "2" Pemeriksaan, "3" Observasi
	TujuanKunjungan string

	// Asesmen pelayanan — kode internal BPJS untuk assessment level
	AsesmenPelayanan string
}

