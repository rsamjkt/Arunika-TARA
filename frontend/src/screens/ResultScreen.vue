<!--
  ResultScreen — render berbeda berdasarkan PatientType.

  Spec ketat P-043:
    - PatientCard dengan 4 pill warna (per kategori)
    - MJKN: detail booking + CTA Konfirmasi → /tiket
    - Kontrol: surat kontrol detail + DokterPicker (panggil
      GetJadwalDokter onMounted, default selected idx 0) +
      CTA "Buat surat layanan" (await BuatSEPKontrol)
    - RujukanBaru: detail rujukan + info bar biometrik
      CONDITIONAL (hanya kalau perluBiometrik = umur>=17 + non-IGD)
    - TidakAktif: info bar merah + CTA "Daftar pasien umum" +
      ghost "Hubungi petugas"
    - Loading state: CTA disabled + spinner saat API call
    - Error: AlertModal dengan pesan dari domain.UserMessage()
-->
<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import PatientCard from '../components/PatientCard.vue'
import IdleOverlay from '../components/IdleOverlay.vue'
import AlertModal from '../components/AlertModal.vue'
import DokterPicker from '../components/DokterPicker.vue'

import { I18N } from '../constants/i18n'
import { KIOSK } from '../constants/kiosk'
import { useIdleTimeout } from '../composables/useIdleTimeout'
import { apmService, PatientType } from '../services/apm'
import { usePatientStore } from '../stores/patient'

const router = useRouter()
const patient = usePatientStore()

if (!patient.detectionResult && !patient.isDetecting) {
  router.replace({ name: 'home' })
}

const result = computed(() => patient.detectionResult)
const peserta = computed(() => patient.peserta)
const ptype = computed(() => result.value?.Type ?? PatientType.Unknown)

// State CTA & error
const ctaLoading = ref(false)
const errorVisible = ref(false)
const errorMsg = ref('')

// Dokter picker state (untuk Kontrol)
const dokterList = ref([]) // JadwalDokter[]
const selectedDokter = ref('')

// Pill config
const pillConfig = computed(() => {
  switch (ptype.value) {
    case PatientType.MJKN: return { label: 'Booking Mobile JKN', variant: 'success' }
    case PatientType.Kontrol: return { label: 'Jadwal Kontrol', variant: 'info' }
    case PatientType.PostRANAP: return { label: 'Pasca Rawat Inap', variant: 'info' }
    case PatientType.PostRAJAL: return { label: 'Lanjutan Rawat Jalan', variant: 'info' }
    case PatientType.RujukanBaru: return { label: 'Kunjungan Baru', variant: 'warning' }
    case PatientType.TidakAktif: return { label: 'Status Tidak Aktif', variant: 'danger' }
    default: return { label: 'Tidak Diketahui', variant: 'danger' }
  }
})

function formatDate(d) {
  if (!d) return ''
  try {
    const dt = new Date(d)
    if (isNaN(dt.getTime())) return ''
    const BULAN = ['Jan','Feb','Mar','Apr','Mei','Jun','Jul','Agu','Sep','Okt','Nov','Des']
    return `${dt.getDate()} ${BULAN[dt.getMonth()]} ${dt.getFullYear()}`
  } catch { return '' }
}
const dateLabel = computed(() => formatDate(result.value?.DetectedAt))

// Compute perluBiometrik di frontend (mirror dari Go domain rule):
//   age < 17 → false
//   kdPoli IGD/UGD → false
//   else → true
function computeAgeYears(tglLahir) {
  if (!tglLahir) return 0
  const t = new Date(tglLahir)
  if (isNaN(t.getTime())) return 0
  const now = new Date()
  let years = now.getFullYear() - t.getFullYear()
  if (now.getMonth() < t.getMonth() ||
      (now.getMonth() === t.getMonth() && now.getDate() < t.getDate())) {
    years--
  }
  return Math.max(0, years)
}
function isIGD(kdPoli) {
  const u = (kdPoli || '').toUpperCase().trim()
  return u.startsWith('IGD') || u.startsWith('UGD') || u === 'EMR'
}
const kdPoliRujukan = computed(() => {
  // Untuk RujukanBaru, kdPoli dari rujukan FKTP — kita tidak punya
  // langsung di detection result. Pakai detail kosong → default
  // butuh biometrik kecuali pasti IGD.
  return ''
})
const perluBiometrik = computed(() => {
  if (!peserta.value) return false
  const age = computeAgeYears(peserta.value.TglLahir)
  if (age < 17) return false
  if (isIGD(kdPoliRujukan.value)) return false
  return true
})

// Detail rows per kategori
const details = computed(() => {
  const p = peserta.value
  if (!p) return []
  const base = [
    { key: 'Nomor RM', value: p.NoRM || '—' },
    { key: 'Tgl. lahir', value: p.TglLahir || '—' },
    { key: 'Kelas hak', value: p.KelasHak ? `Kelas ${p.KelasHak}` : '—' },
  ]
  if (ptype.value === PatientType.MJKN) {
    const b = result.value?.Data
    if (b) {
      base.push(
        { key: 'Poli', value: b.NmPoli || b.KdPoli || '—', accent: true },
        { key: 'Dokter', value: b.NmDokter || '—' },
        { key: 'Estimasi jam', value: b.EstimasiDilayani || b.JamPraktik || '—', accent: true },
        { key: 'No booking', value: b.NoBooking || b.NoAntrian || '—' },
      )
    }
  }
  if (ptype.value === PatientType.Kontrol) {
    const list = result.value?.Data ?? []
    const sk = Array.isArray(list) ? list[0] : list
    if (sk) {
      base.push(
        { key: 'No surat kontrol', value: sk.NoSurat || '—' },
        { key: 'Tgl rencana', value: sk.TglRencana || '—' },
        { key: 'Poli', value: sk.NmPoli || sk.KdPoli || '—', accent: true },
      )
    }
  }
  if (ptype.value === PatientType.RujukanBaru) {
    // Tidak ada detail rujukan FKTP di detection result —
    // service layer akan fetch saat BuatSEPRujukan. Tampilkan
    // placeholder yang akan update di iterasi mendatang.
  }
  return base
})

// Get poli code untuk lookup jadwal dokter (Kontrol)
const kdPoliKontrol = computed(() => {
  const list = result.value?.Data ?? []
  const sk = Array.isArray(list) ? list[0] : list
  return sk?.KdPoli ?? ''
})

// Load jadwal dokter saat screen mount kalau Kontrol
onMounted(async () => {
  if (ptype.value === PatientType.Kontrol && kdPoliKontrol.value) {
    try {
      const list = await apmService.getJadwalDokter(kdPoliKontrol.value)
      dokterList.value = list ?? []
      // Default: pilih dokter aktif pertama
      const firstAktif = (list ?? []).find((d) => d.Aktif)
      if (firstAktif) selectedDokter.value = firstAktif.KdDokter
    } catch (e) {
      // Jadwal tidak tersedia — UI tampilkan empty state via DokterPicker
    }
  }
})

// CTA actions
async function goNext() {
  if (ctaLoading.value) return
  ctaLoading.value = true
  try {
    switch (ptype.value) {
      case PatientType.MJKN:
        // BuatCheckinMJKN belum ada di backend — sementara langsung ke tiket
        router.push({ name: 'tiket', query: { from: 'mjkn' } })
        break
      case PatientType.Kontrol: {
        if (!selectedDokter.value) {
          throw new Error('Silakan pilih dokter terlebih dahulu')
        }
        const list = result.value?.Data ?? []
        const sk = Array.isArray(list) ? list[0] : list
        if (!sk?.NoSurat) {
          throw new Error('Surat kontrol tidak ditemukan')
        }
        const sep = await apmService.buatSEPKontrol(sk.NoSurat, selectedDokter.value)
        patient.setLastSEP(sep)
        router.push({ name: 'tiket', query: { from: 'kontrol' } })
        break
      }
      case PatientType.PostRANAP:
      case PatientType.PostRAJAL:
      case PatientType.RujukanBaru:
        // P-046+ akan punya DokterPickerScreen tersendiri
        router.push({ name: 'tiket', query: { from: 'rujukan' } })
        break
      case PatientType.TidakAktif:
        await patient.reset()
        router.push({ name: 'input', query: { mode: 'umum' } })
        break
      default:
        router.push({ name: 'home' })
    }
  } catch (e) {
    errorMsg.value = e?.message ?? String(e)
    errorVisible.value = true
  } finally {
    ctaLoading.value = false
  }
}

async function ghostAction() {
  if (ptype.value === PatientType.TidakAktif || ptype.value === PatientType.Error) {
    // "Hubungi petugas" — sementara no-op visual
    return
  }
  await patient.reset()
  router.push({ name: 'input', query: { mode: 'bpjs' } })
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
    case PatientType.Kontrol: return 'Buat surat layanan kontrol dan cetak'
    case PatientType.PostRANAP: return 'Pilih poli untuk kontrol'
    case PatientType.PostRAJAL: return 'Pilih poli tujuan'
    case PatientType.RujukanBaru: return 'Pilih dokter dan lanjutkan'
    case PatientType.TidakAktif: return 'Daftar sebagai pasien umum'
    default: return 'Hubungi petugas'
  }
})

// Ghost button label per kategori
const ghostLabel = computed(() => {
  if (ptype.value === PatientType.TidakAktif) return 'Hubungi petugas untuk bantuan'
  if (ptype.value === PatientType.Error) return 'Hubungi petugas'
  return 'Bukan saya — masukkan ulang'
})

// Info bar per kategori
const infoBar = computed(() => {
  switch (ptype.value) {
    case PatientType.MJKN:
      return { variant: 'success', text: 'Booking dari Mobile JKN terkonfirmasi. Cetak tiket untuk konfirmasi kedatangan.' }
    case PatientType.RujukanBaru:
      // Conditional: hanya kalau perluBiometrik
      if (perluBiometrik.value) {
        return { variant: 'warning', text: 'Verifikasi sidik jari diperlukan setelah pilih dokter.' }
      }
      return null
    case PatientType.TidakAktif:
      return {
        variant: 'danger',
        text: 'Status BPJS Anda saat ini tidak aktif. Hubungi BPJS Kesehatan untuk aktivasi, atau daftar sebagai pasien umum di RS ini.',
      }
    case PatientType.Error:
      return { variant: 'danger', text: 'Sistem tidak dapat memeriksa status Anda saat ini. Silakan hubungi petugas.' }
    default: return null
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

const showCTA = computed(() => ptype.value !== PatientType.Error)
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
        @click="ghostAction"
        :disabled="ctaLoading"
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
      <PatientCard
        v-if="peserta"
        :pill-label="pillConfig.label"
        :pill-variant="pillConfig.variant"
        :date-label="dateLabel"
        :nama="peserta.Nama"
        :no-kartu="peserta.NoKartu"
        :details="details"
      />
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

      <!-- Dokter picker untuk Kontrol -->
      <div v-if="ptype === PatientType.Kontrol" class="flex flex-col gap-[clamp(8px,1.2vw,10px)]">
        <p class="text-[clamp(11px,1.5vw,13px)] text-text-secondary font-medium uppercase tracking-wide">
          Pilih dokter
        </p>
        <DokterPicker v-model="selectedDokter" :list="dokterList" />
      </div>

      <!-- Info bar (conditional per kategori) -->
      <div
        v-if="infoBar"
        :class="['rounded-card border p-[clamp(10px,1.8vw,14px)] flex items-start gap-2', infoBarClass(infoBar.variant)]"
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

      <!-- CTA primary dengan loading state -->
      <button
        v-if="showCTA"
        type="button"
        :disabled="ctaLoading"
        :class="[
          'w-full rounded-kiosk transition-opacity active:opacity-85',
          'bg-blue text-white border border-blue',
          'px-[clamp(14px,2.5vw,20px)] py-[clamp(14px,2.5vw,20px)]',
          'text-[clamp(14px,2vw,17px)] font-medium',
          'flex items-center justify-between gap-3',
          'min-h-[clamp(56px,8vw,72px)]',
          'disabled:opacity-60 disabled:cursor-not-allowed',
        ]"
        @click="goNext"
      >
        <span class="flex items-center gap-2">
          <svg
            v-if="ctaLoading"
            class="animate-spin w-5 h-5"
            viewBox="0 0 24 24" fill="none"
          >
            <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3"
                    fill="none" stroke-dasharray="40" stroke-dashoffset="20" />
          </svg>
          {{ ctaLoading ? 'Memproses...' : ctaLabel }}
        </span>
        <svg
          v-if="!ctaLoading"
          xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
          stroke="currentColor" stroke-width="2.5" stroke-linecap="round"
          stroke-linejoin="round" class="w-5 h-5 shrink-0"
        >
          <polyline points="9 18 15 12 9 6" />
        </svg>
      </button>

      <!-- Ghost button -->
      <button
        type="button"
        :disabled="ctaLoading"
        class="w-full rounded-kiosk transition-colors
               bg-surface text-text-secondary border border-border
               hover:border-border-strong active:bg-bg
               px-[clamp(12px,2vw,16px)] py-[clamp(10px,1.8vw,14px)]
               text-[clamp(12px,1.6vw,14px)]
               disabled:opacity-50 disabled:cursor-not-allowed"
        @click="ghostAction"
      >
        {{ ghostLabel }}
      </button>
    </section>

    <IdleOverlay :seconds-left="secondsLeft" :visible="isCountingDown" />

    <!-- Error modal -->
    <AlertModal
      :visible="errorVisible"
      variant="error"
      title="Tidak dapat melanjutkan"
      :message="errorMsg"
      primary-label="Coba lagi"
      close-label="Tutup"
      @primary="errorVisible = false"
      @close="errorVisible = false"
    />
  </main>
</template>
