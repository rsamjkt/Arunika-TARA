//go:build windows

// Win32 UI Automation + clipboard helpers untuk Frista. Pure Go syscall
// (NewLazyDLL/NewProc) — TIDAK butuh CGO, jadi cross-compile dari Mac
// via mingw-w64 tetap jalan.

package frista

import (
	"errors"
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

// Default class names dialog login Frista (Delphi VCL convention,
// sama seperti After.exe). Override via config kalau RS pakai versi
// non-standar — verifikasi dengan Spy++ di kiosk Windows.
const (
	defaultFristaLoginClass  = "TfrmLogin"
	defaultFristaEditClass   = "TEdit"
	defaultFristaButtonClass = "TButton"
)

// Windows message + clipboard format constants
const (
	wmSetText      = 0x000C
	bmClick        = 0x00F5
	cfUnicodeText  = 13
)

var (
	user32              = syscall.NewLazyDLL("user32.dll")
	procFindWindowW     = user32.NewProc("FindWindowW")
	procSendMessageW    = user32.NewProc("SendMessageW")
	procEnumChildWnd    = user32.NewProc("EnumChildWindows")
	procGetClassName    = user32.NewProc("GetClassNameW")
	procOpenClipboard   = user32.NewProc("OpenClipboard")
	procCloseClipboard  = user32.NewProc("CloseClipboard")
	procGetClipboardData = user32.NewProc("GetClipboardData")

	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	procGlobalLock = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")
)

// injectFristaLogin paste username + password ke dialog login Frista
// lalu klik tombol login. Pattern sama dengan After.exe.
func injectFristaLogin(username, password, classLogin, classEdit, classButton string) error {
	if classLogin == "" {
		classLogin = defaultFristaLoginClass
	}
	if classEdit == "" {
		classEdit = defaultFristaEditClass
	}
	if classButton == "" {
		classButton = defaultFristaButtonClass
	}

	hWnd, err := waitForWindow(classLogin, 5*time.Second)
	if err != nil {
		return fmt.Errorf("dialog login Frista (class=%q) tidak ditemukan: %w", classLogin, err)
	}

	editFields := findChildWindowsByClass(hWnd, classEdit, 2)
	if len(editFields) < 2 {
		return fmt.Errorf("hanya menemukan %d %s (butuh ≥2 untuk username+password)",
			len(editFields), classEdit)
	}
	if err := setText(editFields[0], username); err != nil {
		return fmt.Errorf("set username: %w", err)
	}
	if err := setText(editFields[1], password); err != nil {
		return fmt.Errorf("set password: %w", err)
	}

	buttons := findChildWindowsByClass(hWnd, classButton, 1)
	if len(buttons) == 0 {
		return fmt.Errorf("tombol login (%s) tidak ditemukan", classButton)
	}
	clickButton(buttons[0])
	return nil
}

// waitForWindow polling FindWindowW sampai dapat handle atau timeout.
func waitForWindow(className string, timeout time.Duration) (uintptr, error) {
	deadline := time.Now().Add(timeout)
	classPtr, err := syscall.UTF16PtrFromString(className)
	if err != nil {
		return 0, err
	}
	for time.Now().Before(deadline) {
		hWnd, _, _ := procFindWindowW.Call(uintptr(unsafe.Pointer(classPtr)), 0)
		if hWnd != 0 {
			return hWnd, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return 0, fmt.Errorf("timeout waiting window class=%q", className)
}

// findChildWindowsByClass iterasi child windows dari parent dan
// kembalikan handle yang class-nya match. limit > 0 = stop setelah
// kumpul N handle.
func findChildWindowsByClass(parent uintptr, className string, limit int) []uintptr {
	var found []uintptr
	classBuf := make([]uint16, 256)

	cb := syscall.NewCallback(func(hWnd uintptr, lParam uintptr) uintptr {
		ret, _, _ := procGetClassName.Call(hWnd,
			uintptr(unsafe.Pointer(&classBuf[0])),
			uintptr(len(classBuf)))
		if ret > 0 {
			gotClass := syscall.UTF16ToString(classBuf[:ret])
			if gotClass == className {
				found = append(found, hWnd)
				if limit > 0 && len(found) >= limit {
					return 0 // stop enumeration
				}
			}
		}
		return 1
	})

	procEnumChildWnd.Call(parent, cb, 0)
	return found
}

// setText send WM_SETTEXT ke control.
func setText(hWnd uintptr, text string) error {
	textPtr, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return err
	}
	procSendMessageW.Call(hWnd, wmSetText, 0, uintptr(unsafe.Pointer(textPtr)))
	return nil
}

// clickButton send BM_CLICK ke button.
func clickButton(hWnd uintptr) {
	procSendMessageW.Call(hWnd, bmClick, 0, 0)
}

// readClipboardText baca clipboard text Unicode (CF_UNICODETEXT).
// Return error kalau clipboard busy atau tidak ada text format.
//
// Pola Win32 standar:
//
//	OpenClipboard → GetClipboardData(CF_UNICODETEXT)
//	→ GlobalLock(handle) → copy bytes → GlobalUnlock → CloseClipboard
func readClipboardText() (string, error) {
	r1, _, _ := procOpenClipboard.Call(0)
	if r1 == 0 {
		return "", errors.New("OpenClipboard failed (busy?)")
	}
	defer procCloseClipboard.Call()

	hData, _, _ := procGetClipboardData.Call(cfUnicodeText)
	if hData == 0 {
		return "", errors.New("clipboard tidak ada CF_UNICODETEXT")
	}

	pData, _, _ := procGlobalLock.Call(hData)
	if pData == 0 {
		return "", errors.New("GlobalLock failed")
	}
	defer procGlobalUnlock.Call(hData)

	// Read UTF-16 string sampai null terminator (max 64KB safety)
	const maxLen = 32 * 1024 // 32k uint16 = 64KB
	ptr := (*[maxLen]uint16)(unsafe.Pointer(pData))
	n := 0
	for n < maxLen && ptr[n] != 0 {
		n++
	}
	return syscall.UTF16ToString(ptr[:n]), nil
}
