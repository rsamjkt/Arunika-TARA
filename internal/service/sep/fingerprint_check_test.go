package sep

import (
	"testing"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

func TestPerluBiometrik(t *testing.T) {
	now := time.Now()
	birth18 := now.AddDate(-18, 0, 0).Format("2006-01-02")
	birth17Lewat := now.AddDate(-17, -1, 0).Format("2006-01-02") // 17 + 1 bulan
	birth16 := now.AddDate(-16, 0, 0).Format("2006-01-02")
	birth5 := now.AddDate(-5, 0, 0).Format("2006-01-02")
	birthBesokUlangTahun := now.AddDate(-17, 0, 1).Format("2006-01-02") // belum 17 sehari lagi

	tests := []struct {
		name     string
		peserta  domain.Peserta
		kdPoli   string
		want     bool
	}{
		{
			name:    "dewasa_non_IGD_wajib_biometrik",
			peserta: domain.Peserta{TglLahir: birth18},
			kdPoli:  "INT",
			want:    true,
		},
		{
			name:    "dewasa_17_lewat_sebulan_wajib",
			peserta: domain.Peserta{TglLahir: birth17Lewat},
			kdPoli:  "BDH",
			want:    true,
		},
		{
			name:    "anak_16_tahun_tidak_perlu",
			peserta: domain.Peserta{TglLahir: birth16},
			kdPoli:  "INT",
			want:    false,
		},
		{
			name:    "balita_tidak_perlu",
			peserta: domain.Peserta{TglLahir: birth5},
			kdPoli:  "ANAK",
			want:    false,
		},
		{
			name:    "ulang_tahun_17_besok_belum_dewasa",
			peserta: domain.Peserta{TglLahir: birthBesokUlangTahun},
			kdPoli:  "INT",
			want:    false,
		},
		{
			name:    "IGD_dewasa_tidak_perlu",
			peserta: domain.Peserta{TglLahir: birth18},
			kdPoli:  "IGD",
			want:    false,
		},
		{
			name:    "UGD_dewasa_tidak_perlu",
			peserta: domain.Peserta{TglLahir: birth18},
			kdPoli:  "UGD",
			want:    false,
		},
		{
			name:    "IGD24_alias_tidak_perlu",
			peserta: domain.Peserta{TglLahir: birth18},
			kdPoli:  "IGD24",
			want:    false,
		},
		{
			name:    "EMR_emergency_room_tidak_perlu",
			peserta: domain.Peserta{TglLahir: birth18},
			kdPoli:  "EMR",
			want:    false,
		},
		{
			name:    "case_insensitive_igd",
			peserta: domain.Peserta{TglLahir: birth18},
			kdPoli:  "igd",
			want:    false,
		},
		{
			name:    "tgl_lahir_invalid_default_anak",
			peserta: domain.Peserta{TglLahir: "tanggal-rusak"},
			kdPoli:  "INT",
			want:    false, // safe default — tidak blokir pasien data invalid
		},
		{
			name:    "tgl_lahir_kosong_default_anak",
			peserta: domain.Peserta{TglLahir: ""},
			kdPoli:  "INT",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := perluBiometrik(tt.peserta, tt.kdPoli); got != tt.want {
				t.Errorf("perluBiometrik(%+v, %q) = %v, want %v",
					tt.peserta, tt.kdPoli, got, tt.want)
			}
		})
	}
}

func TestComputeAgeYears(t *testing.T) {
	ref := time.Date(2026, 4, 26, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		tgl  string
		want int
	}{
		{"1980-05-15", 45}, // belum ulang tahun di 2026 (Mei > April)
		{"1980-04-26", 46}, // tepat ulang tahun
		{"1980-04-25", 46}, // sehari sebelum ulang tahun = lewat
		{"1980-04-27", 45}, // sehari sesudah → belum ultah
		{"2026-01-01", 0},  // bayi
		{"2027-01-01", 0},  // future date → 0 (clamped)
		{"", 0},
		{"invalid", 0},
	}
	for _, tt := range tests {
		t.Run(tt.tgl, func(t *testing.T) {
			if got := computeAgeYears(tt.tgl, ref); got != tt.want {
				t.Errorf("computeAgeYears(%q) = %d, want %d", tt.tgl, got, tt.want)
			}
		})
	}
}

func TestIsIGD(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"IGD", true},
		{"UGD", true},
		{"IGD24", true},
		{"IGDK", true},
		{"UGD-1", true},
		{"EMR", true},
		{"emr", true},
		{"  IGD  ", true}, // trim
		{"INT", false},
		{"BDH", false},
		{"JTG", false},
		{"", false},
		{"IGDOR", true}, // prefix match — dianggap IGD
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := isIGD(tt.in); got != tt.want {
				t.Errorf("isIGD(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
