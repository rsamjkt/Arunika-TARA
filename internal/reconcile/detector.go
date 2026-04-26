// Package reconcile menyediakan worker background untuk:
//   - Track online/offline state Khanza (offlineDetector)
//   - Sync antrian_lokal pending → Khanza saat online
//   - Sync pending_sep awaiting_sync → Khanza saat operator confirm
//
// Worker tidak emit Wails event langsung — pakai callback supaya
// package ini tidak depend ke Wails runtime. App layer subscribe
// callback dan forward ke runtime.EventsEmit.
package reconcile

import (
	"sync"
	"time"
)

// stateChangeCallback dipanggil hanya saat online state berubah
// (offline→online atau online→offline). Tidak fire setiap probe
// supaya UI tidak spam event.
type stateChangeCallback func(online bool)

// offlineDetector tracking state berdasarkan probe HealthCheck.
// Menyimpan last-known state untuk debounce.
type offlineDetector struct {
	mu      sync.Mutex
	online  bool
	primed  bool // false saat belum pernah probe — first probe selalu fire callback
	onState stateChangeCallback
	now     func() time.Time
	lastChk time.Time
}

func newOfflineDetector(onState stateChangeCallback) *offlineDetector {
	return &offlineDetector{
		online:  true, // optimistic default — tidak fire offline alarm sebelum cek
		onState: onState,
		now:     time.Now,
	}
}

// update terima hasil probe (err nil = online). Fire callback hanya
// kalau state berubah dari last-known. Thread-safe.
func (d *offlineDetector) update(err error) {
	online := err == nil

	d.mu.Lock()
	prev := d.online
	primed := d.primed
	d.lastChk = d.now()
	d.primed = true
	if !primed {
		// First probe: kalau hasil = offline, fire onState(false).
		// Kalau online, jangan fire (default sudah online — tidak ada
		// perubahan visible bagi UI).
		d.online = online
		d.mu.Unlock()
		if !online && d.onState != nil {
			d.onState(false)
		}
		return
	}
	d.online = online
	d.mu.Unlock()

	if online != prev && d.onState != nil {
		d.onState(online)
	}
}

// IsOnline returns last-known state.
func (d *offlineDetector) IsOnline() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.online
}

// LastCheck returns timestamp of latest probe.
func (d *offlineDetector) LastCheck() time.Time {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.lastChk
}
