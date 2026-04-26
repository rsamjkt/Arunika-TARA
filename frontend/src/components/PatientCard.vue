<!--
  PatientCard — kartu info pasien dengan pill status di kanan
  + tanggal kiri-bawah, nama + no kartu masked, divider, dan
  key-value rows untuk detail spesifik per kategori.

  Spec DESIGN_SYSTEM.md - dipakai di semua 4 ResultScreen variant.
-->
<script setup>
import StatusPill from './StatusPill.vue'

defineProps({
  // Pill kanan-atas
  pillLabel: { type: String, required: true },
  pillVariant: {
    type: String,
    default: 'info',
    validator: (v) => ['success', 'info', 'warning', 'danger'].includes(v),
  },
  // Tanggal kiri-atas
  dateLabel: { type: String, default: '' },

  // Identitas pasien
  nama: { type: String, required: true },
  noKartu: { type: String, default: '' },

  // Key-value rows: [{ key, value, accent? }]
  details: { type: Array, default: () => [] },
})

// Format no kartu: 4-digit grouping (1234 5678 9012 3456)
function formatKartu(no) {
  if (!no) return ''
  return no.match(/.{1,4}/g)?.join(' ') ?? no
}
</script>

<template>
  <div
    class="bg-surface border border-border rounded-card
           p-[clamp(14px,2.5vw,20px)]
           flex flex-col gap-[clamp(8px,1.5vw,12px)]"
  >
    <!-- Header: pill + tanggal -->
    <div class="flex items-center justify-between gap-3">
      <StatusPill :label="pillLabel" :variant="pillVariant" />
      <span
        v-if="dateLabel"
        class="text-[clamp(9px,1.1vw,11px)] text-text-muted"
      >
        {{ dateLabel }}
      </span>
    </div>

    <!-- Nama + nomor kartu -->
    <div>
      <div class="text-[clamp(15px,2.5vw,19px)] font-medium text-text-primary leading-tight">
        {{ nama }}
      </div>
      <div
        v-if="noKartu"
        class="font-mono text-[clamp(10px,1.4vw,12px)] text-text-muted mt-1 tracking-wide"
      >
        {{ formatKartu(noKartu) }}
      </div>
    </div>

    <!-- Divider -->
    <hr class="border-border" />

    <!-- Detail rows -->
    <div class="flex flex-col gap-[clamp(3px,0.6vw,5px)]">
      <div
        v-for="row in details"
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
</template>
