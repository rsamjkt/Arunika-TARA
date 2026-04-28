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
import { useWailsEvents } from './services/apm'
import { useBrandingStore } from './stores/branding'

const events = useWailsEvents()
const branding = useBrandingStore()

const isOffline = ref(false)

let unsubOffline = null

onMounted(() => {
  // Load branding (theme color + logo + audio) — apply CSS variables
  branding.load()

  // Reconcile worker emit ketika koneksi Khanza pulih/putus.
  // Backend kirim true saat OFFLINE, false saat online kembali.
  unsubOffline = events.onSystemOffline((offline) => {
    isOffline.value = !!offline
  })
})

onUnmounted(() => {
  if (unsubOffline) unsubOffline()
})
</script>

<template>
  <OfflineBanner :visible="isOffline" />
  <RouterView />
</template>
