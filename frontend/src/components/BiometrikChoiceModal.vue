<!--
  BiometrikChoiceModal — modal pilihan metode verifikasi biometrik
  saat SEP butuh validasi (umur >= 17, non-IGD, RAJAL).

  Mirror pattern vendor Khanza: DlgRegistrasiSEPPertama.java::WindowBiometrik
  yang punya 2 tombol (btnFingerPrint1=Frista wajah, btnFingerPrint2=After.exe).

  Props:
    visible     : boolean   — show / hide
    noPeserta   : string    — No. Kartu BPJS pasien (akan di-mask ***1234)
    title       : string    — override judul (default "Verifikasi Identitas")
    subtitle    : string    — override subtitle
    failCount   : number    — counter berapa kali verifikasi gagal di session
                              ini. Kalau >= 2, tampilkan escape hatch
                              "Pengajuan SEP via BPJS" (vendor pattern
                              btnDiagnosaAwal4 di DlgRegistrasiSEPPertama).

  Emits:
    select       : ('face' | 'fingerprint')
    pengajuan    : ()  — pasien pilih escape hatch (FE harus konfirmasi
                         + panggil apmService.pengajuanSEPFP)
    cancel       : ()

  Spec UX:
    - Centered overlay full-screen, backdrop semi-transparent + blur
    - 2 tombol big setara visual weight (sidik wajah | sidik jari)
    - Touch target min-h clamp(72px, 8vw, 100px)
    - Close X kanan atas + tombol "Batal" di footer (redundan untuk lansia)
    - Click backdrop = cancel
    - PHI safety: noPeserta di-mask jadi *** + last 4 digits saja
-->
<script setup>
import { computed } from 'vue'
import { PhCamera, PhFingerprint, PhX, PhWarningCircle } from '@phosphor-icons/vue'

const props = defineProps({
  visible: { type: Boolean, default: false },
  noPeserta: { type: String, default: '' },
  title: { type: String, default: 'Verifikasi Identitas' },
  subtitle: { type: String, default: 'Pilih metode verifikasi biometrik untuk lanjut membuat SEP' },
  failCount: { type: Number, default: 0 },
})

const emit = defineEmits(['select', 'pengajuan', 'cancel'])

// Mask noPeserta: tampilkan ***1234 (last 4 digits) untuk PHI safety.
const maskedNoPeserta = computed(() => {
  const s = props.noPeserta || ''
  if (s.length < 4) return s ? '***' : ''
  return '***' + s.slice(-4)
})

// Escape hatch hanya tampil setelah pasien gagal 2x — supaya tidak
// jadi shortcut malas, tapi cukup ramah saat hardware/lansia bermasalah.
const showEscapeHatch = computed(() => props.failCount >= 2)

function pick(method) {
  emit('select', method)
}
function pengajuan() {
  emit('pengajuan')
}
function cancel() {
  emit('cancel')
}
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition-opacity duration-200"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition-opacity duration-150"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="visible"
        class="fixed inset-0 z-50 flex items-center justify-center
               p-[clamp(16px,3vw,32px)]
               bg-black/55 backdrop-blur-sm"
        role="dialog"
        aria-modal="true"
        :aria-label="title"
        @click.self="cancel"
      >
        <div
          class="bg-surface rounded-card shadow-xl
                 max-w-[640px] w-full overflow-hidden
                 border border-border"
          @click.stop
        >
          <!-- Header dengan tombol close X -->
          <div
            class="flex items-start justify-between gap-3
                   px-[clamp(20px,3vw,32px)] pt-[clamp(20px,3vw,28px)] pb-[clamp(8px,1.2vw,12px)]"
          >
            <div class="flex-1 min-w-0">
              <h2
                class="text-[clamp(20px,2.8vw,26px)] font-bold text-text-primary leading-tight"
              >
                {{ title }}
              </h2>
              <p
                class="text-[clamp(13px,1.7vw,15px)] text-text-secondary mt-1.5 leading-relaxed"
              >
                {{ subtitle }}
              </p>
              <p
                v-if="maskedNoPeserta"
                class="text-[clamp(12px,1.5vw,14px)] text-text-muted mt-2"
              >
                Pasien — No. Kartu
                <span class="font-mono font-semibold text-text-secondary tracking-wider">
                  {{ maskedNoPeserta }}
                </span>
              </p>
            </div>
            <button
              type="button"
              class="flex-shrink-0 rounded-full
                     w-[clamp(36px,5vw,44px)] h-[clamp(36px,5vw,44px)]
                     flex items-center justify-center
                     text-text-muted hover:text-text-primary
                     hover:bg-bg active:bg-border transition-colors"
              aria-label="Tutup"
              @click="cancel"
            >
              <PhX :size="22" weight="bold" />
            </button>
          </div>

          <!-- 2 tombol pilihan biometrik — setara visual weight -->
          <div
            class="grid grid-cols-1 sm:grid-cols-2
                   gap-[clamp(12px,2vw,18px)]
                   px-[clamp(20px,3vw,32px)] py-[clamp(12px,2vw,18px)]"
          >
            <!-- Sidik Wajah / Frista (kamera) -->
            <button
              type="button"
              class="rounded-card border-2 border-border
                     bg-surface
                     px-[clamp(16px,2.5vw,22px)] py-[clamp(20px,3vw,28px)]
                     flex flex-col items-center justify-center text-center
                     gap-[clamp(8px,1.4vw,12px)]
                     min-h-[clamp(160px,20vw,200px)]
                     hover:border-blue hover:bg-blue-light
                     active:opacity-90 transition-colors"
              @click="pick('face')"
            >
              <div
                class="rounded-full flex items-center justify-center
                       w-[clamp(64px,9vw,80px)] h-[clamp(64px,9vw,80px)]
                       text-white"
                :style="{ backgroundColor: 'var(--color-primary, #1B4FD8)' }"
              >
                <PhCamera :size="44" weight="fill" />
              </div>
              <div
                class="text-[clamp(17px,2.4vw,21px)] font-bold text-text-primary leading-tight"
              >
                Sidik Wajah
              </div>
              <div
                class="text-[clamp(11px,1.4vw,13px)] text-text-muted leading-snug"
              >
                Frista &middot; pakai kamera
              </div>
            </button>

            <!-- Sidik Jari / After.exe (sensor) -->
            <button
              type="button"
              class="rounded-card border-2 border-border
                     bg-surface
                     px-[clamp(16px,2.5vw,22px)] py-[clamp(20px,3vw,28px)]
                     flex flex-col items-center justify-center text-center
                     gap-[clamp(8px,1.4vw,12px)]
                     min-h-[clamp(160px,20vw,200px)]
                     hover:border-blue hover:bg-blue-light
                     active:opacity-90 transition-colors"
              @click="pick('fingerprint')"
            >
              <div
                class="rounded-full flex items-center justify-center
                       w-[clamp(64px,9vw,80px)] h-[clamp(64px,9vw,80px)]
                       bg-emerald-500 text-white"
              >
                <PhFingerprint :size="44" weight="fill" />
              </div>
              <div
                class="text-[clamp(17px,2.4vw,21px)] font-bold text-text-primary leading-tight"
              >
                Sidik Jari
              </div>
              <div
                class="text-[clamp(11px,1.4vw,13px)] text-text-muted leading-snug"
              >
                After.exe &middot; pakai sensor
              </div>
            </button>
          </div>

          <!-- Escape hatch — pengajuan SEP via BPJS kalau pasien tidak
               bisa verifikasi (lansia, sensor rusak, dll). Hanya tampil
               setelah 2x gagal. Mirror vendor /Sep/pengajuanSEP. -->
          <div
            v-if="showEscapeHatch"
            class="mx-[clamp(20px,3vw,32px)] mb-[clamp(12px,2vw,16px)]
                   rounded-card border-2 border-amber-300
                   bg-amber-50
                   px-[clamp(16px,2.5vw,22px)] py-[clamp(14px,2vw,18px)]"
          >
            <div class="flex items-start gap-[clamp(10px,1.5vw,14px)]">
              <div class="flex-shrink-0 mt-0.5 text-amber-600">
                <PhWarningCircle :size="24" weight="fill" />
              </div>
              <div class="flex-1 min-w-0">
                <div
                  class="text-[clamp(14px,1.8vw,16px)] font-bold text-amber-900 leading-tight"
                >
                  Tidak bisa verifikasi biometrik?
                </div>
                <div
                  class="text-[clamp(12px,1.5vw,14px)] text-amber-800 mt-1 leading-relaxed"
                >
                  Pengajuan SEP ke BPJS bisa dilakukan tanpa biometrik
                  untuk kasus khusus (lansia, alat rusak, dll).
                </div>
                <button
                  type="button"
                  class="mt-[clamp(10px,1.5vw,14px)] w-full sm:w-auto
                         rounded-btn bg-amber-600 hover:bg-amber-700
                         active:bg-amber-800 text-white font-semibold
                         px-[clamp(16px,2.2vw,22px)] py-[clamp(10px,1.4vw,12px)]
                         text-[clamp(13px,1.7vw,15px)]
                         min-h-[clamp(44px,5.5vw,52px)]
                         transition-colors"
                  @click="pengajuan"
                >
                  Pengajuan SEP via BPJS
                </button>
              </div>
            </div>
          </div>

          <!-- Footer dengan tombol Batal redundan (lansia friendly) -->
          <div
            class="px-[clamp(20px,3vw,32px)] pb-[clamp(20px,3vw,28px)] pt-[clamp(4px,0.8vw,8px)]"
          >
            <button
              type="button"
              class="w-full rounded-btn
                     bg-surface text-text-secondary border border-border
                     hover:border-border-strong active:bg-bg
                     px-[clamp(14px,2vw,18px)] py-[clamp(10px,1.5vw,14px)]
                     text-[clamp(13px,1.7vw,15px)] font-medium
                     min-h-[clamp(48px,6vw,56px)]"
              @click="cancel"
            >
              Batal
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
