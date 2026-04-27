package khanza

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/arunika/apm-go/internal/config"
	"github.com/arunika/apm-go/internal/domain"
)

// MySQLClient adalah implementasi KhanzaClient yang menembak langsung
// ke MySQL Khanza (tanpa REST). Pola ini mengikuti repo referensi
// RS-INDRIATI/anjunganmandiriSEP yang juga JDBC langsung.
//
// Trade-off vs REST Client:
//   - Pro: tidak perlu Laravel API berjalan di server Khanza, query
//     bisa dituning per-RS (RS pakai schema custom — mis. tabel
//     `rencana_kontrol` bisa belum ada).
//   - Con: terikat skema fisik DB; perubahan schema Khanza akan
//     memecahkan kiosk; kredensial DB harus dilindungi (gunakan
//     KhanzaDSNEnc / master key).
//
// PHI handling: query `pasien` mengembalikan no_ktp, no_peserta,
// alamat — caller wajib pakai logger yang dibungkus PHIMaskingHandler
// (sudah di-wire di main.go saat P-051).
type MySQLClient struct {
	db        *sql.DB
	logger    *slog.Logger
	kdPjUmum  string // mis. "A03" — kode penjamin "Umum/Tunai" di RS ini
	kdPjBPJS  string // mis. "BPJ" — kode penjamin "BPJS" di RS ini
}

var _ KhanzaClient = (*MySQLClient)(nil)

// NewMySQL membuka koneksi MySQL ke Khanza dan mengembalikan client
// siap pakai. Panggil Close() saat aplikasi shutdown.
//
// DSN contoh: "user:pass@tcp(10.0.2.121:3306)/sikrsam260312?parseTime=true&loc=Local&timeout=5s"
//
// kdPjUmum & kdPjBPJS mapping per RS — kalau kosong, fallback ke
// "A03" (umum) dan "BPJ" (BPJS) yang umum di Khanza.
func NewMySQL(cfg config.ServerConfig) (*MySQLClient, error) {
	if strings.TrimSpace(cfg.KhanzaDSN) == "" {
		return nil, errors.New("khanza_dsn kosong — set [server] khanza_dsn di config atau pakai khanza.New() (REST)")
	}

	db, err := sql.Open("mysql", cfg.KhanzaDSN)
	if err != nil {
		return nil, fmt.Errorf("open mysql khanza: %w", err)
	}

	timeout := time.Duration(cfg.TimeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(4)

	pingCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql khanza: %w", wrapOfflineMySQL(err))
	}

	kdPjUmum := strings.TrimSpace(cfg.KhanzaKdPjUmum)
	if kdPjUmum == "" {
		kdPjUmum = "A03"
	}
	kdPjBPJS := strings.TrimSpace(cfg.KhanzaKdPjBPJS)
	if kdPjBPJS == "" {
		kdPjBPJS = "BPJ"
	}

	return &MySQLClient{
		db:       db,
		logger:   slog.Default(),
		kdPjUmum: kdPjUmum,
		kdPjBPJS: kdPjBPJS,
	}, nil
}

// SetLogger mengganti logger (dipakai test atau caller dengan PHIMaskingHandler).
func (c *MySQLClient) SetLogger(l *slog.Logger) {
	if l != nil {
		c.logger = l
	}
}

// Close menutup connection pool. Idempoten.
func (c *MySQLClient) Close() error {
	if c == nil || c.db == nil {
		return nil
	}
	return c.db.Close()
}

// ============================================================
// HealthCheck
// ============================================================

func (c *MySQLClient) HealthCheck(ctx context.Context) error {
	if err := c.db.PingContext(ctx); err != nil {
		return wrapOfflineMySQL(err)
	}
	return nil
}

// ============================================================
// CariPasien
// ============================================================

// CariPasien mencari satu pasien dengan strategi berurutan:
//  1. exact match no_rkm_medis (kalau q tidak terlalu panjang)
//  2. exact match no_ktp (NIK 16 digit)
//  3. exact match no_peserta (BPJS 13 digit)
//  4. fuzzy LIKE nm_pasien
//
// Return (nil, nil) kalau tidak ada match — caller pisahkan dari error.
func (c *MySQLClient) CariPasien(ctx context.Context, q string) (*domain.Pasien, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, errors.New("query pencarian pasien kosong")
	}

	// Satu query OR — biarkan MySQL planner pilih index. Kolom
	// no_rkm_medis adalah PK; no_ktp & no_peserta umumnya ter-index;
	// nm_pasien LIKE 'X%' akan pakai prefix scan kalau ada index.
	const sqlQ = `
		SELECT no_rkm_medis, nm_pasien, no_ktp, no_peserta,
		       DATE_FORMAT(tgl_lahir, '%Y-%m-%d'), jk, alamat, no_tlp
		FROM pasien
		WHERE no_rkm_medis = ?
		   OR no_ktp = ?
		   OR no_peserta = ?
		   OR nm_pasien LIKE ?
		ORDER BY
		   CASE
		     WHEN no_rkm_medis = ? THEN 0
		     WHEN no_ktp = ? THEN 1
		     WHEN no_peserta = ? THEN 2
		     ELSE 3
		   END
		LIMIT 1
	`
	likePat := q + "%"

	var p domain.Pasien
	var nik, kartu, alamat, telp, jk, tgl sql.NullString
	err := c.db.QueryRowContext(ctx, sqlQ,
		q, q, q, likePat,
		q, q, q,
	).Scan(&p.NoRM, &p.Nama, &nik, &kartu, &tgl, &jk, &alamat, &telp)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("cari pasien: %w", wrapOfflineMySQL(err))
	}
	p.NIK = nik.String
	p.NoKartu = kartu.String
	p.TglLahir = tgl.String
	p.JK = jk.String
	p.Alamat = alamat.String
	p.NoTelp = telp.String

	// IHS Number — RS ini hanya simpan flag yes/no (flagging_pasien_satusehat),
	// tidak simpan IHS string. Kosongkan; caller (Satu Sehat onboarding)
	// akan trigger aktivasi ulang kalau perlu.
	p.IhsNumber = ""

	return &p, nil
}

// ============================================================
// GetSuratKontrol
// ============================================================

// GetSuratKontrol di RS yang pakai bridging Khanza standar = SKDP BPJS
// dari `bridging_surat_kontrol_bpjs` (link via `bridging_sep.nomr` =
// no_rkm_medis). Tabel legacy `rencana_kontrol` (versi non-BPJS) belum
// pernah dipakai di RS sikrsam260312, jadi kita langsung query SKDP BPJS.
//
// Filter: tgl_rencana ≥ today-30 hari (window deteksi smart detector).
// Sisa filter time-window ada di detector layer (IsTodayOrPast, dll).
//
// Mapping kd_poli/kd_dokter: kita simpan kode RS (lookup dari
// maping_poli_bpjs / maping_dokter_dpjpvclaim) supaya UI konsisten dengan
// flow Pasien Umum yang juga pakai kode RS.
func (c *MySQLClient) GetSuratKontrol(ctx context.Context, noRM string) ([]domain.SuratKontrol, error) {
	if noRM == "" {
		return nil, nil
	}

	const sqlQ = `
		SELECT
		  bsk.no_surat,
		  bs.nomr AS no_rkm_medis,
		  DATE_FORMAT(bsk.tgl_rencana, '%Y-%m-%d') AS tgl_rencana,
		  COALESCE(mp.kd_poli_rs, bsk.kd_poli_bpjs) AS kd_poli,
		  COALESCE(p.nm_poli, bsk.nm_poli_bpjs, '') AS nm_poli,
		  COALESCE(md.kd_dokter, bsk.kd_dokter_bpjs) AS kd_dokter
		FROM bridging_surat_kontrol_bpjs bsk
		JOIN bridging_sep bs ON bsk.no_sep = bs.no_sep
		LEFT JOIN maping_poli_bpjs mp ON bsk.kd_poli_bpjs = mp.kd_poli_bpjs
		LEFT JOIN poliklinik p        ON mp.kd_poli_rs = p.kd_poli
		LEFT JOIN maping_dokter_dpjpvclaim md ON bsk.kd_dokter_bpjs = md.kd_dokter_bpjs
		WHERE bs.nomr = ?
		  AND bsk.tgl_rencana >= CURDATE() - INTERVAL 30 DAY
		ORDER BY bsk.tgl_rencana DESC
		LIMIT 20
	`
	rows, err := c.db.QueryContext(ctx, sqlQ, noRM)
	if err != nil {
		return nil, fmt.Errorf("get surat kontrol bpjs: %w", wrapOfflineMySQL(err))
	}
	defer rows.Close()

	out := make([]domain.SuratKontrol, 0, 4)
	for rows.Next() {
		var s domain.SuratKontrol
		if err := rows.Scan(&s.NoSurat, &s.NoRM, &s.TglRencana, &s.KdPoli, &s.NmPoli, &s.KdDokter); err != nil {
			return nil, fmt.Errorf("scan surat kontrol bpjs: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ============================================================
// GetBookingMJKN — fallback dari Antrol API ke direct-DB
// ============================================================

// GetBookingMJKN mencari booking pasien (online/MJKN/antrol) di
// `booking_registrasi`. Filter: no_rkm_medis + tanggal_periksa + kd_pj=BPJ
// + status='Terdaftar' (booking aktif yang belum di-cancel/expire).
//
// Konversi ke `*domain.BookingMJKN` supaya signature konsisten dengan
// Antrol HTTP client. Detector akan pakai ini sebagai fallback ketika
// Antrol API gagal/timeout.
func (c *MySQLClient) GetBookingMJKN(ctx context.Context, noRM string, tgl time.Time) (*domain.BookingMJKN, error) {
	if noRM == "" {
		return nil, nil
	}
	tglS := tgl.Format("2006-01-02")
	const sqlQ = `
		SELECT
		  br.no_rkm_medis AS no_kartu,
		  COALESCE(br.kd_poli, ''),
		  COALESCE(p.nm_poli, ''),
		  COALESCE(br.kd_dokter, ''),
		  COALESCE(d.nm_dokter, ''),
		  DATE_FORMAT(br.tanggal_periksa, '%Y-%m-%d'),
		  COALESCE(TIME_FORMAT(br.jam_booking, '%H:%i'), ''),
		  COALESCE(br.no_reg, '')
		FROM booking_registrasi br
		LEFT JOIN poliklinik p ON br.kd_poli = p.kd_poli
		LEFT JOIN dokter d     ON br.kd_dokter = d.kd_dokter
		WHERE br.no_rkm_medis = ?
		  AND br.tanggal_periksa = ?
		  AND br.kd_pj = ?
		  AND br.status = 'Terdaftar'
		ORDER BY br.tanggal_booking DESC
		LIMIT 1
	`
	var b domain.BookingMJKN
	err := c.db.QueryRowContext(ctx, sqlQ, noRM, tglS, c.kdPjBPJS).Scan(
		&b.NoKartu, &b.KdPoli, &b.NmPoli,
		&b.KdDokter, &b.NmDokter,
		&b.Tanggal, &b.JamPraktik, &b.NoAntrian,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get booking mjkn: %w", wrapOfflineMySQL(err))
	}
	// NoBooking sintetis (Khanza tidak punya field eksplisit) — pakai
	// no_rkm_medis+tanggal supaya unique untuk display.
	b.NoBooking = fmt.Sprintf("KHZ-%s-%s", noRM, tglS)
	b.EstimasiDilayani = "" // tidak ada di booking_registrasi
	return &b, nil
}

// ============================================================
// GetRujukanInternalAntarPoli — Post-RAJAL signal
// ============================================================

// GetRujukanInternalAntarPoli mengambil rujukan_internal_poli yang berasal
// dari kunjungan pasien dalam daysBack hari terakhir. Hanya kembalikan
// rujukan yang ke poli berbeda dari poli asal (skenario PostRAJAL).
//
// Schema rujukan_internal_poli minimal: (no_rawat, kd_dokter, kd_poli) PK
// kompositif. JOIN ke reg_periksa untuk dapat tgl/poli/no_rkm_medis asal.
func (c *MySQLClient) GetRujukanInternalAntarPoli(
	ctx context.Context,
	noRM string,
	daysBack int,
) ([]domain.RujukanInternalPoli, error) {
	if noRM == "" {
		return nil, nil
	}
	if daysBack <= 0 {
		daysBack = 14
	}
	const sqlQ = `
		SELECT
		  rp.no_rawat,
		  DATE_FORMAT(rp.tgl_registrasi, '%Y-%m-%d'),
		  rp.kd_poli AS kd_poli_asal,
		  COALESCE(pa.nm_poli, '') AS nm_poli_asal,
		  rip.kd_poli AS kd_poli_tujuan,
		  COALESCE(pt.nm_poli, '') AS nm_poli_tujuan,
		  rip.kd_dokter AS kd_dokter_tujuan,
		  COALESCE(d.nm_dokter, '') AS nm_dokter_tujuan
		FROM rujukan_internal_poli rip
		JOIN reg_periksa rp     ON rip.no_rawat = rp.no_rawat
		LEFT JOIN poliklinik pa ON rp.kd_poli = pa.kd_poli
		LEFT JOIN poliklinik pt ON rip.kd_poli = pt.kd_poli
		LEFT JOIN dokter d      ON rip.kd_dokter = d.kd_dokter
		WHERE rp.no_rkm_medis = ?
		  AND rp.tgl_registrasi >= CURDATE() - INTERVAL ? DAY
		  AND rip.kd_poli <> rp.kd_poli
		ORDER BY rp.tgl_registrasi DESC
		LIMIT 10
	`
	rows, err := c.db.QueryContext(ctx, sqlQ, noRM, daysBack)
	if err != nil {
		return nil, fmt.Errorf("get rujukan internal poli: %w", wrapOfflineMySQL(err))
	}
	defer rows.Close()

	out := make([]domain.RujukanInternalPoli, 0, 4)
	for rows.Next() {
		var r domain.RujukanInternalPoli
		if err := rows.Scan(
			&r.NoRawatAsal, &r.TglKunjunganAsal,
			&r.KdPoliAsal, &r.NmPoliAsal,
			&r.KdPoliTujuan, &r.NmPoliTujuan,
			&r.KdDokterTujuan, &r.NmDokterTujuan,
		); err != nil {
			return nil, fmt.Errorf("scan rujukan internal: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ============================================================
// GetRiwayatRANAP
// ============================================================

func (c *MySQLClient) GetRiwayatRANAP(ctx context.Context, noRM string) ([]domain.RiwayatRANAP, error) {
	if noRM == "" {
		return nil, nil
	}
	const sqlQ = `
		SELECT rp.no_rkm_medis,
		       ki.no_rawat,
		       ki.kd_kamar,
		       COALESCE(b.nm_bangsal, '') AS nm_kamar,
		       DATE_FORMAT(ki.tgl_masuk, '%Y-%m-%d') AS tgl_masuk,
		       COALESCE(NULLIF(DATE_FORMAT(ki.tgl_keluar, '%Y-%m-%d'),'0000-00-00'), '') AS tgl_keluar,
		       COALESCE(ki.stts_pulang, '')
		FROM kamar_inap ki
		JOIN reg_periksa rp ON ki.no_rawat = rp.no_rawat
		LEFT JOIN kamar k   ON ki.kd_kamar = k.kd_kamar
		LEFT JOIN bangsal b ON k.kd_bangsal = b.kd_bangsal
		WHERE rp.no_rkm_medis = ?
		ORDER BY ki.tgl_masuk DESC
		LIMIT 20
	`
	rows, err := c.db.QueryContext(ctx, sqlQ, noRM)
	if err != nil {
		return nil, fmt.Errorf("get riwayat ranap: %w", wrapOfflineMySQL(err))
	}
	defer rows.Close()

	out := make([]domain.RiwayatRANAP, 0, 4)
	for rows.Next() {
		var r domain.RiwayatRANAP
		if err := rows.Scan(&r.NoRM, &r.NoRawat, &r.KdKamar, &r.NmKamar,
			&r.TglMasuk, &r.TglKeluar, &r.StatusPulang); err != nil {
			return nil, fmt.Errorf("scan ranap: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ============================================================
// GetKunjunganAktif
// ============================================================

// GetKunjunganAktif mengembalikan kunjungan rajal yang belum selesai
// (stts != Sudah / Batal / Meninggal). Khanza tidak punya tabel SKDP
// terpisah di sikrsam260312 — field NoSKDP/KdPoliSKDP/TglRencanaSKDP
// dikosongkan, detector tetap berfungsi (PunyaSKDPBedaPoli return false).
func (c *MySQLClient) GetKunjunganAktif(ctx context.Context, noRM string) ([]domain.Kunjungan, error) {
	if noRM == "" {
		return nil, nil
	}
	const sqlQ = `
		SELECT rp.no_rkm_medis,
		       rp.no_rawat,
		       DATE_FORMAT(rp.tgl_registrasi, '%Y-%m-%d') AS tgl,
		       rp.kd_poli,
		       COALESCE(p.nm_poli, '') AS nm_poli,
		       CASE rp.status_lanjut WHEN 'Ralan' THEN '1' WHEN 'Ranap' THEN '2' ELSE '' END AS jns,
		       rp.stts
		FROM reg_periksa rp
		LEFT JOIN poliklinik p ON rp.kd_poli = p.kd_poli
		WHERE rp.no_rkm_medis = ?
		  AND rp.stts NOT IN ('Sudah','Batal','Meninggal')
		ORDER BY rp.tgl_registrasi DESC, rp.jam_reg DESC
		LIMIT 10
	`
	rows, err := c.db.QueryContext(ctx, sqlQ, noRM)
	if err != nil {
		return nil, fmt.Errorf("get kunjungan aktif: %w", wrapOfflineMySQL(err))
	}
	defer rows.Close()

	out := make([]domain.Kunjungan, 0, 4)
	for rows.Next() {
		var k domain.Kunjungan
		if err := rows.Scan(&k.NoRM, &k.NoRawat, &k.TglKunjungan,
			&k.KdPoli, &k.NmPoli, &k.JnsPelayanan, &k.Status); err != nil {
			return nil, fmt.Errorf("scan kunjungan: %w", err)
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// ============================================================
// GetJadwalDokter
// ============================================================

var hariMap = map[time.Weekday]string{
	time.Sunday:    "AKHAD",
	time.Monday:    "SENIN",
	time.Tuesday:   "SELASA",
	time.Wednesday: "RABU",
	time.Thursday:  "KAMIS",
	time.Friday:    "JUMAT",
	time.Saturday:  "SABTU",
}

// hariIndo mengubah weekday ke string Bahasa Indonesia (Senin/Selasa/...)
// untuk display. Berbeda dengan hariMap (yang pakai SENIN/SELASA all-caps
// untuk match enum hari_kerja di tabel jadwal).
var hariIndo = map[time.Weekday]string{
	time.Sunday:    "Minggu",
	time.Monday:    "Senin",
	time.Tuesday:   "Selasa",
	time.Wednesday: "Rabu",
	time.Thursday:  "Kamis",
	time.Friday:    "Jumat",
	time.Saturday:  "Sabtu",
}

// GetJadwalDokter mengambil jadwal dokter aktif di poli pada tgl yang
// diminta. Sisa kuota dihitung dari reg_periksa hari yang sama
// (dokter+poli) — bukan field jadwal.kuota itu sendiri.
func (c *MySQLClient) GetJadwalDokter(ctx context.Context, kdPoli string, tgl time.Time) ([]domain.JadwalDokter, error) {
	if kdPoli == "" {
		return nil, errors.New("kd_poli kosong")
	}
	hari := hariMap[tgl.Weekday()]
	hariDisplay := hariIndo[tgl.Weekday()]

	const sqlQ = `
		SELECT j.kd_dokter,
		       COALESCE(d.nm_dokter, '') AS nm_dokter,
		       j.kd_poli,
		       COALESCE(p.nm_poli, '') AS nm_poli,
		       TIME_FORMAT(j.jam_mulai,   '%H:%i') AS jam_mulai,
		       TIME_FORMAT(j.jam_selesai, '%H:%i') AS jam_selesai,
		       j.kuota,
		       (SELECT COUNT(*) FROM reg_periksa rp
		         WHERE rp.kd_dokter = j.kd_dokter
		           AND rp.kd_poli = j.kd_poli
		           AND rp.tgl_registrasi = ?
		           AND rp.stts <> 'Batal') AS terpakai,
		       COALESCE(d.status, '0') AS dstatus
		FROM jadwal j
		LEFT JOIN dokter d     ON j.kd_dokter = d.kd_dokter
		LEFT JOIN poliklinik p ON j.kd_poli = p.kd_poli
		WHERE j.kd_poli = ? AND j.hari_kerja = ?
		ORDER BY j.jam_mulai
	`
	tglS := tgl.Format("2006-01-02")
	rows, err := c.db.QueryContext(ctx, sqlQ, tglS, kdPoli, hari)
	if err != nil {
		return nil, fmt.Errorf("get jadwal dokter %s: %w", kdPoli, wrapOfflineMySQL(err))
	}
	defer rows.Close()

	out := make([]domain.JadwalDokter, 0, 4)
	for rows.Next() {
		var j domain.JadwalDokter
		var kuota, terpakai int
		var dstatus string
		if err := rows.Scan(&j.KdDokter, &j.NmDokter, &j.KdPoli, &j.NmPoli,
			&j.JamMulai, &j.JamSelesai, &kuota, &terpakai, &dstatus); err != nil {
			return nil, fmt.Errorf("scan jadwal: %w", err)
		}
		j.Hari = hariDisplay
		j.Kuota = kuota
		j.Sisa = kuota - terpakai
		if j.Sisa < 0 {
			j.Sisa = 0
		}
		j.Aktif = dstatus == "1"
		out = append(out, j)
	}
	return out, rows.Err()
}

// ============================================================
// GetPoliklinikAktif
// ============================================================

// GetPoliklinikAktif filter poli yang punya minimal 1 entry di tabel `jadwal`
// — supaya list bersih dari unit layanan (Farmasi, IGD, Lab, dll) yang
// tidak menerima registrasi rawat jalan via kiosk.
func (c *MySQLClient) GetPoliklinikAktif(ctx context.Context) ([]domain.Poliklinik, error) {
	const sqlQ = `
		SELECT DISTINCT p.kd_poli, p.nm_poli, p.registrasi, p.registrasilama, p.status
		FROM poliklinik p
		INNER JOIN jadwal j ON p.kd_poli = j.kd_poli
		WHERE p.status = '1' AND p.kd_poli <> '-'
		ORDER BY p.nm_poli
	`
	rows, err := c.db.QueryContext(ctx, sqlQ)
	if err != nil {
		return nil, fmt.Errorf("get poliklinik aktif: %w", wrapOfflineMySQL(err))
	}
	defer rows.Close()

	out := make([]domain.Poliklinik, 0, 32)
	for rows.Next() {
		var p domain.Poliklinik
		if err := rows.Scan(&p.KdPoli, &p.NmPoli, &p.Registrasi, &p.RegistrasiLama, &p.Status); err != nil {
			return nil, fmt.Errorf("scan poli: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ============================================================
// EnrichPasien — JOIN pasien + kelurahan/kecamatan/kabupaten/propinsi
// + hitung umur. Single query, hindari N+1.
//
// CATATAN: bukan bagian dari KhanzaClient interface (REST tidak punya
// tabel join setara). Dipanggil langsung lewat receiver MySQLClient
// dari BuatPendaftaran.
// ============================================================

// EnrichPasien mengambil data pasien lengkap dengan konkatensi
// alamat (kel/kec/kab/prop) dan menghitung umur per refDate.
//
// refDate format "2006-01-02"; kalau kosong, pakai CURDATE() di server.
//
// Umur:
//   - tahun > 0 → SttsUmur="Th"
//   - else bulan > 0 → SttsUmur="Bl"
//   - else hari → SttsUmur="Hr"
//
// Return (nil, nil) kalau no_rkm_medis tidak ditemukan — caller
// pisahkan dari error.
func (c *MySQLClient) EnrichPasien(ctx context.Context, noRM, refDate string) (*domain.PasienEnriched, error) {
	if strings.TrimSpace(noRM) == "" {
		return nil, errors.New("enrich pasien: no_rkm_medis kosong")
	}
	// Pakai NULLIF supaya '0000-00-00' (zero date Khanza lama) di-treat NULL.
	// COALESCE(?, CURDATE()) supaya refDate kosong fallback ke tanggal server.
	const sqlQ = `
		SELECT p.no_rkm_medis,
		       p.nm_pasien,
		       COALESCE(p.no_ktp,'')   AS no_ktp,
		       COALESCE(p.no_peserta,'') AS no_peserta,
		       COALESCE(NULLIF(DATE_FORMAT(p.tgl_lahir,'%Y-%m-%d'),'0000-00-00'),'') AS tgl_lahir,
		       COALESCE(p.jk,'')       AS jk,
		       COALESCE(p.alamat,'')   AS alamat,
		       COALESCE(p.no_tlp,'')   AS no_tlp,
		       COALESCE(p.namakeluarga,'') AS namakeluarga,
		       COALESCE(p.keluarga,'') AS keluarga,
		       COALESCE(kel.nm_kel,'') AS nm_kel,
		       COALESCE(kec.nm_kec,'') AS nm_kec,
		       COALESCE(kab.nm_kab,'') AS nm_kab,
		       COALESCE(prop.nm_prop,'') AS nm_prop,
		       IFNULL(TIMESTAMPDIFF(YEAR,  p.tgl_lahir, COALESCE(?, CURDATE())),0) AS th,
		       IFNULL(TIMESTAMPDIFF(MONTH, p.tgl_lahir, COALESCE(?, CURDATE())),0) AS bl,
		       IFNULL(TIMESTAMPDIFF(DAY,   p.tgl_lahir, COALESCE(?, CURDATE())),0) AS hr
		FROM pasien p
		LEFT JOIN kelurahan kel ON p.kd_kel  = kel.kd_kel
		LEFT JOIN kecamatan kec ON p.kd_kec  = kec.kd_kec
		LEFT JOIN kabupaten kab ON p.kd_kab  = kab.kd_kab
		LEFT JOIN propinsi  prop ON p.kd_prop = prop.kd_prop
		WHERE p.no_rkm_medis = ?
		LIMIT 1
	`
	// Normalisasi refDate: kosong → NULL (server fallback CURDATE).
	var refArg interface{}
	if strings.TrimSpace(refDate) == "" {
		refArg = nil
	} else {
		refArg = refDate
	}

	var (
		out                                   domain.PasienEnriched
		nmKel, nmKec, nmKab, nmProp           string
		th, bl, hr                            int
	)
	err := c.db.QueryRowContext(ctx, sqlQ,
		refArg, refArg, refArg, noRM,
	).Scan(
		&out.NoRM, &out.Nama, &out.NIK, &out.NoKartu,
		&out.TglLahir, &out.JK, &out.Alamat, &out.NoTelp,
		&out.NamaKeluarga, &out.Keluarga,
		&nmKel, &nmKec, &nmKab, &nmProp,
		&th, &bl, &hr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("enrich pasien: %w", wrapOfflineMySQL(err))
	}

	// Concat alamat lengkap, skip empty parts.
	parts := []string{
		strings.TrimSpace(out.Alamat),
		strings.TrimSpace(nmKel),
		strings.TrimSpace(nmKec),
		strings.TrimSpace(nmKab),
		strings.TrimSpace(nmProp),
	}
	nonEmpty := make([]string, 0, len(parts))
	for _, s := range parts {
		if s != "" {
			nonEmpty = append(nonEmpty, s)
		}
	}
	out.AlamatLengkap = strings.Join(nonEmpty, ", ")

	// Pilih satuan umur: Th > Bl > Hr (mengikuti DlgRegistrasiWalkIn.java)
	switch {
	case th > 0:
		out.UmurValue = th
		out.SttsUmur = "Th"
	case bl > 0:
		out.UmurValue = bl
		out.SttsUmur = "Bl"
	default:
		if hr < 0 {
			hr = 0
		}
		out.UmurValue = hr
		out.SttsUmur = "Hr"
	}

	return &out, nil
}

// GetTarifPoli mengembalikan tarif registrasi sesuai status pasien.
//
//   - isLama == true  → poliklinik.registrasilama
//   - isLama == false → poliklinik.registrasi (tarif baru)
//
// Return 0 (tanpa error) kalau kd_poli tidak ditemukan — biar caller
// (BuatPendaftaran) tetap bisa lanjut dengan biaya 0 daripada gagal.
func (c *MySQLClient) GetTarifPoli(ctx context.Context, kdPoli string, isLama bool) (float64, error) {
	if strings.TrimSpace(kdPoli) == "" {
		return 0, errors.New("get tarif poli: kd_poli kosong")
	}
	col := "registrasi"
	if isLama {
		col = "registrasilama"
	}
	// Kolom tidak boleh di-bind, harus di-format. Aman karena
	// hard-coded di switch di atas.
	q := fmt.Sprintf("SELECT IFNULL(%s,0) FROM poliklinik WHERE kd_poli = ? LIMIT 1", col)
	var tarif float64
	err := c.db.QueryRowContext(ctx, q, kdPoli).Scan(&tarif)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get tarif poli %s: %w", kdPoli, wrapOfflineMySQL(err))
	}
	return tarif, nil
}

// ============================================================
// BuatPendaftaran (transactional)
// ============================================================

// BuatPendaftaran INSERT ke reg_periksa dengan transaction:
//   - Generate no_reg = MAX(no_reg per kd_poli/tgl) + 1
//   - Generate no_rawat = MAX(no_rawat per tgl) + 1, format YYYY/MM/DD/NNNNNN
//   - Cek duplikasi (no_rkm_medis + kd_poli + kd_dokter + tgl + kd_pj)
//   - INSERT 19 kolom standar Khanza
//
// Map domain.Penjamin → kd_pj:
//
//	"BPJS" → c.kdPjBPJS  (default "BPJ")
//	"UMUM" / "" → c.kdPjUmum  (default "A03")
//	lainnya → as-is (asuransi punya kode masing-masing per RS)
func (c *MySQLClient) BuatPendaftaran(ctx context.Context, req domain.PendaftaranRequest) (*domain.Pendaftaran, error) {
	if req.NoRM == "" || req.KdPoli == "" || req.KdDokter == "" {
		return nil, errors.New("pendaftaran: no_rm/kd_poli/kd_dokter wajib diisi")
	}
	if req.Penjamin == "BPJS" && req.NoSEP == "" {
		return nil, errors.New("pendaftaran BPJS: no_sep wajib diisi")
	}
	if req.TglPeriksa == "" {
		req.TglPeriksa = time.Now().Format("2006-01-02")
	}
	tgl, err := time.ParseInLocation("2006-01-02", req.TglPeriksa, time.Local)
	if err != nil {
		return nil, fmt.Errorf("parse tgl_periksa %q: %w", req.TglPeriksa, err)
	}
	kdPj := c.mapPenjamin(req.Penjamin)

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", wrapOfflineMySQL(err))
	}
	defer tx.Rollback()

	// 1. Cek duplikasi
	var dup int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM reg_periksa
		 WHERE no_rkm_medis = ? AND kd_poli = ? AND kd_dokter = ?
		   AND tgl_registrasi = ? AND kd_pj = ?
		   AND stts <> 'Batal'`,
		req.NoRM, req.KdPoli, req.KdDokter, req.TglPeriksa, kdPj,
	).Scan(&dup); err != nil {
		return nil, fmt.Errorf("cek duplikasi: %w", wrapOfflineMySQL(err))
	}
	if dup > 0 {
		return nil, errors.New("pasien sudah terdaftar di poli & dokter ini hari ini")
	}

	// 2. Generate no_reg (per kd_poli/tgl)
	var maxReg sql.NullInt64
	if err := tx.QueryRowContext(ctx,
		`SELECT IFNULL(MAX(CAST(no_reg AS UNSIGNED)),0)
		 FROM reg_periksa WHERE kd_poli = ? AND tgl_registrasi = ?`,
		req.KdPoli, req.TglPeriksa,
	).Scan(&maxReg); err != nil {
		return nil, fmt.Errorf("get max no_reg: %w", wrapOfflineMySQL(err))
	}
	noReg := fmt.Sprintf("%03d", maxReg.Int64+1)

	// 3. Generate no_rawat (global per tgl). Format Khanza: YYYY/MM/DD/NNNNNN
	var maxNoRawatSuffix sql.NullInt64
	if err := tx.QueryRowContext(ctx,
		`SELECT IFNULL(MAX(CAST(SUBSTRING(no_rawat,12,6) AS UNSIGNED)),0)
		 FROM reg_periksa WHERE tgl_registrasi = ?`,
		req.TglPeriksa,
	).Scan(&maxNoRawatSuffix); err != nil {
		return nil, fmt.Errorf("get max no_rawat: %w", wrapOfflineMySQL(err))
	}
	noRawat := fmt.Sprintf("%s/%06d",
		tgl.Format("2006/01/02"), maxNoRawatSuffix.Int64+1)

	// 4. Enrich pasien — JOIN ke kelurahan/kecamatan/kabupaten/propinsi,
	//    hitung umur (Th/Bl/Hr), ambil namakeluarga & keluarga.
	enriched, err := c.EnrichPasien(ctx, req.NoRM, req.TglPeriksa)
	if err != nil {
		return nil, fmt.Errorf("enrich pasien %s: %w", req.NoRM, err)
	}
	if enriched == nil {
		return nil, fmt.Errorf("pendaftaran: pasien %s tidak ditemukan di master", req.NoRM)
	}
	umurDaftar := enriched.UmurValue
	if umurDaftar < 0 {
		umurDaftar = 0
	}
	sttsUmur := enriched.SttsUmur
	if sttsUmur == "" {
		sttsUmur = "Th"
	}

	// 5. Cek lama/baru — sudah pernah daftar sebelum ini?
	var pernah int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM reg_periksa WHERE no_rkm_medis = ?`,
		req.NoRM,
	).Scan(&pernah); err != nil {
		return nil, fmt.Errorf("cek lama/baru: %w", wrapOfflineMySQL(err))
	}
	sttsDaftar := "Lama"
	if pernah == 0 {
		sttsDaftar = "Baru"
	}

	// 5b. Cek status_poli — pasien lama secara global tetap bisa "Baru"
	//     di poli ini kalau belum pernah ke poli ini sebelumnya.
	//     Mengikuti DlgRegistrasiWalkIn.java (count per no_rkm_medis + kd_poli).
	var pernahPoli int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM reg_periksa WHERE no_rkm_medis = ? AND kd_poli = ?`,
		req.NoRM, req.KdPoli,
	).Scan(&pernahPoli); err != nil {
		return nil, fmt.Errorf("cek status_poli: %w", wrapOfflineMySQL(err))
	}
	statusPoli := "Lama"
	if pernahPoli == 0 {
		statusPoli = "Baru"
	}

	// 6. Tarif registrasi — pilih kolom registrasilama (pasien lama)
	//    atau registrasi (pasien baru).
	biayaReg, err := c.GetTarifPoli(ctx, req.KdPoli, sttsDaftar == "Lama")
	if err != nil {
		return nil, fmt.Errorf("get tarif poli %s: %w", req.KdPoli, err)
	}

	// 7. Penanggung jawab — pakai override dari req kalau di-set,
	//    selain itu derive dari master pasien.
	pjawab := strings.TrimSpace(req.PJawab)
	if pjawab == "" {
		pjawab = enriched.NamaKeluarga
	}
	almtPj := strings.TrimSpace(req.AlmtPJ)
	if almtPj == "" {
		almtPj = enriched.AlamatLengkap
	}
	hubunganPj := strings.TrimSpace(req.HubunganPJ)
	if hubunganPj == "" {
		hubunganPj = enriched.Keluarga
	}

	// 8. INSERT
	jamReg := time.Now().In(time.Local).Format("15:04:05")
	statusLanjut := "Ralan"
	stts := "Belum"
	statusBayar := "Belum Bayar"

	const insertSQL = `
		INSERT INTO reg_periksa (
		  no_reg, no_rawat, tgl_registrasi, jam_reg,
		  kd_dokter, no_rkm_medis, kd_poli,
		  p_jawab, almt_pj, hubunganpj, biaya_reg,
		  stts, stts_daftar, status_lanjut,
		  kd_pj, umurdaftar, sttsumur,
		  status_bayar, status_poli
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	`
	if _, err := tx.ExecContext(ctx, insertSQL,
		noReg, noRawat, req.TglPeriksa, jamReg,
		req.KdDokter, req.NoRM, req.KdPoli,
		pjawab, almtPj, hubunganPj, biayaReg,
		stts, sttsDaftar, statusLanjut,
		kdPj, umurDaftar, sttsUmur,
		statusBayar, statusPoli,
	); err != nil {
		return nil, fmt.Errorf("insert reg_periksa: %w", wrapOfflineMySQL(err))
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit pendaftaran: %w", wrapOfflineMySQL(err))
	}

	// Ambil nama poli & dokter untuk return value (best-effort, tidak fatal)
	var nmPoli, nmDokter string
	_ = c.db.QueryRowContext(ctx,
		"SELECT nm_poli FROM poliklinik WHERE kd_poli = ?", req.KdPoli,
	).Scan(&nmPoli)
	_ = c.db.QueryRowContext(ctx,
		"SELECT nm_dokter FROM dokter WHERE kd_dokter = ?", req.KdDokter,
	).Scan(&nmDokter)

	noUrutInt, _ := parseIntSafe(noReg)
	return &domain.Pendaftaran{
		NoRawat:    noRawat,
		NoRM:       req.NoRM,
		KdPoli:     req.KdPoli,
		NmPoli:     nmPoli,
		KdDokter:   req.KdDokter,
		NmDokter:   nmDokter,
		TglPeriksa: req.TglPeriksa,
		NoUrut:     noUrutInt,
	}, nil
}

func (c *MySQLClient) mapPenjamin(p string) string {
	switch strings.ToUpper(strings.TrimSpace(p)) {
	case "", "UMUM", "TUNAI":
		return c.kdPjUmum
	case "BPJS":
		return c.kdPjBPJS
	default:
		// Asuransi & lain-lain: pass-through. Caller diasumsikan sudah
		// pakai kode penjamin Khanza langsung.
		return strings.TrimSpace(p)
	}
}

func parseIntSafe(s string) (int, bool) {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, false
		}
		n = n*10 + int(r-'0')
	}
	return n, true
}

// ============================================================
// BuatAntrian — direct-INSERT ke antrian_loket Khanza
// ============================================================

// antrianMu serialize SELECT MAX + INSERT supaya nomor antrian tidak
// duplikat dalam single kiosk instance. Multi-kiosk: pakai prefix
// berbeda per kiosk (set di config) supaya counter terpisah.
var antrianMu sync.Mutex

// BuatAntrian INSERT ke `antrian_loket` Khanza mengikuti pattern
// vendor V3 (KhanzaHMSAnjunganSEP_RSAMXIP/DlgAmbilAntrean.java:240).
//
// Schema:
//
//	type        — kategori antrian: "Loket", "CS", "Booking", "Ranap"
//	noantrian   — string nomor (mis. "001", "A001")
//	postdate    — DATE hari ini
//	start_time  — TIME saat ambil
//	end_time    — TIME saat dipanggil/selesai (default '00:00:00')
//
// Counter strategy: SELECT MAX(CAST(noantrian AS UNSIGNED)) + 1
// per type per CURDATE() + app-level mutex. Reset harian otomatis
// karena filter CURDATE().
//
// Map domain.AntrianRequest.Jenis → type Khanza:
//
//	"LOKET" → "Loket"  (default RS Anggrek Mas)
//	"POLI"  → req.SubJenis (mis. nm_poli sudah string)
//	"UMUM"  → "CS" (counter service)
//
// Kalau Khanza unreachable, return ErrOffline → caller fallback SQLite.
func (c *MySQLClient) BuatAntrian(ctx context.Context, req domain.AntrianRequest) (*domain.Ticket, error) {
	if strings.TrimSpace(req.Jenis) == "" {
		return nil, errors.New("antrian: jenis wajib")
	}

	tipe := mapJenisToType(req.Jenis, req.SubJenis)
	prefix := prefixForType(tipe)

	antrianMu.Lock()
	defer antrianMu.Unlock()

	// SELECT MAX next number
	var maxNum sql.NullInt64
	err := c.db.QueryRowContext(ctx, `
		SELECT IFNULL(MAX(CAST(noantrian AS UNSIGNED)), 0)
		FROM antrian_loket
		WHERE type = ? AND postdate = CURDATE()
	`, tipe).Scan(&maxNum)
	if err != nil {
		return nil, fmt.Errorf("get max antrian: %w", wrapOfflineMySQL(err))
	}
	next := int(maxNum.Int64) + 1
	noAntrian := fmt.Sprintf("%03d", next) // 001, 002, ...
	displayNomor := fmt.Sprintf("%s%03d", prefix, next) // A-001, B-001 untuk display

	// INSERT row
	now := time.Now()
	_, err = c.db.ExecContext(ctx, `
		INSERT INTO antrian_loket (type, noantrian, postdate, start_time, end_time)
		VALUES (?, ?, CURDATE(), ?, '00:00:00')
	`, tipe, noAntrian, now.Format("15:04:05"))
	if err != nil {
		return nil, fmt.Errorf("insert antrian_loket: %w", wrapOfflineMySQL(err))
	}

	return &domain.Ticket{
		Nomor:     displayNomor,
		Jenis:     req.Jenis,
		SubJenis:  req.SubJenis,
		Prefix:    prefix,
		NoUrut:    next,
		NoRM:      req.NoRM,
		CreatedAt: now,
	}, nil
}

// mapJenisToType memetakan domain.AntrianRequest.Jenis ke value
// kolom `type` di tabel antrian_loket Khanza.
func mapJenisToType(jenis, subJenis string) string {
	switch strings.ToUpper(strings.TrimSpace(jenis)) {
	case "LOKET":
		return "Loket"
	case "UMUM":
		return "CS"
	case "POLI":
		if subJenis != "" {
			return "Poli " + subJenis
		}
		return "Poli"
	case "RANAP", "IGD":
		return "Ranap"
	default:
		// Pass-through (mis. langsung "Loket", "Booking" dari caller)
		return jenis
	}
}

// prefixForType return huruf prefix display untuk tipe antrian
// (mengikuti convention KhanzaHMSAnjunganSEP V3).
func prefixForType(tipe string) string {
	switch tipe {
	case "Loket":
		return "A"
	case "CS":
		return "B"
	case "Booking":
		return "C"
	case "Ranap":
		return "D"
	default:
		return "A"
	}
}

// ErrAntrianHandledLocally — backwards compat untuk service layer yang
// punya special-case lama. Sekarang BuatAntrian return error nyata
// (offline/db) bukan sentinel ini, tapi konstanta tetap exported supaya
// caller existing tidak break.
var ErrAntrianHandledLocally = errors.New("khanza-mysql: antrian di-handle local SQLite")

// ============================================================
// SimpanSEP
// ============================================================

// SimpanSEP INSERT (atau REPLACE) ke `bridging_sep` dengan kolom
// kritikal yang dibutuhkan untuk klaim BPJS. Sisa kolom (klsnaik,
// suplesi, katarak, dll) diset default kosong/enum default — bisa
// di-update kemudian via flow admin.
//
// domain.SEP tidak punya field NoRawat eksplisit; di-resolve dari
// reg_periksa terbaru pasien BPJS di tgl SEP. Kalau tidak ketemu,
// return error supaya caller jalankan BuatPendaftaran dulu.
//
// Side-effect: kalau sep.PRBCode non-empty, juga insert ke `bpjs_prb`
// (PK no_sep) — idempoten via REPLACE.
func (c *MySQLClient) SimpanSEP(ctx context.Context, sep domain.SEP) error {
	if sep.NoSEP == "" || sep.NoKartu == "" {
		return errors.New("simpan sep: no_sep & no_kartu wajib diisi")
	}
	tglSEP := sep.TglSEP
	if tglSEP == "" {
		tglSEP = time.Now().Format("2006-01-02")
	}
	jnsPelayanan := sep.JenisPelayanan
	if jnsPelayanan == "" {
		jnsPelayanan = "2" // default Rawat Jalan
	}
	klsRawat := sep.KelasRawat
	if klsRawat == "" {
		klsRawat = "3" // default kelas 3
	}
	asalRujukan := sep.AsalRujukan
	if asalRujukan == "" {
		asalRujukan = "1. Faskes 1"
	} else if asalRujukan == "1" {
		asalRujukan = "1. Faskes 1"
	} else if asalRujukan == "2" {
		asalRujukan = "2. Faskes 2(RS)"
	}

	// Resolve no_rawat
	var noRawat string
	err := c.db.QueryRowContext(ctx, `
		SELECT rp.no_rawat
		FROM reg_periksa rp
		JOIN pasien p ON rp.no_rkm_medis = p.no_rkm_medis
		WHERE p.no_peserta = ?
		  AND rp.tgl_registrasi = ?
		  AND rp.kd_pj = ?
		  AND rp.stts <> 'Batal'
		ORDER BY rp.jam_reg DESC
		LIMIT 1`,
		sep.NoKartu, tglSEP, c.kdPjBPJS,
	).Scan(&noRawat)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("simpan sep: registrasi BPJS untuk no_kartu pada %s tidak ditemukan — jalankan BuatPendaftaran dulu", tglSEP)
	}
	if err != nil {
		return fmt.Errorf("resolve no_rawat untuk SEP: %w", wrapOfflineMySQL(err))
	}

	// Lookup nomr (no_rkm_medis) dari pasien.no_peserta untuk kolom nomr
	// di bridging_sep (referensi balik ke pasien).
	var nomr, namaPasien, jkPasien, tglLahirPasien sql.NullString
	_ = c.db.QueryRowContext(ctx, `
		SELECT no_rkm_medis, nm_pasien, jk, DATE_FORMAT(tgl_lahir,'%Y-%m-%d')
		FROM pasien WHERE no_peserta = ? LIMIT 1`,
		sep.NoKartu,
	).Scan(&nomr, &namaPasien, &jkPasien, &tglLahirPasien)

	// Override dari sep struct kalau caller sudah isi
	if sep.NoMR != "" {
		nomr = sql.NullString{String: sep.NoMR, Valid: true}
	}
	if sep.NamaPasien != "" {
		namaPasien = sql.NullString{String: sep.NamaPasien, Valid: true}
	}

	// REPLACE INTO supaya re-issue SEP idempoten.
	const sqlQ = `
		REPLACE INTO bridging_sep (
		  no_sep, no_rawat, tglsep,
		  tglrujukan, no_rujukan, kdppkrujukan, nmppkrujukan,
		  jnspelayanan, klsrawat,
		  diagawal, nmdiagnosaawal,
		  kdpolitujuan, nmpolitujuan,
		  asal_rujukan, no_kartu,
		  nomr, nama_pasien, tanggal_lahir, jkel,
		  noskdp, kddpjp, nmdpdjp
		) VALUES (
		  ?,?,?,
		  NULLIF(?, ''), ?, ?, ?,
		  ?, ?,
		  ?, ?,
		  ?, ?,
		  ?, ?,
		  ?, ?, NULLIF(?, ''), ?,
		  ?, ?, ?
		)
	`
	if _, err := c.db.ExecContext(ctx, sqlQ,
		sep.NoSEP, noRawat, tglSEP,
		sep.TglRujukan, sep.NoRujukan, sep.KdPPKRujukan, sep.NmPPKRujukan,
		jnsPelayanan, klsRawat,
		sep.DiagnosaAwal, sep.NamaDiagnosa,
		sep.KdPoli, sep.NmPoli,
		asalRujukan, sep.NoKartu,
		nomr.String, namaPasien.String, tglLahirPasien.String, jkPasien.String,
		sep.NoSKDP, valueOr(sep.KdDPJP, sep.KdDokter), valueOr(sep.NmDPJP, sep.NmDokter),
	); err != nil {
		return fmt.Errorf("simpan sep %s: %w", sep.NoSEP, wrapOfflineMySQL(err))
	}

	// Side-effect: PRB row kalau ada
	if strings.TrimSpace(sep.PRBCode) != "" {
		if _, err := c.db.ExecContext(ctx,
			`REPLACE INTO bpjs_prb (no_sep, prb) VALUES (?, ?)`,
			sep.NoSEP, sep.PRBCode,
		); err != nil {
			c.logger.Warn("simpan sep: PRB row gagal disimpan (SEP utama tetap OK)",
				"no_sep", sep.NoSEP, "err", err.Error())
		}
	}
	return nil
}

// valueOr return primary kalau non-empty, else fallback.
func valueOr(primary, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return fallback
}

// ============================================================
// SimpanRujukMasuk
// ============================================================

// SimpanRujukMasuk INSERT ke `rujuk_masuk` (PK = no_rawat). Idempoten
// via REPLACE — re-call dengan no_rawat sama akan overwrite.
func (c *MySQLClient) SimpanRujukMasuk(ctx context.Context, r domain.RujukMasuk) error {
	if r.NoRawat == "" {
		return errors.New("simpan rujuk masuk: no_rawat wajib")
	}
	kategori := strings.TrimSpace(r.KategoriRujuk)
	if kategori == "" {
		kategori = "-"
	}
	const sqlQ = `
		REPLACE INTO rujuk_masuk (
		  no_rawat, perujuk, alamat, no_rujuk,
		  jm_perujuk, dokter_perujuk, kd_penyakit,
		  kategori_rujuk, keterangan
		) VALUES (?,?,?,?,0,?,?,?,?)
	`
	if _, err := c.db.ExecContext(ctx, sqlQ,
		r.NoRawat, r.Perujuk, r.Alamat, r.NoRujuk,
		r.DokterPerujuk, r.KdPenyakit, kategori, r.Keterangan,
	); err != nil {
		return fmt.Errorf("simpan rujuk masuk %s: %w", r.NoRawat, wrapOfflineMySQL(err))
	}
	return nil
}

// ============================================================
// UpdateSatuSehatID
// ============================================================

// UpdateSatuSehatID — RS sikrsam260312 hanya punya tabel
// flagging_pasien_satusehat (enum yes/no), tidak ada kolom IHS Number.
// Kita hanya set flag yes; IHS string di-log warning.
//
// Kalau ihsNumber kosong: set flag = 'no' (de-aktivasi).
func (c *MySQLClient) UpdateSatuSehatID(ctx context.Context, noRM, ihsNumber string) error {
	if noRM == "" {
		return errors.New("update satusehat: no_rm wajib")
	}
	flag := "yes"
	if strings.TrimSpace(ihsNumber) == "" {
		flag = "no"
	}

	// Probe tabel — kalau tidak ada, return nil (graceful).
	var exists int
	err := c.db.QueryRowContext(ctx,
		`SELECT 1 FROM information_schema.TABLES
		 WHERE TABLE_SCHEMA = DATABASE()
		   AND TABLE_NAME = 'flagging_pasien_satusehat' LIMIT 1`,
	).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		c.logger.Warn("khanza-mysql: tabel flagging_pasien_satusehat tidak ada — IHS tidak disimpan",
			"no_rm_hash", maskNoRM(noRM))
		return nil
	}
	if err != nil {
		return fmt.Errorf("probe satusehat: %w", wrapOfflineMySQL(err))
	}

	const sqlQ = `
		REPLACE INTO flagging_pasien_satusehat (no_rkm_medis, flagging, timestamp)
		VALUES (?, ?, NOW())
	`
	if _, err := c.db.ExecContext(ctx, sqlQ, noRM, flag); err != nil {
		return fmt.Errorf("update satusehat flag: %w", wrapOfflineMySQL(err))
	}
	if flag == "yes" {
		// IHS string tidak punya tempat di Khanza schema. APM bisa simpan
		// di SQLite lokal (untuk kebutuhan Satu Sehat sync) — caller
		// bertanggung jawab untuk itu.
		c.logger.Info("khanza-mysql: flag satusehat=yes diset (IHS hanya dilog, simpan di SQLite lokal)",
			"no_rm_hash", maskNoRM(noRM), "ihs_len", len(ihsNumber))
	}
	return nil
}

// maskNoRM mengembalikan no_rm yang dimask kecuali 2 char terakhir,
// untuk log yang ramah PHI di luar PHIMaskingHandler scope (fallback).
func maskNoRM(noRM string) string {
	if len(noRM) <= 2 {
		return strings.Repeat("*", len(noRM))
	}
	return strings.Repeat("*", len(noRM)-2) + noRM[len(noRM)-2:]
}

// ============================================================
// Offline detection
// ============================================================

// wrapOfflineMySQL deteksi MySQL-specific connection failures dan
// petakan ke domain.ErrOffline. Selain itu, fallback ke wrapOffline()
// generic yang sudah handle syscall-level (ECONNREFUSED, dll).
func wrapOfflineMySQL(err error) error {
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "connection refused"),
		strings.Contains(msg, "no such host"),
		strings.Contains(msg, "i/o timeout"),
		strings.Contains(msg, "broken pipe"),
		strings.Contains(msg, "connect: network is unreachable"),
		strings.Contains(msg, "invalid connection"),
		strings.Contains(msg, "bad connection"),
		strings.Contains(msg, "no route to host"):
		return domain.ErrOffline
	}
	return wrapOffline(err)
}
