<!--
  SecondaryCard — tile button untuk grid 2-kolom kiosk.
  Dipakai HomeScreen (Pasien Umum, Ambil Antrian).

  Spec:
    - bg putih, border 0.5px #E4E6EA, rounded-card
    - icon di kiri dalam container tinted (default biru-light, bisa
      di-override via props variant)
    - hover/active feedback ringan
-->
<script setup>
defineProps({
  title: { type: String, required: true },
  subtitle: { type: String, default: '' },
  variant: {
    type: String,
    default: 'blue',
    validator: (v) => ['blue', 'gray'].includes(v),
  },
})
defineEmits(['click'])

const iconBgMap = {
  blue: 'bg-blue-light text-blue',
  gray: 'bg-gray-100 text-gray-600',
}
</script>

<template>
  <button
    type="button"
    class="w-full text-left rounded-card transition-colors
           bg-surface border border-border
           hover:border-border-strong active:bg-[#FAFBFC]
           flex items-center gap-[clamp(10px,2vw,14px)]
           p-[clamp(12px,2vw,16px)]
           min-h-[clamp(72px,10vw,92px)]"
    @click="$emit('click', $event)"
  >
    <div
      :class="[
        'rounded-[10px] flex items-center justify-center flex-shrink-0',
        'w-[clamp(36px,5vw,46px)] h-[clamp(36px,5vw,46px)]',
        iconBgMap[variant],
      ]"
    >
      <slot name="icon">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          class="w-[clamp(16px,2.4vw,20px)] h-[clamp(16px,2.4vw,20px)]"
        >
          <circle cx="12" cy="8" r="4" />
          <path d="M4 21v-2a4 4 0 0 1 4-4h8a4 4 0 0 1 4 4v2" />
        </svg>
      </slot>
    </div>
    <div class="flex-1 min-w-0">
      <div class="text-[clamp(11px,1.6vw,13px)] font-medium text-text-primary leading-tight">
        {{ title }}
      </div>
      <div
        v-if="subtitle"
        class="text-[clamp(10px,1.3vw,12px)] mt-1 text-text-muted"
      >
        {{ subtitle }}
      </div>
    </div>
  </button>
</template>
