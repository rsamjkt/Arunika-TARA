<!--
  HomeScreen — landing kiosk APM (T.A.R.A) v1.1 "Mahatma".

  Layout baru (post BB3 + C16):
    - Header: logo (dari config.toml) + nama RS + status dots + jam digital
    - Welcome banner BESAR: greeting time-aware + 1 ilustrasi medis SVG
    - Hero "Pasien BPJS" (60% visual weight — mayoritas user RS pemerintah)
    - 2 Secondary cards setara: "Pasien Umum" + "Antrian Loket"
    - Aktivasi Satu Sehat → footer link kecil (niche action)
    - Footer: "Pertama kali? Bantu Saya" + "Panggil Petugas"
-->
<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import {
  PhUser,
  PhTicket,
  PhFingerprint,
  PhLifebuoy,
  PhPhone,
  PhSparkle,
  PhCircleNotch,
  PhCaretRight,
} from '@phosphor-icons/vue'

import StatusDot from '../components/StatusDot.vue'
import IdleOverlay from '../components/IdleOverlay.vue'
import BpjsLogo from '../components/BpjsLogo.vue'

import { I18N } from '../constants/i18n'
import { KIOSK } from '../constants/kiosk'
import { useClock } from '../composables/useClock'
import { useIdleTimeout } from '../composables/useIdleTimeout'
import { useAudioCue } from '../composables/useAudioCue'
import { apmService } from '../services/apm'
import { usePatientStore } from '../stores/patient'
import { useBrandingStore } from '../stores/branding'

const router = useRouter()
const patient = usePatientStore()
const branding = useBrandingStore()
const audio = useAudioCue()
const { time, date } = useClock()

// Status dots
const bpjsStatus = ref('online')
const sistemStatus = ref('online')
const antrianLoading = ref(false)

async function refreshStatus() {
  try {
    const sys = await apmService.getSystemStatus()
    bpjsStatus.value = sys.Online ? 'online' : 'offline'
    sistemStatus.value = sys.Hardware?.Frista || sys.Hardware?.Printer ? 'online' : 'warning'
  } catch {
    // backend belum siap — biarkan default
  }
}

// Greeting time-aware
const greeting = computed(() => {
  const h = new Date().getHours()
  if (h < 11) return 'Selamat pagi!'
  if (h < 15) return 'Selamat siang!'
  if (h < 18) return 'Selamat sore!'
  return 'Selamat malam!'
})

// Idle timeout — di home, reset cuma clear store
const { isCountingDown, secondsLeft } = useIdleTimeout({
  totalSeconds: KIOSK.idleTimeoutSec,
  countdownThreshold: KIOSK.idleCountdownSec,
  onTimeout: async () => {
    await patient.reset()
  },
})

// Click handlers — semua dengan audio cue + reset session
async function startBPJS() {
  audio.tap()
  await patient.reset()
  router.push({ name: 'input', query: { mode: 'bpjs' } })
}
async function startUmum() {
  audio.tap()
  await patient.reset()
  router.push({ name: 'input', query: { mode: 'umum' } })
}
async function startAntrian() {
  if (antrianLoading.value) return
  audio.tap()
  antrianLoading.value = true
  try {
    await patient.reset()
    const ticket = await apmService.createAntrian('LOKET', 'WALKIN')
    patient.setLastTicket(ticket)
    audio.success()
    router.push({ name: 'tiket' })
  } catch (e) {
    audio.error()
    alert('Gagal mengambil nomor antrian. Silakan hubungi petugas.')
  } finally {
    antrianLoading.value = false
  }
}
async function startSatuSehat() {
  audio.tap()
  await patient.reset()
  router.push({ name: 'input', query: { mode: 'satusehat' } })
}
function startBantuSaya() {
  audio.tap()
  // Phase 4 — Bantu Saya wizard. Sementara navigate ke /input mode=bpjs
  // sebagai fallback. Akan diganti ke /bantu-saya saat Phase 4 siap.
  router.push({ name: 'input', query: { mode: 'bpjs' } })
}
function callStaff() {
  audio.notify()
  // TODO: emit Wails event 'staff:call' untuk admin panel notification
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
      <!-- Kiri: logo (dari config) atau fallback "T" mark + nama RS + tagline -->
      <div class="flex items-center gap-[clamp(10px,1.5vw,14px)] min-w-0">
        <img
          v-if="branding.logoDataURL"
          :src="branding.logoDataURL"
          :alt="branding.hospitalName"
          class="object-contain flex-shrink-0
                 w-[clamp(36px,5vw,44px)] h-[clamp(36px,5vw,44px)]"
        />
        <div
          v-else
          class="rounded-[8px] flex items-center justify-center flex-shrink-0
                 w-[clamp(36px,5vw,44px)] h-[clamp(36px,5vw,44px)]"
          style="background-color: var(--color-primary, #1B4FD8)"
        >
          <span class="text-white font-bold text-[clamp(18px,2.5vw,22px)] leading-none">T</span>
        </div>
        <div class="min-w-0">
          <div class="text-[clamp(14px,1.9vw,17px)] font-semibold text-text-primary leading-tight truncate">
            {{ branding.hospitalName }}
          </div>
          <div class="text-[clamp(11px,1.3vw,13px)] text-text-muted leading-tight">
            {{ branding.hospitalTagline }}
          </div>
        </div>
      </div>

      <!-- Kanan: status + jam digital -->
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
          <div class="text-[clamp(15px,2vw,18px)] font-semibold text-text-primary tabular-nums leading-tight">
            {{ time }}
          </div>
          <div class="text-[clamp(10px,1.2vw,12px)] text-text-muted leading-tight">
            {{ date }}
          </div>
        </div>
      </div>
    </header>

    <!-- ============ Body ============ -->
    <section
      class="flex-1 flex flex-col gap-[clamp(12px,2vw,18px)]
             p-[clamp(16px,3vw,24px)]
             max-w-[820px] mx-auto w-full justify-center"
    >
      <!-- Welcome banner besar dengan greeting time-aware + ilustrasi -->
      <div
        class="rounded-card flex items-center gap-[clamp(16px,3vw,24px)]
               px-[clamp(20px,3vw,32px)] py-[clamp(20px,3vw,28px)]
               min-h-[clamp(110px,14vw,150px)]"
        :style="{
          background: `linear-gradient(135deg, var(--color-primary-light, #E8F0FE) 0%, var(--color-accent, #DBEAFE) 100%)`,
        }"
      >
        <div class="flex-1">
          <div class="text-[clamp(24px,3.6vw,34px)] font-semibold text-text-primary leading-tight">
            {{ greeting }}
          </div>
          <p class="text-[clamp(14px,2vw,18px)] text-text-secondary mt-2 leading-relaxed">
            Mari mulai pendaftaran Anda — pilih layanan di bawah.
          </p>
        </div>
        <div
          class="flex-shrink-0 hidden sm:flex items-center justify-center
                 w-[clamp(80px,11vw,120px)] h-[clamp(80px,11vw,120px)]
                 rounded-full bg-white/40 backdrop-blur-sm"
        >
          <PhSparkle :size="48" weight="duotone" :style="{ color: 'var(--color-primary, #1B4FD8)' }" />
        </div>
      </div>

      <!-- Hero BPJS — primary action 60% visual weight + logo BPJS resmi -->
      <button
        type="button"
        class="text-left rounded-card transition-all
               border border-transparent
               hover:opacity-95 active:opacity-90
               flex items-center gap-[clamp(14px,2.5vw,20px)]
               px-[clamp(20px,3vw,28px)] py-[clamp(20px,3vw,28px)]
               min-h-[clamp(110px,14vw,140px)]
               shadow-md text-white"
        :style="{ backgroundColor: 'var(--color-primary, #1B4FD8)' }"
        @click="startBPJS"
      >
        <!-- Logo BPJS Kesehatan resmi (atau dari config kalau ada) -->
        <div
          class="bg-white rounded-[12px] p-[clamp(10px,1.5vw,14px)] flex-shrink-0
                 flex items-center justify-center
                 w-[clamp(72px,10vw,96px)] h-[clamp(72px,10vw,96px)]"
        >
          <BpjsLogo size="md" variant="icon" />
        </div>
        <div class="flex-1">
          <div class="text-[clamp(20px,2.8vw,26px)] font-semibold leading-tight">
            Pasien BPJS
          </div>
          <p class="text-[clamp(13px,1.7vw,15px)] opacity-90 mt-1 leading-snug">
            Tap kartu BPJS atau ketik nomor — sistem otomatis mendeteksi jenis kunjungan
          </p>
        </div>
        <PhCaretRight :size="32" weight="bold" class="opacity-80" />
      </button>

      <!-- 2 Secondary cards: Umum + Antrian Loket -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-[clamp(10px,1.5vw,14px)]">
        <!-- Pasien Umum -->
        <button
          type="button"
          class="text-left rounded-card bg-surface border border-border
                 hover:border-border-strong active:bg-bg
                 flex items-center gap-[clamp(12px,2vw,16px)]
                 px-[clamp(16px,2.5vw,22px)] py-[clamp(16px,2.5vw,22px)]
                 min-h-[clamp(86px,11vw,108px)] shadow-sm"
          @click="startUmum"
        >
          <div
            class="rounded-[10px] flex items-center justify-center flex-shrink-0
                   w-[clamp(48px,6vw,56px)] h-[clamp(48px,6vw,56px)]"
            style="background-color: var(--color-primary-light, #E8F0FE); color: var(--color-primary, #1B4FD8)"
          >
            <PhUser :size="28" weight="bold" />
          </div>
          <div class="flex-1 min-w-0">
            <div class="text-[clamp(15px,2.1vw,18px)] font-semibold text-text-primary leading-tight">
              Pasien Umum
            </div>
            <p class="text-[clamp(11px,1.5vw,13px)] text-text-secondary mt-1 leading-tight">
              Tanpa kartu BPJS
            </p>
          </div>
        </button>

        <!-- Antrian Loket (single-tap, QW1) -->
        <button
          type="button"
          :disabled="antrianLoading"
          class="text-left rounded-card bg-surface border border-border
                 hover:border-border-strong active:bg-bg disabled:opacity-60
                 flex items-center gap-[clamp(12px,2vw,16px)]
                 px-[clamp(16px,2.5vw,22px)] py-[clamp(16px,2.5vw,22px)]
                 min-h-[clamp(86px,11vw,108px)] shadow-sm"
          @click="startAntrian"
        >
          <div
            class="rounded-[10px] flex items-center justify-center flex-shrink-0
                   w-[clamp(48px,6vw,56px)] h-[clamp(48px,6vw,56px)]
                   bg-amber-50 text-amber-700"
          >
            <PhCircleNotch v-if="antrianLoading" :size="28" weight="bold" class="animate-spin" />
            <PhTicket v-else :size="28" weight="fill" />
          </div>
          <div class="flex-1 min-w-0">
            <div class="text-[clamp(15px,2.1vw,18px)] font-semibold text-text-primary leading-tight">
              Ambil Nomor Loket
            </div>
            <p class="text-[clamp(11px,1.5vw,13px)] text-text-secondary mt-1 leading-tight">
              {{ antrianLoading ? 'Mengambil nomor…' : 'Antrian admisi langsung' }}
            </p>
          </div>
        </button>
      </div>
    </section>

    <!-- ============ Footer ============ -->
    <footer
      class="bg-surface border-t border-border
             flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-2
             px-[clamp(16px,3vw,28px)] py-[clamp(10px,1.8vw,14px)]"
    >
      <button
        type="button"
        class="flex items-center gap-2 text-[clamp(13px,1.7vw,15px)] font-medium
               px-[clamp(12px,1.8vw,16px)] py-[clamp(8px,1.3vw,12px)]
               rounded-btn hover:bg-bg active:bg-border"
        :style="{ color: 'var(--color-primary, #1B4FD8)' }"
        @click="startBantuSaya"
      >
        <PhLifebuoy :size="20" weight="bold" />
        <span>Pertama kali? Bantu saya</span>
      </button>

      <div class="flex items-center gap-3">
        <button
          type="button"
          class="flex items-center gap-2 text-[clamp(12px,1.5vw,14px)] text-text-muted
                 hover:text-text-primary px-3 py-2"
          @click="startSatuSehat"
        >
          <PhFingerprint :size="18" weight="bold" />
          <span class="hidden sm:inline">Aktivasi Satu Sehat</span>
        </button>
        <button
          type="button"
          class="flex items-center gap-2 text-[clamp(13px,1.7vw,15px)] font-medium
                 px-[clamp(12px,1.8vw,16px)] py-[clamp(10px,1.5vw,12px)]
                 rounded-btn hover:bg-bg active:bg-border
                 min-h-[clamp(48px,6vw,56px)]"
          :style="{ color: 'var(--color-primary, #1B4FD8)' }"
          @click="callStaff"
        >
          <PhPhone :size="20" weight="bold" />
          <span>Panggil petugas</span>
        </button>
      </div>
    </footer>

    <IdleOverlay :seconds-left="secondsLeft" :visible="isCountingDown" />
  </main>
</template>
