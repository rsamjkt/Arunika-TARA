<!--
  StatusDot — pill kecil dengan dot pulse + label.
  Dipakai di header kiosk untuk status BPJS / Sistem / dll.

  Variant:
    online  → hijau (animate pulse)
    offline → merah
    warning → kuning
    idle    → abu (default kalau status belum diketahui)
-->
<script setup>
defineProps({
  label: { type: String, required: true },
  variant: {
    type: String,
    default: 'idle',
    validator: (v) => ['online', 'offline', 'warning', 'idle'].includes(v),
  },
})

const dotClassMap = {
  online: 'bg-emerald-500 animate-pulse',
  offline: 'bg-rose-500',
  warning: 'bg-amber-500',
  idle: 'bg-gray-400',
}
const pillClassMap = {
  online: 'bg-emerald-50 text-emerald-800',
  offline: 'bg-rose-50 text-rose-800',
  warning: 'bg-amber-50 text-amber-800',
  idle: 'bg-gray-100 text-gray-600',
}
</script>

<template>
  <span
    :class="[
      'inline-flex items-center gap-[6px] rounded-tag font-medium',
      'px-[clamp(8px,1.2vw,12px)] py-[clamp(3px,0.5vw,5px)]',
      'text-[clamp(9px,1.1vw,11px)]',
      pillClassMap[variant],
    ]"
  >
    <span
      :class="[
        'w-[clamp(5px,0.7vw,7px)] h-[clamp(5px,0.7vw,7px)] rounded-full',
        dotClassMap[variant],
      ]"
    />
    {{ label }}
  </span>
</template>
