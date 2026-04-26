<!--
  APM (T.A.R.A) - Root component.

  Layout per-screen di-handle masing-masing screen via RouterView.
  Global wire:
    - Frista card_read listener: auto-navigate ke /detect (kalau
      user di home/input; kalau di flow lain, biarkan).
    - System offline banner: subscribe event "system:offline" dari
      Go reconcile worker. Banner show/hide dengan transition.
-->
<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import OfflineBanner from './components/OfflineBanner.vue'
import { useWailsEvents } from './services/apm'
import { usePatientStore } from './stores/patient'

const router = useRouter()
const events = useWailsEvents()
const patient = usePatientStore()

const isOffline = ref(false)

let unsubCard = null
let unsubOffline = null

onMounted(() => {
  // Frista card_read - auto-navigate ke /detect bawa NIK/NoKartu
  unsubCard = events.onCardRead((data) => {
    const id = data.NoKartu || data.NIK
    if (!id) return
    patient.input = id
    const route = router.currentRoute.value.name
    if (route === 'home' || route === 'input') {
      router.push({ name: 'detect', query: { from: 'frista', id } })
    }
  })

  // Reconcile worker emit ketika koneksi Khanza pulih/putus.
  // Backend kirim true saat OFFLINE, false saat online kembali.
  unsubOffline = events.onSystemOffline((offline) => {
    isOffline.value = !!offline
  })
})

onUnmounted(() => {
  if (unsubCard) unsubCard()
  if (unsubOffline) unsubOffline()
})
</script>

<template>
  <OfflineBanner :visible="isOffline" />
  <RouterView />
</template>
