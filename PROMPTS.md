# Prompt Library — APM-Go Claude Code

> Cara pakai: buka Claude Code di folder proyek (`claude`), paste prompt sesuai task.
> Claude Code otomatis baca `CLAUDE.md` di root folder — jadi konteks selalu ada.

---

## FASE 0 — SETUP PROYEK

### P-000 · Install & verifikasi toolchain (jalankan di terminal dulu, bukan Claude Code)

```bash
# Install semua yang dibutuhkan di Mac
brew install go mingw-w64 node
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Verifikasi
go version        # minimal go1.22
wails version     # minimal v2.8
node --version    # minimal v18
x86_64-w64-mingw32-gcc --version  # cross-compiler Windows
```

---

### P-001 · Inisialisasi struktur proyek

```
Baca CLAUDE.md terlebih dahulu, lalu kerjakan:

Inisialisasi proyek APM-Go dengan struktur lengkap sesuai CLAUDE.md.

Langkah yang harus dikerjakan:
1. Jalankan: wails init -n apm-go -t vue
2. Buat struktur folder di dalam proyek:
   - internal/config/
   - internal/domain/
   - internal/service/detector/
   - internal/service/antrian/
   - internal/service/sep/
   - internal/service/pendaftaran/
   - internal/service/satusehat/
   - internal/integration/vclaim/
   - internal/integration/antrol/
   - internal/integration/mjkn/
   - internal/integration/khanza/
   - internal/hardware/frista/
   - internal/hardware/fingerprint/
   - internal/hardware/printer/
   - internal/store/
   - internal/reconcile/
   - migrations/
   - templates/
3. Buat go.mod dengan module: github.com/arunika/apm-go
4. Install Go dependencies:
   - github.com/spf13/viper (config)
   - github.com/mattn/go-sqlite3 (SQLite)
   - github.com/go-resty/resty/v2 (HTTP client)
   - github.com/robfig/cron/v3 (scheduler reset antrian)
   - github.com/stretchr/testify (testing)
5. Setup Tailwind CSS v3 di folder frontend/
6. Buat config.example.toml dengan semua key dari CLAUDE.md bagian "Referensi Cepat"
   — gunakan nilai placeholder, BUKAN credential nyata
7. Buat .gitignore yang exclude: config.toml, *.exe, dist/, node_modules/
8. Buat Makefile dengan target: dev, build-mac, build-windows, test, test-coverage, lint

Pastikan `wails dev` bisa jalan tanpa error setelah setup selesai.
```

---

### P-002 · Buat domain layer

```
Baca CLAUDE.md dan BPJS_INTEGRATION.md terlebih dahulu, lalu kerjakan:

Buat domain layer di internal/domain/ — ini adalah inti bisnis, TIDAK boleh ada
import package eksternal (hanya Go stdlib).

Buat file-file berikut:

1. internal/domain/patient.go
   Struct yang dibutuhkan:
   - Peserta: NoKartu, Nama, TglLahir, JK, Status, KdKantor, KdPPKPelayanan,
     NoRM, Poli, Kelas, JnsPeserta, Asuransi, TglTMT, TglTAT
   - PatientInput: Identifier string (bisa NoKartu/NIK/NoRM)
   - CardData: NIK, Nama, TglLahir, Alamat, NoKartu (dari Frista)
   Method: func (p *Peserta) IsAktif() bool { return p.Status == "1" }

2. internal/domain/detection.go
   - PatientType sebagai int enum dengan iota:
     PatientTypeUnknown, PatientTypeMJKN, PatientTypeKontrol,
     PatientTypePostRANAP, PatientTypePostRAJAL, PatientTypeRujukanBaru,
     PatientTypeTidakAktif, PatientTypeError
   - func (t PatientType) Label() string — return label bahasa Indonesia
   - DetectionResult: Type PatientType, Peserta *Peserta, Data any, Err error, DetectedAt time.Time
   - DetectStep: ID string, Label string, State string (done/active/wait/error)

3. internal/domain/antrian.go
   - AntrianJenis: LOKET, POLI, UMUM sebagai string const
   - AntrianSubJenis: APPOINTMENT, WALKIN, RANAP_IGD, FARMASI, CS, KASIR
   - Ticket: ID, Jenis, SubJenis, Nomor, Prefix, NoRM, CreatedAt, Synced bool
   - AntrianRequest: Jenis, SubJenis, NoRM string
   - Format nomor: LOKET="A-001", POLI="B-POLI_DALAM-015", UMUM="C-FAR-022"

4. internal/domain/sep.go
   - SEPRequest: NoKartu, TglSEP, JnsPelayanan, Poli, DPJP, NoRujukan,
     NoSuratKontrol, KelasRawat, AsalRujukan, Diagnosa, CatatanSEP
   - SEP: NoSEP, NoKartu, TglSEP, Poli, DPJP, NoRM, JnsPelayanan
   - SuratKontrol: NoSurat, NoSEP, TglRencana, Poli, DPJP, NoRM
   - Rujukan: NoSurat, FKTP, TglRujukan, TglExpiry, Poli, Diagnosa, IsValid bool
   - VClaimErrorMap: map[string]string semua kode error VClaim → pesan Indonesia
     (dari BPJS_INTEGRATION.md bagian "Business Rules VClaim")

5. internal/domain/errors.go
   - ErrPesertaTidakAktif, ErrRujukanExpired, ErrJadwalKontrolBelumTiba,
     ErrDuplikasiPendaftaran, ErrDokterCuti, ErrBiometrikDiperlukan,
     ErrHardwareUnavailable, ErrOfflineMode
   - Semua implementasi interface error dengan pesan bahasa Indonesia

Aturan keras:
- Zero import eksternal di domain/
- Semua struct harus punya JSON tag
- Enum PatientType harus persis sama dengan BPJS_INTEGRATION.md
```

---

## FASE 1 — HARDWARE LAYER

### P-010 · Hardware provider pattern (Mac mock + Windows real)

```
Baca CLAUDE.md dan HARDWARE_PLATFORM.md terlebih dahulu, lalu kerjakan:

Buat hardware provider dengan pattern dual-platform sesuai HARDWARE_PLATFORM.md.

1. Buat interface di masing-masing subfolder:

   internal/hardware/frista/interface.go
   type CardReader interface {
       Start(ctx context.Context) error
       Stop() error
       IsAvailable() bool
       CardRead() <-chan domain.CardData
   }

   internal/hardware/fingerprint/interface.go
   type FingerprintVerifier interface {
       Verify(ctx context.Context, noPeserta string) (FPResult, error)
       IsAvailable() bool
   }
   type FPResult struct {
       Success   bool
       Token     string
       Message   string
       Timestamp time.Time
   }

   internal/hardware/printer/interface.go
   type ThermalPrinter interface {
       Print(docType string, data PrintData) error
       IsAvailable() bool
       Reprint(historyID int64) error
   }

2. Buat internal/hardware/provider.go:
   - Fungsi NewProvider(cfg config.Config) *Provider
   - Gunakan runtime.GOOS == "windows" untuk switch implementasi
   - Di non-Windows: return mock implementations
   - Di Windows: return real implementations (stub dulu, isi nanti)

3. Implementasi MOCK (untuk Mac development):

   internal/hardware/frista/mock.go
   - Start() spawn goroutine HTTP server di cfg.Dev.MockPort (default 9090)
   - Expose endpoint: POST /mock/card-read — terima JSON CardData, kirim ke channel
   - Expose endpoint: GET / — halaman HTML sederhana listing semua endpoint
   - CardRead() return channel yang di-feed dari HTTP endpoint

   internal/hardware/fingerprint/mock.go
   - Verify() tunggu 2 detik (simulasi scan), lalu return FPResult{Success: true}
   - Jika ada flag mock-fail aktif: return error
   - Expose POST /mock/fp-fail di HTTP server yang sama dengan Frista mock

   internal/hardware/printer/console.go
   - Print() format dokumen ke stdout dengan border ASCII yang readable
   - Simpan ke SQLite print_history
   - Reprint() baca dari print_history dan print ulang ke stdout

4. Tambahkan ke Makefile:
   mock-card-read:
     curl -s -X POST http://localhost:9090/mock/card-read \
       -H "Content-Type: application/json" \
       -d '{"nik":"$(NIK)","nama":"$(NAMA)","tgl_lahir":"1980-01-15","alamat":"Jl. Test No.1","no_kartu":"$(NO_KARTU)"}'

   mock-fp-fail:
     curl -s -X POST http://localhost:9090/mock/fp-fail

Pastikan di Mac, `wails dev` + `make mock-card-read NIK=3271234 NAMA="Budi"` bisa jalan.
```

---

### P-011 · Windows implementations (stub untuk compile, isi saat di Windows)

```
Baca HARDWARE_PLATFORM.md terlebih dahulu, lalu kerjakan:

Buat stub implementasi Windows agar cross-compile tidak error.
Ini BUKAN implementasi nyata — hanya agar binary Windows bisa di-build dari Mac.

1. internal/hardware/frista/windows.go
   Build tag: //go:build windows
   - Struct WindowsFristaReader yang implement CardReader interface
   - Start() return nil (stub)
   - Stop() return nil
   - IsAvailable() return true
   - CardRead() return channel kosong (tidak pernah ada data)
   - TODO comment: "Implementasi nyata: spawn frista.exe HIDDEN + Windows UI Automation"

2. internal/hardware/fingerprint/windows.go
   Build tag: //go:build windows
   - Struct WindowsFPVerifier yang implement FingerprintVerifier
   - Verify() return FPResult{Success: true, Token: "STUB"} (stub)
   - TODO comment: "Implementasi nyata: spawn After.exe HIDDEN + REST polling"

3. internal/hardware/printer/escpos.go
   Build tag: //go:build windows
   - Struct ESCPOSPrinter yang implement ThermalPrinter
   - Print() return nil (stub)
   - TODO comment: "Implementasi nyata: USB write ESC/POS commands"

Verifikasi: `make build-windows` harus sukses tanpa error setelah ini.
```

---

## FASE 2 — INTEGRASI BPJS

### P-020 · VClaim client

```
Baca BPJS_INTEGRATION.md dengan teliti terlebih dahulu, lalu kerjakan:

Implementasi BPJS VClaim API v2.0 client di internal/integration/vclaim/.

File yang harus dibuat:

1. internal/integration/vclaim/client.go
   Struct Client dengan field: consID, secretKey, baseURL, httpClient *resty.Client
   Constructor: NewClient(cfg config.BPJSConfig) *Client

2. internal/integration/vclaim/auth.go
   func (c *Client) sign(timestamp int64) string
   - Formula: HMAC-SHA256(consID + "&" + timestamp, secretKey)
   - Return: base64 standard encoding
   
   func (c *Client) headers() map[string]string
   - Return header map: X-cons-id, X-timestamp (epoch sekarang), X-signature

3. internal/integration/vclaim/decrypt.go
   func (c *Client) decrypt(ciphertext string) ([]byte, error)
   - Key = SHA256(secretKey + consID) → 32 bytes
   - IV = first 16 bytes of key
   - Algorithm: AES-256-CBC + PKCS7 unpadding
   - Input ciphertext: base64 encoded

4. internal/integration/vclaim/peserta.go
   func (c *Client) GetPeserta(ctx context.Context, identifier string, tgl time.Time) (*domain.Peserta, error)
   - GET /Peserta/noKartu/{identifier}/{tgl format 2006-01-02}
   - Decrypt response
   - Parse JSON ke domain.Peserta
   - Jika status != "1": return domain.ErrPesertaTidakAktif

5. internal/integration/vclaim/rencana_kontrol.go
   func (c *Client) GetRencanaKontrol(ctx context.Context, noKartu string, tgl time.Time) ([]domain.SuratKontrol, error)
   - GET /RencanaKontrol/List/{noKartu}/{tglAwal}/{tglAkhir}
   - tglAwal = tgl, tglAkhir = tgl (same day check)

6. internal/integration/vclaim/sep.go
   func (c *Client) CreateSEP(ctx context.Context, req domain.SEPRequest) (*domain.SEP, error)
   func (c *Client) CreateSEPKontrol(ctx context.Context, req domain.SEPKontrolRequest) (*domain.SEP, error)

7. internal/integration/vclaim/rujukan.go
   func (c *Client) ValidasiRujukan(ctx context.Context, noSurat string, tgl time.Time) (*domain.Rujukan, error)

8. internal/integration/vclaim/interface.go
   Interface VClaimClient dengan semua method di atas
   — dipakai oleh detector dan sep service

Implementasi detail:
- Semua request harus punya context timeout
- Retry: 2x untuk HTTP 5xx, delay 500ms
- Rate limit: token bucket, max 100 req/menit (golang.org/x/time/rate)
- Error mapping: gunakan VClaimErrorMap dari domain/sep.go
- PHI di log: NoKartu dan NIK harus di-mask

Test: buat vclaim_test.go dengan httptest.NewServer sebagai mock server BPJS.
Test semua endpoint + test decrypt + test HMAC signing.
```

---

### P-021 · Antrol & MJKN client

```
Baca BPJS_INTEGRATION.md terlebih dahulu, lalu kerjakan:

Buat dua client tambahan:

1. internal/integration/antrol/client.go
   Interface AntrolClient:
   - CheckBookingMJKN(ctx, noKartu, tgl) ([]BookingMJKN, error)
   - CheckinMJKN(ctx, noKartu, kodebooking) error
   - PushAntrianPoli(ctx, req AntrianPoliRequest) error  ← fire-and-forget

   Implementasi:
   - Auth sama dengan VClaim (HMAC-SHA256) TAPI pakai credential antrol terpisah
   - BaseURL: dari config.BPJS.AntrolURL
   - PushAntrianPoli: error tidak boleh panic atau return ke user
     → log error saja, lanjut

2. internal/integration/mjkn/client.go  
   Untuk sekarang: cukup wrapper tipis di atas Antrol client
   MJKNClient interface sama dengan subset AntrolClient

Buat mock untuk keduanya menggunakan interface,
dipakai di detector tests.
```

---

### P-022 · Khanza REST client

```
Baca CLAUDE.md bagian "Endpoint Penting" terlebih dahulu, lalu kerjakan:

Buat Khanza API client di internal/integration/khanza/.

Interface KhanzaClient:
- CariPasien(ctx, query string) ([]domain.Pasien, error)
- BuatAntrian(ctx, req domain.AntrianRequest) (*domain.Ticket, error)  ← atomic
- BuatPendaftaran(ctx, req domain.PendaftaranRequest) (*domain.Pendaftaran, error)
- SimpanSEP(ctx, sep domain.SEP) error
- GetJadwalDokter(ctx, poliID string, tgl time.Time) ([]domain.Dokter, error)
- GetSuratKontrol(ctx, noSurat string) (*domain.SuratKontrol, error)
- GetRiwayatRANAP(ctx, noRM string) (*domain.RiwayatRANAP, error)
- UpdateSatuSehatID(ctx, noRM, ihsNumber string) error

Implementasi:
- BaseURL dari config.Server.KhanzaURL
- Auth: Bearer token dari config.Server.KhanzaAPIKey
- Timeout: 10 detik per request
- Retry: 2x untuk 5xx atau network error
- Jika Khanza unreachable: return ErrOfflineMode
- BuatAntrian HARUS atomic di server-side → Khanza yang handle increment

Buat OfflineKhanzaClient yang implement KhanzaClient:
- Semua write operation: simpan ke SQLite pending queue
- Semua read operation: coba dari cache lokal dulu, fallback error
- Dipakai otomatis saat ErrOfflineMode terdeteksi

Test: httptest.NewServer untuk mock semua endpoint.
```

---

## FASE 3 — SMART DETECTOR

### P-030 · Implementasi Smart BPJS Detector Engine

```
Baca BPJS_INTEGRATION.md bagian "Smart BPJS Detector Engine" dengan sangat teliti,
lalu kerjakan:

Implementasi detector di internal/service/detector/.

1. internal/service/detector/interface.go
   type Detector interface {
       Detect(ctx context.Context, input domain.PatientInput) domain.DetectionResult
   }

2. internal/service/detector/detector.go
   Struct DetectorService dengan dependencies:
   - vclaim    vclaim.VClaimClient
   - antrol    antrol.AntrolClient
   - khanza    khanza.KhanzaClient
   - steps     chan domain.DetectStep  ← untuk live update ke UI via Wails

   Implement Detect() dengan alur PERSIS seperti di BPJS_INTEGRATION.md:
   - Step 1 SERIAL: GetPeserta dari VClaim
   - Emit step "Verifikasi status BPJS" = done ke channel
   - Jika tidak aktif: return PatientTypeTidakAktif LANGSUNG
   - Step 2 PARALEL dengan context 5 detik:
     4 goroutine: checkMJKN, checkKontrol, checkPostRANAP, checkPostRAJAL
   - Collect 4 results dari channel
   - Priority resolution: MJKN > KONTROL > POST_RANAP > POST_RAJAL
   - Jika tidak ada hit: return PatientTypeRujukanBaru

3. internal/service/detector/mjkn_check.go
   func (d *DetectorService) checkMJKN(ctx context.Context, noKartu string, ch chan<- typedResult)
   - Panggil antrol.CheckBookingMJKN(ctx, noKartu, today)
   - Jika ada booking aktif: kirim result{hit: true, type: MJKN, data: booking}
   - Emit step update ke d.steps channel

4. internal/service/detector/kontrol_check.go
   func (d *DetectorService) checkKontrol(ctx context.Context, noRM string, ch chan<- typedResult)
   - Panggil khanza.GetSuratKontrol dengan filter tgl = today
   - Jika ditemukan DAN tgl rencana = today: kirim result{hit: true, type: KONTROL}
   - Jika tgl rencana > today: kirim result dengan ErrJadwalKontrolBelumTiba

5. internal/service/detector/ranap_check.go
   func (d *DetectorService) checkPostRANAP(ctx context.Context, noRM string, ch chan<- typedResult)
   - Panggil khanza.GetRiwayatRANAP
   - Jika tgl keluar = yesterday atau today: hit = true, type = POST_RANAP

6. internal/service/detector/rajal_check.go
   func (d *DetectorService) checkPostRAJAL(ctx context.Context, noRM string, ch chan<- typedResult)
   - Cek riwayat kunjungan RAJAL aktif via VClaim RiwayatPelayanan
   - Jika ada kunjungan dalam 30 hari terakhir dengan surat kontrol beda poli: hit = true

7. internal/service/detector/detector_test.go
   Table-driven test WAJIB untuk SEMUA skenario:

   | Nama Test                    | MJKN | Kontrol | RANAP | RAJAL | Expected       |
   |------------------------------|------|---------|-------|-------|----------------|
   | only_mjkn                    | hit  | -       | -     | -     | MJKN           |
   | only_kontrol                 | -    | hit     | -     | -     | KONTROL        |
   | only_ranap                   | -    | -       | hit   | -     | POST_RANAP     |
   | only_rajal                   | -    | -       | -     | hit   | POST_RAJAL     |
   | no_hit                       | -    | -       | -     | -     | RUJUKAN_BARU   |
   | mjkn_priority_over_kontrol   | hit  | hit     | -     | -     | MJKN           |
   | all_hit                      | hit  | hit     | hit   | hit   | MJKN           |
   | peserta_tidak_aktif          | -    | -       | -     | -     | TIDAK_AKTIF    |
   | vclaim_network_error         | -    | -       | -     | -     | ERROR          |
   | all_checks_timeout           | -    | -       | -     | -     | RUJUKAN_BARU   |
   | kontrol_tgl_futuro           | -    | futuro  | -     | -     | ERROR+msg      |

   Mock semua dependencies via interface.
   Test timeout: gunakan context dengan deadline 100ms, semua goroutine harus stop.
```

---

## FASE 4 — ANTRIAN & SEP SERVICE

### P-040 · Antrian service (3 jalur)

```
Baca CLAUDE.md terlebih dahulu, lalu kerjakan:

Implementasi antrian service di internal/service/antrian/.

1. internal/service/antrian/service.go
   Interface AntrianService:
   - Create(ctx, req domain.AntrianRequest) (*domain.Ticket, error)
   - GetCurrent(ctx, jenis, subJenis string) (int, error)
   - ResetAll(ctx) error  ← dipanggil cron 00:01

   Implementasi:
   - Panggil khanza.BuatAntrian untuk counter server-side (anti-duplikat multi-kiosk)
   - Jika offline: generate nomor lokal dari SQLite counter, simpan pending
   - Format nomor:
     LOKET+APPOINTMENT → "A-001"
     LOKET+WALKIN      → "A-002" (counter LOKET shared)
     POLI              → "B-{KODE_POLI}-015"
     UMUM+FARMASI      → "C-FAR-022"
   - Simpan ke print_history via printer.Print()
   - Fire-and-forget: push ke Antrol API setelah simpan lokal

2. internal/service/antrian/scheduler.go
   - Setup cron job: setiap hari 00:01 WIB panggil ResetAll()
   - Gunakan robfig/cron/v3
   - Log setiap reset

3. internal/service/antrian/service_test.go
   - Test Create() dengan mock Khanza online
   - Test Create() dengan Khanza offline → harus masuk SQLite pending
   - Test concurrent Create() dari 2 goroutine → nomor tidak duplikat
   - Test ResetAll() → semua counter kembali ke 0
```

---

### P-041 · SEP builder service

```
Baca BPJS_INTEGRATION.md dan CLAUDE.md, lalu kerjakan:

Implementasi SEP service di internal/service/sep/.

1. internal/service/sep/service.go
   Interface SEPService:
   - BuatSEPRujukanBaru(ctx, req SEPRujukanRequest) (*domain.SEP, error)
   - BuatSEPKontrol(ctx, req SEPKontrolRequest) (*domain.SEP, error)
   - BuatSEPPostRANAP(ctx, req SEPPostRANAPRequest) (*domain.SEP, error)
   - BuatSEPPostRAJAL(ctx, req SEPPostRAJALRequest) (*domain.SEP, error)
   - BuatSEPMJKN(ctx, req SEPMJKNRequest) (*domain.SEP, error)

2. Alur untuk setiap jenis SEP:
   a. Validasi biometrik jika diperlukan:
      - Wajib untuk: RUJUKAN_BARU, KONTROL
      - Usia ≥ 17 tahun, layanan non-IGD
      - Panggil fp.Verify() — blocking
      - Jika gagal 3x: return ErrBiometrikDiperlukan
   b. Validasi business rules (BR-01 s/d BR-13 dari PRD)
   c. Simpan ke SQLite DULU (audit trail)
   d. Kirim ke VClaim API
   e. Jika sukses: update status di SQLite + simpan ke Khanza
   f. Jika VClaim gagal: simpan ke pending_sep dengan status = failed
   g. Trigger print: printer.Print("SEP", sepData)

3. Business rules yang WAJIB di-enforce di layer ini:
   - BR-06: Jadwal kontrol futuro → return error dengan tanggal rencana
   - BR-09: Laka Lantas → wajib ada CatatanSEP berisi keterangan kecelakaan
   - BR-10: Hak kelas tidak sesuai → map VClaim error -7 ke pesan user-friendly
   - BR-11: Batas SEP harian → map VClaim error -8
   - BR-13: Simpan ke SQLite sebelum Khanza

4. Test:
   - Test setiap jenis SEP dengan mock VClaim
   - Test biometrik wajib/tidak wajib berdasarkan usia dan jenis layanan
   - Test Laka Lantas tanpa keterangan → harus error
   - Test VClaim error -7 dan -8 → pesan Indonesia yang benar
   - Test offline: VClaim gagal → masuk pending_sep
```

---

## FASE 5 — OFFLINE & REKONSILIASI

### P-050 · SQLite schema & store

```
Baca BPJS_INTEGRATION.md bagian "Mode Offline" lalu kerjakan:

1. Buat file migrations/001_initial.sql:

CREATE TABLE antrian_lokal (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    jenis       TEXT NOT NULL,
    sub_jenis   TEXT,
    nomor       TEXT NOT NULL,
    prefix      TEXT NOT NULL,
    no_rm       TEXT,
    created_at  DATETIME DEFAULT (datetime('now','localtime')),
    synced_at   DATETIME,
    sync_status TEXT NOT NULL DEFAULT 'pending'
);

CREATE TABLE pending_sep (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    no_kartu        TEXT NOT NULL,
    jenis_sep       TEXT NOT NULL,
    payload_json    TEXT NOT NULL,
    vclaim_response TEXT,
    status          TEXT NOT NULL DEFAULT 'pending',
    retry_count     INTEGER NOT NULL DEFAULT 0,
    last_error      TEXT,
    created_at      DATETIME DEFAULT (datetime('now','localtime')),
    confirmed_by    TEXT,
    confirmed_at    DATETIME
);

CREATE TABLE print_history (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    doc_type      TEXT NOT NULL,
    ref_id        TEXT,
    escpos_bytes  BLOB,
    printed_at    DATETIME DEFAULT (datetime('now','localtime')),
    reprint_count INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE reconcile_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    table_name  TEXT NOT NULL,
    record_id   INTEGER NOT NULL,
    action      TEXT NOT NULL,
    operator_id TEXT,
    result      TEXT,
    ts          DATETIME DEFAULT (datetime('now','localtime'))
);

CREATE TABLE config_cache (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at DATETIME DEFAULT (datetime('now','localtime'))
);

2. Buat internal/store/ dengan sqlc:
   - Buat sqlc.yaml
   - Tulis SQL queries di internal/store/queries/
   - Generate Go code: sqlc generate

3. Buat internal/store/db.go:
   - Fungsi NewDB(path string) (*sql.DB, error)
   - Auto-run migrations saat start
   - Enable WAL mode: PRAGMA journal_mode=WAL
   - Set busy_timeout: 5000

4. Buat Makefile target:
   db-generate:
     sqlc generate
   db-reset:
     rm -f apm.db && go run ./cmd/apm --migrate
```

---

### P-051 · Reconcile worker

```
Lanjutkan dari P-050, lalu kerjakan:

Buat internal/reconcile/worker.go:

type Worker struct {
    db     *store.Queries
    khanza khanza.KhanzaClient
    vclaim vclaim.VClaimClient
    log    *slog.Logger
}

func (w *Worker) Start(ctx context.Context)
- Ticker 30 detik
- Loop:
  1. Cek apakah Khanza reachable (GET /api/apm/ping dengan timeout 3 detik)
  2. Jika tidak: log "offline, skip reconcile", continue
  3. Jika ya:
     a. SyncAntrian: ambil semua antrian_lokal status=pending, POST ke Khanza,
        update status=synced atau status=failed (jika gagal 3x)
     b. SyncPendaftaran: sama seperti antrian
     c. SyncSEP: ambil pending_sep status=awaiting_confirm SAJA
        (TIDAK auto-sync — harus ada confirmed_by dulu)
  4. Catat semua hasil ke reconcile_log
- Graceful shutdown saat ctx.Done()

Expose ke Wails:
- GetOfflineStatus() bool
- GetPendingCounts() PendingCounts{Antrian, Pendaftaran, SEP int}
- ConfirmSEPSync(id int64) error  ← hanya untuk operator

Kirim Wails event "offline:status_changed" setiap kali status berubah
sehingga Vue bisa tampilkan/sembunyi banner "OFFLINE MODE".
```

---

## FASE 6 — WAILS APP & IPC

### P-060 · App struct & Wails bindings

```
Baca DESIGN_SYSTEM.md bagian "Wails IPC Binding Conventions", lalu kerjakan:

Buat cmd/apm/app.go — ini yang di-expose ke Vue frontend:

type App struct {
    ctx         context.Context
    cfg         config.Config
    hardware    *hardware.Provider
    detector    service.Detector
    antrianSvc  service.AntrianService
    sepSvc      service.SEPService
    reconcile   *reconcile.Worker
    db          *store.Queries
}

Semua method public di App otomatis ter-expose ke Vue via Wails.

Method yang harus ada (return selalu (data, error)):

// Detection
func (a *App) DetectPatient(input string) (*domain.DetectionResult, error)
func (a *App) GetDetectSteps() []domain.DetectStep

// Antrian
func (a *App) CreateAntrian(jenis, subJenis string) (*domain.Ticket, error)
func (a *App) GetAntrianCurrent(jenis, subJenis string) (int, error)
func (a *App) ReprintTicket(historyID int64) error

// SEP
func (a *App) CreateSEPMJKN(req domain.SEPMJKNRequest) (*domain.SEP, error)
func (a *App) CreateSEPKontrol(req domain.SEPKontrolRequest) (*domain.SEP, error)
func (a *App) CreateSEPRujukanBaru(req domain.SEPRujukanRequest) (*domain.SEP, error)
func (a *App) CreateSEPPostRANAP(req domain.SEPPostRANAPRequest) (*domain.SEP, error)

// Hardware status
func (a *App) GetHardwareStatus() HardwareStatus
func (a *App) TriggerMockCardRead(data domain.CardData) error  // hanya non-Windows

// Admin
func (a *App) GetDashboardStats() DashboardStats
func (a *App) GetPendingSEPs() ([]domain.PendingSEP, error)
func (a *App) ConfirmSEPSync(id int64) error
func (a *App) GetComponentStatus() []ComponentStatus
func (a *App) ResetAntrianCounter() error
func (a *App) GetReconcileLog(limit int) ([]domain.ReconcileLog, error)

// Frista events dikirim via:
// runtime.EventsEmit(a.ctx, "frista:card_read", cardData)
// Dipanggil dari hardware.Provider saat ada card read

// Offline events:
// runtime.EventsEmit(a.ctx, "offline:status_changed", isOffline)
// runtime.EventsEmit(a.ctx, "detect:step_update", step)

Buat cmd/apm/main.go yang setup Wails dengan App sebagai backend.
```

---

## FASE 7 — FRONTEND VUE 3

### P-070 · Setup frontend & Pinia stores

```
Baca DESIGN_SYSTEM.md seluruhnya, lalu kerjakan:

Setup Vue 3 frontend di folder frontend/.

1. Install dependencies:
   npm install pinia @vueuse/core
   npm install -D tailwindcss postcss autoprefixer

2. Setup Tailwind dengan konfigurasi dari DESIGN_SYSTEM.md bagian "Design Tokens":
   Buat tailwind.config.js dengan semua custom tokens:
   - colors: blue, success, warning, danger, surface, bg, border, text
   - fontSize dengan clamp() untuk semua ukuran
   - borderRadius: kiosk, card, btn, tag

3. Buat Pinia stores di frontend/src/stores/:

   patient.ts — usePatientStore
   State: input, peserta, detectionResult, isDetecting, detectSteps, error
   Actions:
   - detect(input: string): panggil window.go.main.App.DetectPatient()
   - reset(): $reset() semua state
   - listenSteps(): subscribe ke Wails event "detect:step_update"

   antrian.ts — useAntrianStore
   State: lastTicket, counters, isCreating
   Actions:
   - create(jenis, subJenis): panggil CreateAntrian()
   - loadCounters(): panggil GetAntrianCurrent untuk semua jenis

   app.ts — useAppStore
   State: isOffline, hardwareStatus, idleCountdown, showIdleWarning
   Actions:
   - init(): subscribe ke "offline:status_changed" event
   - startIdleTimer(seconds): countdown, emit reset saat habis
   - resetIdle(): clear countdown

4. Buat frontend/src/router/index.ts dengan routes:
   / → HomeScreen
   /detect → DetectScreen
   /result/:type → ResultScreen
   /antrian → AntrianScreen
   /ticket → TicketScreen
   /admin → AdminScreen (protected, butuh PIN)

5. Semua navigation harus reset idle timer.

Verifikasi: `wails dev` tampilkan Vue app tanpa error console.
```

---

### P-071 · HomeScreen

```
Baca DESIGN_SYSTEM.md bagian komponen dan mockup yang sudah disetujui, lalu kerjakan:

Buat frontend/src/screens/HomeScreen.vue sesuai desain yang sudah disetujui.

Layout (WAJIB persis seperti ini):
- Header: logo RS (kiri) + status dots BPJS & Sistem + jam live (kanan)
- Hero button BPJS: full width, background biru, icon kartu, judul, subtitle, tag
- Grid 2 kolom: Pasien Umum (icon tinted biru) + Ambil Antrian (icon neutral)  
- Row penuh: Aktivasi Satu Sehat Mobile
- Footer sticky: "Butuh bantuan?" + tombol "Panggil petugas"

Aturan responsivitas (SEMUA WAJIB pakai clamp):
- Header padding: clamp(10px, 2vw, 16px)
- Hero button padding: clamp(14px, 2.5vw, 20px)
- Hero title font: clamp(14px, 2.2vw, 18px)
- Icon container: clamp(40px, 6vw, 52px) x clamp(40px, 6vw, 52px)
- Card button padding: clamp(12px, 2vw, 16px)
- Touch target min-height: clamp(52px, 7vw, 72px) untuk SEMUA tombol

Warna:
- Hero BPJS: background #1B4FD8
- Pasien Umum icon bg: #EEF2FF (blue-light)
- Antrian icon bg: #F5F6F8 (bg neutral)
- Footer: border-top 0.5px #E4E6EA

Interaksi:
- Klik BPJS → router.push('/detect')
- Klik Antrian → router.push('/antrian')  
- Idle timeout dari useAppStore.startIdleTimer(60)
  → setelah 50 detik tampilkan overlay countdown 10 detik
  → setelah 60 detik router.push('/') dan reset semua state

Jam live: update setiap menit via setInterval.
Status dots: hijau (#065F46) jika hardware tersedia, merah jika tidak.
Fetch status via useAppStore.hardwareStatus.

DILARANG hardcode ukuran dalam px di luar clamp(). Tidak ada media query.
```

---

### P-072 · Input & Detection screens

```
Baca DESIGN_SYSTEM.md, lalu kerjakan dua screen sekaligus:

1. frontend/src/screens/InputScreen.vue

   Komponen yang dipakai:
   - NumPad: grid 3 kolom, tombol 1-9, 0, hapus, cari
   - Display input: border biru saat aktif, monospace, letter-spacing
   - FristaBar: dot hijau + teks "Frista aktif — tempel kartu untuk isi otomatis"
   - Chips: label format yang diterima (JKN/NIK/NoRM)

   Touch target semua tombol NumPad: min-height clamp(52px, 7vw, 72px)
   Font angka: clamp(17px, 3vw, 22px)

   Logic:
   - Saat mount: subscribe ke Wails event "frista:card_read"
     → auto-fill input.value dari event.noKartu atau event.nik
   - Validasi: max 16 karakter untuk angka, alphanumeric untuk NoRM
   - Tombol "cari": panggil usePatientStore.detect(input)
     → loading state saat detecting
     → saat selesai: router.push('/result/' + detectionResult.type)
   - Tombol hapus: hapus satu karakter dari belakang
   - FristaBar: sembunyikan jika !hardwareStatus.frista

2. frontend/src/screens/DetectScreen.vue

   Tampilkan progress deteksi real-time:
   - SpinRing: animasi SVG ring (tidak pakai CSS gradient!)
   - StepList: 5 baris, masing-masing punya state: done/active/wait
   
   State steps (dari usePatientStore.detectSteps):
   - "Verifikasi status BPJS"
   - "Cek booking Mobile JKN"
   - "Cek jadwal kontrol"
   - "Cek riwayat rawat inap"
   - "Menentukan jenis kunjungan"

   Subscribe ke Wails event "detect:step_update":
   - Update state step yang sesuai di store
   - "done" → dot hijau + centang
   - "active" → dot biru pulse
   - "wait" → dot abu

   Auto-navigate ke ResultScreen saat detectionResult berubah.
   Timeout 8 detik: jika belum selesai tampilkan pesan error.
```

---

### P-073 · Result screens (semua 6 variant)

```
Baca DESIGN_SYSTEM.md dan BPJS_INTEGRATION.md, lalu kerjakan:

Buat frontend/src/screens/ResultScreen.vue yang handle semua 6 PatientType.
Gunakan dynamic component atau v-if berdasarkan $route.params.type.

Komponen bersama untuk semua variant:
- PatientCard: nama (font hero clamp), nomor kartu (monospace muted), pill status, divider, kv-rows
- StatusPill: 4 variant (ok/info/warn/danger), dot kecil + label
- CTA button: full width, biru, clamp padding
- Ghost button: border tipis, warna muted

Spec per variant:

MJKN (PatientTypeMJKN):
  Pill: hijau "Booking Mobile JKN"
  KV rows: Poli, Dokter, Estimasi jam (warna biru), No antrian booking
  Info bar: hijau — "Booking dari Mobile JKN sudah terkonfirmasi."
  CTA: "Konfirmasi kedatangan dan cetak tiket"
  Action: CreateSEPMJKN() → print → TicketScreen

KONTROL (PatientTypeKontrol):
  Pill: biru "Jadwal kontrol"
  KV rows: No surat kontrol, Poli, Dokter sebelumnya
  Dokter picker: list dokter aktif dari GetJadwalDokter()
    → item terpilih: border biru, background biru muda, checkmark icon
  CTA: "Buat surat layanan kontrol dan cetak"
  Action: CreateSEPKontrol() → print → TicketScreen

POST_RANAP (PatientTypePostRANAP):
  Pill: teal "Pasca rawat inap"
  KV rows: Tgl keluar, Ruangan, DPJP
  Info bar: kuning — "Pilih poli kontrol untuk kunjungan hari ini."
  Poli picker: grid tombol poli dari GetJadwalDokter semua poli
  CTA: "Buat surat layanan kontrol pasca rawat inap"

POST_RAJAL (PatientTypePostRAJAL):
  Pill: abu "Kontrol beda poli"
  Riwayat SEP: kunjungan terakhir (poli, tanggal)
  CTA: "Pilih poli tujuan dan buat surat layanan"

RUJUKAN_BARU (PatientTypeRujukanBaru):
  Pill: kuning "Kunjungan baru"
  KV rows: Kelas hak, No rujukan (biru), Poli rujukan, Berlaku sampai (hijau)
  Info bar: kuning — "Verifikasi sidik jari diperlukan setelah pilih dokter."
  CTA: "Pilih dokter dan lanjutkan"
  → FingerprintWidget muncul sebelum submit jika wajib

TIDAK_AKTIF (PatientTypeTidakAktif):
  Pill: merah "Kepesertaan tidak aktif"
  KV rows: Status (merah), Penyebab, Tunggak sejak
  Info bar: merah — cara mengaktifkan BPJS
  CTA primer: "Daftar sebagai pasien umum"
  Ghost: "Hubungi petugas"

Untuk semua variant: tombol "Bukan saya — masukkan ulang" di bawah.
```

---

### P-074 · Antrian & Ticket screens

```
Baca DESIGN_SYSTEM.md, lalu kerjakan:

1. frontend/src/screens/AntrianScreen.vue

   Layout persis seperti desain yang disetujui:
   
   Section label "Antrian loket (A)" — uppercase, muted, letter-spacing
   Grid 2 kolom:
   - Card "Admisi appointment" — icon biru
   - Card "Admisi walk-in" — icon biru
   Card full width:
   - "Rawat inap & IGD" — icon biru, flex row
   
   Section label "Antrian layanan umum (C)"
   Grid 2 kolom:
   - "Farmasi / apotek" — icon hijau
   - "Customer service" — icon neutral
   
   Setiap card tampilkan: ikon, judul, "Sekarang: {nomor}"
   Counter "Sekarang" di-fetch dari GetAntrianCurrent() saat mount.
   
   Saat tap card:
   - Set loading state pada card yang di-tap
   - Panggil useAntrianStore.create(jenis, subJenis)
   - Sukses → router.push('/ticket')
   - Gagal → tampilkan error toast

   Konfigurasi jenis antrian harus dari config (tidak hardcode label),
   agar RS bisa custom via admin panel.

2. frontend/src/screens/TicketScreen.vue

   Layout:
   - Check circle icon (hijau, border tipis)
   - Teks "Surat layanan berhasil dibuat" (hijau)
   - Ticket paper: label poli (uppercase muted), nomor besar (clamp 44-60px), dokter+tanggal
   - Dashed divider
   - SEP nomor (monospace, biru)
   - Instruksi area: background bg-color, rounded, teks arah ke poli
   - Countdown: "Kembali ke awal dalam {N} detik"
   - Tombol "Cetak ulang tiket"

   Nomor tiket font: clamp(44px, 8vw, 60px) — harus terbaca dari 1.5 meter
   
   Countdown logic:
   - Mulai dari 10 detik saat screen mount
   - Setiap detik decrement via setInterval
   - Saat 0: router.push('/') dan usePatientStore.reset()
   
   Cetak ulang: panggil ReprintTicket(lastTicket.historyID)
```

---

### P-075 · Admin screen

```
Baca DESIGN_SYSTEM.md, lalu kerjakan:

Buat frontend/src/screens/AdminScreen.vue.

Akses: dilindungi PIN (4 digit). PIN disimpan terenkripsi di config.toml.
Saat masuk AdminScreen: tampilkan modal PIN dulu.

Setelah login:

Layout dalam satu halaman scroll:

1. Stats grid 2x2:
   - "Antrian hari ini" — angka besar
   - "SEP berhasil" — angka besar
   - "Pending rekonsiliasi" — angka besar KUNING jika > 0
   - "Uptime" — format "7j 12m"
   Fetch dari GetDashboardStats() saat mount, auto-refresh setiap 30 detik.

2. Status komponen (card):
   List semua komponen dengan pill status:
   - BPJS VClaim API
   - BPJS Antrol
   - SIMRS Khanza
   - Frista card reader
   - Fingerprint BPJS
   - Printer thermal
   Fetch dari GetComponentStatus().
   Pill: hijau=Online/Terhubung, kuning=Degraded, merah=Offline/Error.

3. SEP pending confirmation (card, hanya tampil jika ada):
   List pending SEP dengan: nama pasien, no kartu, jenis SEP, waktu buat
   Tombol "Konfirmasi" per item → ConfirmSEPSync(id)
   Setelah konfirmasi: hilang dari list.

4. Quick actions grid 2x2:
   - "Reset counter antrian" → konfirmasi dulu via dialog
   - "Log rekonsiliasi" → modal dengan list terbaru
   - "Test cetak printer" → print test page
   - "Keluar admin" → kembali ke HomeScreen

Tombol "Keluar" di header: background danger-bg, text danger.
```

---

## FASE 8 — SECURITY & POLISH

### P-080 · PHI log masking & credential encryption

```
Baca CLAUDE.md bagian "Aturan Coding", lalu kerjakan:

1. internal/log/masker.go
   Buat custom slog.Handler yang wrap handler asli.
   Sebelum setiap log record ditulis:
   - Scan semua Attr (string values)
   - Pattern NIK/NoKartu: 16 digit berurutan → ganti dengan "****-****-****-****"
   - Field name sensitif: "nik", "no_kartu", "no_rm", "password", "username", 
     "secret", "token" → ganti value dengan "***"
   - Regex untuk NoRM: [A-Z0-9]{6,10} di konteks field nama "no_rm"
   
   Pasang masker sebagai default slog handler di main.go.

2. internal/config/encrypt.go
   CLI flag: --encrypt-config
   
   Alur:
   a. Baca config.toml yang sudah ada
   b. Untuk setiap field yang perlu dienkripsi (list dari konstanta):
      frista.username, frista.password, fingerprint.username, fingerprint.password
   c. Prompt di terminal: "Enter Frista username:"
   d. Baca dari stdin (no echo untuk password)
   e. Enkripsi dengan AES-256-GCM:
      - Di Mac/Linux: key dari env APM_MASTER_KEY atau random + simpan ke ~/.apm/master.key
      - Di Windows: key dari Windows DPAPI (CryptProtectData)
   f. Simpan ke config.toml sebagai "ENC:base64..."
   
   Saat startup normal:
   - Baca semua field "ENC:..."
   - Dekripsi menggunakan master key
   - Inject ke config struct (tidak ke file)

3. Tambahkan ke Makefile:
   encrypt-config:
     go run ./cmd/apm --encrypt-config
```

---

### P-081 · Idle timeout & fullscreen kiosk mode

```
Kerjakan fitur kiosk mode:

1. frontend/src/composables/useIdleTimer.ts
   - Parameter: timeoutSeconds (dari config, default 60)
   - Track last interaction: mousemove, touchstart, keydown, click
   - Setelah (timeout - 10) detik idle: emit "idle:warning" event
   - Setelah timeout detik idle: emit "idle:reset" event
   - Warning overlay: countdown dari 10 ke 0 dengan progress bar
   - Saat user interaksi saat warning: cancel, reset timer
   
   Pasang di App.vue level root.
   Saat "idle:reset": router.push('/') + semua store.$reset()

2. cmd/apm/main.go — Wails window options:
   - Fullscreen: true (untuk kiosk production)
   - Frameless: true (tidak ada title bar Windows)
   - DisableResize: true
   - AlwaysOnTop: true (opsional, bisa diconfig)
   
   Tambahkan config:
   [app]
   kiosk_mode = true   # fullscreen + frameless
   # kiosk_mode = false  # untuk development: window normal

3. Escape key admin shortcut (hanya jika bukan kiosk_mode):
   Ctrl+Shift+A → buka AdminScreen
```

---

## FASE 9 — BUILD & DEPLOYMENT

### P-090 · Cross-compile & packaging

```
Kerjakan build & deployment tooling:

1. Update Makefile dengan targets lengkap:

dev:
	wails dev

build-mac:
	wails build -platform darwin/arm64
	@echo "Output: build/bin/apm.app"

build-windows:
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
	CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ \
	wails build -platform windows/amd64
	@echo "Output: build/bin/apm.exe"

package-windows:
	mkdir -p dist/apm-windows
	cp build/bin/apm.exe dist/apm-windows/
	cp config.example.toml dist/apm-windows/config.toml
	cp -r migrations/ dist/apm-windows/migrations/
	cp INSTALL_WINDOWS.md dist/apm-windows/
	zip -r dist/apm-windows-$(VERSION).zip dist/apm-windows/
	@echo "Package: dist/apm-windows-$(VERSION).zip"

test:
	go test ./... -v -count=1

test-coverage:
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html
	@echo "Buka coverage.html di browser"

lint:
	golangci-lint run ./...

2. Buat INSTALL_WINDOWS.md (dalam Bahasa Indonesia):
   
   # Panduan Instalasi APM-Go di Windows
   
   ## Prasyarat
   - Windows 10 Pro/Enterprise x64 atau Windows 11
   - RAM minimum 4GB
   - Microsoft Edge WebView2 Runtime (biasanya sudah ada)
     Download: https://developer.microsoft.com/webview2
   - Frista.exe dari vendor Frista
   - Aplikasi Sidik Jari BPJS Kesehatan v2.0+ (After.exe)
   - Printer thermal USB atau Serial
   
   ## Langkah Instalasi
   1. Ekstrak file zip ke C:\apm\
   2. Edit config.toml sesuai environment RS Anda
   3. Enkripsi credential: double-click apm.exe --encrypt-config
   4. Test koneksi: apm.exe --check-all
   5. Install service: buka PowerShell as Administrator, jalankan:
      C:\apm\apm.exe --install-service
   6. Start service: net start APMService
   
   ## Troubleshooting
   [isi dengan masalah umum dari HARDWARE_PLATFORM.md]

3. Tambahkan CLI flags di main.go:
   --migrate         : jalankan SQLite migrations
   --encrypt-config  : enkripsi credential interaktif  
   --check-all       : test semua koneksi, print tabel status
   --install-service : install Windows Service (hanya Windows)
   --version         : print versi

4. Implementasi --check-all:
   Print tabel ke stdout:
   ┌─────────────────────┬─────────────┬────────────────────┐
   │ Komponen            │ Status      │ Detail             │
   ├─────────────────────┼─────────────┼────────────────────┤
   │ BPJS VClaim API     │ ✓ OK        │ Latency: 245ms     │
   │ BPJS Antrol         │ ✓ OK        │ Latency: 312ms     │
   │ SIMRS Khanza        │ ✗ GAGAL     │ Connection refused │
   │ Frista card reader  │ ✓ Tersedia  │ /dev/tty.usb... (Mac) / COM3 (Win) │
   │ Fingerprint BPJS    │ ✓ Tersedia  │ After.exe ditemukan│
   │ Printer thermal     │ ✓ Tersedia  │ USB, 80mm          │
   │ SQLite database     │ ✓ OK        │ WAL mode aktif     │
   └─────────────────────┴─────────────┴────────────────────┘
```

---

## QUICK REFERENCE — Urutan Eksekusi Prompt

```
Fase 0 Setup:    P-001 → P-002
Fase 1 Hardware: P-010 → P-011
Fase 2 BPJS:     P-020 → P-021 → P-022
Fase 3 Detector: P-030
Fase 4 Services: P-040 → P-041
Fase 5 Offline:  P-050 → P-051
Fase 6 Wails:    P-060
Fase 7 Frontend: P-070 → P-071 → P-072 → P-073 → P-074 → P-075
Fase 8 Security: P-080 → P-081
Fase 9 Deploy:   P-090

Testing per fase: setiap prompt sudah include instruksi test-nya.
Setelah setiap fase: jalankan `make test` untuk verifikasi.
```

