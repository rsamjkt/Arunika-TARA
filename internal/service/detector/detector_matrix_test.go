package detector

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arunika/apm-go/internal/domain"
	"github.com/arunika/apm-go/internal/integration/antrol"
	"github.com/arunika/apm-go/internal/integration/khanza"
	"github.com/arunika/apm-go/internal/integration/vclaim"
)

// ============================================================
// Helper: build Detector dengan kombinasi hit yang dapat dikonfigurasi.
// Pakai untuk semua test matrix di file ini.
// ============================================================

type checkHits struct {
	mjkn    bool
	kontrol bool
	ranap   bool
	rajal   bool
}

// detectorWithHits membangun Detector di mana setiap check return hit/miss
// sesuai flag. Mock peserta selalu aktif.
func detectorWithHits(t *testing.T, h checkHits) *Detector {
	t.Helper()
	v := vclaim.NewMock()
	a := antrol.NewMock()
	k := khanza.NewMock()

	v.GetPesertaFunc = func(ctx context.Context, id string, tgl time.Time) (*domain.Peserta, error) {
		return pesertaAktif, nil
	}

	today := time.Now().Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	a.GetBookingHariIniFunc = func(ctx context.Context, noKartu string, tgl time.Time) (*domain.BookingMJKN, error) {
		if h.mjkn {
			return &domain.BookingMJKN{NoBooking: "BK-MJKN", Tanggal: today}, nil
		}
		return nil, nil
	}
	k.GetSuratKontrolFunc = func(ctx context.Context, noRM string) ([]domain.SuratKontrol, error) {
		if h.kontrol {
			return []domain.SuratKontrol{{NoSurat: "SK-001", TglRencana: today, KdPoli: "INT"}}, nil
		}
		return nil, nil
	}
	k.GetRiwayatRANAPFunc = func(ctx context.Context, noRM string) ([]domain.RiwayatRANAP, error) {
		if h.ranap {
			return []domain.RiwayatRANAP{{NoRawat: "R-RANAP", TglKeluar: yesterday}}, nil
		}
		return nil, nil
	}
	k.GetKunjunganAktifFunc = func(ctx context.Context, noRM string) ([]domain.Kunjungan, error) {
		if h.rajal {
			return []domain.Kunjungan{{
				NoRawat: "R-RAJAL", JnsPelayanan: "1",
				KdPoli: "INT", NoSKDP: "SKDP-1", KdPoliSKDP: "JTG",
			}}, nil
		}
		return nil, nil
	}

	d := New(v, a, k)
	d.parallelTimeout = 500 * time.Millisecond
	return d
}

// ============================================================
// Matrix: happy paths + priority resolution (gabungan, sesuai spec)
// ============================================================

func TestDetectorMatrix_HappyAndPriority(t *testing.T) {
	cases := []struct {
		name     string
		hits     checkHits
		expected domain.PatientType
	}{
		// ── Happy paths (single hit per kategori) ──
		{"only_mjkn", checkHits{mjkn: true}, domain.PatientTypeMJKN},
		{"only_kontrol", checkHits{kontrol: true}, domain.PatientTypeKontrol},
		{"only_ranap", checkHits{ranap: true}, domain.PatientTypePostRANAP},
		{"only_rajal", checkHits{rajal: true}, domain.PatientTypePostRAJAL},
		{"none_hit", checkHits{}, domain.PatientTypeRujukanBaru},

		// ── Priority resolution ──
		{"mjkn_beats_all", checkHits{mjkn: true, kontrol: true, ranap: true, rajal: true}, domain.PatientTypeMJKN},
		{"kontrol_beats_ranap", checkHits{kontrol: true, ranap: true, rajal: true}, domain.PatientTypeKontrol},
		{"ranap_beats_rajal", checkHits{ranap: true, rajal: true}, domain.PatientTypePostRANAP},

		// ── Tambahan kombinasi pasangan ──
		{"mjkn_dan_rajal_pick_mjkn", checkHits{mjkn: true, rajal: true}, domain.PatientTypeMJKN},
		{"kontrol_dan_rajal_pick_kontrol", checkHits{kontrol: true, rajal: true}, domain.PatientTypeKontrol},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := detectorWithHits(t, tc.hits)
			r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
			if r.Type != tc.expected {
				t.Errorf("matrix [%s] hits=%+v: got %v, want %v",
					tc.name, tc.hits, r.Type, tc.expected)
			}
			// IsSuccess() WAJIB true untuk semua kategori valid (semua di matrix
			// adalah kategori valid — tidak ada Error/Unknown).
			if !r.IsSuccess() {
				t.Errorf("[%s] IsSuccess() = false, want true", tc.name)
			}
		})
	}
}

// ============================================================
// Partial failure: 2 dari 4 check error
// ============================================================

func TestDetectorMatrix_PartialFailure_TanpaHit_ReturnRujukanBaru(t *testing.T) {
	v := vclaim.NewMock()
	a := antrol.NewMock()
	k := khanza.NewMock()

	v.GetPesertaFunc = func(ctx context.Context, id string, tgl time.Time) (*domain.Peserta, error) {
		return pesertaAktif, nil
	}
	// 2 check error, 2 miss
	a.GetBookingHariIniFunc = func(ctx context.Context, noKartu string, tgl time.Time) (*domain.BookingMJKN, error) {
		return nil, errors.New("antrol down")
	}
	k.GetSuratKontrolFunc = func(ctx context.Context, noRM string) ([]domain.SuratKontrol, error) {
		return nil, errors.New("khanza timeout")
	}
	// PostRANAP & PostRAJAL: nil (miss tanpa error)

	d := New(v, a, k)
	d.parallelTimeout = 500 * time.Millisecond

	r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	if r.Type != domain.PatientTypeRujukanBaru {
		t.Errorf("partial failure tanpa hit: expected RujukanBaru, got %v", r.Type)
	}
}

func TestDetectorMatrix_PartialFailure_DenganHit_PakaiYangHit(t *testing.T) {
	v := vclaim.NewMock()
	a := antrol.NewMock()
	k := khanza.NewMock()

	v.GetPesertaFunc = func(ctx context.Context, id string, tgl time.Time) (*domain.Peserta, error) {
		return pesertaAktif, nil
	}
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// 2 error, 1 miss, 1 hit (PostRANAP)
	a.GetBookingHariIniFunc = func(ctx context.Context, noKartu string, tgl time.Time) (*domain.BookingMJKN, error) {
		return nil, errors.New("network")
	}
	k.GetSuratKontrolFunc = func(ctx context.Context, noRM string) ([]domain.SuratKontrol, error) {
		return nil, errors.New("network")
	}
	k.GetRiwayatRANAPFunc = func(ctx context.Context, noRM string) ([]domain.RiwayatRANAP, error) {
		return []domain.RiwayatRANAP{{TglKeluar: yesterday}}, nil // HIT
	}
	// PostRAJAL: default nil miss

	d := New(v, a, k)
	d.parallelTimeout = 500 * time.Millisecond

	r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	if r.Type != domain.PatientTypePostRANAP {
		t.Errorf("partial failure dengan 1 hit: expected PostRANAP, got %v", r.Type)
	}
}

// ============================================================
// Business rule: ranap lama (7 hari lalu) — tidak hit
// ============================================================

func TestDetectorMatrix_RanapLama_7HariLalu_TidakHit(t *testing.T) {
	v := vclaim.NewMock()
	k := khanza.NewMock()
	v.GetPesertaFunc = func(ctx context.Context, id string, tgl time.Time) (*domain.Peserta, error) {
		return pesertaAktif, nil
	}

	weekAgo := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	k.GetRiwayatRANAPFunc = func(ctx context.Context, noRM string) ([]domain.RiwayatRANAP, error) {
		return []domain.RiwayatRANAP{{NoRawat: "R-LAMA", TglKeluar: weekAgo}}, nil
	}

	d := New(v, antrol.NewMock(), k)
	d.parallelTimeout = 500 * time.Millisecond

	r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	if r.Type != domain.PatientTypeRujukanBaru {
		t.Errorf("ranap 7 hari lalu seharusnya MISS, got %v", r.Type)
	}
}

// ============================================================
// Business rule: kontrol futuro (table-driven untuk berbagai jarak hari)
// ============================================================

func TestDetectorMatrix_KontrolFuturo_VariousDays(t *testing.T) {
	v := vclaim.NewMock()
	v.GetPesertaFunc = func(ctx context.Context, id string, tgl time.Time) (*domain.Peserta, error) {
		return pesertaAktif, nil
	}

	cases := []struct {
		name      string
		offsetDay int
		wantHit   bool
	}{
		{"hari_ini_HIT", 0, true},
		{"kemarin_HIT", -1, true},
		{"3_hari_lalu_HIT", -3, true},
		{"besok_MISS", 1, false},
		{"3_hari_lagi_MISS", 3, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			k := khanza.NewMock()
			tgl := time.Now().AddDate(0, 0, tc.offsetDay).Format("2006-01-02")
			k.GetSuratKontrolFunc = func(ctx context.Context, noRM string) ([]domain.SuratKontrol, error) {
				return []domain.SuratKontrol{{NoSurat: "SK-1", TglRencana: tgl}}, nil
			}

			d := New(v, antrol.NewMock(), k)
			d.parallelTimeout = 500 * time.Millisecond

			r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
			gotHit := r.Type == domain.PatientTypeKontrol
			if gotHit != tc.wantHit {
				t.Errorf("[%s] gotType=%v wantHit=%v", tc.name, r.Type, tc.wantHit)
			}
		})
	}
}

// ============================================================
// Concurrency: 50 paralel Detect dengan random scenarios
// ============================================================

func TestDetectorMatrix_Concurrent50_HasilSesuaiSkenario(t *testing.T) {
	scenarios := []struct {
		hits     checkHits
		expected domain.PatientType
	}{
		{checkHits{mjkn: true}, domain.PatientTypeMJKN},
		{checkHits{kontrol: true}, domain.PatientTypeKontrol},
		{checkHits{ranap: true}, domain.PatientTypePostRANAP},
		{checkHits{rajal: true}, domain.PatientTypePostRAJAL},
		{checkHits{}, domain.PatientTypeRujukanBaru},
		{checkHits{mjkn: true, kontrol: true}, domain.PatientTypeMJKN},
		{checkHits{kontrol: true, ranap: true}, domain.PatientTypeKontrol},
	}

	const N = 50
	var wg sync.WaitGroup
	wg.Add(N)
	errs := make(chan error, N)

	for i := 0; i < N; i++ {
		i := i
		go func() {
			defer wg.Done()
			s := scenarios[i%len(scenarios)]
			d := detectorWithHits(t, s.hits)
			r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
			if r.Type != s.expected {
				errs <- errors.New("scenario " + s.expected.String() +
					" got " + r.Type.String())
			}
		}()
	}
	wg.Wait()
	close(errs)

	var failures []string
	for e := range errs {
		failures = append(failures, e.Error())
	}
	if len(failures) > 0 {
		t.Errorf("%d/%d scenario salah: %v", len(failures), N, failures)
	}
}

// ============================================================
// Goroutine leak: NumGoroutine snapshot sebelum & sesudah
// ============================================================

func TestDetectorMatrix_TidakLeakGoroutine_Setelah100Detect(t *testing.T) {
	d := detectorWithHits(t, checkHits{mjkn: true})

	// Warmup — beberapa Detect awal sebelum snapshot supaya runtime
	// sudah inisialisasi semua goroutine background.
	for i := 0; i < 5; i++ {
		d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	}
	time.Sleep(100 * time.Millisecond) // grace untuk goroutine sisa exit
	runtime.GC()

	before := runtime.NumGoroutine()

	for i := 0; i < 100; i++ {
		d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	}
	// Grace period — goroutine sisa harus selesai
	time.Sleep(300 * time.Millisecond)
	runtime.GC()

	after := runtime.NumGoroutine()

	// Toleransi ±5 — runtime kadang spawn/reuse worker secara non-deterministic.
	// Kalau ada leak, 100 Detect x 4 goroutine = ~400 leak, jauh di atas
	// toleransi 5.
	delta := after - before
	if delta > 5 {
		t.Errorf("goroutine leak terdeteksi: before=%d after=%d delta=%d",
			before, after, delta)
	}
}

// ============================================================
// Goroutine leak (skenario timeout): goroutine yang ditinggal saat
// parallelTimeout harus exit sendiri ketika ctx mereka cancel.
// ============================================================

func TestDetectorMatrix_TidakLeakGoroutine_SetelahTimeoutSkenario(t *testing.T) {
	v := vclaim.NewMock()
	a := antrol.NewMock()
	k := khanza.NewMock()

	v.GetPesertaFunc = func(ctx context.Context, id string, tgl time.Time) (*domain.Peserta, error) {
		return pesertaAktif, nil
	}
	// Semua check sleep tapi respect ctx — exit kalau ctx cancel
	slow := func(name string) {
		// dummy — diisi inline below
		_ = name
	}
	_ = slow

	a.GetBookingHariIniFunc = func(ctx context.Context, noKartu string, tgl time.Time) (*domain.BookingMJKN, error) {
		select {
		case <-time.After(5 * time.Second):
			return nil, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	k.GetSuratKontrolFunc = func(ctx context.Context, noRM string) ([]domain.SuratKontrol, error) {
		select {
		case <-time.After(5 * time.Second):
			return nil, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	k.GetRiwayatRANAPFunc = func(ctx context.Context, noRM string) ([]domain.RiwayatRANAP, error) {
		select {
		case <-time.After(5 * time.Second):
			return nil, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	k.GetKunjunganAktifFunc = func(ctx context.Context, noRM string) ([]domain.Kunjungan, error) {
		select {
		case <-time.After(5 * time.Second):
			return nil, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	d := New(v, a, k)
	d.parallelTimeout = 80 * time.Millisecond

	// Warmup
	for i := 0; i < 3; i++ {
		d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	}
	time.Sleep(300 * time.Millisecond)
	runtime.GC()
	before := runtime.NumGoroutine()

	// 30 Detect yang semuanya kena timeout
	for i := 0; i < 30; i++ {
		d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
	}
	// Grace lebih panjang — goroutine yang ditinggal butuh waktu untuk
	// menerima ctx.Done() dan exit.
	time.Sleep(500 * time.Millisecond)
	runtime.GC()

	after := runtime.NumGoroutine()
	delta := after - before
	if delta > 5 {
		t.Errorf("setelah 30 timeout-Detect: leak terdeteksi, before=%d after=%d delta=%d",
			before, after, delta)
	}
}

// ============================================================
// Stress: counter integrity di concurrent (memastikan mock mu lock benar)
// ============================================================

func TestDetectorMatrix_MockCounterIntegrityDiConcurrent(t *testing.T) {
	d := detectorWithHits(t, checkHits{kontrol: true})
	// Ambil mock khanza dari detector — kita perlu inject ulang supaya
	// bisa periksa CallCount-nya.
	v := vclaim.NewMock()
	a := antrol.NewMock()
	k := khanza.NewMock()
	v.GetPesertaFunc = func(ctx context.Context, id string, tgl time.Time) (*domain.Peserta, error) {
		return pesertaAktif, nil
	}
	today := time.Now().Format("2006-01-02")
	k.GetSuratKontrolFunc = func(ctx context.Context, noRM string) ([]domain.SuratKontrol, error) {
		return []domain.SuratKontrol{{TglRencana: today}}, nil
	}
	d = New(v, a, k)
	d.parallelTimeout = 500 * time.Millisecond

	const N = 30
	var wg sync.WaitGroup
	var ok atomic.Int32
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			r := d.Detect(context.Background(), domain.PatientInput{Identifier: "X"})
			if r.Type == domain.PatientTypeKontrol {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()

	if int(ok.Load()) != N {
		t.Errorf("expected %d sukses Kontrol, got %d", N, ok.Load())
	}
	// Kontrol = N panggilan (setiap Detect → 1x checkKontrol → 1x GetSuratKontrol)
	if got := k.CallCount("GetSuratKontrol"); got != N {
		t.Errorf("CallCount(GetSuratKontrol) = %d, want %d", got, N)
	}
	// Semua check lain juga dipanggil N kali (paralel — Khanza GetSuratKontrol
	// & GetRiwayatRANAP & GetKunjunganAktif semua di-fire tanpa peduli
	// hasilnya — priority resolution di parent).
	if got := k.CallCount("GetRiwayatRANAP"); got != N {
		t.Errorf("CallCount(GetRiwayatRANAP) = %d, want %d", got, N)
	}
	if got := k.CallCount("GetKunjunganAktif"); got != N {
		t.Errorf("CallCount(GetKunjunganAktif) = %d, want %d", got, N)
	}
	if got := a.CallCount("GetBookingHariIni"); got != N {
		t.Errorf("CallCount(GetBookingHariIni) = %d, want %d", got, N)
	}
}
