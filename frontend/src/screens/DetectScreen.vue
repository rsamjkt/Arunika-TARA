<!--
  DetectScreen — animasi Smart BPJS Detector di backend.

  onMounted:
    1. Subscribe ke event "detect:step_update" (kalau Go emit; saat
       ini detector belum emit per-step, tapi handler tetap aktif
       untuk future).
    2. Trigger patient.detect(input) async.
    3. Sambil tunggu, simulasi step transitions berdasarkan elapsed
       time (~700ms per step) supaya UI bergerak tanpa perlu Go
       emit per-step.
    4. Saat detection selesai (Promise resolved), snap semua step
       ke 'done' lalu navigate ke /result.
    5. Timeout guard: kalau >7 detik belum selesai, tampilkan
       message "Sedang memproses..." (tidak abort).

  Catatan: simulasi waktu cocok untuk UX karena Smart Detector di
  backend memang ~1-5 detik (paralel timeout). Kalau backend nanti
  emit event per-step, handler akan override simulasi.
-->
<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import ProgressRing from '../components/ProgressRing.vue'
import DetectionStep from '../components/DetectionStep.vue'

import { useWailsEvents } from '../services/apm'
import { usePatientStore } from '../stores/patient'

const router = useRouter()
const patient = usePatientStore()
const events = useWailsEvents()

// 5 step labels sesuai spec
const stepDefs = [
  { id: 'verify-bpjs',  label: 'Verifikasi status BPJS' },
  { id: 'check-mjkn',   label: 'Cek booking Mobile JKN' },
  { id: 'check-kontrol', label: 'Cek jadwal kontrol' },
  { id: 'check-ranap',  label: 'Cek riwayat rawat inap' },
  { id: 'finalize',     label: 'Menentukan jenis kunjungan' },
]

const steps = ref(stepDefs.map((s) => ({ ...s, state: 'wait' })))
const isLongRunning = ref(false)
const errorMsg = ref('')

let unsubStepEvent = null
let simulationTimer = null
let longRunningTimer = null

function setStep(id, state) {
  const idx = steps.value.findIndex((s) => s.id === id)
  if (idx >= 0) steps.value[idx].state = state
}
function setAllDone() {
  steps.value.forEach((s) => (s.state = 'done'))
}
function setActiveAt(idx) {
  // Mark previous done, current active, sisanya wait
  steps.value.forEach((s, i) => {
    if (i < idx) s.state = 'done'
    else if (i === idx) s.state = 'active'
    else s.state = 'wait'
  })
}

// Time-based simulation supaya UI bergerak walau Go belum emit event.
// Step transitions: 0ms, 700ms, 1400ms, 2100ms, 2800ms (5 steps).
function startSimulation() {
  setActiveAt(0)
  let idx = 0
  simulationTimer = setInterval(() => {
    idx++
    if (idx >= stepDefs.length) {
      // Sudah sampai step terakhir, biarkan active sampai detection real selesai
      clearInterval(simulationTimer)
      simulationTimer = null
      return
    }
    setActiveAt(idx)
  }, 700)
}

onMounted(async () => {
  if (!patient.input) {
    // User langsung buka /detect tanpa input — balik ke home
    router.replace({ name: 'home' })
    return
  }

  // Subscribe to backend event (kalau Go emit per-step di future)
  unsubStepEvent = events.onDetectStep((data) => {
    if (data && data.step) {
      setStep(data.step, data.state)
    }
  })

  // Long-running guard: 7 detik
  longRunningTimer = setTimeout(() => {
    isLongRunning.value = true
  }, 7000)

  // Start UI simulation
  startSimulation()

  // Trigger real detection
  try {
    const result = await patient.detect(patient.input)
    setAllDone()
    if (result) {
      // Beri jeda 200ms supaya animasi 'done' terlihat sebentar
      setTimeout(() => router.push({ name: 'result' }), 200)
    } else {
      errorMsg.value = patient.error ?? 'Tidak bisa menghubungi server. Silakan coba lagi.'
      // Tetap navigate ke result — ResultScreen handle error display
      setTimeout(() => router.push({ name: 'result' }), 800)
    }
  } catch (e) {
    errorMsg.value = e?.message ?? String(e)
    setTimeout(() => router.push({ name: 'result' }), 800)
  } finally {
    if (longRunningTimer) clearTimeout(longRunningTimer)
    if (simulationTimer) clearInterval(simulationTimer)
  }
})

onUnmounted(() => {
  if (unsubStepEvent) unsubStepEvent()
  if (simulationTimer) clearInterval(simulationTimer)
  if (longRunningTimer) clearTimeout(longRunningTimer)
})
</script>

<template>
  <main class="min-h-screen bg-bg flex flex-col">
    <!-- Header (no back — detection sedang berjalan, tidak boleh dibatalkan
         tengah jalan supaya state backend tidak corrupt) -->
    <header
      class="bg-surface border-b border-border flex items-center
             px-[clamp(16px,3vw,28px)] py-[clamp(10px,1.8vw,16px)]"
    >
      <h1 class="text-[clamp(15px,2.2vw,18px)] font-medium text-text-primary">
        Memproses identitas Anda
      </h1>
    </header>

    <!-- Body -->
    <section
      class="flex-1 flex flex-col gap-[clamp(14px,2.5vw,20px)]
             p-[clamp(14px,3vw,24px)]
             max-w-[480px] mx-auto w-full justify-center"
    >
      <!-- Spinner ring -->
      <div class="flex flex-col items-center gap-[clamp(8px,1.5vw,12px)] py-[clamp(8px,2vw,16px)]">
        <ProgressRing :size="80" :stroke-width="6" />

        <p class="text-[clamp(14px,2.2vw,18px)] font-medium text-text-primary mt-2">
          Mohon tunggu sebentar...
        </p>
        <p class="text-[clamp(11px,1.5vw,13px)] text-text-muted text-center max-w-[320px]">
          Sistem sedang memeriksa data Anda di BPJS dan rumah sakit.
        </p>

        <!-- Long-running guard: muncul kalau >7s -->
        <p
          v-if="isLongRunning"
          class="text-[clamp(11px,1.5vw,13px)] text-amber-700 mt-1
                 bg-amber-50 border border-amber-200 rounded-tag
                 px-[clamp(10px,1.5vw,14px)] py-[clamp(4px,0.8vw,6px)]"
        >
          Sedang memproses, mohon tunggu...
        </p>

        <!-- Error display kalau ada -->
        <p
          v-if="errorMsg"
          class="text-[clamp(11px,1.5vw,13px)] text-rose-700 mt-1 text-center"
        >
          {{ errorMsg }}
        </p>
      </div>

      <!-- Step list -->
      <div class="flex flex-col gap-[clamp(6px,1vw,8px)]">
        <DetectionStep
          v-for="step in steps"
          :key="step.id"
          :label="step.label"
          :state="step.state"
        />
      </div>
    </section>
  </main>
</template>
