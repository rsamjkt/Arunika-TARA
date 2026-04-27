# APM · T.A.R.A

> **Arunihealth · T.A.R.A** — _Total Automated Registration Assistant_
> Anjungan Pasien Mandiri (APM) generasi baru untuk rumah sakit Indonesia, dibangun dengan **Go + Wails + Vue 3**, terintegrasi langsung dengan **SIMRS Khanza** (MySQL atau REST), **BPJS** (VClaim · Antrol · Mobile JKN), **Frista card reader**, dan **Aplikasi Sidik Jari BPJS**.

[![Go Version](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Wails](https://img.shields.io/badge/Wails-v2-FF0000)](https://wails.io)
[![Vue](https://img.shields.io/badge/Vue-3-4FC08D?logo=vue.js&logoColor=white)](https://vuejs.org)
[![Tailwind](https://img.shields.io/badge/Tailwind-3-38B2AC?logo=tailwind-css&logoColor=white)](https://tailwindcss.com)
[![Platform](https://img.shields.io/badge/Production-Windows%2010%2F11-0078D4?logo=windows&logoColor=white)]()
[![Dev](https://img.shields.io/badge/Dev-macOS%20%7C%20Linux-000000?logo=apple&logoColor=white)]()

---

## Daftar Isi

- [Tentang T.A.R.A](#tentang-tara)
- [Scope — Apa yang Bisa & Tidak Lewat Kiosk](#scope--apa-yang-bisa--tidak-lewat-kiosk)
- [Fitur Utama](#fitur-utama)
- [Smart BPJS Detector](#smart-bpjs-detector)
- [Quickstart — Pakai Aplikasi (Production)](#quickstart--pakai-aplikasi-production)
- [Konfigurasi `config.toml`](#konfigurasi-configtoml)
- [Setup Credential](#setup-credential)
- [Setup Hardware](#setup-hardware)
- [Operasi Sehari-hari](#operasi-sehari-hari)
- [Setup Development](#setup-development)
- [Build dari Source](#build-dari-source)
- [Arsitektur](#arsitektur)
- [Troubleshooting](#troubleshooting)
- [Lisensi](#lisensi)

---

## Tentang T.A.R.A

T.A.R.A adalah perangkat lunak Anjungan Pasien Mandiri yang berjalan sebagai **kiosk fullscreen** di rumah sakit. Pasien melakukan registrasi sendiri — tap kartu KTP, ketik NIK, atau ketik No. Kartu BPJS — sistem akan **otomatis mendeteksi jenis kunjungan**, membuat SEP, mendaftar ke poli, dan mencetak tiket antrian.

| Atribut | Detail |
|---|---|
| **Pemilik** | PT. Arunika Komputasi Awan Integrasi |
| **Codename rilis** | Mahatma (latest: v1.4.x) |
| **Repo** | [github.com/rsamjkt/Arunika-TARA](https://github.com/rsamjkt/Arunika-TARA) |
| **Stack** | Go 1.22+ · Wails v2 · Vue 3 · Tailwind CSS · SQLite |
| **Target Deploy** | Windows 10/11 x64 (kiosk) |
| **Development** | macOS / Linux (cross-compile ke Windows) |

---

## Scope — Apa yang Bisa & Tidak Lewat Kiosk

T.A.R.A di-design untuk **kasus mainstream pasien yang stabil dan bisa mandiri**. Edge case medis / administrasi rumit tetap ditangani petugas di loket / aplikasi vendor desktop.

### ✅ Yang DIDUKUNG kiosk (~85-90% kunjungan harian)

| Skenario | Flow |
|---|---|
| Pasien BPJS dengan rujukan FKTP baru | Auto-classify "RujukanBaru" → SEP issued |
| Pasien BPJS dengan jadwal kontrol (SKDP) | Auto-classify "Kontrol" → SEP kontrol |
| Pasien BPJS dari booking Mobile JKN | Auto-classify "MJKN" → konfirmasi → SEP |
| Pasien BPJS pasca rawat inap (≤7 hari) | Auto-classify "PostRANAP" → SEP follow-up |
| Pasien BPJS lanjutan rawat jalan beda poli | Auto-classify "PostRAJAL" → SEP follow-up |
| Pasien BPJS status tidak aktif | Auto-offer "Daftar sebagai Pasien Umum" |
| Pasien Umum / Tunai (sudah punya No. RM) | Search → pilih poli → pilih dokter → cetak |
| Antrian Loket admisi (single-tap) | Generate nomor lokal, cetak tiket |

### 🚫 Yang TIDAK didukung kiosk (lewat petugas / aplikasi vendor)

| Skenario | Alasan | Alternatif |
|---|---|---|
| **Pasien Laka Lantas** (kecelakaan) | Pasien sering luka berat, butuh IGD segera; penjamin Jasa Raharja primary, butuh assesmen petugas; dokumen polisi (BAP, SKBL) | Langsung IGD → petugas daftarkan via aplikasi vendor desktop |
| **Pendaftaran pasien baru** (belum punya No. RM) | Butuh KTP scan, isi data demografi lengkap, foto profil, hubungan keluarga | Loket admisi pasien baru |
| **Pasien dengan COB** (asuransi tambahan kompleks) | Butuh assesmen klaim primer/sekunder | Loket admisi |
| **Kelas naik / Eksekutif** (upgrade kelas rawat) | Butuh konfirmasi pembiayaan tambahan | Loket admisi |
| **Pasien Katarak** (operasi mata, butuh detail mata + stadium) | Klaim spesifik dengan field klinis detail | Petugas Poli MATA |
| **Kasus emergency** | Tidak ada waktu antri | IGD langsung |
| **Pasien dengan nomor rujukan ekspirasi/invalid** | Butuh re-issue rujukan dari FKTP | Pasien hubungi puskesmas FKTP dulu |
| **Pasien usia <17 tahun + BPJS** | Belum bisa fingerprint biometrik (di bawah usia BPJS) | Petugas verifikasi manual + daftarkan |

### Filosofi design

Kiosk bukan pengganti loket — pelengkap untuk percepat antrian. **Tujuannya**: pasien stabil yang sudah biasa kontrol/datang ke RS bisa mandiri tanpa antri panjang di loket admisi, sehingga **petugas loket bisa fokus ke kasus rumit** (laka, pasien baru, edge case BPJS) tanpa terdistraksi pasien sederhana.

Kalau pasien ragu-ragu pakai kiosk, footer ada tombol **"Panggil Petugas"** dan **"Bantu Saya"** wizard — pasien tidak akan terjebak.

---

## Fitur Utama

- 🧠 **Smart BPJS Detector** — auto-classify pasien ke 6 kategori (MJKN / Kontrol / PostRANAP / PostRAJAL / Rujukan Baru / Tidak Aktif) tanpa pilih manual operator.
- 🏥 **Pasien Umum lengkap** — INSERT 19 kolom `reg_periksa` benar (nama PJ, alamat lengkap join 5 tabel master, biaya dari `poliklinik.registrasilama`, smart umur Th/Bl/Hr).
- 🔌 **Khanza Dual-Mode** — Direct MySQL (mengikuti pola `anjunganmandiriSEP`) **atau** REST API Laravel — switchable via 1 baris config.
- 🆔 **Frista auto-launch** — spawn `frista.exe` headless, auto-login via Win32 UI Automation, capture data kartu via clipboard polling.
- 👆 **Aplikasi Sidik Jari BPJS auto-login** — spawn `After.exe` headless, auto-isi username/password, REST polling untuk hasil scan.
- 🖨️ **Thermal Printer ESC/POS** — USB / Serial / Network, template Go `text/template`.
- 📋 **Antrian 3 Jalur** — Loket Admisi / Poli / Umum dengan reset harian via cron.
- 🔄 **Offline Queue + Reconcile** — kalau Khanza atau BPJS down, data antri di SQLite lokal, sync otomatis saat pulih.
- 🔒 **Credential Encryption** — AES-256-GCM, key dari Windows DPAPI / Mac Keychain.
- 🪵 **PHI-Safe Logging** — NIK/No.Kartu/No.RM auto-mask di log.
- 🎨 **Kiosk UI Responsif** — Vue 3 + Tailwind `clamp()`, jalan mulus 15"–32" monitor.

---

## Smart BPJS Detector

Pasien input **satu identitas saja** (No. Kartu / NIK / No. RM). Sistem fire 5 pengecekan **paralel** dengan timeout 5 detik, lalu pilih kategori paling spesifik.

```
       Input identitas
              │
              ▼
       [VClaim · GetPeserta · validasi status aktif]  ← serial
              │
              ▼  4 paralel + fallback chain
   ┌──────────┬──────────┬──────────┬──────────┐
   │  MJKN    │ KONTROL  │ POST_RA  │ POST_RA  │
   │          │   BPJS   │   NAP    │   JAL    │
   ├──────────┼──────────┼──────────┼──────────┤
   │ Antrol   │ bridging_│ kamar_   │ rujukan_ │
   │  API     │ surat_   │ inap +   │ internal │
   │   ↓      │ kontrol_ │ dpjp_    │ _poli    │
   │ Khanza   │ bpjs     │ ranap    │   ↓      │
   │ booking_ │ JOIN     │  (window │  SKDP    │
   │ regis    │ bridging │  ≤7 hr)  │ fallback │
   │ (fall    │ _sep     │          │          │
   │  back)   │  (window │          │          │
   │          │  ≤30 hr) │          │          │
   └──────────┴──────────┴──────────┴──────────┘
              │
   Priority resolution (paling spesifik menang):
   MJKN > KONTROL > POSTRANAP > POSTRAJAL > RUJUKAN_BARU
```

**Yang membuat T.A.R.A lebih SMART** dibanding implementasi manual klasik:

1. **Auto-classify** — operator tidak perlu pilih jenis pasien
2. **Date-window** — Kontrol ≤30 hari, PostRANAP ≤7 hari, MJKN exact today
3. **Multi-source fallback** — kalau Antrol down, fallback ke `booking_registrasi` di Khanza
4. **Schema-aware mapping** — auto-translate `kd_poli` & `kd_dokter` BPJS ↔ kode RS via `maping_poli_bpjs` & `maping_dokter_dpjpvclaim`
5. **Graceful degradation** — kalau VClaim error, tetap bisa lanjut sebagai kategori DB-lokal

---

## Quickstart — Pakai Aplikasi (Production)

> Untuk install sebagai kiosk RS production. Build sudah jadi — tinggal deploy.

### 1. Download Release

Download dari [GitHub Releases](https://github.com/rsamjkt/Arunika-TARA/releases) — pilih sesuai platform:
- **macOS** (Universal — Intel + Apple Silicon): `apm-go-mac-universal.zip`
- **Windows** (x64): `apm-windows-amd64.zip`

### 2. Extract + Letak File

Layout file yang harus ada di **satu folder**:

```
APM/
├── apm-go.app                ← (Mac) atau apm.exe (Windows)
├── config.toml               ← edit sesuai environment RS
├── migrations/
│   └── 001_initial.sql       ← schema SQLite lokal
├── data/                     ← auto-created saat first launch
└── logs/                     ← auto-created
```

**Mac**: extract zip, lalu copy `config.example.toml` (dari repo) ke `config.toml` di folder yang sama dengan `.app`.

**Windows**: extract zip, lalu copy `config.example.toml` ke `config.toml` di folder yang sama dengan `apm.exe`.

### 3. Edit `config.toml`

Minimum yang harus diisi (lihat [section konfigurasi lengkap](#konfigurasi-configtoml)):

```toml
[server]
khanza_dsn = "user:password@tcp(IP_SERVER:3306)/nama_database?parseTime=true&loc=Local&timeout=5s"

[bpjs]
vclaim_url      = "https://apijkn.bpjs-kesehatan.go.id/vclaim-rest/"
cons_id         = "12345"           # dari kantor cabang BPJS
consumer_secret = "xxx-xxx-xxx"     # dari kantor cabang BPJS
user_key        = "xxxxxxxxxxxx"    # dari kantor cabang BPJS
```

### 4. Jalankan

**Mac** — pakai script launcher (recommended):

```bash
cd /path/to/APM
./run.sh
```

Atau manual via terminal:
```bash
cd /path/to/APM
APM_CONFIG_PATH=./config.toml ./apm-go.app/Contents/MacOS/apm-go
```

**Windows**:
- Double-click `apm.exe`, atau
- Run sebagai service (lihat [section Operasi](#operasi-sehari-hari))

### 5. Verifikasi

Cek `logs/apm.log` — harus muncul:

```json
{"level":"INFO","msg":"khanza: mode direct MySQL aktif"}
{"level":"INFO","msg":"app initialized","platform":"darwin","real_hardware":false}
```

Window kiosk fullscreen kebuka di layar — siap dipakai pasien.

---

## Konfigurasi `config.toml`

### Section `[app]`

```toml
[app]
idle_timeout_sec = 60                # detik idle sebelum auto-back ke home
log_level        = "info"            # debug | info | warn | error
log_dir          = "./logs"          # path output log
timezone         = "Asia/Jakarta"
version          = "1.0.0"
```

### Section `[server]` — koneksi SIMRS Khanza

**Mode 1 — Direct MySQL (recommended, lebih cepat)**:

```toml
[server]
khanza_url        = ""                # kosongkan
khanza_api_key    = ""
khanza_dsn        = "user:password@tcp(10.0.2.121:3306)/nama_db?parseTime=true&loc=Local&timeout=5s"
khanza_kd_pj_umum = "A03"             # kode penjamin Umum di tabel penjab RS-mu
khanza_kd_pj_bpjs = "BPJ"             # kode penjamin BPJS
timeout_ms        = 5000
retry             = 2
```

Kode `kd_pj` Umum/BPJS RS-spesifik. Cek dengan SQL:
```sql
SELECT kd_pj, png_jawab, status FROM penjab WHERE status='1';
```

**Mode 2 — REST API Khanza Laravel**:

```toml
[server]
khanza_url     = "http://192.168.1.10:8080"   # base URL Laravel API
khanza_api_key = "REPLACE_WITH_BEARER_TOKEN"
khanza_dsn     = ""                            # kosongkan
timeout_ms     = 10000
retry          = 2
```

Aplikasi auto-pilih mode berdasar mana yang diisi (DSN menang kalau keduanya non-kosong).

### Section `[bpjs]`

```toml
[bpjs]
# Production
vclaim_url      = "https://apijkn.bpjs-kesehatan.go.id/vclaim-rest/"
# Development
# vclaim_url    = "https://dvlp.bpjs-kesehatan.go.id/vclaim-rest/"

cons_id              = "12345"            # dari BPJS — angka konsumen ID
consumer_secret      = "secret-string"    # untuk HMAC-SHA256 signing
user_key             = "user-key-hex"     # API key BPJS (header X-cons-id)
antrol_url           = "https://apijkn.bpjs-kesehatan.go.id/antrean-rest/"
detector_timeout_ms  = 5000               # timeout fase paralel detector
```

**Cara dapat credential BPJS**:
1. RS daftar ke kantor cabang BPJS sebagai integrator
2. BPJS issue: `cons_id`, `consumer_secret`, `user_key`
3. RS kasih: IP address kiosk untuk di-whitelist BPJS

Tanpa whitelist → endpoint production akan return `connection refused`.

### Section `[fingerprint]` — Aplikasi Sidik Jari BPJS

```toml
[fingerprint]
exe_path           = "C:\\Program Files (x86)\\Aplikasi Sidik Jari BPJS Kesehatan\\After.exe"
rest_url           = "http://localhost:9999/finger-rest/"   # internal After.exe REST
username_enc       = "username_anda"                        # plaintext OK, atau ENC: prefix kalau encrypted
password_enc       = "password_anda"
scan_timeout_sec   = 30
poll_interval_ms   = 500

# UI Automation — class names dialog login After.exe
# Default Delphi VCL: TfrmLogin / TEdit / TButton — biasanya cukup
window_class_login    = "TfrmLogin"
window_class_edit     = "TEdit"
window_class_button   = "TButton"
startup_delay_sec     = 3                # tunggu setelah spawn sebelum inject
```

### Section `[frista]` — Card Reader

```toml
[frista]
exe_path          = "C:\\Program Files\\Frista\\frista.exe"
username_enc      = "username_frista"
password_enc      = "password_frista"
read_timeout_ms   = 1000
restart_on_crash  = true

# UI Automation — class names dialog login Frista
window_class_login    = "TfrmLogin"
window_class_edit     = "TEdit"
window_class_button   = "TButton"
startup_delay_sec     = 5                # Frista lebih lambat dari After.exe

# Polling clipboard untuk capture card data
poll_interval_ms      = 500
```

### Section `[printer]`

```toml
[printer]
# Mode pilihan:
#   "console"        → output ke stdout (Mac dev)
#   "escpos_usb"     → USB printer (Windows production)
#   "escpos_serial"  → Serial / RS232 (atau USB-to-Serial adapter)
#   "escpos_network" → Network printer (Wi-Fi / LAN)
mode      = "escpos_usb"
port      = "POS-58"                # USB: nama printer di System Settings
                                    # Serial: COM1 / /dev/cu.usbserial-A1234
                                    # Network: 192.168.1.50:9100
width_mm  = 58                      # 58 atau 80
```

**Cara cek nama printer**:
- **Windows**: `Settings → Bluetooth & devices → Printers & scanners` → copy nama persis
- **Mac**: `lpstat -p` → list printer terdaftar
- **Serial**: `ls /dev/cu.*` (Mac) atau Device Manager → COM Ports (Windows)

### Section `[antrian]`

```toml
[antrian]
loket_prefix = "A"                  # tiket loket: A-001, A-002, ...
poli_prefix  = "B"                  # tiket poli: B-001, B-002, ...
umum_prefix  = "C"                  # tiket umum: C-001, ...
reset_time   = "00:01"              # HH:MM WIB — auto-reset counter harian
```

### Section `[admin]`

```toml
[admin]
pin = "1234"                        # PIN 4-6 digit untuk akses admin panel
                                    # Kosongkan = panel admin tanpa PIN (dev)
```

### Section `[dev]`

```toml
[dev]
mock_hardware    = true             # Mac/Linux dev — auto-mock Frista/Fingerprint/Printer
mock_server_port = 9090             # HTTP mock untuk simulasi tap kartu
```

---

## Setup Credential

T.A.R.A perlu **3 set credential** — semua bisa disimpan plaintext untuk dev, atau di-enkripsi untuk production.

### 1. Khanza Database

DSN MySQL standar Go:
```
user:password@tcp(host:port)/database?parseTime=true&loc=Local&timeout=5s
```

Pakai **user database read-write** yang punya akses ke tabel:
- `pasien`, `reg_periksa`, `poliklinik`, `dokter`, `jadwal`, `penjab`
- `kamar_inap`, `dpjp_ranap`, `bangsal`, `kamar`
- `bridging_sep`, `bridging_surat_kontrol_bpjs`, `bridging_rujukan_bpjs`
- `rujuk_masuk`, `bpjs_prb`, `booking_registrasi`
- `rujukan_internal_poli`, `maping_poli_bpjs`, `maping_dokter_dpjpvclaim`
- `kelurahan`, `kecamatan`, `kabupaten`, `propinsi`, `flagging_pasien_satusehat`

User minimum dengan grant:
```sql
GRANT SELECT, INSERT, UPDATE, DELETE ON nama_db.* TO 'apm_user'@'IP_KIOSK';
FLUSH PRIVILEGES;
```

### 2. BPJS API

Setelah dapat `cons_id`, `consumer_secret`, `user_key` dari kantor cabang BPJS:

```toml
[bpjs]
cons_id         = "12345"
consumer_secret = "abc-def-ghi"
user_key        = "1234567890abcdef"
```

**TIDAK** perlu di-encrypt — credential ini di-share dengan vendor sistem RS (bukan personal).

### 3. Frista + Aplikasi Sidik Jari BPJS

Username/password yang dipakai operator login secara manual. Operator → IT minta credential → masukkan ke `config.toml`:

```toml
[fingerprint]
username_enc = "operator_bpjs"
password_enc = "rahasia123"

[frista]
username_enc = "operator_frista"
password_enc = "rahasia456"
```

> **Plaintext OK kalau `config.toml` di-protect dengan filesystem permission**. Tapi di kiosk yang multi-user fisik, _enkripsi wajib_.

#### Encrypt credential (production)

Jalankan sekali dari PowerShell sebagai Administrator (Windows) atau Terminal (Mac):

```powershell
.\apm.exe --encrypt-config
```

```bash
# Mac
./apm-go.app/Contents/MacOS/apm-go --encrypt-config
```

Apa yang terjadi:
1. App baca `config.toml`
2. Untuk setiap field `*_enc` yang plaintext → encrypt dengan AES-256-GCM
3. Master key di-derive dari **Windows DPAPI** (Windows) atau **Keychain** (Mac)
4. Field di-update jadi `ENC:base64hash...`

Setelah encrypt, `config.toml` aman di-share — tapi master key **tidak portable** (akun OS / mesin tertentu).

Contoh hasil:
```toml
[fingerprint]
username_enc = "ENC:KGc3Y2lQ..."
password_enc = "ENC:TmF4ZGV..."
```

App auto-decrypt saat startup. Tidak perlu rebuild atau setup ulang.

---

## Setup Hardware

### Frista Card Reader (Windows production)

1. **Install Frista** dari vendor (file dari BPJS / RSU)
2. **Konfigurasi reader USB** terpasang dan terdeteksi Windows
3. **Test manual** — buka `frista.exe`, login dengan operator credential, tap KTP → harus muncul data NIK + nama
4. **Edit `config.toml`** — set `[frista] exe_path` = path absolut ke `frista.exe`
5. **Edit `[frista] username_enc / password_enc`** — credential operator
6. **Restart app** — APM akan spawn frista.exe headless + auto-login + monitor clipboard

**Cara kerja capture card**: setelah login sukses, Frista output ke Windows clipboard setiap kartu di-tap. Format yang di-support:
- JSON: `{"nik":"...","nama":"...","tgl_lahir":"...","alamat":"...","no_kartu":"..."}`
- Pipe-delimited: `NIK#NAMA#TGL#ALAMAT#NO_KARTU`

### Aplikasi Sidik Jari BPJS (`After.exe`)

1. **Install dari BPJS** — _Aplikasi Sidik Jari BPJS Kesehatan_ (versi 2.0+)
2. **Test manual** — buka `After.exe`, login, scan jari → harus connect ke server BPJS
3. **Edit `config.toml`** — set `[fingerprint] exe_path`, `username_enc`, `password_enc`
4. **Restart app** — auto-login saat APM start

### Thermal Printer — Setup Detail

T.A.R.A support 4 mode printer di `[printer] mode`:

| Mode | Untuk apa | Dipakai kapan |
|---|---|---|
| `console` | Output ke stdout terminal | Mac/Linux dev |
| `escpos_usb` | USB printer terinstall di OS | **Production Windows** |
| `escpos_serial` | Serial RS-232 atau USB-to-Serial adapter | Printer dot-matrix lama |
| `escpos_network` | LAN/Wi-Fi printer (port 9100) | Shared printer satu jaringan |

#### Step 1 — Install printer driver

1. Hubungkan printer USB ke kiosk
2. Windows biasanya auto-detect (driver Generic / Text Only). Kalau gagal:
   - Install vendor driver dari CD / website (mis. https://gprinter.net, https://xprinter.net)
3. **Test cetak halaman test dari OS** dulu sebelum lanjut:
   - Settings → Bluetooth & devices → Printers & scanners → klik printer → Manage → Print test page

#### Step 2 — Cari nama printer / port

**Windows USB:**
```powershell
Get-Printer | Select-Object Name, PortName, DriverName
```
Sample output:
```
Name              PortName     DriverName
----              --------     ----------
EPSON TM-T82      USB001       Generic / Text Only
POS-58            USB002       Microsoft Generic / Text
```
→ Copy kolom `Name` (bukan `PortName`) — itu yang di-config.

**Serial / COM port:**
```powershell
[System.IO.Ports.SerialPort]::GetPortNames()
# Output: COM1, COM3, COM4
```

**Network:**
- IP printer biasanya di stiker bawah / panel printer LCD
- Port standar ESC/POS network: `9100`
- Format: `192.168.x.x:9100`

**Mac dev (test only):**
```bash
lpstat -p              # USB & network printer terdaftar
ls /dev/cu.usbserial-* # Serial via USB-Serial adapter
```

#### Step 3 — Edit `config.toml`

Pilih sesuai jenis printer. Setelah edit, **restart kiosk** (Cmd+Q + run lagi, atau `Restart-Service APM-TARA`).

**USB (paling umum di kiosk Windows):**
```toml
[printer]
mode      = "escpos_usb"
port      = "POS-58"            # nama persis dari Get-Printer
width_mm  = 58                  # 58 atau 80 sesuai printer
```

**Serial (printer dot-matrix / RS-232 via adapter):**
```toml
[printer]
mode      = "escpos_serial"
port      = "COM3"              # Windows COM port
# port    = "/dev/cu.usbserial-A1234"   # Mac/Linux
width_mm  = 80
```

**Network (shared LAN/Wi-Fi printer):**
```toml
[printer]
mode      = "escpos_network"
port      = "192.168.1.50:9100"
width_mm  = 80
```

#### Step 4 — Test cetak

Setelah restart APM:
1. Buka **Admin Panel** di kiosk (PIN dari `[admin] pin` config — default `1234`)
2. Klik **"Test Cetak Tiket"** — printer akan keluarkan struk dummy berisi nama RS + nomor sample
3. Atau test pasien beneran: ambil antrian loket → struk harus keluar otomatis

#### Troubleshooting cetak

| Gejala | Kemungkinan penyebab | Fix |
|---|---|---|
| Tidak ada cetak sama sekali | Driver salah, printer offline | Test print dari OS dulu (step 1) |
| `port not found` di log | Nama printer salah | Pakai persis output `Get-Printer` (sensitive case + spasi) |
| Cetak garbled / aneh | Encoding / width mismatch | Pastikan `width_mm` match printer; banyak printer Cina butuh 58mm meski casing 80mm |
| Cetak tapi terpotong | Margin terlalu lebar | Edit template ESC/POS di `templates/` folder |
| Network printer reject | Firewall block port 9100 | Buka inbound TCP 9100 di Windows Defender Firewall |
| Serial baud rate mismatch | Default 9600 tidak cocok | Cek manual printer; biasanya 9600/19200/115200 — masih hardcoded di kode (bisa di-config nanti) |

#### Print pattern T.A.R.A — apa yang dicetak kapan

| Aksi pasien | Yang dicetak | Tabel SQLite log |
|---|---|---|
| Ambil antrian loket (single-tap di Home) | Struk antrian: nomor (A001), waktu, nama RS, lokasi tunggu | `print_history` doc_type=ANTRIAN |
| Daftar Pasien Umum | Struk pendaftaran: no_rawat, no_urut, poli, dokter, jam, biaya | `print_history` doc_type=PENDAFTARAN |
| Daftar BPJS Rujukan Baru | Struk + no_sep + no_urut + DPJP | `print_history` doc_type=SEP |
| Daftar BPJS Kontrol (SKDP) | Struk + no_sep + SKDP info + DPJP | `print_history` doc_type=SEP_KONTROL |
| Reprint via Admin | Cetak ulang byte-stream dari `print_history` | (read-only) |

#### Logo BPJS di struk (opsional)

Kalau RS punya file logo BPJS resmi (PNG, 1-bit untuk kompresi terbaik), set di `config.toml`:
```toml
[branding]
bpjs_logo_path = "./assets/logo-bpjs.png"      # akan dipakai di kiosk UI + struk BPJS
```

Logo otomatis dipakai di:
- Hero "Pasien BPJS" di HomeScreen
- Header `InputScreen` saat mode BPJS
- Struk SEP yang dicetak (untuk pasien BPJS)

### Class Name UI Automation (kalau auto-login gagal)

Default class name Delphi VCL biasanya cukup. Tapi kalau Frista atau After.exe versi non-standar:

1. Install [Spy++](https://learn.microsoft.com/en-us/visualstudio/debugger/introducing-spy-increment) (Visual Studio Tools)
2. Buka dialog login Frista / After.exe secara manual
3. Drag Spy++ finder cursor ke window login
4. Catat **Class Name** dari window utama, dari TextBox, dari Button
5. Update di `config.toml`:
   ```toml
   [frista]
   window_class_login  = "ClassNameYangAnda"
   window_class_edit   = "..."
   window_class_button = "..."
   ```

---

## Operasi Sehari-hari

### Start / Stop kiosk

**Mac**:
```bash
cd /path/to/APM
./run.sh                        # start (foreground, Ctrl+C untuk stop)
```

**Windows** (Recommended — install sebagai service):
```powershell
# Sekali setup, run as Administrator
.\apm.exe --install-service --name "APM-TARA"
Set-Service -Name "APM-TARA" -StartupType Automatic
Start-Service -Name "APM-TARA"

# Verifikasi
Get-Service -Name "APM-TARA"

# Restart setelah edit config
Restart-Service -Name "APM-TARA"
```

Atau standalone:
```powershell
# Foreground
.\apm.exe

# Background (close console hidden)
Start-Process -FilePath ".\apm.exe" -WindowStyle Hidden
```

### Setelah edit `config.toml`

1. **Stop app** (Cmd+Q di Mac, atau `Restart-Service` di Windows)
2. **Edit `config.toml`** dengan editor pilihan
3. **Start app** lagi — config baru langsung dipakai

> Tidak perlu rebuild `.app` / `.exe` — config di-load runtime.

### Restart harian

Recommended setup cron Windows untuk restart kiosk setiap pagi:

```powershell
# Jalankan sebagai Administrator di Task Scheduler
schtasks /create /tn "APM Restart Pagi" /tr "powershell Restart-Service APM-TARA" /sc daily /st 04:00
```

### Lihat log

Logs di `logs/apm.log` (relatif ke working directory). Format JSON, structured:
```json
{"time":"2026-04-27T09:17:55","level":"INFO","msg":"khanza: mode direct MySQL aktif"}
```

Tail real-time (Mac):
```bash
tail -f logs/apm.log | jq
```

Tail (Windows PowerShell):
```powershell
Get-Content logs\apm.log -Wait -Tail 20
```

### Backup data

`data/apm.db` (SQLite) berisi:
- `print_history` — backup tiket cetak (untuk reprint)
- `pending_sep` — antrian SEP yang belum sync ke Khanza (offline mode)
- `antrian_counter` — counter antrian harian

Backup harian recommended:
```bash
# Cron Mac
0 3 * * * cp /path/to/APM/data/apm.db /backup/apm-$(date +\%F).db
```

```powershell
# Task Scheduler Windows
Copy-Item C:\APM\data\apm.db D:\Backup\apm-$(Get-Date -Format yyyy-MM-dd).db
```

---

## Setup Development

> Untuk developer yang mau modifikasi kode T.A.R.A.

### Prerequisites Mac

```bash
# Homebrew + tools
brew install go node mingw-w64

# Wails CLI v2
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Verifikasi
go version           # 1.22+
node --version       # 18+
wails version        # v2.x
x86_64-w64-mingw32-gcc --version    # untuk cross-compile Windows
```

### Clone + Run Dev

```bash
git clone https://github.com/rsamjkt/Arunika-TARA.git
cd Arunika-TARA

# Setup config (placeholder values OK untuk dev awal)
cp config.example.toml config.toml
# Edit minimal: khanza_dsn, bpjs.cons_id

# Install deps
go mod download
cd frontend && npm install && cd ..

# Run dev (hot reload Go + Vue)
make dev
```

Window kiosk kebuka, kalau Mac dev hardware otomatis di-mock.

### Simulasi Hardware (Mac dev)

Frista mock punya HTTP server di port 9090:

```bash
# Simulasi tap KTP
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
```

### Smoke Test Khanza Connection

Sebelum run dev, validate koneksi MySQL dulu:

```bash
APM_KHANZA_DSN='user:pass@tcp(10.0.2.121:3306)/db?parseTime=true&timeout=5s&loc=Local' \
APM_QUERY=Budi \
go run -tags smoke ./cmd/khanza-smoke
```

Output:
- HealthCheck status
- CariPasien hasil
- 5 poli aktif
- Jadwal dokter hari ini
- SKDP BPJS pasien (Phase B probes)
- Optional write test (set `APM_WRITE_TEST=1` untuk INSERT + auto-rollback)

---

## Build dari Source

### Build Mac (.app)

```bash
PATH="$PATH:$HOME/.local/node/bin" make build-mac
# Output: build/bin/apm-go.app (universal Intel + Apple Silicon, ~12 MB)
```

### Build Windows (.exe) — cross-compile dari Mac

```bash
# Sekali setup
brew install mingw-w64

# Build
PATH="$PATH:$HOME/.local/node/bin" make build-windows
# Output: build/bin/apm.exe
```

### Build di mesin Windows native

Install Wails CLI di Windows + run `wails build`.

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails build -platform windows/amd64
```

### Test

```bash
# Unit tests
make test

# Coverage report
make test-coverage              # target ≥80% untuk business logic

# Lint
make lint                       # golangci-lint

# Smoke test live (read-only)
APM_KHANZA_DSN='...' go run -tags smoke ./cmd/khanza-smoke

# Smoke test write (INSERT + auto-rollback)
APM_KHANZA_DSN='...' APM_WRITE_TEST=1 go run -tags smoke ./cmd/khanza-smoke
```

---

## Arsitektur

```
apm-go/
├── app.go                      ← Wails App struct (entry IPC Go ↔ Vue)
├── main.go                     ← Wails app bootstrap
├── cmd/
│   └── khanza-smoke/           ← Smoke test runner (build tag: smoke)
├── internal/
│   ├── config/                 ← TOML loader + AES encrypt/decrypt
│   ├── domain/                 ← Pure structs (Pasien, SEP, Pendaftaran, dll)
│   ├── service/
│   │   ├── detector/           ← ★ Smart BPJS Detector
│   │   ├── antrian/            ← Counter management
│   │   └── sep/                ← SEP builder + VClaim integration
│   ├── integration/
│   │   ├── khanza/             ← Direct MySQL + REST client (dual-mode)
│   │   ├── vclaim/             ← BPJS VClaim v2 (HMAC-SHA256)
│   │   └── antrol/             ← BPJS Antrol API
│   ├── hardware/
│   │   ├── frista/             ← Mac mock + Windows real (clipboard pattern)
│   │   ├── fingerprint/        ← Mac mock + Windows real (After.exe headless)
│   │   └── printer/            ← Console / ESC/POS USB / Serial / Network
│   ├── store/                  ← SQLite local (sqlc-generated)
│   └── reconcile/              ← Offline queue worker
├── frontend/
│   ├── src/screens/            ← HomeScreen, InputScreen, DetectScreen,
│   │                              ResultScreen, SearchPasienScreen,
│   │                              RegistrasiUmumScreen, TicketScreen, ...
│   ├── src/components/         ← PathwayMJKN, PathwayPostRANAP, dll
│   └── src/stores/             ← Pinia: patient, antrian, detection
├── migrations/                 ← SQLite schema
├── templates/                  ← ESC/POS print templates
├── config.example.toml         ← Template — JANGAN commit config.toml asli
└── Makefile                    ← dev / build-mac / build-windows / test / lint
```

---

## Troubleshooting

### "khanza: gagal connect MySQL — fallback ke REST"

DSN salah atau MySQL tidak reach. Test dengan smoke runner:

```bash
APM_KHANZA_DSN='...' go run -tags smoke ./cmd/khanza-smoke
```

Cek juga firewall + MySQL `bind-address` + grant user.

### "registrasi BPJS untuk no_kartu pada YYYY-MM-DD tidak ditemukan"

`SimpanSEP` gagal resolve `no_rawat` karena belum ada `reg_periksa` BPJS hari ini untuk pasien tsb. Pastikan `BuatPendaftaran` (BPJS, kd_pj=BPJ) dijalankan dulu sebelum `SimpanSEP`.

### Frista auto-login gagal — operator ketik manual

Cek `logs/apm.log`:
```
"frista: gagal inject login (operator mungkin perlu manual)"
"err":"dialog login Frista (class=\"TfrmLogin\") tidak ditemukan: timeout..."
```

Berarti class name dialog beda. Pakai Spy++ verify, override di `[frista] window_class_login`.

### "tabel rencana_kontrol doesn't exist"

Tidak masalah — RS pakai SKDP BPJS langsung dari `bridging_surat_kontrol_bpjs`. Detector sudah switch ke source ini di v1.0.0+.

### Window tidak muncul saat launch app (.app)

CWD salah. App perlu `config.toml` + `migrations/` di working directory. Pakai `run.sh` launcher (auto-cd) atau set env:

```bash
APM_CONFIG_PATH=/absolute/path/to/config.toml ./apm-go.app/Contents/MacOS/apm-go
```

### "device or resource busy" saat clipboard polling Frista

Ada aplikasi lain sering pakai clipboard (mis. password manager, clipboard history). Tutup app yang konflik atau naikkan `[frista] poll_interval_ms` ke 1000.

---

## Lisensi

**© 2026 PT. Arunika Komputasi Awan Integrasi**

Proprietary software. Untuk pertanyaan, kerja sama, atau dukungan teknis, hubungi tim IT melalui [GitHub Issues](https://github.com/rsamjkt/Arunika-TARA/issues).

---

## Credits

- Pattern direct-DB Khanza diilhami dari [`RS-INDRIATI/anjunganmandiriSEP`](https://github.com/RS-INDRIATI/anjunganmandiriSEP) — Java reference implementation
- Smart Detector + per-pathway UI = original kontribusi T.A.R.A
- Built with ☕ untuk **healthcare** Indonesia

---

> **T.A.R.A · Mahatma** — _Total Automated Registration Assistant_
