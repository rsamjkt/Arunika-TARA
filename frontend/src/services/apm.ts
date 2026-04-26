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
  // Hardware status
  GetHardwareStatus,
  GetSystemStatus,
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
export type HardwareStatus = main.HardwareStatus
export type SystemStatus = main.SystemStatus
export type PendingSep = store.PendingSep
export type AdminStats = main.AdminStats
export type AdminLogEntry = main.AdminLogEntry

// CardData yang di-emit lewat event "frista:card_read".
// Field nama mengikuti Go struct domain.CardData (snake_case → camelCase
// auto-converted oleh Wails JSON marshal).
export interface CardData {
  NIK: string
  Nama: string
  TglLahir: string
  Alamat: string
  NoKartu: string
  Timestamp: string
}

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

  // Pendaftaran umum
  cariPasien: (q: string): Promise<Pasien> => CariPasien(q),
  buatPendaftaran: (req: PendaftaranRequest): Promise<Pendaftaran> =>
    BuatPendaftaran(req),
  getJadwalDokter: (kdPoli: string): Promise<JadwalDokter[]> =>
    GetJadwalDokter(kdPoli),

  // Hardware status
  getHardwareStatus: (): Promise<HardwareStatus> => GetHardwareStatus(),
  getSystemStatus: (): Promise<SystemStatus> => GetSystemStatus(),

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
  // Frista card read — auto-fill form ketika user tap KTP/kartu BPJS.
  onCardRead: (handler: (data: CardData) => void): Unsubscribe => {
    EventsOn('frista:card_read', handler)
    return () => EventsOff('frista:card_read')
  },

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
