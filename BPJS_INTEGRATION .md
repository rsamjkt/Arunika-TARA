# BPJS Integration — Detail Teknis Lengkap

> Memory khusus untuk semua integrasi BPJS: VClaim, Antrol, Mobile JKN, Fingerprint

## Smart BPJS Detector Engine

### Cara Kerja (WAJIB dipahami sebelum menyentuh `service/detector/`)

Pasien input SATU identitas → sistem jalankan 5 check paralel → kembalikan 1 kategori:

```
Input (NoKartu / NIK / NoRM)
        │
        ▼
[1] VClaim: GET Peserta → validasi status aktif (SERIAL, wajib dulu)
        │
        ▼ (paralel, timeout 5 detik)
┌───────┬──────────┬────────────┬────────────┐
│ MJKN  │ KONTROL  │ POST_RANAP │ POST_RAJAL │
│Antrol │ Khanza   │  Khanza    │  Khanza    │
│ API   │ surat    │  kamar_    │ reg_periksa│
│       │ kontrol  │  inap      │            │
└───┬───┴────┬─────┴─────┬──────┴────┬───────┘
    │        │            │           │
    └────────┴────────────┴───────────┘
                    │
            Priority resolution:
            MJKN > KONTROL > POST_RANAP > POST_RAJAL > RUJUKAN_BARU
```

### Enum PatientType (domain/detection.go)

```go
type PatientType int

const (
    PatientTypeUnknown    PatientType = iota
    PatientTypeMJKN                   // Booking Mobile JKN aktif hari ini
    PatientTypeKontrol                 // Ada surat kontrol valid (tgl = today)
    PatientTypePostRANAP               // Baru keluar rawat inap (kemarin/hari ini)
    PatientTypePostRAJAL               // Ada kunjungan RAJAL aktif, kontrol beda poli
    PatientTypeRujukanBaru             // Default: kunjungan baru dengan rujukan FKTP
    PatientTypeTidakAktif              // Status kepesertaan tidak aktif
    PatientTypeError                   // Tidak bisa dicek (network error, dll)
)
```

### Implementasi Detector (service/detector/detector.go)

```go
func (d *Detector) Detect(ctx context.Context, input PatientInput) DetectionResult {
    // Step 1: Wajib serial — lookup peserta
    peserta, err := d.vclaim.GetPeserta(ctx, input.Identifier, time.Now())
    if err != nil {
        return DetectionResult{Type: PatientTypeError, Err: err}
    }
    if !peserta.IsAktif() {
        return DetectionResult{Type: PatientTypeTidakAktif, Peserta: peserta}
    }

    // Step 2: 5 detik timeout untuk semua check paralel
    ctx5, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    type result struct {
        pType PatientType
        data  any
        hit   bool
    }
    ch := make(chan result, 4)

    go func() {
        if hit, data := d.checkMJKN(ctx5, peserta.NoKartu); hit {
            ch <- result{PatientTypeMJKN, data, true}
        } else { ch <- result{} }
    }()
    go func() {
        if hit, data := d.checkKontrol(ctx5, peserta.NoRM); hit {
            ch <- result{PatientTypeKontrol, data, true}
        } else { ch <- result{} }
    }()
    go func() {
        if hit, data := d.checkPostRANAP(ctx5, peserta.NoRM); hit {
            ch <- result{PatientTypePostRANAP, data, true}
        } else { ch <- result{} }
    }()
    go func() {
        if hit, data := d.checkPostRAJAL(ctx5, peserta.NoRM); hit {
            ch <- result{PatientTypePostRAJAL, data, true}
        } else { ch <- result{} }
    }()

    // Collect 4 results, apply priority
    hits := map[PatientType]any{}
    for i := 0; i < 4; i++ {
        r := <-ch
        if r.hit { hits[r.pType] = r.data }
    }

    // Priority order
    for _, t := range []PatientType{
        PatientTypeMJKN, PatientTypeKontrol,
        PatientTypePostRANAP, PatientTypePostRAJAL,
    } {
        if data, ok := hits[t]; ok {
            return DetectionResult{Type: t, Peserta: peserta, Data: data}
        }
    }

    return DetectionResult{Type: PatientTypeRujukanBaru, Peserta: peserta}
}
```

## VClaim API v2.0

### Autentikasi (WAJIB — setiap request)

```go
// integration/vclaim/auth.go
func (c *Client) sign(timestamp int64) string {
    msg := fmt.Sprintf("%s&%d", c.consID, timestamp)
    mac := hmac.New(sha256.New, []byte(c.secretKey))
    mac.Write([]byte(msg))
    return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func (c *Client) headers(timestamp int64) map[string]string {
    return map[string]string{
        "X-cons-id":   c.consID,
        "X-timestamp": strconv.FormatInt(timestamp, 10),
        "X-signature": c.sign(timestamp),
        "Content-Type": "application/json",
    }
}
```

### Dekripsi Response v2.0

```go
// integration/vclaim/decrypt.go
// VClaim v2 mengenkripsi semua response body dengan AES-256-CBC
func (c *Client) decrypt(ciphertext string) ([]byte, error) {
    key := sha256.Sum256([]byte(c.secretKey + c.consID))
    // ... AES-256-CBC decrypt dengan PKCS7 padding
    // Key = SHA256(secretKey + consID) — 32 bytes
    // IV  = first 16 bytes of key
}
```

### Endpoint Map — Semua yang Dipakai APM

```
GET  /Peserta/noKartu/{noKartu}/{tglSEP}
     → Lookup peserta, validasi status aktif
     → Dipakai: Step 1 detector (SERIAL)

GET  /RencanaKontrol/List/{noKartu}/{tglAwal}/{tglAkhir}
     → List surat kontrol aktif
     → Dipakai: checkKontrol() (filter tgl = today)

GET  /antrean/booking/list/{noKartu}/{tgl}
     → Cek booking Mobile JKN hari ini  [ANTROL API, bukan VClaim]
     → Dipakai: checkMJKN()

POST /antrean/booking/checkin
     → Konfirmasi check-in MJKN

POST /SEP/2.0/insert
     → Buat SEP baru (rujukan pertama, post-RANAP, post-RAJAL)
     Body: { noKartu, tglSep, ppkPelayanan, jnsPelayanan, ... }

POST /SEP/2.0/kontrol/insert
     → Buat SEP Kontrol dari surat kontrol (SKDP)

GET  /Rujukan/noSurat/{noSurat}/{tgl}
     → Validasi nomor rujukan FKTP

GET  /RiwayatPelayanan/{noKartu}/{tglAwal}/{tglAkhir}
     → Riwayat kunjungan untuk checkPostRAJAL
```

### Business Rules VClaim yang Harus Di-Handle

```go
// Error codes VClaim dan mapping ke pesan Indonesia
var vclaimeErrors = map[string]string{
    "0"   : "OK",
    "-2"  : "Pasien tidak ditemukan",
    "-3"  : "Status kepesertaan tidak aktif",
    "-4"  : "Tanggal SEP tidak valid",
    "-5"  : "Nomor rujukan tidak ditemukan atau sudah expired",
    "-6"  : "SEP sudah dibuat untuk tanggal ini (duplikasi)",
    "-7"  : "Hak kelas peserta tidak sesuai dengan kelas yang dipilih",
    "-8"  : "Batas pembuatan SEP harian telah tercapai",
    "-10" : "Fingerprint belum terverifikasi untuk pasien ini",
}
```

## BPJS Antrol (Antrean Online)

```
Base URL: https://apijkn.bpjs-kesehatan.go.id/antrean-rest/

Auth: sama dengan VClaim (X-cons-id, X-timestamp, X-signature)
      Tapi menggunakan credential Antrol terpisah dari VClaim!

POST /antrean/pendaftaran/buat
     → Push nomor antrian poli ke Antrol agar muncul di Mobile JKN
     → Fire-and-forget: error tidak memblokir proses cetak lokal

GET  /antrean/booking/list/{noKartu}/{tgl}
     → Dipakai checkMJKN() di detector

POST /antrean/booking/checkin
     → Konfirmasi hadir setelah pasien konfirmasi di kiosk
```

## Fingerprint BPJS (After.exe)

### Arsitektur Headless (Windows only)

```
Go process
    │
    ├─ spawn After.exe (CREATE_NO_WINDOW flag — tidak terlihat user)
    │
    ├─ inject login via Windows UI Automation:
    │    FindWindow("TfrmLogin") → SetText(username) → SetText(password) → Click(login)
    │
    └─ call REST API lokal After.exe:
         POST https://fp.bpjs-kesehatan.go.id/finger-rest/api/fingerprint
         Body: { "userId": "...", "userPassword": "...", "noPeserta": "..." }
         
         Poll GET /api/fingerprint/status setiap 500ms
         Timeout: 30 detik
         On success: lanjutkan submit SEP dengan token FP
```

### Interface (diimplementasi berbeda di Mac vs Windows)

```go
// internal/hardware/fingerprint/interface.go
type FingerprintVerifier interface {
    // Verify memulai proses verifikasi sidik jari
    // Blocking sampai berhasil, gagal, atau timeout
    Verify(ctx context.Context, noPeserta string) (FPResult, error)
    IsAvailable() bool
}

type FPResult struct {
    Success   bool
    Token     string    // dipakai di SEP payload
    Timestamp time.Time
}
```

## Frista Card Reader

### Arsitektur Auto-Login (Windows only)

```
APM start
    │
    └─ spawn frista.exe HIDDEN
           │
           ├─ [Windows] inject login via SendMessage WM_SETTEXT
           │    atau UI Automation COM API
           │
           └─ subscribe ke stdout JSON pipe / named pipe
                  │
                  └─ saat kartu ditempel:
                       emit JSON: { nik, nama, tglLahir, alamat, noKartu? }
                              │
                              └─ Wails: runtime.EventsEmit(ctx, "frista:card_read", data)
                                         │
                                         └─ Vue: listen → auto-fill form fields
```

### Interface

```go
// internal/hardware/frista/interface.go
type CardReader interface {
    Start(ctx context.Context) error
    Stop() error
    IsAvailable() bool
    // CardRead channel menerima data setiap kali kartu ditempel
    CardRead() <-chan CardData
}

type CardData struct {
    NIK       string
    Nama      string
    TglLahir  time.Time
    Alamat    string
    NoKartu   string  // No peserta JKN jika ada di kartu
}
```

## Cara Mock untuk Development di Mac

```go
// internal/hardware/frista/mock.go
type MockFristaReader struct {
    ch chan CardData
}

func newMockFristaReader() *MockFristaReader {
    m := &MockFristaReader{ch: make(chan CardData, 1)}
    return m
}

// Di dev mode, bisa trigger mock via Wails dev tool atau HTTP endpoint lokal:
// POST http://localhost:9090/mock/card-read
// Body: { "nik": "3271...", "nama": "Test Pasien", ... }
```

Config untuk enable mock endpoint di Mac:
```toml
[dev]
mock_hardware = true
mock_server_port = 9090
```

