package domain

import "time"

// PatientType adalah hasil klasifikasi Smart BPJS Detector.
// Urutan iota TIDAK boleh diubah — dipakai untuk priority resolution
// di service/detector (MJKN > Kontrol > PostRANAP > PostRAJAL > RujukanBaru).
type PatientType int

const (
	PatientTypeUnknown PatientType = iota
	PatientTypeMJKN
	PatientTypeKontrol
	PatientTypePostRANAP
	PatientTypePostRAJAL
	PatientTypeRujukanBaru
	PatientTypeTidakAktif
	PatientTypeError
)

// String mengembalikan label Bahasa Indonesia untuk PatientType.
// Dipakai di logging dan UI label.
func (t PatientType) String() string {
	switch t {
	case PatientTypeMJKN:
		return "Booking Mobile JKN"
	case PatientTypeKontrol:
		return "Jadwal Kontrol"
	case PatientTypePostRANAP:
		return "Pasca Rawat Inap"
	case PatientTypePostRAJAL:
		return "Lanjutan Rawat Jalan"
	case PatientTypeRujukanBaru:
		return "Kunjungan Baru"
	case PatientTypeTidakAktif:
		return "Status Kepesertaan Tidak Aktif"
	case PatientTypeError:
		return "Gagal Mengecek Status"
	default:
		return "Tidak Diketahui"
	}
}

// DetectionResult adalah output Smart BPJS Detector untuk satu input pasien.
// Field Data berisi konteks tambahan tergantung Type:
//   - MJKN     → booking object dari Antrol API
//   - Kontrol  → *SuratKontrol
//   - Lainnya  → nil atau payload spesifik
type DetectionResult struct {
	Type       PatientType
	Peserta    *Peserta
	Data       any
	Err        error
	DetectedAt time.Time
}

// IsSuccess mengembalikan true jika deteksi menghasilkan kategori yang
// dapat dilanjutkan oleh user (bukan Error / Unknown). TidakAktif tetap
// dianggap "sukses deteksi" karena memberi info valid ke pasien.
func (r DetectionResult) IsSuccess() bool {
	if r.Err != nil {
		return false
	}
	return r.Type != PatientTypeError && r.Type != PatientTypeUnknown
}

// UserMessage mengembalikan pesan ramah Bahasa Indonesia untuk ditampilkan
// ke pasien di kiosk. Tidak mengandung istilah teknis BPJS.
func (r DetectionResult) UserMessage() string {
	switch r.Type {
	case PatientTypeMJKN:
		return "Booking Mobile JKN Anda terdeteksi. Silakan konfirmasi kedatangan untuk mencetak tiket."
	case PatientTypeKontrol:
		return "Jadwal kontrol Anda hari ini sudah siap. Silakan pilih dokter dan cetak surat layanan."
	case PatientTypePostRANAP:
		return "Anda terdaftar untuk kontrol pasca rawat inap. Silakan pilih poli tujuan."
	case PatientTypePostRAJAL:
		return "Anda terdaftar untuk kontrol lanjutan rawat jalan. Silakan pilih poli tujuan."
	case PatientTypeRujukanBaru:
		return "Silakan pilih dokter untuk kunjungan baru Anda."
	case PatientTypeTidakAktif:
		return "Status BPJS Anda saat ini tidak aktif. Silakan hubungi petugas atau daftar sebagai pasien umum."
	case PatientTypeError:
		return "Sistem tidak dapat memeriksa status Anda saat ini. Silakan hubungi petugas."
	default:
		return "Mohon coba beberapa saat lagi atau hubungi petugas."
	}
}
