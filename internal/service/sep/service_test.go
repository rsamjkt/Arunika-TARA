package sep

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/arunika/apm-go/internal/domain"
	"github.com/arunika/apm-go/internal/hardware/fingerprint"
	"github.com/arunika/apm-go/internal/integration/khanza"
	"github.com/arunika/apm-go/internal/integration/vclaim"
	"github.com/arunika/apm-go/internal/store"
)

// loadSchema baca migration relatif ke project root.
func loadSchema(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("tidak bisa resolve path test")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "..")
	b, err := os.ReadFile(filepath.Join(root, "migrations", "001_initial.sql"))
	if err != nil {
		t.Fatalf("baca schema: %v", err)
	}
	return string(b)
}

type mocks struct {
	v  *vclaim.MockVClaimClient
	k  *khanza.MockKhanzaClient
	fp *fingerprint.MockVerifier
	db *sql.DB
}

// setupSEP membangun SEPService dengan mock semua dependency + in-memory SQLite.
func setupSEP(t *testing.T) (*SEPService, *mocks) {
	t.Helper()
	db, _, err := store.Open(context.Background(), ":memory:", loadSchema(t))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	v := vclaim.NewMock()
	k := khanza.NewMock()
	fp := fingerprint.NewMock()
	fp.SetScanDelay(0) // test cepat

	return New(v, k, fp, db), &mocks{v: v, k: k, fp: fp, db: db}
}

// pesertaDewasa: 30 tahun (jelas wajib biometrik untuk poli non-IGD).
func pesertaDewasa() *domain.Peserta {
	return &domain.Peserta{
		NoKartu:     "0001234567890012",
		NoRM:        "RM-001",
		NIK:         "3271234567890001",
		Nama:        "Budi Santoso",
		TglLahir:    time.Now().AddDate(-30, 0, 0).Format("2006-01-02"),
		StatusAktif: "1",
		KelasHak:    "2",
	}
}

// pesertaAnak: 8 tahun (tidak wajib biometrik).
func pesertaAnak() *domain.Peserta {
	return &domain.Peserta{
		NoKartu:     "0009999",
		NoRM:        "RM-009",
		Nama:        "Anak Test",
		TglLahir:    time.Now().AddDate(-8, 0, 0).Format("2006-01-02"),
		StatusAktif: "1",
		KelasHak:    "3",
	}
}

// stubVClaimSuccess: bikin v.CreateSEP / CreateSEPKontrol return SEP sukses.
func stubVClaimSuccess(m *mocks, noSEP string) {
	m.v.CreateSEPFunc = func(ctx context.Context, req domain.SEPRequest) (*domain.SEP, error) {
		return &domain.SEP{
			NoSEP: noSEP, NoKartu: req.NoKartu, TglSEP: req.TglSEP,
			KdPoli: req.KdPoli, KdDokter: req.KdDokter, NmPoli: "Poli " + req.KdPoli,
			NmDokter: "dr. " + req.KdDokter, CreatedAt: time.Now(),
		}, nil
	}
	m.v.CreateSEPKontrolFunc = func(ctx context.Context, req domain.SEPKontrolRequest) (*domain.SEP, error) {
		return &domain.SEP{
			NoSEP: noSEP, NoKartu: req.NoKartu, TglSEP: req.TglSEP,
			KdDokter: req.KdDokter, NmDokter: "dr. " + req.KdDokter, CreatedAt: time.Now(),
		}, nil
	}
	m.v.ValidasiRujukanFunc = func(ctx context.Context, noSurat string, tgl time.Time) (*domain.Rujukan, error) {
		return &domain.Rujukan{NoSurat: noSurat, TglRujukan: time.Now().AddDate(0, -1, 0).Format("2006-01-02")}, nil
	}
}

// ============================================================
// BuatSEPRujukan
// ============================================================

func TestBuatSEPRujukan_HappyPath_Biometrik(t *testing.T) {
	svc, m := setupSEP(t)
	stubVClaimSuccess(m, "SEP-001")

	got, err := svc.BuatSEPRujukan(context.Background(), pesertaDewasa(), domain.SEPRequest{
		NoRujukan:    "RUJ-001",
		KdPoli:       "INT",
		KdDokter:     "D-001",
		JnsPelayanan: "1",
		KelasRawat:   "2",
	})
	if err != nil {
		t.Fatalf("BuatSEPRujukan: %v", err)
	}
	if got.NoSEP != "SEP-001" {
		t.Errorf("NoSEP = %q", got.NoSEP)
	}
	if m.fp.IsAvailable() && m.v.CallCount("CreateSEP") != 1 {
		t.Errorf("CreateSEP harus dipanggil 1x")
	}
	if m.k.CallCount("SimpanSEP") != 1 {
		t.Errorf("SimpanSEP harus dipanggil 1x setelah sukses")
	}
}

func TestBuatSEPRujukan_PesertaAnak_TanpaBiometrik(t *testing.T) {
	svc, m := setupSEP(t)
	stubVClaimSuccess(m, "SEP-002")

	// Set fp Verify untuk fail kalau dipanggil — verifies anak tidak panggil
	m.fp.SetNextFail()

	_, err := svc.BuatSEPRujukan(context.Background(), pesertaAnak(), domain.SEPRequest{
		NoRujukan: "RUJ-001", KdPoli: "ANAK", KdDokter: "D-001",
	})
	if err != nil {
		t.Fatalf("anak tidak boleh dipanggil biometrik, got err: %v", err)
	}
}

func TestBuatSEPRujukan_BiometrikGagal_ReturnErrBiometrikDiperlukan(t *testing.T) {
	svc, m := setupSEP(t)
	stubVClaimSuccess(m, "SEP-FAIL")
	m.fp.SetNextFail() // dewasa + non-IGD → wajib biometrik → fail

	_, err := svc.BuatSEPRujukan(context.Background(), pesertaDewasa(), domain.SEPRequest{
		NoRujukan: "RUJ-001", KdPoli: "INT", KdDokter: "D-001",
	})
	if err == nil {
		t.Fatal("expected error karena biometrik gagal")
	}
	if !errors.Is(err, domain.ErrBiometrikDiperlukan) {
		t.Errorf("err harus wrap ErrBiometrikDiperlukan, got: %v", err)
	}
	// VClaim CreateSEP TIDAK boleh dipanggil kalau biometrik gagal
	if m.v.CallCount("CreateSEP") != 0 {
		t.Errorf("CreateSEP tidak boleh dipanggil saat biometrik gagal")
	}
}

func TestBuatSEPRujukan_NoRujukanKosong_Error(t *testing.T) {
	svc, _ := setupSEP(t)
	_, err := svc.BuatSEPRujukan(context.Background(), pesertaDewasa(), domain.SEPRequest{
		KdPoli: "INT", KdDokter: "D-001",
	})
	if err == nil {
		t.Fatal("NoRujukan kosong harus error")
	}
}

func TestBuatSEPRujukan_PesertaNil_Error(t *testing.T) {
	svc, _ := setupSEP(t)
	_, err := svc.BuatSEPRujukan(context.Background(), nil, domain.SEPRequest{NoRujukan: "X"})
	if err == nil {
		t.Fatal("peserta nil harus error")
	}
}

func TestBuatSEPRujukan_RujukanInvalid_Error(t *testing.T) {
	svc, m := setupSEP(t)
	m.v.ValidasiRujukanFunc = func(ctx context.Context, noSurat string, tgl time.Time) (*domain.Rujukan, error) {
		return nil, domain.ErrRujukanExpired
	}

	_, err := svc.BuatSEPRujukan(context.Background(), pesertaAnak(), domain.SEPRequest{
		NoRujukan: "RUJ-EXP", KdPoli: "ANAK",
	})
	if err == nil {
		t.Fatal("rujukan expired harus error")
	}
	if !errors.Is(err, domain.ErrRujukanExpired) {
		t.Errorf("err harus wrap ErrRujukanExpired, got: %v", err)
	}
}

// ============================================================
// Khanza offline → pending_sep
// ============================================================

func TestBuatSEPRujukan_KhanzaOffline_DisimpanKePendingSEP(t *testing.T) {
	svc, m := setupSEP(t)
	stubVClaimSuccess(m, "SEP-OFF")
	m.k.SetResponse("SimpanSEP", nil, domain.ErrOffline)

	got, err := svc.BuatSEPRujukan(context.Background(), pesertaAnak(), domain.SEPRequest{
		NoRujukan: "RUJ-001", KdPoli: "ANAK", KdDokter: "D",
	})
	if err != nil {
		t.Fatalf("Khanza offline tidak boleh gagalkan SEP, got err: %v", err)
	}
	if got.NoSEP != "SEP-OFF" {
		t.Errorf("SEP tetap di-return ke caller")
	}

	// Verifikasi pending_sep ada entry
	q := store.New(m.db)
	pending, err := q.GetPendingSEPs(context.Background(), store.GetPendingSEPsParams{
		Status: sql.NullString{String: "pending", Valid: true},
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("GetPendingSEPs: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 entry pending_sep, got %d", len(pending))
	}
	if pending[0].Kategori != "RUJUKAN" {
		t.Errorf("kategori = %q, want RUJUKAN", pending[0].Kategori)
	}
}

func TestBuatSEPRujukan_KhanzaErrorNonOffline_TetapDisimpanKePendingSEP(t *testing.T) {
	svc, m := setupSEP(t)
	stubVClaimSuccess(m, "SEP-ERR")
	m.k.SetResponse("SimpanSEP", nil, errors.New("khanza 500 internal"))

	got, err := svc.BuatSEPRujukan(context.Background(), pesertaAnak(), domain.SEPRequest{
		NoRujukan: "RUJ-001", KdPoli: "ANAK", KdDokter: "D",
	})
	if err != nil {
		t.Fatalf("Khanza error non-offline tidak boleh gagalkan SEP, got: %v", err)
	}
	if got.NoSEP != "SEP-ERR" {
		t.Errorf("SEP harus tetap di-return")
	}

	q := store.New(m.db)
	pending, _ := q.GetPendingSEPs(context.Background(), store.GetPendingSEPsParams{
		Status: sql.NullString{String: "pending", Valid: true},
		Limit:  10,
	})
	if len(pending) != 1 {
		t.Fatalf("expected pending_sep entry untuk non-offline error, got %d", len(pending))
	}
}

// ============================================================
// VClaim error → propagate (SEP belum issued)
// ============================================================

func TestBuatSEPRujukan_VClaimError_Propagate(t *testing.T) {
	svc, m := setupSEP(t)
	m.v.ValidasiRujukanFunc = func(ctx context.Context, noSurat string, tgl time.Time) (*domain.Rujukan, error) {
		return &domain.Rujukan{NoSurat: noSurat}, nil
	}
	m.v.CreateSEPFunc = func(ctx context.Context, req domain.SEPRequest) (*domain.SEP, error) {
		return nil, domain.ErrDuplikasiSEP
	}

	_, err := svc.BuatSEPRujukan(context.Background(), pesertaAnak(), domain.SEPRequest{
		NoRujukan: "RUJ-001", KdPoli: "ANAK",
	})
	if err == nil {
		t.Fatal("VClaim error harus propagate")
	}
	if !errors.Is(err, domain.ErrDuplikasiSEP) {
		t.Errorf("err harus wrap ErrDuplikasiSEP, got: %v", err)
	}

	// Khanza tidak boleh dipanggil — SEP tidak pernah ter-issue
	if m.k.CallCount("SimpanSEP") != 0 {
		t.Errorf("SimpanSEP tidak boleh dipanggil saat CreateSEP gagal")
	}
}

// ============================================================
// BuatSEPKontrol
// ============================================================

func TestBuatSEPKontrol_HappyPath(t *testing.T) {
	svc, m := setupSEP(t)
	stubVClaimSuccess(m, "SEP-K-001")
	today := time.Now().Format("2006-01-02")
	m.k.GetSuratKontrolFunc = func(ctx context.Context, noRM string) ([]domain.SuratKontrol, error) {
		return []domain.SuratKontrol{
			{NoSurat: "SK-001", NoRM: noRM, TglRencana: today, KdPoli: "INT", KdDokter: "D-001"},
		}, nil
	}

	got, err := svc.BuatSEPKontrol(context.Background(), pesertaAnak(), "SK-001", "")
	if err != nil {
		t.Fatalf("BuatSEPKontrol: %v", err)
	}
	if got.NoSEP != "SEP-K-001" {
		t.Errorf("NoSEP = %q", got.NoSEP)
	}
}

func TestBuatSEPKontrol_SuratTidakDitemukan(t *testing.T) {
	svc, m := setupSEP(t)
	m.k.GetSuratKontrolFunc = func(ctx context.Context, noRM string) ([]domain.SuratKontrol, error) {
		return []domain.SuratKontrol{}, nil // empty
	}

	_, err := svc.BuatSEPKontrol(context.Background(), pesertaAnak(), "SK-NOT-EXIST", "")
	if !errors.Is(err, domain.ErrSuratKontrolTidakDitemukan) {
		t.Errorf("err harus wrap ErrSuratKontrolTidakDitemukan, got: %v", err)
	}
}

func TestBuatSEPKontrol_JadwalBelumTiba_PakaiErrJadwalKontrolBelumTiba(t *testing.T) {
	svc, m := setupSEP(t)
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	m.k.GetSuratKontrolFunc = func(ctx context.Context, noRM string) ([]domain.SuratKontrol, error) {
		return []domain.SuratKontrol{
			{NoSurat: "SK-FUTURO", TglRencana: tomorrow, KdPoli: "INT"},
		}, nil
	}

	_, err := svc.BuatSEPKontrol(context.Background(), pesertaAnak(), "SK-FUTURO", "")
	if err == nil {
		t.Fatal("expected error untuk jadwal besok")
	}
	if !domain.IsErrJadwalKontrolBelumTiba(err) {
		t.Errorf("err harus IsErrJadwalKontrolBelumTiba=true, got: %v", err)
	}
	// Pesan harus menyertakan tanggal
	if !contains(err.Error(), tomorrow) {
		t.Errorf("pesan err tidak menyertakan TglRencana %q: %v", tomorrow, err)
	}
}

// ============================================================
// BuatSEPPostRANAP / BuatSEPPostRAJAL
// ============================================================

func TestBuatSEPPostRANAP_HappyPath(t *testing.T) {
	svc, m := setupSEP(t)
	stubVClaimSuccess(m, "SEP-RANAP")

	got, err := svc.BuatSEPPostRANAP(context.Background(), pesertaAnak(), "INT", "D-001")
	if err != nil {
		t.Fatalf("BuatSEPPostRANAP: %v", err)
	}
	if got.NoSEP != "SEP-RANAP" {
		t.Errorf("NoSEP = %q", got.NoSEP)
	}
	if m.v.CallCount("CreateSEP") != 1 {
		t.Errorf("CreateSEP harus dipanggil 1x untuk POST-RANAP")
	}
}

func TestBuatSEPPostRAJAL_HappyPath(t *testing.T) {
	svc, m := setupSEP(t)
	stubVClaimSuccess(m, "SEP-RAJAL")

	got, err := svc.BuatSEPPostRAJAL(context.Background(), pesertaAnak(), "JTG", "D-002")
	if err != nil {
		t.Fatalf("BuatSEPPostRAJAL: %v", err)
	}
	if got.NoSEP != "SEP-RAJAL" {
		t.Errorf("NoSEP = %q", got.NoSEP)
	}
}

func TestBuatSEPPostRANAP_KdPoliKosong_Error(t *testing.T) {
	svc, _ := setupSEP(t)
	_, err := svc.BuatSEPPostRANAP(context.Background(), pesertaAnak(), "", "D")
	if err == nil {
		t.Fatal("kdPoli kosong harus error")
	}
}

// ============================================================
// FP not available → SEP tetap issued (degradasi)
// ============================================================

func TestSEPService_FPVerifierTidakAvailable_SEPTetapIssue(t *testing.T) {
	svc, m := setupSEP(t)
	stubVClaimSuccess(m, "SEP-NO-FP")
	m.fp.SetAvailable(false)

	got, err := svc.BuatSEPRujukan(context.Background(), pesertaDewasa(), domain.SEPRequest{
		NoRujukan: "RUJ-001", KdPoli: "INT", KdDokter: "D",
	})
	if err != nil {
		t.Fatalf("FP unavailable tidak boleh blokir SEP, got: %v", err)
	}
	if got.NoSEP != "SEP-NO-FP" {
		t.Errorf("SEP harus tetap di-issue")
	}
}

// ============================================================
// print_history persisted
// ============================================================

func TestSEPService_InsertPrintHistory(t *testing.T) {
	svc, m := setupSEP(t)
	stubVClaimSuccess(m, "SEP-PH-001")

	_, err := svc.BuatSEPRujukan(context.Background(), pesertaAnak(), domain.SEPRequest{
		NoRujukan: "RUJ-001", KdPoli: "ANAK", KdDokter: "D",
	})
	if err != nil {
		t.Fatalf("BuatSEPRujukan: %v", err)
	}

	q := store.New(m.db)
	got, err := q.GetPrintHistoryByRefID(context.Background(), store.GetPrintHistoryByRefIDParams{
		DocType: "SEP",
		RefID:   sql.NullString{String: "SEP-PH-001", Valid: true},
	})
	if err != nil {
		t.Fatalf("GetPrintHistoryByRefID: %v", err)
	}
	if got.DocType != "SEP" {
		t.Errorf("doc_type = %q", got.DocType)
	}
	if len(got.EscposBytes) == 0 {
		t.Errorf("escpos_bytes harus ada (placeholder JSON sampai P-033)")
	}
}

// ============================================================
// Helpers
// ============================================================

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub ||
		(len(sub) > 0 && (s[:len(sub)] == sub || s[len(s)-len(sub):] == sub ||
			indexOf(s, sub) >= 0)))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
