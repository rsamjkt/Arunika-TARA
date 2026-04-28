// APM (T.A.R.A) — TypeScript wrapper untuk semua Wails IPC call.
//
// Layer ini memberi tipe-aman + namespace bersih supaya komponen Vue
// tidak harus import langsung dari wailsjs/. Juga jadi single source
// untuk swap mock saat test (Vitest dapat mock module ini).

import {
  // Detection
  DetectPatient,
  ResetSession,
  // Antrian
  CreateAntrian,
  GetCounters,
  // SEP
  BuatSEPRujukan,
  BuatSEPKontrol,
  BuatSEPPostRANAP,
  BuatSEPPostRAJAL,
  // Pendaftaran umum
  CariPasien,
  BuatPendaftaran,
  GetJadwalDokter,
  GetPoliklinikAktif,
  // Branding (theme + logo + audio)
  GetBranding,
  // Hardware status
  GetHardwareStatus,
  GetSystemStatus,
  RunStartupChecks,
  // Reprint
  Reprint,
  // Admin
  GetPendingSEPs,
  ConfirmSEPSync,
  ResetCounters,
  VerifyAdminPIN,
  GetAdminStats,
  GetRecentLogs,
  TestPrint,
} from '../../wailsjs/go/main/App'

// Biometrik verifikasi — SEP butuh token validasi (umur >=17, non-IGD).
// Backend agent (paralel) akan bind methods ini di Go side; bindings
// `wailsjs/go/main/App.d.ts` akan re-generate di run berikutnya `wails dev`.
// Sementara: stub declarations ditambahkan manual di App.js + App.d.ts
// supaya import-nya resolve. Wails regen akan overwrite stub itu.
import { VerifikasiWajah, VerifikasiSidikJari } from '../../wailsjs/go/main/App'

import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

import type { domain, main, store } from '../../wailsjs/go/models'

// ============================================================
// Re-export tipe untuk komponen Vue
// ============================================================

export type DetectionResult = domain.DetectionResult
export type Peserta = domain.Peserta
export type Ticket = domain.Ticket
export type SEP = domain.SEP
export type SEPRequest = domain.SEPRequest
export type Pasien = domain.Pasien
export type Pendaftaran = domain.Pendaftaran
export type PendaftaranRequest = domain.PendaftaranRequest
export type JadwalDokter = domain.JadwalDokter
export type Poliklinik = domain.Poliklinik
export type HardwareStatus = main.HardwareStatus
export type SystemStatus = main.SystemStatus
export type CheckResult = main.CheckResult
export type Branding = main.Branding
export type PendingSep = store.PendingSep
export type AdminStats = main.AdminStats
export type AdminLogEntry = main.AdminLogEntry

// PatientType enum mirror dari domain/detection.go (urut iota).
export const PatientType = {
  Unknown: 0,
  MJKN: 1,
  Kontrol: 2,
  PostRANAP: 3,
  PostRAJAL: 4,
  RujukanBaru: 5,
  TidakAktif: 6,
  Error: 7,
} as const

export type PatientTypeValue = typeof PatientType[keyof typeof PatientType]

// Method biometrik yang dipilih user di BiometrikChoiceModal.
// face        → Frista (kamera)
// fingerprint → After.exe (sensor)
export type BiometrikMethod = 'face' | 'fingerprint'

// Sentinel string yang di-emit backend kalau SEP butuh biometrik.
// Vue match pakai err.message.includes(BIOMETRIK_REQUIRED_HINT).
export const BIOMETRIK_REQUIRED_HINT = 'biometrik diperlukan'

// ============================================================
// apmService — wrapper functional untuk semua call backend
// ============================================================

export const apmService = {
  // Detection
  detect: (identifier: string): Promise<DetectionResult> =>
    DetectPatient(identifier),
  resetSession: (): Promise<void> => ResetSession(),

  // Antrian
  createAntrian: (jenis: string, subJenis: string): Promise<Ticket> =>
    CreateAntrian(jenis, subJenis),
  getCounters: (): Promise<Record<string, number>> => GetCounters(),

  // SEP — Peserta diambil dari cache backend (set by detect())
  buatSEPRujukan: (req: SEPRequest): Promise<SEP> => BuatSEPRujukan(req),
  buatSEPKontrol: (noSuratKontrol: string, kdDokter = ''): Promise<SEP> =>
    BuatSEPKontrol(noSuratKontrol, kdDokter),
  buatSEPPostRANAP: (kdPoli: string, kdDokter: string): Promise<SEP> =>
    BuatSEPPostRANAP(kdPoli, kdDokter),
  buatSEPPostRAJAL: (kdPoli: string, kdDokter: string): Promise<SEP> =>
    BuatSEPPostRAJAL(kdPoli, kdDokter),

  // Biometrik — return token string (dipakai di field SEPRequest.BiometrikToken /
  // FPToken sesuai backend wiring). Method berbeda per provider:
  //   verifikasiWajah     → trigger Frista (kamera)
  //   verifikasiSidikJari → trigger After.exe (sensor sidik jari)
  verifikasiWajah: (noPeserta: string): Promise<string> => VerifikasiWajah(noPeserta),
  verifikasiSidikJari: (noPeserta: string): Promise<string> => VerifikasiSidikJari(noPeserta),

  // Pendaftaran umum
  cariPasien: (q: string): Promise<Pasien> => CariPasien(q),
  buatPendaftaran: (req: PendaftaranRequest): Promise<Pendaftaran> =>
    BuatPendaftaran(req),
  getJadwalDokter: (kdPoli: string): Promise<JadwalDokter[]> =>
    GetJadwalDokter(kdPoli),
  getPoliklinikAktif: (): Promise<Poliklinik[]> => GetPoliklinikAktif(),

  // Branding
  getBranding: (): Promise<Branding> => GetBranding(),

  // Hardware status
  getHardwareStatus: (): Promise<HardwareStatus> => GetHardwareStatus(),
  getSystemStatus: (): Promise<SystemStatus> => GetSystemStatus(),
  runStartupChecks: (): Promise<CheckResult[]> => RunStartupChecks(),

  // Reprint
  reprint: (printHistoryID: number): Promise<void> => Reprint(printHistoryID),

  // Admin
  getPendingSEPs: (): Promise<PendingSep[]> => GetPendingSEPs(),
  confirmSEPSync: (id: number): Promise<void> => ConfirmSEPSync(id),
  resetCounters: (): Promise<void> => ResetCounters(),
  verifyAdminPIN: (pin: string): Promise<boolean> => VerifyAdminPIN(pin),
  getAdminStats: (): Promise<AdminStats> => GetAdminStats(),
  getRecentLogs: (limit = 50): Promise<AdminLogEntry[]> => GetRecentLogs(limit),
  testPrint: (): Promise<void> => TestPrint(),
}

// ============================================================
// useWailsEvents — composable untuk subscribe Wails events
// ============================================================

// Helper unsubscribe untuk dipanggil di onUnmounted Vue component.
export type Unsubscribe = () => void

// Detect step state — dipakai DetectScreen progress list.
export type DetectStepState = 'wait' | 'active' | 'done' | 'error'
export interface DetectStepUpdate {
  step: string
  state: DetectStepState
}

export const useWailsEvents = () => ({
  // Smart Detector step progress (P-011 emit dari Go saat tiap check selesai).
  onDetectStep: (handler: (data: DetectStepUpdate) => void): Unsubscribe => {
    EventsOn('detect:step_update', handler)
    return () => EventsOff('detect:step_update')
  },

  // Reconcile worker emit ketika koneksi Khanza pulih/putus.
  onSystemOffline: (handler: (offline: boolean) => void): Unsubscribe => {
    EventsOn('system:offline', handler)
    return () => EventsOff('system:offline')
  },

  // Printer emit error (kertas habis, USB disconnect, dll).
  onPrinterError: (handler: (msg: string) => void): Unsubscribe => {
    EventsOn('printer:error', handler)
    return () => EventsOff('printer:error')
  },

  // Hardware status broadcast — admin panel auto-refresh.
  onHardwareStatus: (handler: (st: HardwareStatus) => void): Unsubscribe => {
    EventsOn('hardware:status', handler)
    return () => EventsOff('hardware:status')
  },
})
