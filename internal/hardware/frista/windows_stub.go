//go:build !windows

package frista

import (
	"github.com/arunika/apm-go/internal/config"
)

// NewWindowsHeadless di non-Windows hanya delegate ke MockVerifier.
// Provider switch (runtime.GOOS) di internal/hardware/provider.go
// memastikan function ini hanya dipanggil dengan path "windows" branch
// — di Mac/Linux dev, default branch pakai NewMock() langsung.
//
// Stub ini ada supaya:
//   - Provider.go (no build tag) bisa merefer NewWindowsHeadless tanpa
//     compile error di non-Windows.
//   - Test/dev di Mac yang explicitly call NewWindowsHeadless masih
//     dapat mock yang berfungsi (bukan crash).
//
// Pattern identik dengan fingerprint/windows_stub.go.
func NewWindowsHeadless(cfg config.FristaConfig) FaceVerifier {
	_ = cfg
	return NewMock()
}
