// Package vclaim adalah client untuk BPJS VClaim API v2.0.
//
// Auth: HMAC-SHA256(consID + "&" + timestamp, secretKey) → base64
// Response body: AES-256-CBC ciphertext (base64) yang harus didekripsi
// dengan key=SHA256(secretKey+consID), IV=key[:16].
//
// Setiap method WAJIB di-pasok context.Context — semua HTTP call
// di-cancel kalau context dibatalkan, dan timeout per-request mengikuti
// cfg.Server.TimeoutMs (P-003).
package vclaim

import (
	"context"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

// VClaimClient adalah surface API yang dipakai service layer.
// Implementasi: *Client (real HTTP) dan *MockVClaimClient (testing).
type VClaimClient interface {
	// GetPeserta mencari peserta berdasarkan identifier (No. Kartu JKN
	// atau NIK). Cascade: coba sebagai noKartu dulu, fallback ke NIK
	// jika gagal. tgl dipakai sebagai tglSEP untuk validasi periode aktif.
	GetPeserta(ctx context.Context, identifier string, tgl time.Time) (*domain.Peserta, error)

	// GetRencanaKontrol mengembalikan list surat kontrol yang masih
	// aktif untuk noKartu pada rentang tanggal [today, tgl].
	GetRencanaKontrol(ctx context.Context, noKartu string, tgl time.Time) ([]domain.SuratKontrol, error)

	// GetRiwayatPelayanan mengembalikan riwayat kunjungan/pelayanan
	// pasien antara tglAwal dan tglAkhir (inklusif).
	GetRiwayatPelayanan(ctx context.Context, noKartu string, tglAwal, tglAkhir time.Time) ([]domain.RiwayatPelayanan, error)

	// ValidasiRujukan memeriksa nomor rujukan FKTP. tgl adalah
	// tanggal SEP yang akan dibuat (untuk cek masa berlaku).
	ValidasiRujukan(ctx context.Context, noSurat string, tgl time.Time) (*domain.Rujukan, error)

	// CreateSEP membuat SEP baru dari rujukan FKTP.
	CreateSEP(ctx context.Context, req domain.SEPRequest) (*domain.SEP, error)

	// CreateSEPKontrol membuat SEP dari surat kontrol (SKDP) yang
	// sudah ada — dipakai untuk pasien dengan jadwal kontrol.
	CreateSEPKontrol(ctx context.Context, req domain.SEPKontrolRequest) (*domain.SEP, error)

	// CekSEPDuplikasi memeriksa di server BPJS apakah pasien sudah
	// punya SEP aktif untuk noKartu pada tglSEP. Return *SEP kalau
	// duplikasi ditemukan, nil kalau aman.
	//
	// Penting untuk anti-fraud + billing safety — local cek di
	// bridging_sep tidak cukup karena server BPJS bisa punya state
	// yang berbeda (mis. kalau kiosk crash post-VClaim insert tapi
	// pre-DB insert).
	CekSEPDuplikasi(ctx context.Context, noKartu, tglSEP string) (*domain.SEP, error)

	// BuatRencanaKontrol POST /RencanaKontrol/insert untuk schedule
	// SKDP pasien post-discharge. Output: noSuratKontrol baru yang
	// caller harus simpan ke bridging_surat_kontrol_bpjs.
	//
	// Berbeda dengan CreateSEPKontrol yang ISSUE SEP dari SKDP
	// existing — method ini CREATE SKDP baru.
	BuatRencanaKontrol(ctx context.Context, req domain.RencanaKontrolRequest) (*domain.RencanaKontrol, error)
}
