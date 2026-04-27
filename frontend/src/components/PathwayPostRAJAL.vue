<!--
  PathwayPostRAJAL — Smart Detector pathway untuk PatientType.PostRAJAL.

  Lanjutan rawat jalan: pasien punya kunjungan rawat jalan terbaru yang
  butuh follow-up (mis. SKDP belum habis). Backend (Phase B) akan inject
  Kunjungan terakhir ke Data; field NoSKDP populated kalau SKDP aktif.

  Props:
    kunjungan — Kunjungan | null (existing struct).
    loading — boolean.

  Events:
    lanjut — user tap "Lanjut pilih dokter".
-->
<script setup>
import { computed } from 'vue'

const props = defineProps({
  kunjungan: { type: Object, default: null },
  loading: { type: Boolean, default: false },
})
defineEmits(['lanjut'])

const hasData = computed(() => !!props.kunjungan)
const hasSKDP = computed(() => !!props.kunjungan?.NoSKDP)

const rows = computed(() => {
  const k = props.kunjungan
  if (!k) return []
  const out = [
    { key: 'No rawat', value: k.NoRawat || '—' },
    { key: 'Tgl kunjungan', value: k.TglKunjungan || k.TglRegistrasi || '—' },
    { key: 'Poli asal', value: k.NmPoli || k.KdPoli || '—', accent: true },
    { key: 'Status', value: k.Status || k.StatusKunjungan || '—' },
  ]
  if (k.NoSKDP) {
    out.push({ key: 'No SKDP', value: k.NoSKDP, accent: true })
  }
  return out
})
</script>

<template>
  <section class="flex flex-col gap-[clamp(10px,1.8vw,14px)]">
    <!-- Info bar — penjelasan kategori -->
    <div
      class="rounded-card border bg-blue-light text-blue-dark border-blue/20
             p-[clamp(10px,1.8vw,14px)] flex items-start gap-2"
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
        Anda memiliki kunjungan lanjutan rawat jalan
        <span v-if="hasSKDP" class="font-medium">dengan SKDP aktif</span>.
        Silakan pilih poli tujuan untuk lanjut.
      </p>
    </div>

    <!-- Detail kunjungan asal -->
    <div
      v-if="hasData"
      class="bg-surface border border-border rounded-card
             p-[clamp(14px,2.5vw,20px)] flex flex-col gap-[clamp(6px,1vw,10px)]"
    >
      <p class="text-[clamp(11px,1.5vw,13px)] text-text-secondary
                font-medium uppercase tracking-wide">
        Kunjungan asal
      </p>
      <div class="flex flex-col gap-[clamp(3px,0.6vw,5px)]">
        <div
          v-for="row in rows"
          :key="row.key"
          class="flex items-start justify-between gap-3 py-[clamp(2px,0.4vw,4px)]"
        >
          <span class="text-[clamp(10px,1.3vw,12px)] text-text-muted shrink-0">
            {{ row.key }}
          </span>
          <span
            :class="[
              'text-[clamp(11px,1.6vw,14px)] font-medium text-right',
              row.accent ? 'text-blue' : 'text-text-primary',
            ]"
          >
            {{ row.value }}
          </span>
        </div>
      </div>
    </div>
    <div
      v-else
      class="bg-surface border border-border rounded-card p-6 text-center"
    >
      <p class="text-[clamp(13px,1.8vw,16px)] font-medium text-text-primary">
        Detail kunjungan belum tersedia
      </p>
      <p class="text-[clamp(11px,1.5vw,13px)] text-text-muted mt-2">
        Anda tetap dapat melanjutkan pemilihan poli di layar berikutnya.
      </p>
    </div>

    <!-- CTA primary -->
    <button
      type="button"
      :disabled="loading"
      :class="[
        'w-full rounded-kiosk transition-opacity active:opacity-85',
        'bg-blue text-white border border-blue',
        'px-[clamp(14px,2.5vw,20px)] py-[clamp(14px,2.5vw,20px)]',
        'text-[clamp(14px,2vw,17px)] font-medium',
        'flex items-center justify-between gap-3',
        'min-h-[clamp(56px,8vw,72px)]',
        'disabled:opacity-60 disabled:cursor-not-allowed',
      ]"
      @click="$emit('lanjut')"
    >
      <span class="flex items-center gap-2">
        <svg
          v-if="loading"
          class="animate-spin w-5 h-5"
          viewBox="0 0 24 24" fill="none"
        >
          <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3"
                  fill="none" stroke-dasharray="40" stroke-dashoffset="20" />
        </svg>
        {{ loading ? 'Memproses...' : 'Lanjut pilih dokter' }}
      </span>
      <svg
        v-if="!loading"
        xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
        stroke="currentColor" stroke-width="2.5" stroke-linecap="round"
        stroke-linejoin="round" class="w-5 h-5 shrink-0"
      >
        <polyline points="9 18 15 12 9 6" />
      </svg>
    </button>
  </section>
</template>
