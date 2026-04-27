<!--
  StepperBar — visual progress segmented bar untuk multi-step flow.

  Sebelumnya cuma teks kecil "1/3 — Pilih Poli" di pojok header (lansia
  tidak akan baca). Sekarang full-width segmented:
    [✓ Selesai] — [◉ Sedang] — [○ Belum]

  Props:
    steps: array { label, icon (Phosphor component) }
    currentIndex: 0-based
    onStepClick: kalau di-handle, step yang sudah lewat bisa di-tap untuk back
-->
<script setup>
import { computed } from 'vue'
import { PhCheckCircle, PhCircle, PhCircleNotch } from '@phosphor-icons/vue'

const props = defineProps({
  steps: { type: Array, required: true },           // [{ label: 'Pilih Poli', icon: PhBuildings }]
  currentIndex: { type: Number, default: 0 },
  clickable: { type: Boolean, default: false },     // tap step lewat untuk back
})
const emit = defineEmits(['step-click'])

function stateOf(i) {
  if (i < props.currentIndex) return 'done'
  if (i === props.currentIndex) return 'active'
  return 'future'
}
function clickStep(i) {
  if (!props.clickable) return
  if (i >= props.currentIndex) return
  emit('step-click', i)
}

const totalSteps = computed(() => props.steps.length)
</script>

<template>
  <div
    class="flex items-stretch w-full bg-bg rounded-card overflow-hidden
           border border-border"
    role="navigation"
    aria-label="Langkah pendaftaran"
  >
    <template v-for="(step, i) in steps" :key="i">
      <button
        type="button"
        :disabled="!clickable || i >= currentIndex"
        :aria-current="i === currentIndex ? 'step' : undefined"
        :class="[
          'flex-1 px-[clamp(8px,1.5vw,12px)] py-[clamp(10px,1.6vw,14px)]',
          'flex items-center justify-center gap-[clamp(6px,1vw,10px)]',
          'transition-colors min-h-[clamp(52px,7vw,64px)]',
          'text-[clamp(13px,1.7vw,16px)] font-medium',
          stateOf(i) === 'done' &&
            'bg-emerald-50 text-emerald-700 hover:bg-emerald-100 cursor-pointer',
          stateOf(i) === 'active' && 'text-white',
          stateOf(i) === 'future' && 'bg-surface text-text-muted',
        ]"
        :style="stateOf(i) === 'active'
          ? { backgroundColor: 'var(--color-primary, #1B4FD8)' }
          : {}"
        @click="clickStep(i)"
      >
        <component
          v-if="step.icon"
          :is="step.icon"
          :size="20"
          :weight="stateOf(i) === 'active' ? 'fill' : 'bold'"
        />
        <PhCheckCircle v-else-if="stateOf(i) === 'done'" :size="20" weight="fill" />
        <PhCircleNotch v-else-if="stateOf(i) === 'active'" :size="20" weight="bold" class="animate-spin" />
        <PhCircle v-else :size="20" weight="bold" />

        <span class="hidden sm:inline">
          {{ i + 1 }}. {{ step.label }}
        </span>
        <span class="sm:hidden">{{ i + 1 }}</span>
      </button>

      <div
        v-if="i < totalSteps - 1"
        class="w-px bg-border self-stretch flex-shrink-0"
        aria-hidden="true"
      ></div>
    </template>
  </div>
</template>
