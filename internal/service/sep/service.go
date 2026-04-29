// Package sep adalah service layer untuk penerbitan SEP (Surat
// Eligibilitas Peserta) BPJS.
//
// 4 jenis SEP yang ditangani:
//
//   - BuatSEPRujukan     pasien dengan rujukan FKTP (kunjungan baru)
//   - BuatSEPKontrol     pasien dengan SKDP (jadwal kontrol)
//   - BuatSEPPostRANAP   kontrol pasca rawat inap (≤7 hari keluar)
//   - BuatSEPPostRAJAL   lanjutan rawat jalan ke poli berbeda
//
// Pattern umum semua metode:
//
//  1. Validasi pre-condition (rujukan/surat kontrol/dll)
//  2. Verifikasi sidik jari kalau perluBiometrik() = true
//  3. Call vclaim.CreateSEP* dengan FPToken bila ada
//  4. Persist ke print_history (audit + bahan reprint)
//  5. Post ke Khanza; kalau gagal/offline → simpan ke pending_sep
//     supaya reconcile worker (P-050) bisa flush nanti
//
// Setelah VClaim issue SEP, SEP itu adalah obligasi nyata di BPJS.
// Apapun yang gagal di langkah berikutnya TIDAK boleh menggagalkan
// return — yang penting SEP sampai ke pasien & operator. Kegagalan
// downstream di-log + simpan ke pending_sep.
package sep

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/arunika/apm-go/internal/domain"
	"github.com/arunika/apm-go/internal/hardware/fingerprint"
	"github.com/arunika/apm-go/internal/integration/khanza"
	"github.com/arunika/apm-go/internal/integration/vclaim"
	"github.com/arunika/apm-go/internal/store"
)

// SEPService menggabungkan vclaim (issue SEP) + khanza (audit & klaim)
// + fingerprint (biometrik) + store (offline persistence).
type SEPService struct {
	vclaim vclaim.VClaimClient
	khanza khanza.KhanzaClient
	fp     fingerprint.FingerprintVerifier
	store  *store.Queries

	// PPK Pelayanan RS (kode + nama) — di-inject dari config.BPJS +
	// config.Branding. Vendor populate ini ke bridging_sep.kdppkpelayanan
	// & nmppkpelayanan (line 2702-2703).
	ppkPelayanan     string
	ppkPelayananName string

	now    func() time.Time
	logger *slog.Logger
}

// New membangun SEPService dari dependencies.
func New(
	v vclaim.VClaimClient,
	k khanza.KhanzaClient,
	fp fingerprint.FingerprintVerifier,
	db *sql.DB,
) *SEPService {
	return &SEPService{
		vclaim: v,
		khanza: k,
		fp:     fp,
		store:  store.New(db),
		now:    time.Now,
		logger: slog.Default(),
	}
}

// SetLogger mengganti logger (dipakai test atau caller dengan
// PHIMaskingHandler global di P-051).
func (s *SEPService) SetLogger(l *slog.Logger) {
	if l != nil {
		s.logger = l
	}
}

// SetPPKPelayanan inject PPK code + nama RS dari config (dipanggil
// di App.initialize).
func (s *SEPService) SetPPKPelayanan(kode, nama string) {
	s.ppkPelayanan = kode
	s.ppkPelayananName = nama
}

// ============================================================
// BuatSEPRujukan — pasien rujukan FKTP (kunjungan baru)
// ============================================================
//
// Catatan signature: spec menulis (ctx, req) tapi metode butuh data
// peserta (untuk perluBiometrik check usia + pesan log). Pasien sudah
// di-resolve di DetectScreen, jadi caller pasti punya — kita ambil
// sebagai param eksplisit supaya tidak ada lookup ulang.
func (s *SEPService) BuatSEPRujukan(
	ctx context.Context,
	peserta *domain.Peserta,
	req domain.SEPRequest,
) (*domain.SEP, error) {
	if peserta == nil {
		return nil, fmt.Errorf("buat sep rujukan: peserta nil")
	}
	if req.NoRujukan == "" {
		return nil, fmt.Errorf("buat sep rujukan: NoRujukan wajib diisi")
	}

	tglSEP := parseDateOrNow(req.TglSEP, s.now()).Format("2006-01-02")
	if req.TglSEP == "" {
		req.TglSEP = tglSEP
	}

	// Auto-populate field yang vendor expect tapi caller mungkin tidak
	// pasok eksplisit. Vendor line 2623 noMR + line 2668 user.
	// NoTelp tidak ada di domain.Peserta — caller (frontend) yang pasok
	// kalau ada (mis. dari pasien.no_telp Khanza).
	if req.NoMR == "" {
		req.NoMR = peserta.NoRM
	}
	if req.User == "" {
		req.User = peserta.NoKartu
	}

	// Step 1: pre-flight checks (mirror Java DlgRegistrasiSEPPertama:1734-1794)
	if err := s.runPreflight(ctx, peserta, req.KdPoli, req.KdDokter, peserta.NoKartu, tglSEP); err != nil {
		return nil, err
	}

	// Step 2: biometrik kalau perlu (umur >=17 + non-IGD).
	// Kalau caller (Wails frontend) sudah panggil VerifikasiWajah /
	// VerifikasiSidikJari dan pasok req.BiometrikToken, pakai langsung.
	fpToken, err := s.maybeBiometrik(ctx, *peserta, req.KdPoli, req.BiometrikToken)
	if err != nil {
		return nil, err
	}
	req.FPToken = fpToken

	// Step 3: validasi rujukan FKTP via VClaim
	if _, err := s.vclaim.ValidasiRujukan(ctx, req.NoRujukan, parseDateOrNow(tglSEP, s.now())); err != nil {
		return nil, fmt.Errorf("validasi rujukan: %w", err)
	}

	// Step 4: INSERT reg_periksa di Khanza DULU (mirror Java line 2682-2685
	// — bridging_sep punya FK ke reg_periksa.no_rawat, jadi reg_periksa
	// harus exist sebelum SimpanSEP).
	pendaftaran, err := s.khanza.BuatPendaftaran(ctx, domain.PendaftaranRequest{
		NoRM:       peserta.NoRM,
		KdPoli:     req.KdPoli,
		KdDokter:   req.KdDokter,
		TglPeriksa: tglSEP,
		Penjamin:   "BPJS",
		Catatan:    req.CatatanPelayanan,
	})
	if err != nil {
		return nil, fmt.Errorf("buat pendaftaran BPJS: %w", err)
	}
	if pendaftaran == nil {
		pendaftaran = &domain.Pendaftaran{}
	}
	s.logger.Info("sep rujukan: pendaftaran tercatat",
		"no_rawat", pendaftaran.NoRawat, "no_urut", pendaftaran.NoUrut)

	// Step 5: issue SEP via VClaim
	sepObj, err := s.vclaim.CreateSEP(ctx, req)
	if err != nil {
		// reg_periksa sudah ke-INSERT — log warning, biarkan untuk reconcile manual
		s.logger.Error("sep rujukan: VClaim CreateSEP gagal — reg_periksa sudah tercatat",
			"no_rawat", pendaftaran.NoRawat, "err", err.Error())
		return nil, fmt.Errorf("vclaim CreateSEP: %w", err)
	}

	// Step 6: persist + post ke Khanza (bridging_sep + rujuk_masuk + bridging_rujukan_bpjs)
	s.persistAndSyncKhanza(ctx, sepObj, peserta, "RUJUKAN", req)

	// Step 7: simpan rujukan FKTP audit trail (Java line 2688-2692)
	_ = s.khanza.SimpanRujukMasuk(ctx, domain.RujukMasuk{
		NoRawat:       pendaftaran.NoRawat,
		Perujuk:       req.NmPPKRujukan,
		NoRujuk:       req.NoRujukan,
		KdPenyakit:    req.DiagnosaAwal,
		KategoriRujuk: "-",
	})
	_ = s.khanza.SimpanRujukanBPJS(ctx, domain.RujukanBPJS{
		NoSEP:         sepObj.NoSEP,
		NoRujukan:     req.NoRujukan,
		TglRujukan:    req.TglRujukan,
		PPKDirujuk:    req.KdPPKRujukan,
		NmPPKDirujuk:  req.NmPPKRujukan,
		JnsPelayanan:  "2",
		DiagRujukan:   req.DiagnosaAwal,
		NmDiagRujukan: req.NamaDiagnosa,
		PoliRujukan:   req.KdPoli,
		NmPoliRujukan: sepObj.NmPoli,
		User:          "kiosk-tara",
	})

	return sepObj, nil
}

// runPreflight menjalankan 3 cek wajib sebelum issue SEP (mirror
// vendor Java validasi flow):
//
//  1. CheckDuplicateRegistration di reg_periksa lokal
//  2. CheckDoctorOnLeave di jadwal_cuti_libur lokal
//  3. CekSEPDuplikasi di server BPJS via VClaim
//
// Pre-flight failure → return error supaya UI tidak masuk ke flow VClaim
// (cegah duplikasi state lokal vs BPJS).
func (s *SEPService) runPreflight(
	ctx context.Context,
	peserta *domain.Peserta,
	kdPoli, kdDokter, noKartu, tglSEP string,
) error {
	if dup, err := s.khanza.CheckDuplicateRegistration(ctx, peserta.NoRM, kdPoli, kdDokter, tglSEP, "BPJ"); err != nil {
		s.logger.Warn("preflight: check duplicate registration error", "err", err.Error())
	} else if dup {
		return domain.ErrSudahTerdaftarHariIni
	}

	if cuti, err := s.khanza.CheckDoctorOnLeave(ctx, kdDokter, tglSEP); err != nil {
		s.logger.Warn("preflight: check doctor on leave error", "err", err.Error())
	} else if cuti {
		return domain.ErrDokterCuti
	}

	if dupSEP, err := s.vclaim.CekSEPDuplikasi(ctx, noKartu, tglSEP); err != nil {
		// VClaim error tidak fatal — log + lanjutkan (BPJS server akan reject sendiri kalau memang dup)
		s.logger.Warn("preflight: CekSEPDuplikasi VClaim error (lanjut)", "err", err.Error())
	} else if dupSEP != nil {
		return fmt.Errorf("%w: SEP existing %s", domain.ErrDuplikasiSEP, dupSEP.NoSEP)
	}
	return nil
}

// ============================================================
// BuatSEPKontrol — pasien dengan SKDP
// ============================================================

func (s *SEPService) BuatSEPKontrol(
	ctx context.Context,
	peserta *domain.Peserta,
	noSuratKontrol string,
	kdDokter string,
	biometrikToken string,
) (*domain.SEP, error) {
	if peserta == nil {
		return nil, fmt.Errorf("buat sep kontrol: peserta nil")
	}
	if noSuratKontrol == "" {
		return nil, fmt.Errorf("buat sep kontrol: noSuratKontrol wajib diisi")
	}

	// Step 1: cari surat kontrol di Khanza
	list, err := s.khanza.GetSuratKontrol(ctx, peserta.NoRM)
	if err != nil {
		return nil, fmt.Errorf("ambil surat kontrol: %w", err)
	}
	var sk *domain.SuratKontrol
	for i := range list {
		if list[i].NoSurat == noSuratKontrol {
			sk = &list[i]
			break
		}
	}
	if sk == nil {
		return nil, domain.ErrSuratKontrolTidakDitemukan
	}

	// Step 2: validasi tanggal
	if !sk.IsTodayOrPast() {
		return nil, domain.ErrJadwalKontrolBelumTiba(sk.TglRencana)
	}

	tglStr := s.now().Format("2006-01-02")
	resolvedKdDokter := firstNonEmpty(kdDokter, sk.KdDokter)

	// Step 2.5: pre-flight checks (mirror Java DlgRegistrasiSEPPertama)
	if err := s.runPreflight(ctx, peserta, sk.KdPoli, resolvedKdDokter, peserta.NoKartu, tglStr); err != nil {
		return nil, err
	}

	// Step 3: biometrik kalau perlu — frontend supply token kalau pasien
	// baru saja verifikasi via BiometrikChoiceModal (Frista/After.exe).
	// Kalau biometrikToken kosong, maybeBiometrik cek server BPJS dulu;
	// kalau Verified=true SEP boleh issued tanpa token.
	fpToken, err := s.maybeBiometrik(ctx, *peserta, sk.KdPoli, biometrikToken)
	if err != nil {
		return nil, err
	}

	// Step 4: INSERT reg_periksa BPJS (sebelum issue SEP)
	pendaftaran, err := s.khanza.BuatPendaftaran(ctx, domain.PendaftaranRequest{
		NoRM:       peserta.NoRM,
		KdPoli:     sk.KdPoli,
		KdDokter:   resolvedKdDokter,
		TglPeriksa: tglStr,
		Penjamin:   "BPJS",
		Catatan:    "Kontrol — " + sk.NoSurat,
	})
	if err != nil {
		return nil, fmt.Errorf("buat pendaftaran kontrol: %w", err)
	}
	if pendaftaran == nil {
		pendaftaran = &domain.Pendaftaran{}
	}
	s.logger.Info("sep kontrol: pendaftaran tercatat",
		"no_rawat", pendaftaran.NoRawat, "no_surat", sk.NoSurat)

	// Step 5: build request + issue SEP via VClaim
	req := domain.SEPKontrolRequest{
		NoSuratKontrol: noSuratKontrol,
		NoKartu:        peserta.NoKartu,
		TglSEP:         tglStr,
		KdDokter:       resolvedKdDokter,
		KelasRawat:     peserta.KelasHak,
		JnsPelayanan:   "1", // Rawat Jalan
		FPToken:        fpToken,
	}
	sepObj, err := s.vclaim.CreateSEPKontrol(ctx, req)
	if err != nil {
		s.logger.Error("sep kontrol: VClaim gagal — reg_periksa sudah tercatat",
			"no_rawat", pendaftaran.NoRawat, "err", err.Error())
		return nil, fmt.Errorf("vclaim CreateSEPKontrol: %w", err)
	}

	// Step 6: persist
	s.persistAndSyncKhanza(ctx, sepObj, peserta, "KONTROL", req)
	return sepObj, nil
}

// ============================================================
// BuatSEPPostRANAP — kontrol pasca rawat inap
// ============================================================

func (s *SEPService) BuatSEPPostRANAP(
	ctx context.Context,
	peserta *domain.Peserta,
	kdPoliKontrol string,
	kdDokter string,
	biometrikToken string,
) (*domain.SEP, error) {
	return s.buatSEPPasca(ctx, peserta, kdPoliKontrol, kdDokter, biometrikToken, "POST_RANAP", "POST-RANAP", "2")
}

// ============================================================
// BuatSEPPostRAJAL — lanjutan rawat jalan beda poli
// ============================================================

func (s *SEPService) BuatSEPPostRAJAL(
	ctx context.Context,
	peserta *domain.Peserta,
	kdPoliTujuan string,
	kdDokter string,
	biometrikToken string,
) (*domain.SEP, error) {
	return s.buatSEPPasca(ctx, peserta, kdPoliTujuan, kdDokter, biometrikToken, "POST_RAJAL", "POST-RAJAL", "1")
}

// buatSEPPasca shared flow untuk POST-RANAP & POST-RAJAL.
// Keduanya tidak butuh NoRujukan (basis: rawat sebelumnya), tapi
// tetap perlu biometrik check & persist.
//
// jnsPelayanan: "1" Rawat Jalan / "2" Rawat Inap. POST-RANAP biasanya
// dilanjutkan kontrol RJ (jenis "1") tapi vendor allow override —
// caller (BuatSEPPostRANAP/RAJAL) pasok jenis sesuai konteks.
func (s *SEPService) buatSEPPasca(
	ctx context.Context,
	peserta *domain.Peserta,
	kdPoli, kdDokter string,
	biometrikToken string,
	kategori, label string,
	jnsPelayanan string,
) (*domain.SEP, error) {
	if peserta == nil {
		return nil, fmt.Errorf("buat sep %s: peserta nil", label)
	}
	if kdPoli == "" || kdDokter == "" {
		return nil, fmt.Errorf("buat sep %s: kdPoli/kdDokter wajib diisi", label)
	}

	tglStr := s.now().Format("2006-01-02")

	// Pre-flight checks
	if err := s.runPreflight(ctx, peserta, kdPoli, kdDokter, peserta.NoKartu, tglStr); err != nil {
		return nil, err
	}

	// Biometrik — frontend supply token jika sudah verifikasi via modal,
	// atau kosong → maybeBiometrik cek server BPJS dulu.
	fpToken, err := s.maybeBiometrik(ctx, *peserta, kdPoli, biometrikToken)
	if err != nil {
		return nil, err
	}

	// INSERT reg_periksa BPJS (sebelum SEP issue)
	pendaftaran, err := s.khanza.BuatPendaftaran(ctx, domain.PendaftaranRequest{
		NoRM:       peserta.NoRM,
		KdPoli:     kdPoli,
		KdDokter:   kdDokter,
		TglPeriksa: tglStr,
		Penjamin:   "BPJS",
		Catatan:    label,
	})
	if err != nil {
		return nil, fmt.Errorf("buat pendaftaran %s: %w", label, err)
	}
	if pendaftaran == nil {
		pendaftaran = &domain.Pendaftaran{}
	}
	s.logger.Info("sep "+strings.ToLower(label)+": pendaftaran tercatat",
		"no_rawat", pendaftaran.NoRawat)

	req := domain.SEPRequest{
		NoKartu:      peserta.NoKartu,
		TglSEP:       tglStr,
		KdPoli:       kdPoli,
		KdDokter:     kdDokter,
		JnsPelayanan: jnsPelayanan,
		KelasRawat:   peserta.KelasHak,
		FPToken:      fpToken,
		NoMR:         peserta.NoRM,
		User:         peserta.NoKartu,
		// NoRujukan kosong — backend BPJS akan auto-link ke episode
		// rawat sebelumnya untuk POST_RANAP/POST_RAJAL kalau valid.
	}

	sepObj, err := s.vclaim.CreateSEP(ctx, req)
	if err != nil {
		s.logger.Error("sep "+strings.ToLower(label)+": VClaim gagal — reg_periksa sudah tercatat",
			"no_rawat", pendaftaran.NoRawat, "err", err.Error())
		return nil, fmt.Errorf("vclaim CreateSEP %s: %w", label, err)
	}

	s.persistAndSyncKhanza(ctx, sepObj, peserta, kategori, req)
	return sepObj, nil
}

// ============================================================
// Helpers
// ============================================================

// maybeBiometrik menentukan apakah biometrik perlu di-prompt sebelum
// issue SEP. Mirror vendor cekFinger() di DlgRegistrasiSEPPertama.java:
//
//  1. Kalau perluBiometrik() = false (anak-anak / IGD) → skip total.
//  2. Kalau perluBiometrik() = true → cek status server BPJS dulu via
//     /SEP/FingerPrint/Peserta/{noka}/TglPelayanan/{tgl}.
//     - Verified=true (statusfinger=true di vendor) → SEP boleh di-issue
//       langsung, BPJS server sudah punya record. Return "" (tidak ada
//       token yg perlu di-attach — vendor juga tidak attach).
//     - Verified=false → frontend wajib prompt modal dulu (Frista atau
//       After.exe), lalu retry BuatSEP. Return ErrBiometrikDibutuhkan.
//
// Param externalToken di-terima untuk backward compat tapi TIDAK
// di-attach ke payload SEP (vendor tidak punya field "finger" di body).
// Token cuma sebagai sinyal "frontend sudah selesai pop modal" — kalau
// non-empty, kita asumsikan flow biometrik sudah dijalankan & re-cek
// status server untuk konfirmasi.
//
// Return value: string token utk legacy compat (selalu "" sekarang),
// dan error ErrBiometrikDibutuhkan kalau pasien belum verified server-side.
func (s *SEPService) maybeBiometrik(
	ctx context.Context,
	peserta domain.Peserta,
	kdPoli string,
	externalToken string,
) (string, error) {
	if !perluBiometrik(peserta, kdPoli) {
		return "", nil
	}

	// Cek status biometrik di server BPJS (vendor pattern).
	tglNow := s.now()
	st, err := s.vclaim.CekFingerprintStatus(ctx, peserta.NoKartu, tglNow)
	if err != nil {
		// Network/decrypt error — degradasi: kalau frontend sudah pasok
		// externalToken (artinya pasien baru saja verifikasi via modal),
		// trust itu sebagai sinyal kesuksesan biometrik. Vendor server
		// pun bisa lag setelah Frista/After.exe submit.
		s.logger.Warn("sep: cek fingerprint server gagal",
			"no_kartu_masked", maskID(peserta.NoKartu), "err", err.Error())
		if externalToken != "" {
			s.logger.Info("sep: trust externalToken karena cekFinger error",
				"no_kartu_masked", maskID(peserta.NoKartu))
			return "", nil
		}
		return "", domain.ErrBiometrikDibutuhkan
	}

	if st != nil && st.Verified {
		s.logger.Info("sep: biometrik server-side verified, lanjut issue SEP",
			"no_kartu_masked", maskID(peserta.NoKartu), "kd_poli", kdPoli)
		return "", nil
	}

	// Belum verified server-side — frontend harus prompt modal.
	s.logger.Info("sep: biometrik belum verified server-side, prompt modal",
		"no_kartu_masked", maskID(peserta.NoKartu), "kd_poli", kdPoli,
		"vendor_status", func() string {
			if st != nil {
				return st.Message
			}
			return ""
		}())
	return "", domain.ErrBiometrikDibutuhkan
}

// persistAndSyncKhanza menjalankan langkah 4-5 spec:
//
//	4. Insert ke print_history (backup + bahan reprint)
//	5. Post ke Khanza; kalau gagal/offline → insert ke pending_sep
//
// Sebelum post: enrichSEPForBridging copy field-field tambahan dari
// originalReq + peserta ke sepObj supaya bridging_sep INSERT lengkap
// (vendor 52 kolom mirror).
//
// Tidak pernah mengembalikan error — SEP sudah issued di VClaim,
// kegagalan downstream di-log dan masuk pending_sep untuk reconcile.
func (s *SEPService) persistAndSyncKhanza(
	ctx context.Context,
	sepObj *domain.SEP,
	peserta *domain.Peserta,
	kategori string,
	originalReq any,
) {
	s.enrichSEPForBridging(sepObj, peserta, originalReq)
	// Step 4: print_history (escpos_bytes placeholder JSON sampai
	// ESC/POS template di P-033 mengganti dengan raw bytes asli).
	sepBytes, _ := json.Marshal(sepObj)
	if _, err := s.store.InsertPrintHistory(ctx, store.InsertPrintHistoryParams{
		DocType:     "SEP",
		RefID:       sql.NullString{String: sepObj.NoSEP, Valid: sepObj.NoSEP != ""},
		EscposBytes: sepBytes,
	}); err != nil {
		s.logger.Warn("sep: gagal insert print_history",
			"no_sep", sepObj.NoSEP, "err", err.Error())
		// continue — bukan blocker
	}

	// Step 5: post ke Khanza
	khanzaErr := s.khanza.SimpanSEP(ctx, *sepObj)
	if khanzaErr == nil {
		s.logger.Info("sep: SEP sukses di-post ke Khanza",
			"no_sep", sepObj.NoSEP, "kategori", kategori)
		return
	}

	// Khanza gagal — insert ke pending_sep, audit log, return tetap sukses
	payload, _ := json.Marshal(originalReq)
	respBytes, _ := json.Marshal(sepObj)

	pending, err := s.store.InsertPendingSEP(ctx, store.InsertPendingSEPParams{
		NoKartu:        sepObj.NoKartu,
		Kategori:       kategori,
		PayloadJson:    string(payload),
		VclaimResponse: sql.NullString{String: string(respBytes), Valid: true},
	})
	if err != nil {
		s.logger.Error("sep: SEP issued tapi GAGAL simpan ke pending_sep",
			"no_sep", sepObj.NoSEP, "err", err.Error(),
			"khanza_err", khanzaErr.Error())
		return
	}

	if errors.Is(khanzaErr, domain.ErrOffline) {
		s.logger.Warn("sep: Khanza offline, SEP disimpan ke pending_sep",
			"no_sep", sepObj.NoSEP, "pending_id", pending.ID)
	} else {
		s.logger.Warn("sep: Khanza error, SEP disimpan ke pending_sep",
			"no_sep", sepObj.NoSEP, "pending_id", pending.ID,
			"err", khanzaErr.Error())
	}
}

// parseDateOrNow parse "2006-01-02" string atau fallback ke fallback time.
func parseDateOrNow(s string, fallback time.Time) time.Time {
	if s == "" {
		return fallback
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return fallback
	}
	return t
}

// firstNonEmpty mengembalikan a kalau non-empty, otherwise b.
func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// enrichSEPForBridging mengisi field-field tambahan di sepObj dari
// originalReq supaya khanza.SimpanSEP punya data lengkap untuk INSERT
// bridging_sep 52 kolom (vendor parity).
//
// VClaim CreateSEP response hanya return field core (noSep, tglSep,
// kdPoli, kdDokter). Field tambahan (Catatan, JenisPeserta, NoTelp,
// Suplesi, FlagProcedure, dll) ada di SEPRequest tapi tidak di response.
// Service layer copy supaya tidak hilang saat persistence.
func (s *SEPService) enrichSEPForBridging(sepObj *domain.SEP, peserta *domain.Peserta, originalReq any) {
	if sepObj == nil {
		return
	}

	// Profil pasien snapshot dari Peserta (vendor line 2719-2720, 2726).
	// Field-field ini tidak ada di SEPRequest tapi vendor INSERT-nya.
	if peserta != nil {
		if sepObj.NamaPasien == "" {
			sepObj.NamaPasien = peserta.Nama
		}
		if sepObj.JenisPeserta == "" {
			sepObj.JenisPeserta = peserta.JenisPeserta
		}
		if sepObj.TglLahir == "" {
			sepObj.TglLahir = peserta.TglLahir
		}
		if sepObj.NoMR == "" {
			sepObj.NoMR = peserta.NoRM
		}
		if sepObj.User == "" {
			sepObj.User = peserta.NoKartu
		}
		// JK tidak ada di domain.Peserta — mysql_client fallback ke
		// lookup tabel pasien (column jk).
	}

	// PPK Pelayanan dari config (vendor line 2702-2703).
	if sepObj.KdPPKPelayanan == "" {
		sepObj.KdPPKPelayanan = s.ppkPelayanan
	}
	if sepObj.NmPPKPelayanan == "" {
		sepObj.NmPPKPelayanan = s.ppkPelayananName
	}

	switch req := originalReq.(type) {
	case domain.SEPRequest:
		copySEPRequestToSEP(sepObj, &req)
	case *domain.SEPRequest:
		copySEPRequestToSEP(sepObj, req)
	case domain.SEPKontrolRequest:
		if sepObj.User == "" {
			sepObj.User = req.NoKartu
		}
	}
}

func copySEPRequestToSEP(sepObj *domain.SEP, req *domain.SEPRequest) {
	if req == nil || sepObj == nil {
		return
	}
	if sepObj.Catatan == "" {
		sepObj.Catatan = req.CatatanPelayanan
	}
	if sepObj.NoTelp == "" {
		sepObj.NoTelp = req.NoTelp
	}
	if sepObj.Suplesi == "" {
		sepObj.Suplesi = req.Suplesi
	}
	if sepObj.NoSepSuplesi == "" {
		sepObj.NoSepSuplesi = req.NoSepSuplesi
	}
	if sepObj.FlagProcedure == "" {
		sepObj.FlagProcedure = req.FlagProcedure
	}
	if sepObj.KdPenunjang == "" {
		sepObj.KdPenunjang = req.KdPenunjang
	}
	if sepObj.KdDPJPLayanan == "" {
		sepObj.KdDPJPLayanan = req.KdDPJPLayanan
	}
	if sepObj.NoMR == "" {
		sepObj.NoMR = req.NoMR
	}
	if sepObj.User == "" {
		sepObj.User = req.User
	}
	// Field core mirror (kalau VClaim response kosong)
	if sepObj.LakaLantas == "" {
		sepObj.LakaLantas = req.LakaLantas
	}
	if sepObj.TglKejadian == "" {
		sepObj.TglKejadian = req.TglKejadian
	}
	if sepObj.KetKecelakaan == "" {
		sepObj.KetKecelakaan = req.KetKecelakaan
	}
	if sepObj.KdPropinsi == "" {
		sepObj.KdPropinsi = req.KdPropinsi
		sepObj.NmPropinsi = req.NmPropinsi
		sepObj.KdKabupaten = req.KdKabupaten
		sepObj.NmKabupaten = req.NmKabupaten
		sepObj.KdKecamatan = req.KdKecamatan
		sepObj.NmKecamatan = req.NmKecamatan
	}
	if sepObj.AsesmenPelayanan == "" {
		sepObj.AsesmenPelayanan = req.AsesmenPelayanan
	}
	if sepObj.TujuanKunjungan == "" {
		sepObj.TujuanKunjungan = req.TujuanKunjungan
	}
	if sepObj.COB == "" {
		sepObj.COB = req.COB
	}
	if sepObj.Eksekutif == "" {
		sepObj.Eksekutif = req.Eksekutif
	}
	// Katarak tidak di-mirror — domain.SEP tidak punya field, mysql_client
	// pakai default "0. Tidak" (mirror vendor line 2727).
	_ = req.Katarak
}

// maskID memendekkan no_kartu / no_rm jadi "************XXXX" untuk log
// (PHI safety). Duplikasi kecil dari detector.maskID — bisa di-extract
// ke internal/log/phi.go saat P-051 implement global handler.
func maskID(id string) string {
	if len(id) < 8 {
		return "***"
	}
	out := make([]byte, len(id))
	for i := range out {
		if i >= len(id)-4 {
			out[i] = id[i]
		} else {
			out[i] = '*'
		}
	}
	return string(out)
}
