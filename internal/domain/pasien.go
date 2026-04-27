package domain

// Pasien adalah data master pasien dari Khanza (tabel pasien).
// Berbeda dengan Peserta — Peserta dari VClaim BPJS, Pasien dari SIMRS RS.
type Pasien struct {
	NoRM      string
	Nama      string
	NIK       string
	NoKartu   string // No Kartu BPJS jika ter-link
	TglLahir  string // "2006-01-02"
	JK        string // "L" / "P"
	Alamat    string
	NoTelp    string
	IhsNumber string // Satu Sehat ID jika sudah aktif
}

// PasienEnriched augments Pasien with derived fields for registrasi
// (penanggung jawab, alamat lengkap, umur). Sumber: query JOIN pasien
// + kelurahan + kecamatan + kabupaten + propinsi.
//
// Dipakai oleh MySQLClient.BuatPendaftaran untuk mengisi kolom
// reg_periksa: p_jawab (= NamaKeluarga), almt_pj (= AlamatLengkap),
// hubunganpj (= Keluarga), umurdaftar (= UmurValue), sttsumur (= SttsUmur).
type PasienEnriched struct {
	Pasien
	NamaKeluarga  string // dari pasien.namakeluarga
	Keluarga      string // dari pasien.keluarga (relasi: ISTRI/SUAMI/dll)
	AlamatLengkap string // alamat + kel + kec + kab + prop di-concat, trim empty parts
	UmurValue     int    // angka — bisa Th, Bl, atau Hr
	SttsUmur      string // "Th" / "Bl" / "Hr"
}
