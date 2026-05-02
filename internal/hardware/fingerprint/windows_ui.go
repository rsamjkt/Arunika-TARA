//go:build windows

// UIAutomation helper untuk After.exe (WPF app - tidak bisa pakai
// Win32 WM_SETTEXT biasa). Semua interaksi lewat PowerShell yang
// memanggil System.Windows.Automation COM API.
//
// AutomationId sudah diverifikasi via Spy++ UIAutomation di kiosk:
//   aj = username field (login panel)
//   ak = password field (login panel)
//   al = Login button
//   ao = radio "No. Kartu BPJS Kesehatan"
//   ar = input noKartu/NIK (main panel)

package fingerprint

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// psUIAScript adalah PowerShell script yang:
//  1. Tunggu window After.exe muncul (max 15 detik)
//  2. Cek apakah sudah login atau masih di form login
//  3. Kalau belum login: inject username+password, klik Login
//  4. Tunggu main panel (id=ar) siap
//  5. Select radio noKartu (id=ao), set value noPeserta
// Output: "OK" kalau sukses, error message kalau gagal.
const psUIAScript = `
param($Username, $Password, $NoPeserta)
Add-Type -AssemblyName UIAutomationClient,UIAutomationTypes,System.Windows.Forms

$root = [System.Windows.Automation.AutomationElement]::RootElement

function Find-ById($id) {
    $cond = New-Object System.Windows.Automation.PropertyCondition(
        [System.Windows.Automation.AutomationElement]::AutomationIdProperty, $id)
    return $root.FindFirst(
        [System.Windows.Automation.TreeScope]::Descendants, $cond)
}

function Set-Value($el, $val) {
    try {
        $vp = $el.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern)
        $vp.SetValue($val)
        return $true
    } catch { return $false }
}

# Set-ValueViaClipboard: untuk WPF PasswordBox yang tidak support ValuePattern
function Set-ValueViaClipboard($el, $val) {
    try { $el.SetFocus() } catch {}
    Start-Sleep -Milliseconds 100
    [System.Windows.Forms.Clipboard]::SetText($val)
    [System.Windows.Forms.SendKeys]::SendWait('^a')
    [System.Windows.Forms.SendKeys]::SendWait('^v')
    Start-Sleep -Milliseconds 100
}

function Click-El($el) {
    try {
        $ip = $el.GetCurrentPattern([System.Windows.Automation.InvokePattern]::Pattern)
        $ip.Invoke()
        return $true
    } catch { return $false }
}

function Select-Radio($el) {
    try {
        $sp = $el.GetCurrentPattern([System.Windows.Automation.SelectionItemPattern]::Pattern)
        $sp.Select()
        return $true
    } catch { return Click-El $el }
}

# Tunggu window After.exe muncul (max 20 detik)
$deadline = (Get-Date).AddSeconds(20)
$afterWin = $null
while ((Get-Date) -lt $deadline) {
    $nameCond = New-Object System.Windows.Automation.PropertyCondition(
        [System.Windows.Automation.AutomationElement]::NameProperty,
        "Aplikasi Registrasi Sidik Jari")
    $afterWin = $root.FindFirst([System.Windows.Automation.TreeScope]::Children, $nameCond)
    if ($afterWin) { break }
    Start-Sleep -Milliseconds 300
}
if (-not $afterWin) {
    Write-Error "Window 'Aplikasi Registrasi Sidik Jari' tidak muncul"
    exit 1
}

# Deteksi status login: id='a3' ada = sudah login (menampilkan "Username : xxx")
$loginStatusEl = Find-ById 'a3'
$needLogin = ($loginStatusEl -eq $null)

if ($needLogin) {
    $userEl = Find-ById 'aj'
    $passEl = Find-ById 'ak'
    $loginBtn = Find-ById 'al'
    if (-not $userEl -or -not $passEl -or -not $loginBtn) {
        Write-Error "Form login tidak ditemukan (id aj/ak/al)"
        exit 1
    }

    # Username: ValuePattern
    Set-Value $userEl $Username | Out-Null

    # Password: WPF PasswordBox butuh clipboard (tidak support ValuePattern)
    Set-ValueViaClipboard $passEl $Password

    # Klik Login
    Click-El $loginBtn | Out-Null

    # Tunggu login selesai: id='a3' muncul (max 15 detik - perlu koneksi BPJS server)
    $loginDeadline = (Get-Date).AddSeconds(15)
    $loggedIn = $false
    while ((Get-Date) -lt $loginDeadline) {
        $loginStatusEl = Find-ById 'a3'
        if ($loginStatusEl) { $loggedIn = $true; break }
        Start-Sleep -Milliseconds 500
    }
    if (-not $loggedIn) {
        Write-Error "Login gagal - id=a3 (username label) tidak muncul dalam 15 detik"
        exit 1
    }
}

# Select radio "No. Kartu BPJS Kesehatan" (id=ao)
$radioEl = Find-ById 'ao'
if ($radioEl) { Select-Radio $radioEl | Out-Null }
Start-Sleep -Milliseconds 300

# Set noPeserta ke input field (id=ar)
$noKartuEl = Find-ById 'ar'
if (-not $noKartuEl) {
    Write-Error "Field noKartu (id=ar) tidak ditemukan"
    exit 1
}
$ok = Set-Value $noKartuEl $NoPeserta
if (-not $ok) {
    # Fallback: clipboard paste
    Set-ValueViaClipboard $noKartuEl $NoPeserta
}

Write-Host "OK"
`

// injectAfterUI menulis script ke temp file lalu eksekusi dengan -File
// supaya param() block menerima argument dengan benar.
func injectAfterUI(username, password, noPeserta string) error {
	tmp, err := os.CreateTemp("", "after-uia-*.ps1")
	if err != nil {
		return fmt.Errorf("buat temp script: %w", err)
	}
	defer os.Remove(tmp.Name())
	// UTF-8 BOM supaya PowerShell 5.1 baca file sebagai UTF-8
	if _, err := tmp.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return fmt.Errorf("tulis BOM: %w", err)
	}
	if _, err := tmp.WriteString(psUIAScript); err != nil {
		return fmt.Errorf("tulis temp script: %w", err)
	}
	tmp.Close()

	cmd := exec.Command("powershell",
		"-NonInteractive", "-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-File", tmp.Name(),
		"-Username", username,
		"-Password", password,
		"-NoPeserta", noPeserta,
	)
	out, err := cmd.CombinedOutput()
	outStr := strings.TrimSpace(string(out))
	if err != nil {
		return fmt.Errorf("UIAutomation After.exe: %v - %s", err, outStr)
	}
	if !strings.Contains(outStr, "OK") {
		return fmt.Errorf("UIAutomation After.exe tidak OK: %s", outStr)
	}
	return nil
}
