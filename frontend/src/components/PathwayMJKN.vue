<!--
  PathwayMJKN — Smart Detector pathway untuk PatientType.MJKN.

  Booking sudah dibuat oleh pasien lewat Mobile JKN. Tugas kiosk =
  konfirmasi kedatangan + cetak tiket. Tidak perlu pilih dokter karena
  dokter & jam praktik sudah fixed di payload booking.

  Props:
    booking — BookingMJKN | null. Backend (Phase B) akan inject struct
              dengan field NoBooking, NoKartu, KdPoli, NmPoli, KdDokter,
              NmDokter, Tanggal, JamPraktik, EstimasiDilayani, NoAntrian.
    loading — boolean, sinkron dgn parent ctaLoading saat call backend.

  Events:
    confirm — user tap "Konfirmasi kedatangan". Parent yg navigate ke
              /tiket setelah backend sync (BuatCheckinMJKN belum ada,
              parent akan fall-through ke router.push tiket sementara).
-->
<script setup>
import { computed } from 'vue'

const props = defineProps({
  booking: { type: Object, default: null },
  loading: { type: Boolean, default: false },
})
defineEmits(['confirm'])

const hasData = computed(() => !!props.booking)

const rows = computed(() => {
  const b = props.booking
  if (!b) return []
  return [
    { key: 'Poli', value: b.NmPoli || b.KdPoli || '—', accent: true },
    { key: 'Dokter', value: b.NmDokter || b.KdDokter || '—' },
    { key: 'Tanggal', value: b.Tanggal || '—' },
    { key: 'Jam praktik', value: b.JamPraktik || '—' },
    { key: 'Estimasi dilayani', value: b.EstimasiDilayani || '—', accent: true },
    { key: 'No antrian', value: b.NoAntrian || b.NoBooking || '—' },
  ]
})
</script>

<template>
  <section class="flex flex-col gap-[clamp(10px,1.8vw,14px)]">
    <!-- Info bar success — booking ditemukan -->
    <div
      class="rounded-card border bg-success-bg text-success border-success-border
             p-[clamp(10px,1.8vw,14px)] flex items-start gap-2"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
        stroke="currentColor" stroke-width="2.2" stroke-linecap="round"
        stroke-linejoin="round" class="w-5 h-5 mt-[2px] shrink-0"
      >
        <polyline points="20 6 9 17 4 12" />
      </svg>
      <p class="text-[clamp(11px,1.5vw,13px)] leading-snug">
        Booking dari Mobile JKN terkonfirmasi. Cetak tiket untuk konfirmasi
        kedatangan ke poli.
      </p>
    </div>

    <!-- Detail card -->
    <div
      v-if="hasData"
      class="bg-surface border border-border rounded-card
             p-[clamp(14px,2.5vw,20px)] flex flex-col gap-[clamp(6px,1vw,10px)]"
    >
      <p class="text-[clamp(11px,1.5vw,13px)] text-text-secondary
                font-medium uppercase tracking-wide">
        Detail booking
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
        Detail booking belum tersedia
      </p>
      <p class="text-[clamp(11px,1.5vw,13px)] text-text-muted mt-2">
        Silakan tunggu sebentar atau hubungi petugas.
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
      @click="$emit('confirm')"
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
        {{ loading ? 'Memproses...' : 'Konfirmasi kedatangan dan cetak tiket' }}
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
