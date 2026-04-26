package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestPatientType_String(t *testing.T) {
	tests := []struct {
		t    PatientType
		want string
	}{
		{PatientTypeUnknown, "Tidak Diketahui"},
		{PatientTypeMJKN, "Booking Mobile JKN"},
		{PatientTypeKontrol, "Jadwal Kontrol"},
		{PatientTypePostRANAP, "Pasca Rawat Inap"},
		{PatientTypePostRAJAL, "Lanjutan Rawat Jalan"},
		{PatientTypeRujukanBaru, "Kunjungan Baru"},
		{PatientTypeTidakAktif, "Status Kepesertaan Tidak Aktif"},
		{PatientTypeError, "Gagal Mengecek Status"},
		{PatientType(99), "Tidak Diketahui"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.t.String(); got != tt.want {
				t.Errorf("PatientType(%d).String() = %q, want %q", tt.t, got, tt.want)
			}
		})
	}
}

func TestPatientType_PriorityOrder(t *testing.T) {
	// Urutan iota dipakai untuk priority resolution di detector.
	// MJKN < Kontrol < PostRANAP < PostRAJAL < RujukanBaru (nilai numerik
	// lebih kecil = prioritas lebih tinggi karena loop priority memilihnya dulu).
	if PatientTypeMJKN >= PatientTypeKontrol {
		t.Errorf("MJKN harus prioritas lebih tinggi dari Kontrol")
	}
	if PatientTypeKontrol >= PatientTypePostRANAP {
		t.Errorf("Kontrol harus prioritas lebih tinggi dari PostRANAP")
	}
	if PatientTypePostRANAP >= PatientTypePostRAJAL {
		t.Errorf("PostRANAP harus prioritas lebih tinggi dari PostRAJAL")
	}
	if PatientTypePostRAJAL >= PatientTypeRujukanBaru {
		t.Errorf("PostRAJAL harus prioritas lebih tinggi dari RujukanBaru")
	}
}

func TestDetectionResult_IsSuccess(t *testing.T) {
	tests := []struct {
		name string
		r    DetectionResult
		want bool
	}{
		{"mjkn_sukses", DetectionResult{Type: PatientTypeMJKN}, true},
		{"kontrol_sukses", DetectionResult{Type: PatientTypeKontrol}, true},
		{"rujukan_baru_sukses", DetectionResult{Type: PatientTypeRujukanBaru}, true},
		{"tidak_aktif_tetap_sukses_deteksi", DetectionResult{Type: PatientTypeTidakAktif}, true},
		{"error_type_gagal", DetectionResult{Type: PatientTypeError}, false},
		{"unknown_type_gagal", DetectionResult{Type: PatientTypeUnknown}, false},
		{"err_field_set_gagal", DetectionResult{Type: PatientTypeMJKN, Err: errors.New("network")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.IsSuccess(); got != tt.want {
				t.Errorf("IsSuccess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectionResult_UserMessage(t *testing.T) {
	// Verifikasi: setiap PatientType punya pesan non-kosong dan tidak
	// mengandung istilah teknis BPJS yang membingungkan pasien.
	istilahTerlarang := []string{"VClaim", "PPK", "SEP", "FKTP"}

	for _, pt := range []PatientType{
		PatientTypeUnknown, PatientTypeMJKN, PatientTypeKontrol,
		PatientTypePostRANAP, PatientTypePostRAJAL, PatientTypeRujukanBaru,
		PatientTypeTidakAktif, PatientTypeError,
	} {
		t.Run(pt.String(), func(t *testing.T) {
			msg := DetectionResult{Type: pt}.UserMessage()
			if msg == "" {
				t.Errorf("UserMessage() untuk %v kosong", pt)
			}
			for _, terlarang := range istilahTerlarang {
				if strings.Contains(msg, terlarang) {
					t.Errorf("UserMessage() untuk %v mengandung istilah teknis %q: %q",
						pt, terlarang, msg)
				}
			}
		})
	}
}
