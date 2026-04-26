//go:build windows

// File ini hanya di-compile saat GOOS=windows.
// Berisi syscall ke user32.dll untuk inject login otomatis ke window
// dialog After.exe. Pakai pure Go syscall (NewLazyDLL/NewProc) — TIDAK
// butuh CGO, jadi cross-compile dari Mac via mingw-w64 tetap bisa.

package fingerprint

import (
	"errors"
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

// Class name dialog login After.exe yang sudah diketahui dari riset
// lapangan. Bisa berubah kalau vendor After.exe update — kalau itu
// terjadi, update konstanta ini + verifikasi lewat tool seperti Spy++.
const (
	afterLoginClassName = "TfrmLogin"
	afterUsernameField  = "TEdit"
	afterPasswordField  = "TEdit"
	afterLoginButton    = "TButton"
)

// Windows message constants
const (
	wmSetText = 0x000C
	bmClick   = 0x00F5
)

var (
	user32           = syscall.NewLazyDLL("user32.dll")
	procFindWindowW  = user32.NewProc("FindWindowW")
	procFindWindowEx = user32.NewProc("FindWindowExW")
	procSendMessageW = user32.NewProc("SendMessageW")
	procEnumChild    = user32.NewProc("EnumChildWindows")
	procGetClassName = user32.NewProc("GetClassNameW")
)

// injectAfterLogin mencari window login After.exe lalu set username +
// password + click tombol login.
//
// Strategi:
//
//  1. Tunggu sampai dialog login muncul (max 5 detik retry)
//  2. FindWindowW(class=TfrmLogin) → handle hWnd
//  3. Iterasi child windows → 2 TEdit pertama = username & password
//     (urutan z-order — verified manual via Spy++)
//  4. SendMessageW(WM_SETTEXT) ke masing-masing
//  5. FindWindowExW(class=TButton) → click via SendMessage(BM_CLICK)
func injectAfterLogin(username, password string) error {
	hWnd, err := waitForWindow(afterLoginClassName, 5*time.Second)
	if err != nil {
		return fmt.Errorf("dialog login After.exe tidak ditemukan: %w", err)
	}

	// Cari child TEdit (username & password).
	editFields := findChildWindowsByClass(hWnd, afterUsernameField, 2)
	if len(editFields) < 2 {
		return fmt.Errorf("hanya menemukan %d TEdit (butuh ≥2 untuk username+password)",
			len(editFields))
	}
	if err := setText(editFields[0], username); err != nil {
		return fmt.Errorf("set username: %w", err)
	}
	if err := setText(editFields[1], password); err != nil {
		return fmt.Errorf("set password: %w", err)
	}

	// Cari TButton login — biasanya tombol pertama.
	buttons := findChildWindowsByClass(hWnd, afterLoginButton, 1)
	if len(buttons) == 0 {
		return errors.New("tombol login (TButton) tidak ditemukan")
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
//
// Implementasi pakai EnumChildWindows callback — di Go ini perlu
// syscall.NewCallback() untuk wrap closure.
func findChildWindowsByClass(parent uintptr, className string, limit int) []uintptr {
	var found []uintptr
	classBuf := make([]uint16, 256)

	cb := syscall.NewCallback(func(hWnd uintptr, lParam uintptr) uintptr {
		// Get class name
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
		return 1 // continue
	})

	procEnumChild.Call(parent, cb, 0)
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
