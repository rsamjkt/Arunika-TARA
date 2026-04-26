package domain

// BookingMJKN merepresentasikan hasil booking pasien lewat aplikasi
// Mobile JKN (BPJS Antrol API). Detector pakai keberadaan booking aktif
// hari ini sebagai sinyal kategori PatientTypeMJKN.
type BookingMJKN struct {
	NoBooking        string
	NoKartu          string
	KdPoli           string
	NmPoli           string
	KdDokter         string
	NmDokter         string
	Tanggal          string // "2006-01-02"
	JamPraktik       string // "08:00-12:00"
	EstimasiDilayani string // "08:30"
	NoAntrian        string // "B-INT-005"
}

// IsValidUntuk memeriksa booking masih untuk hari yang dimaksud
// (dipakai checkMJKN agar tidak salah anggap booking kemarin sebagai aktif).
func (b *BookingMJKN) IsValidUntuk(tglYYYYMMDD string) bool {
	return b != nil && b.Tanggal == tglYYYYMMDD
}
