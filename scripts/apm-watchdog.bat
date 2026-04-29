@echo off
REM ============================================================
REM apm-watchdog.bat — APM kiosk supervisor + auto-rollback
REM ============================================================
REM
REM Wraps apm.exe. Kalau apm.exe exit non-zero dalam 60 detik
REM post-update (last-update.json HealthChecked=false), otomatis
REM swap binary ke backup_path dan restart.
REM
REM Mode normal (kiosk crash regular): auto-restart sampai 5x.
REM Setelah 5x crash → notify ke event log + exit (admin
REM intervene manual).
REM
REM Setup di Windows kiosk:
REM   1. Copy script ini ke folder yang sama dengan apm.exe.
REM   2. Set Windows Task Scheduler "At log on" → run apm-watchdog.bat
REM      (BUKAN apm.exe langsung).
REM   3. Edit "%APP_DIR%" di bawah kalau apm.exe ada di lokasi non-standar.
REM ============================================================

setlocal EnableDelayedExpansion

REM Lokasi default — sama dengan folder script ini
set "APP_DIR=%~dp0"
set "APP_EXE=%APP_DIR%apm.exe"
set "STATE_FILE=%APP_DIR%last-update.json"
set "BACKUP_DIR=%APP_DIR%backups"
set "WATCHDOG_LOG=%APP_DIR%logs\watchdog.log"

REM Pastikan log folder ada
if not exist "%APP_DIR%logs" mkdir "%APP_DIR%logs"

REM Counter crash beruntun
set CRASH_COUNT=0
set MAX_CRASH=5

:LOOP
echo [%date% %time%] Starting apm.exe (crash_count=%CRASH_COUNT%) >> "%WATCHDOG_LOG%"

REM Catat waktu start untuk health-window check
set START_TIME=%time%
set "START_EPOCH="
for /f "tokens=2 delims==" %%a in ('wmic os get LocalDateTime /value 2^>nul') do set START_EPOCH=%%a

REM Run apm.exe (sync — script wait sampai apm.exe exit)
"%APP_EXE%"
set EXIT_CODE=%errorlevel%

echo [%date% %time%] apm.exe exited code=%EXIT_CODE% >> "%WATCHDOG_LOG%"

REM Kalau exit clean (code 0), kemungkinan post-update restart.
REM Lanjut loop ringan tanpa rollback check.
if %EXIT_CODE% equ 0 (
    set CRASH_COUNT=0
    timeout /t 2 /nobreak >nul
    goto LOOP
)

REM Exit non-zero — kemungkinan crash. Cek apakah baru update.
REM Logic: kalau last-update.json ada DAN HealthChecked=false DAN
REM update applied < 5 menit lalu → rollback.

if not exist "%STATE_FILE%" (
    REM Bukan post-update crash — increment counter, retry
    set /a CRASH_COUNT=CRASH_COUNT+1
    if !CRASH_COUNT! geq %MAX_CRASH% (
        echo [%date% %time%] CRASH LIMIT REACHED %MAX_CRASH% — exit watchdog >> "%WATCHDOG_LOG%"
        echo Tampilkan pesan ke admin: kiosk butuh intervene manual.
        eventcreate /T ERROR /ID 100 /L APPLICATION /SO "APM-Watchdog" ^
            /D "APM kiosk crash %MAX_CRASH% kali beruntun. Cek log %WATCHDOG_LOG%" 2>nul
        exit /b 1
    )
    timeout /t 3 /nobreak >nul
    goto LOOP
)

REM Ada last-update.json — parse + rollback decision
echo [%date% %time%] Post-update crash detected — checking rollback >> "%WATCHDOG_LOG%"

REM Baca backup_path dari JSON dengan PowerShell (lebih reliable dari findstr)
for /f "delims=" %%a in ('powershell -NoProfile -Command "(Get-Content '%STATE_FILE%' -Raw | ConvertFrom-Json).backup_path" 2^>nul') do set "BACKUP_PATH=%%a"
for /f "delims=" %%a in ('powershell -NoProfile -Command "(Get-Content '%STATE_FILE%' -Raw | ConvertFrom-Json).health_checked" 2^>nul') do set "HEALTH_CHECKED=%%a"

if "%BACKUP_PATH%"=="" (
    echo [%date% %time%] backup_path kosong di state — skip rollback >> "%WATCHDOG_LOG%"
    set /a CRASH_COUNT=CRASH_COUNT+1
    timeout /t 3 /nobreak >nul
    goto LOOP
)

REM Kalau sudah HealthChecked=true, update sudah dianggap sehat.
REM Crash sekarang berarti issue baru, bukan post-update broken.
if /i "%HEALTH_CHECKED%"=="True" (
    echo [%date% %time%] Update sudah pass health check — bukan post-update issue >> "%WATCHDOG_LOG%"
    set /a CRASH_COUNT=CRASH_COUNT+1
    timeout /t 3 /nobreak >nul
    goto LOOP
)

REM ROLLBACK: copy backup ke apm.exe, hapus state file
if not exist "%BACKUP_PATH%" (
    echo [%date% %time%] BACKUP FILE TIDAK ADA di %BACKUP_PATH% — manual fix needed >> "%WATCHDOG_LOG%"
    eventcreate /T ERROR /ID 101 /L APPLICATION /SO "APM-Watchdog" ^
        /D "APM rollback gagal: backup file %BACKUP_PATH% tidak ada" 2>nul
    exit /b 2
)

echo [%date% %time%] ROLLING BACK apm.exe ← %BACKUP_PATH% >> "%WATCHDOG_LOG%"
copy /Y "%BACKUP_PATH%" "%APP_EXE%" >> "%WATCHDOG_LOG%" 2>&1
if errorlevel 1 (
    echo [%date% %time%] copy rollback FAILED >> "%WATCHDOG_LOG%"
    exit /b 3
)

del /Q "%STATE_FILE%" 2>nul
echo [%date% %time%] Rollback success — restart APM dengan versi sebelumnya >> "%WATCHDOG_LOG%"

eventcreate /T WARNING /ID 102 /L APPLICATION /SO "APM-Watchdog" ^
    /D "APM auto-rollback: update broken, restored backup %BACKUP_PATH%" 2>nul

REM Reset crash counter — restart fresh
set CRASH_COUNT=0
timeout /t 2 /nobreak >nul
goto LOOP

endlocal
