// Package detector adalah Smart BPJS Detector — komponen inti APM.
//
// Cara kerja (lihat juga BPJS_INTEGRATION.md):
//
//  1. Serial: VClaim GetPeserta → validasi status aktif.
//  2. Paralel (4 goroutine, timeout 5 detik):
//     checkMJKN, checkKontrol, checkPostRANAP, checkPostRAJAL
//  3. Priority resolution: MJKN > Kontrol > PostRANAP > PostRAJAL > RujukanBaru.
//
// Edge cases yang ditangani:
//   - Goroutine panic: di-recover, treat as miss.
//   - Semua check timeout: return RujukanBaru (default fallback yang aman).
//   - 2+ check hit: ambil prioritas tertinggi.
//   - Caller cancel context: stop collecting cepat (goroutine sisa exit
//     sendiri saat HTTP call mereka cancel).
//   - Log per check: PHI (no_kartu, no_rm) di-mask jadi "************XXXX".
package detector

import (
	"context"
	"log/slog"
	"time"

	"github.com/arunika/apm-go/internal/domain"
	"github.com/arunika/apm-go/internal/integration/antrol"
	"github.com/arunika/apm-go/internal/integration/khanza"
	"github.com/arunika/apm-go/internal/integration/vclaim"
)

// parallelTimeout adalah cap maksimum untuk fase 4 paralel check.
// Sesuai spec: 5 detik. Bisa di-override di test pakai field
// detector.parallelTimeout (lewat NewWithTimeout).
const parallelTimeout = 5 * time.Second

// Detector menggabungkan 3 integrasi backend BPJS+RS untuk klasifikasi
// pasien dalam satu panggilan Detect.
type Detector struct {
	vclaim vclaim.VClaimClient
	antrol antrol.AntrolClient
	khanza khanza.KhanzaClient

	// timeout fase paralel — diset ke parallelTimeout di New().
	// Field di-export untuk test (test bisa set ke 50ms agar cepat).
	parallelTimeout time.Duration

	// now di-injected supaya test bisa freeze time.
	now func() time.Time

	// logger di-injected; default slog.Default(). Logger seharusnya
	// menggunakan PHI masking handler di production (P-051).
	logger *slog.Logger
}

// New membangun Detector dengan dependencies. Caller wajib pasok ketiga
// client (real atau mock) — tidak ada lazy init.
func New(v vclaim.VClaimClient, a antrol.AntrolClient, k khanza.KhanzaClient) *Detector {
	return &Detector{
		vclaim:          v,
		antrol:          a,
		khanza:          k,
		parallelTimeout: parallelTimeout,
		now:             time.Now,
		logger:          slog.Default(),
	}
}

// SetLogger mengganti logger — dipakai test atau caller yang punya
// PHIMaskingHandler khusus.
func (d *Detector) SetLogger(l *slog.Logger) {
	if l != nil {
		d.logger = l
	}
}

// checkResult adalah satu hasil dari goroutine check.
// pType selalu di-set ke kategori yang dicek (untuk identifikasi kalau
// hit), data adalah payload spesifik (BookingMJKN / SuratKontrol / dll),
// hit adalah flag positif.
type checkResult struct {
	pType domain.PatientType
	data  any
	hit   bool
}

// Detect mengklasifikasi pasien berdasarkan input identifier.
// Mengembalikan DetectionResult yang siap dipakai layer UI.
//
// Method ini selalu return — tidak pernah panic — bahkan kalau
// salah satu dependency panic.
func (d *Detector) Detect(ctx context.Context, input domain.PatientInput) domain.DetectionResult {
	startedAt := d.now()
	today := startedAt

	// ─── STEP 1: Serial — VClaim GetPeserta ───────────────────
	peserta, err := d.vclaim.GetPeserta(ctx, input.Identifier, today)
	if err != nil {
		d.logger.Warn("detector: gagal lookup peserta",
			"id_masked", maskID(input.Identifier),
			"err", err.Error())
		return domain.DetectionResult{
			Type:       domain.PatientTypeError,
			Err:        err,
			DetectedAt: d.now(),
		}
	}
	if !peserta.IsAktif() {
		d.logger.Info("detector: peserta tidak aktif",
			"no_kartu_masked", maskID(peserta.NoKartu))
		return domain.DetectionResult{
			Type:       domain.PatientTypeTidakAktif,
			Peserta:    peserta,
			DetectedAt: d.now(),
		}
	}

	// ─── STEP 2: Paralel — 4 goroutine dengan timeout ─────────
	parCtx, cancel := context.WithTimeout(ctx, d.parallelTimeout)
	defer cancel()

	// Buffered ch=4 supaya goroutine bisa send tanpa blok bahkan
	// kalau parent abandon collecting (saat ctx timeout).
	ch := make(chan checkResult, 4)

	go d.checkMJKN(parCtx, peserta.NoKartu, today, ch)
	go d.checkKontrol(parCtx, peserta.NoRM, today, ch)
	go d.checkPostRANAP(parCtx, peserta.NoRM, today, ch)
	go d.checkPostRAJAL(parCtx, peserta.NoRM, today, ch)

	hits := d.collectResults(parCtx, ch, 4)

	// ─── STEP 3: Priority resolution ───────────────────────────
	for _, t := range []domain.PatientType{
		domain.PatientTypeMJKN,
		domain.PatientTypeKontrol,
		domain.PatientTypePostRANAP,
		domain.PatientTypePostRAJAL,
	} {
		if data, ok := hits[t]; ok {
			d.logger.Info("detector: kategori terdeteksi",
				"no_kartu_masked", maskID(peserta.NoKartu),
				"type", t.String())
			return domain.DetectionResult{
				Type:       t,
				Peserta:    peserta,
				Data:       data,
				DetectedAt: d.now(),
			}
		}
	}

	// Default: rujukan baru
	d.logger.Info("detector: kategori default RujukanBaru",
		"no_kartu_masked", maskID(peserta.NoKartu))
	return domain.DetectionResult{
		Type:       domain.PatientTypeRujukanBaru,
		Peserta:    peserta,
		DetectedAt: d.now(),
	}
}

// collectResults membaca hingga n hasil dari ch. Berhenti kalau ctx done
// (timeout/cancel) — goroutine sisa akan exit sendiri (HTTP call mereka
// cancel) dan write-nya ke buffered ch tidak blok.
func (d *Detector) collectResults(ctx context.Context, ch <-chan checkResult, n int) map[domain.PatientType]any {
	hits := make(map[domain.PatientType]any, n)
	for i := 0; i < n; i++ {
		select {
		case r := <-ch:
			if r.hit {
				hits[r.pType] = r.data
			}
		case <-ctx.Done():
			d.logger.Warn("detector: parallel checks timeout/cancel",
				"collected", i, "expected", n)
			return hits
		}
	}
	return hits
}

// safeRun adalah helper umum untuk semua check goroutine: panggil fn,
// recover panic, kirim hasilnya ke ch. fn return (data, hit, err).
// err di-log tapi tidak diteruskan ke channel — error tetap dianggap miss.
//
// Channel send TIDAK boleh blok — caller MUST pasok buffered channel.
func (d *Detector) safeRun(
	pType domain.PatientType,
	checkName string,
	fn func() (data any, hit bool, err error),
	ch chan<- checkResult,
) {
	defer func() {
		if r := recover(); r != nil {
			d.logger.Error("detector: panic di check goroutine",
				"check", checkName, "panic", r)
			ch <- checkResult{pType: pType}
		}
	}()

	data, hit, err := fn()
	if err != nil {
		d.logger.Warn("detector: check error (treated as miss)",
			"check", checkName, "err", err.Error())
		ch <- checkResult{pType: pType}
		return
	}
	ch <- checkResult{pType: pType, data: data, hit: hit}
}
