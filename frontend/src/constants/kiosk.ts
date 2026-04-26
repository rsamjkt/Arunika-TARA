// Kiosk runtime constants — di-tune per deployment via config (P-051+).
// Sementara ini hardcoded default yang aman.
export const KIOSK = {
  // Idle timeout - pasien yang ngga ada interaksi 60 detik di-reset
  // ke home (ada countdown 10 detik terakhir).
  idleTimeoutSec: 60,
  idleCountdownSec: 10,

  // Auto-redirect TicketScreen ke home setelah cetak
  ticketAutoBackSec: 10,

  // Refresh counter antrian
  counterRefreshMs: 30000,

  // RS info default — production-nya dari config.toml
  // (cfg.app.rs_name belum ada di P-003, tambah saat P-051 hardening)
  defaultRSName: 'RS Anggrek Mas',
} as const
