# Quickstart — APM-Go di Mac untuk Testing

## 1. Install Prerequisites

```bash
# Install Go 1.22+
brew install go

# Install Wails v2
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Install Node.js 18+ (untuk Vue frontend)
brew install node

# Install cross-compile toolchain (untuk build Windows dari Mac)
brew install mingw-w64

# Verifikasi
go version       # go1.22+
wails version    # v2.x
node --version   # v18+
```

## 2. Clone & Setup

```bash
git clone <repo-url>
cd apm-go

# Copy config template
cp config.example.toml config.toml

# Edit config — minimal yang perlu diisi untuk dev di Mac:
# [server] khanza_url = "http://IP-SERVER-KHANZA:PORT"
# [bpjs] cons_id dan consumer_secret dari BPJS dev
# Sisanya bisa pakai default (hardware di-mock otomatis di Mac)

# Install Go dependencies
go mod download

# Install Vue dependencies
cd frontend && npm install && cd ..
```

## 3. Jalankan di Mac (Dev Mode)

```bash
# Mode development — hot reload Vue + Go
make dev
# atau
wails dev
```

Browser dev tools otomatis terbuka. APM berjalan di window Wails.

## 4. Simulasi Hardware di Mac

```bash
# Terminal baru — simulasi tap kartu Frista
make mock-card-read NIK=3271234567890001 NAMA="Budi Santoso"

# Simulasi gagal fingerprint
make mock-fp-fail

# Lihat "cetak" tiket (stdout)
# → Muncul otomatis di terminal saat user selesai antrian/SEP
```

## 5. Jalankan Tests

```bash
# Unit tests semua
make test

# Dengan coverage report
make test-coverage

# Integration tests (butuh mock BPJS server)
make test-integration
```

## 6. Build untuk Windows

```bash
# Cross-compile dari Mac
make build-windows
# Output: dist/apm-windows-amd64/apm.exe

# Transfer ke Windows via SCP atau USB
scp dist/apm-windows-amd64/apm.exe user@windows-machine:C:/apm/
```

## Pertanyaan Umum

**Q: Frista dan Fingerprint BPJS hanya di Windows — gimana di Mac?**  
A: Otomatis pakai mock. Tidak perlu konfigurasi apapun. Lihat `HARDWARE_PLATFORM.md`.

**Q: BPJS API tidak bisa diakses dari Mac kantor?**  
A: Gunakan endpoint development BPJS: `https://dvlp.bpjs-kesehatan.go.id/vclaim-rest/`  
   Minta credential dev ke kantor cabang BPJS atau hubungi tim IT RS.

**Q: Khanza server ada di jaringan RS, tidak bisa diakses dari luar?**  
A: Opsi: (1) VPN ke jaringan RS, (2) SSH tunnel, (3) Set `APM_KHANZA_URL` ke URL yang bisa diakses.

**Q: Urutan file mana yang harus dibaca Claude Code?**  
A: SELALU mulai dengan `CLAUDE.md`. Sisanya sesuai kebutuhan task:
   - Task BPJS → baca `BPJS_INTEGRATION.md`
   - Task UI → baca `DESIGN_SYSTEM.md`  
   - Task hardware → baca `HARDWARE_PLATFORM.md`
   - Copy prompt dari `PROMPTS.md`

