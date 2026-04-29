<!--
  APM (T.A.R.A) - Root component.

  Layout per-screen di-handle masing-masing screen via RouterView.
  Global wire:
    - System offline banner: subscribe event "system:offline" dari
      Go reconcile worker. Banner show/hide dengan transition.

  Note: card reader auto-fill di-remove (RS Anggrek Mas tidak punya
  card reader; Frista = aplikasi sidik wajah BPJS, bukan reader). Pasien
  selalu input manual via NumPad. Verifikasi biometrik (Frista wajah /
  After.exe sidik jari) hanya dipanggil di SEP flow via BiometrikChoiceModal.
-->
<script setup>
import { onMounted, onUnmounted, ref } from 'vue'

import OfflineBanner from './components/OfflineBanner.vue'
import UpdateProgressModal from './components/UpdateProgressModal.vue'
import UpdateAutoApplyCountdown from './components/UpdateAutoApplyCountdown.vue'
import { apmService, useWailsEvents } from './services/apm'
import { useBrandingStore } from './stores/branding'

const events = useWailsEvents()
const branding = useBrandingStore()

const isOffline = ref(false)

// Update state — global supaya modal/countdown muncul kapan saja.
const updateInfo = ref(null) // { latest_version, current_version, ... }
const showCountdown = ref(false)
const countdownSeconds = ref(30)

const showProgressModal = ref(false)
const updatePhase = ref('download')
const downloaded = ref(0)
const total = ref(0)
const updateError = ref(null)
const appliedVersion = ref('')

const unsubs = []

onMounted(() => {
  branding.load()

  unsubs.push(events.onSystemOffline((offline) => { isOffline.value = !!offline }))

  // Auto-update events
  unsubs.push(events.onUpdateAvailable((info) => {
    updateInfo.value = info
    // Tidak auto-show progress; admin trigger manual via AdminScreen.
    // Banner subtle bisa di-add di HomeScreen header (TODO post-MVP).
  }))

  unsubs.push(events.onUpdateAutoApplyCountdown((sec) => {
    countdownSeconds.value = sec
    showCountdown.value = true
  }))

  unsubs.push(events.onUpdateAutoApplyCancelled(() => {
    showCountdown.value = false
  }))

  unsubs.push(events.onUpdateProgress((data) => {
    showProgressModal.value = true
    updatePhase.value = data.phase
    if (typeof data.downloaded === 'number') downloaded.value = data.downloaded
    if (typeof data.total === 'number') total.value = data.total
  }))

  unsubs.push(events.onUpdateApplied((data) => {
    appliedVersion.value = data.version || ''
    updatePhase.value = 'restart'
    // Modal tetap visible — backend akan restart dalam 500ms
  }))

  unsubs.push(events.onUpdateError((msg) => {
    updateError.value = String(msg)
  }))
})

onUnmounted(() => {
  for (const u of unsubs) {
    try { u() } catch {}
  }
})

async function onCountdownTimeout() {
  // Hidden by progress modal trigger from backend — biarkan backend yang
  // emit progress/applied event. Tidak perlu trigger ulang.
  showCountdown.value = false
}

async function onCountdownCancel() {
  showCountdown.value = false
  try {
    await apmService.cancelAutoApplyUpdate()
  } catch (e) {
    console.error('cancel auto-apply gagal', e)
  }
}
</script>

<template>
  <OfflineBanner :visible="isOffline" />
  <RouterView />

  <UpdateAutoApplyCountdown
    :visible="showCountdown"
    :seconds="countdownSeconds"
    :version="updateInfo?.latest_version ?? ''"
    @cancel="onCountdownCancel"
    @timeout="onCountdownTimeout"
  />

  <UpdateProgressModal
    :visible="showProgressModal"
    :phase="updatePhase"
    :downloaded="downloaded"
    :total="total"
    :error="updateError"
    :applied-version="appliedVersion"
  />
</template>
