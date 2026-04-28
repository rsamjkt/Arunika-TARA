<!--
  InputScreen v1.3 — single-channel input layout.

  History: v1.2 punya dual-channel "Tap Kartu" + "Ketik". Setelah audit
  ke RS Anggrek Mas, ternyata RS tidak punya card reader sama sekali —
  "Frista" yang di-anggap sebelumnya adalah aplikasi BPJS sidik wajah,
  bukan card reader. Channel "Tap Kartu" dihapus; pasien input via
  NumPad saja. Verifikasi biometrik (wajah/sidik jari) hanya muncul
  saat SEP butuh, via BiometrikChoiceModal di flow SEP.

  Layout (landscape kiosk monitor 22"+):
    +--------------------------------+
    | Header                         |
    +--------------------------------+
    |  KETIK NIK / NO. KARTU BPJS    |
    |  [Display]                     |
    |  [progress hint]               |
    |  [NumPad 3x4]                  |
    +--------------------------------+
    | [← Kembali]   [Panggil Petugas]|
    +--------------------------------+
-->
<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { PhPhone, PhKeyboard } from '@phosphor-icons/vue'

import NumPad from '../components/NumPad.vue'
import InputDisplay from '../components/InputDisplay.vue'
import IdleOverlay from '../components/IdleOverlay.vue'
import BackButton from '../components/BackButton.vue'
import BpjsLogo from '../components/BpjsLogo.vue'

import { I18N } from '../constants/i18n'
import { KIOSK } from '../constants/kiosk'
import { useIdleTimeout } from '../composables/useIdleTimeout'
import { useAudioCue } from '../composables/useAudioCue'
import { usePatientStore } from '../stores/patient'

const route = useRoute()
const router = useRouter()
const patient = usePatientStore()
const audio = useAudioCue()

const input = ref(patient.input || '')
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
// Hint utama: instruksi singkat untuk pasien — apa yang harus diketik.
const inputHint = computed(() => {
  switch (mode.value) {
    case 'umum': return 'Ketik NIK KTP atau No. Rekam Medis untuk mulai'
    case 'satusehat': return 'Ketik NIK atau No. Kartu BPJS untuk mulai'
    default: return 'Ketik NIK atau No. Kartu BPJS untuk mulai'
  }
})
const canSubmit = computed(() => input.value.length >= MIN_LENGTH)

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

    <!-- Body — single-channel keypad -->
    <section
      class="flex-1 flex flex-col items-center justify-center
             gap-[clamp(14px,2vw,20px)]
             p-[clamp(16px,3vw,24px)]
             max-w-[640px] mx-auto w-full"
    >
      <!-- Hint header -->
      <div
        class="bg-surface border border-border rounded-card
               px-[clamp(20px,3vw,28px)] py-[clamp(16px,2.5vw,22px)]
               flex items-center gap-[clamp(12px,2vw,16px)] w-full"
      >
        <div
          class="rounded-full flex items-center justify-center flex-shrink-0
                 w-[clamp(48px,6.5vw,64px)] h-[clamp(48px,6.5vw,64px)]
                 text-white"
          :style="{ backgroundColor: 'var(--color-primary, #1B4FD8)' }"
        >
          <PhKeyboard :size="32" weight="fill" />
        </div>
        <div class="flex-1 min-w-0">
          <div class="text-[clamp(17px,2.3vw,20px)] font-semibold text-text-primary leading-tight">
            {{ inputHint }}
          </div>
          <p class="text-[clamp(11px,1.4vw,13px)] text-text-muted mt-1 leading-tight">
            No. JKN / NIK KTP / No. Rekam Medis &middot; minimal {{ MIN_LENGTH }} angka
          </p>
        </div>
      </div>

      <!-- Display + numpad -->
      <div
        class="bg-surface border border-border rounded-card
               p-[clamp(16px,2.5vw,24px)]
               flex flex-col gap-[clamp(12px,1.8vw,16px)] w-full"
      >
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
