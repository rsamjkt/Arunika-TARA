package khanza

import (
	"errors"
	"net"
	"strings"
	"syscall"

	"github.com/arunika/apm-go/internal/domain"
)

// isOfflineError mendeteksi apakah error mengindikasikan Khanza server
// tidak bisa dihubungi (connection refused, DNS resolve gagal, no route,
// timeout di TCP layer).
//
// Dipakai untuk transformasi error → domain.ErrOffline supaya caller
// (terutama service layer) bisa fallback ke offline queue.
func isOfflineError(err error) bool {
	if err == nil {
		return false
	}

	// syscall ECONNREFUSED — server mati / port tidak listen
	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}
	// EHOSTUNREACH — no route to host (LAN issue)
	if errors.Is(err, syscall.EHOSTUNREACH) {
		return true
	}
	// ENETUNREACH
	if errors.Is(err, syscall.ENETUNREACH) {
		return true
	}

	// DNS resolve failure
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	// Generic *net.OpError yang membungkus connection refused —
	// kadang error chain tidak punya syscall di Go yang lebih lama
	// atau di Windows.
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		// Periksa pesan error sebagai fallback (best-effort)
		msg := strings.ToLower(opErr.Error())
		if strings.Contains(msg, "connection refused") ||
			strings.Contains(msg, "no such host") ||
			strings.Contains(msg, "network is unreachable") {
			return true
		}
	}

	// Last-resort: substring match pesan
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "network is unreachable")
}

// wrapOffline mengembalikan domain.ErrOffline jika err mengindikasikan
// jaringan mati. Selain itu, return err apa adanya.
func wrapOffline(err error) error {
	if isOfflineError(err) {
		return domain.ErrOffline
	}
	return err
}
