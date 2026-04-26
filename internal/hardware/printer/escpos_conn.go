package printer

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/arunika/apm-go/internal/config"
)

// openESCPOSConn buka koneksi printer berdasarkan cfg.Mode.
// Implementasi portable (bekerja di semua OS untuk console & network).
// Untuk USB device file, behavior tergantung OS — di Linux/Windows
// open file biasa; di Mac umumnya tidak ada device USB printer.
//
// Mode yang didukung:
//
//	"console"        → os.Stdout (debug/dev — bytes ESC/POS akan
//	                   muncul sebagai garbage di terminal)
//	"escpos_network" → tcp.Dial(cfg.Port) — port format "host:port"
//	                   (mis. "192.168.1.50:9100")
//	"escpos_usb"     → os.OpenFile(cfg.Port, O_WRONLY)
//	                   Windows: "\\\\.\\COM3" or "COM3"
//	                   Linux:   "/dev/usb/lp0"
//	"escpos_serial"  → alias escpos_usb
func openESCPOSConn(cfg config.PrinterConfig) (escposConn, error) {
	switch cfg.Mode {
	case "console", "":
		return stdoutConn{}, nil

	case "escpos_network":
		if cfg.Port == "" {
			return nil, fmt.Errorf("escpos_network: cfg.port wajib diisi (host:port)")
		}
		conn, err := net.DialTimeout("tcp", cfg.Port, 5*time.Second)
		if err != nil {
			return nil, fmt.Errorf("dial %s: %w", cfg.Port, err)
		}
		return tcpConn{Conn: conn}, nil

	case "escpos_usb", "escpos_serial":
		if cfg.Port == "" {
			return nil, fmt.Errorf("%s: cfg.port wajib diisi (device path / COM port)",
				cfg.Mode)
		}
		f, err := os.OpenFile(cfg.Port, os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return nil, fmt.Errorf("open device %s: %w", cfg.Port, err)
		}
		return fileConn{File: f}, nil

	default:
		return nil, fmt.Errorf("printer mode tidak dikenal: %q (valid: console, escpos_network, escpos_usb, escpos_serial)",
			cfg.Mode)
	}
}

// ============================================================
// Concrete connection wrappers
// ============================================================

// stdoutConn write ke os.Stdout. Close no-op (jangan close stdout
// proses).
type stdoutConn struct{}

func (stdoutConn) Write(p []byte) (int, error) { return os.Stdout.Write(p) }
func (stdoutConn) Close() error                { return nil }

// tcpConn wrap net.Conn supaya match escposConn interface.
type tcpConn struct{ net.Conn }

// fileConn wrap *os.File supaya match interface.
type fileConn struct{ *os.File }

// Compile-time interface assertions
var (
	_ escposConn = stdoutConn{}
	_ escposConn = tcpConn{}
	_ escposConn = fileConn{}
	_ io.Writer  = stdoutConn{} // sanity
)
