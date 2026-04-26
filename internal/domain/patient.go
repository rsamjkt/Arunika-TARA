package domain

import "time"

// PatientInput adalah input identitas pasien dari kiosk.
// Identifier bisa berupa NIK 16 digit, Nomor Kartu JKN 16 digit,
// atau Nomor Rekam Medis (alphanumeric) — Detector akan mendeteksi tipenya.
type PatientInput struct {
	Identifier string
}

// Peserta merepresentasikan data peserta JKN/BPJS hasil lookup VClaim.
type Peserta struct {
	NoKartu      string
	NoRM         string
	NIK          string
	Nama         string
	TglLahir     string
	StatusAktif  string // "1" = aktif (dari VClaim)
	KelasHak     string // "1", "2", "3"
	JenisPeserta string // "PBI", "PPU", "PBPU", dll
}

// IsAktif mengembalikan true jika status kepesertaan aktif menurut VClaim.
func (p *Peserta) IsAktif() bool {
	return p != nil && p.StatusAktif == "1"
}

// SuratKontrol adalah surat rencana kontrol (SKDP) dari Khanza/VClaim.
type SuratKontrol struct {
	NoSurat    string
	NoRM       string
	TglRencana string // format "2006-01-02"
	KdPoli     string
	NmPoli     string
	KdDokter   string
}

// IsTodayOrPast mengembalikan true jika TglRencana adalah hari ini atau sudah lewat
// (waktu WIB / Asia/Jakarta). Surat kontrol untuk tanggal masa depan dianggap belum
// bisa dipakai untuk membuat SEP kontrol.
func (s *SuratKontrol) IsTodayOrPast() bool {
	if s == nil || s.TglRencana == "" {
		return false
	}
	wib := wibLoc()
	rencana, err := time.ParseInLocation("2006-01-02", s.TglRencana, wib)
	if err != nil {
		return false
	}
	return !rencana.After(todayWIB(wib))
}

// Rujukan adalah surat rujukan dari FKTP (puskesmas / klinik primer).
type Rujukan struct {
	NoSurat    string
	TglRujukan string // format "2006-01-02"
	TglBerlaku string // format "2006-01-02" — batas akhir berlaku
	KdPoli     string
	KdDokter   string
	NmFaskes   string
}

// IsValid mengembalikan true jika rujukan masih dalam masa berlaku
// (TglBerlaku > hari ini WIB).
func (r *Rujukan) IsValid() bool {
	if r == nil || r.TglBerlaku == "" {
		return false
	}
	wib := wibLoc()
	berlaku, err := time.ParseInLocation("2006-01-02", r.TglBerlaku, wib)
	if err != nil {
		return false
	}
	return berlaku.After(todayWIB(wib))
}

// wibLoc mengembalikan zona waktu Asia/Jakarta. Fallback ke offset +07:00
// jika tzdata tidak tersedia (misalnya di Windows tanpa Go tzdata embed).
func wibLoc() *time.Location {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return time.FixedZone("WIB", 7*3600)
	}
	return loc
}

// todayWIB mengembalikan tengah malam hari ini di zona WIB.
func todayWIB(wib *time.Location) time.Time {
	now := time.Now().In(wib)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, wib)
}
