<!--
  InputDisplay — display nomor yang sedang diketik dengan format
  4-digit grouping (1234 5678 9012 3456) + placeholder underscores +
  cursor blink animation.

  Spec DESIGN_SYSTEM.md:
    font monospace, clamp(18px, 3.5vw, 26px)
    Placeholder: "_ _ _ _  _ _ _ _  _ _ _ _  _ _ _ _" (abu muted)
-->
<script setup>
import { computed } from 'vue'

const props = defineProps({
  // Raw digit string (no spaces) — parent yang manage state
  value: { type: String, required: true },
  // Max digit (default 16 untuk NIK / NoKartu BPJS)
  maxLength: { type: Number, default: 16 },
})

// Format: insert spasi setiap 4 digit
const formatted = computed(() => {
  if (!props.value) return ''
  return props.value.match(/.{1,4}/g)?.join(' ') ?? ''
})

// Placeholder: "_ _ _ _  _ _ _ _  _ _ _ _  _ _ _ _"
// Untuk maxLength 16 → 4 group of "_ _ _ _" separated by 2 spaces
const placeholder = computed(() => {
  const groups = Math.ceil(props.maxLength / 4)
  const parts = []
  for (let i = 0; i < groups; i++) {
    const sz = i === groups - 1 ? props.maxLength - i * 4 : 4
    parts.push(Array(sz).fill('_').join(' '))
  }
  return parts.join('  ')
})

const isEmpty = computed(() => props.value.length === 0)
</script>

<template>
  <div
    class="bg-surface border border-border rounded-card
           px-[clamp(14px,2.5vw,22px)] py-[clamp(16px,2.8vw,24px)]
           flex items-center justify-center
           min-h-[clamp(56px,8vw,80px)]
           relative overflow-hidden"
    role="textbox"
    aria-label="Nomor yang diketik"
    :aria-valuetext="value"
  >
    <span
      v-if="isEmpty"
      class="font-mono tracking-wider text-text-muted select-none
             text-[clamp(16px,3vw,24px)]"
    >
      {{ placeholder }}
    </span>
    <span
      v-else
      class="font-mono tracking-wider text-text-primary
             text-[clamp(18px,3.5vw,26px)] font-medium"
    >
      {{ formatted }}
      <span class="cursor-blink ml-1 inline-block w-[2px] h-[clamp(20px,3.2vw,26px)] bg-text-primary align-middle" />
    </span>
  </div>
</template>

<style scoped>
.cursor-blink {
  animation: blink 1s step-end infinite;
}
@keyframes blink {
  50% { opacity: 0; }
}
</style>
