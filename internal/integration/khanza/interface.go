// Package khanza adalah client untuk SIMRS Khanza Laravel REST API.
//
// Auth: Bearer token (X-API-Key header).
// Status: interface + mock saja untuk dependency injection di Detector.
// Real HTTP client akan di-implement di P-020.
package khanza

import (
	"context"

	"github.com/arunika/apm-go/internal/domain"
)

// KhanzaClient adalah surface API untuk SIMRS Khanza.
// Detector (P-011) butuh 3 method: GetSuratKontrol, GetRiwayatRANAP,
// GetKunjunganAktif. Method lain (CariPasien, BuatPendaftaran, dll)
// akan ditambahkan di P-020+.
type KhanzaClient interface {
	// GetSuratKontrol mengembalikan list surat kontrol pasien
	// (rentang waktu di-filter di sisi caller).
	GetSuratKontrol(ctx context.Context, noRM string) ([]domain.SuratKontrol, error)

	// GetRiwayatRANAP mengembalikan episode rawat inap pasien.
	// Detector pakai TglKeluar untuk filter post-RANAP.
	GetRiwayatRANAP(ctx context.Context, noRM string) ([]domain.RiwayatRANAP, error)

	// GetKunjunganAktif mengembalikan kunjungan rawat jalan aktif
	// (status != 'Sudah'/'Batal'). Detector pakai untuk identifikasi
	// pasien post-RAJAL dengan SKDP ke poli berbeda.
	GetKunjunganAktif(ctx context.Context, noRM string) ([]domain.Kunjungan, error)
}
