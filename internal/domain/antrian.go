package domain

import (
	"fmt"
	"time"
)

// AntrianJenis adalah kategori utama antrian.
type AntrianJenis string

const (
	AntrianJenisLoket AntrianJenis = "LOKET"
	AntrianJenisPoli  AntrianJenis = "POLI"
	AntrianJenisUmum  AntrianJenis = "UMUM"
)

// AntrianSubJenis adalah sub-kategori antrian (untuk Loket dan Umum).
type AntrianSubJenis string

const (
	AntrianSubAppointment AntrianSubJenis = "APPOINTMENT"
	AntrianSubWalkIn      AntrianSubJenis = "WALKIN"
	AntrianSubRanapIGD    AntrianSubJenis = "RANAP_IGD"
	AntrianSubFarmasi     AntrianSubJenis = "FARMASI"
	AntrianSubCS          AntrianSubJenis = "CS"
	AntrianSubKasir       AntrianSubJenis = "KASIR"
)

// Ticket adalah satu nomor antrian yang sudah diberikan ke pasien.
// Prefix bersifat per-jenis: "A" untuk LOKET, "B" untuk POLI, "C" untuk UMUM
// (default — bisa di-override via config.AntrianConfig).
type Ticket struct {
	ID        string
	Nomor     string
	Jenis     string
	SubJenis  string
	Prefix    string
	NoUrut    int
	NoRM      string
	NoPoli    string
	CreatedAt time.Time
	PrintedAt *time.Time
}

// FormatNomor membentuk nomor antrian yang dicetak ke tiket dan ditampilkan
// di display panggilan. Aturan:
//
//	LOKET → "<Prefix>-<NNN>"               contoh: "A-035"
//	POLI  → "<Prefix>-<NoPoli>-<NNN>"      contoh: "B-DALAM-015"
//	UMUM  → "<Prefix>-<SubKode>-<NNN>"     contoh: "C-FAR-022"
//
// SubKode untuk UMUM: FARMASI→FAR, CS→CS, KASIR→KSR.
func (t *Ticket) FormatNomor() string {
	if t == nil {
		return ""
	}
	seq := fmt.Sprintf("%03d", t.NoUrut)

	if t.Jenis == string(AntrianJenisPoli) && t.NoPoli != "" {
		return fmt.Sprintf("%s-%s-%s", t.Prefix, t.NoPoli, seq)
	}

	if t.Jenis == string(AntrianJenisUmum) {
		if sub := subKodeShort(t.SubJenis); sub != "" {
			return fmt.Sprintf("%s-%s-%s", t.Prefix, sub, seq)
		}
	}

	return fmt.Sprintf("%s-%s", t.Prefix, seq)
}

func subKodeShort(sub string) string {
	switch sub {
	case string(AntrianSubFarmasi):
		return "FAR"
	case string(AntrianSubCS):
		return "CS"
	case string(AntrianSubKasir):
		return "KSR"
	default:
		return ""
	}
}
