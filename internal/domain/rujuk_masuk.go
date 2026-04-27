package domain

// RujukMasuk adalah catatan rujukan dari FKTP/luar yang masuk ke RS,
// di-link ke kunjungan rawat (no_rawat) setelah BuatPendaftaran BPJS.
//
// Schema Khanza tabel `rujuk_masuk`:
//
//	no_rawat       PK
//	perujuk        nama faskes perujuk
//	alamat         alamat faskes perujuk
//	no_rujuk       nomor surat rujukan
//	jm_perujuk     jumlah biaya perujuk (opsional, default 0)
//	dokter_perujuk nama dokter perujuk (opsional)
//	kd_penyakit    ICD-10 diagnosa awal
//	kategori_rujuk enum '-','Bedah','Non-Bedah','Kebidanan','Anak'
//	keterangan     catatan tambahan
//	no_balasan     nomor balasan (kosong saat insert pertama)
type RujukMasuk struct {
	NoRawat       string
	Perujuk       string
	Alamat        string
	NoRujuk       string
	DokterPerujuk string
	KdPenyakit    string // ICD-10
	KategoriRujuk string // default "-"
	Keterangan    string
}
