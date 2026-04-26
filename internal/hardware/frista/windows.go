package frista

import (
	"github.com/arunika/apm-go/internal/config"
)

// NewWindowsReader sementara return MockReader sampai real impl
// USB HID + Windows UI Automation di-implement di P-031.
//
// Saat real impl masuk, function ini akan:
//   - exec.CommandContext(ctx, cfg.ExePath) dengan CREATE_NO_WINDOW
//   - Inject login lewat user32.dll FindWindowW + SendMessageW
//   - Pipe stdout JSON → parse ke CardData → kirim ke channel
//
// Untuk sekarang, NewWindowsReader hanya membungkus MockReader
// supaya Provider on Windows tidak crash; service layer & UI
// tetap bisa di-test di Mac dengan mock yang sama.
//
// File ini SENGAJA tidak diberi build tag — function harus exist
// di semua platform supaya provider.go (yang non-tagged) bisa
// merefer. Real Windows impl akan dipindah ke windows_actual.go
// dengan //go:build windows.
func NewWindowsReader(cfg config.FristaConfig) CardReader {
	// TODO P-031: ganti dengan WindowsReader sebenarnya
	return NewMock(0)
}
