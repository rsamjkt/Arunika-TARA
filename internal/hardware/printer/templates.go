package printer

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"sync"
	"text/template"
)

// embeddedTemplates berisi 3 template default yang di-bundle ke binary.
// Lihat templates/README.md di project root untuk panduan editing.
//
//go:embed templates/*.tmpl
var embeddedTemplates embed.FS

// docTypeToFile mapping docType (case-insensitive) ke nama file
// template di embedded FS. Disusun supaya caller pakai docType
// uppercase consistent dengan store schema (TIKET, SEP, REGISTRASI).
var docTypeToFile = map[string]string{
	"TIKET":      "templates/tiket_antrian.tmpl",
	"SEP":        "templates/sep.tmpl",
	"REGISTRASI": "templates/registrasi.tmpl",
	"TEST":       "templates/test.tmpl",
}

var (
	templatesOnce sync.Once
	templatesMap  map[string]*template.Template
	templatesErr  error
)

// loadTemplates parse semua template embedded sekali (lazy init).
// Thread-safe via sync.Once.
func loadTemplates() (map[string]*template.Template, error) {
	templatesOnce.Do(func() {
		templatesMap = make(map[string]*template.Template, len(docTypeToFile))
		for docType, path := range docTypeToFile {
			data, err := embeddedTemplates.ReadFile(path)
			if err != nil {
				templatesErr = fmt.Errorf("read embedded template %s: %w", path, err)
				return
			}
			t, err := template.New(docType).Parse(string(data))
			if err != nil {
				templatesErr = fmt.Errorf("parse template %s: %w", path, err)
				return
			}
			templatesMap[docType] = t
		}
	})
	return templatesMap, templatesErr
}

// renderTemplate render dokumen sesuai docType dan data. Return
// teks plain (sudah di-substitute field). Cocok untuk console
// output atau sebagai input ke ESC/POS encoder.
//
// Error:
//   - docType tidak dikenal → ErrTemplateNotFound
//   - Field di template tidak ada di data → text/template default
//     behavior (substitute "<no value>") — UI ditampilkan tetap,
//     tapi ada penanda yang jelas saat developer salah pasok struct.
func renderTemplate(docType string, data any) (string, error) {
	tmpls, err := loadTemplates()
	if err != nil {
		return "", err
	}

	upper := strings.ToUpper(strings.TrimSpace(docType))
	t, ok := tmpls[upper]
	if !ok {
		return "", fmt.Errorf("%w: docType %q", ErrTemplateNotFound, docType)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template %s: %w", upper, err)
	}
	return buf.String(), nil
}

// ErrTemplateNotFound dikembalikan saat docType tidak punya template
// terdaftar. Caller bisa fallback ke JSON dump untuk debugging.
var ErrTemplateNotFound = fmt.Errorf("template tidak ditemukan untuk docType")
