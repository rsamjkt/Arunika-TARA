// Package frista adalah abstraksi pembaca kartu (Frista card reader).
// Pasien mengetuk KTP/kartu BPJS di reader → Wails frontend menerima
// event "frista:card_read" dengan CardData → form input auto-fill.
//
// Status implementasi:
//   - interface.go (file ini) — kontrak yang dipakai service layer + Wails app
//   - mock.go                  — implementasi development di Mac/Linux
//   - windows.go               — stub yang sementara return mock; akan
//                                diganti real impl headless USB HID di P-031
package frista

import (
	"context"
	"time"
)

// CardReader adalah surface API yang dipakai Wails app + handler:
//   - Start: nyalakan reader (spawn process di Windows, listen HTTP di Mac)
//   - Stop: graceful shutdown
//   - IsAvailable: untuk status panel
//   - CardRead: channel hasil baca kartu (read-only untuk caller)
type CardReader interface {
	Start(ctx context.Context) error
	Stop() error
	IsAvailable() bool
	CardRead() <-chan CardData
}

// CardData adalah hasil baca satu kartu. Field NoKartu hanya terisi
// untuk kartu BPJS — KTP biasa hanya isi NIK/Nama/dll.
type CardData struct {
	NIK       string
	Nama      string
	TglLahir  string // "2006-01-02"
	Alamat    string
	NoKartu   string // No Kartu BPJS, kosong jika hanya KTP
	Timestamp time.Time
}
