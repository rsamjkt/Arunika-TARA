<!--
  TicketScreen — tampilan setelah antrian / SEP berhasil dibuat.

  Sumber data:
    patient.lastTicket  → dari AntrianScreen flow (jenis: antrian)
    patient.lastSEP     → dari ResultScreen Kontrol flow
    Bisa keduanya kalau path BPJS rujukan menghasilkan ticket + SEP.

  Spec P-045 layout (max-w 380, center):
    CheckCircle 44-50px hijau
    Success message
    Tiket paper:
      Label uppercase (mis. "ANTRIAN POLI PENYAKIT DALAM")
      Nomor besar clamp(44px,8vw,60px) font-medium
      Info dokter + tanggal muted
      Dashed divider
      No SEP monospace dengan nilai biru
    Info box bg #F5F6F8: lokasi tunggu + pengingat panggilan display
    Countdown 10s auto-back
    Tombol "Cetak ulang tiket" → Reprint(printHistoryID)
-->
<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import CheckCircle from '../components/CheckCircle.vue'
import AlertModal from '../components/AlertModal.vue'

import { I18N } from '../constants/i18n'
import { KIOSK } from '../constants/kiosk'
import { apmService } from '../services/apm'
import { usePatientStore } from '../stores/patient'

const router = useRouter()
const patient = usePatientStore()

const ticket = computed(() => patient.lastTicket)
const sep = computed(() => patient.lastSEP)

// Kalau buka /tiket tanpa data → balik ke home
if (!ticket.value && !sep.value) {
  router.replace({ name: 'home' })
}

const reprintLoading = ref(false)
const errorVisible = ref(false)
const errorMsg = ref('')

// Countdown auto-back
const secondsLeft = ref(KIOSK.ticketAutoBackSec)
let countdownTimer = null

async function backToHome() {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
  await patient.reset()
  router.push({ name: 'home' })
}

function startCountdown() {
  countdownTimer = setInterval(() => {
    secondsLeft.value--
    if (secondsLeft.value <= 0) {
      backToHome()
    }
  }, 1000)
}

// Reset countdown kalau user interaksi (mis. tap Cetak ulang)
function resetCountdown() {
  secondsLeft.value = KIOSK.ticketAutoBackSec
}

async function reprint() {
  if (reprintLoading.value) return
  if (!ticket.value?.PrintHistoryID || ticket.value.PrintHistoryID <= 0) {
    errorMsg.value = 'ID cetak tidak tersedia. Hubungi petugas untuk cetak ulang manual.'
    errorVisible.value = true
    return
  }
  reprintLoading.value = true
  resetCountdown()
  try {
    await apmService.reprint(ticket.value.PrintHistoryID)
  } catch (e) {
    errorMsg.value = `Gagal mencetak ulang. ${e?.message ?? ''}`.trim()
    errorVisible.value = true
  } finally {
    reprintLoading.value = false
  }
}

// Format tanggal "26 Apr 2026 14:30"
function formatDateTime(d) {
  if (!d) return ''
  try {
    const dt = new Date(d)
    if (isNaN(dt.getTime())) return ''
    const pad = (n) => String(n).padStart(2, '0')
    const BULAN = ['Jan','Feb','Mar','Apr','Mei','Jun','Jul','Agu','Sep','Okt','Nov','Des']
    return `${dt.getDate()} ${BULAN[dt.getMonth()]} ${dt.getFullYear()} ${pad(dt.getHours())}:${pad(dt.getMinutes())}`
  } catch { return '' }
}

// Compute label berdasarkan ticket jenis + sub
const ticketLabel = computed(() => {
  const t = ticket.value
  if (!t) return 'TIKET'
  switch (t.Jenis) {
    case 'POLI':
      return `ANTRIAN POLI ${t.NoPoli || ''}`.trim()
    case 'LOKET': {
      const sub = (t.SubJenis || '').replace(/_/g, ' ')
      return `ANTRIAN LOKET ${sub}`.trim()
    }
    case 'UMUM': {
      const sub = (t.SubJenis || '').replace(/_/g, ' ')
      return `ANTRIAN ${sub}`.trim()
    }
    default:
      return 'ANTRIAN'
  }
})

// Format nomor antrian (sudah Ticket.Nomor dari backend)
const ticketNomor = computed(() => ticket.value?.Nomor ?? '—')

// Info dokter + tanggal — ambil dari sep kalau ada, fallback waktu sekarang
const dokterLabel = computed(() => sep.value?.NmDokter || '—')
const tanggalLabel = computed(() => formatDateTime(ticket.value?.CreatedAt ?? new Date()))

// SEP info row
const noSEP = computed(() => sep.value?.NoSEP || '')

// Lokasi info — dari SEP poli atau ticket NoPoli
const lokasiPoli = computed(() => {
  if (sep.value?.NmPoli) return sep.value.NmPoli
  if (ticket.value?.NoPoli) return `Poli ${ticket.value.NoPoli}`
  return ''
})

onMounted(startCountdown)
onUnmounted(() => {
  if (countdownTimer) clearInterval(countdownTimer)
})
</script>

<template>
  <main class="min-h-screen bg-bg flex flex-col">
    <!-- Body center -->
    <section
      class="flex-1 flex flex-col items-center justify-center
             p-[clamp(16px,3vw,24px)] gap-[clamp(10px,1.6vw,14px)]
             max-w-[380px] mx-auto w-full text-center"
    >
      <!-- Sukses indicator -->
      <CheckCircle :size="50" />

      <p class="text-[clamp(12px,1.8vw,14px)] font-medium text-success">
        {{ sep ? 'Surat layanan berhasil dibuat' : 'Nomor antrian berhasil diambil' }}
      </p>

      <!-- Tiket paper -->
      <div
        class="bg-surface border border-border rounded-card w-full
               p-[clamp(18px,3vw,28px)]
               flex flex-col gap-[clamp(8px,1.4vw,12px)]"
      >
        <!-- Label uppercase -->
        <div class="text-[clamp(9px,1.2vw,11px)] uppercase tracking-[0.5px]
                    text-text-muted font-medium">
          {{ ticketLabel }}
        </div>

        <!-- Nomor antrian besar -->
        <div class="text-[clamp(44px,8vw,60px)] font-medium text-text-primary leading-none">
          {{ ticketNomor }}
        </div>

        <!-- Dokter + tanggal -->
        <div class="text-[clamp(11px,1.5vw,13px)] text-text-muted">
          <span v-if="sep">{{ dokterLabel }} · </span>
          {{ tanggalLabel }}
        </div>

        <!-- Offline badge kalau ticket dari mode offline -->
        <div
          v-if="ticket && ticket.IsOffline"
          class="inline-flex items-center gap-1 self-start
                 bg-warning-bg text-warning border border-warning-border rounded-tag
                 px-2 py-1 text-[clamp(9px,1.1vw,11px)] font-medium"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
            stroke="currentColor" stroke-width="2" stroke-linecap="round"
            stroke-linejoin="round" class="w-3 h-3"
          >
            <line x1="1" y1="1" x2="23" y2="23" />
            <path d="M16.72 11.06A10.94 10.94 0 0 1 19 12.55M5 12.55a10.94 10.94 0 0 1 5.17-2.39M10.71 5.05A16 16 0 0 1 22.58 9M1.42 9a15.91 15.91 0 0 1 4.7-2.88M8.53 16.11a6 6 0 0 1 6.95 0" />
            <line x1="12" y1="20" x2="12.01" y2="20" />
          </svg>
          OFFLINE — akan disinkronkan
        </div>

        <!-- Dashed divider -->
        <hr v-if="noSEP" class="border-t border-dashed border-border my-1" />

        <!-- No SEP -->
        <div v-if="noSEP" class="flex items-center justify-between gap-2">
          <span class="text-[clamp(10px,1.3vw,12px)] text-text-muted font-mono">
            No. SEP
          </span>
          <span class="text-[clamp(11px,1.5vw,13px)] font-mono font-medium text-blue">
            {{ noSEP }}
          </span>
        </div>
      </div>

      <!-- Info box -->
      <div
        class="w-full bg-bg rounded-[9px] p-[clamp(12px,2vw,16px)]
               text-[clamp(10px,1.4vw,12px)] leading-[1.7] text-text-secondary"
      >
        <p>Silakan menuju area tunggu</p>
        <p v-if="lokasiPoli" class="font-medium text-text-primary mt-1">
          {{ lokasiPoli }}
        </p>
        <p class="mt-1">Nomor Anda akan dipanggil di layar display</p>
      </div>

      <!-- Countdown -->
      <p class="text-[clamp(10px,1.3vw,12px)] text-text-muted">
        Kembali ke awal dalam {{ secondsLeft }} detik
      </p>

      <!-- Cetak ulang button -->
      <button
        v-if="ticket && ticket.PrintHistoryID > 0"
        type="button"
        :disabled="reprintLoading"
        class="w-full rounded-btn bg-surface border border-border
               text-text-primary font-medium
               px-4 py-[clamp(10px,1.6vw,12px)]
               text-[clamp(12px,1.6vw,14px)]
               hover:border-border-strong active:bg-bg
               disabled:opacity-60 disabled:cursor-not-allowed
               flex items-center justify-center gap-2"
        @click="reprint"
      >
        <svg
          v-if="reprintLoading"
          class="animate-spin w-4 h-4"
          viewBox="0 0 24 24" fill="none"
        >
          <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3"
                  fill="none" stroke-dasharray="40" stroke-dashoffset="20" />
        </svg>
        <svg
          v-else
          xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
          stroke="currentColor" stroke-width="2" stroke-linecap="round"
          stroke-linejoin="round" class="w-4 h-4"
        >
          <polyline points="6 9 6 2 18 2 18 9" />
          <path d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2" />
          <rect x="6" y="14" width="12" height="8" />
        </svg>
        {{ reprintLoading ? 'Mencetak...' : I18N.common.reprint + ' tiket' }}
      </button>

      <!-- Tombol "Selesai" untuk back manual sebelum countdown -->
      <button
        type="button"
        class="text-[clamp(11px,1.5vw,13px)] text-text-secondary
               hover:text-text-primary px-2 py-1"
        @click="backToHome"
      >
        {{ I18N.common.done }}
      </button>
    </section>

    <!-- Error modal -->
    <AlertModal
      :visible="errorVisible"
      variant="error"
      title="Tidak dapat mencetak"
      :message="errorMsg"
      primary-label="Tutup"
      close-label="Batal"
      @primary="errorVisible = false"
      @close="errorVisible = false"
    />
  </main>
</template>
