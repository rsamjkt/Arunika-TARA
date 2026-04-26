# T.A.R.A — Total Automated Registration Assistant

> **Arunika · TaRa** — Anjungan Pasien Mandiri (APM) generasi baru untuk rumah sakit Indonesia.
> Reimplementasi Go + Wails dari sistem APM legacy, dengan **Smart BPJS Detector**, integrasi penuh BPJS (VClaim · Antrol · Mobile JKN) dan SIMRS Khanza.

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Wails](https://img.shields.io/badge/Wails-v2-FF0000)](https://wails.io)
[![Vue](https://img.shields.io/badge/Vue-3-4FC08D?logo=vue.js&logoColor=white)](https://vuejs.org)
[![Tailwind](https://img.shields.io/badge/Tailwind-3-38B2AC?logo=tailwind-css&logoColor=white)](https://tailwindcss.com)
[![Platform](https://img.shields.io/badge/Platform-Windows%2010%2F11-0078D4?logo=windows&logoColor=white)]()
[![Dev](https://img.shields.io/badge/Dev-macOS%20%7C%20Linux-000000?logo=apple&logoColor=white)]()

---

## Tentang T.A.R.A

**T.A.R.A** (*Total Automated Registration Assistant*) adalah perangkat lunak Anjungan Pasien Mandiri yang berjalan sebagai **kiosk fullscreen** di rumah sakit. Pasien melakukan registrasi sendiri — cukup tap kartu KTP atau ketik NIK/No. Kartu BPJS — sistem akan otomatis mendeteksi jenis kunjungan, membuat SEP, mendaftar ke poli, dan mencetak tiket antrian.

| Atribut | Detail |
|---|---|
| **Pemilik** | PT. Arunika Komputasi Awan Integrasi |
| **Repo** | [github.com/rsamjkt/Arunika-TARA](https://github.com/rsamjkt/Arunika-TARA) |
| **Stack** | Go 1.22+ · Wails v2 · Vue 3 · Tailwind CSS · SQLite |
| **Target Deploy** | Windows 10/11 x64 (kiosk & loket) |
| **Development** | macOS / Linux (cross-compile ke Windows) |

---

## Fitur Utama

### Smart BPJS Detector — *unggulan T.A.R.A*

Pasien input **satu identitas saja** (No. Kartu / NIK / No. RM). Sistem menjalankan 5 pengecekan **paralel** (4 goroutine + 1 serial), lalu mengembalikan **kategori kunjungan** dengan prioritas yang benar.

```
       Input identitas (No. Kartu / NIK / No. RM)
                       │
                       ▼
       [1] VClaim · GetPeserta · validasi status aktif (SERIAL)
                       │
                       ▼  (4 paralel · timeout 5 detik)
   ┌───────────┬──────────────┬─────────────┬───────────────┐
   │   MJKN    │   KONTROL    │ POST_RANAP  │  POST_RAJAL   │
   │  Antrol   │ Khanza surat │ Khanza      │ Khanza        │
   │           │ kontrol      │ kamar_inap  │ reg_periksa   │
   └───────────┴──────────────┴─────────────┴───────────────┘
                       │
              Priority resolution:
   MJKN > KONTROL > POST_RANAP > POST_RAJAL > RUJUKAN_BARU
```

### Daftar Fitur Lengkap

- 🎯 **Smart Detector** — 5 pemeriksaan paralel, prioritas otomatis, hasil <5 detik
- 🆔 **Frista Card Reader** — auto-baca KTP, auto-fill form (Windows: USB HID + auto-login headless)
- 👆 **Fingerprint BPJS** — integrasi `After.exe` headless via Windows UI Automation
- 🖨️ **Thermal Printer ESC/POS** — cetak tiket antrian dengan template Go `text/template`
- 📋 **3 Jalur Antrian** — Loket Admisi, Poli, Umum (counter management)
- 🏥 **Pendaftaran Otomatis** — Poli Umum & BPJS, integrasi langsung Khanza SIMRS
- 📲 **Aktivasi Satu Sehat Mobile** — flow OAuth2 untuk pasien
- 🔄 **Offline Queue & Reconcile** — background worker, tahan saat BPJS API down
- 🎨 **Kiosk UI Responsif** — `clamp()` di mana-mana, jalan mulus 15"–32" monitor
- 🔐 **Credential Encryption** — Windows DPAPI + AES-256-GCM untuk config sensitif
- 🪵 **PHI-Safe Logging** — NIK/No.Kartu/No.RM otomatis di-mask di log

---

## Arsitektur Singkat

```
apm-go/
├── cmd/apm/main.go              ← Entry point Wails
├── internal/
│   ├── config/                  ← Viper TOML loader + hot-reload
│   ├── domain/                  ← Pure structs, interfaces, enums (no external deps)
│   ├── service/
│   │   ├── detector/            ← ★ Smart BPJS Detector (5 goroutine paralel)
│   │   ├── antrian/             ← Counter management, 3 jalur
│   │   ├── sep/                 ← SEP builder + VClaim integration
│   │   ├── pendaftaran/         ← Registrasi poli umum & BPJS
│   │   └── satusehat/           ← Aktivasi Satu Sehat Mobile
│   ├── integration/
│   │   ├── vclaim/              ← BPJS VClaim v2 (HMAC-SHA256 + AES-256-CBC decrypt)
│   │   ├── antrol/              ← BPJS Antrean Online API
│   │   ├── mjkn/                ← Mobile JKN API
│   │   └── khanza/              ← Laravel SIMRS Khanza REST client
│   ├── hardware/
│   │   ├── frista/              ← Mac: mock | Windows: USB HID + auto-login
│   │   ├── fingerprint/         ← Mac: mock | Windows: headless After.exe
│   │   └── printer/             ← ESC/POS thermal printer
│   ├── store/                   ← sqlc generated SQLite access
│   └── reconcile/               ← Offline queue background worker
├── frontend/                    ← Vue 3 + Tailwind CSS kiosk UI
│   ├── src/screens/             ← HomeScreen, DetectScreen, ResultScreen, dll
│   ├── src/components/          ← BigButton, PatientCard, NumPad, dll
│   └── src/stores/              ← Pinia: patient, antrian, detection
├── migrations/                  ← SQLite schema SQL
├── templates/                   ← ESC/POS print templates
├── config.example.toml          ← Template konfigurasi (commit ini, BUKAN config.toml asli)
└── Makefile                     ← dev · build-mac · build-windows · test
```

### Dual-Platform: Satu Codebase, Dua Runtime

T.A.R.A dikembangkan di **Mac/Linux** (mock hardware) lalu di-deploy ke **Windows** (real hardware). Deteksi platform dilakukan sekali di `internal/hardware/provider.go`:

```go
func NewProvider(cfg config.Config) *Provider {
    switch runtime.GOOS {
    case "windows":
        return &Provider{
            Frista:      frista.NewWindowsReader(cfg.Frista),
            Fingerprint: fingerprint.NewWindowsHeadless(cfg.Fingerprint),
            Printer:     printer.NewESCPOS(cfg.Printer),
        }
    default: // darwin (Mac), linux — development
        return &Provider{
            Frista:      frista.NewMock(cfg.Dev.MockServerPort),
            Fingerprint: fingerprint.NewMock(),
            Printer:     printer.NewConsolePrinter(),
        }
    }
}
```

---

## Instalasi

### A. Setup Development di macOS / Linux

#### 1. Install Prerequisites

```bash
# Go 1.22+
brew install go

# Wails v2
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Node.js 18+ (frontend Vue)
brew install node

# Cross-compile toolchain (untuk build Windows dari Mac)
brew install mingw-w64

# Verifikasi
go version                          # go1.22+
wails version                       # v2.x
node --version                      # v18+
x86_64-w64-mingw32-gcc --version    # cross-compiler Windows
```

#### 2. Clone Repository

```bash
git clone https://github.com/rsamjkt/Arunika-TARA.git
cd Arunika-TARA
```

#### 3. Setup Konfigurasi

```bash
# Copy template config
cp config.example.toml config.toml

# Edit config.toml — minimal yang perlu diisi untuk dev:
#   [server] khanza_url = "http://IP-SERVER-KHANZA:PORT"
#   [bpjs]   cons_id, consumer_secret  ← dari BPJS dev env
#
# Hardware (Frista, Fingerprint, Printer) di-mock OTOMATIS di non-Windows.
```

#### 4. Install Dependencies

```bash
# Go modules
go mod download

# Frontend
cd frontend && npm install && cd ..
```

#### 5. Jalankan Dev Mode

```bash
# Hot reload Vue + Go
make dev
# atau
wails dev
```

#### 6. Simulasi Hardware (Mac/Linux)

```bash
# Simulasi tap kartu KTP via Frista
curl -X POST http://localhost:9090/mock/card-read \
  -H "Content-Type: application/json" \
  -d '{
    "nik": "3271234567890001",
    "nama": "Budi Santoso",
    "tgl_lahir": "1980-05-15",
    "alamat": "Jl. Merdeka No. 1, Jakarta",
    "no_kartu": "0001234567890012"
  }'

# Simulasi fingerprint gagal (sekali pakai)
curl -X POST http://localhost:9090/mock/fp-fail

# Output printer akan muncul di stdout terminal saat user selesai antrian/SEP
```

---

### B. Build & Deploy ke Windows Production

#### 1. Cross-Compile dari Mac

```bash
make build-windows
# Output: dist/apm-windows-amd64/apm.exe
```

#### 2. Prasyarat Mesin Windows

- Windows 10 Pro/Enterprise x64 (build 1903+) atau Windows 11
- WebView2 Runtime ([download](https://developer.microsoft.com/en-us/microsoft-edge/webview2/))
- Microsoft Visual C++ Redistributable 2022
- Frista.exe dari vendor (path dikonfigurasi di `config.toml`)
- After.exe — Aplikasi Sidik Jari BPJS Kesehatan v2.0+
  *Default: `C:\Program Files (x86)\Aplikasi Sidik Jari BPJS Kesehatan\After.exe`*
- Thermal printer ESC/POS (USB atau Serial)
- LAN ke server Khanza · Internet ke BPJS API

#### 3. Encrypt Credential (jalankan sekali)

```powershell
apm.exe --encrypt-config
# Akan prompt username/password Frista & FP BPJS
# Output: config.toml di-update dengan ENC:... (AES-256-GCM, key dari Windows DPAPI)
```

#### 4. Install sebagai Windows Service

```powershell
# Jalankan PowerShell sebagai Administrator
.\apm.exe --install-service --name "APMService" --display "Anjungan Pasien Mandiri"
Set-Service -Name "APMService" -StartupType Automatic
Start-Service -Name "APMService"

# Verifikasi
Get-Service -Name "APMService"
Get-EventLog -LogName Application -Source "APMService" -Newest 50
```

---

## Konfigurasi

Lihat `config.example.toml` untuk template lengkap. Bagian yang perlu diisi:

| Bagian | Wajib | Keterangan |
|---|---|---|
| `[server]` `khanza_url` | ✅ | URL REST API Khanza SIMRS di jaringan RS |
| `[bpjs]` `vclaim_url` | ✅ | `https://apijkn.bpjs-kesehatan.go.id/vclaim-rest/` (prod) atau `https://dvlp...` (dev) |
| `[bpjs]` `cons_id`, `consumer_secret` | ✅ | Credential dari kantor cabang BPJS |
| `[bpjs]` `antrol_*` | ✅ | Credential terpisah untuk Antrol |
| `[satusehat]` | optional | OAuth2 client untuk aktivasi SSM |
| `[frista]` `exe_path` | Win only | Path ke `frista.exe` |
| `[fingerprint]` `exe_path` | Win only | Path ke `After.exe` |
| `[printer]` | Win only | Port USB / nama printer ESC/POS |
| `[dev]` `mock_hardware` | Mac/Linux | `true` di dev (default), `false` di prod |

### Endpoint Referensi BPJS

| Service | Base URL | Auth |
|---------|----------|------|
| BPJS VClaim | `https://apijkn.bpjs-kesehatan.go.id/vclaim-rest/` | HMAC-SHA256 |
| BPJS Antrol | `https://apijkn.bpjs-kesehatan.go.id/antrean-rest/` | HMAC-SHA256 |
| Satu Sehat | `https://api-satusehat.kemkes.go.id/` | OAuth2 client credentials |
| Khanza API | `http://{RS_HOST}:{PORT}/api/apm/` | Bearer token |
| FP BPJS lokal | `https://fp.bpjs-kesehatan.go.id/finger-rest/` | Basic auth |

---

## Testing

```bash
# Unit tests — semua
make test

# Coverage report
make test-coverage          # target ≥ 80% untuk business logic

# Integration tests (butuh mock BPJS server)
make test-integration

# Lint
make lint                   # golangci-lint
```

**Mocks tersedia untuk:**
- `MockFristaReader` (HTTP endpoint untuk simulasi tap kartu)
- `MockFingerprintVerifier` (return success setelah 2 detik delay)
- `MockESCPOSPrinter` (output ke stdout dalam format readable)
- `MockVClaimClient`, `MockAntrolClient`, `MockKhanzaClient`

---

## Aturan Coding

T.A.R.A mengikuti aturan ketat — lihat `CLAUDE.md` untuk detail lengkap.

### Go
- Semua error **wajib** di-wrap: `fmt.Errorf("context: %w", err)`
- Semua HTTP call **wajib** pakai `context.Context` dengan timeout
- Goroutine **wajib** bisa di-cancel via context
- Hardware **wajib** dibungkus interface (untuk mockability)
- **Tidak boleh** ada hardcoded credential di source code
- Logging pakai `slog` (Go 1.21+ stdlib)
- **PHI** (NIK, No. Kartu, No. RM) **wajib** di-mask di log → `***`

### Vue / Frontend
- Semua font & padding pakai `clamp()` untuk responsivitas
- Touch target minimum: `min-height: clamp(52px, 7vw, 72px)`
- Warna **hanya** dari design token (lihat `DESIGN_SYSTEM.md`)
- State via Pinia stores — **tidak ada** logic bisnis di component
- Wails IPC call **wajib** `async/await` dengan error handling

### Testing
- Business logic: **min 80% coverage**
- Semua BPJS API call **wajib** bisa di-mock via interface

---

## Dokumentasi Tambahan

| File | Isi |
|---|---|
| [`CLAUDE.md`](./CLAUDE.md) | Master memory untuk Claude Code (baca pertama!) |
| [`BPJS_INTEGRATION.md`](./BPJS_INTEGRATION%20.md) | Detail teknis VClaim, Antrol, MJKN, Fingerprint |
| [`DESIGN_SYSTEM.md`](./DESIGN_SYSTEM.md) | Token warna, spacing, komponen Vue + responsivitas |
| [`HARDWARE_PLATFORM.md`](./HARDWARE_PLATFORM.md) | Platform detection, mock vs real, Windows-specific |
| [`PROMPTS.md`](./PROMPTS.md) | Prompt library lengkap untuk setiap task coding |
| [`QUICKSTART.md`](./QUICKSTART.md) | Quickstart development di Mac |

---

## Roadmap

- [x] Smart BPJS Detector engine (5 goroutine paralel)
- [x] Frista card reader (mock + Windows USB HID)
- [x] Fingerprint BPJS headless integration
- [x] ESC/POS thermal printer
- [x] Antrian 3 jalur (Loket / Poli / Umum)
- [ ] Aktivasi Satu Sehat Mobile (in progress)
- [ ] Offline reconcile worker (in progress)
- [ ] Admin dashboard via PIN
- [ ] Multi-cabang RS support

---

## Lisensi & Kontak

**© 2026 PT. Arunika Komputasi Awan Integrasi**

Software ini adalah produk proprietary milik PT. Arunika Komputasi Awan Integrasi.
Untuk pertanyaan, kerja sama, atau dukungan teknis, hubungi tim IT melalui repo issue tracker.

---

> **T.A.R.A** — *Total Automated Registration Assistant*
> Built with ☕ for Indonesian healthcare.
