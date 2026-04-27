# Arunihealth T.A.R.A — _Total Automated Registration Assistant_

> Self-service kiosk for hospital outpatient registration, BPJS SEP issuance, and queue management. Direct-DB integration with SIMRS Khanza, Smart BPJS Detector with auto-classification, and full Windows hardware automation (Frista card reader, Aplikasi Sidik Jari BPJS, ESC/POS thermal printer).

## Highlights

### 🏥 Pasien Umum (Walk-in Registration) — Complete
- Full 19-column `reg_periksa` INSERT validated against production schema
- Smart age unit (Th/Bl/Hr) computed from `pasien.tgl_lahir`
- Full address concatenation from `pasien` + `kelurahan` + `kecamatan` + `kabupaten` + `propinsi`
- `p_jawab` / `almt_pj` / `hubunganpj` auto-derived from master data
- Tariff from `poliklinik.registrasilama`
- Per-poliklinik visit history detection
- 3-step kiosk UI: search patient → pick poli → pick doctor → confirm → print ticket

### 🧠 Smart BPJS Detector — Auto-Classify
The Java reference implementation requires the operator to manually select the patient pathway. T.A.R.A automatically classifies patients into one of six categories using parallel data probes with priority resolution:

| Priority | Category | Source |
|----------|----------|--------|
| 1 | Mobile JKN booking | Antrol API → Khanza `booking_registrasi` fallback |
| 2 | Surat Kontrol (SKDP) | `bridging_surat_kontrol_bpjs` JOIN `bridging_sep`, ≤30 day window |
| 3 | Post-RANAP (post-discharge) | `kamar_inap` + `dpjp_ranap`, ≤7 day window |
| 4 | Post-RAJAL (inter-poli referral) | `rujukan_internal_poli` (preferred) / SKDP fallback |
| 5 | Rujukan Baru FKTP | VClaim API |
| Default | TidakAktif / RujukanBaru | Safe fallback |

All probes fire in parallel with a 5-second timeout; the most specific match wins.

### 🔌 Direct-DB Khanza Integration
Native Go MySQL client connecting directly to SIMRS Khanza database — no REST middleware required. Mirrors the `anjunganmandiriSEP` Java reference pattern with:
- Configurable kd_pj mapping (default: A03 = Umum, BPJ = BPJS)
- RS code translation via `maping_poli_bpjs` and `maping_dokter_dpjpvclaim`
- Transactional `BuatPendaftaran` with duplicate detection and atomic no_reg / no_rawat generation
- Full bridging integration: `bridging_sep` (18 critical columns), `rujuk_masuk`, `bpjs_prb`
- Graceful fallback to REST mode if `khanza_dsn` is empty

### 🖨️ Kiosk Hardware (Windows Production)
- **Frista card reader** auto-launch + auto-login via Win32 UI Automation; clipboard polling for card data capture (auto-detect JSON / pipe-delimited format)
- **After.exe (BPJS Sidik Jari)** auto-launch + auto-login + REST polling for biometric verification
- All Win32 calls use pure Go syscall — no CGO, supports cross-compilation from macOS via mingw-w64
- Configurable window class names per RS without recompilation

### 🔒 Security Hardening
- AES-256-GCM credential encryption with `ENC:` prefix support for `khanza_dsn`, fingerprint and frista credentials
- Master key derivation from OS keychain (Mac) / DPAPI (Windows)
- PHI log masking via custom `slog.Handler` wrapper

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
# edit config.toml — fill khanza_dsn + bpjs credentials
./run-mac.sh
```

First launch on macOS: right-click `apm-go.app` → Open (to bypass Gatekeeper unsigned warning).

### Windows

```cmd
REM extract apm-windows-amd64.zip ke C:\APM
cd C:\APM
copy config.example.toml config.toml
notepad config.toml
REM ...edit khanza_dsn + bpjs credentials...
run-windows.bat
```

For unattended kiosk operation, install as Windows Service — see [README](https://github.com/rsamjkt/Arunika-TARA/blob/main/README.md#operasi-sehari-hari).

## Documentation

Full documentation in [README.md](https://github.com/rsamjkt/Arunika-TARA/blob/main/README.md):
- Configuration reference (every section explained)
- Credential setup (BPJS API onboarding, Frista + After.exe creds, encryption)
- Hardware setup (USB / serial / network printer; Spy++ for window class discovery)
- Operations runbook (start/stop, restart after config edit, log tailing, backups)
- Development guide
- Troubleshooting

## What's Next

- Antrol checkin endpoint wiring (currently UI placeholder)
- Dedicated DokterPickerScreen for PostRANAP/PostRAJAL pathways
- ESC/POS template polish for 80mm printers
- Multi-cabang RS support
- Admin dashboard analytics

---

Built with ❤️ for healthcare in Indonesia — based on the patterns established by `anjunganmandiriSEP` (RS Indriati), reimagined in Go for native cross-platform deployment and a smarter, friendlier patient experience.
