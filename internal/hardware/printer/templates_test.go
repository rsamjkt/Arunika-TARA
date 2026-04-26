package printer

import (
	"errors"
	"strings"
	"testing"
)

func TestRenderTemplate_Tiket(t *testing.T) {
	data := map[string]any{
		"RSName":       "RS Anggrek Mas",
		"Tanggal":      "2026-04-26 14:30",
		"JenisAntrian": "POLI PENYAKIT DALAM",
		"Nomor":        "B-INT-005",
	}
	got, err := renderTemplate("TIKET", data)
	if err != nil {
		t.Fatalf("renderTemplate: %v", err)
	}
	for _, sub := range []string{
		"RS Anggrek Mas",
		"2026-04-26 14:30",
		"POLI PENYAKIT DALAM",
		"B-INT-005",
		"Harap tunggu panggilan",
	} {
		if !strings.Contains(got, sub) {
			t.Errorf("output tidak mengandung %q:\n%s", sub, got)
		}
	}
}

func TestRenderTemplate_SEP(t *testing.T) {
	data := map[string]any{
		"NoSEP":      "0123456789012345678",
		"Nama":       "Budi Santoso",
		"NoKartu":    "0001234567890012",
		"NmPoli":     "Penyakit Dalam",
		"NmDokter":   "dr. Alpha",
		"TglSEP":     "2026-04-26",
		"KelasRawat": "2",
	}
	got, err := renderTemplate("SEP", data)
	if err != nil {
		t.Fatalf("renderTemplate: %v", err)
	}
	for _, sub := range []string{
		"SURAT ELIGIBILITAS PESERTA",
		"0123456789012345678",
		"Budi Santoso",
		"0001234567890012",
		"Penyakit Dalam",
		"dr. Alpha",
		"2026-04-26",
	} {
		if !strings.Contains(got, sub) {
			t.Errorf("output tidak mengandung %q:\n%s", sub, got)
		}
	}
}

func TestRenderTemplate_Registrasi(t *testing.T) {
	data := map[string]any{
		"NoRawat":      "2026/04/26/0001",
		"Nama":         "Budi Santoso",
		"NmPoli":       "Penyakit Dalam",
		"NmDokter":     "dr. Beta",
		"TglKunjungan": "2026-04-26",
		"NoAntrian":    "B-INT-005",
	}
	got, err := renderTemplate("REGISTRASI", data)
	if err != nil {
		t.Fatalf("renderTemplate: %v", err)
	}
	for _, sub := range []string{
		"BUKTI PENDAFTARAN",
		"2026/04/26/0001",
		"Budi Santoso",
		"B-INT-005",
	} {
		if !strings.Contains(got, sub) {
			t.Errorf("output tidak mengandung %q:\n%s", sub, got)
		}
	}
}

func TestRenderTemplate_DocTypeUnknown_Error(t *testing.T) {
	_, err := renderTemplate("UNKNOWN_DOC", nil)
	if err == nil {
		t.Fatal("expected error untuk docType unknown")
	}
	if !errors.Is(err, ErrTemplateNotFound) {
		t.Errorf("err harus wrap ErrTemplateNotFound, got: %v", err)
	}
}

func TestRenderTemplate_CaseInsensitiveDocType(t *testing.T) {
	data := map[string]any{
		"RSName": "X", "Tanggal": "X", "JenisAntrian": "X", "Nomor": "X",
	}
	for _, dt := range []string{"tiket", "Tiket", "TIKET", "  TIKET  "} {
		if _, err := renderTemplate(dt, data); err != nil {
			t.Errorf("docType %q seharusnya match TIKET: %v", dt, err)
		}
	}
}

func TestRenderTemplate_FieldHilang_PlaceholderNoValue(t *testing.T) {
	// Pasok hanya sebagian field — text/template default substitusi
	// "<no value>" untuk field hilang. UI tetap render, tapi developer
	// bisa lihat field salah.
	data := map[string]any{"RSName": "RS X"} // missing Tanggal, JenisAntrian, Nomor
	got, err := renderTemplate("TIKET", data)
	if err != nil {
		t.Fatalf("renderTemplate (partial): %v", err)
	}
	if !strings.Contains(got, "<no value>") {
		t.Errorf("field hilang seharusnya muncul sebagai <no value>:\n%s", got)
	}
}
