<!--
  ResultScreen — render berbeda berdasarkan PatientType (Smart Detector).

  Phase D refactor:
    - Setiap PatientType punya path render dedicated (komponen Pathway*).
    - Kontrol & RujukanBaru tetap inline supaya logic existing (load
      jadwal dokter, BuatSEPKontrol, info bar biometrik) tidak ter-
      regress.
    - Pathway baru: MJKN, PostRANAP, PostRAJAL, TidakAktif. Backend
      Smart Detector enhancement (Phase B) akan inject Data payload
      sesuai shape yg di-doc di props masing-masing komponen.
    - Render aman bila Data null — komponen anak punya empty state.

  Spec ketat P-043 (legacy):
    - PatientCard dengan 4 pill warna (per kategori).
    - Kontrol: surat kontrol detail + DokterPicker (default idx 0) +
      CTA "Buat surat layanan" (await BuatSEPKontrol).
    - RujukanBaru: detail rujukan + info bar biometrik CONDITIONAL
      (hanya kalau perluBiometrik = umur>=17 + non-IGD).
    - TidakAktif: info bar merah + CTA "Daftar pasien umum" + ghost
      "Hubungi petugas".
    - Loading state: CTA disabled + spinner saat API call.
    - Error: AlertModal dengan pesan dari domain.UserMessage().
-->
<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'

import PatientCard from '../components/PatientCard.vue'
import IdleOverlay from '../components/IdleOverlay.vue'
import AlertModal from '../components/AlertModal.vue'
import DokterPicker from '../components/DokterPicker.vue'
import BiometrikChoiceModal from '../components/BiometrikChoiceModal.vue'
import PathwayMJKN from '../components/PathwayMJKN.vue'
import PathwayPostRANAP from '../components/PathwayPostRANAP.vue'
import PathwayPostRAJAL from '../components/PathwayPostRAJAL.vue'
import PathwayTidakAktif from '../components/PathwayTidakAktif.vue'

import { I18N } from '../constants/i18n'
import { KIOSK } from '../constants/kiosk'
import { useIdleTimeout } from '../composables/useIdleTimeout'
import { apmService, PatientType, BIOMETRIK_REQUIRED_HINT } from '../services/apm'
import { usePatientStore } from '../stores/patient'

const router = useRouter()
const patient = usePatientStore()

if (!patient.detectionResult && !patient.isDetecting) {
  router.replace({ name: 'home' })
}

const result = computed(() => patient.detectionResult)
const peserta = computed(() => patient.peserta)
const ptype = computed(() => result.value?.Type ?? PatientType.Unknown)

// State CTA & error
const ctaLoading = ref(false)
const errorVisible = ref(false)
const errorMsg = ref('')

// Biometrik modal state — muncul saat backend return error "biometrik diperlukan".
// Tipe: 'kontrol' | 'rujukan' | null. Menentukan flow retry mana yang dipanggil
// setelah token didapat.
const showBiometrikModal = ref(false)
const biometrikContext = ref(null) // 'kontrol' | 'rujukan' | null
const biometrikLoading = ref(false)

// Dokter picker state (untuk Kontrol)
const dokterList = ref([]) // JadwalDokter[]
const selectedDokter = ref('')

// Pill config
const pillConfig = computed(() => {
  switch (ptype.value) {
    case PatientType.MJKN: return { label: 'Booking Mobile JKN', variant: 'success' }
    case PatientType.Kontrol: return { label: 'Jadwal Kontrol', variant: 'info' }
    case PatientType.PostRANAP: return { label: 'Pasca Rawat Inap', variant: 'info' }
    case PatientType.PostRAJAL: return { label: 'Lanjutan Rawat Jalan', variant: 'info' }
    case PatientType.RujukanBaru: return { label: 'Kunjungan Baru', variant: 'warning' }
    case PatientType.TidakAktif: return { label: 'Status Tidak Aktif', variant: 'danger' }
    default: return { label: 'Tidak Diketahui', variant: 'danger' }
  }
})

function formatDate(d) {
  if (!d) return ''
  try {
    const dt = new Date(d)
    if (isNaN(dt.getTime())) return ''
    const BULAN = ['Jan','Feb','Mar','Apr','Mei','Jun','Jul','Agu','Sep','Okt','Nov','Des']
    return `${dt.getDate()} ${BULAN[dt.getMonth()]} ${dt.getFullYear()}`
  } catch { return '' }
}
const dateLabel = computed(() => formatDate(result.value?.DetectedAt))

// Compute perluBiometrik di frontend (mirror dari Go domain rule):
//   age < 17 → false
//   kdPoli IGD/UGD → false
//   else → true
function computeAgeYears(tglLahir) {
  if (!tglLahir) return 0
  const t = new Date(tglLahir)
  if (isNaN(t.getTime())) return 0
  const now = new Date()
  let years = now.getFullYear() - t.getFullYear()
  if (now.getMonth() < t.getMonth() ||
      (now.getMonth() === t.getMonth() && now.getDate() < t.getDate())) {
    years--
  }
  return Math.max(0, years)
}
function isIGD(kdPoli) {
  const u = (kdPoli || '').toUpperCase().trim()
  return u.startsWith('IGD') || u.startsWith('UGD') || u === 'EMR'
}
const kdPoliRujukan = computed(() => {
  // Untuk RujukanBaru, kdPoli dari rujukan FKTP — kita tidak punya
  // langsung di detection result. Pakai detail kosong → default
  // butuh biometrik kecuali pasti IGD.
  return ''
})
const perluBiometrik = computed(() => {
  if (!peserta.value) return false
  const age = computeAgeYears(peserta.value.TglLahir)
  if (age < 17) return false
  if (isIGD(kdPoliRujukan.value)) return false
  return true
})

// Detail rows untuk PatientCard — base (selalu) + Kontrol/RujukanBaru
// (yang masih inline). Pathway lain render detail di komponen mereka.
const details = computed(() => {
  const p = peserta.value
  if (!p) return []
  const base = [
    { key: 'Nomor RM', value: p.NoRM || '—' },
    { key: 'Tgl. lahir', value: p.TglLahir || '—' },
    { key: 'Kelas hak', value: p.KelasHak ? `Kelas ${p.KelasHak}` : '—' },
  ]
  if (ptype.value === PatientType.Kontrol) {
    const list = result.value?.Data ?? []
    const sk = Array.isArray(list) ? list[0] : list
    if (sk) {
      base.push(
        { key: 'No surat kontrol', value: sk.NoSurat || '—' },
        { key: 'Tgl rencana', value: sk.TglRencana || '—' },
        { key: 'Poli', value: sk.NmPoli || sk.KdPoli || '—', accent: true },
      )
    }
  }
  // RujukanBaru: tidak ada detail rujukan FKTP di detection result.
  // Service layer akan fetch saat BuatSEPRujukan; placeholder kosong.
  return base
})

// Get poli code untuk lookup jadwal dokter (Kontrol)
const kdPoliKontrol = computed(() => {
  const list = result.value?.Data ?? []
  const sk = Array.isArray(list) ? list[0] : list
  return sk?.KdPoli ?? ''
})

// Pathway-specific data accessors (gunakan optional chaining supaya
// aman kalau Data null — komponen anak menampilkan empty state).
const bookingMJKN = computed(() => result.value?.Data ?? null)
const riwayatRANAP = computed(() => {
  // Backend bisa kirim object langsung atau array (latest first) —
  // normalize ke single object.
  const d = result.value?.Data
  if (Array.isArray(d)) return d[0] ?? null
  return d ?? null
})
const kunjunganRAJAL = computed(() => {
  const d = result.value?.Data
  if (Array.isArray(d)) return d[0] ?? null
  return d ?? null
})

// Load jadwal dokter saat screen mount kalau Kontrol
onMounted(async () => {
  if (ptype.value === PatientType.Kontrol && kdPoliKontrol.value) {
    try {
      const list = await apmService.getJadwalDokter(kdPoliKontrol.value)
      dokterList.value = list ?? []
      // Default: pilih dokter aktif pertama
      const firstAktif = (list ?? []).find((d) => d.Aktif)
      if (firstAktif) selectedDokter.value = firstAktif.KdDokter
    } catch (e) {
      // Jadwal tidak tersedia — UI tampilkan empty state via DokterPicker
    }
  }
})

// =====================================================================
// CTA actions per pathway
// =====================================================================

async function onConfirmMJKN() {
  if (ctaLoading.value) return
  ctaLoading.value = true
  try {
    // BuatCheckinMJKN belum ada di backend (Phase B/E TBD).
    // Sementara langsung navigate ke tiket — backend akan inject SEP/
    // ticket data saat endpoint tersedia.
    console.log('[PathwayMJKN] confirm → backend BuatCheckinMJKN belum siap, fallback ke tiket')
    router.push({ name: 'tiket', query: { from: 'mjkn' } })
  } catch (e) {
    errorMsg.value = e?.message ?? String(e)
    errorVisible.value = true
  } finally {
    ctaLoading.value = false
  }
}

// Helper: detect kalau error message dari backend mengindikasikan biometrik
// diperlukan (sentinel BIOMETRIK_REQUIRED_HINT). Backend agent emit pesan
// yang contains substring "biometrik diperlukan".
function isBiometrikRequiredErr(e) {
  const msg = String(e?.message ?? e ?? '').toLowerCase()
  return msg.includes(BIOMETRIK_REQUIRED_HINT.toLowerCase())
}

async function onIssueSEPKontrol() {
  if (ctaLoading.value) return
  ctaLoading.value = true
  try {
    if (!selectedDokter.value) {
      throw new Error('Silakan pilih dokter terlebih dahulu')
    }
    const list = result.value?.Data ?? []
    const sk = Array.isArray(list) ? list[0] : list
    if (!sk?.NoSurat) {
      throw new Error('Surat kontrol tidak ditemukan')
    }
    const sep = await apmService.buatSEPKontrol(sk.NoSurat, selectedDokter.value)
    patient.setLastSEP(sep)
    router.push({ name: 'tiket', query: { from: 'kontrol' } })
  } catch (e) {
    // Jika backend bilang biometrik dibutuhkan, buka modal pilihan;
    // jangan tampilkan error modal — UX pasien tidak perlu lihat pesan teknis.
    if (isBiometrikRequiredErr(e)) {
      biometrikContext.value = 'kontrol'
      showBiometrikModal.value = true
    } else {
      errorMsg.value = e?.message ?? String(e)
      errorVisible.value = true
    }
  } finally {
    ctaLoading.value = false
  }
}

async function onLanjutPostRANAP() {
  if (ctaLoading.value) return
  ctaLoading.value = true
  try {
    // P-046+ akan punya DokterPickerScreen dedicated. Sementara
    // navigate ke tiket; backend akan handle SEP creation.
    console.log('[PathwayPostRANAP] lanjut → DokterPickerScreen belum siap')
    router.push({ name: 'tiket', query: { from: 'postranap' } })
  } catch (e) {
    errorMsg.value = e?.message ?? String(e)
    errorVisible.value = true
  } finally {
    ctaLoading.value = false
  }
}

async function onLanjutPostRAJAL() {
  if (ctaLoading.value) return
  ctaLoading.value = true
  try {
    console.log('[PathwayPostRAJAL] lanjut → DokterPickerScreen belum siap')
    router.push({ name: 'tiket', query: { from: 'postrajal' } })
  } catch (e) {
    errorMsg.value = e?.message ?? String(e)
    errorVisible.value = true
  } finally {
    ctaLoading.value = false
  }
}

async function onIssueSEPRujukan() {
  if (ctaLoading.value) return
  ctaLoading.value = true
  try {
    // P-046+ akan punya DokterPickerScreen tersendiri untuk RujukanBaru.
    // Sementara: kalau perluBiometrik = true, kita preemptive buka modal
    // (pasien dewasa non-IGD selalu butuh biometrik per spec). Hindari
    // round-trip ke backend hanya untuk dapat error "biometrik diperlukan".
    if (perluBiometrik.value) {
      biometrikContext.value = 'rujukan'
      showBiometrikModal.value = true
      return
    }
    router.push({ name: 'tiket', query: { from: 'rujukan' } })
  } catch (e) {
    if (isBiometrikRequiredErr(e)) {
      biometrikContext.value = 'rujukan'
      showBiometrikModal.value = true
    } else {
      errorMsg.value = e?.message ?? String(e)
      errorVisible.value = true
    }
  } finally {
    ctaLoading.value = false
  }
}

// =====================================================================
// Biometrik modal — handler pick & cancel
// =====================================================================

async function onPickBiometrik(method) {
  // method: 'face' | 'fingerprint'
  if (biometrikLoading.value) return
  if (!peserta.value?.NoKartu) {
    errorMsg.value = 'Data peserta tidak lengkap untuk verifikasi biometrik.'
    errorVisible.value = true
    showBiometrikModal.value = false
    return
  }

  biometrikLoading.value = true
  try {
    const noKartu = peserta.value.NoKartu
    const token = method === 'face'
      ? await apmService.verifikasiWajah(noKartu)
      : await apmService.verifikasiSidikJari(noKartu)
    if (!token) {
      throw new Error('Verifikasi tidak menghasilkan token. Silakan coba lagi.')
    }

    // Tutup modal lalu retry SEP creation dengan token.
    showBiometrikModal.value = false

    if (biometrikContext.value === 'kontrol') {
      const list = result.value?.Data ?? []
      const sk = Array.isArray(list) ? list[0] : list
      if (!sk?.NoSurat) throw new Error('Surat kontrol tidak ditemukan')
      // BuatSEPKontrol signature saat ini: (noSurat, kdDokter). Backend
      // agent akan inject FPToken via session cache (keset oleh
      // VerifikasiWajah/VerifikasiSidikJari) sehingga signature tidak
      // perlu berubah. Kalau signature berubah jadi (noSurat, kdDokter, token),
      // tinggal forward token di sini.
      ctaLoading.value = true
      try {
        const sep = await apmService.buatSEPKontrol(sk.NoSurat, selectedDokter.value)
        patient.setLastSEP(sep)
        router.push({ name: 'tiket', query: { from: 'kontrol' } })
      } finally {
        ctaLoading.value = false
      }
    } else if (biometrikContext.value === 'rujukan') {
      // P-046+: SEPRequest construction lengkap masih TBD — sementara
      // navigate ke tiket; backend simpan token untuk dipakai saat builder
      // SEP rujukan jalan.
      router.push({ name: 'tiket', query: { from: 'rujukan' } })
    }

    biometrikContext.value = null
  } catch (e) {
    // Verify gagal (timeout / cancel hardware) — tutup modal, tampilkan error toast
    showBiometrikModal.value = false
    biometrikContext.value = null
    errorMsg.value = e?.message ?? String(e)
    errorVisible.value = true
  } finally {
    biometrikLoading.value = false
  }
}

function onCancelBiometrik() {
  if (biometrikLoading.value) return
  showBiometrikModal.value = false
  biometrikContext.value = null
}

async function onDaftarUmum() {
  await patient.reset()
  router.push({ name: 'input', query: { mode: 'umum' } })
}

function onHubungiPetugas() {
  // P-046 admin panel akan emit event "staff:call" → audible alert.
  // Sementara no-op visual.
}

async function ghostAction() {
  if (ptype.value === PatientType.TidakAktif || ptype.value === PatientType.Error) {
    // "Hubungi petugas" — sementara no-op visual
    return
  }
  await patient.reset()
  router.push({ name: 'input', query: { mode: 'bpjs' } })
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

// Ghost button label per kategori (legacy back/Bukan saya).
const ghostLabel = computed(() => {
  if (ptype.value === PatientType.TidakAktif) return 'Hubungi petugas untuk bantuan'
  if (ptype.value === PatientType.Error) return 'Hubungi petugas'
  return 'Bukan saya — masukkan ulang'
})

// Apakah ghost button perlu di-render di footer? PathwayTidakAktif
// sudah punya CTA daftar-umum + ghost hubungi-petugas sendiri, jadi
// jangan duplicate ghost di parent. Sama untuk Error (no CTA).
const showFooterGhost = computed(() =>
  ptype.value !== PatientType.TidakAktif,
)

// Info bar untuk RujukanBaru (conditional biometrik) — masih di parent
// karena logic-nya bergantung peserta + age yang dihitung di sini.
const rujukanInfoBar = computed(() => {
  if (ptype.value !== PatientType.RujukanBaru) return null
  if (perluBiometrik.value) {
    return { variant: 'warning', text: 'Verifikasi biometrik (sidik wajah atau sidik jari) diperlukan setelah pilih dokter.' }
  }
  return null
})

// Info bar untuk Error pathway.
const errorInfoBar = computed(() => {
  if (ptype.value !== PatientType.Error) return null
  return { variant: 'danger', text: 'Sistem tidak dapat memeriksa status Anda saat ini. Silakan hubungi petugas.' }
})

const infoBarClass = (v) => {
  switch (v) {
    case 'success': return 'bg-success-bg text-success border-success-border'
    case 'warning': return 'bg-warning-bg text-warning border-warning-border'
    case 'danger': return 'bg-danger-bg text-danger border-danger-border'
    default: return 'bg-bg text-text-secondary border-border'
  }
}
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
        @click="ghostAction"
        :disabled="ctaLoading"
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
        Hasil pemeriksaan
      </h1>
    </header>

    <!-- Body -->
    <section
      class="flex-1 flex flex-col gap-[clamp(10px,2vw,16px)]
             p-[clamp(12px,2.5vw,20px)]
             max-w-[560px] mx-auto w-full"
    >
      <PatientCard
        v-if="peserta"
        :pill-label="pillConfig.label"
        :pill-variant="pillConfig.variant"
        :date-label="dateLabel"
        :nama="peserta.Nama"
        :no-kartu="peserta.NoKartu"
        :details="details"
      />
      <div
        v-else
        class="bg-surface border border-border rounded-card p-6 text-center"
      >
        <p class="text-[clamp(13px,1.8vw,16px)] font-medium text-text-primary">
          Data peserta tidak tersedia
        </p>
        <p class="text-[clamp(11px,1.5vw,13px)] text-text-muted mt-2">
          Sistem belum bisa memeriksa data Anda. Silakan coba lagi atau hubungi petugas.
        </p>
      </div>

      <!-- ====== Per-pathway slot ====== -->

      <!-- MJKN: tampilkan booking detail + CTA confirm -->
      <PathwayMJKN
        v-if="ptype === PatientType.MJKN"
        :booking="bookingMJKN"
        :loading="ctaLoading"
        @confirm="onConfirmMJKN"
      />

      <!-- Kontrol: dokter picker + CTA buat surat layanan (inline,
           preserve flow existing) -->
      <template v-else-if="ptype === PatientType.Kontrol">
        <div class="flex flex-col gap-[clamp(8px,1.2vw,10px)]">
          <p class="text-[clamp(11px,1.5vw,13px)] text-text-secondary font-medium uppercase tracking-wide">
            Pilih dokter
          </p>
          <DokterPicker v-model="selectedDokter" :list="dokterList" />
        </div>
        <button
          type="button"
          :disabled="ctaLoading"
          :class="[
            'w-full rounded-kiosk transition-opacity active:opacity-85',
            'bg-blue text-white border border-blue',
            'px-[clamp(14px,2.5vw,20px)] py-[clamp(14px,2.5vw,20px)]',
            'text-[clamp(14px,2vw,17px)] font-medium',
            'flex items-center justify-between gap-3',
            'min-h-[clamp(56px,8vw,72px)]',
            'disabled:opacity-60 disabled:cursor-not-allowed',
          ]"
          @click="onIssueSEPKontrol"
        >
          <span class="flex items-center gap-2">
            <svg
              v-if="ctaLoading"
              class="animate-spin w-5 h-5"
              viewBox="0 0 24 24" fill="none"
            >
              <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3"
                      fill="none" stroke-dasharray="40" stroke-dashoffset="20" />
            </svg>
            {{ ctaLoading ? 'Memproses...' : 'Buat surat layanan kontrol dan cetak' }}
          </span>
          <svg
            v-if="!ctaLoading"
            xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
            stroke="currentColor" stroke-width="2.5" stroke-linecap="round"
            stroke-linejoin="round" class="w-5 h-5 shrink-0"
          >
            <polyline points="9 18 15 12 9 6" />
          </svg>
        </button>
      </template>

      <!-- PostRANAP -->
      <PathwayPostRANAP
        v-else-if="ptype === PatientType.PostRANAP"
        :riwayat="riwayatRANAP"
        :loading="ctaLoading"
        @lanjut="onLanjutPostRANAP"
      />

      <!-- PostRAJAL -->
      <PathwayPostRAJAL
        v-else-if="ptype === PatientType.PostRAJAL"
        :kunjungan="kunjunganRAJAL"
        :loading="ctaLoading"
        @lanjut="onLanjutPostRAJAL"
      />

      <!-- RujukanBaru: info bar biometrik conditional + CTA pilih dokter
           (inline — DokterPickerScreen masuk P-046, sementara CTA navigate
           langsung ke tiket sesuai behavior existing) -->
      <template v-else-if="ptype === PatientType.RujukanBaru">
        <div
          v-if="rujukanInfoBar"
          :class="['rounded-card border p-[clamp(10px,1.8vw,14px)] flex items-start gap-2', infoBarClass(rujukanInfoBar.variant)]"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
            stroke="currentColor" stroke-width="2.2" stroke-linecap="round"
            stroke-linejoin="round" class="w-5 h-5 mt-[2px] shrink-0"
          >
            <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
            <line x1="12" y1="9" x2="12" y2="13" />
            <line x1="12" y1="17" x2="12.01" y2="17" />
          </svg>
          <p class="text-[clamp(11px,1.5vw,13px)] leading-snug">
            {{ rujukanInfoBar.text }}
          </p>
        </div>
        <button
          type="button"
          :disabled="ctaLoading"
          :class="[
            'w-full rounded-kiosk transition-opacity active:opacity-85',
            'bg-blue text-white border border-blue',
            'px-[clamp(14px,2.5vw,20px)] py-[clamp(14px,2.5vw,20px)]',
            'text-[clamp(14px,2vw,17px)] font-medium',
            'flex items-center justify-between gap-3',
            'min-h-[clamp(56px,8vw,72px)]',
            'disabled:opacity-60 disabled:cursor-not-allowed',
          ]"
          @click="onIssueSEPRujukan"
        >
          <span class="flex items-center gap-2">
            <svg
              v-if="ctaLoading"
              class="animate-spin w-5 h-5"
              viewBox="0 0 24 24" fill="none"
            >
              <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3"
                      fill="none" stroke-dasharray="40" stroke-dashoffset="20" />
            </svg>
            {{ ctaLoading ? 'Memproses...' : 'Pilih dokter dan lanjutkan' }}
          </span>
          <svg
            v-if="!ctaLoading"
            xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
            stroke="currentColor" stroke-width="2.5" stroke-linecap="round"
            stroke-linejoin="round" class="w-5 h-5 shrink-0"
          >
            <polyline points="9 18 15 12 9 6" />
          </svg>
        </button>
      </template>

      <!-- TidakAktif -->
      <PathwayTidakAktif
        v-else-if="ptype === PatientType.TidakAktif"
        @daftar-umum="onDaftarUmum"
        @hubungi-petugas="onHubungiPetugas"
      />

      <!-- Error / Unknown: hanya info bar, no CTA -->
      <div
        v-else-if="errorInfoBar"
        :class="['rounded-card border p-[clamp(10px,1.8vw,14px)] flex items-start gap-2', infoBarClass(errorInfoBar.variant)]"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
          stroke="currentColor" stroke-width="2.2" stroke-linecap="round"
          stroke-linejoin="round" class="w-5 h-5 mt-[2px] shrink-0"
        >
          <circle cx="12" cy="12" r="10" />
          <line x1="12" y1="8" x2="12" y2="12" />
          <line x1="12" y1="16" x2="12.01" y2="16" />
        </svg>
        <p class="text-[clamp(11px,1.5vw,13px)] leading-snug">
          {{ errorInfoBar.text }}
        </p>
      </div>

      <!-- Footer ghost button — sembunyikan untuk TidakAktif (sudah ada
           "Hubungi petugas" sebagai bagian dari Pathway component-nya) -->
      <button
        v-if="showFooterGhost"
        type="button"
        :disabled="ctaLoading"
        class="w-full rounded-kiosk transition-colors
               bg-surface text-text-secondary border border-border
               hover:border-border-strong active:bg-bg
               px-[clamp(12px,2vw,16px)] py-[clamp(10px,1.8vw,14px)]
               text-[clamp(12px,1.6vw,14px)]
               disabled:opacity-50 disabled:cursor-not-allowed"
        @click="ghostAction"
      >
        {{ ghostLabel }}
      </button>
    </section>

    <IdleOverlay :seconds-left="secondsLeft" :visible="isCountingDown" />

    <!-- Error modal -->
    <AlertModal
      :visible="errorVisible"
      variant="error"
      title="Tidak dapat melanjutkan"
      :message="errorMsg"
      primary-label="Coba lagi"
      close-label="Tutup"
      @primary="errorVisible = false"
      @close="errorVisible = false"
    />

    <!-- Biometrik choice modal — muncul saat SEP butuh validasi biometrik
         (umur >=17, non-IGD). Pasien pilih Sidik Wajah (Frista) atau
         Sidik Jari (After.exe) — mirror UX vendor Khanza WindowBiometrik. -->
    <BiometrikChoiceModal
      :visible="showBiometrikModal"
      :no-peserta="peserta?.NoKartu ?? ''"
      title="Verifikasi Biometrik Diperlukan"
      subtitle="Sebelum SEP dibuat, mohon verifikasi identitas dengan salah satu metode di bawah."
      @select="onPickBiometrik"
      @cancel="onCancelBiometrik"
    />
  </main>
</template>
