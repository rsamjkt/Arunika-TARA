<!--
  SplashScreen — boot self-test sebelum kiosk siap dipakai pasien.

  Flow:
    1. Mount → panggil App.RunStartupChecks() (paralel di Go side)
    2. Animate appearance per check (stagger 80ms)
    3. Tampilkan ringkasan: berapa OK / WARN / FAIL
    4. Auto-navigate ke HomeScreen setelah:
       - semua check selesai DAN tidak ada critical fail → fade ke /
       - critical fail → tetap di splash, tombol "Buka Admin Panel" + "Coba Lagi"

  Petugas RS dapet snapshot kondisi kiosk sebelum operasional,
  pasien yang dateng pas startup tidak bingung kenapa kiosk gak respon.
-->
<script setup>
import { onMounted, ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import {
  PhCheckCircle,
  PhWarningCircle,
  PhXCircle,
  PhMinusCircle,
  PhCircleNotch,
} from '@phosphor-icons/vue'

import { apmService } from '../services/apm'
import { useBrandingStore } from '../stores/branding'

const router = useRouter()
const branding = useBrandingStore()

const checks = ref([])
const running = ref(true)
const error = ref('')

const summary = computed(() => {
  const tally = { ok: 0, warn: 0, fail: 0, skip: 0 }
  for (const c of checks.value) tally[c.status] = (tally[c.status] || 0) + 1
  return tally
})

const criticalFail = computed(() =>
  checks.value.some((c) => c.critical && c.status === 'fail'),
)

const ICONS = {
  ok: PhCheckCircle,
  warn: PhWarningCircle,
  fail: PhXCircle,
  skip: PhMinusCircle,
}

const COLORS = {
  ok: 'text-emerald-500',
  warn: 'text-amber-500',
  fail: 'text-red-500',
  skip: 'text-gray-400',
}

const LABELS = {
  ok: 'Siap',
  warn: 'Perhatian',
  fail: 'Gagal',
  skip: 'Lewati',
}

async function run() {
  running.value = true
  error.value = ''
  checks.value = []
  try {
    const res = await apmService.runStartupChecks()
    // Stagger reveal for visual effect (~80ms per item).
    for (let i = 0; i < res.length; i++) {
      checks.value.push(res[i])
      await new Promise((r) => setTimeout(r, 80))
    }
  } catch (e) {
    error.value = String(e?.message || e)
  } finally {
    running.value = false
  }

  // Auto-navigate kalau semua aman (atau cuma warn/skip).
  if (!criticalFail.value && !error.value) {
    setTimeout(() => router.replace({ name: 'home' }), 600)
  }
}

function retry() {
  run()
}

function gotoAdmin() {
  router.replace({ name: 'admin' })
}

onMounted(run)
</script>

<template>
  <div
    class="min-h-screen flex flex-col items-center justify-center
           bg-gradient-to-br from-bg via-surface to-bg p-8"
  >
    <!-- Branding header -->
    <div class="flex flex-col items-center mb-12">
      <div
        class="w-20 h-20 rounded-2xl flex items-center justify-center
               text-white font-black text-4xl shadow-lg mb-4"
        :style="{ background: 'var(--color-primary)' }"
      >
        T
      </div>
      <h1 class="text-text-primary font-bold text-[clamp(22px,2.6vw,30px)] mb-1">
        {{ branding.hospitalName || 'Anjungan Pasien Mandiri' }}
      </h1>
      <p class="text-text-secondary text-[clamp(13px,1.4vw,16px)]">
        Memeriksa kesiapan kiosk…
      </p>
    </div>

    <!-- Check list card -->
    <div
      class="bg-surface border border-border rounded-2xl shadow-md
             w-full max-w-xl px-6 py-5"
    >
      <div v-if="checks.length === 0 && running" class="flex items-center gap-3 py-4">
        <PhCircleNotch :size="22" class="animate-spin text-primary" />
        <span class="text-text-secondary text-[clamp(14px,1.5vw,16px)]">
          Menjalankan probe…
        </span>
      </div>

      <ul class="divide-y divide-border">
        <li
          v-for="c in checks"
          :key="c.component"
          class="flex items-center justify-between gap-4 py-3 animate-fade-in"
        >
          <div class="flex items-center gap-3 flex-1 min-w-0">
            <component
              :is="ICONS[c.status] || PhCircleNotch"
              :size="26"
              :class="COLORS[c.status] || 'text-gray-400'"
              weight="fill"
            />
            <div class="min-w-0">
              <div class="text-text-primary font-semibold text-[clamp(14px,1.6vw,17px)]">
                {{ c.component }}
              </div>
              <div
                class="text-text-secondary text-[clamp(12px,1.3vw,14px)] truncate"
                :title="c.message"
              >
                {{ c.message }}
              </div>
            </div>
          </div>
          <div class="flex flex-col items-end shrink-0">
            <span
              class="text-[clamp(11px,1.2vw,13px)] font-bold uppercase tracking-wide"
              :class="COLORS[c.status] || 'text-gray-400'"
            >
              {{ LABELS[c.status] || c.status }}
            </span>
            <span class="text-text-secondary text-[10px] tabular-nums">
              {{ c.duration_ms }}ms
            </span>
          </div>
        </li>
      </ul>

      <!-- Summary footer -->
      <div
        v-if="!running"
        class="mt-4 pt-4 border-t border-border
               flex items-center justify-between
               text-[clamp(13px,1.4vw,15px)]"
      >
        <div class="flex items-center gap-4">
          <span class="text-emerald-600 font-semibold">{{ summary.ok }} OK</span>
          <span v-if="summary.warn > 0" class="text-amber-600 font-semibold">
            {{ summary.warn }} Perhatian
          </span>
          <span v-if="summary.fail > 0" class="text-red-600 font-semibold">
            {{ summary.fail }} Gagal
          </span>
          <span v-if="summary.skip > 0" class="text-gray-400">
            {{ summary.skip }} Skip
          </span>
        </div>
        <span v-if="!criticalFail && !error" class="text-text-secondary">
          Mengarahkan ke layar utama…
        </span>
      </div>
    </div>

    <!-- Error / critical-fail recovery -->
    <div v-if="error" class="mt-6 max-w-xl text-center">
      <p class="text-red-600 font-semibold mb-3">{{ error }}</p>
      <button
        type="button"
        class="px-6 py-3 rounded-btn bg-primary text-white font-semibold
               min-h-[clamp(44px,5vw,52px)] shadow-sm hover:opacity-90"
        @click="retry"
      >
        Coba Lagi
      </button>
    </div>

    <div
      v-if="criticalFail && !error"
      class="mt-6 max-w-xl flex flex-col items-center gap-3"
    >
      <p class="text-red-600 font-semibold text-center">
        Sistem belum siap melayani pasien. Hubungi petugas IT.
      </p>
      <div class="flex gap-3">
        <button
          type="button"
          class="px-6 py-3 rounded-btn bg-surface border border-border
                 text-text-primary font-semibold
                 min-h-[clamp(44px,5vw,52px)] hover:border-border-strong"
          @click="retry"
        >
          Coba Lagi
        </button>
        <button
          type="button"
          class="px-6 py-3 rounded-btn bg-primary text-white font-semibold
                 min-h-[clamp(44px,5vw,52px)] shadow-sm hover:opacity-90"
          @click="gotoAdmin"
        >
          Buka Admin Panel
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.animate-fade-in {
  animation: fadeIn 220ms ease-out;
}
@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
