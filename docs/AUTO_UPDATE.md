# Auto-update — Setup & Operasional

APM sejak `v2.2.0-mahatma` punya auto-update via GitHub Releases. Kiosk
cek release baru saat startup, dan admin tap "Update sekarang" di panel
admin untuk install + restart otomatis.

## 1. Generate GitHub PAT

Repo `rsamjkt/Arunika-TARA` private — tanpa token, GitHub API return 404.

**Steps:**

1. Buka https://github.com/settings/tokens?type=beta (Fine-grained tokens)
2. Klik **Generate new token**
3. Form:
   - **Token name**: `APM-kiosk-RSAM-2026` (atau ID kiosk lain)
   - **Expiration**: 1 year (rotate setiap tahun)
   - **Resource owner**: `rsamjkt`
   - **Repository access**: **Only select repositories** → pilih `Arunika-TARA`
   - **Permissions** → **Repository permissions**:
     - **Contents**: **Read-only** ✓
     - Sisanya biarkan No access
4. **Generate token** → copy `github_pat_...` (sekali tampil, simpan!)

## 2. Inject ke `config.toml`

```toml
[update]
enabled               = true
repo                  = "rsamjkt/Arunika-TARA"
github_token          = "github_pat_11ABCDEF...xxxx"   # ← paste disini
check_on_startup      = true
auto_apply            = false                          # production: false
check_interval_hours  = 24
asset_pattern         = "apm-windows-amd64.exe"
```

## 3. Verifikasi

Run kiosk → log harus muncul:

```
INFO update check: tidak ada versi baru current=v2.3.0-mahatma latest=v2.3.0-mahatma
```

Atau kalau ada release baru:

```
INFO update tersedia current=v2.3.0-mahatma latest=v2.3.1-mahatma asset=apm-windows-amd64.exe size_mb=72
```

Frontend admin panel akan tampilkan tile warning **"Update sekarang: v2.3.1-mahatma"**.

## 4. Apply update (manual)

1. Login admin panel (PIN dari `[admin] pin`)
2. Lihat tile **"Update sekarang: vX.Y.Z"** (warning variant amber)
3. Tap → confirm dialog → tap **Yes**
4. Kiosk download asset dari GitHub Releases (~70 MB) — progress modal
5. Atomic replace `apm.exe` → exit → spawn baru detached
6. Kiosk restart dengan versi baru ~5 detik
7. Post-startup health check 30s — kalau Khanza tidak reachable, log
   `update:health-failed` event

## 5. Rollback kalau update broken

**Otomatis (watchdog .bat):**

Kalau pakai `scripts/apm-watchdog.bat` (lihat [WATCHDOG.md](WATCHDOG.md)),
crash post-update otomatis trigger restore dari `./backups/apm-<previous>.exe`.

**Manual via admin panel:**

1. Admin panel → tile **"Rollback ke versi sebelumnya"** (visible kalau ada
   `last-update.json` di folder)
2. Confirm → APM swap binary dengan backup di `./backups/` → restart

## 6. Auto-apply mode (advanced)

Set `auto_apply = true` di `config.toml`. Saat startup detect update
baru:

1. Frontend tampilkan modal full-screen 30s countdown ring
2. User bisa tap **"Tunda update"** untuk cancel (admin tetap bisa
   apply manual nanti)
3. Kalau countdown habis tanpa intervene → auto-download + restart

**Risiko**: kalau release baru broken, kiosk akan boot loop kecuali
ada watchdog. Default: `auto_apply = false`.

## 7. Backup retention

- Setiap apply update, current `apm.exe` di-copy ke `./backups/apm-<version>-<timestamp>.exe`
- Cleanup otomatis: backup > 7 hari di-hapus saat startup berikutnya
- Manual cleanup: hapus folder `./backups/` (kalau yakin tidak butuh rollback)

## 8. CI/CD release flow

Tag `v*` → GitHub Actions `release.yml` → build mac universal +
Windows amd64 + create release page dengan asset:

- `apm-mac-universal.app.zip`
- `apm-windows-amd64.exe`

Pastikan `asset_pattern` di config match nama asset di release page.

## 9. Token rotation

PAT expire 1 tahun. Set kalender reminder. Kalau token expire:
- Kiosk log akan muncul `github token invalid / expired (401)`
- Admin panel tile "Cek update" akan return error
- Generate PAT baru → update `config.toml` → restart kiosk

## Troubleshooting

| Symptom | Penyebab | Fix |
|---|---|---|
| `404` di log update check | Token tidak punya akses ke repo | Verify token scope: Contents read-only ke `rsamjkt/Arunika-TARA` |
| `401` | Token expired | Generate PAT baru |
| Tile "Update sekarang" tidak muncul | `cfg.update.enabled = false` atau token salah | Cek log + config |
| Update apply hang di 99% | File lock — antivirus scan | Whitelist `apm.exe` di Defender |
| Boot loop post-update | Update broken + tidak ada watchdog | Manual: SSH/RDP ke kiosk, copy `./backups/apm-*.exe` ke `apm.exe`, restart |
