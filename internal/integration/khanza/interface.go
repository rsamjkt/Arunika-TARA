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

	// GetPoliklinikAktif mengembalikan list poliklinik dengan status='1'
	// (aktif). Dipakai screen Pasien Umum untuk picker poli.
	GetPoliklinikAktif(ctx context.Context) ([]domain.Poliklinik, error)

	// GetBookingMJKN mencari booking online (Mobile JKN, antrol BPJS,
	// reguler RS) untuk no_rkm_medis pada tanggal tgl. Return nil jika
	// tidak ada. Dipakai detector PostMJKN sebagai fallback ketika
	// Antrol API tidak responsif.
	GetBookingMJKN(ctx context.Context, noRM string, tgl time.Time) (*domain.BookingMJKN, error)

	// GetRujukanInternalAntarPoli mencari rujukan internal antar-poli
	// (rujukan_internal_poli) yang relate ke kunjungan pasien dalam
	// daysBack hari terakhir. Filter: kd_poli rujukan != kd_poli asal.
	// Dipakai detector PostRAJAL.
	GetRujukanInternalAntarPoli(ctx context.Context, noRM string, daysBack int) ([]domain.RujukanInternalPoli, error)

	// CheckDuplicateRegistration cek di reg_periksa apakah pasien sudah
	// terdaftar di poli+dokter+kd_pj yang sama hari ini. Return true
	// kalau sudah ada — caller harus reject untuk hindari duplikasi
	// klaim BPJS atau pendaftaran ganda.
	CheckDuplicateRegistration(ctx context.Context, noRM, kdPoli, kdDokter, tglRegistrasi, kdPj string) (bool, error)

	// CheckDoctorOnLeave cek di tabel jadwal_cuti_libur apakah dokter
	// cuti pada tgl tertentu. Return true kalau cuti — caller harus
	// reject pendaftaran supaya pasien tidak terlantar.
	CheckDoctorOnLeave(ctx context.Context, kdDokter, tglRegistrasi string) (bool, error)

	// BuatPendaftaran mendaftarkan pasien ke poli dengan dokter & tgl
	// tertentu. Untuk pasien BPJS wajib pasok NoSEP yang sudah di-issue.
	BuatPendaftaran(ctx context.Context, req domain.PendaftaranRequest) (*domain.Pendaftaran, error)

	// BuatAntrian meminta nomor antrian dari Khanza (atomic — anti
	// duplikasi multi-kiosk).
	BuatAntrian(ctx context.Context, req domain.AntrianRequest) (*domain.Ticket, error)

	// SimpanSEP menyimpan SEP yang sudah di-issue VClaim ke Khanza
	// (untuk audit trail dan klaim BPJS). Akan auto-link ke
	// reg_periksa terbaru (no_rawat).
	//
	// Side-effect: kalau sep.PRBCode non-empty, juga insert ke `bpjs_prb`.
	SimpanSEP(ctx context.Context, sep domain.SEP) error

	// SimpanRujukMasuk insert ke tabel `rujuk_masuk` — link rujukan FKTP
	// ke kunjungan rawat (no_rawat). Dipanggil setelah BuatPendaftaran
	// BPJS yang berasal dari rujukan baru.
	SimpanRujukMasuk(ctx context.Context, r domain.RujukMasuk) error

	// SimpanRujukanBPJS insert ke `bridging_rujukan_bpjs` — audit trail
	// rujukan VClaim untuk klaim BPJS (separate dari rujuk_masuk yang
	// link ke no_rawat).
	SimpanRujukanBPJS(ctx context.Context, r domain.RujukanBPJS) error

	// UpdateSatuSehatID menyimpan IHS Number (Satu Sehat ID) ke master
	// pasien Khanza setelah aktivasi sukses.
	UpdateSatuSehatID(ctx context.Context, noRM, ihsNumber string) error
}
