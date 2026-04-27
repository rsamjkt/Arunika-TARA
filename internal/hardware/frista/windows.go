//go:build !windows

package frista

import (
	"github.com/arunika/apm-go/internal/config"
)

// NewWindowsReader pada non-Windows return MockReader supaya
// development di Mac/Linux tetap jalan tanpa Win32 dependency.
// Real Windows implementation ada di windows_real.go (build tag: windows).
func NewWindowsReader(cfg config.FristaConfig) CardReader {
	_ = cfg
	return NewMock(0)
}
