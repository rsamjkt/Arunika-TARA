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

	// Step 1: biometrik kalau perlu
	fpToken, err := s.maybeBiometrik(ctx, *peserta, req.KdPoli)
	if err != nil {
		return nil, err // ErrBiometrikDiperlukan
	}
	req.FPToken = fpToken

	// Step 2: validasi rujukan
	tglSEP := parseDateOrNow(req.TglSEP, s.now())
	if _, err := s.vclaim.ValidasiRujukan(ctx, req.NoRujukan, tglSEP); err != nil {
		return nil, fmt.Errorf("validasi rujukan: %w", err)
	}

	// Step 3: issue SEP via VClaim
	sepObj, err := s.vclaim.CreateSEP(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vclaim CreateSEP: %w", err)
	}

	// Step 4-5: persist + post ke Khanza
	s.persistAndSyncKhanza(ctx, sepObj, "RUJUKAN", req)
	return sepObj, nil
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

	// Biometrik kalau perlu (poli kontrol)
	fpToken, err := s.maybeBiometrik(ctx, *peserta, sk.KdPoli)
	if err != nil {
		return nil, err
	}

	// Step 3: build request + issue
	tglStr := s.now().Format("2006-01-02")
	req := domain.SEPKontrolRequest{
		NoSuratKontrol: noSuratKontrol,
		NoKartu:        peserta.NoKartu,
		TglSEP:         tglStr,
		KdDokter:       firstNonEmpty(kdDokter, sk.KdDokter),
		KelasRawat:     peserta.KelasHak,
		JnsPelayanan:   "1", // Rawat Jalan
		FPToken:        fpToken,
	}
	sepObj, err := s.vclaim.CreateSEPKontrol(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vclaim CreateSEPKontrol: %w", err)
	}

	// Step 4-5: persist
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

	fpToken, err := s.maybeBiometrik(ctx, *peserta, kdPoli)
	if err != nil {
		return nil, err
	}

	tglStr := s.now().Format("2006-01-02")
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
		return nil, fmt.Errorf("vclaim CreateSEP %s: %w", label, err)
	}

	s.persistAndSyncKhanza(ctx, sepObj, kategori, req)
	return sepObj, nil
}

// ============================================================
// Helpers
// ============================================================

// maybeBiometrik menjalankan fingerprint.Verify() bila perluBiometrik()
// return true. Mengembalikan token (kosong kalau tidak perlu biometrik
// atau verifier tidak available) atau ErrBiometrikDiperlukan kalau
// verifikasi gagal.
//
// Behavior degradasi: kalau verifier tidak available (mis. After.exe
// crash), TIDAK gagalkan SEP — log warning + lanjut tanpa token.
// Operator BPJS akan reject SEP saat klaim kalau token kurang, tapi
// itu masalah audit yang lebih baik daripada blokir pasien di kiosk.
func (s *SEPService) maybeBiometrik(ctx context.Context, peserta domain.Peserta, kdPoli string) (string, error) {
	if !perluBiometrik(peserta, kdPoli) {
		return "", nil
	}
	if s.fp == nil || !s.fp.IsAvailable() {
		s.logger.Warn("sep: biometrik diperlukan tapi verifier tidak available",
			"no_kartu_masked", maskID(peserta.NoKartu), "kd_poli", kdPoli)
		return "", nil
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
