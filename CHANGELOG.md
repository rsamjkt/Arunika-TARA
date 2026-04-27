# Changelog — Arunihealth T.A.R.A

Codename: **Mahatma**. Tag pattern: `vX.Y.Z-mahatma`. Auto-build via GitHub Actions saat tag push.

Format mengikuti [Keep a Changelog](https://keepachangelog.com/) sederhana — semua release di bawah sudah di-tag dan terbangun otomatis di [GitHub Releases](https://github.com/rsamjkt/Arunika-TARA/releases).

---

## [v1.5.0-mahatma] — Print Templates Polish

### Added
- **Marker-based ESC/POS encoder** — template designer pakai `[C]/[/C]` (center), `[B]/[/B]` (bold), `[XL]/[/XL]` (quad-size), `[BIG]/[/BIG]` (double-size) di template, encoder substitute ke ESC/POS bytes saat print.
- **Tiket Antrian** template baru — header RS bold center → "ANTRIAN LOKET" → nomor quad-size → tanggal/jam → footer "Silakan menunggu di Loket".
- **Pendaftaran Pasien Umum** template baru — bold sections (NoRawat/NoRM/Nama/Penjamin), tabel kunjungan, nomor antrian besar, biaya conditional, footer cetak+petugas.
- **SEP BPJS** template baru — full detail kunjungan tabel (poli/dokter/asal rujukan/no rujukan/faskes perujuk/diagnosa/SKDP), nomor antrian besar, footer "Bawa kartu BPJS dan KTP".

### Changed
- `encodeESCPOS()` safety reset di akhir (bold off, size normal, align left) kalau template lupa close marker.
- Field opsional rendered conditional dengan Go template `{{- if .Field }}` — caller tidak pasok = baris hilang = struk lebih clean.

### Tests
- Removed: `TestEncodeESCPOS_ContainsHeaderDocType`, `TestEncodeESCPOS_AlignCenterUntukHeader_AlignLeftUntukBody` (behavior change).
- Added: `TestEncodeESCPOS_MarkerCenter/Bold/XL_DiSubstitusi`, `TestEncodeESCPOS_FooterAdaCutPaper`.

---

## [v1.4.2-mahatma] — Scope Clarification

### Added
- README section **"Scope — Apa yang Bisa & Tidak Lewat Kiosk"** — table 8 supported flows (BPJS Rujukan FKTP/Kontrol/MJKN/PostRANAP/PostRAJAL/TidakAktif, Pasien Umum, Antrian Loket) vs 8 out-of-scope (Laka Lantas, pasien baru, COB rumit, kelas naik, Katarak, emergency, rujukan expired, pasien <17 BPJS).
- Filosofi design: kiosk pelengkap loket, bukan pengganti. Footer "Panggil Petugas" + "Bantu Saya" wizard sebagai safety net.

---

## [v1.4.1-mahatma] — Hidden Admin Trigger

### Added
- **Tap logo 5x cepat** (≤2 detik antar tap) di HomeScreen header → buka `/admin` (PIN gate tetap aktif).
- **Keyboard shortcut Ctrl+Alt+A** (Cmd+Alt+A di Mac) untuk staff dengan keyboard.

### Why
Sebelumnya admin panel hanya bisa diakses via URL `/admin` langsung — tidak praktis di kiosk fullscreen.

---

## [v1.4.0-mahatma] — Critical SEP Flow Fix

### Fixed
- **CRITICAL**: Service layer flow BPJS sekarang panggil `khanza.BuatPendaftaran` (INSERT `reg_periksa` BPJS dengan kd_pj=BPJ) **sebelum** `vclaim.CreateSEP` + `khanza.SimpanSEP`. Sebelumnya skip step ini → SimpanSEP gagal resolve no_rawat → SEP issued di BPJS server tapi tidak tercatat di Khanza lokal → antrian/rekam medis broken. Berlaku untuk semua flow: Rujukan/Kontrol/PostRANAP/PostRAJAL.
- Mirror urutan vendor Java `DlgRegistrasiSEPPertama.java:2682-2685`.

### Added
- `khanza.CheckDuplicateRegistration(noRM, kdPoli, kdDokter, tglReg, kdPj)` — query reg_periksa duplikasi → `ErrSudahTerdaftarHariIni`.
- `khanza.CheckDoctorOnLeave(kdDokter, tglReg)` — query jadwal_cuti_libur (graceful skip kalau tabel tidak ada) → `ErrDokterCuti`.
- `runPreflight()` helper terpadu — di-call semua flow BPJS sebelum issue SEP. Cek 3 hal: duplikasi reg_periksa, dokter cuti, SEP duplikasi VClaim remote.
- Auto post-SEP audit trail untuk flow Rujukan FKTP — `SimpanRujukMasuk` (rujuk_masuk) + `SimpanRujukanBPJS` (bridging_rujukan_bpjs) auto-call setelah SimpanSEP sukses.
- `domain.SEPRequest` extended dengan ~30 field opsional (TglRujukan/KdPPK/AsalRujukan/Diagnosa/NoSKDP/KdDPJP/Eksekutif/COB/Katarak/LakaLantas/lokasi 4-tier/TujuanKunjungan/FlagProcedure/KdPenunjang/AsesmenPelayanan/User) untuk parity Java vendor payload.
- `domain.ErrSudahTerdaftarHariIni`.

---

## [v1.3.0-mahatma] — BPJS Parity Push + Bantu Saya Wizard

### Added
- **Bantu Saya wizard** (`BantuSayaScreen.vue`) — 4-screen guided flow untuk first-time elderly users (Welcome → Q1 punya BPJS → Q2 first-time → Hasil rekomendasi → auto-route ke flow yang sesuai).
- `vclaim.CekSEPDuplikasi(noKartu, tglSEP)` — `GET /SEP/{noKartu}/{tglSEP}` untuk anti-fraud server-side check sebelum CreateSEP.
- `vclaim.BuatRencanaKontrol(req)` — `POST /RencanaKontrol/insert` untuk schedule SKDP baru pasien post-discharge.
- `khanza.SimpanRujukanBPJS(r)` — INSERT `bridging_rujukan_bpjs` (audit trail rujukan FKTP, 14 kolom).
- `khanza.SimpanSuratKontrolBPJS(sk)` — INSERT `bridging_surat_kontrol_bpjs` setelah BuatRencanaKontrol sukses (Smart Detector pickup berikutnya).
- Domain types: `RencanaKontrolRequest`, `RencanaKontrol`, `RujukanBPJS`.

### Changed
- `SimpanSEP` MySQLClient extend dari 18→30+ kolom — populate critical fields: rujukan FKTP (TglRujukan/KdPPK/NmPPK), DPJP (KdDPJP/NmDPJP), kelas rawat, asal rujukan, lokasi 4-tier, COB/Eksekutif/LakaLantas/TujuanKunjungan/AsesmenPelayanan dengan default safe.

---

## [v1.2.2-mahatma] — Default Printer USB POS-58

### Changed
- Default `[printer] mode = "escpos_usb"` + `port = "POS-58"` (yang paling umum di kiosk RS Indonesia). Sebelumnya default "console" yang misleading untuk production.
- `provider.go` Windows respect `mode=console` (admin override untuk test layout tanpa printer fisik). Mac/Linux tetap selalu pakai ConsolePrinter regardless mode.

---

## [v1.2.1-mahatma] — Logo BPJS + Printer Setup Guide

### Added
- `BpjsLogo.vue` component — SVG inline default (perisai + plus medis + text "BPJS Kesehatan / Jaminan Kesehatan Nasional"), variants `full/icon/text`, sizes `sm/md/lg`, `inverse` mode.
- HomeScreen hero "Pasien BPJS" pakai BpjsLogo dalam card putih bg.
- InputScreen header tampilkan BpjsLogo full saat mode=BPJS.
- Override BPJS logo via `[branding] bpjs_logo_path` di config (kalau RS punya file resmi).
- README section **"Thermal Printer — Setup Detail"** — 4 mode comparison, step-by-step Windows USB/serial/network setup, troubleshooting matrix, print pattern table.

---

## [v1.2.0-mahatma] — InputScreen Dual-Channel + Stepper + MJKN Status

### Added
- **InputScreen dual-channel** layout — panel kiri "Tap Kartu" setara visual weight dengan panel kanan NumPad. Pulse animation, live reader status badge.
- **DetectScreen safety net** — long-running guard 7s→5s, plus tappable "Panggil Petugas" button setelah trigger (tidak abort detection).
- **RegistrasiUmumScreen StepperBar** — segmented full-width dengan PhBuildings/PhStethoscope/PhCheckSquare icons, tap-back pada step yang sudah selesai.
- **MJKN booking status auto-update** post-SEP — `UPDATE booking_registrasi SET status='Terdaftar', waktu_kunjungan=NOW()`. Mirror Java `DlgRegistrasiSEPPertama.java:2760-2767`.

### Changed
- `idleTimeoutSec` 60→90 detik (lansia butuh waktu baca instruksi).
- `ticketAutoBackSec` 10→25 detik + tap-anywhere reset countdown + visual SVG progress ring.

---

## [v1.1.0-mahatma] — UX Foundation + HomeScreen Overhaul

### Added
- **Phosphor Icons** library (`@phosphor-icons/vue`) replace inline SVGs.
- **Branding config** — `[branding] logo_path / hospital_name / hospital_tagline / primary_color / primary_color_dark / accent_color`. Theme color CSS vars apply di document.documentElement, semua Vue components pakai `var(--color-primary, fallback)`.
- **Audio cue system** — `useAudioCue` composable synth tap/success/error/notify via Web Audio API (no asset bundle). Toggle on/off + volume di `[audio]` config.
- **Reusable components**: `BackButton` (floating bottom-left 64-72px), `ConfirmBackModal` (sopan, 2 button setara), `StepperBar` (segmented progress dengan ARIA `aria-current="step"`).
- **HomeScreen overhaul** (BB3) — welcome banner besar dengan greeting time-aware (Selamat pagi/siang/sore/malam) + ilustrasi PhSparkle, 1 hero "Pasien BPJS" 60% visual weight, 2 secondary cards setara (Pasien Umum + Antrian Loket).
- **AntrianScreen removed** — single-tap di Home → `apmService.createAntrian('LOKET','WALKIN')` → /tiket.
- **Antrian align ke Khanza V3 vendor pattern** — INSERT `antrian_loket` (type/noantrian/postdate/start_time/end_time), counter `SELECT MAX(noantrian) + 1` per type+CURDATE() + app-level mutex, format prefix+3-digit (A001/B015/C089). Mirror `DlgAmbilAntrean.java:240+`.

### Changed
- NumPad: "HAPUS" text label (no ambiguous backspace icon), CARI dengan PhMagnifyingGlass + larger label, touch 60→80px (was 52→72), gap 10→14px (was 6px) per WCAG 2.5.5 + senior UX research.
- InputDisplay: 4 dynamic states (empty / typing / can submit / full) dengan color shift emerald saat ready.

---

## [v1.0.1-mahatma] — BPJS user_key Header + Config Refresh

### Added
- `BPJSConfig.UserKey` field — wajib untuk decrypt response BPJS (AES-256-CBC). VClaim client kirim header `user_key` di setiap request.
- `config.example.toml` lengkap dengan semua field current — `khanza_dsn`, `khanza_kd_pj_*`, `user_key`, `window_class_*`, `startup_delay_sec`, `poll_interval_ms`.

---

## [v1.0.0-mahatma] — First Release: Smart Pendaftaran Direct-DB

### Added
- **Direct-MySQL Khanza client** — `MySQLClient` implement penuh `KhanzaClient` interface (10 methods). Switchable via `[server].khanza_dsn` — REST mode kept as fallback.
- **Pasien Umum end-to-end** — `EnrichPasien` JOIN 5 tabel master (kelurahan/kecamatan/kabupaten/propinsi), smart umur Th/Bl/Hr, `GetTarifPoli` pakai `poliklinik.registrasilama`, status_poli per-poli. 19-col reg_periksa INSERT validated against live DB.
- **Smart BPJS Detector** — auto-classify pasien ke 6 kategori paralel:
  - Priority: MJKN > Kontrol > PostRANAP > PostRAJAL > RujukanBaru > TidakAktif
  - Sources: booking_registrasi (MJKN fallback Antrol API), bridging_surat_kontrol_bpjs JOIN bridging_sep (Kontrol ≤30d), kamar_inap+dpjp_ranap (PostRANAP ≤7d), rujukan_internal_poli (PostRAJAL preferred / SKDP fallback), VClaim Rujukan (RujukanBaru).
- **BPJS bridging full** — `SimpanSEP` 18 kolom kritikal `bridging_sep` + auto `SimpanRujukMasuk` (rujuk_masuk) + auto `SimpanPRB` (bpjs_prb side-insert).
- **Vue per-pathway UI** — `PathwayMJKN/PostRANAP/PostRAJAL/TidakAktif` components + `SearchPasienScreen` + `RegistrasiUmumScreen` (3-step picker).
- **Hardware automation** Windows production:
  - Fingerprint After.exe `injectAfterLogin()` parametrize via config (WindowClass + StartupDelay) — cross-RS flexibility.
  - Frista `WindowsReader` real impl: spawn frista.exe (CREATE_NO_WINDOW), Win32 UI Automation login injection, clipboard polling untuk capture card scan (auto-detect JSON / pipe-delimited format). Pure Go syscall — no CGO.
- **GitHub Actions release workflow** — auto-build Mac universal `.app` + Windows amd64 `.exe` saat push tag `v*-mahatma`, publish GitHub Release dengan zip artifacts attached.

---

## Codebase Architecture (snapshot v1.5.0)

```
apm-go/
├── app.go                       Wails IPC bindings (50+ methods)
├── main.go                      Wails bootstrap
├── cmd/khanza-smoke/            Smoke test runner (build tag: smoke)
├── internal/
│   ├── config/                  TOML loader + AES encrypt/decrypt
│   │                            (branding/audio/server/bpjs/fingerprint/frista/printer/antrian/admin/dev sections)
│   ├── domain/                  Pure types (Pasien/SEP/SEPRequest/RencanaKontrolRequest/RujukanBPJS/Pendaftaran/...)
│   ├── service/
│   │   ├── detector/            Smart BPJS Detector (parallel 4 probe + priority)
│   │   ├── antrian/             Counter management
│   │   └── sep/                 SEP issuance orchestration (preflight + BuatPendaftaran + CreateSEP + persistAndSyncKhanza + audit trail)
│   ├── integration/
│   │   ├── khanza/              MySQL direct + REST dual-mode (15 methods)
│   │   ├── vclaim/              BPJS VClaim v2 (HMAC-SHA256 + AES-256-CBC + user_key header)
│   │   └── antrol/              BPJS Antrol API
│   ├── hardware/
│   │   ├── frista/              Mac mock + Windows real (clipboard pattern)
│   │   ├── fingerprint/         Mac mock + Windows real (After.exe headless via Win32 UI Automation)
│   │   └── printer/             Console / ESC/POS USB / Serial / Network — 3 templates dengan marker-based encoder
│   ├── store/                   SQLite local (sqlc-generated)
│   └── reconcile/               Offline queue worker
├── frontend/
│   ├── src/screens/             HomeScreen, InputScreen, DetectScreen, ResultScreen,
│   │                            SearchPasienScreen, RegistrasiUmumScreen, BantuSayaScreen,
│   │                            TicketScreen, AdminScreen
│   ├── src/components/          PathwayMJKN/PostRANAP/PostRAJAL/TidakAktif, BpjsLogo, BackButton,
│   │                            ConfirmBackModal, StepperBar, NumPad, InputDisplay, FristaBar, dll
│   ├── src/composables/         useAudioCue, useIdleTimeout, useClock
│   └── src/stores/              patient, branding (Pinia)
├── migrations/                  SQLite schema
├── .github/workflows/release.yml  Tag-driven Mac+Windows auto-build
├── PLAN_SMART_PENDAFTARAN.md    Original plan doc + audit findings
├── README.md                    Full deployment + config + ops guide
└── CHANGELOG.md                 (file ini)
```

## Tags Summary (chronological)

| Tag | Date (approx) | Theme |
|---|---|---|
| v1.0.0-mahatma | 2026-04-27 | Smart Pendaftaran direct-DB (first release) |
| v1.0.1-mahatma | 2026-04-27 | BPJS user_key header |
| v1.1.0-mahatma | 2026-04-27 | UX Foundation + HomeScreen overhaul |
| v1.2.0-mahatma | 2026-04-27 | InputScreen dual-channel + Stepper + MJKN |
| v1.2.1-mahatma | 2026-04-27 | Logo BPJS + printer guide |
| v1.2.2-mahatma | 2026-04-27 | Default printer USB POS-58 |
| v1.3.0-mahatma | 2026-04-27 | BPJS parity + Bantu Saya wizard |
| v1.4.0-mahatma | 2026-04-27 | Critical SEP flow fix |
| v1.4.1-mahatma | 2026-04-27 | Admin trigger (5x tap + Ctrl+Alt+A) |
| v1.4.2-mahatma | 2026-04-27 | Scope clarification |
| **v1.5.0-mahatma** | 2026-04-27 | Print template polish (current) |
