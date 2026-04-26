<!--
  AntrianCard — tile button untuk pilihan antrian.

  Spec P-044:
    bg putih, border 0.5px #E4E6EA, rounded-card 12px
    icon dalam container tinted (biru/hijau), SVG bukan emoji 16-18px
    judul clamp(11px,1.6vw,13px) font-medium
    counter "Sekarang: X-XXX" clamp(10px,1.3vw,12px) muted
    hover: border #CDD1D9, active: bg #FAFBFC
    loading: disabled + spinner overlay
-->
<script setup>
defineProps({
  title: { type: String, required: true },
  // Counter terakhir hari ini (mis. "A-035"); kosong = "Sekarang: —"
  counter: { type: String, default: '' },
  variant: {
    type: String, default: 'blue',
    validator: (v) => ['blue', 'green', 'gray'].includes(v),
  },
  loading: { type: Boolean, default: false },
})
defineEmits(['click'])

const iconBg = {
  blue: 'bg-blue-light text-blue',
  green: 'bg-emerald-50 text-emerald-700',
  gray: 'bg-gray-100 text-gray-600',
}
</script>

<template>
  <button
    type="button"
    :disabled="loading"
    class="relative w-full text-left rounded-card transition-colors
           bg-surface border border-border
           hover:border-border-strong active:bg-[#FAFBFC]
           p-[clamp(12px,2vw,16px)]
           min-h-[clamp(76px,11vw,100px)]
           flex items-center gap-[clamp(10px,1.5vw,14px)]
           disabled:opacity-60 disabled:cursor-not-allowed"
    @click="$emit('click', $event)"
  >
    <div
      :class="[
        'rounded-[10px] flex items-center justify-center flex-shrink-0',
        'w-[clamp(36px,5vw,46px)] h-[clamp(36px,5vw,46px)]',
        iconBg[variant],
      ]"
    >
      <slot name="icon" />
    </div>

    <div class="flex-1 min-w-0">
      <div class="text-[clamp(11px,1.6vw,13px)] font-medium text-text-primary leading-tight">
        {{ title }}
      </div>
      <div class="text-[clamp(10px,1.3vw,12px)] text-text-muted mt-1 leading-tight">
        Sekarang: <span class="font-mono font-medium text-text-secondary">{{ counter || '—' }}</span>
      </div>
    </div>

    <!-- Loading spinner overlay -->
    <Transition
      enter-active-class="transition-opacity duration-150"
      enter-from-class="opacity-0" enter-to-class="opacity-100"
    >
      <div
        v-if="loading"
        class="absolute inset-0 bg-surface/80 rounded-card flex items-center justify-center"
      >
        <svg
          class="animate-spin w-6 h-6 text-blue"
          viewBox="0 0 24 24" fill="none"
        >
          <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3"
                  fill="none" stroke-dasharray="40" stroke-dashoffset="20" />
        </svg>
      </div>
    </Transition>
  </button>
</template>
