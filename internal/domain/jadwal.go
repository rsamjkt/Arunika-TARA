package domain

// JadwalDokter adalah satu slot praktik dokter pada poli & hari tertentu.
// Dipakai screen pemilihan dokter setelah deteksi kategori pasien.
type JadwalDokter struct {
	KdDokter   string
	NmDokter   string
	KdPoli     string
	NmPoli     string
	Hari       string // "Senin", "Selasa", dst (Bahasa Indonesia)
	JamMulai   string // "08:00"
	JamSelesai string // "12:00"
	Kuota      int    // total kuota harian
	Sisa       int    // sisa kuota saat query (dapat = Kuota jika belum ada antrian)
	Aktif      bool   // false jika dokter cuti pada tgl yang ditanyakan
}

// PendaftaranRequest adalah payload untuk daftar pasien ke poli (registrasi
// rajal). Penjamin "BPJS" wajib menyertakan NoSEP yang sudah di-issue VClaim.
type PendaftaranRequest struct {
	NoRM       string
	KdPoli     string
	KdDokter   string
	TglPeriksa string // "2006-01-02"
	JamPeriksa string // "08:30" — opsional
	Penjamin   string // "BPJS" / "UMUM" / "ASURANSI"
	NoSEP      string // wajib jika Penjamin == "BPJS"
	Catatan    string
}

// Pendaftaran adalah hasil registrasi yang dibuat di Khanza.
type Pendaftaran struct {
	NoRawat    string
	NoRM       string
	KdPoli     string
	NmPoli     string
	KdDokter   string
	NmDokter   string
	TglPeriksa string
	NoUrut     int // urutan dalam poli untuk hari itu
}

// AntrianRequest adalah payload untuk request nomor antrian ke Khanza.
// Khanza atomic-counter mencegah duplikasi nomor antar-kiosk.
type AntrianRequest struct {
	Jenis    string // LOKET / POLI / UMUM
	SubJenis string // APPOINTMENT / WALKIN / FARMASI / dll
	KdPoli   string // wajib jika Jenis == POLI
	NoRM     string // opsional, jika antrian terkait pasien tertentu
	NoSEP    string // opsional, jika antrian post-SEP
}
