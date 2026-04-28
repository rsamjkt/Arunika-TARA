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
	s.persistAndSyncKhanza(ctx, sepObj, "RUJUKAN", req)

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

	// Step 3: biometrik kalau perlu (poli kontrol).
	// BuatSEPKontrol di Wails app ambil BiometrikToken dari cache
	// (di-set saat VerifikasiWajah / VerifikasiSidikJari) — saat ini
	// signature method tidak terima req langsung, jadi kita pakai
	// internal verify (degradasi ke fingerprint mock kalau ada).
	// TODO P-031+: refactor signature kontrol terima domain.SEPKontrolRequest
	// supaya frontend bisa supply BiometrikToken seperti SEPRujukan.
	fpToken, err := s.maybeBiometrik(ctx, *peserta, sk.KdPoli, "")
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
	s.persistAndSyncKhanza(ctx, sepObj, "KONTROL", req)
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
) (*domain.SEP, error) {
	return s.buatSEPPasca(ctx, peserta, kdPoliKontrol, kdDokter, "POST_RANAP", "POST-RANAP")
}

// ============================================================
// BuatSEPPostRAJAL — lanjutan rawat jalan beda poli
// ============================================================

func (s *SEPService) BuatSEPPostRAJAL(
	ctx context.Context,
	peserta *domain.Peserta,
	kdPoliTujuan string,
	kdDokter string,
) (*domain.SEP, error) {
	return s.buatSEPPasca(ctx, peserta, kdPoliTujuan, kdDokter, "POST_RAJAL", "POST-RAJAL")
}

// buatSEPPasca shared flow untuk POST-RANAP & POST-RAJAL.
// Keduanya tidak butuh NoRujukan (basis: rawat sebelumnya), tapi
// tetap perlu biometrik check & persist.
func (s *SEPService) buatSEPPasca(
	ctx context.Context,
	peserta *domain.Peserta,
	kdPoli, kdDokter string,
	kategori, label string,
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

	// POST_RANAP/POST_RAJAL signature lama tidak terima BiometrikToken
	// dari frontend — pakai internal verify (degradasi: fingerprint mock).
	// TODO P-031+: ubah signature menerima req struct supaya frontend
	// bisa supply token sebagaimana SEPRujukan.
	fpToken, err := s.maybeBiometrik(ctx, *peserta, kdPoli, "")
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
		JnsPelayanan: "1",
		KelasRawat:   peserta.KelasHak,
		FPToken:      fpToken,
		// NoRujukan kosong — backend BPJS akan auto-link ke episode
		// rawat sebelumnya untuk POST_RANAP/POST_RAJAL kalau valid.
	}

	sepObj, err := s.vclaim.CreateSEP(ctx, req)
	if err != nil {
		s.logger.Error("sep "+strings.ToLower(label)+": VClaim gagal — reg_periksa sudah tercatat",
			"no_rawat", pendaftaran.NoRawat, "err", err.Error())
		return nil, fmt.Errorf("vclaim CreateSEP %s: %w", label, err)
	}

	s.persistAndSyncKhanza(ctx, sepObj, kategori, req)
	return sepObj, nil
}

// ============================================================
// Helpers
// ============================================================

// maybeBiometrik menentukan token biometrik untuk dilampirkan ke SEP.
//
// Prioritas:
//
//  1. Kalau perluBiometrik() = false (anak-anak / IGD) → return "" tanpa
//     verify, FPToken di payload BPJS akan kosong.
//  2. Kalau caller pasok externalToken (dari req.BiometrikToken — hasil
//     VerifikasiWajah/VerifikasiSidikJari yang sudah dilakukan frontend
//     lebih dulu) → pakai langsung, skip internal verify.
//  3. Kalau externalToken kosong DAN biometrik diperlukan → return
//     ErrBiometrikDibutuhkan supaya frontend tahu harus panggil
//     VerifikasiWajah / VerifikasiSidikJari dulu sebelum BuatSEP.
//
// CATATAN soal degradasi: dulu saat verifier tidak available kita
// log warning + skip biometrik (return ""). Sekarang flow biometrik
// sepenuhnya digerakkan frontend (panggil method khusus → pasok token),
// jadi service layer cukup percaya externalToken atau gagal cepat.
// Kalau hardware Frista/After.exe down, frontend akan tahu dari error
// VerifikasiWajah/VerifikasiSidikJari dan bisa pilih path manual
// (operator override) atau cancel.
func (s *SEPService) maybeBiometrik(
	ctx context.Context,
	peserta domain.Peserta,
	kdPoli string,
	externalToken string,
) (string, error) {
	if !perluBiometrik(peserta, kdPoli) {
		return "", nil
	}

	// Path 1: frontend sudah pasok token (dari Wails VerifikasiWajah
	// atau VerifikasiSidikJari). Pakai langsung — tidak ada internal
	// verify lagi, supaya tidak ada double-prompt ke pasien.
	if externalToken != "" {
		s.logger.Info("sep: biometrik token diterima dari frontend",
			"no_kartu_masked", maskID(peserta.NoKartu), "kd_poli", kdPoli,
			"token_len", len(externalToken))
		return externalToken, nil
	}

	// Path 2: tidak ada token external — fallback ke internal fingerprint
	// verifier (legacy path; flow ini akan jadi fallback saat frontend
	// belum migrasi ke VerifikasiWajah/VerifikasiSidikJari pattern).
	// Kalau verifier tidak available, return ErrBiometrikDibutuhkan
	// supaya frontend explicit handle (panggil method biometrik).
	if s.fp == nil || !s.fp.IsAvailable() {
		s.logger.Warn("sep: biometrik diperlukan & token belum disediakan frontend",
			"no_kartu_masked", maskID(peserta.NoKartu), "kd_poli", kdPoli)
		return "", domain.ErrBiometrikDibutuhkan
	}

	res, err := s.fp.Verify(ctx, peserta.NoKartu)
	if err != nil {
		s.logger.Warn("sep: verifikasi sidik jari gagal",
			"no_kartu_masked", maskID(peserta.NoKartu), "err", err.Error())
		return "", domain.ErrBiometrikDiperlukan
	}
	if !res.Success {
		return "", domain.ErrBiometrikDiperlukan
	}
	return res.Token, nil
}

// persistAndSyncKhanza menjalankan langkah 4-5 spec:
//
//	4. Insert ke print_history (backup + bahan reprint)
//	5. Post ke Khanza; kalau gagal/offline → insert ke pending_sep
//
// Tidak pernah mengembalikan error — SEP sudah issued di VClaim,
// kegagalan downstream di-log dan masuk pending_sep untuk reconcile.
func (s *SEPService) persistAndSyncKhanza(
	ctx context.Context,
	sepObj *domain.SEP,
	kategori string,
	originalReq any,
) {
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
