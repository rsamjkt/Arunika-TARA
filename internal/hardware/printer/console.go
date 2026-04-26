package printer

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/arunika/apm-go/internal/store"
)

// ConsolePrinter adalah implementasi development di Mac/Linux.
// Print() render dokumen ke io.Writer (default os.Stdout) dalam
// format yang mudah dibaca developer:
//
//	===================================
//	[CETAK] TIKET
//	2026-04-26 14:30:15 WIB
//	-----------------------------------
//	{ "Nomor": "B-INT-005", ... }      ← JSON dari data param
//	===================================
//
// Reprint mengambil bytes dari print_history (yang sudah di-insert
// service layer) dan menulis ulang. Increment reprint_count untuk
// audit.
type ConsolePrinter struct {
	mu        sync.Mutex
	store     *store.Queries
	out       io.Writer
	available bool
	now       func() time.Time
}

var _ ThermalPrinter = (*ConsolePrinter)(nil)

// NewConsolePrinter membuat printer console dengan store untuk
// operasi Reprint. db boleh nil — kalau nil, Reprint akan return
// error (untuk test yang tidak butuh print history).
func NewConsolePrinter(db *sql.DB) *ConsolePrinter {
	p := &ConsolePrinter{
		out:       os.Stdout,
		available: true,
		now:       time.Now,
	}
	if db != nil {
		p.store = store.New(db)
	}
	return p
}

// SetWriter mengganti output writer (dipakai test untuk capture stdout).
func (p *ConsolePrinter) SetWriter(w io.Writer) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.out = w
}

// SetAvailable toggle status — simulasi kertas habis dll.
func (p *ConsolePrinter) SetAvailable(v bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.available = v
}

func (p *ConsolePrinter) IsAvailable() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.available
}

// Print render dokumen. Concurrency-safe — output di-serialize
// agar tidak interleave saat 2 goroutine print bersamaan.
func (p *ConsolePrinter) Print(ctx context.Context, docType string, data any) error {
	if !p.IsAvailable() {
		return fmt.Errorf("console printer: not available")
	}

	body, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("console printer: marshal data: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	ts := p.now().Format("2006-01-02 15:04:05 WIB")
	fmt.Fprintln(p.out, "================================================")
	fmt.Fprintf(p.out, "[CETAK] %s\n", docType)
	fmt.Fprintln(p.out, ts)
	fmt.Fprintln(p.out, "------------------------------------------------")
	p.out.Write(body)
	fmt.Fprintln(p.out)
	fmt.Fprintln(p.out, "================================================")
	return nil
}

// Reprint baca bytes dari print_history.id, tulis ulang, increment counter.
func (p *ConsolePrinter) Reprint(ctx context.Context, printHistoryID int64) error {
	if p.store == nil {
		return fmt.Errorf("console printer: reprint butuh store *sql.DB di constructor")
	}

	h, err := p.store.GetPrintHistory(ctx, printHistoryID)
	if err != nil {
		return fmt.Errorf("ambil print_history id=%d: %w", printHistoryID, err)
	}

	p.mu.Lock()
	fmt.Fprintln(p.out, "================================================")
	fmt.Fprintf(p.out, "[REPRINT] %s (id=%d, count=%d)\n",
		h.DocType, h.ID, h.ReprintCount.Int64+1)
	fmt.Fprintln(p.out, "------------------------------------------------")
	p.out.Write(h.EscposBytes)
	fmt.Fprintln(p.out)
	fmt.Fprintln(p.out, "================================================")
	p.mu.Unlock()

	if err := p.store.IncrementReprintCount(ctx, printHistoryID); err != nil {
		return fmt.Errorf("increment reprint count id=%d: %w", printHistoryID, err)
	}
	return nil
}
