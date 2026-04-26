<!--
  AntrianScreen — pilih jenis antrian (5 sub-jenis dalam 2 section).

  Spec P-044 layout WAJIB:
    Section "Antrian loket (A)" uppercase 11px muted letter-spacing 0.5px
      Grid 2-col:
        Admisi appointment + counter
        Admisi walk-in + counter
      Card full-width:
        Rawat inap & IGD + counter

    Section "Antrian layanan umum (C)" uppercase 11px muted
      Grid 2-col:
        Farmasi / apotek + counter
        Customer service + counter

  On tap:
    1. Set card loading
    2. Call apmService.createAntrian(jenis, subJenis)
    3. Sukses → patient.setLastTicket + navigate /tiket
    4. Error → AlertModal "Gagal mengambil nomor antrian. Coba lagi."

  Counter:
    Fetch GetCounters() onMounted + setInterval 30s.
    Backend GetCounters return per-jenis (LOKET/POLI/UMUM), tidak
    per-subJenis. Tampilkan counter per-jenis di semua card jenis itu
    (best-effort sampai backend punya counter per-subJenis nanti).
-->
<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import AntrianCard from '../components/AntrianCard.vue'
import AlertModal from '../components/AlertModal.vue'
import IdleOverlay from '../components/IdleOverlay.vue'

import { I18N } from '../constants/i18n'
import { KIOSK } from '../constants/kiosk'
import { useIdleTimeout } from '../composables/useIdleTimeout'
import { apmService } from '../services/apm'
import { usePatientStore } from '../stores/patient'

const router = useRouter()
const patient = usePatientStore()

// Counter per jenis dari GetCounters
const counters = ref({ LOKET: 0, POLI: 0, UMUM: 0 })

// State loading per card (key: subJenis)
const loadingCard = ref('')

// Error modal
const errorVisible = ref(false)
const errorMsg = ref('')

// Format counter "A-035" dari (prefix, lastNoUrut)
function formatCounter(prefix, lastNoUrut) {
  if (!lastNoUrut || lastNoUrut <= 0) return ''
  return `${prefix}-${String(lastNoUrut).padStart(3, '0')}`
}
const loketCounter = computed(() => formatCounter('A', counters.value.LOKET))
const umumCounter = computed(() => formatCounter('C', counters.value.UMUM))

let pollingTimer = null

async function refreshCounters() {
  try {
    const c = await apmService.getCounters()
    counters.value = {
      LOKET: c.LOKET ?? 0,
      POLI: c.POLI ?? 0,
      UMUM: c.UMUM ?? 0,
    }
  } catch {
    // Backend offline — keep existing values
  }
}

async function pickAntrian(jenis, subJenis) {
  if (loadingCard.value) return
  loadingCard.value = subJenis
  try {
    const ticket = await apmService.createAntrian(jenis, subJenis)
    if (!ticket) throw new Error('Server tidak mengembalikan nomor antrian')
    patient.setLastTicket(ticket)
    router.push({ name: 'tiket', query: { from: 'antrian' } })
  } catch (e) {
    errorMsg.value = `Gagal mengambil nomor antrian. ${e?.message ?? ''} Silakan coba lagi.`.trim()
    errorVisible.value = true
  } finally {
    loadingCard.value = ''
  }
}

function back() {
  router.push({ name: 'home' })
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
  refreshCounters()
  pollingTimer = setInterval(refreshCounters, KIOSK.counterRefreshMs)
})
onUnmounted(() => {
  if (pollingTimer) clearInterval(pollingTimer)
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
        {{ I18N.home.antrian.title }}
      </h1>
    </header>

    <!-- Body -->
    <section
      class="flex-1 flex flex-col gap-[clamp(14px,2.5vw,20px)]
             p-[clamp(12px,2.5vw,20px)]
             max-w-[680px] mx-auto w-full"
    >
      <!-- Section: Antrian loket (A) -->
      <div class="flex flex-col gap-[clamp(8px,1.4vw,12px)]">
        <h2 class="text-[clamp(10px,1.3vw,11px)] uppercase tracking-[0.5px]
                   text-text-muted font-medium">
          Antrian loket (A)
        </h2>

        <div class="grid grid-cols-2 gap-[clamp(8px,1.5vw,12px)]">
          <!-- Admisi appointment -->
          <AntrianCard
            title="Admisi (appointment)"
            :counter="loketCounter"
            :loading="loadingCard === 'APPOINTMENT'"
            variant="blue"
            @click="pickAntrian('LOKET', 'APPOINTMENT')"
          >
            <template #icon>
              <svg
                xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
                stroke="currentColor" stroke-width="2" stroke-linecap="round"
                stroke-linejoin="round" class="w-[clamp(18px,2.6vw,22px)] h-[clamp(18px,2.6vw,22px)]"
              >
                <rect x="3" y="4" width="18" height="18" rx="2" />
                <line x1="16" y1="2" x2="16" y2="6" />
                <line x1="8" y1="2" x2="8" y2="6" />
                <line x1="3" y1="10" x2="21" y2="10" />
                <circle cx="12" cy="15" r="2" />
              </svg>
            </template>
          </AntrianCard>

          <!-- Admisi walk-in -->
          <AntrianCard
            title="Admisi (walk-in)"
            :counter="loketCounter"
            :loading="loadingCard === 'WALKIN'"
            variant="blue"
            @click="pickAntrian('LOKET', 'WALKIN')"
          >
            <template #icon>
              <svg
                xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
                stroke="currentColor" stroke-width="2" stroke-linecap="round"
                stroke-linejoin="round" class="w-[clamp(18px,2.6vw,22px)] h-[clamp(18px,2.6vw,22px)]"
              >
                <circle cx="12" cy="6" r="3" />
                <path d="m9 14 1.5 8M15 14l-1.5 8M9 14h6l-1-5h-4z" />
              </svg>
            </template>
          </AntrianCard>
        </div>

        <!-- Rawat inap & IGD full width -->
        <AntrianCard
          title="Rawat inap & IGD"
          :counter="loketCounter"
          :loading="loadingCard === 'RANAP_IGD'"
          variant="green"
          @click="pickAntrian('LOKET', 'RANAP_IGD')"
        >
          <template #icon>
            <svg
              xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
              stroke="currentColor" stroke-width="2" stroke-linecap="round"
              stroke-linejoin="round" class="w-[clamp(18px,2.6vw,22px)] h-[clamp(18px,2.6vw,22px)]"
            >
              <path d="M22 12h-4l-3 9L9 3l-3 9H2" />
            </svg>
          </template>
        </AntrianCard>
      </div>

      <!-- Section: Antrian layanan umum (C) -->
      <div class="flex flex-col gap-[clamp(8px,1.4vw,12px)]">
        <h2 class="text-[clamp(10px,1.3vw,11px)] uppercase tracking-[0.5px]
                   text-text-muted font-medium">
          Antrian layanan umum (C)
        </h2>

        <div class="grid grid-cols-2 gap-[clamp(8px,1.5vw,12px)]">
          <AntrianCard
            title="Farmasi / apotek"
            :counter="umumCounter"
            :loading="loadingCard === 'FARMASI'"
            variant="green"
            @click="pickAntrian('UMUM', 'FARMASI')"
          >
            <template #icon>
              <svg
                xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
                stroke="currentColor" stroke-width="2" stroke-linecap="round"
                stroke-linejoin="round" class="w-[clamp(18px,2.6vw,22px)] h-[clamp(18px,2.6vw,22px)]"
              >
                <path d="M10.5 5.5h3a4 4 0 0 1 4 4v9a4 4 0 0 1-4 4h-3a4 4 0 0 1-4-4v-9a4 4 0 0 1 4-4Z" />
                <path d="M2 12h20" />
              </svg>
            </template>
          </AntrianCard>

          <AntrianCard
            title="Customer service"
            :counter="umumCounter"
            :loading="loadingCard === 'CS'"
            variant="gray"
            @click="pickAntrian('UMUM', 'CS')"
          >
            <template #icon>
              <svg
                xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
                stroke="currentColor" stroke-width="2" stroke-linecap="round"
                stroke-linejoin="round" class="w-[clamp(18px,2.6vw,22px)] h-[clamp(18px,2.6vw,22px)]"
              >
                <path d="M3 18v-6a9 9 0 0 1 18 0v6" />
                <path d="M21 19a2 2 0 0 1-2 2h-1v-6h3zM3 19a2 2 0 0 0 2 2h1v-6H3z" />
              </svg>
            </template>
          </AntrianCard>
        </div>
      </div>
    </section>

    <IdleOverlay :seconds-left="secondsLeft" :visible="isCountingDown" />

    <AlertModal
      :visible="errorVisible"
      variant="error"
      title="Gagal mengambil nomor antrian"
      :message="errorMsg"
      primary-label="Coba lagi"
      close-label="Tutup"
      @primary="errorVisible = false"
      @close="errorVisible = false"
    />
  </main>
</template>
