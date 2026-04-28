<!--
  HomeScreen — landing kiosk APM (T.A.R.A) v1.1 "Mahatma".

  Layout baru (post BB3 + C16):
    - Header: logo (dari config.toml) + nama RS + status dots + jam digital
    - Welcome banner BESAR: greeting time-aware + 1 ilustrasi medis SVG
    - Hero "Pasien BPJS" (60% visual weight — mayoritas user RS pemerintah)
    - 2 Secondary cards setara: "Pasien Umum" + "Antrian Loket"
    - Aktivasi Satu Sehat → footer link kecil (niche action)
    - Footer: "Pertama kali? Bantu Saya" + "Panggil Petugas"
-->
<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import {
  PhUser,
  PhTicket,
  PhFingerprint,
  PhLifebuoy,
  PhPhone,
  PhCircleNotch,
  PhCaretRight,
} from '@phosphor-icons/vue'

import StatusDot from '../components/StatusDot.vue'
import BpjsLogo from '../components/BpjsLogo.vue'
import WelcomeIllustration from '../components/WelcomeIllustration.vue'

import { I18N } from '../constants/i18n'
import { useClock } from '../composables/useClock'
import { useAudioCue } from '../composables/useAudioCue'
import { apmService } from '../services/apm'
import { usePatientStore } from '../stores/patient'
import { useBrandingStore } from '../stores/branding'

const router = useRouter()
const patient = usePatientStore()
const branding = useBrandingStore()
const audio = useAudioCue()
const { time, date } = useClock()

// Status dots
const bpjsStatus = ref('online')
const sistemStatus = ref('online')
const antrianLoading = ref(false)

async function refreshStatus() {
  try {
    const sys = await apmService.getSystemStatus()
    bpjsStatus.value = sys.Online ? 'online' : 'offline'
    // Sistem dot reflect kondisi hardware penting: printer + biometrik (Frista wajah / After.exe).
    // RS Anggrek Mas tidak punya card reader; Frista di sini = aplikasi sidik wajah BPJS.
    sistemStatus.value = sys.Hardware?.Printer || sys.Hardware?.Frista || sys.Hardware?.Fingerprint ? 'online' : 'warning'
  } catch {
    // backend belum siap — biarkan default
  }
}

// Greeting time-aware
const greeting = computed(() => {
  const h = new Date().getHours()
  if (h < 11) return 'Selamat pagi!'
  if (h < 15) return 'Selamat siang!'
  if (h < 18) return 'Selamat sore!'
  return 'Selamat malam!'
})

// Idle timeout di-skip di HomeScreen — pasien belum ada flow
// in-progress, jadi countdown overlay tidak relevan dan bingungkan
// lansia. Patient store akan di-reset saat user tap salah satu CTA.

// Click handlers — semua dengan audio cue + reset session
async function startBPJS() {
  audio.tap()
  await patient.reset()
  router.push({ name: 'input', query: { mode: 'bpjs' } })
}
async function startUmum() {
  audio.tap()
  await patient.reset()
  router.push({ name: 'input', query: { mode: 'umum' } })
}
async function startAntrian() {
  if (antrianLoading.value) return
  audio.tap()
  antrianLoading.value = true
  try {
    await patient.reset()
    const ticket = await apmService.createAntrian('LOKET', 'WALKIN')
    patient.setLastTicket(ticket)
    audio.success()
    router.push({ name: 'tiket' })
  } catch (e) {
    audio.error()
    alert('Gagal mengambil nomor antrian. Silakan hubungi petugas.')
  } finally {
    antrianLoading.value = false
  }
}
async function startSatuSehat() {
  audio.tap()
  await patient.reset()
  router.push({ name: 'input', query: { mode: 'satusehat' } })
}
function startBantuSaya() {
  audio.tap()
  router.push({ name: 'bantu-saya' })
}
function callStaff() {
  audio.notify()
  // TODO: emit Wails event 'staff:call' untuk admin panel notification
}

// Hidden admin trigger — tap 5x cepat (≤2s) di logo header RS = buka /admin.
// Pasien biasa tidak akan accidentally trigger karena pattern ini deliberate.
// Pin gate (`config.toml [admin] pin`) tetap aktif setelah masuk.
const adminTapCount = ref(0)
let adminTapTimer = null
function onAdminTap() {
  adminTapCount.value++
  if (adminTapTimer) clearTimeout(adminTapTimer)
  adminTapTimer = setTimeout(() => { adminTapCount.value = 0 }, 2000)
  if (adminTapCount.value >= 5) {
    adminTapCount.value = 0
    if (adminTapTimer) clearTimeout(adminTapTimer)
    audio.tap()
    router.push({ name: 'admin' })
  }
}

// Keyboard shortcut Ctrl+Alt+A (Cmd+Alt+A di Mac) — staff dengan keyboard.
function onKeydown(e) {
  if ((e.ctrlKey || e.metaKey) && e.altKey && e.key.toLowerCase() === 'a') {
    e.preventDefault()
    audio.tap()
    router.push({ name: 'admin' })
  }
}

onMounted(() => {
  refreshStatus()
  window.addEventListener('keydown', onKeydown)
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
  if (adminTapTimer) clearTimeout(adminTapTimer)
})
</script>

<template>
  <main class="min-h-screen bg-bg flex flex-col">
    <!-- ============ Header ============ -->
    <header
      class="bg-surface border-b border-border
             flex items-center justify-between
             px-[clamp(16px,3vw,28px)] py-[clamp(10px,1.8vw,16px)]"
    >
      <!-- Kiri: logo (dari config) atau fallback "T" mark + nama RS + tagline.
           Tap logo 5x cepat = hidden admin trigger. -->
      <div
        class="flex items-center gap-[clamp(10px,1.5vw,14px)] min-w-0 cursor-pointer select-none"
        @click="onAdminTap"
      >
        <img
          v-if="branding.logoDataURL"
          :src="branding.logoDataURL"
          :alt="branding.hospitalName"
          class="object-contain flex-shrink-0
                 w-[clamp(36px,5vw,44px)] h-[clamp(36px,5vw,44px)]"
        />
        <div
          v-else
          class="rounded-[8px] flex items-center justify-center flex-shrink-0
                 w-[clamp(36px,5vw,44px)] h-[clamp(36px,5vw,44px)]"
          style="background-color: var(--color-primary, #1B4FD8)"
        >
          <span class="text-white font-bold text-[clamp(18px,2.5vw,22px)] leading-none">T</span>
        </div>
        <div class="min-w-0">
          <div class="text-[clamp(14px,1.9vw,17px)] font-semibold text-text-primary leading-tight truncate">
            {{ branding.hospitalName }}
          </div>
          <div class="text-[clamp(11px,1.3vw,13px)] text-text-muted leading-tight">
            {{ branding.hospitalTagline }}
          </div>
        </div>
      </div>

      <!-- Kanan: status + jam digital -->
      <div class="flex items-center gap-[clamp(8px,1.5vw,14px)] flex-shrink-0">
        <StatusDot
          :label="`${I18N.status.bpjs} ${bpjsStatus === 'online' ? I18N.status.online : I18N.status.offline}`"
          :variant="bpjsStatus"
        />
        <StatusDot
          :label="`${I18N.status.sistem} ${sistemStatus === 'online' ? I18N.status.online : I18N.status.warning}`"
          :variant="sistemStatus"
        />
        <div class="hidden sm:flex flex-col items-end">
          <div class="text-[clamp(15px,2vw,18px)] font-semibold text-text-primary tabular-nums leading-tight">
            {{ time }}
          </div>
          <div class="text-[clamp(10px,1.2vw,12px)] text-text-muted leading-tight">
            {{ date }}
          </div>
        </div>
      </div>
    </header>

    <!-- ============ Body ============ -->
    <section
      class="flex-1 flex flex-col gap-[clamp(12px,2vw,18px)]
             p-[clamp(16px,3vw,24px)]
             max-w-[820px] mx-auto w-full justify-center"
    >
      <!-- ╔══════════════════════════════════════════════════════════╗
           ║ Welcome banner besar — combo unDraw illustration +      ║
           ║ greeting time-aware + soft gradient bg                   ║
           ╚══════════════════════════════════════════════════════════╝ -->
      <div
        class="rounded-card flex items-center gap-[clamp(16px,3vw,28px)]
               px-[clamp(24px,4vw,40px)] py-[clamp(20px,3vw,32px)]
               min-h-[clamp(160px,20vw,220px)]"
        :style="{
          background: `linear-gradient(135deg, var(--color-primary-light, #E8F0FE) 0%, var(--color-accent, #DBEAFE) 100%)`,
        }"
      >
        <div class="flex-1 min-w-0">
          <div class="text-[clamp(28px,4.2vw,40px)] font-bold text-text-primary leading-tight">
            {{ greeting }}
          </div>
          <p class="text-[clamp(15px,2.2vw,20px)] text-text-secondary mt-2 leading-relaxed">
            Mari mulai pendaftaran Anda<br/>
            — pilih layanan di bawah.
          </p>
        </div>
        <!-- Illustration unDraw style — color follow theme primary -->
        <div
          class="flex-shrink-0 hidden md:flex"
          :style="{ color: 'var(--color-primary, #1B4FD8)' }"
        >
          <WelcomeIllustration size="lg" />
        </div>
      </div>

      <!-- ╔══════════════════════════════════════════════════════════╗
           ║ Hero BPJS — primary action 60% visual weight             ║
           ║ Gradient biru korporat + BPJS logo                       ║
           ╚══════════════════════════════════════════════════════════╝ -->
      <button
        type="button"
        class="text-left rounded-card transition-all
               border border-transparent
               hover:opacity-95 active:opacity-90
               flex items-center gap-[clamp(16px,2.5vw,24px)]
               px-[clamp(24px,3.5vw,32px)] py-[clamp(22px,3vw,30px)]
               min-h-[clamp(120px,15vw,150px)]
               shadow-lg text-white"
        :style="{
          background: `linear-gradient(135deg, var(--color-primary, #1B4FD8) 0%, var(--color-primary-dark, #143ba8) 100%)`,
        }"
        @click="startBPJS"
      >
        <!-- Logo BPJS Kesehatan resmi (atau dari config kalau ada) -->
        <div
          class="bg-white rounded-[14px] p-[clamp(10px,1.5vw,16px)] flex-shrink-0
                 flex items-center justify-center
                 w-[clamp(80px,11vw,108px)] h-[clamp(80px,11vw,108px)]
                 shadow-md"
        >
          <BpjsLogo size="md" variant="icon" />
        </div>
        <div class="flex-1">
          <div class="text-[clamp(22px,3vw,28px)] font-bold leading-tight">
            Pasien BPJS
          </div>
          <p class="text-[clamp(14px,1.8vw,16px)] opacity-95 mt-1.5 leading-snug">
            Ketik No. Kartu BPJS atau NIK — sistem otomatis mendeteksi jenis kunjungan
          </p>
        </div>
        <PhCaretRight :size="36" weight="bold" class="opacity-90" />
      </button>

      <!-- ╔══════════════════════════════════════════════════════════╗
           ║ 2 Secondary cards co-equal — gradient lembut per kategori║
           ║ Umum (hijau lembut) · Antrian Loket (kuning hangat)      ║
           ╚══════════════════════════════════════════════════════════╝ -->
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-[clamp(10px,1.5vw,16px)]">
        <!-- Pasien Umum — gradient hijau lembut -->
        <button
          type="button"
          class="text-left rounded-card transition-all
                 hover:opacity-95 active:opacity-90 border border-emerald-100
                 flex items-center gap-[clamp(12px,2vw,18px)]
                 px-[clamp(18px,2.5vw,24px)] py-[clamp(18px,2.5vw,24px)]
                 min-h-[clamp(96px,12vw,118px)] shadow-sm"
          style="background: linear-gradient(135deg, #ECFDF5 0%, #D1FAE5 100%)"
          @click="startUmum"
        >
          <div
            class="rounded-[12px] flex items-center justify-center flex-shrink-0
                   w-[clamp(54px,7vw,64px)] h-[clamp(54px,7vw,64px)]
                   bg-emerald-500 text-white shadow-sm"
          >
            <PhUser :size="32" weight="fill" />
          </div>
          <div class="flex-1 min-w-0">
            <div class="text-[clamp(17px,2.3vw,20px)] font-bold text-emerald-900 leading-tight">
              Pasien Umum
            </div>
            <p class="text-[clamp(12px,1.6vw,14px)] text-emerald-800/80 mt-1 leading-tight">
              Tanpa kartu BPJS · Bayar di kasir
            </p>
          </div>
          <PhCaretRight :size="22" weight="bold" class="text-emerald-700/70" />
        </button>

        <!-- Antrian Loket — gradient kuning hangat -->
        <button
          type="button"
          :disabled="antrianLoading"
          class="text-left rounded-card transition-all
                 hover:opacity-95 active:opacity-90 disabled:opacity-60 border border-amber-100
                 flex items-center gap-[clamp(12px,2vw,18px)]
                 px-[clamp(18px,2.5vw,24px)] py-[clamp(18px,2.5vw,24px)]
                 min-h-[clamp(96px,12vw,118px)] shadow-sm"
          style="background: linear-gradient(135deg, #FFFBEB 0%, #FEF3C7 100%)"
          @click="startAntrian"
        >
          <div
            class="rounded-[12px] flex items-center justify-center flex-shrink-0
                   w-[clamp(54px,7vw,64px)] h-[clamp(54px,7vw,64px)]
                   bg-amber-500 text-white shadow-sm"
          >
            <PhCircleNotch v-if="antrianLoading" :size="32" weight="bold" class="animate-spin" />
            <PhTicket v-else :size="32" weight="fill" />
          </div>
          <div class="flex-1 min-w-0">
            <div class="text-[clamp(17px,2.3vw,20px)] font-bold text-amber-900 leading-tight">
              Ambil Nomor Loket
            </div>
            <p class="text-[clamp(12px,1.6vw,14px)] text-amber-800/80 mt-1 leading-tight">
              {{ antrianLoading ? 'Mengambil nomor…' : 'Antrian admisi · cetak langsung' }}
            </p>
          </div>
          <PhCaretRight :size="22" weight="bold" class="text-amber-700/70" />
        </button>
      </div>
    </section>

    <!-- ============ Footer ============ -->
    <footer
      class="bg-surface border-t border-border
             flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-2
             px-[clamp(16px,3vw,28px)] py-[clamp(10px,1.8vw,14px)]"
    >
      <button
        type="button"
        class="flex items-center gap-2 text-[clamp(13px,1.7vw,15px)] font-medium
               px-[clamp(12px,1.8vw,16px)] py-[clamp(8px,1.3vw,12px)]
               rounded-btn hover:bg-bg active:bg-border"
        :style="{ color: 'var(--color-primary, #1B4FD8)' }"
        @click="startBantuSaya"
      >
        <PhLifebuoy :size="20" weight="bold" />
        <span>Pertama kali? Bantu saya</span>
      </button>

      <div class="flex items-center gap-3">
        <button
          type="button"
          class="flex items-center gap-2 text-[clamp(12px,1.5vw,14px)] text-text-muted
                 hover:text-text-primary px-3 py-2"
          @click="startSatuSehat"
        >
          <PhFingerprint :size="18" weight="bold" />
          <span class="hidden sm:inline">Aktivasi Satu Sehat</span>
        </button>
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
      </div>
    </footer>
  </main>
</template>
