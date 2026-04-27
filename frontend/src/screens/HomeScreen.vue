<!--
  HomeScreen — entry point kiosk APM (T.A.R.A).

  Layout (sesuai DESIGN_SYSTEM.md spec ketat):
    - Background #F5F6F8
    - Header putih, border-bottom 0.5px #E4E6EA
        kiri: logo mark biru 30x30 + nama RS + tagline
        kanan: status dots (BPJS + Sistem) + jam digital live
    - Body: 4 button area, padding clamp(12px,2.5vw,20px)
        Hero BPJS (full width, primary)
        Grid 2 col: Pasien Umum + Ambil Antrian
        Hero Aktivasi Satu Sehat (full width, secondary style)
    - Footer border-top: "Butuh bantuan?" + tombol "Panggil petugas"

  Idle timeout: 60 detik → reset ke /. 10 detik terakhir muncul
  IdleOverlay dengan countdown.

  Frista event listener di-handle global di App.vue (auto-redirect
  ke /detect kalau di home/input).
-->
<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import StatusDot from '../components/StatusDot.vue'
import HeroButton from '../components/HeroButton.vue'
import SecondaryCard from '../components/SecondaryCard.vue'
import IdleOverlay from '../components/IdleOverlay.vue'

import { I18N } from '../constants/i18n'
import { KIOSK } from '../constants/kiosk'
import { useClock } from '../composables/useClock'
import { useIdleTimeout } from '../composables/useIdleTimeout'
import { apmService } from '../services/apm'
import { usePatientStore } from '../stores/patient'

const router = useRouter()
const patient = usePatientStore()
const { time, date } = useClock()

// Status dots — refresh dari backend setiap mount + event hardware:status
const bpjsStatus = ref('online')
const sistemStatus = ref('online')

async function refreshStatus() {
  try {
    const sys = await apmService.getSystemStatus()
    bpjsStatus.value = sys.Online ? 'online' : 'offline'
    sistemStatus.value = sys.Hardware?.Frista || sys.Hardware?.Printer ? 'online' : 'warning'
  } catch {
    // Backend belum siap di first load — biarkan default 'online'
  }
}

// Idle timeout — auto-reset ke home (di home, reset hanya clear store)
const { isCountingDown, secondsLeft } = useIdleTimeout({
  totalSeconds: KIOSK.idleTimeoutSec,
  countdownThreshold: KIOSK.idleCountdownSec,
  onTimeout: async () => {
    await patient.reset()
    // Sudah di home — cukup reload state, tidak perlu navigate
  },
})

// Click handlers
async function startBPJS() {
  await patient.reset()
  router.push({ name: 'input', query: { mode: 'bpjs' } })
}
async function startUmum() {
  await patient.reset()
  router.push({ name: 'input', query: { mode: 'umum' } })
}
// QW1: Antrian Loket = single-tap → langsung create Loket ticket + navigate ke /tiket
const antrianLoading = ref(false)
async function startAntrian() {
  if (antrianLoading.value) return
  antrianLoading.value = true
  try {
    await patient.reset()
    const ticket = await apmService.createAntrian('LOKET', 'WALKIN')
    patient.setLastTicket(ticket)
    router.push({ name: 'tiket' })
  } catch (e) {
    // fallback: kalau gagal, balik ke /antrian (kalau masih ada) atau alert
    alert('Gagal mengambil nomor antrian. Silakan hubungi petugas.')
  } finally {
    antrianLoading.value = false
  }
}
async function startSatuSehat() {
  await patient.reset()
  router.push({ name: 'input', query: { mode: 'satusehat' } })
}
function callStaff() {
  // P-046 admin panel akan handle. Sementara ini no-op visual.
  // Bisa juga emit event "staff:call" yang Wails App handle (audible alert).
}

onMounted(refreshStatus)
</script>

<template>
  <main class="min-h-screen bg-bg flex flex-col">
    <!-- ============ Header ============ -->
    <header
      class="bg-surface border-b border-border
             flex items-center justify-between
             px-[clamp(16px,3vw,28px)] py-[clamp(10px,1.8vw,16px)]"
    >
      <!-- Kiri: logo + nama RS + tagline -->
      <div class="flex items-center gap-[clamp(10px,1.5vw,14px)] min-w-0">
        <div
          class="bg-blue rounded-[8px] flex items-center justify-center flex-shrink-0
                 w-[clamp(28px,4vw,34px)] h-[clamp(28px,4vw,34px)]"
        >
          <!-- Logo mark: huruf "T" untuk T.A.R.A -->
          <span class="text-white font-bold text-[clamp(15px,2vw,18px)] leading-none">
            T
          </span>
        </div>
        <div class="min-w-0">
          <div class="text-[clamp(13px,1.8vw,16px)] font-medium text-text-primary leading-tight truncate">
            {{ KIOSK.defaultRSName }}
          </div>
          <div class="text-[clamp(10px,1.2vw,12px)] text-text-muted leading-tight">
            {{ I18N.app.tagline }}
          </div>
        </div>
      </div>

      <!-- Kanan: status + jam -->
      <div class="flex items-center gap-[clamp(8px,1.5vw,14px)] flex-shrink-0">
        <StatusDot
          :label="`${I18N.status.bpjs} ${bpjsStatus === 'online' ? I18N.status.online : I18N.status.offline}`"
          :variant="bpjsStatus"
        />
        <StatusDot
          :label="`${I18N.status.sistem} ${sistemStatus === 'online' ? I18N.status.online : I18N.status.warning}`"
          :variant="sistemStatus"
        />
        <div class="hidden sm:flex flex-col items-end">
          <div class="text-[clamp(13px,1.8vw,16px)] font-medium text-text-primary tabular-nums leading-tight">
            {{ time }}
          </div>
          <div class="text-[clamp(9px,1.1vw,11px)] text-text-muted leading-tight">
            {{ date }}
          </div>
        </div>
      </div>
    </header>

    <!-- ============ Body — 4 button area ============ -->
    <section
      class="flex-1 flex flex-col gap-[clamp(10px,2vw,16px)]
             p-[clamp(12px,2.5vw,20px)]
             max-w-[680px] mx-auto w-full justify-center"
    >
      <!-- Welcome heading -->
      <div class="mb-[clamp(8px,1.5vw,12px)]">
        <h1 class="text-[clamp(20px,3.5vw,28px)] font-medium text-text-primary leading-tight">
          Selamat datang
        </h1>
        <p class="text-[clamp(11px,1.6vw,14px)] text-text-secondary mt-1">
          Pilih layanan yang Anda butuhkan
        </p>
      </div>

      <!-- Hero BPJS -->
      <HeroButton
        :title="I18N.home.bpjs.title"
        :subtitle="I18N.home.bpjs.subtitle"
        :tag="I18N.home.bpjs.tag"
        @click="startBPJS"
      >
        <template #icon>
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            class="w-[clamp(20px,3vw,26px)] h-[clamp(20px,3vw,26px)]"
          >
            <rect x="2" y="5" width="20" height="14" rx="2" />
            <line x1="2" y1="10" x2="22" y2="10" />
            <line x1="6" y1="15" x2="10" y2="15" />
          </svg>
        </template>
      </HeroButton>

      <!-- Grid 2 kolom: Umum + Antrian -->
      <div class="grid grid-cols-2 gap-[clamp(8px,1.5vw,12px)]">
        <SecondaryCard
          :title="I18N.home.umum.title"
          :subtitle="I18N.home.umum.subtitle"
          variant="blue"
          @click="startUmum"
        >
          <template #icon>
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              class="w-[clamp(16px,2.4vw,20px)] h-[clamp(16px,2.4vw,20px)]"
            >
              <circle cx="12" cy="8" r="4" />
              <path d="M4 21v-2a4 4 0 0 1 4-4h8a4 4 0 0 1 4 4v2" />
            </svg>
          </template>
        </SecondaryCard>

        <SecondaryCard
          :title="I18N.home.antrian.title"
          :subtitle="I18N.home.antrian.subtitle"
          variant="gray"
          @click="startAntrian"
        >
          <template #icon>
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              class="w-[clamp(16px,2.4vw,20px)] h-[clamp(16px,2.4vw,20px)]"
            >
              <line x1="8" y1="6" x2="21" y2="6" />
              <line x1="8" y1="12" x2="21" y2="12" />
              <line x1="8" y1="18" x2="21" y2="18" />
              <circle cx="4" cy="6" r="1" />
              <circle cx="4" cy="12" r="1" />
              <circle cx="4" cy="18" r="1" />
            </svg>
          </template>
        </SecondaryCard>
      </div>

      <!-- Aktivasi Satu Sehat (full width, gaya secondary) -->
      <button
        type="button"
        class="w-full text-left rounded-card transition-colors
               bg-surface border border-border hover:border-border-strong active:bg-[#FAFBFC]
               flex items-center gap-[clamp(10px,2vw,16px)]
               p-[clamp(14px,2.2vw,18px)]
               min-h-[clamp(64px,9vw,84px)]"
        @click="startSatuSehat"
      >
        <div
          class="bg-emerald-50 text-emerald-700 rounded-[10px] flex items-center justify-center flex-shrink-0
                 w-[clamp(38px,5.5vw,48px)] h-[clamp(38px,5.5vw,48px)]"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
            stroke-linejoin="round"
            class="w-[clamp(18px,2.6vw,22px)] h-[clamp(18px,2.6vw,22px)]"
          >
            <rect x="5" y="2" width="14" height="20" rx="2" />
            <line x1="12" y1="18" x2="12" y2="18" />
          </svg>
        </div>
        <div class="flex-1">
          <div class="text-[clamp(13px,2vw,16px)] font-medium text-text-primary leading-tight">
            {{ I18N.home.satusehat.title }}
          </div>
          <div class="text-[clamp(10px,1.4vw,13px)] text-text-muted mt-1 leading-tight">
            {{ I18N.home.satusehat.subtitle }}
          </div>
        </div>
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
          class="w-[clamp(14px,2vw,18px)] h-[clamp(14px,2vw,18px)] text-text-muted"
        >
          <polyline points="9 18 15 12 9 6" />
        </svg>
      </button>
    </section>

    <!-- ============ Footer ============ -->
    <footer
      class="bg-surface border-t border-border
             flex items-center justify-between
             px-[clamp(16px,3vw,28px)] py-[clamp(8px,1.5vw,12px)]"
    >
      <span class="text-[clamp(11px,1.4vw,13px)] text-text-secondary">
        {{ I18N.footer.needHelp }}
      </span>
      <button
        type="button"
        class="text-[clamp(11px,1.4vw,13px)] font-medium text-blue
               px-[clamp(10px,1.5vw,14px)] py-[clamp(6px,1vw,8px)]
               rounded-btn hover:bg-blue-light active:bg-blue-light/80
               flex items-center gap-2"
        @click="callStaff"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          class="w-[clamp(14px,1.8vw,16px)] h-[clamp(14px,1.8vw,16px)]"
        >
          <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z" />
        </svg>
        {{ I18N.footer.callStaff }}
      </button>
    </footer>

    <!-- Idle countdown overlay (visible saat 10 detik terakhir) -->
    <IdleOverlay :seconds-left="secondsLeft" :visible="isCountingDown" />
  </main>
</template>
