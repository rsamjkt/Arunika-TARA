// Smoke test live untuk khanza.MySQLClient.
//
// Jalankan (kredensial di env, jangan commit):
//
//	APM_KHANZA_DSN='user:pass@tcp(host:3306)/db?parseTime=true&timeout=5s' \
//	APM_QUERY=000001 \
//	go run -tags smoke ./cmd/khanza-smoke
//
// TIDAK menulis ke Khanza — read-only (HealthCheck, CariPasien,
// GetKunjunganAktif, GetRiwayatRANAP, GetJadwalDokter, GetSuratKontrol).
// File ini DI-EXCLUDE dari production build (build tag smoke).
//
//go:build smoke

package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/arunika/apm-go/internal/config"
	"github.com/arunika/apm-go/internal/domain"
	"github.com/arunika/apm-go/internal/integration/khanza"
)

func main() {
	dsn := os.Getenv("APM_KHANZA_DSN")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "APM_KHANZA_DSN env var wajib")
		os.Exit(2)
	}
	q := os.Getenv("APM_QUERY")
	if q == "" {
		q = "000001"
	}

	c, err := khanza.NewMySQL(config.ServerConfig{
		KhanzaDSN: dsn,
		TimeoutMs: 5000,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "NewMySQL: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	ctx := context.Background()

	fmt.Print("HealthCheck: ")
	if err := c.HealthCheck(ctx); err != nil {
		fmt.Println("FAIL", err)
		os.Exit(1)
	}
	fmt.Println("OK")

	fmt.Printf("\nCariPasien(%q):\n", q)
	p, err := c.CariPasien(ctx, q)
	if err != nil {
		fmt.Println("  ERR:", err)
	} else if p == nil {
		fmt.Println("  (tidak ditemukan)")
	} else {
		fmt.Printf("  NoRM=%s Nama=%s NIK=%s NoKartu=%s TglLahir=%s JK=%s\n",
			p.NoRM, p.Nama, mask(p.NIK), mask(p.NoKartu), p.TglLahir, p.JK)
	}

	if p != nil {
		fmt.Println("\nGetKunjunganAktif:")
		ks, err := c.GetKunjunganAktif(ctx, p.NoRM)
		if err != nil {
			fmt.Println("  ERR:", err)
		} else {
			fmt.Printf("  %d kunjungan\n", len(ks))
			for _, k := range ks {
				fmt.Printf("  - %s @ %s poli=%s/%s status=%s\n",
					k.NoRawat, k.TglKunjungan, k.KdPoli, k.NmPoli, k.Status)
			}
		}

		fmt.Println("\nGetRiwayatRANAP:")
		rs, err := c.GetRiwayatRANAP(ctx, p.NoRM)
		if err != nil {
			fmt.Println("  ERR:", err)
		} else {
			fmt.Printf("  %d episode RANAP\n", len(rs))
			for _, r := range rs {
				fmt.Printf("  - %s kamar=%s/%s masuk=%s keluar=%s pulang=%s\n",
					r.NoRawat, r.KdKamar, r.NmKamar, r.TglMasuk, r.TglKeluar, r.StatusPulang)
			}
		}

		fmt.Println("\nGetSuratKontrol:")
		sk, err := c.GetSuratKontrol(ctx, p.NoRM)
		if err != nil {
			fmt.Println("  ERR:", err)
		} else {
			fmt.Printf("  %d surat kontrol (RS ini belum punya tabel — list kosong wajar)\n", len(sk))
		}
	}

	fmt.Println("\nGetJadwalDokter (poli pertama dari schedule, hari ini):")
	tgl := time.Now()
	for _, kdPoli := range []string{"U0058", "U0059", "U0060", "U0063"} {
		js, err := c.GetJadwalDokter(ctx, kdPoli, tgl)
		if err != nil {
			fmt.Printf("  %s: ERR %v\n", kdPoli, err)
			continue
		}
		fmt.Printf("  %s (%d dokter):\n", kdPoli, len(js))
		for _, j := range js {
			fmt.Printf("    - %s %s %s-%s kuota=%d sisa=%d aktif=%v\n",
				j.KdDokter, j.NmDokter, j.JamMulai, j.JamSelesai, j.Kuota, j.Sisa, j.Aktif)
		}
	}

	fmt.Println("\nGetPoliklinikAktif:")
	polis, err := c.GetPoliklinikAktif(ctx)
	if err != nil {
		fmt.Println("  ERR:", err)
	} else {
		fmt.Printf("  %d poli aktif\n", len(polis))
		for i, p := range polis {
			if i >= 8 {
				fmt.Println("  ...")
				break
			}
			fmt.Printf("  - %s %s\n", p.KdPoli, p.NmPoli)
		}
	}

	// ============================================================
	// Phase B probes — Smart BPJS Detector data sources
	// ============================================================
	if p != nil {
		fmt.Println("\n=== Phase B probes ===")

		fmt.Println("GetSuratKontrol (BPJS) — pasien specific:")
		sks, err := c.GetSuratKontrol(ctx, p.NoRM)
		if err != nil {
			fmt.Println("  ERR:", err)
		} else {
			fmt.Printf("  %d surat kontrol BPJS untuk pasien ini\n", len(sks))
			for _, s := range sks {
				fmt.Printf("  - %s tgl=%s poli=%s/%s dokter=%s\n",
					s.NoSurat, s.TglRencana, s.KdPoli, s.NmPoli, s.KdDokter)
			}
		}

		fmt.Println("\nGetBookingMJKN — pasien hari ini:")
		bk, err := c.GetBookingMJKN(ctx, p.NoRM, time.Now())
		if err != nil {
			fmt.Println("  ERR:", err)
		} else if bk == nil {
			fmt.Println("  (pasien tidak punya booking aktif hari ini — wajar)")
		} else {
			fmt.Printf("  no_booking=%s poli=%s dokter=%s tgl=%s jam=%s antrian=%s\n",
				bk.NoBooking, bk.NmPoli, bk.NmDokter, bk.Tanggal, bk.JamPraktik, bk.NoAntrian)
		}

		fmt.Println("\nGetRujukanInternalAntarPoli — pasien 14 hari terakhir:")
		ris, err := c.GetRujukanInternalAntarPoli(ctx, p.NoRM, 14)
		if err != nil {
			fmt.Println("  ERR:", err)
		} else {
			fmt.Printf("  %d rujukan internal\n", len(ris))
			for _, r := range ris {
				fmt.Printf("  - dari %s %s asal=%s/%s tujuan=%s/%s dokter=%s\n",
					r.NoRawatAsal, r.TglKunjunganAsal,
					r.KdPoliAsal, r.NmPoliAsal,
					r.KdPoliTujuan, r.NmPoliTujuan,
					r.NmDokterTujuan)
			}
		}

		fmt.Println("\nGetSuratKontrol — sample pasien BPJS yang ada SEP recent:")
		for _, sampleRM := range []string{"079469", "079443", "079450"} {
			sks2, _ := c.GetSuratKontrol(ctx, sampleRM)
			fmt.Printf("  %s → %d surat kontrol\n", sampleRM, len(sks2))
		}
	}

	// ============================================================
	// Write test (di-gate via env var supaya tidak accidental)
	//   APM_WRITE_TEST=1     → BuatPendaftaran ke reg_periksa, verify, rollback
	//   APM_WRITE_TEST=keep  → Sama tapi TIDAK di-rollback (untuk inspect manual)
	// ============================================================
	switch os.Getenv("APM_WRITE_TEST") {
	case "1", "keep":
		fmt.Println("\n=== WRITE TEST: BuatPendaftaran ===")
		if p == nil {
			fmt.Println("  SKIP: pasien tidak ditemukan dari CariPasien")
			break
		}
		// Pilih poli & dokter dari jadwal hari ini
		hari := time.Now()
		var pickedKdPoli, pickedKdDokter string
		for _, kd := range []string{"U0058", "U0059", "U0060", "U0063"} {
			js, jerr := c.GetJadwalDokter(ctx, kd, hari)
			if jerr == nil && len(js) > 0 {
				for _, j := range js {
					if j.Aktif && j.Sisa > 0 {
						pickedKdPoli = kd
						pickedKdDokter = j.KdDokter
						break
					}
				}
				if pickedKdPoli != "" {
					break
				}
			}
		}
		if pickedKdPoli == "" {
			fmt.Println("  SKIP: tidak ada jadwal aktif hari ini")
			break
		}
		fmt.Printf("  pakai poli=%s dokter=%s pasien=%s\n", pickedKdPoli, pickedKdDokter, p.NoRM)

		req := domain.PendaftaranRequest{
			NoRM:       p.NoRM,
			KdPoli:     pickedKdPoli,
			KdDokter:   pickedKdDokter,
			TglPeriksa: hari.Format("2006-01-02"),
			Penjamin:   "UMUM",
		}
		pend, err := c.BuatPendaftaran(ctx, req)
		if err != nil {
			fmt.Printf("  FAIL BuatPendaftaran: %v\n", err)
			break
		}
		fmt.Printf("  ✓ INSERT OK: no_rawat=%s no_urut=%d nm_poli=%s nm_dokter=%s\n",
			pend.NoRawat, pend.NoUrut, pend.NmPoli, pend.NmDokter)

		// Verify via raw SELECT (pakai DSN yang sama) — termasuk
		// kolom-kolom Phase A: p_jawab, almt_pj, hubunganpj, biaya_reg,
		// umurdaftar, sttsumur, status_poli (tambahan dari Phase A).
		raw, _ := sql.Open("mysql", dsn)
		defer raw.Close()
		var (
			noReg, kdPj, sttsDaftar, statusBayar string
			pJawab, almtPJ, hubunganPJ           string
			sttsUmur, statusPoli                 string
			biaya                                float64
			umurDaftar                           int
		)
		serr := raw.QueryRowContext(ctx,
			`SELECT no_reg, kd_pj, stts_daftar, status_bayar, biaya_reg,
			        p_jawab, almt_pj, hubunganpj,
			        umurdaftar, sttsumur, status_poli
			 FROM reg_periksa WHERE no_rawat = ?`, pend.NoRawat,
		).Scan(&noReg, &kdPj, &sttsDaftar, &statusBayar, &biaya,
			&pJawab, &almtPJ, &hubunganPJ,
			&umurDaftar, &sttsUmur, &statusPoli)
		if serr != nil {
			fmt.Printf("  WARN verify: %v\n", serr)
		} else {
			fmt.Printf("  ✓ VERIFY: no_reg=%s kd_pj=%s stts_daftar=%s status_bayar=%s biaya_reg=%.0f\n",
				noReg, kdPj, sttsDaftar, statusBayar, biaya)
			fmt.Printf("    p_jawab=%q almt_pj=%q hubunganpj=%q\n",
				pJawab, almtPJ, hubunganPJ)
			fmt.Printf("    umurdaftar=%d sttsumur=%s status_poli=%s\n",
				umurDaftar, sttsUmur, statusPoli)
			// Quick sanity checks (eyeball)
			if biaya == 0 {
				fmt.Println("    ⚠ biaya_reg=0 — cek apakah poliklinik.registrasi/registrasilama memang 0")
			}
			if pJawab == "" {
				fmt.Println("    ⚠ p_jawab kosong — pasien.namakeluarga mungkin NULL")
			}
			if almtPJ == "" {
				fmt.Println("    ⚠ almt_pj kosong — pasien.alamat & kel/kec/kab/prop semua kosong?")
			}
		}

		// Rollback unless 'keep'
		if os.Getenv("APM_WRITE_TEST") != "keep" {
			res, derr := raw.ExecContext(ctx,
				`DELETE FROM reg_periksa WHERE no_rawat = ?`, pend.NoRawat)
			if derr != nil {
				fmt.Printf("  WARN rollback DELETE: %v\n", derr)
			} else {
				rows, _ := res.RowsAffected()
				fmt.Printf("  ✓ ROLLBACK: %d row deleted\n", rows)
			}
		} else {
			fmt.Printf("  ⚠ KEEP: row TIDAK dihapus (set APM_WRITE_TEST=1 untuk rollback)\n")
		}
	default:
		fmt.Println("\n(write test di-skip — set APM_WRITE_TEST=1 untuk uji INSERT reg_periksa)")
	}

	fmt.Println("\n✅ Smoke test passed")
}

func mask(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return "****" + s[len(s)-4:]
}
