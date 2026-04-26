package fingerprint

import (
	"github.com/arunika/apm-go/internal/config"
)

// NewWindowsHeadless sementara return MockVerifier sampai real impl
// di P-032 dengan flow:
//
//   - Spawn cfg.ExePath (After.exe) dengan CREATE_NO_WINDOW flag
//   - Inject login lewat user32.dll (FindWindowW + SendMessageW)
//   - POST ke cfg.RestURL/api/fingerprint untuk start scan
//   - Poll GET /api/fingerprint/status setiap cfg.PollIntervalMs
//   - Sukses: return FPResult{Success: true, Token: ...}
//   - Timeout cfg.ScanTimeoutSec: return error
//
// File ini SENGAJA tidak diberi build tag — function harus exist
// di semua platform supaya provider.go (yang non-tagged) bisa merefer.
func NewWindowsHeadless(cfg config.FingerprintConfig) FingerprintVerifier {
	// TODO P-032: ganti dengan WindowsHeadlessVerifier sebenarnya
	return NewMock()
}
