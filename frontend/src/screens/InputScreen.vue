<!--
  InputScreen — input identitas pasien (NIK / No Kartu BPJS / No RM).

  Layout:
    Header: back button + title (mode-aware: BPJS/Umum/Satu Sehat)
    Display: InputDisplay component (4-digit grouping, blink cursor)
    Frista bar: status banner (available/not)
    Chip hints: "16 digit no. JKN", "16 digit NIK KTP", "No. rekam medis"
    NumPad: 3x4 grid digit + hapus + cari
    Idle overlay aktif

  Submit flow:
    1. Validasi minimal 6 karakter
    2. patient.input = current value
    3. router.push '/detect'
    4. DetectScreen yang trigger detect call
-->
<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import NumPad from '../components/NumPad.vue'
import InputDisplay from '../components/InputDisplay.vue'
import FristaBar from '../components/FristaBar.vue'
import IdleOverlay from '../components/IdleOverlay.vue'

import { I18N } from '../constants/i18n'
import { KIOSK } from '../constants/kiosk'
import { useIdleTimeout } from '../composables/useIdleTimeout'
import { apmService } from '../services/apm'
import { usePatientStore } from '../stores/patient'

const route = useRoute()
const router = useRouter()
const patient = usePatientStore()

const input = ref(patient.input || '')
const fristaAvailable = ref(true)
const MIN_LENGTH = 6
const MAX_LENGTH = 16

// Mode aware title
const mode = computed(() => route.query.mode ?? 'bpjs')
const title = computed(() => {
  switch (mode.value) {
    case 'umum': return I18N.home.umum.title
    case 'satusehat': return I18N.home.satusehat.title
    default: return I18N.home.bpjs.title
  }
})
const canSubmit = computed(() => input.value.length >= MIN_LENGTH)

// Refresh frista status onMounted
async function refreshHardware() {
  try {
    const hw = await apmService.getHardwareStatus()
    fristaAvailable.value = hw.Frista
  } catch {
    fristaAvailable.value = false
  }
}

// Handlers
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
  patient.input = input.value
  // Mode-aware routing:
  //   bpjs / satusehat → Smart Detector (VClaim)
  //   umum            → Search pasien di Khanza langsung (no VClaim)
  if (mode.value === 'umum') {
    router.push({ name: 'cari-pasien' })
  } else {
    router.push({ name: 'detect' })
  }
}
function back() {
  router.push({ name: 'home' })
}

// Keyboard support (Enter, Backspace, digit) — DX bonus untuk dev di Mac
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

// Idle timeout
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
        @click="back"
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
        {{ title }}
      </h1>
    </header>

    <!-- Body -->
    <section
      class="flex-1 flex flex-col gap-[clamp(10px,2vw,16px)]
             p-[clamp(12px,2.5vw,20px)]
             max-w-[560px] mx-auto w-full justify-center"
    >
      <!-- Display -->
      <InputDisplay :value="input" :max-length="MAX_LENGTH" />

      <!-- QW5: Progress text "X dari N angka" — kasih sense of progress untuk lansia -->
      <p
        class="text-center text-[clamp(13px,1.7vw,15px)]"
        :class="canSubmit ? 'text-emerald-700 font-medium' : 'text-text-secondary'"
      >
        <template v-if="input.length === 0">
          Silakan ketik nomor Anda
        </template>
        <template v-else-if="canSubmit && input.length >= MAX_LENGTH">
          Sudah lengkap — silakan tekan CARI
        </template>
        <template v-else-if="canSubmit">
          {{ input.length }} angka — siap cari
        </template>
        <template v-else>
          Anda mengetik {{ input.length }} angka — minimal {{ MIN_LENGTH }}
        </template>
      </p>

      <!-- Chip hints -->
      <div class="flex flex-wrap items-center gap-[clamp(6px,1vw,10px)]
                  justify-center">
        <span
          v-for="hint in [
            '16 digit no. JKN',
            '16 digit NIK KTP',
            'No. rekam medis',
          ]"
          :key="hint"
          class="text-[clamp(10px,1.3vw,12px)] text-text-muted
                 bg-bg border border-border rounded-tag
                 px-[clamp(8px,1.2vw,11px)] py-[clamp(2px,0.4vw,4px)]"
        >
          {{ hint }}
        </span>
      </div>

      <!-- Frista bar -->
      <FristaBar :available="fristaAvailable" />

      <!-- NumPad -->
      <NumPad
        :can-submit="canSubmit"
        @digit="appendDigit"
        @delete="deleteDigit"
        @submit="submit"
      />
    </section>

    <IdleOverlay :seconds-left="secondsLeft" :visible="isCountingDown" />
  </main>
</template>
