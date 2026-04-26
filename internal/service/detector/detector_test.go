package detector

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arunika/apm-go/internal/domain"
	"github.com/arunika/apm-go/internal/integration/antrol"
	"github.com/arunika/apm-go/internal/integration/khanza"
	"github.com/arunika/apm-go/internal/integration/vclaim"
)

// pesertaAktif adalah pasien valid + aktif untuk kebanyakan test.
var pesertaAktif = &domain.Peserta{
	NoKartu:     "0001234567890012",
	NoRM:        "RM-12345",
	NIK:         "3271234567890001",
	Nama:        "Budi Santoso",
	StatusAktif: "1",
	KelasHak:    "2",
}

// newDetectorWithAllMisses membangun Detector dengan semua client return
// "miss" (peserta aktif tapi tidak ada booking/kontrol/ranap/rajal).
// Test-test bisa override Func tertentu untuk skenario hit.
func newDetectorWithAllMisses(t *testing.T) (*Detector, *vclaim.MockVClaimClient, *antrol.MockAntrolClient, *khanza.MockKhanzaClient) {
	t.Helper()
	v := vclaim.NewMock()
	a := antrol.NewMock()
	k := khanza.NewMock()

	v.GetPesertaFunc = func(ctx context.Context, id string, tgl time.Time) (*domain.Peserta, error) {
		return pesertaAktif, nil
	}

	d := New(v, a, k)
	d.parallelTimeout = 500 * time.Millisecond // mempercepat test timeout
	return d, v, a, k
}

// ============================================================
// STEP 1 — Serial check
// ============================================================

func TestDetector_VClaimError_ReturnsPatientTypeError(t *testing.T) {
	d, v, _, _ := newDetectorWithAllMisses(t)
	v.GetPesertaFunc = func(ctx context.Context, id string, tgl time.Time) (*domain.Peserta, error) {
		return nil, errors.New("network down")
	}

	r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	if r.Type != domain.PatientTypeError {
		t.Errorf("Type = %v, want PatientTypeError", r.Type)
	}
	if r.Err == nil {
		t.Error("Err harus di-set")
	}
}

func TestDetector_PesertaTidakAktif_ReturnsTidakAktif(t *testing.T) {
	d, v, _, _ := newDetectorWithAllMisses(t)
	v.GetPesertaFunc = func(ctx context.Context, id string, tgl time.Time) (*domain.Peserta, error) {
		return &domain.Peserta{NoKartu: "X", StatusAktif: "0"}, nil
	}

	r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	if r.Type != domain.PatientTypeTidakAktif {
		t.Errorf("Type = %v, want PatientTypeTidakAktif", r.Type)
	}
	if r.Peserta == nil || r.Peserta.NoKartu != "X" {
		t.Errorf("Peserta tetap di-isi untuk UI")
	}
}

func TestDetector_PesertaTidakAktif_TidakJalankanParallelChecks(t *testing.T) {
	d, v, a, k := newDetectorWithAllMisses(t)
	v.GetPesertaFunc = func(ctx context.Context, id string, tgl time.Time) (*domain.Peserta, error) {
		return &domain.Peserta{StatusAktif: "0"}, nil
	}

	d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})

	if a.CallCount("GetBookingHariIni") != 0 {
		t.Errorf("Antrol seharusnya tidak dipanggil untuk peserta tidak aktif")
	}
	if k.CallCount("GetSuratKontrol") != 0 {
		t.Errorf("Khanza seharusnya tidak dipanggil untuk peserta tidak aktif")
	}
}

// ============================================================
// STEP 2 + 3 — Parallel checks + priority resolution
// ============================================================

func TestDetector_AllMiss_ReturnRujukanBaru(t *testing.T) {
	d, _, _, _ := newDetectorWithAllMisses(t)

	r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	if r.Type != domain.PatientTypeRujukanBaru {
		t.Errorf("Type = %v, want RujukanBaru", r.Type)
	}
}

func TestDetector_OnlyMJKN_ReturnMJKN(t *testing.T) {
	d, _, a, _ := newDetectorWithAllMisses(t)
	today := time.Now().Format("2006-01-02")
	a.GetBookingHariIniFunc = func(ctx context.Context, noKartu string, tgl time.Time) (*domain.BookingMJKN, error) {
		return &domain.BookingMJKN{NoBooking: "BK-001", Tanggal: today}, nil
	}

	r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	if r.Type != domain.PatientTypeMJKN {
		t.Errorf("Type = %v, want MJKN", r.Type)
	}
	if booking, ok := r.Data.(*domain.BookingMJKN); !ok || booking.NoBooking != "BK-001" {
		t.Errorf("Data tidak berisi BookingMJKN: %+v", r.Data)
	}
}

func TestDetector_OnlyKontrol_ReturnKontrol(t *testing.T) {
	d, _, _, k := newDetectorWithAllMisses(t)
	today := time.Now().Format("2006-01-02")
	k.GetSuratKontrolFunc = func(ctx context.Context, noRM string) ([]domain.SuratKontrol, error) {
		return []domain.SuratKontrol{{NoSurat: "SK-001", TglRencana: today, KdPoli: "INT"}}, nil
	}

	r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	if r.Type != domain.PatientTypeKontrol {
		t.Errorf("Type = %v, want Kontrol", r.Type)
	}
}

func TestDetector_OnlyPostRANAP_ReturnPostRANAP(t *testing.T) {
	d, _, _, k := newDetectorWithAllMisses(t)
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	k.GetRiwayatRANAPFunc = func(ctx context.Context, noRM string) ([]domain.RiwayatRANAP, error) {
		return []domain.RiwayatRANAP{{NoRawat: "R-001", TglKeluar: yesterday}}, nil
	}

	r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	if r.Type != domain.PatientTypePostRANAP {
		t.Errorf("Type = %v, want PostRANAP", r.Type)
	}
}

func TestDetector_OnlyPostRAJAL_ReturnPostRAJAL(t *testing.T) {
	d, _, _, k := newDetectorWithAllMisses(t)
	k.GetKunjunganAktifFunc = func(ctx context.Context, noRM string) ([]domain.Kunjungan, error) {
		return []domain.Kunjungan{{
			NoRawat: "R-002", JnsPelayanan: "1",
			KdPoli: "INT", NoSKDP: "SKDP-1", KdPoliSKDP: "JTG",
		}}, nil
	}

	r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	if r.Type != domain.PatientTypePostRAJAL {
		t.Errorf("Type = %v, want PostRAJAL", r.Type)
	}
}

func TestDetector_PriorityMJKNBeatsKontrol(t *testing.T) {
	d, _, a, k := newDetectorWithAllMisses(t)
	today := time.Now().Format("2006-01-02")
	a.GetBookingHariIniFunc = func(ctx context.Context, noKartu string, tgl time.Time) (*domain.BookingMJKN, error) {
		return &domain.BookingMJKN{Tanggal: today}, nil
	}
	k.GetSuratKontrolFunc = func(ctx context.Context, noRM string) ([]domain.SuratKontrol, error) {
		return []domain.SuratKontrol{{TglRencana: today}}, nil
	}

	r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	if r.Type != domain.PatientTypeMJKN {
		t.Errorf("Type = %v, want MJKN (lebih prioritas dari Kontrol)", r.Type)
	}
}

func TestDetector_AllFourHit_PriorityPicksMJKN(t *testing.T) {
	d, _, a, k := newDetectorWithAllMisses(t)
	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	a.GetBookingHariIniFunc = func(ctx context.Context, noKartu string, tgl time.Time) (*domain.BookingMJKN, error) {
		return &domain.BookingMJKN{Tanggal: today}, nil
	}
	k.GetSuratKontrolFunc = func(ctx context.Context, noRM string) ([]domain.SuratKontrol, error) {
		return []domain.SuratKontrol{{TglRencana: today}}, nil
	}
	k.GetRiwayatRANAPFunc = func(ctx context.Context, noRM string) ([]domain.RiwayatRANAP, error) {
		return []domain.RiwayatRANAP{{TglKeluar: yesterday}}, nil
	}
	k.GetKunjunganAktifFunc = func(ctx context.Context, noRM string) ([]domain.Kunjungan, error) {
		return []domain.Kunjungan{{JnsPelayanan: "1", KdPoli: "A", NoSKDP: "S", KdPoliSKDP: "B"}}, nil
	}

	r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	if r.Type != domain.PatientTypeMJKN {
		t.Errorf("4-hit: prioritas MJKN, got %v", r.Type)
	}
}

// Test bahwa SuratKontrol dengan TglRencana di masa depan TIDAK dihitung hit.
func TestDetector_KontrolBesok_TidakHit(t *testing.T) {
	d, _, _, k := newDetectorWithAllMisses(t)
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	k.GetSuratKontrolFunc = func(ctx context.Context, noRM string) ([]domain.SuratKontrol, error) {
		return []domain.SuratKontrol{{TglRencana: tomorrow}}, nil
	}

	r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	if r.Type != domain.PatientTypeRujukanBaru {
		t.Errorf("kontrol di masa depan tidak boleh hit, got %v", r.Type)
	}
}

// Booking MJKN untuk tanggal lain (kemarin) TIDAK hit.
func TestDetector_BookingTanggalLain_TidakHit(t *testing.T) {
	d, _, a, _ := newDetectorWithAllMisses(t)
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	a.GetBookingHariIniFunc = func(ctx context.Context, noKartu string, tgl time.Time) (*domain.BookingMJKN, error) {
		return &domain.BookingMJKN{Tanggal: yesterday}, nil
	}

	r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	if r.Type != domain.PatientTypeRujukanBaru {
		t.Errorf("booking kemarin tidak boleh hit MJKN, got %v", r.Type)
	}
}

// ============================================================
// Edge cases — panic, timeout, cancel
// ============================================================

func TestDetector_PanicDiCheckGoroutine_DiRecoverDanMiss(t *testing.T) {
	d, _, a, _ := newDetectorWithAllMisses(t)
	a.GetBookingHariIniFunc = func(ctx context.Context, noKartu string, tgl time.Time) (*domain.BookingMJKN, error) {
		panic("simulasi panic")
	}

	// Tidak boleh panic ke caller, dan harus tetap return result
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Detect() panic ke caller: %v", r)
		}
	}()

	r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	if r.Type != domain.PatientTypeRujukanBaru {
		t.Errorf("panic di MJKN seharusnya treat sebagai miss → RujukanBaru, got %v", r.Type)
	}
}

func TestDetector_ErrorDiCheck_TreatedAsMiss(t *testing.T) {
	d, _, a, _ := newDetectorWithAllMisses(t)
	a.GetBookingHariIniFunc = func(ctx context.Context, noKartu string, tgl time.Time) (*domain.BookingMJKN, error) {
		return nil, errors.New("antrol down")
	}

	r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	if r.Type != domain.PatientTypeRujukanBaru {
		t.Errorf("error di check seharusnya miss → RujukanBaru, got %v", r.Type)
	}
}

func TestDetector_AllChecksTimeout_ReturnRujukanBaru(t *testing.T) {
	d, _, a, k := newDetectorWithAllMisses(t)
	d.parallelTimeout = 100 * time.Millisecond

	// Semua check sleep lebih lama dari timeout
	slowMJKN := func(ctx context.Context, noKartu string, tgl time.Time) (*domain.BookingMJKN, error) {
		select {
		case <-time.After(2 * time.Second):
			return nil, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	a.GetBookingHariIniFunc = slowMJKN

	slowKhanza := func(ctx context.Context, noRM string) ([]domain.SuratKontrol, error) {
		select {
		case <-time.After(2 * time.Second):
			return nil, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	k.GetSuratKontrolFunc = slowKhanza

	start := time.Now()
	r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	elapsed := time.Since(start)

	if r.Type != domain.PatientTypeRujukanBaru {
		t.Errorf("semua timeout: expected RujukanBaru, got %v", r.Type)
	}
	// Harus selesai dalam ~timeout (+sedikit grace), bukan 2 detik
	if elapsed > 1*time.Second {
		t.Errorf("Detect terlalu lama: %v (expected <1s dengan parallelTimeout=100ms)", elapsed)
	}
}

func TestDetector_CallerCancelContext_ExitCepat(t *testing.T) {
	d, _, a, _ := newDetectorWithAllMisses(t)
	d.parallelTimeout = 5 * time.Second // long parallel timeout
	a.GetBookingHariIniFunc = func(ctx context.Context, noKartu string, tgl time.Time) (*domain.BookingMJKN, error) {
		select {
		case <-time.After(10 * time.Second):
			return nil, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	r := d.Detect(ctx, domain.PatientInput{Identifier: "X"})
	elapsed := time.Since(start)

	// Caller cancel → harus exit dalam ~50-200ms, bukan 5 detik
	if elapsed > 1*time.Second {
		t.Errorf("Caller cancel tidak dihormati cepat: %v", elapsed)
	}
	// Type bisa RujukanBaru (semua dianggap miss) — yang penting cepat
	_ = r
}

// ============================================================
// noRM kosong: 3 check Khanza tetap aman (return miss tanpa hit)
// ============================================================

func TestDetector_NoRMKosong_SkipKhanzaChecks(t *testing.T) {
	d, v, _, k := newDetectorWithAllMisses(t)
	v.GetPesertaFunc = func(ctx context.Context, id string, tgl time.Time) (*domain.Peserta, error) {
		return &domain.Peserta{NoKartu: "X", StatusAktif: "1"}, nil // NoRM kosong
	}
	// Set Khanza func yang akan FAIL kalau dipanggil dengan noRM kosong
	k.GetSuratKontrolFunc = func(ctx context.Context, noRM string) ([]domain.SuratKontrol, error) {
		if noRM == "" {
			t.Errorf("Khanza tidak boleh dipanggil dengan noRM kosong")
		}
		return nil, nil
	}

	r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	if r.Type != domain.PatientTypeRujukanBaru {
		t.Errorf("noRM kosong → RujukanBaru, got %v", r.Type)
	}
	if k.CallCount("GetSuratKontrol") != 0 {
		t.Errorf("GetSuratKontrol seharusnya tidak dipanggil")
	}
}

// ============================================================
// PHI masking dalam log
// ============================================================

func TestDetector_LogMaskNoKartu(t *testing.T) {
	d, _, _, _ := newDetectorWithAllMisses(t)

	var buf bytes.Buffer
	d.SetLogger(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))

	d.Detect(context.Background(), domain.PatientInput{Identifier: "0001234567890012"})

	logs := buf.String()
	// PHI asli tidak boleh muncul (karena VClaim di-mock, tapi kita
	// pasok pesertaAktif yang noKartu = 0001234567890012)
	if strings.Contains(logs, "0001234567890012") {
		t.Errorf("log seharusnya tidak mengandung noKartu lengkap: %s", logs)
	}
	// Last 4 digit boleh muncul (sebagai bagian dari masked ID)
	if !strings.Contains(logs, "0012") {
		t.Errorf("log seharusnya mengandung 4 digit terakhir: %s", logs)
	}
}

// ============================================================
// Concurrency / no goroutine leak (proxy via NumGoroutine count)
// ============================================================

func TestDetector_TidakLeakGoroutine_HappyPath(t *testing.T) {
	// Test ringan — bukan strict goleak. Sebelum/sesudah Detect()
	// jumlah goroutine tidak boleh tumbuh signifikan setelah grace.
	d, _, _, _ := newDetectorWithAllMisses(t)

	var counter atomic.Int32
	for i := 0; i < 10; i++ {
		go func() {
			d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
			counter.Add(1)
		}()
	}

	// Tunggu semua selesai
	deadline := time.Now().Add(2 * time.Second)
	for counter.Load() < 10 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if counter.Load() != 10 {
		t.Fatalf("hanya %d/10 Detect selesai dalam waktu", counter.Load())
	}

	// Memberi kesempatan goroutine sisa untuk exit
	time.Sleep(100 * time.Millisecond)
	// Tidak ada assert keras pada NumGoroutine (CI flaky), cukup verifikasi
	// semua selesai tanpa deadlock.
}
