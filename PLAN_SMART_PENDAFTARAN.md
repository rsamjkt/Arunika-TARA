# PLAN — APM Smart Pendaftaran (T.A.R.A)

> **Sumber**: gap antara MVP direct-DB Khanza yang sudah jalan vs spec ideal,
> di-grounding dari repo referensi `github.com/RS-INDRIATI/anjunganmandiriSEP`
> dan schema RS `sikrsam260312` (sudah dipetakan).

## Ringkasan

MVP Pasien Umum sudah berhasil INSERT ke `reg_periksa` end-to-end (smoke test
write+rollback PASS). Tapi 4 field penting masih placeholder kosong / hardcode.
Untuk Pasien BPJS, Smart Detector framework sudah ada di Go (enum PatientType
6 kategori) tapi belum di-wire ke direct-DB Khanza dan belum auto-resolve
priority.

## Gap saat ini

### Pasien Umum (`MySQLClient.BuatPendaftaran`)

| Kolom        | Implementasi sekarang | Seharusnya (per `DlgRegistrasiWalkIn.java`) |
|--------------|-----------------------|---------------------------------------------|
| `p_jawab`    | `req.Catatan` ❌       | `pasien.namakeluarga`                       |
| `almt_pj`    | `""` ❌                | `pasien.alamat + kel.nm_kel + kec.nm_kec + kab.nm_kab + prop.nm_prop` |
| `hubunganpj` | `""` ❌                | `pasien.keluarga`                           |
| `biaya_reg`  | `0` ❌                 | `poliklinik.registrasilama` (lama) / `poliklinik.registrasi` (baru) |
| `umurdaftar` + `sttsumur` | hanya tahun → "Th" ⚠️ | smart unit: Th>0 else Bl>0 else Hr |
| `status_poli` | sama dgn `stts_daftar` ❌ | per-poli: `COUNT WHERE no_rkm_medis=? AND kd_poli=?` |

### Pasien BPJS — Smart Detector

- Enum `domain.PatientType{Unknown,MJKN,Kontrol,PostRANAP,PostRAJAL,RujukanBaru,TidakAktif,Error}` sudah ada
- Service `internal/service/detector/` belum di-audit untuk integrasi direct-DB Khanza
- Repo referensi tidak punya auto-detect — user pilih manual via tombol. Kita bikin lebih SMART: auto-classify + priority resolution.

## Schema RS `sikrsam260312` yang dikonfirmasi siap

| Tabel | Tujuan |
|---|---|
| `bridging_surat_kontrol_bpjs` | SKDP / kontrol BPJS (+ field PRB lengkap) |
| `bridging_rujukan_bpjs` | Rujukan FKTP keluar |
| `rujuk_masuk` | Rujukan FKTP masuk yang link ke `no_rawat` |
| `bpjs_prb` | Kode PRB hasil SEP |
| `booking_registrasi` | Booking online + Mobile JKN (status='Terdaftar'/'Belum') |
| `maping_poli_bpjs` | Translasi `kd_poli_rs` ↔ `kd_poli_bpjs` |
| `maping_dokter_dpjpvclaim` | Translasi `kd_dokter` ↔ `kd_dokter_bpjs` |
| `kelurahan` / `kecamatan` / `kabupaten` / `propinsi` | Alamat lengkap |
| `rujukan_internal_poli` | Post-RAJAL antar-poli |
| `kamar_inap` + `dpjp_ranap` | Post-RANAP |
| `aruni_surat_kontrol` | Kontrol custom RS (Aruni) |

Tidak tersedia di RS ini:
- `rencana_kontrol` — pakai `bridging_surat_kontrol_bpjs` saja
- `pasien_satusehat` — IHS Number tidak punya storage di Khanza (RS ini); akan disimpan di SQLite lokal kalau perlu
- `flagging_pasien_satusehat` ada (enum yes/no) — sudah dipakai

## Eksekusi — fase per fase

### Phase A — Pasien Umum lengkap (small, isolated)

1. **`EnrichPasien(noRM)`** — single query JOIN ke `kelurahan/kecamatan/kabupaten/propinsi`, compute umur 3 unit (Th/Bl/Hr), ambil `namakeluarga` + `keluarga` + alamat raw. Return `domain.PasienEnriched`.
2. **`GetTarifPoli(kdPoli, isLama)`** — query `poliklinik.registrasi` vs `registrasilama`.
3. **Refactor `BuatPendaftaran`** — pakai keduanya, isi 19 kolom benar. Tetap support optional override via `PendaftaranRequest{PJawab,AlmtPJ,HubunganPJ}`.
4. **Smoke test write** — verify all 19 cols match expected (biaya real, p_jawab real, alamat real).

### Phase B — Smart BPJS Detector (parallel + priority)

1. **Audit existing `internal/service/detector`** — dokumentasikan apa yang sudah ada.
2. **Probe methods di MySQLClient**:
   - `GetBookingMJKN(noRM, tgl)` → query `booking_registrasi`
   - `GetSuratKontrolBPJS(noRM)` → query `bridging_surat_kontrol_bpjs` JOIN bridging_sep WHERE noRM
   - `GetPostRANAPInfo(noRM, days)` → query `kamar_inap` + `dpjp_ranap`
   - `GetRujukanInternalAntarPoli(noRM, days)` → query `rujukan_internal_poli`
   - `MapKdPoliBPJS(kdRS)`, `MapKdDokterBPJS(kdRS)` — translate
3. **Wire ke detector** — parallel fire 5 probe + VClaim peserta check, resolve priority:
   ```
   MJKN > Kontrol > PostRANAP > PostRAJAL > RujukanBaru > KunjunganLanjutan > TidakAktif
   ```
4. **Smart rules**:
   - Date-window: kontrol ≤30 hari, post-RANAP ≤7 hari
   - Multi-match: MJKN menang dari Kontrol kalau keduanya match
   - Fallback chain: VClaim down → tetap bisa lanjut sebagai DB-lokal kategori
   - Auto-pick poli Post-RANAP dari DPJP riwayat

### Phase C — BuatPendaftaran versi BPJS lengkap

1. **`SimpanSEP` lengkap** — 52 kolom `bridging_sep` (rujukan, kls rawat, COB, asal_rujukan, kddpjp, dll) — bukan minimal 11 kolom seperti sekarang.
2. **`SimpanRujukMasuk`** — auto-link rujukan VClaim ke `rujuk_masuk` setelah BuatPendaftaran.
3. **`SimpanPRB`** — kalau ada kode PRB di SEP → insert `bpjs_prb`.
4. Map kd_poli & kd_dokter via `maping_poli_bpjs` / `maping_dokter_dpjpvclaim` saat input dari VClaim balik.

### Phase D — Frontend per-pathway

`ResultScreen.vue` saat ini cuma render generic + Kontrol. Tambah render-by-pathway:
- **MJKN**: card konfirmasi booking + tombol "Konfirmasi kedatangan"
- **Kontrol**: SKDP detail (no_surat, tgl_rencana, poli, dpjp) + DokterPicker default dpjp dari SKDP + tombol "Buat SEP Kontrol"
- **Post-RANAP**: badge "Pasca RI" + auto-pilih poli sesuai DPJP RANAP + DokterPicker
- **Post-RAJAL**: tampilkan rujukan internal antar-poli + auto-pre-fill poli tujuan
- **Rujukan FKTP**: tampilkan rujukan VClaim + DokterPicker poli tujuan
- **Tidak Aktif**: alert + opsi "Daftar sebagai Pasien Umum"

### Phase E — Testing & Validation

- Unit test EnrichPasien dengan fixtures
- Smoke test write end-to-end: BuatPendaftaran lengkap → SELECT verifikasi
- Smoke test Smart Detector: pasien dengan booking, post-RANAP, SKDP

## Urutan eksekusi (dengan paralelisasi)

| Wave | Aktivitas | Owner |
|---|---|---|
| 1 (parallel) | Audit detector | Agent Explore |
| 1 (parallel) | Phase A — Pasien Umum lengkap | Agent general-purpose |
| 1 (parallel) | Phase D — Vue per-pathway scaffold | Agent general-purpose |
| 2 (sequential, after Wave 1) | Phase B — wire smart detector ke direct-DB | Main session |
| 3 (sequential) | Phase C — SEP/Rujukan/PRB lengkap | Main session |
| 4 (final) | Phase E — smoke test all flows | Main session |

## Konstanta yang sudah dikonfirmasi (RS `sikrsam260312`)

```go
const (
    KdPjUmum  = "A03"  // UMUM / TUNAI
    KdPjBPJS  = "BPJ"
)
// no_rawat format: "YYYY/MM/DD/NNNNNN" (17 char)
// no_reg format: 3-digit per (kd_poli, tgl)
// hari_kerja enum: SENIN/SELASA/RABU/KAMIS/JUMAT/SABTU/AKHAD
```
