package domain

import "testing"

func TestTicket_FormatNomor(t *testing.T) {
	tests := []struct {
		name string
		t    *Ticket
		want string
	}{
		{
			name: "loket_simple",
			t:    &Ticket{Jenis: "LOKET", Prefix: "A", NoUrut: 35},
			want: "A-035",
		},
		{
			name: "loket_appointment_prefix_A",
			t:    &Ticket{Jenis: "LOKET", SubJenis: "APPOINTMENT", Prefix: "A", NoUrut: 12},
			want: "A-012",
		},
		{
			name: "loket_walkin",
			t:    &Ticket{Jenis: "LOKET", SubJenis: "WALKIN", Prefix: "A", NoUrut: 7},
			want: "A-007",
		},
		{
			name: "poli_dengan_kode_poli",
			t:    &Ticket{Jenis: "POLI", Prefix: "B", NoPoli: "DALAM", NoUrut: 15},
			want: "B-DALAM-015",
		},
		{
			name: "poli_tanpa_kode_poli_fallback_simple",
			t:    &Ticket{Jenis: "POLI", Prefix: "B", NoUrut: 9},
			want: "B-009",
		},
		{
			name: "umum_farmasi",
			t:    &Ticket{Jenis: "UMUM", SubJenis: "FARMASI", Prefix: "C", NoUrut: 22},
			want: "C-FAR-022",
		},
		{
			name: "umum_cs",
			t:    &Ticket{Jenis: "UMUM", SubJenis: "CS", Prefix: "C", NoUrut: 8},
			want: "C-CS-008",
		},
		{
			name: "umum_kasir",
			t:    &Ticket{Jenis: "UMUM", SubJenis: "KASIR", Prefix: "C", NoUrut: 100},
			want: "C-KSR-100",
		},
		{
			name: "umum_subjenis_unknown_fallback_simple",
			t:    &Ticket{Jenis: "UMUM", SubJenis: "RANDOM", Prefix: "C", NoUrut: 3},
			want: "C-003",
		},
		{
			name: "no_urut_padding_3_digit",
			t:    &Ticket{Jenis: "LOKET", Prefix: "A", NoUrut: 1},
			want: "A-001",
		},
		{
			name: "no_urut_4_digit_overflow_tetap_dipakai",
			t:    &Ticket{Jenis: "LOKET", Prefix: "A", NoUrut: 1234},
			want: "A-1234",
		},
		{
			name: "nil_ticket_aman",
			t:    nil,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.t.FormatNomor(); got != tt.want {
				t.Errorf("FormatNomor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAntrianJenis_Constants(t *testing.T) {
	// Pastikan string value tidak berubah — dipakai di SQLite migration
	// dan API contract dengan Khanza.
	expected := map[AntrianJenis]string{
		AntrianJenisLoket: "LOKET",
		AntrianJenisPoli:  "POLI",
		AntrianJenisUmum:  "UMUM",
	}
	for jenis, want := range expected {
		if string(jenis) != want {
			t.Errorf("AntrianJenis %q tidak sesuai: got %q", want, string(jenis))
		}
	}
}

func TestAntrianSubJenis_Constants(t *testing.T) {
	expected := map[AntrianSubJenis]string{
		AntrianSubAppointment: "APPOINTMENT",
		AntrianSubWalkIn:      "WALKIN",
		AntrianSubRanapIGD:    "RANAP_IGD",
		AntrianSubFarmasi:     "FARMASI",
		AntrianSubCS:          "CS",
		AntrianSubKasir:       "KASIR",
	}
	for sub, want := range expected {
		if string(sub) != want {
			t.Errorf("AntrianSubJenis %q tidak sesuai: got %q", want, string(sub))
		}
	}
}
