//go:build windows

// Win32 helpers untuk Frista vendor pattern (BukaFrista di
// DlgRegistrasiSEPPertama.java:3764). Vendor pakai java.awt.Robot +
// Toolkit.getDefaultToolkit().getSystemClipboard(); kita replikasi
// dengan syscall langsung ke user32.dll + kernel32.dll.
//
// Operasi yang di-cover:
//   - SetClipboardText: copy string ke clipboard system
//   - SendKeyCombo: simulasi keystroke (Ctrl+V, Tab, Enter, dll)
//   - BringWindowToFront: aktivasi window berdasarkan executable name

package frista

import (
	"fmt"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procOpenClipboard       = user32.NewProc("OpenClipboard")
	procCloseClipboard      = user32.NewProc("CloseClipboard")
	procEmptyClipboard      = user32.NewProc("EmptyClipboard")
	procSetClipboardData    = user32.NewProc("SetClipboardData")
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procFindWindowW         = user32.NewProc("FindWindowW")
	procEnumWindows         = user32.NewProc("EnumWindows")
	procGetWindowTextW      = user32.NewProc("GetWindowTextW")
	procIsWindowVisible     = user32.NewProc("IsWindowVisible")
	procShowWindow          = user32.NewProc("ShowWindow")
	procSendInput           = user32.NewProc("SendInput")
	procPostMessageW        = user32.NewProc("PostMessageW")

	procGlobalAlloc  = kernel32.NewProc("GlobalAlloc")
	procGlobalLock   = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")

	procGetWindowRect   = user32.NewProc("GetWindowRect")
	procSetCursorPos    = user32.NewProc("SetCursorPos")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
	swShow        = 5
	swRestore     = 9

	wmClose = 0x0010

	// SendInput INPUT_KEYBOARD constants
	inputKeyboard      = 1
	keyEventfKeyUp     = 0x0002
	keyEventfUnicode   = 0x0004
	keyEventfScancode  = 0x0008

	// Virtual-key codes
	vkControl = 0x11
	vkTab     = 0x09
	vkReturn  = 0x0D
	vkSpace   = 0x20
	vkV       = 0x56
)

// SetClipboardText — set clipboard system ke string s. Mirror Java
// Toolkit.getDefaultToolkit().getSystemClipboard().setContents(...).
func SetClipboardText(s string) error {
	// Open clipboard (NULL hwnd OK untuk single-app set)
	r, _, e := procOpenClipboard.Call(0)
	if r == 0 {
		return fmt.Errorf("OpenClipboard: %v", e)
	}
	defer procCloseClipboard.Call()

	if r, _, e := procEmptyClipboard.Call(); r == 0 {
		return fmt.Errorf("EmptyClipboard: %v", e)
	}

	// Convert string ke UTF-16 + null terminator
	utf16 := syscall.StringToUTF16(s)
	size := uintptr(len(utf16) * 2) // 2 bytes per WCHAR

	// Allocate global memory
	hMem, _, _ := procGlobalAlloc.Call(gmemMoveable, size)
	if hMem == 0 {
		return fmt.Errorf("GlobalAlloc gagal")
	}
	pMem, _, _ := procGlobalLock.Call(hMem)
	if pMem == 0 {
		return fmt.Errorf("GlobalLock gagal")
	}
	// Copy UTF-16 ke memory
	dst := unsafe.Slice((*uint16)(unsafe.Pointer(pMem)), len(utf16))
	copy(dst, utf16)
	procGlobalUnlock.Call(hMem)

	r, _, e = procSetClipboardData.Call(cfUnicodeText, hMem)
	if r == 0 {
		return fmt.Errorf("SetClipboardData: %v", e)
	}
	return nil
}

// INPUT struct untuk SendInput. Layout sesuai Win32 API.
type input struct {
	Type uint32
	Ki   keyboardInput
	_    [8]byte // padding supaya alignment match KEYBDINPUT/MOUSEINPUT/HARDWAREINPUT union
}

type keyboardInput struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

// sendKey — single virtual-key event (down or up).
func sendKey(vk uint16, keyUp bool) error {
	flags := uint32(0)
	if keyUp {
		flags = keyEventfKeyUp
	}
	in := input{
		Type: inputKeyboard,
		Ki: keyboardInput{
			WVk:     vk,
			DwFlags: flags,
		},
	}
	r, _, e := procSendInput.Call(
		1,
		uintptr(unsafe.Pointer(&in)),
		unsafe.Sizeof(in),
	)
	if r != 1 {
		return fmt.Errorf("SendInput: %v", e)
	}
	return nil
}

// PressKey — single key press (down then up).
func PressKey(vk uint16) error {
	if err := sendKey(vk, false); err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)
	return sendKey(vk, true)
}

// PressCtrlV — kombo Ctrl+V untuk paste clipboard.
func PressCtrlV() error {
	if err := sendKey(vkControl, false); err != nil {
		return err
	}
	time.Sleep(15 * time.Millisecond)
	if err := sendKey(vkV, false); err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)
	if err := sendKey(vkV, true); err != nil {
		return err
	}
	if err := sendKey(vkControl, true); err != nil {
		return err
	}
	return nil
}

// PressTab — Tab key (next focus).
func PressTab() error { return PressKey(vkTab) }

// PressEnter — Enter / Return key (submit).
func PressEnter() error { return PressKey(vkReturn) }

// PressSpace — Space (kadang dipakai checkbox/button).
func PressSpace() error { return PressKey(vkSpace) }

// PasteText — set clipboard + Ctrl+V. Helper umum untuk inject string
// ke field input yang aktif.
func PasteText(text string) error {
	if err := SetClipboardText(text); err != nil {
		return fmt.Errorf("set clipboard: %w", err)
	}
	// Beri delay supaya clipboard settle sebelum paste
	time.Sleep(50 * time.Millisecond)
	if err := PressCtrlV(); err != nil {
		return fmt.Errorf("ctrl+v: %w", err)
	}
	return nil
}

// FindWindowByTitleSubstring cari window yang title-nya contains
// substring (case-insensitive). Return HWND atau 0 kalau tidak ada.
//
// Dipakai untuk locate Frista window setelah spawn — exec.Cmd.Process.Pid
// tidak langsung memberi HWND; cara paling reliable adalah scan titles.
func FindWindowByTitleSubstring(substr string) uintptr {
	substrLower := strings.ToLower(substr)
	var found uintptr

	cb := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		visR, _, _ := procIsWindowVisible.Call(hwnd)
		if visR == 0 {
			return 1 // continue
		}
		buf := make([]uint16, 256)
		n, _, _ := procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), 256)
		if n == 0 {
			return 1
		}
		title := syscall.UTF16ToString(buf[:n])
		if strings.Contains(strings.ToLower(title), substrLower) {
			found = hwnd
			return 0 // stop enum
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)
	return found
}

// BringToFront aktivasi window (HWND) ke foreground + restore kalau
// minimized. Mirror vendor `bringToFront(urlfrista)`.
func BringToFront(hwnd uintptr) error {
	if hwnd == 0 {
		return fmt.Errorf("HWND 0 — window tidak ada")
	}
	procShowWindow.Call(hwnd, swRestore)
	procShowWindow.Call(hwnd, swShow)
	r, _, e := procSetForegroundWindow.Call(hwnd)
	if r == 0 {
		return fmt.Errorf("SetForegroundWindow: %v", e)
	}
	return nil
}

// ClickWindowCenter klik kiri di tengah window (focus field noKartu Frista).
// Mirror vendor: mouse click center untuk fokus field input setelah login.
func ClickWindowCenter(hwnd uintptr) error {
	if hwnd == 0 {
		return fmt.Errorf("HWND 0")
	}
	var rect struct{ Left, Top, Right, Bottom int32 }
	r, _, e := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
	if r == 0 {
		return fmt.Errorf("GetWindowRect: %v", e)
	}
	cx := (rect.Left + rect.Right) / 2
	cy := (rect.Top + rect.Bottom) / 2

	// GetSystemMetrics(SM_CXSCREEN=0, SM_CYSCREEN=1) untuk normalisasi SendInput
	sw, _, _ := procGetSystemMetrics.Call(0)
	sh, _, _ := procGetSystemMetrics.Call(1)
	if sw == 0 || sh == 0 {
		return fmt.Errorf("GetSystemMetrics gagal")
	}
	// Koordinat SendInput mouse harus dalam range 0-65535
	nx := uintptr(int64(cx) * 65535 / int64(sw))
	ny := uintptr(int64(cy) * 65535 / int64(sh))

	type mouseInput struct {
		Type uint32
		Dx   int32
		Dy   int32
		Data uint32
		Flags uint32
		Time  uint32
		Extra uintptr
		_     [8]byte
	}
	const (
		inputMouse      = 0
		mousefMove      = 0x0001
		mousefLeftDown  = 0x0002
		mousefLeftUp    = 0x0004
		mousefAbsolute  = 0x8000
	)
	move := mouseInput{Type: inputMouse, Dx: int32(nx), Dy: int32(ny), Flags: mousefMove | mousefAbsolute}
	down := mouseInput{Type: inputMouse, Dx: int32(nx), Dy: int32(ny), Flags: mousefLeftDown | mousefAbsolute}
	up   := mouseInput{Type: inputMouse, Dx: int32(nx), Dy: int32(ny), Flags: mousefLeftUp | mousefAbsolute}

	for _, inp := range []mouseInput{move, down, up} {
		inp := inp
		procSendInput.Call(1, uintptr(unsafe.Pointer(&inp)), unsafe.Sizeof(inp))
		time.Sleep(30 * time.Millisecond)
	}
	return nil
}

// CloseWindow kirim WM_CLOSE ke window. Frista akan exit gracefully.
func CloseWindow(hwnd uintptr) error {
	if hwnd == 0 {
		return nil
	}
	r, _, e := procPostMessageW.Call(hwnd, wmClose, 0, 0)
	if r == 0 {
		return fmt.Errorf("PostMessage WM_CLOSE: %v", e)
	}
	return nil
}
