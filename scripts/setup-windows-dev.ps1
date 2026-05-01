# ============================================================
# APM — Windows Development Setup (one-click)
# ============================================================
#
# Auto-install semua prerequisite untuk develop APM di Windows:
#   - Go 1.22+, Node 20 LTS, Git, GitHub CLI
#   - Wails CLI v2, MinGW (CGo compiler), WebView2 Runtime
#   - VS Code + Claude Code CLI
#   - Clone repo Arunika-TARA + setup deps
#   - Configure git identity rsamjkt untuk commit author
#
# Run sekali sebagai Administrator. Idempotent — safe re-run.
#
# Usage:
#   PowerShell as Admin:
#     Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass
#     .\setup-windows-dev.ps1
#
#   Atau dengan target folder custom:
#     .\setup-windows-dev.ps1 -TargetDir "D:\Dev\APM"
# ============================================================

param(
    [string]$TargetDir = "$env:USERPROFILE\APM",
    [string]$RepoUrl   = "https://github.com/rsamjkt/Arunika-TARA.git",
    [switch]$SkipClone = $false,
    [switch]$SkipNpm   = $false
)

$ErrorActionPreference = "Stop"

# ============================================================
# Helpers
# ============================================================

function Write-Step($message) {
    Write-Host ""
    Write-Host "==> $message" -ForegroundColor Cyan
}

function Write-Success($message) {
    Write-Host "  ✓ $message" -ForegroundColor Green
}

function Write-Skip($message) {
    Write-Host "  ↷ $message" -ForegroundColor Yellow
}

function Write-Err($message) {
    Write-Host "  ✗ $message" -ForegroundColor Red
}

function Test-Admin {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Test-CommandExists($command) {
    return [bool](Get-Command $command -ErrorAction SilentlyContinue)
}

function Install-WingetPackage($packageId, $name) {
    Write-Step "Installing $name ($packageId)"
    $existing = winget list --id $packageId --exact 2>$null | Select-String $packageId
    if ($existing) {
        Write-Skip "$name sudah terinstall"
        return
    }
    winget install --id $packageId --silent --accept-package-agreements --accept-source-agreements
    if ($LASTEXITCODE -eq 0) {
        Write-Success "$name installed"
    } else {
        Write-Err "$name install gagal (exit $LASTEXITCODE)"
    }
}

function Refresh-Path {
    $env:Path = [Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + `
                [Environment]::GetEnvironmentVariable("Path", "User")
}

# ============================================================
# Pre-checks
# ============================================================

Write-Host ""
Write-Host "  ═══════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  APM Windows Development Setup" -ForegroundColor Cyan
Write-Host "  ═══════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

if (-not (Test-Admin)) {
    Write-Err "Script harus run sebagai Administrator!"
    Write-Host ""
    Write-Host "  Klik kanan PowerShell → 'Run as Administrator', lalu:"
    Write-Host "    Set-ExecutionPolicy -Scope Process -ExecutionPolicy Bypass"
    Write-Host "    .\setup-windows-dev.ps1"
    Write-Host ""
    exit 1
}

if (-not (Test-CommandExists "winget")) {
    Write-Err "winget tidak ada. Install App Installer dari Microsoft Store dulu."
    exit 1
}

Write-Success "Running as Administrator + winget tersedia"

# ============================================================
# 1. Install development tools via winget
# ============================================================

Install-WingetPackage "GoLang.Go" "Go (compiler)"
Install-WingetPackage "OpenJS.NodeJS.LTS" "Node.js LTS"
Install-WingetPackage "Git.Git" "Git"
Install-WingetPackage "GitHub.cli" "GitHub CLI"
Install-WingetPackage "Microsoft.VisualStudioCode" "VS Code"
Install-WingetPackage "Microsoft.EdgeWebView2Runtime" "WebView2 Runtime (Wails)"
Install-WingetPackage "MartinStorsjo.LLVM-MinGW.UCRT" "MinGW (CGo compiler)"

# Refresh PATH supaya tools yang baru di-install bisa di-execute di sesi ini
Refresh-Path

# ============================================================
# 2. Verify versions
# ============================================================

Write-Step "Verify installed versions"

$tools = @{
    "go"   = @{ cmd = "go version"; required = $true }
    "node" = @{ cmd = "node --version"; required = $true }
    "npm"  = @{ cmd = "npm --version"; required = $true }
    "git"  = @{ cmd = "git --version"; required = $true }
    "gh"   = @{ cmd = "gh --version"; required = $false }
}

foreach ($tool in $tools.Keys) {
    if (Test-CommandExists $tool) {
        $version = (Invoke-Expression $tools[$tool].cmd 2>&1) -join " "
        Write-Success "$tool : $version"
    } else {
        if ($tools[$tool].required) {
            Write-Err "$tool tidak ditemukan — buka PowerShell baru dan re-run"
            exit 1
        } else {
            Write-Skip "$tool tidak ada (optional)"
        }
    }
}

# ============================================================
# 3. Install Wails CLI v2
# ============================================================

Write-Step "Install Wails CLI v2"

$goBinPath = "$env:USERPROFILE\go\bin"
if ($env:Path -notlike "*$goBinPath*") {
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$goBinPath", "User")
    $env:Path += ";$goBinPath"
    Write-Success "Added $goBinPath ke PATH"
}

if (Test-CommandExists "wails") {
    $wailsVer = (wails version 2>&1) -join " "
    Write-Skip "Wails sudah terinstall : $wailsVer"
} else {
    Write-Host "  Running: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Wails CLI installed"
    } else {
        Write-Err "Wails install gagal"
        exit 1
    }
}

# Wails doctor — verify all deps OK
Write-Host "  Running: wails doctor"
& wails doctor

# ============================================================
# 4. Install Claude Code CLI
# ============================================================

Write-Step "Install Claude Code CLI"

if (Test-CommandExists "claude") {
    $claudeVer = (claude --version 2>&1) -join " "
    Write-Skip "Claude Code sudah terinstall : $claudeVer"
} else {
    Write-Host "  Running: npm install -g @anthropic-ai/claude-code"
    npm install -g @anthropic-ai/claude-code
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Claude Code installed"
    } else {
        Write-Err "Claude Code install gagal — try manual: npm install -g @anthropic-ai/claude-code"
    }
}

# ============================================================
# 5. Configure git identity
# ============================================================

Write-Step "Configure git identity"

$globalEmail = git config --global user.email 2>$null
$globalName = git config --global user.name 2>$null

if (-not $globalEmail) {
    git config --global user.email "randymandala@gmail.com"
    Write-Success "Set global user.email = randymandala@gmail.com"
} else {
    Write-Skip "user.email already set: $globalEmail"
}

if (-not $globalName) {
    git config --global user.name "Randy"
    Write-Success "Set global user.name = Randy"
} else {
    Write-Skip "user.name already set: $globalName"
}

# Line ending — Windows convert otomatis
git config --global core.autocrlf true
Write-Success "core.autocrlf = true (Windows line endings)"

# Default branch main
git config --global init.defaultBranch main
Write-Success "init.defaultBranch = main"

# ============================================================
# 6. Clone repo (atau pull kalau sudah ada)
# ============================================================

if (-not $SkipClone) {
    Write-Step "Clone / update repo Arunika-TARA"

    if (Test-Path "$TargetDir\.git") {
        Write-Skip "Repo sudah ada di $TargetDir — running git pull"
        Set-Location $TargetDir
        git pull origin main
    } else {
        if (Test-Path $TargetDir) {
            Write-Err "$TargetDir sudah ada tapi bukan git repo — backup dulu atau pilih folder lain"
            exit 1
        }
        Write-Host "  Cloning $RepoUrl ke $TargetDir"
        git clone $RepoUrl $TargetDir
        Set-Location $TargetDir
    }

    # Set repo-local author rsamjkt (untuk commit konsisten dengan history existing)
    git config user.name "rsamjkt"
    git config user.email "randy@rsanggrekmas.com"
    Write-Success "Repo-local git author = rsamjkt <randy@rsanggrekmas.com>"
}

# ============================================================
# 7. Install Go + Node deps
# ============================================================

if (-not $SkipNpm) {
    Write-Step "Download Go modules"
    Set-Location $TargetDir
    go mod download
    if ($LASTEXITCODE -eq 0) {
        Write-Success "Go deps downloaded"
    } else {
        Write-Err "go mod download gagal"
    }

    Write-Step "Install frontend npm deps"
    Set-Location "$TargetDir\frontend"
    npm install
    if ($LASTEXITCODE -eq 0) {
        Write-Success "npm install done"
    } else {
        Write-Err "npm install gagal"
    }
    Set-Location $TargetDir
}

# ============================================================
# 8. Smoke test build
# ============================================================

Write-Step "Smoke test — go build"
Set-Location $TargetDir
go build ./...
if ($LASTEXITCODE -eq 0) {
    Write-Success "go build sukses"
} else {
    Write-Err "go build gagal — cek error di atas"
}

# ============================================================
# 9. Final instructions
# ============================================================

Write-Host ""
Write-Host "  ═══════════════════════════════════════════════" -ForegroundColor Green
Write-Host "  Setup selesai!" -ForegroundColor Green
Write-Host "  ═══════════════════════════════════════════════" -ForegroundColor Green
Write-Host ""
Write-Host "  Repo location : $TargetDir" -ForegroundColor White
Write-Host "  Git author    : rsamjkt <randy@rsanggrekmas.com>" -ForegroundColor White
Write-Host ""
Write-Host "  Next steps:"
Write-Host ""
Write-Host "  1. Buka VS Code di repo:" -ForegroundColor Cyan
Write-Host "       code $TargetDir"
Write-Host ""
Write-Host "  2. Login Claude Code (sekali per mesin):" -ForegroundColor Cyan
Write-Host "       claude"
Write-Host ""
Write-Host "  3. Login GitHub CLI (untuk push tanpa password):" -ForegroundColor Cyan
Write-Host "       gh auth login"
Write-Host ""
Write-Host "  4. Edit config.toml untuk lokal dev:" -ForegroundColor Cyan
Write-Host "       notepad $TargetDir\config.toml"
Write-Host "       # Set [bpjs] cons_id, consumer_secret, user_key, ppk_pelayanan"
Write-Host "       # Set [frista] exe_path, username_enc, password_enc"
Write-Host "       # Set [fingerprint] exe_path, username_enc, password_enc"
Write-Host "       # Untuk dev awal: [bpjs] mock = true"
Write-Host ""
Write-Host "  5. Run dev mode (Wails hot-reload):" -ForegroundColor Cyan
Write-Host "       cd $TargetDir"
Write-Host "       wails dev"
Write-Host ""
Write-Host "  6. Build production binary:" -ForegroundColor Cyan
Write-Host "       wails build -platform windows/amd64"
Write-Host "       # Output: build\bin\apm.exe"
Write-Host ""
Write-Host "  Documentation:"
Write-Host "    docs\AUTO_UPDATE.md   — auto-update + PAT setup"
Write-Host "    docs\WATCHDOG.md      — supervisor + auto-rollback"
Write-Host "    docs\MANUAL_TEST_CHECKLIST.md — verify deployment"
Write-Host ""
