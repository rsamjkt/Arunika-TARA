<!--
  SearchPasienScreen — pencarian pasien Umum di Khanza (direct-DB).

  Flow:
    1. onMounted: auto-search pakai patient.input dari InputScreen.
    2. Tampilkan 3 state:
       - loading (skeleton)
       - hasil ditemukan → PatientCard + CTA "Lanjut ke Pendaftaran"
       - tidak ketemu → message + opsi cari ulang / hubungi petugas
    3. Tombol "Cari ulang" untuk modify keyword (push back ke /input).
    4. Tombol "Lanjut": simpan ke store.pasienUmum + push ke /registrasi-umum.

  Berbeda dengan DetectScreen (BPJS) — di sini kita tidak panggil VClaim,
  cuma query master pasien di Khanza. Kalau pasien belum terdaftar, user
  diarahkan untuk hubungi petugas (registrasi pasien baru bukan flow MVP).
-->
<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'

import IdleOverlay from '../components/IdleOverlay.vue'
import AlertModal from '../components/AlertModal.vue'

import { I18N } from '../constants/i18n'
import { KIOSK } from '../constants/kiosk'
import { useIdleTimeout } from '../composables/useIdleTimeout'
import { apmService } from '../services/apm'
import { usePatientStore } from '../stores/patient'

const router = useRouter()
const patient = usePatientStore()

const loading = ref(true)
const result = ref(null)        // domain.Pasien | null
const errorMsg = ref('')
const errorVisible = ref(false)

const query = computed(() => patient.input || '')

if (!query.value) {
  // Tidak ada keyword — kembali ke input
  router.replace({ name: 'input', query: { mode: 'umum' } })
}

async function doSearch() {
  loading.value = true
  result.value = null
  try {
    const p = await apmService.cariPasien(query.value)
    result.value = p && p.NoRM ? p : null
  } catch (e) {
    errorMsg.value = (e && e.message) ? e.message : String(e)
    errorVisible.value = true
  } finally {
    loading.value = false
  }
}

function lanjutkan() {
  if (!result.value) return
  patient.setPasienUmum(result.value)
  router.push({ name: 'registrasi-umum' })
}

function cariUlang() {
  router.replace({ name: 'input', query: { mode: 'umum' } })
}

function pulang() {
  router.replace({ name: 'home' })
}

function formatTglIndo(d) {
  if (!d) return ''
  try {
    const dt = new Date(d)
    if (isNaN(dt.getTime())) return d
    const BULAN = ['Jan','Feb','Mar','Apr','Mei','Jun','Jul','Agu','Sep','Okt','Nov','Des']
    return `${dt.getDate()} ${BULAN[dt.getMonth()]} ${dt.getFullYear()}`
  } catch { return d }
}
function jkLabel(jk) {
  return jk === 'L' ? 'Laki-laki' : jk === 'P' ? 'Perempuan' : '-'
}

const { isCountingDown, secondsLeft } = useIdleTimeout({
  totalSeconds: KIOSK.idleTimeoutSec,
  countdownThreshold: KIOSK.idleCountdownSec,
  onTimeout: async () => {
    await patient.reset()
    router.push({ name: 'home' })
  },
})

onMounted(doSearch)
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
        @click="cariUlang"
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
        Pasien Umum — Pencarian
      </h1>
    </header>

    <!-- Body -->
    <section
      class="flex-1 flex flex-col gap-[clamp(10px,2vw,16px)]
             p-[clamp(12px,2.5vw,20px)]
             max-w-[640px] mx-auto w-full"
    >
      <!-- Query info -->
      <div class="bg-surface border border-border rounded-card
                  px-[clamp(12px,2vw,16px)] py-[clamp(10px,1.6vw,12px)]
                  text-[clamp(11px,1.5vw,13px)] text-text-secondary">
        Mencari berdasarkan
        <span class="font-medium text-text-primary">{{ query }}</span>
      </div>

      <!-- Loading skeleton -->
      <div v-if="loading"
           class="bg-surface border border-border rounded-card
                  p-[clamp(16px,2.5vw,24px)] flex items-center justify-center
                  text-[clamp(12px,1.6vw,14px)] text-text-muted">
        <svg class="animate-spin -ml-1 mr-3 h-5 w-5 text-blue" xmlns="http://www.w3.org/2000/svg"
             fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
          <path class="opacity-75" fill="currentColor"
                d="M4 12a8 8 0 0 1 8-8V0C5.4 0 0 5.4 0 12h4zm2 5.3A8 8 0 0 1 4 12H0c0 3 1.1 5.8 3 7.9l3-2.6z"/>
        </svg>
        Mencari data pasien…
      </div>

      <!-- Hasil ketemu -->
      <div v-else-if="result"
           class="bg-surface border border-border rounded-card
                  p-[clamp(16px,2.5vw,24px)] flex flex-col gap-[clamp(10px,1.5vw,14px)]">
        <div class="flex items-center gap-[clamp(8px,1.5vw,12px)]">
          <span class="bg-emerald-50 text-emerald-700 rounded-full
                       px-[clamp(8px,1.2vw,12px)] py-[clamp(2px,0.4vw,4px)]
                       text-[clamp(10px,1.3vw,12px)] font-medium">
            ✓ Pasien ditemukan
          </span>
        </div>
        <div class="text-[clamp(18px,2.8vw,22px)] font-medium text-text-primary leading-tight">
          {{ result.Nama }}
        </div>
        <dl class="grid grid-cols-2 gap-x-[clamp(12px,2vw,16px)] gap-y-[clamp(6px,1vw,8px)]
                   text-[clamp(11px,1.5vw,13px)]">
          <dt class="text-text-muted">No. RM</dt>
          <dd class="text-text-primary font-medium">{{ result.NoRM }}</dd>
          <dt class="text-text-muted">Tgl Lahir</dt>
          <dd class="text-text-primary">{{ formatTglIndo(result.TglLahir) }}</dd>
          <dt class="text-text-muted">Jenis Kelamin</dt>
          <dd class="text-text-primary">{{ jkLabel(result.JK) }}</dd>
          <dt v-if="result.NoTelp" class="text-text-muted">No. Telp</dt>
          <dd v-if="result.NoTelp" class="text-text-primary">{{ result.NoTelp }}</dd>
        </dl>
        <button
          type="button"
          class="mt-[clamp(6px,1vw,8px)] bg-blue text-white font-medium
                 rounded-btn py-[clamp(12px,1.8vw,14px)]
                 text-[clamp(13px,1.8vw,15px)]
                 hover:bg-blue-dark active:bg-blue-dark/90
                 min-h-[clamp(48px,6vw,56px)]"
          @click="lanjutkan"
        >
          Lanjut ke Pendaftaran →
        </button>
      </div>

      <!-- Tidak ketemu -->
      <div v-else
           class="bg-surface border border-border rounded-card
                  p-[clamp(16px,2.5vw,24px)] flex flex-col gap-[clamp(10px,1.5vw,14px)]">
        <div class="flex items-center gap-[clamp(8px,1.5vw,12px)]">
          <span class="bg-amber-50 text-amber-700 rounded-full
                       px-[clamp(8px,1.2vw,12px)] py-[clamp(2px,0.4vw,4px)]
                       text-[clamp(10px,1.3vw,12px)] font-medium">
            Tidak ditemukan
          </span>
        </div>
        <p class="text-[clamp(13px,1.8vw,15px)] text-text-primary">
          Tidak ada pasien yang cocok dengan
          <span class="font-medium">"{{ query }}"</span>.
        </p>
        <p class="text-[clamp(11px,1.5vw,13px)] text-text-muted">
          Pastikan Anda memasukkan No. Rekam Medis, NIK KTP,
          atau nama yang benar. Bila ini kunjungan pertama Anda,
          silakan hubungi petugas pendaftaran.
        </p>
        <div class="flex flex-col sm:flex-row gap-[clamp(8px,1.2vw,10px)]
                    mt-[clamp(6px,1vw,8px)]">
          <button
            type="button"
            class="flex-1 bg-blue text-white font-medium rounded-btn
                   py-[clamp(12px,1.8vw,14px)]
                   text-[clamp(13px,1.8vw,15px)]
                   hover:bg-blue-dark active:bg-blue-dark/90
                   min-h-[clamp(48px,6vw,56px)]"
            @click="cariUlang"
          >
            Cari ulang
          </button>
          <button
            type="button"
            class="flex-1 bg-surface border border-border text-text-primary font-medium
                   rounded-btn py-[clamp(12px,1.8vw,14px)]
                   text-[clamp(13px,1.8vw,15px)]
                   hover:border-border-strong
                   min-h-[clamp(48px,6vw,56px)]"
            @click="pulang"
          >
            Kembali ke Beranda
          </button>
        </div>
      </div>
    </section>

    <AlertModal
      :visible="errorVisible"
      :title="'Gagal mencari pasien'"
      :message="errorMsg"
      @close="errorVisible = false"
    />

    <IdleOverlay :seconds-left="secondsLeft" :visible="isCountingDown" />
  </main>
</template>
