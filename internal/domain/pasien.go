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
