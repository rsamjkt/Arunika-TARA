# APM Watchdog — Auto-Rollback

`scripts/apm-watchdog.bat` adalah supervisor untuk `apm.exe` di Windows
kiosk. Tujuannya:

1. **Auto-restart** kalau apm.exe crash regular (bukan post-update).
2. **Auto-rollback** kalau apm.exe crash dalam 60 detik post-update —
   restore backup binary dari `./backups/`.
3. **Bail-out** kalau crash 5x beruntun (admin intervene manual).

## Setup

### 1. Copy script ke folder kiosk

Letakkan `apm-watchdog.bat` di **folder yang sama dengan `apm.exe`**:

```
C:\APM\
├── apm.exe
├── apm-watchdog.bat   ← disini
├── config.toml
├── last-update.json   (created post-update)
├── backups\
│   └── apm-v2.3.0-mahatma-20260429-103000.exe
└── logs\
    └── watchdog.log
```

### 2. Configure Windows Task Scheduler

Daripada autostart `apm.exe` langsung, autostart `apm-watchdog.bat`:

1. Buka **Task Scheduler** (Windows + R → `taskschd.msc`)
2. **Create Task** (bukan Basic Task)
3. **General**:
   - Name: `APM Kiosk`
   - "Run whether user is logged on or not" → check
   - "Run with highest privileges" → check
4. **Triggers**: At log on (atau At startup)
5. **Actions**:
   - Action: Start a program
   - Program: `C:\APM\apm-watchdog.bat`
   - Start in: `C:\APM`
6. **Conditions**: uncheck "Start only if computer is on AC power" (kiosk mostly on AC tapi safety)
7. **Settings**: 
   - "Allow task to be run on demand" → check
   - "If the running task does not end when requested, force it to stop" → check

### 3. Test manual

Buka cmd di `C:\APM\` lalu:

```cmd
apm-watchdog.bat
```

Logs ke `logs\watchdog.log`. Stop pakai Ctrl+C.

## Cara kerja rollback

**Skenario**: APM update dari `v2.3.0` ke `v2.3.1`. v2.3.1 crash di startup karena bug (mis. config field baru null pointer).

**Tanpa watchdog**: kiosk boot → crash → kiosk boot → crash (loop). Admin harus
RDP/USB akses kiosk untuk fix manual.

**Dengan watchdog**:

1. v2.3.1 spawn → crash dalam < 60s
2. Watchdog detect exit code != 0
3. Watchdog baca `last-update.json`:
   ```json
   {
     "previous_version": "v2.3.0-mahatma",
     "new_version": "v2.3.1-mahatma",
     "backup_path": "C:\\APM\\backups\\apm-v2.3.0-mahatma-20260429-103000.exe",
     "applied_at": "2026-04-29T10:30:00Z",
     "health_checked": false
   }
   ```
4. `health_checked = false` → rollback decision
5. Watchdog `copy /Y backup_path apm.exe` → kiosk apm.exe sekarang v2.3.0 lagi
6. Hapus `last-update.json`
7. Spawn `apm.exe` → boot OK
8. Windows Event Log entry "APM-Watchdog" warning:
   `APM auto-rollback: update broken, restored backup ...`

**Setelah rollback**:
- Admin lihat di kiosk panel: versi balik ke v2.3.0
- Tile "Cek update GitHub" → akan tampilkan v2.3.1 lagi sebagai available
- Admin **JANGAN** apply ulang — alert ke developer dulu

## Kapan health_checked = true?

APM `runPostUpdateHealthCheck` (lihat `app.go`) jalan 30 detik post-startup
saat `last-update.json.applied_at < 10 menit lalu`. Cek reconciler online
state. Sukses → `MarkHealthy()` set field `health_checked = true`.

Setelah ini, kalau crash, watchdog **tidak akan rollback** — assume
crash baru bukan related to update.

## Limit retry & manual intervene

Watchdog reset `crash_count` setiap exit code 0 (clean exit, mis.
post-update restart). Counter increment cuma untuk crash beruntun
non-post-update.

Setelah 5x crash beruntun, watchdog exit dengan event log:
```
APM kiosk crash 5 kali beruntun. Cek log C:\APM\logs\watchdog.log
```

Admin perlu cek manual:
1. Buka `C:\APM\logs\watchdog.log` + `C:\APM\logs\app.log`
2. Diagnose — config error? hardware unplugged? RDB credential expired?
3. Restart watchdog manual via Task Scheduler atau `apm-watchdog.bat`

## Troubleshooting

| Symptom | Penyebab | Fix |
|---|---|---|
| Watchdog stuck di `:LOOP` tapi apm.exe tidak spawn | Path `APP_EXE` salah | Edit `set "APP_EXE="` di top file |
| Rollback gagal — backup file tidak ada | Backup dihapus manual / cleanup terlalu agresif | Restore manual dari ZIP backup yang lebih lama |
| Event log tidak muncul | `eventcreate` butuh admin privilege | Task Scheduler "Run with highest privileges" |
| PowerShell parse JSON gagal | `last-update.json` corrupt | Hapus file, watchdog akan treat as no-update crash |

## Disable auto-rollback

Kalau lo prefer manual handling (mis. dev environment), edit batch:

```bat
REM Skip rollback section
goto LOOP
```

Setelah baris `if not exist "%STATE_FILE%"`. Watchdog tetap auto-restart
tapi tidak swap binary.
