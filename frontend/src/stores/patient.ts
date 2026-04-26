// Pinia store untuk state pasien sepanjang flow Detection → SEP → Tiket.
//
// Single source of truth — komponen Vue cukup baca dari sini, tidak
// perlu prop drill atau emit chain antar-screen.
import { defineStore } from 'pinia'
import { apmService } from '../services/apm'
import type {
  DetectionResult,
  Peserta,
  Ticket,
  SEP,
} from '../services/apm'

export interface DetectStep {
  id: string
  label: string
  state: 'wait' | 'active' | 'done' | 'error'
}

interface PatientState {
  input: string
  peserta: Peserta | null
  detectionResult: DetectionResult | null
  isDetecting: boolean
  detectSteps: DetectStep[]
  error: string | null
  lastTicket: Ticket | null
  lastSEP: SEP | null
}

export const usePatientStore = defineStore('patient', {
  state: (): PatientState => ({
    input: '',
    peserta: null,
    detectionResult: null,
    isDetecting: false,
    detectSteps: [],
    error: null,
    lastTicket: null,
    lastSEP: null,
  }),

  getters: {
    isAktif: (state) => state.peserta?.StatusAktif === '1',
    hasResult: (state) => state.detectionResult !== null,
  },

  actions: {
    /**
     * detect — call backend Smart BPJS Detector lalu cache hasil.
     * Set isDetecting true selama proses, error kalau gagal.
     */
    async detect(input: string): Promise<DetectionResult | null> {
      this.input = input
      this.isDetecting = true
      this.error = null
      this.detectionResult = null
      this.peserta = null

      try {
        const result = await apmService.detect(input)
        this.detectionResult = result
        this.peserta = result.Peserta ?? null
        return result
      } catch (e) {
        this.error = (e as Error).message ?? String(e)
        return null
      } finally {
        this.isDetecting = false
      }
    },

    /**
     * reset — clear semua state pasien + signal backend ResetSession
     * (untuk clear cached lastPeserta di App). Dipanggil saat:
     *  - Idle timeout
     *  - User klik "Mulai dari awal"
     *  - Setelah TicketScreen selesai countdown
     */
    async reset(): Promise<void> {
      this.input = ''
      this.peserta = null
      this.detectionResult = null
      this.isDetecting = false
      this.detectSteps = []
      this.error = null
      this.lastTicket = null
      this.lastSEP = null
      try {
        await apmService.resetSession()
      } catch {
        // Backend mungkin belum siap atau session sudah clean -
        // tidak fatal, lanjut.
      }
    },

    setLastTicket(t: Ticket) {
      this.lastTicket = t
    },

    setLastSEP(s: SEP) {
      this.lastSEP = s
    },
  },
})
