// Kiosk runtime constants — di-tune per deployment via config (P-051+).
// Sementara ini hardcoded default yang aman.
export const KIOSK = {
  // Idle timeout - pasien yang ngga ada interaksi 90 detik di-reset
  // ke home (ada countdown 10 detik terakhir).
  // Naik dari 60 → 90 supaya lansia punya waktu baca instruksi panjang.
  idleTimeoutSec: 90,
  idleCountdownSec: 10,

  // Auto-redirect TicketScreen ke home setelah cetak.
  // Naik dari 10 → 25 supaya lansia bisa baca tiket lengkap.
  // Reset di tap apa saja (lihat TicketScreen.vue).
  ticketAutoBackSec: 25,

  // Refresh counter antrian
  counterRefreshMs: 30000,

  // RS info default — production-nya dari config.toml
  // (cfg.app.rs_name belum ada di P-003, tambah saat P-051 hardening)
  defaultRSName: 'RS Anggrek Mas',
} as const
