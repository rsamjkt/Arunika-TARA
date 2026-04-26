package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors — bisa dibandingkan dengan errors.Is.
// Pesan dalam Bahasa Indonesia (akan di-relay langsung ke pasien
// via DetectionResult.UserMessage atau dialog error).
var (
	ErrPesertaTidakAktif          = errors.New("status kepesertaan BPJS tidak aktif")
	ErrRujukanExpired             = errors.New("nomor rujukan FKTP tidak ditemukan atau sudah expired")
	ErrDuplikasiSEP               = errors.New("SEP untuk tanggal ini sudah pernah dibuat")
	ErrBiometrikDiperlukan        = errors.New("verifikasi sidik jari diperlukan untuk pasien ini")
	ErrSuratKontrolTidakDitemukan = errors.New("surat kontrol tidak ditemukan")
	ErrDokterCuti                 = errors.New("dokter sedang cuti pada tanggal yang dipilih")
	ErrKuotaPenuh                 = errors.New("kuota dokter pada hari ini sudah penuh")
	ErrOffline                    = errors.New("sistem sedang offline — antrian disimpan untuk sinkronisasi nanti")
)

// jadwalKontrolBelumTibaError adalah error parametrik yang menyertakan
// TglRencana di pesannya. Dibuat lewat ErrJadwalKontrolBelumTiba dan
// dideteksi lewat IsErrJadwalKontrolBelumTiba.
type jadwalKontrolBelumTibaError struct {
	TglRencana string
}

func (e *jadwalKontrolBelumTibaError) Error() string {
	return fmt.Sprintf("jadwal kontrol belum tiba — silakan kembali pada %s", e.TglRencana)
}

// ErrJadwalKontrolBelumTiba mengembalikan error yang menandakan tanggal
// rencana kontrol belum tiba. Pesan errornya menyertakan TglRencana
// agar pasien tahu kapan harus kembali.
func ErrJadwalKontrolBelumTiba(tglRencana string) error {
	return &jadwalKontrolBelumTibaError{TglRencana: tglRencana}
}

// IsErrJadwalKontrolBelumTiba mengecek apakah err adalah jenis
// JadwalKontrolBelumTiba (untuk pemeriksaan di service layer).
func IsErrJadwalKontrolBelumTiba(err error) bool {
	var e *jadwalKontrolBelumTibaError
	return errors.As(err, &e)
}
