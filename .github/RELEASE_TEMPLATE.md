# Arunihealth T.A.R.A — _Total Automated Registration Assistant_

> Self-service kiosk for hospital outpatient registration, BPJS SEP issuance, and queue management. Direct-DB integration with SIMRS Khanza, Smart BPJS Detector with auto-classification, modern accessibility-first UI built for **multi-generation Indonesian patients** (elderly + middle-aged + young).

## What's New in v1.5 ("Mahatma 1.5")

### 🖨️ Print Template Polish — Struk Profesional ESC/POS

Templates `tiket_antrian.tmpl`, `registrasi.tmpl`, `sep.tmpl` di-rewrite total dengan format struk RS Indonesia standar.

**Marker syntax baru** di template ESC/POS:
- `[C]...[/C]` — center align
- `[B]...[/B]` — bold
- `[XL]...[/XL]` — quad-size (untuk nomor antrian besar di tengah struk)
- `[BIG]...[/BIG]` — double-size (heading)

`encodeESCPOS()` parse marker → substitute ke byte sequence ESC/POS langsung. Plus safety reset di akhir (bold off, size normal, align left) kalau template lupa close marker.

**Layout 3 struk baru**:

**Antrian Loket**: header RS bold center → "ANTRIAN LOKET" → nomor quad-size → tanggal+jam → footer "Silakan menunggu panggilan di Loket"

**Pendaftaran Pasien Umum**: header RS → "BUKTI PENDAFTARAN (UMUM)" → bold sections (No.Rawat / NoRM / Nama / Penjamin) → tabel kunjungan (Tujuan/Dokter/Tgl/Jam) → nomor antrian quad-size → biaya registrasi conditional → footer cetakan+petugas

**SEP BPJS**: header RS → "SURAT ELIGIBILITAS PESERTA" → bold sections (No.SEP / NoKartu / NIK / Nama / TglLahir+JK / Kelas) → "DETAIL KUNJUNGAN" tabel (Poli/Dokter/TglSEP/Asal Rujukan/No Rujukan/Faskes Perujuk/Diagnosa/SKDP) → nomor antrian quad-size → "PENTING: Bawa kartu BPJS dan KTP" → footer audit

**Field opsional rendered conditional** dengan Go template `{{- if .Field }}` — caller tidak pasok = baris tidak muncul = struk lebih clean.

### 🛡️ Backward Compat
- `encodeESCPOS(docType, body)` signature tetap — caller existing tetap kerja
- Reset alignment+size di akhir untuk safety
- Print history tetap simpan bytes hasil encoding (reprint via Admin Panel tetap akurat)

---

## What's New in v1.4 ("Mahatma 1.4")

**Critical SEP flow alignment dengan Java vendor reference** — fix bug yang mencegah flow Pasien BPJS bekerja end-to-end.

### 🚨 CRITICAL FIX — BuatPendaftaran sebelum SimpanSEP
Sebelumnya flow BPJS skip `BuatPendaftaran` → `SimpanSEP` gagal resolve `no_rawat` → SEP issued di BPJS server tapi tidak tercatat di Khanza lokal → antrian/rekam medis broken.

Sekarang (mirror `DlgRegistrasiSEPPertama.java:2682-2685`):
- `BuatSEPRujukan` → preflight → biometrik → ValidasiRujukan → **BuatPendaftaran** → CreateSEP (VClaim) → SimpanSEP → SimpanRujukMasuk + SimpanRujukanBPJS
- `BuatSEPKontrol` → preflight → biometrik → BuatPendaftaran → CreateSEPKontrol → SimpanSEP
- `BuatSEPPostRANAP` / `BuatSEPPostRAJAL` → preflight → biometrik → BuatPendaftaran → CreateSEP → SimpanSEP

### 🛡️ Pre-flight Checks (mirror Java line 1734-1794)
Method baru di `KhanzaClient`:
- `CheckDuplicateRegistration(noRM, kdPoli, kdDokter, tglRegistrasi, kdPj)` → reject kalau pasien sudah daftar di poli+dokter+kdPj sama hari ini → `domain.ErrSudahTerdaftarHariIni`
- `CheckDoctorOnLeave(kdDokter, tglRegistrasi)` → reject kalau dokter cuti → `domain.ErrDokterCuti`

Plus `vclaim.CekSEPDuplikasi(noKartu, tglSEP)` dipanggil di preflight (anti-fraud server-side).

`runPreflight()` helper terpadu di `service/sep/service.go` — dipanggil semua flow BPJS.

### 📦 SEPRequest Extended (parity Java payload)
Tambahan field opsional: detail rujukan (TglRujukan/KdPPK/NmPPK/AsalRujukan/Diagnosa), SKDP context (NoSKDP/KdDPJP), conditional (Eksekutif/COB/Katarak/LakaLantas + lokasi 4-tier), routing (TujuanKunjungan/FlagProcedure/KdPenunjang/AsesmenPelayanan), User audit.

UI panel conditional (Laka Lantas form, COB confirmation) defer ke v1.5.

### 🩺 Auto Audit Trail Post-SEP
Flow Rujukan FKTP: setelah SimpanSEP sukses, otomatis call `SimpanRujukMasuk` (`rujuk_masuk`) + `SimpanRujukanBPJS` (`bridging_rujukan_bpjs` compliance audit BPJS).

### 🎯 Error Sentinel Baru
- `domain.ErrSudahTerdaftarHariIni`

---

## What's New in v1.3 ("Mahatma 1.3")

Major BPJS parity push + UX completion.

### 🆘 "Bantu Saya" Wizard untuk First-Time Users (BB1)
4-screen guided wizard untuk pasien yang pertama kali pakai kiosk:
1. **Welcome** — reassurance "Tidak masalah, saya tanya 2 hal saja"
2. **Q: Punya kartu BPJS?** — 2 tombol besar (Ya, punya / Tidak punya) dengan ikon Phosphor + warna feedback
3. **Q: Pertama kali di RS ini?** — 2 tombol (skip kalau punya BPJS sudah cukup)
4. **Hasil rekomendasi** — auto-route ke flow yang sesuai dengan tombol primary "Lanjut ke Pendaftaran X" + ulangi pertanyaan

Design lansia-friendly:
- Font ≥18px body, ≥24px heading
- Tombol 80px+ touch target
- Audio cue tap di setiap pilihan
- Tombol "Kembali" + "Panggil petugas" selalu visible
- 1 instruksi per screen, tone calm/mengayomi

### 🩺 BPJS Wave 1 — Critical Compliance + Anti-Fraud
**1. SEP Duplicate Detection via VClaim** — `CekSEPDuplikasi(noKartu, tglSEP)` yang panggil `GET /SEP/{noKartu}/{tglSEP}` untuk verify server-side sebelum CreateSEP. Mencegah ghost SEP saat kiosk crash post-VClaim insert tapi pre-DB insert (billing safety).

**2. `bridging_rujukan_bpjs` Audit Trail** — `SimpanRujukanBPJS()` insert ke 14-kolom audit table untuk compliance + klaim BPJS verifikasi (rujukan FKTP terlacak terpisah dari `rujuk_masuk`).

### 🩺 BPJS Wave 2 — RencanaKontrol API + Conditional Fields
**1. RencanaKontrol Endpoint** — `BuatRencanaKontrol(req)` yang panggil `POST /RencanaKontrol/insert` VClaim API untuk schedule SKDP baru pasien post-discharge. Hasil `noSuratKontrol` auto-disimpan ke `bridging_surat_kontrol_bpjs` lewat method baru `SimpanSuratKontrolBPJS()` — supaya Smart Detector kunjungan berikutnya bisa pickup kontrolnya.

**2. Laka Lantas + COB + Eksekutif Conditional Fields** — `domain.SEP` extended dengan:
- `LakaLantas` (0/1/2/3) + `TglKejadian` + `KetKecelakaan` + lokasi 4-tier (propinsi/kabupaten/kecamatan)
- `COB` flag untuk dual-insurance pasien
- `Eksekutif` untuk kelas naik
- `TujuanKunjungan` dan `AsesmenPelayanan` untuk routing tariff

`SimpanSEP` extended dari 18→30+ kolom — populate semua field critical dengan default safe (`0. Tidak` untuk enum). UI panel conditional untuk Laka Lantas defer ke v1.4.

### 📦 Schema additions
- `domain.RencanaKontrolRequest` + `domain.RencanaKontrol` — input/output VClaim
- `domain.RujukanBPJS` — audit trail rujukan FKTP
- `KhanzaClient` interface: `+SimpanRujukanBPJS`, `+SimpanSuratKontrolBPJS`
- `VClaimClient` interface: `+CekSEPDuplikasi`, `+BuatRencanaKontrol`

## Coming in v1.4
- COB UI panel (dual-insurance handling)
- Laka Lantas conditional UI form
- Antrol push notification (booking BPJS muncul di Mobile JKN app)
- Katarak/Penunjang/Asesmen detail validation
- Multi-kiosk antrian counter (prefix per kiosk via config)
- DPJP auto-pick from `dpjp_ranap` untuk Post-RANAP
- Hamil → OBGYN auto-routing dari diagnosa awal

---

## What's New in v1.2 ("Mahatma 1.2")

Continuation of UX brainstorm implementation — addressing the remaining elderly-friendly gaps and one critical BPJS integration parity:

### 🆕 Dual-Channel InputScreen (BB2)
The old layout buried the Frista card reader as a thin status banner above the NumPad. New layout — **two equal-weight panels side-by-side**:
- **Left panel: TAP KARTU** — large card icon with pulse animation, 80-120px tap zone, instruction "Tempel kartu Anda di reader bawah", bullet list of supported cards (BPJS / KTP elektronik), live status badge (Reader aktif / tidak aktif)
- **Right panel: KETIK** — InputDisplay + progress text + 3x4 NumPad
- Lansia yang punya kartu fisik → tap & done (~5 detik) instead of typing 16 digits with typo risk
- Footer with `<BackButton>` + "Panggil petugas" safety net

### 🆕 DetectScreen "Hubungi Petugas" Safety Net
- Long-running guard threshold lowered from 7s → 5s
- When Smart Detector takes longer than expected, friendly amber notice appears: "Sistem agak lama hari ini. Mohon ditunggu sebentar..."
- Plus tappable "📞 Panggil Petugas" button — does NOT abort detection (continues in background), just signals staff to assist
- Calm copywriting (no urgent/scary tone)

### 🆕 RegistrasiUmum Visual Stepper
- Replaced tiny "1/3 — Pilih Poli" text in header corner with full-width segmented `<StepperBar>`
- Each step has Phosphor icon (PhBuildings → PhStethoscope → PhCheckSquare)
- States: ✓ done (emerald), ◉ active (theme primary, animated), ○ future (gray)
- Tap-back enabled: lansia bisa tap step yang sudah selesai untuk balik (lebih intuitive dari Back button)

### 🩺 BPJS Mobile JKN Booking Status Update (Audit Phase 0b Gap #6)
- After `SimpanSEP` success, automatically `UPDATE booking_registrasi SET status='Terdaftar', waktu_kunjungan=NOW()` for the matching patient
- Tanpa fix ini, antrian Mobile JKN BPJS tetap show "Belum" di app pasien meskipun sudah didaftarkan di kiosk → bingung pasien
- Mirror pattern Java vendor `DlgRegistrasiSEPPertama.java:2760-2767`

## Coming in v1.3
- "Bantu Saya" wizard (BB1) — 4-screen guided flow + audio TTS
- BPJS gap closure: COB handling, Laka Lantas conditional, RencanaKontrol API endpoint
- Per-pathway PatientCard "Apakah ini Anda?" verification
- Multi-kiosk antrian counter via different prefix

---

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
