# Manual Test Checklist — APM (T.A.R.A)

Runbook P-060 langkah 3-5: visual + offline test setelah `wails build`
sukses. Untuk LANGKAH 1 (test) dan LANGKAH 6 (lint) sudah otomatis di
CI / `make test` + `make lint` (lihat ringkasan di bawah).

## Prasyarat

```bash
# Install Node.js 18+ (kalau belum):
brew install node          # macOS
# atau download installer dari https://nodejs.org

# Verifikasi
node --version             # >= v18.0.0
npm --version

# Install frontend dependencies + cek wails
cd frontend && npm install && cd ..
wails doctor               # harus all-green
```

## LANGKAH 1 — Test (otomatis)

```bash
make test                  # all packages, race-clean
make test-coverage         # generate coverage.html
```

Ekspektasi:
- 15 package PASS
- Total coverage ≥70%
- `service/detector` ≥99%, `service/sep` ≥84%
- Buka `coverage.html` di browser untuk inspeksi per-line.

## LANGKAH 2 — Build macOS

```bash
make build-mac
```

Output: `build/bin/APM.app` (atau `apm-go` binary di Linux).

## LANGKAH 3 — Window basic test

```bash
open build/bin/APM.app
# atau:
wails dev    # untuk hot reload (dev mode)
```

Checklist:
- [ ] Window muncul dengan judul **"Anjungan Pasien Mandiri"**
- [ ] HomeScreen render: header (logo "T" biru + "RS Anggrek Mas" + status dots + jam digital), 4 button area, footer
- [ ] Status dots awal: BPJS hijau / Sistem hijau (kalau backend mock OK)
- [ ] Jam digital update tiap detik
- [ ] Resize window 800×600 → 1920×1080 — semua elemen scale mulus, tidak overlap

## LANGKAH 4 — Flow BPJS happy path

Checklist:
- [ ] Klik **"Pasien BPJS"** → navigate ke InputScreen, mode "bpjs"
- [ ] InputScreen: NumPad muncul, FristaBar status hijau "Frista aktif"
- [ ] Ketik 6+ angka di numpad → muncul format-grouped di display dengan blink cursor
- [ ] Tombol "Cari" disabled saat <6 angka, enabled setelah
- [ ] Tombol "Hapus" delete digit terakhir
- [ ] Tombol "Cari" → DetectScreen
- [ ] DetectScreen: ProgressRing animasi rotate, 5 step list animate sequential
- [ ] Auto-navigate ke ResultScreen setelah ~3-5 detik
- [ ] ResultScreen: PatientCard dengan pill warna sesuai kategori, info bar, CTA primary, ghost button
- [ ] Test 4 kategori (mock backend response):
  - [ ] MJKN: pill hijau, CTA "Konfirmasi kedatangan"
  - [ ] Kontrol: pill biru, DokterPicker visible, CTA "Buat surat layanan"
  - [ ] RujukanBaru: pill kuning, info biometrik (kalau dewasa non-IGD)
  - [ ] TidakAktif: pill merah, ghost "Hubungi petugas"

## LANGKAH 5 — Mock card-read (Frista)

Buka terminal lain saat APM dev mode jalan:

```bash
make mock-card-default
# atau dengan custom NIK/NAMA/KARTU:
make mock-card-read NIK=3271234567890001 NAMA="Test Pasien" KARTU=0001234567890012
# atau dengan delay:
make mock-card-delay SECONDS=3
```

Checklist:
- [ ] Frista mock server up di `http://localhost:9090` (cek browser)
- [ ] POST card-read → di HomeScreen / InputScreen, auto-navigate ke `/detect` dengan input ter-isi
- [ ] Form patient store ter-update dengan NIK + Nama + No Kartu

## LANGKAH 6 — Antrian flow

Checklist:
- [ ] HomeScreen → klik **"Ambil Antrian"** → AntrianScreen
- [ ] 5 card terlihat: Admisi appointment, Admisi walk-in, Rawat inap & IGD, Farmasi, CS
- [ ] Counter "Sekarang: A-XXX" ter-update setiap 30 detik
- [ ] Tap salah satu card → loading spinner di card, lalu navigate ke TicketScreen
- [ ] TicketScreen: CheckCircle hijau, "Nomor antrian berhasil diambil", paper dengan nomor besar
- [ ] Countdown 10 detik di bawah, auto-back ke Home
- [ ] Klik "Cetak ulang tiket" → spinner, output muncul di terminal (ConsolePrinter)

## LANGKAH 7 — Idle timeout

Checklist:
- [ ] Di HomeScreen, tunggu 50 detik tanpa interaksi
- [ ] Di detik ke-50 (10 detik sebelum timeout): IdleOverlay muncul dengan countdown
- [ ] Sentuh layar → overlay hilang, timer reset ke 60 detik
- [ ] Tunggu hingga 0 → reset ke HomeScreen + clear patient store

## LANGKAH 8 — Offline mode

Cara test (Mac):
```bash
# Matikan wifi via menubar, atau:
sudo ifconfig en0 down
```

Atau lebih realistis: matikan Khanza mock server. Karena kita belum punya
Khanza mock server jalan di test, alternatif:
1. Set `khanza_url` di `config.toml` ke URL yang tidak reachable (mis. `http://127.0.0.1:1`).
2. Restart APM.

Checklist:
- [ ] Banner kuning **"Mode offline — antrian disimpan sementara"** muncul di atas semua screen (slide-down transition)
- [ ] Status dot BPJS / Sistem berubah merah di header HomeScreen
- [ ] Klik **"Ambil Antrian"** → tetap berhasil (offline fallback)
- [ ] TicketScreen: badge **"OFFLINE — akan disinkronkan"** muncul di paper
- [ ] Restore koneksi (wifi on / config benar) → setelah max 30s, banner hilang dengan slide-up transition
- [ ] Cek `data/apm.db` — tabel `antrian_lokal` masih ada record dengan `sync_status='pending'` sampai reconcile sukses
- [ ] Setelah online, cek `reconcile_log` — ada entry `SYNC_SUCCESS` per record yang ter-sync

## LANGKAH 9 — Admin Panel

Cara akses (sementara — production akan ada deep link):
- Manual edit URL di webview dev tools: `/admin`
- Atau tambahkan keyboard shortcut di App.vue (TODO P-052)

Checklist:
- [ ] Pin gate muncul: 6-dot display + numpad
- [ ] PIN salah (mis. 9999) → shake animation, dot merah
- [ ] PIN benar (default `1234` di config.toml) → unlock
- [ ] Stat grid 2x2 menampilkan angka real-time
- [ ] Status komponen 6 item dengan pill warna sesuai realitas
- [ ] Pending SEP table (kalau ada) — Confirm button → confirmation modal → status berubah ke `awaiting_sync`
- [ ] Action: Reset counter → confirmation modal → success modal
- [ ] Action: Lihat log rekonsiliasi → modal dengan table 50 entry
- [ ] Action: Test cetak printer → success modal, output muncul di terminal
- [ ] Action: Info mock server (visible kalau platform ≠ windows) → modal dengan endpoint info

## LANGKAH 10 — Lint (otomatis)

```bash
make lint
```

Setelah P-060 fixes:
- [ ] 0 issues di production code
- [ ] ~27 errcheck warnings di `_test.go` (acceptable — test code best-effort)

## Hasil

Kalau semua checklist ✅, system siap untuk **P-061 cross-compile Windows**.

Kalau ada checklist yang gagal:
1. Catat di GitHub Issue
2. Reproduce dengan `wails dev` + Chrome DevTools
3. Lihat `logs/apm.log` (Go side — sudah ter-PHI-mask)
4. Buka [coverage.html](../coverage.html) untuk cek apakah path yang fail di-cover test
