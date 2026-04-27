<!--
  RegistrasiUmumScreen — flow lanjutan setelah SearchPasienScreen.

  Stepper internal 3 langkah:
    1. Pilih Poli — list dari GetPoliklinikAktif (poli yang punya jadwal)
    2. Pilih Dokter — list dari GetJadwalDokter(kdPoli) untuk hari ini
    3. Konfirmasi — review pasien + poli + dokter, klik "Daftar" → BuatPendaftaran

  Sukses:
    - Simpan hasil ke patient.lastPendaftaran
    - router.push /tiket (TicketScreen sudah handle render no_urut)

  Gagal: AlertModal — biarkan user retry / cancel ke home.

  Backend yang dipanggil (semua via direct-DB Khanza, mode mysql):
    - apmService.getPoliklinikAktif()
    - apmService.getJadwalDokter(kdPoli)
    - apmService.buatPendaftaran({ NoRM, KdPoli, KdDokter, TglPeriksa, Penjamin: 'UMUM' })
-->
<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'

import { PhBuildings, PhStethoscope, PhCheckSquare } from '@phosphor-icons/vue'
import DokterPicker from '../components/DokterPicker.vue'
import IdleOverlay from '../components/IdleOverlay.vue'
import AlertModal from '../components/AlertModal.vue'
import StepperBar from '../components/StepperBar.vue'

import { I18N } from '../constants/i18n'
import { KIOSK } from '../constants/kiosk'
import { useIdleTimeout } from '../composables/useIdleTimeout'
import { apmService } from '../services/apm'
import { usePatientStore } from '../stores/patient'

const router = useRouter()
const patient = usePatientStore()

if (!patient.pasienUmum) {
  router.replace({ name: 'home' })
}

// step: 'poli' | 'dokter' | 'konfirmasi' | 'submitting'
const step = ref('poli')

const poliList = ref([])      // Poliklinik[]
const poliLoading = ref(false)

const dokterList = ref([])    // JadwalDokter[]
const dokterLoading = ref(false)
const selectedDokterKd = ref('')

const errorVisible = ref(false)
const errorMsg = ref('')

const pasien = computed(() => patient.pasienUmum)
const selectedPoli = computed(() => patient.selectedPoli)
const selectedDokter = computed(() =>
  dokterList.value.find((d) => d.KdDokter === selectedDokterKd.value) ?? null
)

// ----- Step 1: Poli -----
async function loadPoli() {
  poliLoading.value = true
  try {
    poliList.value = await apmService.getPoliklinikAktif()
  } catch (e) {
    showError('Gagal memuat daftar poli', e)
  } finally {
    poliLoading.value = false
  }
}

function pickPoli(p) {
  patient.setSelectedPoli(p)
  selectedDokterKd.value = ''
  step.value = 'dokter'
  loadDokter(p.kd_poli)
}

// ----- Step 2: Dokter -----
async function loadDokter(kdPoli) {
  dokterLoading.value = true
  dokterList.value = []
  try {
    const list = await apmService.getJadwalDokter(kdPoli)
    dokterList.value = list || []
    // Default pilih dokter pertama yang aktif
    const firstAktif = dokterList.value.find((d) => d.Aktif)
    if (firstAktif) selectedDokterKd.value = firstAktif.KdDokter
  } catch (e) {
    showError('Gagal memuat jadwal dokter', e)
  } finally {
    dokterLoading.value = false
  }
}

function backToPoli() {
  step.value = 'poli'
  selectedDokterKd.value = ''
}

function lanjutKonfirmasi() {
  if (!selectedDokter.value) {
    showError('Pilih dokter dulu', 'Belum ada dokter yang dipilih.')
    return
  }
  patient.setSelectedDokter(selectedDokter.value)
  step.value = 'konfirmasi'
}

// ----- Step 3: Konfirmasi -----
async function submitDaftar() {
  if (!pasien.value || !selectedPoli.value || !selectedDokter.value) return
  step.value = 'submitting'
  try {
    const today = new Date()
    const tglPeriksa = `${today.getFullYear()}-${String(today.getMonth()+1).padStart(2,'0')}-${String(today.getDate()).padStart(2,'0')}`
    const result = await apmService.buatPendaftaran({
      NoRM: pasien.value.NoRM,
      KdPoli: selectedPoli.value.kd_poli,
      KdDokter: selectedDokter.value.KdDokter,
      TglPeriksa: tglPeriksa,
      Penjamin: 'UMUM',
      JamPeriksa: '',
      NoSEP: '',
      Catatan: '',
    })
    patient.setLastPendaftaran(result)
    router.push({ name: 'tiket' })
  } catch (e) {
    step.value = 'konfirmasi'
    showError('Gagal mendaftarkan pasien', e)
  }
}

function backToDokter() {
  step.value = 'dokter'
}

// ----- Util -----
function showError(title, e) {
  errorMsg.value = e && e.message ? e.message : (typeof e === 'string' ? e : String(e))
  errorVisible.value = true
}

function back() {
  if (step.value === 'dokter') {
    backToPoli()
  } else if (step.value === 'konfirmasi') {
    backToDokter()
  } else {
    router.push({ name: 'cari-pasien' })
  }
}

function pulang() {
  router.replace({ name: 'home' })
}

function jkLabel(jk) {
  return jk === 'L' ? 'Laki-laki' : jk === 'P' ? 'Perempuan' : '-'
}
function rupiah(n) {
  if (!n || n === 0) return 'Gratis'
  try { return 'Rp ' + Number(n).toLocaleString('id-ID') } catch { return n }
}

const stepLabel = computed(() => {
  switch (step.value) {
    case 'poli':       return '1 / 3 — Pilih Poli'
    case 'dokter':     return '2 / 3 — Pilih Dokter'
    case 'konfirmasi': return '3 / 3 — Konfirmasi Pendaftaran'
    case 'submitting': return 'Menyimpan…'
    default:           return ''
  }
})

// StepperBar config — 3 step dengan icon Phosphor
const stepperSteps = [
  { label: 'Pilih Poli', icon: PhBuildings },
  { label: 'Pilih Dokter', icon: PhStethoscope },
  { label: 'Konfirmasi', icon: PhCheckSquare },
]
const stepperCurrentIndex = computed(() => {
  switch (step.value) {
    case 'poli': return 0
    case 'dokter': return 1
    case 'konfirmasi':
    case 'submitting': return 2
    default: return 0
  }
})
function onStepperClick(idx) {
  // User tap step yang sudah lewat → balik ke step itu (hanya kalau tidak submitting)
  if (step.value === 'submitting') return
  if (idx === 0) backToPoli()
  else if (idx === 1) backToDokter()
}

const { isCountingDown, secondsLeft } = useIdleTimeout({
  totalSeconds: KIOSK.idleTimeoutSec,
  countdownThreshold: KIOSK.idleCountdownSec,
  onTimeout: async () => {
    await patient.reset()
    router.push({ name: 'home' })
  },
})

onMounted(loadPoli)
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
      <h1 class="text-[clamp(15px,2.2vw,18px)] font-medium text-text-primary truncate">
        Pasien Umum
      </h1>
      <span class="ml-auto text-[clamp(10px,1.4vw,12px)] text-text-muted">
        {{ stepLabel }}
      </span>
    </header>

    <!-- StepperBar v1.2 — visual progress segmented (was tiny text in header) -->
    <div class="bg-surface px-[clamp(16px,3vw,28px)] py-[clamp(8px,1.2vw,12px)] border-b border-border">
      <StepperBar
        :steps="stepperSteps"
        :current-index="stepperCurrentIndex"
        :clickable="true"
        @step-click="onStepperClick"
      />
    </div>

    <!-- Body -->
    <section
      class="flex-1 flex flex-col gap-[clamp(10px,2vw,16px)]
             p-[clamp(12px,2.5vw,20px)]
             max-w-[680px] mx-auto w-full"
    >
      <!-- Pasien card (selalu tampil di atas) -->
      <div v-if="pasien"
           class="bg-surface border border-border rounded-card
                  px-[clamp(12px,2vw,16px)] py-[clamp(10px,1.6vw,12px)]
                  flex items-center gap-[clamp(10px,1.5vw,14px)]">
        <div class="bg-blue-light text-blue-dark rounded-full flex items-center justify-center
                    w-[clamp(36px,5vw,42px)] h-[clamp(36px,5vw,42px)] flex-shrink-0">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
               stroke="currentColor" stroke-width="2"
               stroke-linecap="round" stroke-linejoin="round"
               class="w-[clamp(18px,2.4vw,22px)] h-[clamp(18px,2.4vw,22px)]">
            <circle cx="12" cy="8" r="4"/>
            <path d="M4 21v-2a4 4 0 0 1 4-4h8a4 4 0 0 1 4 4v2"/>
          </svg>
        </div>
        <div class="min-w-0 flex-1">
          <div class="text-[clamp(13px,1.8vw,15px)] font-medium text-text-primary leading-tight truncate">
            {{ pasien.Nama }}
          </div>
          <div class="text-[clamp(10px,1.3vw,12px)] text-text-muted leading-tight">
            No. RM {{ pasien.NoRM }} · {{ jkLabel(pasien.JK) }}
          </div>
        </div>
      </div>

      <!-- ============ Step 1: Poli ============ -->
      <template v-if="step === 'poli'">
        <h2 class="text-[clamp(14px,2vw,16px)] font-medium text-text-primary">
          Pilih Poli Tujuan
        </h2>

        <div v-if="poliLoading"
             class="bg-surface border border-border rounded-card
                    p-[clamp(16px,2.5vw,20px)] text-center
                    text-[clamp(11px,1.5vw,13px)] text-text-muted">
          Memuat daftar poli…
        </div>

        <div v-else class="flex flex-col gap-[clamp(6px,1vw,8px)]">
          <p v-if="poliList.length === 0"
             class="text-[clamp(11px,1.5vw,13px)] text-text-muted text-center
                    py-4 bg-bg rounded-card border border-border">
            Belum ada poli yang menerima pendaftaran hari ini.
          </p>
          <button
            v-for="p in poliList"
            :key="p.kd_poli"
            type="button"
            class="rounded-card border border-border bg-surface
                   px-[clamp(12px,2vw,16px)] py-[clamp(12px,1.8vw,14px)]
                   min-h-[clamp(56px,7vw,68px)]
                   flex items-center gap-[clamp(10px,1.5vw,14px)]
                   hover:border-border-strong active:bg-bg text-left"
            @click="pickPoli(p)"
          >
            <div class="bg-blue-light text-blue rounded-[8px] flex items-center justify-center
                        w-[clamp(34px,4.5vw,40px)] h-[clamp(34px,4.5vw,40px)] flex-shrink-0">
              <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
                   stroke="currentColor" stroke-width="2"
                   stroke-linecap="round" stroke-linejoin="round"
                   class="w-[clamp(16px,2.2vw,20px)] h-[clamp(16px,2.2vw,20px)]">
                <path d="M19 14H5"/><path d="M12 8v12"/>
                <rect x="3" y="3" width="18" height="6" rx="2"/>
              </svg>
            </div>
            <div class="flex-1 min-w-0">
              <div class="text-[clamp(13px,1.8vw,15px)] font-medium text-text-primary leading-tight">
                {{ p.nm_poli }}
              </div>
              <div class="text-[clamp(10px,1.3vw,12px)] text-text-muted mt-1 leading-tight">
                Tarif registrasi: {{ rupiah(p.registrasi_lama || p.registrasi) }}
              </div>
            </div>
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
                 stroke="currentColor" stroke-width="2.5"
                 stroke-linecap="round" stroke-linejoin="round"
                 class="w-[clamp(14px,1.8vw,16px)] h-[clamp(14px,1.8vw,16px)] text-text-muted">
              <polyline points="9 18 15 12 9 6"/>
            </svg>
          </button>
        </div>
      </template>

      <!-- ============ Step 2: Dokter ============ -->
      <template v-else-if="step === 'dokter'">
        <h2 class="text-[clamp(14px,2vw,16px)] font-medium text-text-primary">
          Pilih Dokter — {{ selectedPoli?.nm_poli }}
        </h2>

        <div v-if="dokterLoading"
             class="bg-surface border border-border rounded-card
                    p-[clamp(16px,2.5vw,20px)] text-center
                    text-[clamp(11px,1.5vw,13px)] text-text-muted">
          Memuat jadwal dokter…
        </div>

        <DokterPicker
          v-else
          :list="dokterList"
          v-model="selectedDokterKd"
        />

        <div class="flex gap-[clamp(8px,1.2vw,10px)] mt-[clamp(8px,1.2vw,10px)]">
          <button
            type="button"
            class="flex-1 bg-surface border border-border text-text-primary font-medium
                   rounded-btn py-[clamp(12px,1.8vw,14px)]
                   text-[clamp(13px,1.8vw,15px)]
                   hover:border-border-strong
                   min-h-[clamp(48px,6vw,56px)]"
            @click="backToPoli"
          >
            Kembali
          </button>
          <button
            type="button"
            class="flex-2 bg-blue text-white font-medium rounded-btn
                   py-[clamp(12px,1.8vw,14px)]
                   text-[clamp(13px,1.8vw,15px)]
                   hover:bg-blue-dark active:bg-blue-dark/90 disabled:opacity-50
                   min-h-[clamp(48px,6vw,56px)]"
            style="flex:2 1 0%"
            :disabled="!selectedDokterKd || dokterLoading"
            @click="lanjutKonfirmasi"
          >
            Lanjut →
          </button>
        </div>
      </template>

      <!-- ============ Step 3: Konfirmasi ============ -->
      <template v-else>
        <h2 class="text-[clamp(14px,2vw,16px)] font-medium text-text-primary">
          Konfirmasi Pendaftaran
        </h2>

        <div class="bg-surface border border-border rounded-card
                    p-[clamp(14px,2.2vw,18px)] flex flex-col gap-[clamp(10px,1.5vw,12px)]">
          <dl class="grid grid-cols-3 gap-x-[clamp(8px,1.5vw,12px)] gap-y-[clamp(6px,1vw,8px)]
                     text-[clamp(11px,1.5vw,13px)]">
            <dt class="text-text-muted col-span-1">Poli</dt>
            <dd class="text-text-primary col-span-2 font-medium">
              {{ selectedPoli?.nm_poli }} ({{ selectedPoli?.kd_poli }})
            </dd>
            <dt class="text-text-muted col-span-1">Dokter</dt>
            <dd class="text-text-primary col-span-2 font-medium">
              {{ selectedDokter?.NmDokter }}
            </dd>
            <dt class="text-text-muted col-span-1">Jam Praktik</dt>
            <dd class="text-text-primary col-span-2">
              {{ selectedDokter?.JamMulai }}–{{ selectedDokter?.JamSelesai }}
            </dd>
            <dt class="text-text-muted col-span-1">Penjamin</dt>
            <dd class="text-text-primary col-span-2">Umum / Tunai</dd>
            <dt class="text-text-muted col-span-1">Tarif</dt>
            <dd class="text-text-primary col-span-2">
              {{ rupiah(selectedPoli?.registrasi_lama || selectedPoli?.registrasi) }}
            </dd>
          </dl>
          <div class="text-[clamp(10px,1.3vw,12px)] text-text-muted
                      bg-bg rounded-btn px-[clamp(10px,1.5vw,12px)] py-[clamp(6px,1vw,8px)]">
            Pembayaran dilakukan di kasir setelah pemeriksaan.
            Silakan menunggu nomor antrian Anda dipanggil.
          </div>
        </div>

        <div class="flex gap-[clamp(8px,1.2vw,10px)] mt-[clamp(6px,1vw,8px)]">
          <button
            type="button"
            class="flex-1 bg-surface border border-border text-text-primary font-medium
                   rounded-btn py-[clamp(12px,1.8vw,14px)]
                   text-[clamp(13px,1.8vw,15px)]
                   hover:border-border-strong
                   min-h-[clamp(48px,6vw,56px)]"
            :disabled="step === 'submitting'"
            @click="backToDokter"
          >
            Kembali
          </button>
          <button
            type="button"
            class="bg-blue text-white font-medium rounded-btn
                   py-[clamp(12px,1.8vw,14px)] px-[clamp(20px,3vw,28px)]
                   text-[clamp(13px,1.8vw,15px)]
                   hover:bg-blue-dark active:bg-blue-dark/90 disabled:opacity-50
                   min-h-[clamp(48px,6vw,56px)]
                   flex-2 flex items-center justify-center gap-2"
            style="flex:2 1 0%"
            :disabled="step === 'submitting'"
            @click="submitDaftar"
          >
            <svg v-if="step === 'submitting'"
                 class="animate-spin h-4 w-4 text-white" xmlns="http://www.w3.org/2000/svg"
                 fill="none" viewBox="0 0 24 24">
              <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
              <path class="opacity-75" fill="currentColor"
                    d="M4 12a8 8 0 0 1 8-8V0C5.4 0 0 5.4 0 12h4zm2 5.3A8 8 0 0 1 4 12H0c0 3 1.1 5.8 3 7.9l3-2.6z"/>
            </svg>
            <span v-if="step === 'submitting'">Menyimpan…</span>
            <span v-else>Daftar Sekarang</span>
          </button>
        </div>
      </template>
    </section>

    <AlertModal
      :visible="errorVisible"
      title="Terjadi kesalahan"
      :message="errorMsg"
      @close="errorVisible = false"
    />

    <IdleOverlay :seconds-left="secondsLeft" :visible="isCountingDown" />
  </main>
</template>
