<!--
  ResultScreen — render berbeda berdasarkan PatientType hasil
  Smart BPJS Detector. Satu file, conditional template.

  7 kasus:
    MJKN         (1) → success pill, info "booking terkonfirmasi", CTA Konfirmasi
    Kontrol      (2) → info pill, surat kontrol detail, CTA Buat surat layanan
    PostRANAP    (3) → info pill, "pasca rawat inap", CTA Pilih poli kontrol
    PostRAJAL    (4) → info pill, "lanjutan rawat jalan", CTA Pilih poli tujuan
    RujukanBaru  (5) → warning pill, info biometrik, CTA Pilih dokter
    TidakAktif   (6) → danger pill, info aktivasi, CTA Daftar umum
    Error        (7) → danger pill, info hubungi petugas

  Loading state: kalau patient.isDetecting (jarang terjadi karena
  DetectScreen yang block sampai selesai, tapi defensive).

  Back button: ke /input untuk coba lagi (bukan /home — supaya
  user yang salah ketik bisa edit).
-->
<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'

import PatientCard from '../components/PatientCard.vue'
import IdleOverlay from '../components/IdleOverlay.vue'

import { I18N } from '../constants/i18n'
import { KIOSK } from '../constants/kiosk'
import { useIdleTimeout } from '../composables/useIdleTimeout'
import { PatientType } from '../services/apm'
import { usePatientStore } from '../stores/patient'

const router = useRouter()
const patient = usePatientStore()

// Kalau user buka /result tanpa detection, redirect ke home
if (!patient.detectionResult && !patient.isDetecting) {
  router.replace({ name: 'home' })
}

const result = computed(() => patient.detectionResult)
const peserta = computed(() => patient.peserta)
const ptype = computed(() => result.value?.Type ?? PatientType.Unknown)

// Map PatientType ke pill label + variant
const pillConfig = computed(() => {
  switch (ptype.value) {
    case PatientType.MJKN:
      return { label: 'Booking Mobile JKN', variant: 'success' }
    case PatientType.Kontrol:
      return { label: 'Jadwal Kontrol', variant: 'info' }
    case PatientType.PostRANAP:
      return { label: 'Pasca Rawat Inap', variant: 'info' }
    case PatientType.PostRAJAL:
      return { label: 'Lanjutan Rawat Jalan', variant: 'info' }
    case PatientType.RujukanBaru:
      return { label: 'Kunjungan Baru', variant: 'warning' }
    case PatientType.TidakAktif:
      return { label: 'Status Tidak Aktif', variant: 'danger' }
    default:
      return { label: 'Tidak Diketahui', variant: 'danger' }
  }
})

// Format tanggal "26 Apr 2026"
function formatDate(d) {
  if (!d) return ''
  try {
    const dt = new Date(d)
    if (isNaN(dt.getTime())) return ''
    const BULAN = ['Jan','Feb','Mar','Apr','Mei','Jun','Jul','Agu','Sep','Okt','Nov','Des']
    return `${dt.getDate()} ${BULAN[dt.getMonth()]} ${dt.getFullYear()}`
  } catch {
    return ''
  }
}

const dateLabel = computed(() => formatDate(result.value?.DetectedAt))

// Detail rows per kategori
const details = computed(() => {
  const p = peserta.value
  if (!p) return []

  const base = [
    { key: 'Nomor RM', value: p.NoRM || '—' },
    { key: 'Tgl. lahir', value: p.TglLahir || '—' },
    { key: 'Kelas hak', value: p.KelasHak ? `Kelas ${p.KelasHak}` : '—' },
  ]

  // MJKN → tambah info booking dari result.Data
  if (ptype.value === PatientType.MJKN) {
    const b = result.value?.Data
    if (b) {
      base.push(
        { key: 'Poli', value: b.NmPoli || b.KdPoli || '—', accent: true },
        { key: 'Dokter', value: b.NmDokter || '—' },
        { key: 'Estimasi', value: b.EstimasiDilayani || '—' },
        { key: 'No booking', value: b.NoBooking || b.NoAntrian || '—' },
      )
    }
  }
  // Kontrol → tambah surat kontrol info
  if (ptype.value === PatientType.Kontrol) {
    const list = result.value?.Data ?? []
    const sk = Array.isArray(list) ? list[0] : list
    if (sk) {
      base.push(
        { key: 'No surat', value: sk.NoSurat || '—' },
        { key: 'Tgl rencana', value: sk.TglRencana || '—' },
        { key: 'Poli', value: sk.NmPoli || sk.KdPoli || '—', accent: true },
      )
    }
  }
  return base
})

// CTA per kategori
async function goNext() {
  switch (ptype.value) {
    case PatientType.MJKN:
      // P-045 TicketScreen — flow MJKN check-in
      router.push({ name: 'tiket', query: { from: 'mjkn' } })
      break
    case PatientType.Kontrol:
      // Lanjut buat SEP kontrol — sementara ke tiket
      router.push({ name: 'tiket', query: { from: 'kontrol' } })
      break
    case PatientType.PostRANAP:
    case PatientType.PostRAJAL:
    case PatientType.RujukanBaru:
      // Pick dokter dulu — sementara ke tiket
      router.push({ name: 'tiket', query: { from: 'rujukan' } })
      break
    case PatientType.TidakAktif:
      // Daftar sebagai pasien umum
      router.push({ name: 'input', query: { mode: 'umum' } })
      break
    default:
      router.push({ name: 'home' })
  }
}

async function reInput() {
  await patient.reset()
  router.push({ name: 'input', query: { mode: 'bpjs' } })
}

async function callStaff() {
  // No-op visual — admin panel/event akan handle
}

// Idle timeout
const { isCountingDown, secondsLeft } = useIdleTimeout({
  totalSeconds: KIOSK.idleTimeoutSec,
  countdownThreshold: KIOSK.idleCountdownSec,
  onTimeout: async () => {
    await patient.reset()
    router.push({ name: 'home' })
  },
})

// CTA label per kategori
const ctaLabel = computed(() => {
  switch (ptype.value) {
    case PatientType.MJKN: return 'Konfirmasi kedatangan dan cetak tiket'
    case PatientType.Kontrol: return 'Buat surat layanan kontrol'
    case PatientType.PostRANAP: return 'Pilih poli untuk kontrol'
    case PatientType.PostRAJAL: return 'Pilih poli tujuan'
    case PatientType.RujukanBaru: return 'Pilih dokter dan lanjutkan'
    case PatientType.TidakAktif: return 'Daftar sebagai pasien umum'
    default: return 'Hubungi petugas'
  }
})

// Info bar per kategori (warna bg + pesan)
const infoBar = computed(() => {
  switch (ptype.value) {
    case PatientType.MJKN:
      return { variant: 'success', text: 'Booking dari Mobile JKN terkonfirmasi. Cetak tiket untuk konfirmasi kedatangan.' }
    case PatientType.RujukanBaru:
      return { variant: 'warning', text: 'Verifikasi sidik jari diperlukan setelah pilih dokter (untuk pasien dewasa non-IGD).' }
    case PatientType.TidakAktif:
      return { variant: 'danger', text: 'Status BPJS Anda saat ini tidak aktif. Hubungi BPJS Kesehatan untuk aktivasi, atau daftar sebagai pasien umum di RS ini.' }
    case PatientType.Error:
      return { variant: 'danger', text: 'Sistem tidak dapat memeriksa status Anda saat ini. Silakan hubungi petugas.' }
    default:
      return null
  }
})

const infoBarClass = (v) => {
  switch (v) {
    case 'success': return 'bg-success-bg text-success border-success-border'
    case 'warning': return 'bg-warning-bg text-warning border-warning-border'
    case 'danger': return 'bg-danger-bg text-danger border-danger-border'
    default: return 'bg-bg text-text-secondary border-border'
  }
}
</script>

<template>
  <main class="min-h-screen bg-bg flex flex-col">
    <!-- Header -->
    <header
      class="bg-surface border-b border-border flex items-center
             px-[clamp(16px,3vw,28px)] py-[clamp(10px,1.8vw,16px)]
             gap-[clamp(8px,1.5vw,14px)]"
    >
      <button
        type="button"
        class="text-text-secondary hover:text-text-primary
               px-3 py-1 rounded-btn flex items-center gap-1
               text-[clamp(12px,1.6vw,14px)]"
        @click="reInput"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
          stroke="currentColor" stroke-width="2.5" stroke-linecap="round"
          stroke-linejoin="round" class="w-4 h-4"
        >
          <polyline points="15 18 9 12 15 6" />
        </svg>
        {{ I18N.common.back }}
      </button>
      <h1 class="text-[clamp(15px,2.2vw,18px)] font-medium text-text-primary">
        Hasil pemeriksaan
      </h1>
    </header>

    <!-- Body -->
    <section
      class="flex-1 flex flex-col gap-[clamp(10px,2vw,16px)]
             p-[clamp(12px,2.5vw,20px)]
             max-w-[560px] mx-auto w-full"
    >
      <!-- Loading state (defensive) -->
      <div
        v-if="patient.isDetecting"
        class="bg-surface border border-border rounded-card p-6 text-center"
      >
        <p class="text-text-secondary">Memuat hasil...</p>
      </div>

      <!-- Patient card -->
      <PatientCard
        v-else-if="peserta"
        :pill-label="pillConfig.label"
        :pill-variant="pillConfig.variant"
        :date-label="dateLabel"
        :nama="peserta.Nama"
        :no-kartu="peserta.NoKartu"
        :details="details"
      />

      <!-- Tidak ada peserta (Error) tetap tampilkan box minimum -->
      <div
        v-else
        class="bg-surface border border-border rounded-card p-6 text-center"
      >
        <p class="text-[clamp(13px,1.8vw,16px)] font-medium text-text-primary">
          Data peserta tidak tersedia
        </p>
        <p class="text-[clamp(11px,1.5vw,13px)] text-text-muted mt-2">
          Sistem belum bisa memeriksa data Anda. Silakan coba lagi atau hubungi petugas.
        </p>
      </div>

      <!-- Info bar -->
      <div
        v-if="infoBar"
        :class="[
          'rounded-card border p-[clamp(10px,1.8vw,14px)] flex items-start gap-2',
          infoBarClass(infoBar.variant),
        ]"
      >
        <svg
          v-if="infoBar.variant === 'success'"
          xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
          stroke="currentColor" stroke-width="2.2" stroke-linecap="round"
          stroke-linejoin="round" class="w-5 h-5 mt-[2px] shrink-0"
        >
          <polyline points="20 6 9 17 4 12" />
        </svg>
        <svg
          v-else-if="infoBar.variant === 'warning'"
          xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
          stroke="currentColor" stroke-width="2.2" stroke-linecap="round"
          stroke-linejoin="round" class="w-5 h-5 mt-[2px] shrink-0"
        >
          <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
          <line x1="12" y1="9" x2="12" y2="13" />
          <line x1="12" y1="17" x2="12.01" y2="17" />
        </svg>
        <svg
          v-else
          xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
          stroke="currentColor" stroke-width="2.2" stroke-linecap="round"
          stroke-linejoin="round" class="w-5 h-5 mt-[2px] shrink-0"
        >
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="8" x2="12" y2="12" />
          <line x1="12" y1="16" x2="12.01" y2="16" />
        </svg>
        <p class="text-[clamp(11px,1.5vw,13px)] leading-snug">
          {{ infoBar.text }}
        </p>
      </div>

      <!-- CTA primary -->
      <button
        v-if="ptype !== PatientType.Error"
        type="button"
        class="w-full rounded-kiosk transition-opacity active:opacity-85
               bg-blue text-white border border-blue
               px-[clamp(14px,2.5vw,20px)] py-[clamp(14px,2.5vw,20px)]
               text-[clamp(14px,2vw,17px)] font-medium
               flex items-center justify-between gap-3
               min-h-[clamp(56px,8vw,72px)]"
        @click="goNext"
      >
        <span>{{ ctaLabel }}</span>
        <svg
          xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
          stroke="currentColor" stroke-width="2.5" stroke-linecap="round"
          stroke-linejoin="round" class="w-5 h-5 shrink-0"
        >
          <polyline points="9 18 15 12 9 6" />
        </svg>
      </button>

      <!-- Ghost button: "Bukan saya — masukkan ulang" / "Hubungi petugas" -->
      <button
        type="button"
        class="w-full rounded-kiosk transition-colors
               bg-surface text-text-secondary border border-border
               hover:border-border-strong active:bg-bg
               px-[clamp(12px,2vw,16px)] py-[clamp(10px,1.8vw,14px)]
               text-[clamp(12px,1.6vw,14px)]"
        @click="ptype === PatientType.Error ? callStaff() : reInput()"
      >
        <template v-if="ptype === PatientType.Error">
          Hubungi petugas
        </template>
        <template v-else>
          Bukan saya — masukkan ulang
        </template>
      </button>
    </section>

    <IdleOverlay :seconds-left="secondsLeft" :visible="isCountingDown" />
  </main>
</template>
