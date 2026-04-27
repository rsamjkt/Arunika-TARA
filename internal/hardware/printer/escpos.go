package printer

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"

	"github.com/arunika/apm-go/internal/config"
	"github.com/arunika/apm-go/internal/store"
)

// ============================================================
// ESC/POS command constants — bytes yang dipakai untuk format printer
// ============================================================

var (
	escReset       = []byte{0x1B, 0x40}       // ESC @  — reset printer
	escBoldOn      = []byte{0x1B, 0x45, 0x01} // ESC E 1
	escBoldOff     = []byte{0x1B, 0x45, 0x00} // ESC E 0
	escAlignLeft   = []byte{0x1B, 0x61, 0x00} // ESC a 0
	escAlignCenter = []byte{0x1B, 0x61, 0x01} // ESC a 1
	escFeed        = []byte{0x0A}             // LF
	escCutPaper    = []byte{0x1D, 0x56, 0x00} // GS V 0 — full cut
	escFeedTriple  = []byte{0x0A, 0x0A, 0x0A} // 3 baris kosong sebelum cut
	escSizeNormal  = []byte{0x1D, 0x21, 0x00} // GS ! 0  — normal
	escSizeDouble  = []byte{0x1D, 0x21, 0x11} // GS ! 0x11 — double width+height
	escSizeXL      = []byte{0x1D, 0x21, 0x33} // GS ! 0x33 — quad size (untuk nomor antrian)
)

// ============================================================
// escposConn — abstraksi koneksi printer
// ============================================================

// escposConn adalah io.Writer + Closer ke device printer.
// Implementasi konkret di escpos_conn_*.go (per platform/mode).
type escposConn interface {
	io.Writer
	io.Closer
}

// ============================================================
// ESCPOSPrinter — production printer untuk Windows/USB/Network
// ============================================================

// ESCPOSPrinter render template + convert ke ESC/POS bytes + write
// ke koneksi printer fisik. Cocok untuk Windows production.
//
// Mode di config.PrinterConfig:
//
//	"console"        → tulis ke os.Stdout (ESC/POS bytes raw — gunakan
//	                   untuk debug, lihat hexdump)
//	"escpos_network" → TCP dial ke cfg.Port (mis. "192.168.1.50:9100")
//	"escpos_usb"     → device file (Windows: COM port mis. "COM3";
//	                   Linux: /dev/usb/lp0)
//	"escpos_serial"  → sama dengan escpos_usb (alias)
//
// Lazy connection: koneksi baru di-open saat Print pertama kali
// (atau Reprint). Stop() close koneksi.
type ESCPOSPrinter struct {
	cfg   config.PrinterConfig
	store *store.Queries

	mu        sync.Mutex
	conn      escposConn
	available bool
	logger    *slog.Logger
}

var _ ThermalPrinter = (*ESCPOSPrinter)(nil)

// NewESCPOSPrinter membangun printer ESC/POS. Tidak open koneksi
// langsung — lazy saat Print pertama. Pakai dari Provider switch
// (lihat NewESCPOS factory di bawah).
func NewESCPOSPrinter(cfg config.PrinterConfig, db *sql.DB) *ESCPOSPrinter {
	p := &ESCPOSPrinter{
		cfg:       cfg,
		available: true,
		logger:    slog.Default(),
	}
	if db != nil {
		p.store = store.New(db)
	}
	return p
}

// NewESCPOS adalah factory yang dipakai Provider switch (P-030).
// Return ESCPOSPrinter untuk Windows production. Mac/Linux dev path
// di provider.go pakai ConsolePrinter langsung.
func NewESCPOS(cfg config.PrinterConfig, db *sql.DB) ThermalPrinter {
	return NewESCPOSPrinter(cfg, db)
}

// SetLogger inject logger custom.
func (p *ESCPOSPrinter) SetLogger(l *slog.Logger) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if l != nil {
		p.logger = l
	}
}

// SetAvailable toggle untuk simulasi (test atau saat printer dideteksi error).
func (p *ESCPOSPrinter) SetAvailable(v bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.available = v
}

func (p *ESCPOSPrinter) IsAvailable() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.available
}

// Print render dokumen → ESC/POS bytes → tulis ke koneksi.
func (p *ESCPOSPrinter) Print(ctx context.Context, docType string, data any) error {
	if !p.IsAvailable() {
		return fmt.Errorf("escpos printer: not available")
	}

	rendered, err := renderTemplate(docType, data)
	if err != nil {
		return fmt.Errorf("escpos render template: %w", err)
	}

	bytes := encodeESCPOS(docType, rendered)

	return p.write(bytes)
}

// Reprint baca bytes dari print_history.id, langsung tulis ulang ke
// printer (bytes sudah ESC/POS-encoded, jangan re-encode), increment counter.
func (p *ESCPOSPrinter) Reprint(ctx context.Context, printHistoryID int64) error {
	if p.store == nil {
		return fmt.Errorf("escpos printer: reprint butuh store *sql.DB di constructor")
	}

	h, err := p.store.GetPrintHistory(ctx, printHistoryID)
	if err != nil {
		return fmt.Errorf("ambil print_history id=%d: %w", printHistoryID, err)
	}

	if err := p.write(h.EscposBytes); err != nil {
		return fmt.Errorf("write reprint: %w", err)
	}

	if err := p.store.IncrementReprintCount(ctx, printHistoryID); err != nil {
		return fmt.Errorf("increment reprint count id=%d: %w", printHistoryID, err)
	}
	return nil
}

// Stop close koneksi printer (graceful shutdown). Idempotent.
func (p *ESCPOSPrinter) Stop() error {
	p.mu.Lock()
	conn := p.conn
	p.conn = nil
	p.mu.Unlock()

	if conn == nil {
		return nil
	}
	return conn.Close()
}

// write ke koneksi printer; lazy-open kalau belum.
func (p *ESCPOSPrinter) write(b []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil {
		conn, err := openESCPOSConn(p.cfg)
		if err != nil {
			return fmt.Errorf("open escpos conn (mode=%s port=%s): %w",
				p.cfg.Mode, p.cfg.Port, err)
		}
		p.conn = conn
		p.logger.Info("escpos: connection opened",
			"mode", p.cfg.Mode, "port", p.cfg.Port)
	}

	if _, err := p.conn.Write(b); err != nil {
		// Close koneksi yang putus supaya next call retry open
		_ = p.conn.Close()
		p.conn = nil
		return fmt.Errorf("write escpos bytes: %w", err)
	}
	return nil
}

// ============================================================
// encodeESCPOS — convert text rendered ke bytes ESC/POS
// ============================================================

// Marker tag → ESC/POS byte sequence. Template designer pakai marker
// inline (mis. [C]CENTERED[/C]) supaya output struk terformat tanpa
// harus tau byte sequence ESC/POS. Marker yang didukung:
//
//	[C]...[/C]   — center align
//	[B]...[/B]   — bold
//	[XL]...[/XL] — extra-large (quad size, untuk nomor antrian)
//	[BIG]...[/BIG] — double size (untuk nama RS / heading)
//	[CUT]        — paper cut di tengah dokumen (jarang dipakai)
//
// Marker tidak boleh nested kompleks — gunakan sequence linear:
//
//	[C][B]Header[/B][/C]\n
//	[C][XL]A001[/XL][/C]\n
//
// Marker tidak match kena `escSizeNormal` reset di akhir untuk safety.
var markerReplacements = []struct {
	from string
	to   []byte
}{
	{"[C]", escAlignCenter},
	{"[/C]", escAlignLeft},
	{"[B]", escBoldOn},
	{"[/B]", escBoldOff},
	{"[BIG]", escSizeDouble},
	{"[/BIG]", escSizeNormal},
	{"[XL]", escSizeXL},
	{"[/XL]", escSizeNormal},
	{"[CUT]", escCutPaper},
}

// encodeESCPOS render template output ke ESC/POS bytes:
//  1. ESC @ reset printer
//  2. Substitute marker tags ke byte sequence
//  3. 3x LF + GS V 0 cut di akhir
//
// docType saat ini diabaikan (semua marker di-handle di template);
// parameter dipertahankan untuk backward compat + future variant
// (mis. width-aware encoding 58mm vs 80mm).
func encodeESCPOS(docType, rendered string) []byte {
	_ = docType

	var buf []byte
	buf = append(buf, escReset...)
	buf = append(buf, []byte(substituteMarkers(rendered))...)

	// Safety reset — kalau template lupa tutup [/B] atau [/BIG]
	buf = append(buf, escBoldOff...)
	buf = append(buf, escSizeNormal...)
	buf = append(buf, escAlignLeft...)

	// Footer
	buf = append(buf, escFeedTriple...)
	buf = append(buf, escCutPaper...)
	return buf
}

// substituteMarkers replace marker text dengan ESC/POS bytes.
// Multi-byte safe (marker pakai ASCII, target byte arbitrary).
func substituteMarkers(s string) string {
	for _, r := range markerReplacements {
		s = strings.ReplaceAll(s, r.from, string(r.to))
	}
	return s
}
