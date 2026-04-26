package domain

import (
	"testing"
	"time"
)

func TestPeserta_IsAktif(t *testing.T) {
	tests := []struct {
		name string
		p    *Peserta
		want bool
	}{
		{"status_1_aktif", &Peserta{StatusAktif: "1"}, true},
		{"status_0_nonaktif", &Peserta{StatusAktif: "0"}, false},
		{"status_kosong", &Peserta{}, false},
		{"status_huruf", &Peserta{StatusAktif: "AKTIF"}, false},
		{"nil_pointer_aman", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.IsAktif(); got != tt.want {
				t.Errorf("IsAktif() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSuratKontrol_IsTodayOrPast(t *testing.T) {
	wib := wibLoc()
	now := time.Now().In(wib)
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")
	weekAgo := now.AddDate(0, 0, -7).Format("2006-01-02")

	tests := []struct {
		name string
		s    *SuratKontrol
		want bool
	}{
		{"hari_ini", &SuratKontrol{TglRencana: today}, true},
		{"kemarin", &SuratKontrol{TglRencana: yesterday}, true},
		{"seminggu_lalu", &SuratKontrol{TglRencana: weekAgo}, true},
		{"besok_belum_tiba", &SuratKontrol{TglRencana: tomorrow}, false},
		{"format_invalid", &SuratKontrol{TglRencana: "invalid-date"}, false},
		{"tgl_kosong", &SuratKontrol{}, false},
		{"nil_pointer_aman", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s.IsTodayOrPast(); got != tt.want {
				t.Errorf("IsTodayOrPast() for %q = %v, want %v",
					stringOrEmpty(tt.s), got, tt.want)
			}
		})
	}
}

func TestRujukan_IsValid(t *testing.T) {
	wib := wibLoc()
	now := time.Now().In(wib)
	today := now.Format("2006-01-02")
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")
	nextMonth := now.AddDate(0, 1, 0).Format("2006-01-02")

	tests := []struct {
		name string
		r    *Rujukan
		want bool
	}{
		{"berlaku_besok", &Rujukan{TglBerlaku: tomorrow}, true},
		{"berlaku_bulan_depan", &Rujukan{TglBerlaku: nextMonth}, true},
		{"berlaku_hari_ini_strictly_greater", &Rujukan{TglBerlaku: today}, false},
		{"sudah_expired_kemarin", &Rujukan{TglBerlaku: yesterday}, false},
		{"format_invalid", &Rujukan{TglBerlaku: "31-12-2026"}, false},
		{"kosong", &Rujukan{}, false},
		{"nil_pointer_aman", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.IsValid(); got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func stringOrEmpty(s *SuratKontrol) string {
	if s == nil {
		return "<nil>"
	}
	return s.TglRencana
}
