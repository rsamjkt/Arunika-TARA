# PROMPTS LENGKAP — APM-Go Claude Code

> Cara pakai: buka Claude Code di folder proyek, lalu copy-paste prompt yang sesuai task.
> Claude Code otomatis baca CLAUDE.md di root folder setiap sesi.

---

## URUTAN EKSEKUSI YANG BENAR

```
P-000  Setup Mac (SEKALI)
P-001  Init proyek
P-002  Domain layer
P-003  Config & SQLite
         ↓
P-010  VClaim client
P-011  Smart Detector
P-012  Test Detector
         ↓
P-020  Khanza client
P-021  Antrian service
P-022  SEP service
         ↓
P-030  Hardware provider
P-031  Frista mock (Mac)
P-032  Fingerprint mock (Mac)
P-033  Printer (console Mac / ESC-POS Windows)
         ↓
P-040  Wails setup + IPC bindings
P-041  Home screen
P-042  Input + Detect screen
P-043  Result screens (4 tipe)
P-044  Antrian screen
P-045  Tiket screen
P-046  Admin panel
         ↓
P-050  Offline & rekonsiliasi
P-051  Security (enkripsi credential + PHI masking)
         ↓
P-060  Build Mac testing
P-061  Cross-compile Windows
P-062  Windows Service installer
```

---

## P-000 — SETUP MAC (jalankan SEKALI sebelum apapun)

```
Saya mau mulai proyek APM-Go di Mac M4. Bantu saya setup semua prerequisites.

Jalankan langkah-langkah ini secara berurutan dan konfirmasi tiap langkah berhasil:

1. Cek apakah Go 1.22+ sudah terinstal: go version
   Jika belum: brew install go

2. Cek apakah Wails v2 sudah terinstal: wails version
   Jika belum: go install github.com/wailsapp/wails/v2/cmd/wails@latest

3. Cek Node.js 18+: node --version
   Jika belum: brew install node

4. Install cross-compile toolchain untuk Windows: brew install mingw-w64

5. Install golangci-lint untuk linting: brew install golangci-lint

6. Verifikasi semua:
   go version        → harus 1.22+
   wails version     → harus v2.x
   node --version    → harus v18+
   x86_64-w64-mingw32-gcc --version  → harus ada output

7. Tampilkan semua hasil verifikasi dalam tabel yang jelas.

Jika ada yang gagal, diagnosa masalahnya dan perbaiki.
```

---

## P-001 — INISIALISASI PROYEK

```
Read CLAUDE.md first.

Buat struktur proyek APM-Go dari nol. Ini adalah aplikasi kiosk rumah sakit 
berbasis Go + Wails v2 + Vue 3.

LANGKAH 1 — Init Wails project:
  wails init -n apm-go -t vue
  cd apm-go

LANGKAH 2 — Buat struktur folder internal sesuai CLAUDE.md:
  mkdir -p internal/{config,domain,service/{detector,antrian,sep,pendaftaran,satusehat}}
  mkdir -p internal/{integration/{vclaim,antrol,mjkn,khanza},hardware/{frista,fingerprint,printer}}
  mkdir -p internal/{store,reconcile,log}
  mkdir -p {migrations,templates}

LANGKAH 3 — Buat go.mod dengan dependencies:
  - github.com/wailsapp/wails/v2
  - github.com/spf13/viper
  - github.com/mattn/go-sqlite3
  - github.com/jmoiron/sqlx
  - github.com/go-resty/resty/v2
  - github.com/robfig/cron/v3
  - github.com/stretchr/testify
  - github.com/stretchr/mockery/v2 (dev)
  Jalankan: go mod tidy

LANGKAH 4 — Buat config.example.toml dengan SEMUA section dari CLAUDE.md.
  PENTING: Jangan taruh credential nyata. Gunakan placeholder.

LANGKAH 5 — Buat Makefile dengan targets:
  dev              → wails dev
  build-mac        → wails build
  build-windows    → GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc wails build -platform windows/amd64
  test             → go test ./... -v -cover
  test-integration → go test ./... -tags integration -v
  lint             → golangci-lint run ./...
  mock-card-read   → curl -X POST http://localhost:9090/mock/card-read -H "Content-Type: application/json" -d '{"nik":"3271234567890001","nama":"Test Pasien","tgl_lahir":"1990-01-15","alamat":"Jl. Test No.1"}'
  mock-fp-fail     → curl -X POST http://localhost:9090/mock/fp-fail

LANGKAH 6 — Buat .gitignore:
  config.toml (ada credential)
  *.exe
  dist/
  build/bin/
  frontend/node_modules/
  frontend/dist/

Setelah selesai, jalankan: make dev
Pastikan window Wails muncul dengan Vue template default.
Tampilkan output terminal yang membuktikan berhasil.
```

---

## P-002 — DOMAIN LAYER (pure structs, no deps)

```
Read CLAUDE.md and BPJS_INTEGRATION.md first.

Buat domain layer di internal/domain/. 
ATURAN KERAS: tidak boleh ada import package eksternal di folder ini. 
Hanya Go stdlib yang boleh.

Buat file-file berikut:

━━━ internal/domain/patient.go ━━━
Structs:
- PatientInput { Identifier string } 
  (bisa NIK 16 digit, No Kartu JKN 16 digit, atau No RM alphanumeric)
- Peserta {
    NoKartu, NoRM, NIK, Nama, TglLahir string
    StatusAktif string  // "1" = aktif
    KelasHak string     // "1", "2", "3"
    JenisPeserta string // "PBI", "PPU", "PBPU", dll
  }
  Method: IsAktif() bool
- SuratKontrol { NoSurat, NoRM, TglRencana, KdPoli, NmPoli, KdDokter string }
  Method: IsTodayOrPast() bool — bandingkan TglRencana dengan time.Now() WIB
- Rujukan { NoSurat, TglRujukan, TglBerlaku, KdPoli, KdDokter, NmFaskes string }
  Method: IsValid() bool — cek TglBerlaku > today

━━━ internal/domain/detection.go ━━━
- PatientType enum (int) dengan 8 nilai:
  Unknown, MJKN, Kontrol, PostRANAP, PostRAJAL, RujukanBaru, TidakAktif, Error
  Tambahkan method String() string untuk setiap nilai (dalam Bahasa Indonesia)
- DetectionResult {
    Type PatientType
    Peserta *Peserta
    Data any       // SuratKontrol, booking MJKN, dll tergantung Type
    Err error
    DetectedAt time.Time
  }
  Method: IsSuccess() bool, UserMessage() string (pesan ramah untuk pasien)

━━━ internal/domain/antrian.go ━━━
- AntrianJenis string enum: "LOKET", "POLI", "UMUM"
- AntrianSubJenis string: "APPOINTMENT", "WALKIN", "RANAP_IGD", "FARMASI", "CS", "KASIR"
- Ticket {
    ID, Nomor, Jenis, SubJenis string
    Prefix string   // "A", "B", "C"
    NoUrut int
    NoRM, NoPoli string  // jika terkait pasien
    CreatedAt time.Time
    PrintedAt *time.Time
  }
  Method: FormatNomor() string → "A-035", "B-DALAM-015", "C-FAR-022"

━━━ internal/domain/sep.go ━━━
- SEPRequest {
    NoKartu, TglSEP, KdPoli, KdDokter string
    JnsPelayanan string  // "1"=RawatJalan, "2"=RawatInap
    KelasRawat string
    NoRujukan string
    CatatanPelayanan string
    FPToken string  // dari fingerprint verification
  }
- SEP {
    NoSEP, NoKartu, TglSEP, KdPoli, NmPoli string
    KdDokter, NmDokter string
    CreatedAt time.Time
  }

━━━ internal/domain/errors.go ━━━
Buat error types spesifik dengan pesan Bahasa Indonesia:
- ErrPesertaTidakAktif
- ErrRujukanExpired  
- ErrDuplikasiSEP
- ErrBiometrikDiperlukan
- ErrJadwalKontrolBelumTiba (sertakan TglRencana di error message)
- ErrSuratKontrolTidakDitemukan
- ErrDokterCuti
- ErrKuotaPenuH
- ErrOffline

Tulis unit test untuk semua method (IsAktif, IsTodayOrPast, IsValid, FormatNomor, dll).
Jalankan: go test ./internal/domain/... -v
Semua harus hijau.
```

---

## P-003 — CONFIG & SQLITE

```
Read CLAUDE.md and HARDWARE_PLATFORM.md first.

Buat dua hal: config loader dan SQLite store.

━━━ BAGIAN 1: internal/config/config.go ━━━

Struct Config dengan semua field dari config.example.toml:
  AppConfig { IdleTimeoutSec, LogLevel, LogDir, Timezone, Version string }
  ServerConfig { KhanzaURL, KhanzaAPIKey string; TimeoutMs, Retry int }
  BPJSConfig { VClaimURL, ConsID, ConsumerSecret string; AntrolURL string; DetectorTimeoutMs int }
  FingerprintConfig { ExePath, RestURL, UsernameEnc, PasswordEnc string; ScanTimeoutSec, PollIntervalMs int }
  FristaConfig { ExePath, UsernameEnc, PasswordEnc string; ReadTimeoutMs int; RestartOnCrash bool }
  PrinterConfig { Mode, Port string; WidthMm int }
  AntrianConfig { LoketPrefix, PoliPrefix, UmumPrefix, ResetTime string }
  DevConfig { MockHardware bool; MockServerPort int }

Fungsi:
  Load(path string) (*Config, error)  → pakai Viper, TOML
  (c *Config) Validate() error        → cek field wajib tidak kosong

━━━ BAGIAN 2: migrations/001_initial.sql ━━━

Buat 5 tabel SQLite:

antrian_lokal:
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  jenis TEXT NOT NULL,
  sub_jenis TEXT,
  nomor TEXT NOT NULL,
  prefix TEXT NOT NULL,
  no_urut INTEGER NOT NULL,
  no_rm TEXT,
  no_poli TEXT,
  created_at DATETIME DEFAULT (datetime('now','localtime')),
  synced_at DATETIME,
  sync_status TEXT DEFAULT 'pending'

pending_sep:
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  no_kartu TEXT NOT NULL,
  kategori TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  vclaim_response TEXT,
  status TEXT DEFAULT 'pending',
  retry_count INTEGER DEFAULT 0,
  last_error TEXT,
  created_at DATETIME DEFAULT (datetime('now','localtime')),
  confirmed_by TEXT,
  confirmed_at DATETIME

print_history:
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  doc_type TEXT NOT NULL,
  ref_id TEXT,
  escpos_bytes BLOB NOT NULL,
  printed_at DATETIME DEFAULT (datetime('now','localtime')),
  reprint_count INTEGER DEFAULT 0

reconcile_log:
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  table_name TEXT NOT NULL,
  record_id INTEGER NOT NULL,
  action TEXT NOT NULL,
  operator_id TEXT,
  result TEXT,
  timestamp DATETIME DEFAULT (datetime('now','localtime'))

config_cache:
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at DATETIME DEFAULT (datetime('now','localtime'))

━━━ BAGIAN 3: internal/store/ ━━━

Buat query.sql dengan semua query yang dibutuhkan, lalu generate dengan sqlc.
Jika sqlc belum ada: go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

Query yang dibutuhkan:
  InsertAntrian, GetPendingAntrian, MarkAntrianSynced
  InsertPendingSEP, GetPendingSEPs, ConfirmSEP, MarkSEPSynced
  InsertPrintHistory, GetPrintHistory, IncrementReprintCount
  InsertReconcileLog, GetRecentLogs

Jalankan: go test ./internal/config/... ./internal/store/... -v
```

---

## P-010 — VCLAIM CLIENT

```
Read BPJS_INTEGRATION.md section "VClaim API v2.0" secara lengkap dulu.

Buat BPJS VClaim v2.0 client di internal/integration/vclaim/.

━━━ interface.go ━━━
type VClaimClient interface {
  GetPeserta(ctx context.Context, identifier string, tgl time.Time) (*domain.Peserta, error)
  GetRencanaKontrol(ctx context.Context, noKartu string, tgl time.Time) ([]domain.SuratKontrol, error)
  GetRiwayatPelayanan(ctx context.Context, noKartu string, tglAwal, tglAkhir time.Time) ([]domain.RiwayatPelayanan, error)
  ValidasiRujukan(ctx context.Context, noSurat string, tgl time.Time) (*domain.Rujukan, error)
  CreateSEP(ctx context.Context, req domain.SEPRequest) (*domain.SEP, error)
  CreateSEPKontrol(ctx context.Context, req domain.SEPKontrolRequest) (*domain.SEP, error)
}

━━━ client.go ━━━
Struct Client dengan:
  consID, secretKey, baseURL string
  httpClient *resty.Client

Fungsi New(cfg config.BPJSConfig) *Client

━━━ auth.go ━━━
func (c *Client) sign(timestamp int64) string
  → HMAC-SHA256(consID + "&" + timestamp, secretKey) → base64
func (c *Client) headers() map[string]string
  → X-cons-id, X-timestamp, X-signature

━━━ decrypt.go ━━━
func (c *Client) decrypt(ciphertext string) ([]byte, error)
  → AES-256-CBC
  → key = SHA256(secretKey + consID)  [32 bytes]
  → IV  = key[:16]
  → Ciphertext adalah base64 → decode dulu → decrypt → PKCS7 unpad

━━━ peserta.go ━━━
GetPeserta() dengan lookup cascade:
  1. Coba sebagai noKartu: GET /Peserta/noKartu/{identifier}/{tgl}
  2. Jika gagal, coba sebagai NIK: GET /Peserta/nik/{identifier}/{tgl}  
  (No RM tidak dicari ke VClaim, cari ke Khanza dulu)
  Parse response JSON, decrypt dulu, return *domain.Peserta

━━━ error_codes.go ━━━
Map semua VClaim error code ke domain error dan pesan Indonesia
(lihat BPJS_INTEGRATION.md bagian "Business Rules VClaim")

━━━ mock.go ━━━
MockVClaimClient yang implementasi VClaimClient interface.
Gunakan untuk testing. Bisa di-configure response per test case.

━━━ client_test.go ━━━
Test dengan httptest.NewServer sebagai mock BPJS server:
- Test sign() menghasilkan HMAC yang benar
- Test decrypt() dengan test vector yang diketahui
- Test GetPeserta() success path
- Test GetPeserta() ketika peserta tidak aktif → return ErrPesertaTidakAktif
- Test retry behavior saat 500 error
- Test timeout handling

Jalankan: go test ./internal/integration/vclaim/... -v -run TestVClaim
```

---

## P-011 — SMART BPJS DETECTOR

```
Read BPJS_INTEGRATION.md section "Smart BPJS Detector Engine" secara lengkap.
Read juga internal/domain/detection.go yang sudah dibuat.

Ini adalah komponen terpenting APM. Implement dengan sangat teliti.

━━━ internal/service/detector/detector.go ━━━

Struct Detector {
  vclaim  vclaim.VClaimClient
  antrol  antrol.AntrolClient  
  khanza  khanza.KhanzaClient
}

func New(vclaim, antrol, khanza) *Detector

func (d *Detector) Detect(ctx context.Context, input domain.PatientInput) domain.DetectionResult:
  
  STEP 1 — Serial, wajib dulu:
    peserta, err := d.vclaim.GetPeserta(ctx, input.Identifier, today)
    jika err → return DetectionResult{Type: Error}
    jika !peserta.IsAktif() → return DetectionResult{Type: TidakAktif}
  
  STEP 2 — 4 goroutine paralel, timeout 5 detik:
    Buat context dengan timeout 5 detik
    Jalankan 4 goroutine: checkMJKN, checkKontrol, checkPostRANAP, checkPostRAJAL
    Kumpulkan hasilnya via channel
  
  STEP 3 — Priority resolution:
    MJKN > Kontrol > PostRANAP > PostRAJAL > RujukanBaru

━━━ mjkn_check.go ━━━
func (d *Detector) checkMJKN(ctx, noKartu string, ch chan<- checkResult)
  → call antrol.GetBookingHariIni(ctx, noKartu, today)
  → jika ada booking aktif → kirim hit=true ke channel

━━━ kontrol_check.go ━━━
func (d *Detector) checkKontrol(ctx, noRM string, ch chan<- checkResult)
  → call khanza.GetSuratKontrol(ctx, noRM)
  → filter: hanya yang TglRencana == today atau sudah lewat (tidak futuro)
  → jika ada → hit=true

━━━ ranap_check.go ━━━
func (d *Detector) checkPostRANAP(ctx, noRM string, ch chan<- checkResult)
  → call khanza.GetRiwayatRANAP(ctx, noRM)
  → filter: TglKeluar == yesterday atau today
  → jika ada → hit=true

━━━ rajal_check.go ━━━
func (d *Detector) checkPostRAJAL(ctx, noRM string, ch chan<- checkResult)
  → call khanza.GetKunjunganAktif(ctx, noRM)
  → filter: ada kunjungan RAJAL aktif yang punya SKDP beda poli
  → jika ada → hit=true

━━━ PENTING — Edge cases yang HARUS dihandle: ━━━
- Jika salah satu goroutine panic → recover, treat as miss (bukan error total)
- Jika semua goroutine timeout → return RujukanBaru (bukan Error)
- Jika 2 check return hit → ambil yang prioritasnya lebih tinggi
- Context dari caller sudah di-cancel → stop semua goroutine segera
- Log setiap check result dengan slog (tanpa PHI — mask noKartu)

Jalankan: go test ./internal/service/detector/... -v
```

---

## P-012 — TEST DETECTOR LENGKAP

```
Read BPJS_INTEGRATION.md dan internal/service/detector/ yang sudah dibuat.

Tulis test suite LENGKAP untuk Smart Detector.
Gunakan table-driven tests.

Test cases yang WAJIB ada:

━━━ Happy paths ━━━
| Nama                  | MJKN | Kontrol | RANAP | RAJAL | Expected      |
|-----------------------|------|---------|-------|-------|---------------|
| only_mjkn             |  ✓   |   ✗     |   ✗   |   ✗   | MJKN          |
| only_kontrol          |  ✗   |   ✓     |   ✗   |   ✗   | Kontrol       |
| only_ranap            |  ✗   |   ✗     |   ✓   |   ✗   | PostRANAP     |
| only_rajal            |  ✗   |   ✗     |   ✗   |   ✓   | PostRAJAL     |
| none_hit              |  ✗   |   ✗     |   ✗   |   ✗   | RujukanBaru   |

━━━ Priority tests ━━━
| Nama                  | MJKN | Kontrol | RANAP | RAJAL | Expected      |
|-----------------------|------|---------|-------|-------|---------------|
| mjkn_beats_all        |  ✓   |   ✓     |   ✓   |   ✓   | MJKN          |
| kontrol_beats_ranap   |  ✗   |   ✓     |   ✓   |   ✓   | Kontrol       |
| ranap_beats_rajal     |  ✗   |   ✗     |   ✓   |   ✓   | PostRANAP     |

━━━ Error paths ━━━
- peserta_tidak_aktif: VClaim return status "0" → TidakAktif (tidak jalankan parallel checks)
- vclaim_network_error: VClaim timeout → Error
- all_checks_timeout: semua goroutine timeout 5 detik → RujukanBaru (bukan Error!)
- partial_failure: 2 dari 4 check error, 2 lainnya miss → RujukanBaru
- partial_failure_with_hit: 2 error, 1 miss, 1 hit → gunakan yang hit

━━━ Concurrency tests ━━━
- concurrent_detect: jalankan 10 Detect() bersamaan, semua harus return hasil benar
- no_goroutine_leak: setelah Detect() selesai, semua goroutine harus sudah berhenti
  (gunakan goleak atau runtime.NumGoroutine() sebelum dan sesudah)

━━━ Business rule tests ━━━
- kontrol_futuro: surat kontrol ada tapi TglRencana besok → miss (bukan hit!)
- ranap_lama: pasien keluar RANAP 7 hari lalu → miss (bukan post-RANAP)

Untuk setiap test: setup mock, jalankan Detect(), assert Type dan Data.
Jalankan: go test ./internal/service/detector/... -v -race -count=3
Harus hijau semua, tidak ada race condition.
```

---

## P-020 — KHANZA CLIENT

```
Read CLAUDE.md section "SIMRS Khanza (Laravel REST API)".

Buat Khanza API client di internal/integration/khanza/.

━━━ interface.go ━━━
type KhanzaClient interface {
  CariPasien(ctx context.Context, q string) (*domain.Pasien, error)
  GetSuratKontrol(ctx context.Context, noRM string) ([]domain.SuratKontrol, error)
  GetRiwayatRANAP(ctx context.Context, noRM string) ([]domain.RiwayatRANAP, error)
  GetKunjunganAktif(ctx context.Context, noRM string) ([]domain.Kunjungan, error)
  GetJadwalDokter(ctx context.Context, kdPoli string, tgl time.Time) ([]domain.JadwalDokter, error)
  BuatPendaftaran(ctx context.Context, req domain.PendaftaranRequest) (*domain.Pendaftaran, error)
  BuatAntrian(ctx context.Context, req domain.AntrianRequest) (*domain.Ticket, error)
  SimpanSEP(ctx context.Context, sep domain.SEP) error
  UpdateSatuSehatID(ctx context.Context, noRM, ihsNumber string) error
}

━━━ client.go ━━━
- Bearer token auth (X-API-Key header)
- Timeout: 10 detik per request
- Retry: 2x dengan 500ms backoff hanya untuk 5xx
- Jika 401: log warning "Khanza API key expired atau salah"
- Jika connection refused: return domain.ErrOffline (bukan panic)

━━━ mock.go ━━━
MockKhanzaClient yang bisa dikonfigurasi per test.
Method: SetResponse(method string, response any, err error)

━━━ client_test.go ━━━
Test dengan httptest.NewServer.
Wajib test skenario offline (server mati → return ErrOffline).
```

---

## P-021 — ANTRIAN SERVICE

```
Read CLAUDE.md section "Sistem Antrian 3 Jalur".
Read internal/domain/antrian.go yang sudah dibuat.

Buat antrian service di internal/service/antrian/.

━━━ service.go ━━━

Struct AntrianService { khanza KhanzaClient; store *store.Queries; antrol AntrolClient }

func (s *AntrianService) Create(ctx context.Context, req domain.AntrianRequest) (*domain.Ticket, error):
  1. Call khanza.BuatAntrian() untuk get nomor (atomic, anti-duplikasi multi-kiosk)
  2. Jika khanza offline:
     - Generate nomor dari SQLite lokal (counter per jenis)
     - Simpan ke antrian_lokal dengan status="pending"
     - Return ticket dengan flag IsOffline=true
  3. Simpan ke print_history
  4. Push ke Antrol API (fire-and-forget — error tidak blokir return)
  5. Return ticket

func (s *AntrianService) GetCounter(ctx, jenis string) (int, error)
  → ambil nomor antrian terakhir hari ini per jenis

func (s *AntrianService) ResetAll(ctx context.Context) error
  → reset semua counter (panggil dari admin panel atau cron job)

━━━ cron.go ━━━
Setup cron job reset harian:
  - Gunakan robfig/cron/v3
  - Schedule: "1 0 * * *" (00:01 setiap hari)
  - Timezone: Asia/Jakarta (WIB)
  - Panggil ResetAll()
  - Log hasil reset

━━━ service_test.go ━━━
Test Create() dalam 3 skenario:
  1. Online: Khanza tersedia → nomor dari server
  2. Offline: Khanza down → nomor dari SQLite lokal
  3. Concurrent: 5 goroutine Create() bersamaan → tidak ada nomor duplikat
```

---

## P-022 — SEP SERVICE

```
Read BPJS_INTEGRATION.md section "VClaim API" dan domain/sep.go.

Buat SEP service di internal/service/sep/.

━━━ service.go ━━━

func (s *SEPService) BuatSEPRujukan(ctx context.Context, req domain.SEPRequest) (*domain.SEP, error):
  1. Validasi biometrik jika diperlukan (usia ≥17 th + non-IGD)
     → call fingerprint.Verify() jika wajib
     → jika gagal → return ErrBiometrikDiperlukan
  2. Validasi rujukan masih berlaku: vclaim.ValidasiRujukan()
  3. Call vclaim.CreateSEP()
  4. Simpan ke SQLite lokal (print_history + backup)
  5. Post ke Khanza: khanza.SimpanSEP()
     → jika Khanza offline: simpan ke pending_sep dengan status="pending"
  6. Return SEP

func (s *SEPService) BuatSEPKontrol(ctx, noSuratKontrol string) (*domain.SEP, error):
  1. Cari surat kontrol dari Khanza
  2. Validasi tanggal: !suratKontrol.IsTodayOrPast() → return ErrJadwalKontrolBelumTiba
  3. Call vclaim.CreateSEPKontrol()
  4. Simpan dan return

func (s *SEPService) BuatSEPPostRANAP(ctx, noRM, kdPoliKontrol string) (*domain.SEP, error)
func (s *SEPService) BuatSEPPostRAJAL(ctx, noRM, kdPoliTujuan string) (*domain.SEP, error)

━━━ fingerprint_check.go ━━━
func perluBiometrik(peserta domain.Peserta, kdPoli string) bool:
  - Usia < 17 tahun → false
  - kdPoli adalah IGD → false  
  - Semua lainnya → true
```

---

## P-030 — HARDWARE PROVIDER

```
Read HARDWARE_PLATFORM.md section "Platform Detection Pattern" secara lengkap.

Buat hardware provider layer di internal/hardware/.

━━━ provider.go ━━━
Ini adalah SATU-SATUNYA tempat runtime.GOOS check dilakukan.

type Provider struct {
  Frista      frista.CardReader
  Fingerprint fingerprint.FingerprintVerifier
  Printer     printer.ThermalPrinter
}

func NewProvider(cfg config.Config) *Provider {
  switch runtime.GOOS {
  case "windows":
    return &Provider{
      Frista:      frista.NewWindowsReader(cfg.Frista),
      Fingerprint: fingerprint.NewWindowsHeadless(cfg.Fingerprint),
      Printer:     printer.NewESCPOS(cfg.Printer),
    }
  default: // darwin (Mac), linux
    return &Provider{
      Frista:      frista.NewMock(cfg.Dev.MockServerPort),
      Fingerprint: fingerprint.NewMock(),
      Printer:     printer.NewConsolePrinter(),
    }
  }
}

━━━ frista/interface.go ━━━
type CardReader interface {
  Start(ctx context.Context) error
  Stop() error
  IsAvailable() bool
  CardRead() <-chan CardData
}
type CardData struct { NIK, Nama, TglLahir, Alamat, NoKartu string }

━━━ fingerprint/interface.go ━━━  
type FingerprintVerifier interface {
  Verify(ctx context.Context, noPeserta string) (FPResult, error)
  IsAvailable() bool
}
type FPResult struct { Success bool; Token string; Timestamp time.Time }

━━━ printer/interface.go ━━━
type ThermalPrinter interface {
  Print(docType string, data any) error
  IsAvailable() bool
  Reprint(printHistoryID int64) error
}
```

---

## P-031 — FRISTA MOCK + HTTP SERVER (Mac)

```
Read HARDWARE_PLATFORM.md section "Mac Development".

Buat Frista mock untuk development di Mac.

━━━ internal/hardware/frista/mock.go ━━━

type MockFristaReader struct {
  ch         chan CardData
  serverPort int
  server     *http.Server
}

func NewMock(port int) *MockFristaReader

func (m *MockFristaReader) Start(ctx context.Context) error:
  - Buat channel CardData dengan buffer 5
  - Start HTTP server di port yang dikonfigurasi (default 9090)
  - Routes:
    POST /mock/card-read        → inject CardData ke channel langsung
    POST /mock/card-read-delay  → inject setelah delay (query param: seconds=3)
    GET  /                      → HTML halaman info endpoint (untuk developer)
  - Log: "🔌 Frista mock aktif di http://localhost:{port}"

func (m *MockFristaReader) CardRead() <-chan CardData:
  return m.ch

━━━ Halaman info (GET /) ━━━
Tampilkan HTML sederhana dengan instruksi curl:
  curl -X POST http://localhost:9090/mock/card-read \
    -H "Content-Type: application/json" \
    -d '{"nik":"3271234567890001","nama":"Budi Santoso","tgl_lahir":"1990-01-15","alamat":"Jl. Merdeka No.1","no_kartu":"0001234567890012"}'

━━━ internal/hardware/frista/mock_test.go ━━━
Test:
  - Start mock, POST ke endpoint, cek data muncul di CardRead() channel
  - Test delay endpoint
  - Test server cleanup saat Stop() dipanggil

━━━ Makefile targets (tambahkan) ━━━
mock-card-read:
  @curl -s -X POST http://localhost:9090/mock/card-read \
    -H "Content-Type: application/json" \
    -d '{"nik":"$(NIK)","nama":"$(NAMA)","tgl_lahir":"1990-01-15","alamat":"Jl. Test No.1","no_kartu":"$(KARTU)"}' \
    | echo "✅ Card read injected"

mock-card-default:
  @$(MAKE) mock-card-read NIK=3271234567890001 NAMA="Budi Santoso" KARTU=0001234567890012
```

---

## P-032 — FINGERPRINT MOCK + WINDOWS HEADLESS

```
Read HARDWARE_PLATFORM.md section "Fingerprint BPJS — Headless Auto-Submit".

━━━ internal/hardware/fingerprint/mock.go ━━━

type MockVerifier struct {
  forceFailNext bool
  mu            sync.Mutex
}

func NewMock() *MockVerifier

func (m *MockVerifier) Verify(ctx context.Context, noPeserta string) (FPResult, error):
  m.mu.Lock()
  fail := m.forceFailNext
  m.forceFailNext = false
  m.mu.Unlock()
  
  if fail {
    return FPResult{}, errors.New("simulasi: verifikasi sidik jari gagal")
  }
  
  // Simulasi delay scan (2 detik)
  select {
  case <-time.After(2 * time.Second):
    return FPResult{
      Success: true,
      Token: "MOCK_FP_" + noPeserta + "_" + time.Now().Format("150405"),
    }, nil
  case <-ctx.Done():
    return FPResult{}, ctx.Err()
  }

Method SetNextFail() → set forceFailNext = true
  (dipanggil dari HTTP endpoint /mock/fp-fail)

━━━ Tambahkan ke mock server Frista ━━━
Tambahkan route POST /mock/fp-fail di HTTP server Frista mock
(karena mereka satu server di port 9090):
  → panggil fingerprintMock.SetNextFail()

━━━ internal/hardware/fingerprint/windows.go ━━━
// +build windows

Struct WindowsHeadlessVerifier dengan:
  cfg    config.FingerprintConfig
  cmd    *exec.Cmd
  client *resty.Client

func (w *WindowsHeadlessVerifier) ensureRunning() error:
  - Spawn After.exe dengan CREATE_NO_WINDOW flag
  - Wait 3 detik untuk app load
  - Inject login via Windows UI Automation (lihat catatan di bawah)

func (w *WindowsHeadlessVerifier) Verify(ctx context.Context, noPeserta string) (FPResult, error):
  - ensureRunning() jika belum
  - POST ke fp.bpjs-kesehatan.go.id/finger-rest/ dengan credential
  - Poll GET /api/fingerprint/status setiap 500ms sampai sukses atau timeout
  - Return FPResult

CATATAN Windows UI Automation:
  Untuk inject login ke After.exe, gunakan user32.dll via syscall:
  FindWindowW → FindWindowExW → SendMessageW(WM_SETTEXT)
  Ini CGO — taruh di file terpisah windows_ui.go dengan build tag windows.
  Buat stub kosong di windows_ui_stub.go untuk non-Windows agar compile.
```

---

## P-033 — PRINTER (Console Mac + ESC-POS Windows)

```
Read HARDWARE_PLATFORM.md section "Thermal Printer".
Read templates/ folder.

━━━ internal/hardware/printer/console.go ━━━
ConsolePrinter untuk development Mac.
Print ke stdout dalam format yang mudah dibaca:

func (p *ConsolePrinter) Print(docType string, data any) error:
  fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━")
  fmt.Printf("  [CETAK] %s\n", docType)
  // render template jadi teks, print baris per baris
  fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━")
  
  // Simpan ke print_history (raw bytes = template output)
  return nil

━━━ templates/ ━━━
Buat Go text/template untuk setiap dokumen:

templates/tiket_antrian.tmpl:
  Rumah Sakit: {{.RSName}}
  Tanggal: {{.Tanggal}}
  ─────────────────
  {{.JenisAntrian}}
  No. Antrian:
  {{.Nomor}}
  ─────────────────
  Harap tunggu panggilan

templates/sep.tmpl:
  SURAT ELIGIBILITAS PESERTA
  No. SEP: {{.NoSEP}}
  Nama: {{.Nama}}
  No. Kartu: {{.NoKartu}}
  Poli: {{.NmPoli}}
  Dokter: {{.NmDokter}}
  Tgl: {{.TglSEP}}
  Kelas: {{.KelasRawat}}

templates/registrasi.tmpl:
  BUKTI PENDAFTARAN
  No. Rawat: {{.NoRawat}}
  Nama: {{.Nama}}
  Poli: {{.NmPoli}}
  Dokter: {{.NmDokter}}
  Tgl: {{.TglKunjungan}}
  Antrian: {{.NoAntrian}}

━━━ internal/hardware/printer/escpos.go ━━━
ESCPOSPrinter untuk Windows production.
Gunakan library: github.com/kenshaw/escpos atau direct USB write via gousb.

Fungsi utama:
  connect() error    → buka koneksi USB/Serial/Network
  Print(docType, data) error → render template → convert ke ESC/POS bytes → write
  
ESC/POS commands yang dipakai:
  ESC @ → reset printer
  ESC E 1 → bold on
  ESC E 0 → bold off  
  ESC a 1 → center align
  ESC a 0 → left align
  GS ! n  → font size (double height untuk nomor antrian)
  LF      → line feed
  GS V 0  → cut paper
```

---

## P-040 — WAILS SETUP + IPC BINDINGS

```
Read CLAUDE.md section "Wails IPC Binding Conventions".
Read DESIGN_SYSTEM.md section "Pinia Stores".

Buat Wails app bindings di cmd/apm/.

━━━ cmd/apm/main.go ━━━
Setup Wails app dengan:
  - Window title: "Anjungan Pasien Mandiri"
  - Window size: 1024x768 minimum, resizable true
  - Fullscreen: baca dari config (default true untuk production)
  - Background: #F5F6F8 (warna bg kiosk)

━━━ cmd/apm/app.go ━━━
Struct App dengan semua service dan hardware provider.

Expose method-method ini ke Vue (SEMUA exported, return (data, error)):

// Detection
func (a *App) DetectPatient(identifier string) (*domain.DetectionResult, error)

// Antrian  
func (a *App) CreateAntrian(jenis, subJenis string) (*domain.Ticket, error)
func (a *App) GetCounters() (map[string]int, error)

// SEP
func (a *App) BuatSEPRujukan(req SEPRequest) (*domain.SEP, error)
func (a *App) BuatSEPKontrol(noSuratKontrol string) (*domain.SEP, error)
func (a *App) BuatSEPPostRANAP(noRM, kdPoli string) (*domain.SEP, error)

// Pendaftaran umum
func (a *App) CariPasien(q string) (*domain.Pasien, error)
func (a *App) BuatPendaftaran(req PendaftaranRequest) (*domain.Pendaftaran, error)

// Hardware status
func (a *App) GetHardwareStatus() HardwareStatus
func (a *App) GetSystemStatus() SystemStatus

// Reprint
func (a *App) Reprint(printHistoryID int64) error

// Admin
func (a *App) GetPendingSEPs() ([]store.PendingSEP, error)
func (a *App) ConfirmSEPSync(id int64) error
func (a *App) ResetCounters() error

━━━ Wails Events (Go → Vue) ━━━
Emit events ini dari Go ke Vue menggunakan runtime.EventsEmit():

  "frista:card_read"    → CardData (saat kartu ditempel)
  "detect:step_update"  → {step: string, state: "done"|"active"|"wait"}
  "system:offline"      → bool (true=offline, false=online kembali)
  "printer:error"       → string (pesan error printer)
  "hardware:status"     → HardwareStatus

━━━ frontend/src/wailsjs/ ━━━
Generate TypeScript types:
  wails generate module
Output: wailsjs/go/main/App.d.ts dengan semua binding

━━━ frontend/src/services/apm.ts ━━━
Wrapper TypeScript untuk semua Wails calls:
import { DetectPatient, CreateAntrian, ... } from '../wailsjs/go/main/App'
import { EventsOn } from '../wailsjs/runtime/runtime'

export const apmService = {
  detect: (id: string) => DetectPatient(id),
  createAntrian: (jenis: string, subJenis: string) => CreateAntrian(jenis, subJenis),
  // dll
}

export const useWailsEvents = () => {
  const onCardRead = (handler: (data: CardData) => void) => 
    EventsOn('frista:card_read', handler)
  // dll
}
```

---

## P-041 — HOME SCREEN (Vue)

```
Read DESIGN_SYSTEM.md LENGKAP sebelum menulis satu baris Vue.

Buat frontend/src/screens/HomeScreen.vue.

DESIGN SPEC (WAJIB diikuti persis):
- Background: #F5F6F8
- Header: putih, border-bottom 0.5px #E4E6EA
  - Kiri: logo mark biru 30x30px (border-radius 8px) + nama RS + tagline
  - Kanan: status dots (BPJS + Sistem) + jam digital live
- Body: 4 button area dengan padding clamp(12px, 2.5vw, 20px)
  - Hero button BPJS: full width, background #1B4FD8, icon kartu + text + tag "otomatis mendeteksi"
  - Grid 2 kolom: Pasien Umum (icon biru) + Ambil Antrian (icon abu)
  - Full width: Aktivasi Satu Sehat
- Footer: border-top, text "Butuh bantuan?" + tombol "Panggil petugas"

SEMUA ukuran WAJIB pakai clamp():
  font-size: clamp(14px, 2.2vw, 18px)  → teks hero
  font-size: clamp(11px, 1.6vw, 14px)  → teks card
  padding: clamp(12px, 2.5vw, 20px)
  min-height: clamp(52px, 7vw, 72px)   → touch target

Idle timeout:
  - 60 detik tidak ada interaksi → reset ke home
  - 10 detik terakhir → overlay countdown "Kembali ke awal dalam X detik"
  - Reset timer setiap ada mousemove, touchstart, keydown
  - Baca idleTimeoutSec dari AppConfig via Wails

Frista event listener:
  onMounted: EventsOn('frista:card_read', (data) => router.push('/detect?from=frista&id='+data.noKartu))

Gunakan Pinia usePatientStore untuk state.
Gunakan Vue Router untuk navigasi.
Jangan ada hardcoded string — semua dari i18n atau constants.
```

---

## P-042 — INPUT SCREEN + DETECT SCREEN (Vue)

```
Read DESIGN_SYSTEM.md section "Komponen Wajib".
Read BPJS_INTEGRATION.md section "Cara Kerja" untuk memahami step detection.

━━━ frontend/src/screens/InputScreen.vue ━━━

Input area:
  - Display nomor yang diketik: font monospace, clamp(18px, 3.5vw, 26px)
  - Format otomatis: setiap 4 digit → spasi (1234 5678 9012 3456)
  - Placeholder: "_ _ _ _  _ _ _ _  _ _ _ _  _ _ _ _" (abu muted)
  - Cursor blink animation

NumPad:
  - Grid 3 kolom
  - Tombol 1-9, 0, hapus, cari
  - min-height: clamp(52px, 7vw, 72px) WAJIB
  - Tombol "cari": background #1B4FD8, color putih
  - Tombol "hapus": background #F5F6F8

Frista bar (selalu tampil):
  - Background #ECFDF5, border #6EE7B7
  - Dot hijau animasi pulse
  - Teks: "Frista aktif — tempel kartu atau KTP untuk isi otomatis"
  - Jika Frista tidak available: dot abu, teks "Frista tidak terhubung"

Chip hints di bawah input:
  "16 digit no. JKN" | "16 digit NIK KTP" | "No. rekam medis"

On submit (tombol cari atau Enter):
  - Validasi: minimal 6 karakter
  - router.push('/detecting') + trigger detect di background

━━━ frontend/src/screens/DetectScreen.vue ━━━

Progress ring (SVG, BUKAN CSS gradient):
  <svg><circle track/><circle arc class="animate-spin-arc"/></svg>
  
Step list (5 steps, update realtime dari Wails event):
  step 1: "Verifikasi status BPJS"
  step 2: "Cek booking Mobile JKN"
  step 3: "Cek jadwal kontrol"
  step 4: "Cek riwayat rawat inap"
  step 5: "Menentukan jenis kunjungan"
  
State setiap step: "wait" | "active" | "done"
  done  → dot hijau + teks hijau + ✓ di kanan
  active → dot biru, animasi pulse
  wait  → dot abu

Subscribe ke event: EventsOn('detect:step_update', ...)
Timeout guard: jika 7 detik belum selesai → tampilkan "Sedang memproses, mohon tunggu..."
Auto-navigate ke ResultScreen saat detection selesai.
```

---

## P-043 — RESULT SCREENS (4 tipe)

```
Read DESIGN_SYSTEM.md.
Read BPJS_INTEGRATION.md section "Hasil Tampilan UI per Kategori".

Buat satu file: frontend/src/screens/ResultScreen.vue
yang render berbeda berdasarkan detectionResult.type dari Pinia store.

━━━ KOMPONEN BERSAMA ━━━

PatientCard (tampil di semua result):
  - Pill status (kiri): "Booking Mobile JKN" | "Jadwal kontrol" | "Kunjungan baru" | "Tidak aktif"
  - Warna pill:
    MJKN       → background #ECFDF5, color #065F46, dot hijau
    Kontrol    → background #EEF2FF, color #1E40AF, dot biru
    RujukanBaru → background #FFFBEB, color #92400E, dot kuning
    TidakAktif → background #FEF2F2, color #991B1B, dot merah
  - Nama pasien: clamp(15px, 2.5vw, 19px), font-weight 500
  - No kartu: monospace, clamp(10px, 1.4vw, 12px), color muted
  - Divider: border-top 0.5px
  - Key-value rows: label kiri (muted) + value kanan (bold)

━━━ MJKN RESULT ━━━
Tampilkan: Poli, Dokter, Estimasi jam (warna biru), No antrian booking
Info bar hijau: "Booking dari Mobile JKN terkonfirmasi. Cetak tiket untuk konfirmasi kedatangan."
CTA: "Konfirmasi kedatangan dan cetak tiket" → call BuatCheckinMJKN() → navigate ke TicketScreen
Ghost: "Bukan saya — masukkan ulang" → back ke InputScreen

━━━ KONTROL RESULT ━━━
Tampilkan: No surat kontrol, Poli, Dokter sebelumnya
Dokter picker: list dokter yang bertugas hari ini di poli tersebut
  → call GetJadwalDokter() saat screen mount
  → highlight pilihan pertama sebagai default (selected)
CTA: "Buat surat layanan kontrol dan cetak" → BuatSEPKontrol()

━━━ RUJUKAN BARU RESULT ━━━  
Tampilkan: Kelas hak, No rujukan FKTP, Poli rujukan, Berlaku sampai
Info bar kuning: "Verifikasi sidik jari diperlukan setelah pilih dokter."
  (hanya tampil jika perluBiometrik() = true)
CTA: "Pilih dokter dan lanjutkan" → navigate ke DokterPickerScreen

━━━ TIDAK AKTIF RESULT ━━━
Info bar merah: full message tentang cara aktivasi BPJS
CTA primary: "Daftar sebagai pasien umum"
CTA ghost: "Hubungi petugas untuk bantuan"

━━━ STATE LOADING & ERROR ━━━
Saat API call berjalan: CTA button disabled + spinner
Jika error: AlertModal dengan pesan error dari domain.UserMessage()
```

---

## P-044 — ANTRIAN SCREEN (Vue)

```
Read DESIGN_SYSTEM.md section "Komponen Wajib".

Buat frontend/src/screens/AntrianScreen.vue.

Layout WAJIB (sesuai approved design):
  Section label "Antrian loket (A)" — uppercase 11px, color muted, letter-spacing 0.5px
  Grid 2 kolom:
    Card: Admisi appointment + counter "Sekarang: A-012"
    Card: Admisi walk-in + counter "Sekarang: A-034"
  Card full width: Rawat inap & IGD + counter

  Section label "Antrian layanan umum (C)"
  Grid 2 kolom:
    Card: Farmasi / apotek + counter "Sekarang: C-022"
    Card: Customer service + counter "Sekarang: C-008"

Setiap AntrianCard:
  - background putih, border 0.5px #E4E6EA
  - border-radius 12px
  - padding clamp(12px, 2vw, 16px)
  - icon dalam container tinted (biru atau hijau)
  - icon SVG (bukan emoji) 16-18px
  - judul: clamp(11px, 1.6vw, 13px) font-weight 500
  - counter "Sekarang: X-XXX": clamp(10px, 1.3vw, 12px) color muted
  - hover: border-color #CDD1D9
  - active: background #FAFBFC
  - loading state: disabled + spinner overlay pada card yang diklik

On tap card:
  1. Set card state = loading
  2. Call CreateAntrian(jenis, subJenis)
  3. Jika sukses → navigate ke TicketScreen dengan ticket data
  4. Jika error → AlertModal "Gagal mengambil nomor antrian. Coba lagi."

Counter update:
  - Fetch GetCounters() saat screen mount
  - Refresh setiap 30 detik (setInterval)

Konfigurasi label & prefix dari AppConfig (bisa beda tiap RS):
  antrianConfig.loketPrefix, antrianConfig.umumPrefix, dll
  Label jenis antrian dari config juga (bukan hardcoded)
```

---

## P-045 — TIKET SCREEN (Vue)

```
Buat frontend/src/screens/TicketScreen.vue.

Tampilan setelah SEP/pendaftaran/antrian berhasil.

Layout (center aligned, max-width 380px, margin auto):

  Check circle (44-50px) → background #ECFDF5, border #6EE7B7
    SVG checkmark stroke #065F46

  Teks sukses: "Surat layanan berhasil dibuat"
    font-size clamp(12px, 1.8vw, 14px), color #065F46

  Tiket paper (card putih, border 0.5px):
    Label uppercase kecil: "ANTRIAN POLI PENYAKIT DALAM"
    Nomor besar: clamp(44px, 8vw, 60px), font-weight 500, color #0E1117
    Info dokter + tanggal: clamp(11px, 1.5vw, 13px) muted
    Dashed divider (border-top 1px dashed)
    No SEP: monospace, muted + nilai biru

  Info box (background #F5F6F8, border-radius 9px):
    "Silakan menuju area tunggu"
    nama poli + lantai (bold)
    "Nomor Anda akan dipanggil di layar display"
    font-size clamp(10px, 1.4vw, 12px), line-height 1.7

  Countdown: "Kembali ke awal dalam X detik"
    X mulai dari 10, countdown setiap detik
    Saat 0 → navigate ke HomeScreen + reset semua store

  Tombol "Cetak ulang tiket":
    background putih, border 0.5px
    call Reprint(ticket.printHistoryID)

Data tiket dari useAntrianStore atau dari route params (jika dari SEP).
Auto-print dipanggil di onMounted: Print("TIKET", ticketData)
```

---

## P-046 — ADMIN PANEL (Vue)

```
Buat frontend/src/screens/AdminScreen.vue.

Akses: PIN 4-6 digit (dikonfigurasi di config.toml)
Tampilkan PIN pad saat pertama masuk, validasi, baru tampilkan panel.

Layout admin (tidak perlu touchscreen-friendly, ini untuk operator IT):

Header: "Panel admin" kiri + tombol "Keluar" merah kanan

Stat grid (2x2):
  Antrian hari ini: angka besar + "Reset pukul 00:01"
  SEP berhasil: angka + "Hari ini"
  Pending rekonsiliasi: angka (warna warn jika > 0) + "Butuh konfirmasi"
  Uptime: "7j 12m" + "Sejak HH:MM WIB"

Status komponen (card):
  Daftar komponen dengan pill status:
    BPJS VClaim API     → Online/Offline
    BPJS Antrol         → Online/Offline
    SIMRS Khanza        → Online/Offline
    Frista card reader  → Terhubung/Tidak terhubung
    Fingerprint BPJS    → Headless aktif/Tidak aktif
    Printer thermal     → OK/Kertas hampir habis/Tidak terhubung
  Refresh status setiap 10 detik via GetSystemStatus()

Tabel pending SEP (jika ada):
  Kolom: No Kartu (masked), Kategori, Waktu, Status
  Tombol "Konfirmasi" per row → ConfirmSEPSync(id)
  Konfirmasi modal: "SEP ini akan dikirim ke Khanza. Lanjutkan?"

Action buttons (2x2 grid):
  Reset counter antrian → konfirmasi modal dulu
  Lihat log rekonsiliasi → modal dengan tabel log 50 entry terakhir
  Test cetak printer → Print("TEST", {}) 
  Buka mock server info → hanya tampil di non-Windows (dev mode)
```

---

## P-050 — OFFLINE MODE + REKONSILIASI

```
Read CLAUDE.md section "Mode Offline & Rekonsiliasi".
Read internal/store/ yang sudah dibuat.

Buat offline + reconcile system di internal/reconcile/.

━━━ worker.go ━━━

type ReconcileWorker struct {
  db     *store.Queries
  khanza khanza.KhanzaClient
  ticker *time.Ticker
  done   chan struct{}
}

func New(db, khanza) *ReconcileWorker
func (w *ReconcileWorker) Start(ctx context.Context)
func (w *ReconcileWorker) Stop()

Background goroutine (setiap 30 detik):
  1. Cek koneksi: w.khanza.HealthCheck(ctx)
  2. Jika offline: continue (skip)
  3. Jika baru online kembali:
     - Emit Wails event "system:offline" dengan false
     - Log: "Koneksi Khanza pulih, mulai rekonsiliasi"
  4. SyncPendingAntrian(ctx)
  5. SyncConfirmedSEP(ctx)  ← HANYA yang sudah dikonfirmasi operator

func SyncPendingAntrian:
  - Ambil semua antrian_lokal WHERE sync_status = 'pending'
  - Untuk setiap record: POST ke Khanza API
  - Jika sukses: update sync_status = 'synced', synced_at = now()
  - Jika gagal: increment retry_count, set last_error
  - Setelah 5x gagal: set status = 'failed', log warning
  - Insert ke reconcile_log tiap attempt

func SyncConfirmedSEP:
  - Ambil pending_sep WHERE status = 'awaiting_sync'
    (status ini diset setelah operator konfirmasi via admin panel)
  - POST ke Khanza API SimpanSEP()
  - Update status sesuai hasil

━━━ offline_detector.go ━━━
Deteksi offline/online dengan ping ke Khanza health endpoint.
Cache status terakhir. Hanya emit event jika status BERUBAH.
Jangan spam event setiap 30 detik kalau statusnya sama.

━━━ Vue: OfflineBanner.vue ━━━
Komponen banner yang muncul di atas semua screen saat offline:
  background #FFFBEB (warn)
  teks: "Mode offline — antrian disimpan sementara"
  dot kuning pulse

Subscribe ke EventsOn('system:offline') di App.vue.
Tambahkan banner di App.vue secara conditional.
```

---

## P-051 — SECURITY (Enkripsi + PHI Masking)

```
Read CLAUDE.md section "Aturan Coding — Go".

Buat dua security features.

━━━ FITUR 1: Credential Encryption ━━━

internal/config/encrypt.go:

func EncryptConfig(configPath string) error:
  - Baca config.toml
  - Prompt username & password Frista via stdin (tanpa echo)
  - Prompt username & password Fingerprint BPJS via stdin
  - Encrypt setiap nilai dengan AES-256-GCM
  - Master key:
    → macOS: ambil dari Keychain (security find-generic-password -s apm-go)
    → Windows: gunakan Windows DPAPI (machine-bound)
    → Fallback: env var APM_MASTER_KEY (untuk CI/dev)
  - Update config.toml: ganti nilai plaintext dengan "ENC:base64..."
  - Print: "✅ Credential berhasil dienkripsi"

func DecryptValue(encryptedB64 string, masterKey []byte) (string, error):
  - Parse "ENC:" prefix
  - Base64 decode
  - AES-256-GCM decrypt
  - Return plaintext

Panggil DecryptValue di config loader untuk semua field "ENC:...".

CLI entry point di cmd/apm/main.go:
  flag --encrypt-config → panggil EncryptConfig()

━━━ FITUR 2: PHI Log Masking ━━━

internal/log/phi_handler.go:

type PHIMaskingHandler struct {
  inner slog.Handler
}

func NewPHIMaskingHandler(inner slog.Handler) *PHIMaskingHandler

func (h *PHIMaskingHandler) Handle(ctx context.Context, r slog.Record) error:
  // Buat record baru dengan attr yang sudah di-mask
  var maskedAttrs []slog.Attr
  r.Attrs(func(a slog.Attr) bool {
    maskedAttrs = append(maskedAttrs, maskAttr(a))
    return true
  })
  // Build masked record dan forward ke inner handler

func maskAttr(a slog.Attr) slog.Attr:
  // Field names yang selalu di-mask:
  sensitiveFields := []string{"nik", "no_kartu", "no_rm", "username", "password", "token"}
  if contains(sensitiveFields, strings.ToLower(a.Key)) {
    return slog.String(a.Key, "***")
  }
  // Pattern: 16 digit angka berurutan → mask jadi "****1234" (4 digit terakhir saja)
  if isLikelyPHI(a.Value.String()) {
    return slog.String(a.Key, maskPHI(a.Value.String()))
  }
  return a

Setup di main.go:
  baseHandler := slog.NewJSONHandler(logFile, &slog.HandlerOptions{Level: logLevel})
  logger := slog.New(NewPHIMaskingHandler(baseHandler))
  slog.SetDefault(logger)

Test: tulis log dengan NIK dan no kartu → verifikasi output file tidak punya nilai asli.
```

---

## P-060 — BUILD & TEST DI MAC

```
Ini adalah prompt untuk test akhir sebelum kirim ke Windows.

LANGKAH 1 — Pastikan semua test hijau:
  make test
  Tampilkan coverage report. Minimum 70% untuk service/detector dan service/sep.

LANGKAH 2 — Build macOS:
  make build-mac
  Output harus ada di: build/bin/APM.app (atau build/bin/apm-go)

LANGKAH 3 — Jalankan aplikasi:
  open build/bin/APM.app
  ATAU: ./build/bin/apm-go (jika binary, bukan .app)

LANGKAH 4 — Test checklist manual:
  □ Window muncul dengan HomeScreen
  □ Ukuran window bisa di-resize — semua elemen scale dengan benar
  □ Klik "Pasien BPJS" → navigate ke InputScreen
  □ Ketik angka di numpad → muncul di display
  □ Buka terminal baru: make mock-card-default
    → form terisi otomatis
  □ Klik "cari" → DetectScreen muncul dengan animasi
  □ Navigate ke berbagai result screen
  □ Klik "Ambil Antrian" → AntrianScreen
  □ Klik salah satu card → TicketScreen dengan countdown
  □ Tunggu 60 detik tanpa interaksi → overlay countdown → reset ke Home

LANGKAH 5 — Test offline mode:
  Matikan koneksi Mac dari wifi/ethernet
  Coba ambil antrian → harus tetap berhasil (offline mode)
  Nyalakan koneksi kembali → banner offline menghilang

LANGKAH 6 — Lint:
  make lint
  Perbaiki semua warning sebelum build Windows.

Jika semua langkah berhasil, lanjutkan ke P-061.
```

---

## P-061 — CROSS-COMPILE WINDOWS

```
LANGKAH 1 — Verifikasi toolchain:
  which x86_64-w64-mingw32-gcc
  → harus ada output path (jika belum: brew install mingw-w64)

LANGKAH 2 — Build Windows binary:
  make build-windows
  → Ini jalankan: GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc wails build -platform windows/amd64

  Output yang diharapkan:
  build/bin/apm-go.exe  (atau nama sesuai wails.json)

LANGKAH 3 — Buat deployment package:
  mkdir -p dist/apm-windows-amd64
  cp build/bin/apm-go.exe dist/apm-windows-amd64/apm.exe
  cp config.example.toml dist/apm-windows-amd64/
  cp -r migrations/ dist/apm-windows-amd64/
  cp -r templates/ dist/apm-windows-amd64/

LANGKAH 4 — Buat INSTALL.md dalam Bahasa Indonesia:
  Taruh di dist/apm-windows-amd64/INSTALL.md
  Isi:
    1. Prasyarat (OS, RAM, WebView2, Frista.exe, After.exe)
    2. Copy folder ini ke C:\apm\
    3. Edit config.toml — isi IP Khanza, credential BPJS
    4. Enkripsi credential: apm.exe --encrypt-config
    5. Test koneksi: apm.exe --check-connections
    6. Install Windows Service: apm.exe --install-service
    7. Start: net start APMService
    8. Troubleshooting umum

LANGKAH 5 — Zip dan siap kirim:
  cd dist && zip -r apm-windows-v1.0.0.zip apm-windows-amd64/

Tampilkan ukuran file .exe dan .zip akhir.
```

---

## P-062 — WINDOWS SERVICE INSTALLER

```
Read HARDWARE_PLATFORM.md section "Install sebagai Windows Service".

Tambahkan CLI flags ke cmd/apm/main.go untuk Windows deployment.

━━━ CLI Flags ━━━
Gunakan flag package stdlib:

--encrypt-config      → jalankan EncryptConfig() lalu exit
--check-connections   → test semua koneksi, print tabel status, exit
--migrate             → jalankan SQL migrations, exit
--install-service     → install sebagai Windows Service (Windows only)
--uninstall-service   → hapus Windows Service
--version             → print versi dari build info

━━━ --check-connections ━━━
Print tabel seperti ini:
  ┌─────────────────────┬──────────┬─────────────────┐
  │ Komponen            │ Status   │ Detail          │
  ├─────────────────────┼──────────┼─────────────────┤
  │ BPJS VClaim API     │ ✅ OK    │ 245ms           │
  │ BPJS Antrol         │ ✅ OK    │ 312ms           │
  │ SIMRS Khanza        │ ✅ OK    │ 45ms            │
  │ SQLite              │ ✅ OK    │ local           │
  │ Frista.exe          │ ❌ GAGAL │ File not found  │
  │ After.exe (FP BPJS) │ ⚠️ SKIP  │ Dev mode (Mac)  │
  └─────────────────────┴──────────┴─────────────────┘
Exit code 0 jika semua critical OK, 1 jika ada yang critical gagal.

━━━ --install-service (Windows only) ━━━
// +build windows

Gunakan golang.org/x/sys/windows/svc/mgr untuk install service:
  mgr.Connect() → mgr.CreateService("APMService", exePath, ...)
  Service config: StartType=mgr.StartAutomatic, Description="Anjungan Pasien Mandiri"
  Set recovery actions: restart setelah 5 detik jika crash (3x)

━━━ Windows Service wrapper ━━━
Gunakan golang.org/x/sys/windows/svc untuk run sebagai service:
  Implement svc.Handler interface
  Method Execute(): start normal app, listen untuk Stop/Shutdown signal

Test: build Windows binary, jalankan --check-connections dari terminal.
```

