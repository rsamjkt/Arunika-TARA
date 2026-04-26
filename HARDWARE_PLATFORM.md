# Hardware & Platform — Mac Dev vs Windows Production

> File ini menjelaskan strategi dual-platform: develop di Mac, deploy di Windows.

## Prinsip Utama

**SATU codebase, TWO runtime behaviors.**  
Semua hardware-specific code dibungkus interface.  
`runtime.GOOS` menentukan implementasi yang dipakai saat startup.

## Platform Detection Pattern

```go
// internal/hardware/provider.go
// SATU-SATUNYA tempat platform detection dilakukan

package hardware

import "runtime"

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
    default: // darwin (Mac), linux — development mode
        return &Provider{
            Frista:      frista.NewMock(cfg.Dev.MockServerPort),
            Fingerprint: fingerprint.NewMock(),
            Printer:     printer.NewConsolePrinter(), // print ke stdout
        }
    }
}
```

## Mac Development — Cara Kerja

### 1. Frista Mock

Di Mac, `MockFristaReader` expose HTTP endpoint untuk simulasi tap kartu:

```bash
# Terminal 1: jalankan APM dev mode
make dev

# Terminal 2: simulasi tap kartu KTP
curl -X POST http://localhost:9090/mock/card-read \
  -H "Content-Type: application/json" \
  -d '{
    "nik": "3271234567890001",
    "nama": "Budi Santoso",
    "tgl_lahir": "1980-05-15",
    "alamat": "Jl. Merdeka No. 1, Jakarta",
    "no_kartu": "0001234567890012"
  }'
```

Vue frontend akan menerima event `frista:card_read` dan auto-fill form.

### 2. Fingerprint Mock

Mock FP selalu return success setelah 2 detik delay (simulasi scan):

```go
// internal/hardware/fingerprint/mock.go
func (m *MockVerifier) Verify(ctx context.Context, noPeserta string) (FPResult, error) {
    // Simulasi delay scan
    select {
    case <-time.After(2 * time.Second):
        return FPResult{Success: true, Token: "MOCK_FP_TOKEN_" + noPeserta}, nil
    case <-ctx.Done():
        return FPResult{}, ctx.Err()
    }
}
```

Untuk simulasi failure:
```bash
curl -X POST http://localhost:9090/mock/fp-fail
# Setelah ini, Verify() akan return error untuk 1x call berikutnya
```

### 3. Printer Mock

Semua ESC/POS command di-log ke stdout dalam format readable:

```
[PRINTER] === CETAK TIKET ANTRIAN ===
[PRINTER] Jenis: LOKET ADMISI (WALK-IN)
[PRINTER] Nomor: A-035
[PRINTER] Tanggal: 25 Apr 2026 08:47
[PRINTER] ================================
[PRINTER] *** PRINT COMPLETE ***
```

### 4. BPJS API — Development vs Production

```toml
# config.toml — bagian yang berbeda antara dev dan prod

[bpjs]
# Untuk development: gunakan endpoint dev BPJS
vclaim_url    = "https://dvlp.bpjs-kesehatan.go.id/vclaim-rest/"
cons_id       = "DEV_CONS_ID"      # dari BPJS untuk development
consumer_secret = "DEV_SECRET"

# Untuk production: 
# vclaim_url  = "https://apijkn.bpjs-kesehatan.go.id/vclaim-rest/"
# cons_id     = "PROD_CONS_ID"
```

BPJS menyediakan akun development terpisah. Hubungi kantor cabang BPJS untuk mendapatkan credential dev.

## Windows Production — Implementasi Nyata

### Frista Windows Implementation

```go
// internal/hardware/frista/windows.go
// +build windows

package frista

import (
    "os/exec"
    "syscall"
    // CGO untuk Windows UI Automation
)

type WindowsFristaReader struct {
    cmd    *exec.Cmd
    ch     chan CardData
    cfg    Config
}

func (r *WindowsFristaReader) Start(ctx context.Context) error {
    // 1. Spawn frista.exe hidden
    r.cmd = exec.CommandContext(ctx, r.cfg.ExePath)
    r.cmd.SysProcAttr = &syscall.SysProcAttr{
        CreationFlags: syscall.CREATE_NO_WINDOW,
    }
    
    // 2. Pipe stdout untuk baca JSON output
    stdout, _ := r.cmd.StdoutPipe()
    r.cmd.Start()
    
    // 3. Auto-login via Windows UI Automation
    // (butuh delay 2-3 detik untuk app load)
    time.Sleep(3 * time.Second)
    r.injectLogin()
    
    // 4. Start reading goroutine
    go r.readLoop(ctx, stdout)
    return nil
}

func (r *WindowsFristaReader) injectLogin() {
    // Windows SendMessage / UI Automation untuk isi username & password
    // Implementasi menggunakan CGO + user32.dll
    // FindWindowW("TfrmLogin") → FindWindowEx → SendMessageW(WM_SETTEXT)
    // ... lihat windows_ui_automation.go untuk detail
}
```

### Build Tag Strategy

```go
// File naming convention:
// frista_windows.go    → kompilasi hanya di Windows (build tag otomatis)
// frista_mock.go       → kompilasi di Mac/Linux
// frista_interface.go  → interface (semua platform)
```

### Makefile Cross-Compile

```makefile
# Makefile

.PHONY: dev build-windows test

# Development di Mac — dengan hot reload
dev:
    wails dev

# Build untuk Windows dari Mac (cross-compile)
build-windows:
    GOOS=windows GOARCH=amd64 CGO_ENABLED=1 \
    CC=x86_64-w64-mingw32-gcc \
    wails build -platform windows/amd64

# Install cross-compile toolchain di Mac
setup-cross-compile:
    brew install mingw-w64

# Run tests (semua platform)
test:
    go test ./... -v -cover

# Lint
lint:
    golangci-lint run ./...
```

## Deployment ke Windows — Checklist

### Prasyarat di Mesin Windows

```
□ Windows 10 Pro/Enterprise x64 (build 1903+) atau Windows 11
□ WebView2 Runtime terinstal (biasanya sudah ada di Win10/11 modern)
  → Download: https://developer.microsoft.com/en-us/microsoft-edge/webview2/
□ Microsoft Visual C++ Redistributable 2022 (untuk CGO runtime)
□ Frista.exe dari vendor Frista — path dikonfigurasi di config.toml
□ After.exe (Aplikasi Sidik Jari BPJS Kesehatan v2.0+)
  → Default path: C:\Program Files (x86)\Aplikasi Sidik Jari BPJS Kesehatan\After.exe
□ Thermal printer terhubung (USB atau Serial)
□ Koneksi LAN ke server Khanza
□ Koneksi internet untuk BPJS API
```

### Encrypt Credential di Config (Windows)

```bash
# Jalankan sekali setelah copy config.toml
apm.exe --encrypt-config

# Program akan prompt:
# Enter Frista username: [ketik username]
# Enter Frista password: [ketik password]
# Enter FP BPJS username: [ketik username]
# Enter FP BPJS password: [ketik password]
#
# Output: config.toml diupdate dengan nilai ENC:... (AES-256-GCM)
# Master key: Windows DPAPI (terikat ke mesin ini)
```

### Install sebagai Windows Service

```powershell
# PowerShell (run as Administrator)

# Install service
.\apm.exe --install-service --name "APMService" --display "Anjungan Pasien Mandiri"

# Set startup type: automatic
Set-Service -Name "APMService" -StartupType Automatic

# Start service
Start-Service -Name "APMService"

# Verifikasi
Get-Service -Name "APMService"

# View logs
Get-EventLog -LogName Application -Source "APMService" -Newest 50
```

## Environment Variables untuk Testing

```bash
# Mac — override config tanpa edit file
export APM_KHANZA_URL="http://192.168.1.10:8080"
export APM_BPJS_CONS_ID="test_cons_id"
export APM_BPJS_SECRET="test_secret"
export APM_DEV_MOCK_HARDWARE=true

# Atau gunakan .env file (di-load oleh Viper)
cp config.example.toml config.toml
# Edit sesuai environment
```

## Troubleshooting Umum

### Mac: Wails dev tidak bisa start
```bash
# Pastikan XCode Command Line Tools terinstall
xcode-select --install

# Pastikan Go dan Wails versi benar
go version  # harus 1.22+
wails version  # harus v2.x
```

### Windows: After.exe tidak bisa di-spawn hidden
```
Kemungkinan: After.exe membutuhkan privilege elevasi
Solusi: Jalankan APM sebagai Administrator, atau set manifest UAC
```

### Windows: Frista tidak terdeteksi
```
1. Cek device manager: Frista harus muncul sebagai HID device
2. Cek path di config.toml
3. Coba jalankan frista.exe manual dulu untuk verifikasi
4. Cek log APM: tail -f C:\apm\logs\apm.log
```

