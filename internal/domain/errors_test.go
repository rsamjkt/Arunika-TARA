package domain

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestSentinelErrors_PesanBahasaIndonesia(t *testing.T) {
	// Pesan error WAJIB Bahasa Indonesia (akan ditampilkan ke pasien).
	cases := []struct {
		err     error
		nama    string
		mustHas []string
	}{
		{ErrPesertaTidakAktif, "ErrPesertaTidakAktif", []string{"tidak aktif"}},
		{ErrRujukanExpired, "ErrRujukanExpired", []string{"rujukan"}},
		{ErrDuplikasiSEP, "ErrDuplikasiSEP", []string{"SEP", "sudah"}},
		{ErrBiometrikDiperlukan, "ErrBiometrikDiperlukan", []string{"sidik jari"}},
		{ErrSuratKontrolTidakDitemukan, "ErrSuratKontrolTidakDitemukan", []string{"surat kontrol"}},
		{ErrDokterCuti, "ErrDokterCuti", []string{"dokter", "cuti"}},
		{ErrKuotaPenuh, "ErrKuotaPenuh", []string{"kuota"}},
		{ErrOffline, "ErrOffline", []string{"offline"}},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			if c.err == nil {
				t.Fatal("error sentinel nil")
			}
			msg := c.err.Error()
			for _, frag := range c.mustHas {
				if !strings.Contains(strings.ToLower(msg), strings.ToLower(frag)) {
					t.Errorf("pesan %q tidak mengandung %q", msg, frag)
				}
			}
		})
	}
}

func TestSentinelErrors_BisaDiBandingkanDenganErrorsIs(t *testing.T) {
	// Wrap dengan fmt.Errorf("%w") lalu cek errors.Is — wajib true.
	wrapped := fmt.Errorf("ketika lookup peserta: %w", ErrPesertaTidakAktif)
	if !errors.Is(wrapped, ErrPesertaTidakAktif) {
		t.Errorf("errors.Is(wrapped, ErrPesertaTidakAktif) = false, want true")
	}

	// Sentinel error berbeda harus tidak match.
	if errors.Is(wrapped, ErrOffline) {
		t.Errorf("errors.Is(wrapped, ErrOffline) = true, want false")
	}
}

func TestErrJadwalKontrolBelumTiba_MenyertakanTglRencana(t *testing.T) {
	tgl := "2026-05-15"
	err := ErrJadwalKontrolBelumTiba(tgl)
	if err == nil {
		t.Fatal("ErrJadwalKontrolBelumTiba mengembalikan nil")
	}
	if !strings.Contains(err.Error(), tgl) {
		t.Errorf("pesan error %q tidak mengandung TglRencana %q", err.Error(), tgl)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "kontrol") {
		t.Errorf("pesan error %q tidak menyebut 'kontrol'", err.Error())
	}
}

func TestIsErrJadwalKontrolBelumTiba(t *testing.T) {
	jadwalErr := ErrJadwalKontrolBelumTiba("2026-05-15")
	wrapped := fmt.Errorf("validasi kontrol gagal: %w", jadwalErr)

	if !IsErrJadwalKontrolBelumTiba(jadwalErr) {
		t.Errorf("IsErrJadwalKontrolBelumTiba(jadwalErr) = false, want true")
	}
	if !IsErrJadwalKontrolBelumTiba(wrapped) {
		t.Errorf("IsErrJadwalKontrolBelumTiba(wrapped) = false, want true")
	}
	if IsErrJadwalKontrolBelumTiba(ErrPesertaTidakAktif) {
		t.Errorf("IsErrJadwalKontrolBelumTiba(ErrPesertaTidakAktif) = true, want false")
	}
	if IsErrJadwalKontrolBelumTiba(nil) {
		t.Errorf("IsErrJadwalKontrolBelumTiba(nil) = true, want false")
	}
}
