<!--
  StatCard — angka besar + label muted untuk admin stat grid.
  Variant 'warn' untuk angka pending (kuning) saat > 0.
-->
<script setup>
defineProps({
  label: { type: String, required: true },
  value: { type: [String, Number], required: true },
  sub: { type: String, default: '' },
  variant: {
    type: String, default: 'default',
    validator: (v) => ['default', 'warn'].includes(v),
  },
})

const valueClass = {
  default: 'text-text-primary',
  warn: 'text-amber-700',
}
</script>

<template>
  <div class="bg-surface border border-border rounded-card p-[clamp(14px,2vw,18px)]">
    <div class="text-[clamp(10px,1.3vw,12px)] uppercase tracking-wide text-text-muted">
      {{ label }}
    </div>
    <div :class="['text-[clamp(28px,4.5vw,36px)] font-medium leading-tight mt-2', valueClass[variant]]">
      {{ value }}
    </div>
    <div v-if="sub" class="text-[clamp(10px,1.3vw,12px)] text-text-muted mt-1">
      {{ sub }}
    </div>
  </div>
</template>
