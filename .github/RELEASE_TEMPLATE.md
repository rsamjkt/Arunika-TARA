# Arunihealth T.A.R.A — _Total Automated Registration Assistant_

> Self-service kiosk for hospital outpatient registration, BPJS SEP issuance, and queue management. Direct-DB integration with SIMRS Khanza, Smart BPJS Detector with auto-classification, modern accessibility-first UI built for **multi-generation Indonesian patients** (elderly + middle-aged + young).

## What's New in v1.1 ("Mahatma 1.1")

### 🎨 UI/UX Refresh — Built for Elderly & Tech-Savvy Alike
Based on a deep UX audit covering WCAG 2.1/2.2 + W3C WAI-AGE guidelines + multi-generation usability research:

**Foundation (Phase 1):**
- 🎨 **Phosphor Icons** — modern, friendly, weight-aware icon set replacing inline SVGs across the app
- 🌈 **Configurable theme color via `config.toml`** — biru korporat default, easily switch to teal RS Anggrek Mas (#00897B), green medical, or your hospital's identity color. Auto-derives `--color-primary-dark` and `--color-primary-light` for hover/accent states
- 🏥 **Configurable hospital logo + name + tagline** via `[branding]` section — drop your logo file (PNG/SVG/JPG/WebP), set hospital name, done
- 🔊 **Audio cue system** (Web Audio API synth — no asset files): tap sound, success chime, error tone, notification. Toggle on/off + volume in `[audio]` config
- 🧩 **Reusable components**: `<BackButton>` floating bottom-left with optional `<ConfirmBackModal>`, `<StepperBar>` segmented progress with Phosphor icons + ARIA `aria-current="step"`

**Quick Wins (Phase 2):**
- ⚡ **QW1**: AntrianScreen removed → "Ambil Nomor Loket" is now single-tap on Home (no sub-menu). Hospital doesn't need farmasi/CS separate queue.
- ⚡ **QW2**: NumPad redesign — "HAPUS" text label (no ambiguous backspace icon), "CARI" with magnifying glass + larger label. Touch targets 60→80px (was 52→72px), spacing 10→14px (was 6px) per WCAG 2.5.5 + senior-citizen UX research.
- ⚡ **QW3**: Ticket countdown 10s → 25s, idle timeout 60s → 90s, **tap anywhere to reset countdown**, visual SVG progress ring using theme color (was tiny text only). Elderly users now have time to read the full ticket.
- ⚡ **QW5**: InputDisplay progress text — 4 dynamic states (empty / typing / can submit / full). Removes "is this enough?" ambiguity. Color shifts to emerald when ready.

**Big Bet (Phase 3 — partial):**
- 🏠 **HomeScreen overhaul**: welcome banner with time-aware greeting (Selamat pagi/siang/sore/malam) + illustrated ambient background, 1 dominant "Pasien BPJS" hero (60% visual weight — mass user RS Pemerintah), 2 secondary cards equal weight (Pasien Umum + Antrian Loket), Aktivasi Satu Sehat moved to footer (niche action)
- 🆘 **"Pertama kali? Bantu saya"** + **"Panggil petugas"** safety net buttons in footer — visible on every visit

### 🎫 Antrian Loket — Aligned to Khanza V3 Vendor Pattern
- INSERT to Khanza `antrian_loket` table directly (was SQLite-only fallback)
- Counter strategy: `SELECT IFNULL(MAX(CAST(noantrian AS UNSIGNED)),0) + 1` per type + CURDATE() — mirrors `KhanzaHMSAnjunganSEP_RSAMXIP/DlgAmbilAntrean.java:240+`
- Display format: prefix + 3-digit (A001 for Loket, B015 for CS) per vendor convention
- App-level mutex serializes SELECT MAX → INSERT for single-kiosk safety
- Falls back gracefully to SQLite local on Khanza outage

### 🩺 BPJS Refinements
- `user_key` BPJS header now sent in every VClaim request (required for AES-256-CBC response decryption — production endpoint compliance)
- `BPJSConfig.UserKey` added to config

### 📦 Configuration Additions

```toml
[branding]
logo_path        = ""              # path to PNG/SVG/JPG logo
hospital_name    = "Rumah Sakit Anggrek Mas"
hospital_tagline = "Anjungan Pasien Mandiri"
primary_color    = "#1B4FD8"       # biru korporat — or "#00897B" teal RS
primary_color_dark = ""            # auto-derive 12% darker if empty
accent_color     = ""              # auto-derive lighter

[audio]
enabled = true
volume  = 0.6                      # 0.0–1.0
```

## Carryover from v1.0 ("Mahatma")

### 🏥 Pasien Umum (Walk-in Registration) — Complete
Full 19-column `reg_periksa` INSERT, smart age unit (Th/Bl/Hr), full address concatenation from 5 master tables, tariff from `poliklinik.registrasilama`, per-poliklinik visit history.

### 🧠 Smart BPJS Detector — Auto-Classify (No Manual Operator Selection)
Parallel probes with priority resolution:

| Priority | Category | Source |
|----------|----------|--------|
| 1 | Mobile JKN booking | Antrol API → Khanza `booking_registrasi` fallback |
| 2 | Surat Kontrol (SKDP) | `bridging_surat_kontrol_bpjs` JOIN `bridging_sep`, ≤30 day window |
| 3 | Post-RANAP | `kamar_inap` + `dpjp_ranap`, ≤7 day window |
| 4 | Post-RAJAL | `rujukan_internal_poli` (preferred) / SKDP fallback |
| 5 | Rujukan Baru FKTP | VClaim API |
| Default | TidakAktif / RujukanBaru | Safe fallback |

### 🔌 Direct-DB Khanza Integration
Native Go MySQL — no REST middleware. Configurable kd_pj mapping (default A03 Umum / BPJ BPJS), `maping_poli_bpjs` & `maping_dokter_dpjpvclaim` translation, transactional `BuatPendaftaran`, full bridging integration (bridging_sep 18 critical columns, rujuk_masuk, bpjs_prb).

### 🖨️ Kiosk Hardware (Windows Production)
- **Frista card reader** auto-launch + Win32 UI Automation login + clipboard polling for card data
- **After.exe (BPJS Sidik Jari)** auto-launch + auto-login + REST polling
- All Win32 calls pure Go syscall — no CGO, cross-compile from macOS via mingw-w64

### 🔒 Security
AES-256-GCM credential encryption, master key from OS keychain/DPAPI, PHI log masking via `slog.Handler` wrapper.

## Download

| Platform | File |
|----------|------|
| macOS Universal (Intel + Apple Silicon) | `apm-mac-universal.zip` |
| Windows x64 (10/11) | `apm-windows-amd64.zip` |

## Quick Install

### macOS

```bash
unzip apm-mac-universal.zip
cd apm-mac-universal
cp config.example.toml config.toml
# edit config.toml — fill khanza_dsn, bpjs creds, [branding] color/logo
./run-mac.sh
```

First launch: right-click `apm-go.app` → Open (bypass Gatekeeper).

### Windows

```cmd
REM extract apm-windows-amd64.zip ke C:\APM
cd C:\APM
copy config.example.toml config.toml
notepad config.toml
REM ...edit khanza_dsn, bpjs creds, [branding]...
run-windows.bat
```

For unattended kiosk: install as Windows Service — see [README](https://github.com/rsamjkt/Arunika-TARA/blob/main/README.md#operasi-sehari-hari).

## Documentation

Full guide in [README.md](https://github.com/rsamjkt/Arunika-TARA/blob/main/README.md):
- Configuration reference (every section)
- Credential setup (BPJS API onboarding, Frista + After.exe, encryption)
- Hardware setup (USB/serial/network printer; Spy++ for window class discovery)
- Operations runbook
- Troubleshooting

## Coming in v1.2

- "Bantu Saya" wizard for first-time elderly users (BB1)
- InputScreen dual-channel layout (Tap Kartu equal weight to NumPad) (BB2)
- DetectScreen "Hubungi Petugas" appears after 5s
- Per-pathway PatientCard verification ("Apakah ini Anda?")
- BPJS gap closure: COB handling, Laka Lantas conditional, RencanaKontrol API endpoint
- Mobile JKN booking status update post-SEP

---

Built with ❤️ for healthcare in Indonesia — based on patterns from `KhanzaHMSAnjunganSEP_RSAMXIP` (RS Indriati / Anggrek Mas), reimagined in Go for native cross-platform deployment and a smarter, friendlier patient experience.
