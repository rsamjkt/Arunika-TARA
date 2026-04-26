<!--
  DetectionStep — satu baris step di DetectScreen step list.

  State:
    wait   → dot abu, label muted
    active → dot biru pulse, label primary
    done   → dot hijau, label hijau, ✓ icon di kanan
    error  → dot merah, label merah, × icon di kanan
-->
<script setup>
defineProps({
  label: { type: String, required: true },
  state: {
    type: String,
    default: 'wait',
    validator: (v) => ['wait', 'active', 'done', 'error'].includes(v),
  },
})

const dotClass = {
  wait: 'bg-text-muted',
  active: 'bg-blue animate-pulse',
  done: 'bg-emerald-500',
  error: 'bg-rose-500',
}
const labelClass = {
  wait: 'text-text-muted',
  active: 'text-text-primary font-medium',
  done: 'text-emerald-700 font-medium',
  error: 'text-rose-700 font-medium',
}
</script>

<template>
  <div
    class="flex items-center gap-[clamp(10px,1.5vw,14px)]
           px-[clamp(10px,1.5vw,14px)] py-[clamp(8px,1.2vw,11px)]
           rounded-card bg-surface border border-border"
  >
    <span
      :class="[
        'w-[clamp(8px,1.2vw,11px)] h-[clamp(8px,1.2vw,11px)] rounded-full flex-shrink-0',
        dotClass[state],
      ]"
    />
    <span
      :class="[
        'flex-1 text-[clamp(11px,1.5vw,13px)] leading-snug',
        labelClass[state],
      ]"
    >
      {{ label }}
    </span>

    <!-- Status icon di kanan -->
    <svg
      v-if="state === 'done'"
      xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
      stroke="currentColor" stroke-width="3" stroke-linecap="round"
      stroke-linejoin="round"
      class="w-[clamp(14px,2vw,18px)] h-[clamp(14px,2vw,18px)] text-emerald-600"
    >
      <polyline points="20 6 9 17 4 12" />
    </svg>
    <svg
      v-else-if="state === 'error'"
      xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
      stroke="currentColor" stroke-width="3" stroke-linecap="round"
      stroke-linejoin="round"
      class="w-[clamp(14px,2vw,18px)] h-[clamp(14px,2vw,18px)] text-rose-600"
    >
      <line x1="18" y1="6" x2="6" y2="18" />
      <line x1="6" y1="6" x2="18" y2="18" />
    </svg>
  </div>
</template>
