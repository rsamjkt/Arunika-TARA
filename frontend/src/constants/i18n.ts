// APM (T.A.R.A) - i18n strings (Bahasa Indonesia).
//
// Single source untuk semua label yang ditampilkan ke pasien.
// Aturan: bahasa sehari-hari, JANGAN istilah teknis BPJS yang
// membingungkan (no "VClaim", "PPK", "FKTP" — ini untuk operator,
// bukan pasien).
//
// Multi-language support (kalau dibutuhkan kemudian) tinggal export
// peta language → strings, dan composable useI18n() pilih sesuai
// config.
export const I18N = {
  // Kiosk identity
  app: {
    name: 'Anjungan Pasien Mandiri',
    tagline: 'Layanan registrasi otomatis',
  },

  // Header status labels
  status: {
    bpjs: 'BPJS',
    sistem: 'Sistem',
    online: 'Terhubung',
    offline: 'Offline',
    warning: 'Perhatian',
  },

  // Home screen — 4 button area
  home: {
    bpjs: {
      title: 'Pasien BPJS',
      subtitle: 'Tap kartu BPJS atau ketik nomor untuk mulai',
      tag: 'Otomatis mendeteksi jenis kunjungan',
    },
    umum: {
      title: 'Pasien Umum',
      subtitle: 'Daftar tanpa kartu BPJS',
    },
    antrian: {
      title: 'Ambil Antrian',
      subtitle: 'Loket admisi, farmasi, atau customer service',
    },
    satusehat: {
      title: 'Aktivasi Satu Sehat Mobile',
      subtitle: 'Aktifkan akun Satu Sehat Anda',
    },
  },

  // Footer
  footer: {
    needHelp: 'Butuh bantuan?',
    callStaff: 'Panggil petugas',
  },

  // Idle overlay
  idle: {
    countdown: (sec: number) => `${sec}`,
    title: 'Anda masih di sini?',
    sub: 'Kembali ke awal dalam beberapa detik',
    tap: 'Sentuh layar untuk lanjut isi',
  },

  // Common buttons
  common: {
    back: 'Kembali',
    cancel: 'Batal',
    confirm: 'Konfirmasi',
    next: 'Lanjut',
    done: 'Selesai',
    print: 'Cetak',
    reprint: 'Cetak ulang',
  },
} as const
