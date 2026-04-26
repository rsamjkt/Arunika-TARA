// Package khanza adalah client untuk SIMRS Khanza Laravel REST API.
//
// Auth: token API key dikirim via header X-API-Key (bukan standard
// Authorization: Bearer — Khanza punya middleware custom).
//
// Behavior penting:
//   - Connection refused (Khanza server mati) → return domain.ErrOffline
//     (BUKAN panic, BUKAN error generic) supaya caller bisa fallback
//     ke offline queue.
//   - 401 → log warning + return wrapped error (kemungkinan token salah/expired).
//   - 5xx → retry 2x dengan backoff 500ms.
//   - 4xx → tidak di-retry (validation error tidak akan berubah).
package khanza

import (
	"context"
	"time"

	"github.com/arunika/apm-go/internal/domain"
)

// KhanzaClient adalah surface API untuk SIMRS Khanza.
//
// Implementasi nyata: *Client (HTTP via resty).
// Implementasi test: *MockKhanzaClient.
type KhanzaClient interface {
	// HealthCheck ping server. nil = reachable, error = unreachable
	// (mungkin ErrOffline kalau koneksi mati). Dipakai oleh reconcile
	// worker untuk track online/offline state.
	HealthCheck(ctx context.Context) error

	// CariPasien melakukan pencarian pasien berdasarkan NoRM, NIK, nama,
	// atau no telp. Khanza menentukan strategi match di sisi server.
	// Return (nil, nil) jika tidak ada match (bukan error).
	CariPasien(ctx context.Context, q string) (*domain.Pasien, error)

	// GetSuratKontrol mengembalikan list surat kontrol pasien.
	// Filter tanggal dilakukan di sisi caller (lihat detector.checkKontrol).
	GetSuratKontrol(ctx context.Context, noRM string) ([]domain.SuratKontrol, error)

	// GetRiwayatRANAP mengembalikan episode rawat inap pasien.
	GetRiwayatRANAP(ctx context.Context, noRM string) ([]domain.RiwayatRANAP, error)

	// GetKunjunganAktif mengembalikan kunjungan rawat jalan aktif
	// (status != 'Sudah'/'Batal').
	GetKunjunganAktif(ctx context.Context, noRM string) ([]domain.Kunjungan, error)

	// GetJadwalDokter mengembalikan dokter yang praktik di poli pada tgl.
	// Khanza filter berdasarkan hari & status cuti.
	GetJadwalDokter(ctx context.Context, kdPoli string, tgl time.Time) ([]domain.JadwalDokter, error)

	// BuatPendaftaran mendaftarkan pasien ke poli dengan dokter & tgl
	// tertentu. Untuk pasien BPJS wajib pasok NoSEP yang sudah di-issue.
	BuatPendaftaran(ctx context.Context, req domain.PendaftaranRequest) (*domain.Pendaftaran, error)

	// BuatAntrian meminta nomor antrian dari Khanza (atomic — anti
	// duplikasi multi-kiosk).
	BuatAntrian(ctx context.Context, req domain.AntrianRequest) (*domain.Ticket, error)

	// SimpanSEP menyimpan SEP yang sudah di-issue VClaim ke Khanza
	// (untuk audit trail dan klaim BPJS).
	SimpanSEP(ctx context.Context, sep domain.SEP) error

	// UpdateSatuSehatID menyimpan IHS Number (Satu Sehat ID) ke master
	// pasien Khanza setelah aktivasi sukses.
	UpdateSatuSehatID(ctx context.Context, noRM, ihsNumber string) error
}
