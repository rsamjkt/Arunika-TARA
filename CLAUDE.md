# APM-Go — Claude Code Master Memory

> Baca file ini PERTAMA sebelum melakukan apapun. Ini adalah memory utama proyek.

## Identitas Proyek

**Nama:** Anjungan Pasien Mandiri (APM) — Reimplementasi Go  
**Pemilik:** PT. Arunika Komputasi Awan Integrasi  
**Stack Utama:** Go 1.22+ · Wails v2 · Vue 3 · Tailwind CSS · SQLite  
**Platform Deploy:** Windows 10/11 x64 (kiosk & loket)  
**Development Machine:** macOS (Apple Silicon / Intel) — cross-compile ke Windows

## Arsitektur Singkat

```
apm-go/
├── cmd/apm/main.go              ← Entry point Wails
├── internal/
│   ├── config/                  ← Viper TOML loader + hot-reload
│   ├── domain/                  ← Pure structs, interfaces, enums (NO external deps)
│   ├── service/
│   │   ├── detector/            ← ★ SMART BPJS DETECTOR (5 goroutine paralel)
│   │   ├── antrian/             ← Counter management, 3 jalur (Loket/Poli/Umum)
│   │   ├── sep/                 ← SEP builder + VClaim integration
│   │   ├── pendaftaran/         ← Registrasi poli umum & BPJS
│   │   └── satusehat/           ← Aktivasi Satu Sehat Mobile
│   ├── integration/
│   │   ├── vclaim/              ← BPJS VClaim v2 client (HMAC-SHA256 + AES decrypt)
│   │   ├── antrol/              ← BPJS Antrean Online API
│   │   ├── mjkn/                ← Mobile JKN API
│   │   └── khanza/              ← Laravel SIMRS Khanza REST client
│   ├── hardware/
│   │   ├── frista/              ← Card reader (Mac: mock | Windows: USB HID + auto-login)
│   │   ├── fingerprint/         ← FP BPJS (Mac: mock | Windows: headless After.exe)
│   │   └── printer/             ← ESC/POS thermal printer
│   ├── store/                   ← sqlc generated SQLite access
│   └── reconcile/               ← Offline queue background worker
├── frontend/                    ← Vue 3 + Tailwind CSS kiosk UI
│   ├── src/screens/             ← HomeScreen, DetectScreen, ResultScreen, dll
│   ├── src/components/          ← BigButton, PatientCard, NumPad, dll
│   └── src/stores/              ← Pinia: patient, antrian, detection
├── migrations/                  ← SQLite schema SQL files
├── templates/                   ← ESC/POS print templates (Go text/template)
├── config.toml                  ← Konfigurasi deployment (JANGAN commit credential)
├── Makefile                     ← build, test, cross-compile
└── CLAUDE.md                    ← ← FILE INI

## Aturan Coding — WAJIB DIIKUTI

### Go
- Semua error HARUS di-wrap: `fmt.Errorf("context: %w", err)`
- Semua HTTP call HARUS pakai context dengan timeout
- Goroutine HARUS bisa di-cancel via context
- Interface untuk semua hardware: `HardwareProvider` pattern
- Tidak boleh ada hardcoded credential di source code
- Log HARUS menggunakan `slog` (stdlib Go 1.21+)
- PHI (NIK, No Kartu, No RM) WAJIB di-mask di log: ganti dengan `***`

### Vue / Frontend
- Semua ukuran font dan padding HARUS menggunakan `clamp()` untuk responsivitas
- Touch target minimum: `min-height: clamp(52px, 7vw, 72px)`
- Warna: HANYA dari design token (lihat `DESIGN_SYSTEM.md`)
- State management via Pinia stores — tidak boleh ada logic bisnis di component
- Wails IPC call HARUS async/await dengan error handling

### Testing
- Business logic: min 80% coverage
- Semua BPJS API call HARUS bisa di-mock via interface
- Hardware HARUS bisa di-mock (ada `MockFrista`, `MockFingerprint`, dll)

## Cara Jalankan di Mac (Development)

```bash
# Install dependencies
brew install go wails
cd apm-go

# Jalankan dengan hardware mock (default di non-Windows)
make dev
# atau: wails dev

# Build untuk Windows (cross-compile dari Mac)
make build-windows
# Output: build/bin/apm-windows-amd64.exe
```

## Platform Detection — KRITIS

Hardware Frista dan Fingerprint BPJS hanya tersedia di Windows.
Di Mac/Linux, sistem OTOMATIS menggunakan implementasi mock.

```go
// Contoh pattern — selalu gunakan ini
// internal/hardware/provider.go
func NewFristaReader(cfg Config) FristaReader {
    if runtime.GOOS == "windows" {
        return newWindowsFristaReader(cfg)  // USB HID + auto-login
    }
    return newMockFristaReader()            // Simulasi untuk dev Mac
}
```

File terkait detail: lihat `HARDWARE_PLATFORM.md`

## File Memory Tambahan

| File | Isi |
|------|-----|
| `BPJS_INTEGRATION.md` | Semua endpoint VClaim, Antrol, MJKN + auth flow |
| `DESIGN_SYSTEM.md` | Token warna, spacing, komponen Vue + responsivitas |
| `HARDWARE_PLATFORM.md` | Platform detection, mock vs real, Windows-specific |
| `PROMPTS.md` | Prompt library lengkap untuk setiap task coding |

## Referensi Cepat — Endpoint Penting

| Service | Base URL | Auth |
|---------|----------|------|
| BPJS VClaim | `https://apijkn.bpjs-kesehatan.go.id/vclaim-rest/` | HMAC-SHA256 |
| BPJS Antrol | `https://apijkn.bpjs-kesehatan.go.id/antrean-rest/` | HMAC-SHA256 |
| Satu Sehat | `https://api-satusehat.kemkes.go.id/` | OAuth2 client credentials |
| Khanza API | `http://{RS_HOST}:{PORT}/api/apm/` | Bearer token |
| FP BPJS lokal | `https://fp.bpjs-kesehatan.go.id/finger-rest/` | Basic auth |

