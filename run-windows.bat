@echo off
REM Launcher APM/T.A.R.A untuk Windows production deployment.
REM
REM Penempatan: copy file ini ke folder yang sama dengan apm.exe + config.toml.
REM Layout final:
REM   APM\
REM   ├── run-windows.bat       (file ini — double-click untuk jalankan)
REM   ├── apm.exe
REM   ├── config.toml
REM   └── migrations\
REM
REM Usage:
REM   run-windows.bat              jalankan kiosk
REM   run-windows.bat --encrypt-config    encrypt credential di config.toml

setlocal
cd /d "%~dp0"

if not exist "apm.exe" (
    echo ERROR: apm.exe tidak ketemu di %CD%
    echo Pastikan apm.exe ada di folder ini.
    pause
    exit /b 1
)

if not exist "config.toml" (
    echo ERROR: config.toml tidak ada di %CD%
    echo Copy dari config.example.toml lalu isi sesuai environment.
    pause
    exit /b 1
)

if not exist "migrations\001_initial.sql" (
    echo ERROR: folder migrations\ tidak ada di %CD%
    pause
    exit /b 1
)

set APM_CONFIG_PATH=.\config.toml

REM Jalankan tanpa console window kalau di-double-click
if "%1"=="" (
    start "" "%CD%\apm.exe"
) else (
    apm.exe %*
)

endlocal
