<!--
  APM (T.A.R.A) - Root component.

  Hanya RouterView - layout per-screen di-handle masing-masing screen.
  Global wire: Frista card_read listener auto-navigate ke /detect (kalau
  user di home; kalau di tengah flow lain, biarkan).
-->
<script setup>
import { onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'

import { useWailsEvents } from './services/apm'
import { usePatientStore } from './stores/patient'

const router = useRouter()
const events = useWailsEvents()
const patient = usePatientStore()

let unsubCard = null

onMounted(() => {
  // Frista card_read - auto-navigate ke /detect bawa NIK/NoKartu
  unsubCard = events.onCardRead((data) => {
    const id = data.NoKartu || data.NIK
    if (!id) return
    patient.input = id
    // Hanya redirect kalau user belum di tengah flow detect/result
    const route = router.currentRoute.value.name
    if (route === 'home' || route === 'input') {
      router.push({ name: 'detect', query: { from: 'frista', id } })
    }
  })
})

onUnmounted(() => {
  if (unsubCard) unsubCard()
})
</script>

<template>
  <RouterView />
</template>
