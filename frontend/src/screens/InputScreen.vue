<!--
  InputScreen v1.2 — dual-channel input layout (BB2 dari UX brainstorm).

  Layout (landscape kiosk monitor 22"+):
    +--------------------------------+
    | Header                         |
    +--------------------------------+
    |  TAP KARTU      |  KETIK       |
    |  [card icon     |  [Display]   |
    |   pulse anim]   |              |
    |                 |  [progress]  |
    |  Tempel kartu   |              |
    |  KTP / BPJS     |  [chip hints]|
    |  di reader      |              |
    |                 |  [NumPad 3x4]|
    |  ✓ Reader aktif |              |
    +--------------------------------+
    | [← Kembali]   [Panggil Petugas]|
    +--------------------------------+

  Setiap channel input setara visual weight — bukan banner kecil di atas
  numpad. Lansia yang punya kartu lebih sering tap (5 detik) vs ketik
  16 digit (60+ detik dengan typo risk).
-->
<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { PhCreditCard, PhPhone, PhCheckCircle, PhWarningCircle } from '@phosphor-icons/vue'

import NumPad from '../components/NumPad.vue'
import InputDisplay from '../components/InputDisplay.vue'
import IdleOverlay from '../components/IdleOverlay.vue'
import BackButton from '../components/BackButton.vue'
import BpjsLogo from '../components/BpjsLogo.vue'

import { I18N } from '../constants/i18n'
import { KIOSK } from '../constants/kiosk'
import { useIdleTimeout } from '../composables/useIdleTimeout'
import { useAudioCue } from '../composables/useAudioCue'
import { apmService } from '../services/apm'
import { usePatientStore } from '../stores/patient'

const route = useRoute()
const router = useRouter()
const patient = usePatientStore()
const audio = useAudioCue()

const input = ref(patient.input || '')
const fristaAvailable = ref(true)
const MIN_LENGTH = 6
const MAX_LENGTH = 16

const mode = computed(() => route.query.mode ?? 'bpjs')
const title = computed(() => {
  switch (mode.value) {
    case 'umum': return 'Pendaftaran Pasien Umum'
    case 'satusehat': return 'Aktivasi Satu Sehat'
    default: return 'Pendaftaran Pasien BPJS'
  }
})
const cardHint = computed(() => {
  switch (mode.value) {
    case 'umum': return 'Tap KTP elektronik di reader'
    case 'satusehat': return 'Tap KTP atau kartu BPJS di reader'
    default: return 'Tap kartu BPJS atau KTP di reader'
  }
})
const canSubmit = computed(() => input.value.length >= MIN_LENGTH)

async function refreshHardware() {
  try {
    const hw = await apmService.getHardwareStatus()
    fristaAvailable.value = hw.frista
  } catch {
    fristaAvailable.value = false
  }
}

function appendDigit(d) {
  if (input.value.length >= MAX_LENGTH) return
  input.value = input.value + d
}
function deleteDigit() {
  if (input.value.length === 0) return
  input.value = input.value.slice(0, -1)
}
function submit() {
  if (!canSubmit.value) return
  audio.tap()
  patient.input = input.value
  if (mode.value === 'umum') {
    router.push({ name: 'cari-pasien' })
  } else {
    router.push({ name: 'detect' })
  }
}
function back() {
  router.push({ name: 'home' })
}
function callStaff() {
  audio.notify()
  // TODO event 'staff:call'
}

// Keyboard support (DX bonus untuk dev di Mac)
function onKeydown(e) {
  if (e.key === 'Enter') {
    e.preventDefault()
    submit()
  } else if (e.key === 'Backspace') {
    e.preventDefault()
    deleteDigit()
  } else if (/^[0-9]$/.test(e.key)) {
    appendDigit(e.key)
  }
}

const { isCountingDown, secondsLeft } = useIdleTimeout({
  totalSeconds: KIOSK.idleTimeoutSec,
  countdownThreshold: KIOSK.idleCountdownSec,
  onTimeout: async () => {
    await patient.reset()
    router.push({ name: 'home' })
  },
})

onMounted(() => {
  refreshHardware()
  window.addEventListener('keydown', onKeydown)
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <main class="min-h-screen bg-bg flex flex-col">
    <!-- Header — kalau mode BPJS, tampilkan logo BPJS Kesehatan resmi -->
    <header
      class="bg-surface border-b border-border flex items-center justify-between
             px-[clamp(16px,3vw,28px)] py-[clamp(12px,1.8vw,16px)]"
    >
      <h1 class="text-[clamp(16px,2.2vw,20px)] font-semibold text-text-primary">
        {{ title }}
      </h1>
      <BpjsLogo
        v-if="mode === 'bpjs'"
        size="sm"
        variant="full"
      />
    </header>

    <!-- Body — dual-channel split -->
    <section
      class="flex-1 grid grid-cols-1 lg:grid-cols-2
             gap-[clamp(12px,2vw,20px)]
             p-[clamp(16px,3vw,24px)]
             max-w-[1100px] mx-auto w-full content-center"
    >
      <!-- ============ Channel 1: TAP KARTU ============ -->
      <div
        class="bg-surface border-2 rounded-card
               p-[clamp(20px,3vw,28px)]
               flex flex-col items-center justify-center
               text-center gap-[clamp(14px,2vw,18px)]
               min-h-[clamp(280px,38vw,440px)]
               transition-colors"
        :class="fristaAvailable ? 'border-emerald-200' : 'border-border'"
        :style="fristaAvailable ? { backgroundColor: 'var(--color-primary-light, #E8F0FE)' } : {}"
      >
        <div
          class="rounded-full flex items-center justify-center
                 w-[clamp(80px,11vw,120px)] h-[clamp(80px,11vw,120px)]
                 transition-all"
          :class="fristaAvailable ? 'animate-pulse' : 'opacity-40'"
          :style="{ backgroundColor: fristaAvailable ? 'var(--color-primary, #1B4FD8)' : '#9CA3AF', color: 'white' }"
        >
          <PhCreditCard :size="56" weight="fill" />
        </div>

        <div class="space-y-2">
          <div class="text-[clamp(20px,2.6vw,26px)] font-semibold text-text-primary leading-tight">
            {{ cardHint }}
          </div>
          <p class="text-[clamp(13px,1.7vw,15px)] text-text-secondary">
            Lebih cepat — tidak perlu mengetik
          </p>
        </div>

        <ul class="text-left text-[clamp(13px,1.6vw,15px)] text-text-secondary space-y-1">
          <li class="flex items-center gap-2">
            <span class="w-1.5 h-1.5 rounded-full bg-text-muted"></span>
            Kartu BPJS / JKN
          </li>
          <li class="flex items-center gap-2">
            <span class="w-1.5 h-1.5 rounded-full bg-text-muted"></span>
            KTP elektronik
          </li>
        </ul>

        <div
          class="text-[clamp(12px,1.5vw,14px)] font-medium flex items-center gap-2
                 px-3 py-1.5 rounded-tag"
          :class="fristaAvailable
            ? 'bg-emerald-100 text-emerald-700'
            : 'bg-amber-50 text-amber-700'"
        >
          <PhCheckCircle v-if="fristaAvailable" :size="16" weight="fill" />
          <PhWarningCircle v-else :size="16" weight="fill" />
          {{ fristaAvailable ? 'Reader aktif — silakan tap' : 'Reader tidak aktif — silakan ketik' }}
        </div>
      </div>

      <!-- ============ Channel 2: KETIK ============ -->
      <div
        class="bg-surface border border-border rounded-card
               p-[clamp(16px,2.5vw,24px)]
               flex flex-col gap-[clamp(12px,1.8vw,16px)]
               min-h-[clamp(280px,38vw,440px)]"
      >
        <div class="text-center">
          <div class="text-[clamp(16px,2.2vw,20px)] font-semibold text-text-primary">
            Atau ketik nomor di sini
          </div>
          <p class="text-[clamp(11px,1.4vw,13px)] text-text-muted mt-1">
            No. JKN / NIK KTP / No. Rekam Medis
          </p>
        </div>

        <InputDisplay :value="input" :max-length="MAX_LENGTH" />

        <!-- Progress text -->
        <p
          class="text-center text-[clamp(13px,1.6vw,15px)]"
          :class="canSubmit ? 'text-emerald-700 font-medium' : 'text-text-secondary'"
        >
          <template v-if="input.length === 0">
            Silakan ketik nomor Anda
          </template>
          <template v-else-if="canSubmit && input.length >= MAX_LENGTH">
            Sudah lengkap — tekan CARI
          </template>
          <template v-else-if="canSubmit">
            {{ input.length }} angka — siap cari
          </template>
          <template v-else>
            {{ input.length }} angka — minimal {{ MIN_LENGTH }}
          </template>
        </p>

        <NumPad
          :can-submit="canSubmit"
          @digit="appendDigit"
          @delete="deleteDigit"
          @submit="submit"
        />
      </div>
    </section>

    <!-- Footer: Back + Panggil Petugas safety net -->
    <footer
      class="bg-surface border-t border-border
             flex items-center justify-between
             px-[clamp(16px,3vw,28px)] py-[clamp(10px,1.5vw,14px)]"
    >
      <BackButton @click="back" />
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
    </footer>

    <IdleOverlay :seconds-left="secondsLeft" :visible="isCountingDown" />
  </main>
</template>
