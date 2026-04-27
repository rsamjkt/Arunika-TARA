<!--
  BantuSayaScreen — guided wizard untuk first-time elderly users (BB1).

  4 step:
    1. Welcome + reassurance
    2. Q1: "Apakah Anda punya kartu BPJS?" Y/N
    3. Q2: "Apakah ini kunjungan pertama Anda di RS ini?" Y/N
    4. Hasil rekomendasi + tombol "Ke layanan yang sesuai" / "Hubungi petugas"

  Design choices untuk lansia:
    - Font besar (≥18px body, 24px+ heading)
    - Tombol setinggi 72px+
    - Kontras tinggi
    - 1 instruksi per screen
    - Tombol "Kembali" + "Hubungi petugas" selalu ada
    - Audio cue tap setiap pilih
-->
<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'

import {
  PhHandshake,
  PhCheckCircle,
  PhXCircle,
  PhPhone,
  PhCaretRight,
  PhArrowsClockwise,
} from '@phosphor-icons/vue'

import IdleOverlay from '../components/IdleOverlay.vue'
import BackButton from '../components/BackButton.vue'

import { KIOSK } from '../constants/kiosk'
import { useIdleTimeout } from '../composables/useIdleTimeout'
import { useAudioCue } from '../composables/useAudioCue'
import { usePatientStore } from '../stores/patient'

const router = useRouter()
const patient = usePatientStore()
const audio = useAudioCue()

// step: 0 welcome, 1 punya BPJS, 2 first-time, 3 hasil
const step = ref(0)
const punyaBPJS = ref(null)       // true/false/null
const firstTime = ref(null)       // true/false/null

const totalSteps = 4

function next() {
  audio.tap()
  if (step.value < totalSteps - 1) step.value++
}
function prev() {
  audio.tap()
  if (step.value > 0) step.value--
}
function pickBPJS(val) {
  audio.tap()
  punyaBPJS.value = val
  setTimeout(next, 300) // smooth feedback delay
}
function pickFirstTime(val) {
  audio.tap()
  firstTime.value = val
  setTimeout(next, 300)
}

// Recommendation logic
const recommendation = computed(() => {
  if (punyaBPJS.value === true) {
    return {
      title: 'Daftar sebagai Pasien BPJS',
      desc: 'Sistem akan otomatis mendeteksi rujukan, kontrol, atau booking Mobile JKN Anda. Cukup tap kartu atau ketik nomor.',
      icon: 'bpjs',
      action: () => {
        patient.reset()
        router.push({ name: 'input', query: { mode: 'bpjs' } })
      },
      actionLabel: 'Lanjut ke Pendaftaran BPJS',
    }
  }
  if (punyaBPJS.value === false && firstTime.value === false) {
    return {
      title: 'Daftar sebagai Pasien Umum',
      desc: 'Cari data Anda dengan No. Rekam Medis, NIK KTP, atau nama. Pembayaran dilakukan di kasir setelah pemeriksaan.',
      icon: 'user',
      action: () => {
        patient.reset()
        router.push({ name: 'input', query: { mode: 'umum' } })
      },
      actionLabel: 'Lanjut ke Pendaftaran Umum',
    }
  }
  // First-time + tidak punya BPJS = harus ke loket pendaftaran manual
  return {
    title: 'Silakan ke Loket Pendaftaran',
    desc: 'Untuk kunjungan pertama tanpa kartu BPJS, petugas akan membantu Anda registrasi sebagai pasien baru. Ambil nomor antrian loket dulu.',
    icon: 'staff',
    action: () => {
      audio.notify()
      router.push({ name: 'home' })
      // TODO emit staff:call event
    },
    actionLabel: 'Ke Beranda — Ambil Antrian Loket',
  }
})

function backToHome() {
  audio.tap()
  router.push({ name: 'home' })
}
function callStaff() {
  audio.notify()
}
function restart() {
  audio.tap()
  step.value = 0
  punyaBPJS.value = null
  firstTime.value = null
}

const { isCountingDown, secondsLeft } = useIdleTimeout({
  totalSeconds: KIOSK.idleTimeoutSec,
  countdownThreshold: KIOSK.idleCountdownSec,
  onTimeout: () => router.push({ name: 'home' }),
})
</script>

<template>
  <main class="min-h-screen bg-bg flex flex-col">
    <!-- Header sederhana -->
    <header
      class="bg-surface border-b border-border flex items-center justify-between
             px-[clamp(16px,3vw,28px)] py-[clamp(12px,1.8vw,16px)]"
    >
      <div class="flex items-center gap-[clamp(10px,1.5vw,14px)]">
        <PhHandshake :size="32" weight="duotone" :style="{ color: 'var(--color-primary, #1B4FD8)' }" />
        <h1 class="text-[clamp(18px,2.4vw,22px)] font-semibold text-text-primary">
          Bantuan Pendaftaran
        </h1>
      </div>
      <div class="text-[clamp(13px,1.7vw,15px)] text-text-muted">
        Langkah {{ step + 1 }} / {{ totalSteps }}
      </div>
    </header>

    <!-- Body -->
    <section
      class="flex-1 flex flex-col items-center justify-center
             p-[clamp(20px,4vw,40px)] gap-[clamp(20px,3vw,32px)]
             max-w-[720px] mx-auto w-full"
    >
      <!-- Step 0: Welcome -->
      <div v-if="step === 0" class="text-center space-y-[clamp(16px,2.5vw,24px)]">
        <div
          class="mx-auto rounded-full flex items-center justify-center
                 w-[clamp(120px,16vw,160px)] h-[clamp(120px,16vw,160px)]"
          :style="{ backgroundColor: 'var(--color-primary-light, #E8F0FE)', color: 'var(--color-primary, #1B4FD8)' }"
        >
          <PhHandshake :size="80" weight="duotone" />
        </div>

        <h2 class="text-[clamp(26px,4vw,38px)] font-bold text-text-primary leading-tight">
          Selamat datang!
        </h2>
        <p class="text-[clamp(16px,2.2vw,20px)] text-text-secondary leading-relaxed">
          Tidak masalah kalau ini pertama kali Anda di kiosk ini.<br/>
          Saya akan tanya 2 hal saja, lalu arahkan Anda ke layanan yang tepat.
        </p>

        <button
          type="button"
          class="text-white font-semibold rounded-btn
                 px-[clamp(28px,4vw,40px)] py-[clamp(16px,2.2vw,20px)]
                 text-[clamp(17px,2.3vw,20px)]
                 min-h-[clamp(64px,8vw,80px)]
                 hover:opacity-90 active:opacity-80
                 inline-flex items-center gap-3 shadow-md"
          :style="{ backgroundColor: 'var(--color-primary, #1B4FD8)' }"
          @click="next"
        >
          Mulai Bantuan
          <PhCaretRight :size="24" weight="bold" />
        </button>
      </div>

      <!-- Step 1: Punya BPJS? -->
      <div v-else-if="step === 1" class="text-center space-y-[clamp(20px,3vw,32px)] w-full">
        <h2 class="text-[clamp(24px,3.6vw,32px)] font-bold text-text-primary leading-tight">
          Apakah Anda punya kartu BPJS / JKN?
        </h2>
        <p class="text-[clamp(14px,2vw,17px)] text-text-secondary">
          Kartu BPJS biasanya berwarna hijau atau biru, ada logo JKN-KIS di depan.
        </p>

        <div class="grid grid-cols-2 gap-[clamp(12px,2vw,16px)] mt-[clamp(20px,3vw,32px)]">
          <button
            type="button"
            class="bg-surface border-2 rounded-card
                   p-[clamp(20px,3vw,28px)] text-left
                   min-h-[clamp(140px,18vw,180px)]
                   flex flex-col items-center justify-center gap-3
                   hover:opacity-95 active:opacity-90 transition-all"
            :class="punyaBPJS === true ? 'border-emerald-500 bg-emerald-50' : 'border-border'"
            @click="pickBPJS(true)"
          >
            <PhCheckCircle :size="56" weight="duotone" class="text-emerald-600" />
            <div class="text-[clamp(20px,2.8vw,26px)] font-bold text-text-primary">
              Ya, punya
            </div>
          </button>
          <button
            type="button"
            class="bg-surface border-2 rounded-card
                   p-[clamp(20px,3vw,28px)] text-left
                   min-h-[clamp(140px,18vw,180px)]
                   flex flex-col items-center justify-center gap-3
                   hover:opacity-95 active:opacity-90 transition-all"
            :class="punyaBPJS === false ? 'border-amber-500 bg-amber-50' : 'border-border'"
            @click="pickBPJS(false)"
          >
            <PhXCircle :size="56" weight="duotone" class="text-amber-600" />
            <div class="text-[clamp(20px,2.8vw,26px)] font-bold text-text-primary">
              Tidak punya
            </div>
          </button>
        </div>
      </div>

      <!-- Step 2: First time? (skip kalau punya BPJS, langsung ke step 3) -->
      <div v-else-if="step === 2" class="text-center space-y-[clamp(20px,3vw,32px)] w-full">
        <h2 class="text-[clamp(24px,3.6vw,32px)] font-bold text-text-primary leading-tight">
          Apakah ini kunjungan pertama Anda di RS ini?
        </h2>
        <p class="text-[clamp(14px,2vw,17px)] text-text-secondary">
          Kalau Anda pernah berobat di sini sebelumnya, datanya sudah tersimpan.
        </p>

        <div class="grid grid-cols-2 gap-[clamp(12px,2vw,16px)] mt-[clamp(20px,3vw,32px)]">
          <button
            type="button"
            class="bg-surface border-2 rounded-card
                   p-[clamp(20px,3vw,28px)] text-left
                   min-h-[clamp(140px,18vw,180px)]
                   flex flex-col items-center justify-center gap-3
                   hover:opacity-95 active:opacity-90 transition-all"
            :class="firstTime === true ? 'border-blue-500 bg-blue-50' : 'border-border'"
            @click="pickFirstTime(true)"
          >
            <div class="text-[clamp(40px,5vw,56px)]">🆕</div>
            <div class="text-[clamp(20px,2.8vw,26px)] font-bold text-text-primary">
              Ya, pertama kali
            </div>
          </button>
          <button
            type="button"
            class="bg-surface border-2 rounded-card
                   p-[clamp(20px,3vw,28px)] text-left
                   min-h-[clamp(140px,18vw,180px)]
                   flex flex-col items-center justify-center gap-3
                   hover:opacity-95 active:opacity-90 transition-all"
            :class="firstTime === false ? 'border-blue-500 bg-blue-50' : 'border-border'"
            @click="pickFirstTime(false)"
          >
            <div class="text-[clamp(40px,5vw,56px)]">👤</div>
            <div class="text-[clamp(20px,2.8vw,26px)] font-bold text-text-primary">
              Sudah pernah
            </div>
          </button>
        </div>
      </div>

      <!-- Step 3: Hasil rekomendasi -->
      <div v-else class="text-center space-y-[clamp(20px,3vw,28px)] w-full">
        <div
          class="mx-auto rounded-full flex items-center justify-center
                 w-[clamp(100px,14vw,140px)] h-[clamp(100px,14vw,140px)]"
          :style="{ backgroundColor: 'var(--color-primary-light, #E8F0FE)', color: 'var(--color-primary, #1B4FD8)' }"
        >
          <PhCheckCircle :size="64" weight="duotone" />
        </div>

        <h2 class="text-[clamp(24px,3.6vw,32px)] font-bold text-text-primary leading-tight">
          {{ recommendation.title }}
        </h2>
        <p class="text-[clamp(15px,2.1vw,18px)] text-text-secondary leading-relaxed max-w-[560px] mx-auto">
          {{ recommendation.desc }}
        </p>

        <div class="flex flex-col gap-3 mt-[clamp(16px,2.5vw,24px)]">
          <button
            type="button"
            class="text-white font-semibold rounded-btn
                   px-[clamp(28px,4vw,40px)] py-[clamp(16px,2.2vw,20px)]
                   text-[clamp(17px,2.3vw,20px)]
                   min-h-[clamp(64px,8vw,80px)]
                   hover:opacity-90 active:opacity-80
                   inline-flex items-center justify-center gap-3 shadow-md"
            :style="{ backgroundColor: 'var(--color-primary, #1B4FD8)' }"
            @click="recommendation.action"
          >
            {{ recommendation.actionLabel }}
            <PhCaretRight :size="24" weight="bold" />
          </button>

          <button
            type="button"
            class="text-text-secondary font-medium
                   px-[clamp(20px,3vw,28px)] py-[clamp(12px,1.8vw,16px)]
                   text-[clamp(14px,1.8vw,16px)]
                   min-h-[clamp(56px,7vw,64px)]
                   rounded-btn hover:bg-bg
                   inline-flex items-center justify-center gap-2"
            @click="restart"
          >
            <PhArrowsClockwise :size="20" weight="bold" />
            Ulangi pertanyaan
          </button>
        </div>
      </div>
    </section>

    <!-- Footer: Back + Panggil Petugas selalu ada -->
    <footer
      class="bg-surface border-t border-border
             flex items-center justify-between
             px-[clamp(16px,3vw,28px)] py-[clamp(10px,1.5vw,14px)]"
    >
      <BackButton @click="step > 0 ? prev() : backToHome()" :label="step > 0 ? 'Kembali' : 'Beranda'" />
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
