<!--
  StatusPill — pill berwarna lebih kontras dari StatusDot, dipakai
  sebagai header label di PatientCard.

  Variant warna sesuai design tokens:
    success → bg #ECFDF5, text #065F46, dot hijau
    info    → bg #EEF2FF, text #1E40AF, dot biru
    warning → bg #FFFBEB, text #92400E, dot kuning
    danger  → bg #FEF2F2, text #991B1B, dot merah
-->
<script setup>
defineProps({
  label: { type: String, required: true },
  variant: {
    type: String,
    default: 'info',
    validator: (v) => ['success', 'info', 'warning', 'danger'].includes(v),
  },
})

const pillClass = {
  success: 'bg-success-bg text-success border border-success-border',
  info: 'bg-blue-light text-blue-dark border border-blue/20',
  warning: 'bg-warning-bg text-warning border border-warning-border',
  danger: 'bg-danger-bg text-danger border border-danger-border',
}
const dotClass = {
  success: 'bg-emerald-500',
  info: 'bg-blue',
  warning: 'bg-amber-500',
  danger: 'bg-rose-500',
}
</script>

<template>
  <span
    :class="[
      'inline-flex items-center gap-[6px] rounded-tag font-medium',
      'px-[clamp(8px,1.4vw,12px)] py-[clamp(3px,0.6vw,5px)]',
      'text-[clamp(10px,1.3vw,12px)]',
      pillClass[variant],
    ]"
  >
    <span
      :class="[
        'w-[clamp(5px,0.8vw,7px)] h-[clamp(5px,0.8vw,7px)] rounded-full',
        dotClass[variant],
      ]"
    />
    {{ label }}
  </span>
</template>
