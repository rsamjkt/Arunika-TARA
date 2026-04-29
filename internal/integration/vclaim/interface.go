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

	// CekFingerprintStatus mengecek apakah pasien sudah melakukan
	// verifikasi biometrik (sidik jari/wajah) di server BPJS untuk
	// tanggal pelayanan tertentu. Mirror vendor cekFinger().
	//
	// Verified=true → SEP boleh di-issue tanpa prompt biometrik tambahan.
	// Verified=false → frontend WAJIB prompt BiometrikChoiceModal dulu.
	CekFingerprintStatus(ctx context.Context, noKartu string, tgl time.Time) (*FingerprintStatus, error)

	// AprovalSEP — fallback approval saat FP gagal (vendor line 2122).
	// Dipakai operator override supaya SEP bisa di-issue walau biometrik
	// belum lewat. POST /Sep/aprovalSEP.
	AprovalSEP(ctx context.Context, req FPFallbackRequest) (*FPFallbackResponse, error)

	// PengajuanSEP — pengajuan resmi ke BPJS supaya SEP bisa diterbitkan
	// walau FP tidak match (vendor line 2173). POST /Sep/pengajuanSEP.
	PengajuanSEP(ctx context.Context, req FPFallbackRequest) (*FPFallbackResponse, error)
}
