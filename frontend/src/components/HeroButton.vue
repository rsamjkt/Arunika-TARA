<!--
  HeroButton — CTA besar primary kiosk (full width).
  Dipakai HomeScreen untuk button BPJS dan Aktivasi Satu Sehat.

  Spec DESIGN_SYSTEM.md:
    - bg #1B4FD8 (primary), text putih
    - Icon di kiri (bg white/20, square rounded), title + subtitle + tag opsional
    - padding clamp(14px, 2.5vw, 20px)
    - touch target min-height clamp(52px, 7vw, 72px)
-->
<script setup>
defineProps({
  title: { type: String, required: true },
  subtitle: { type: String, default: '' },
  tag: { type: String, default: '' },
  loading: { type: Boolean, default: false },
})
defineEmits(['click'])
</script>

<template>
  <button
    type="button"
    class="w-full text-left rounded-kiosk transition-opacity active:opacity-85
           bg-blue text-white border border-blue
           flex items-center gap-[clamp(10px,2vw,16px)]
           p-[clamp(14px,2.5vw,20px)]
           min-h-[clamp(72px,10vw,96px)]
           disabled:opacity-60 disabled:cursor-not-allowed"
    :disabled="loading"
    @click="$emit('click', $event)"
  >
    <!-- Icon container -->
    <div
      class="rounded-[10px] flex items-center justify-center flex-shrink-0
             bg-white/20
             w-[clamp(40px,6vw,52px)] h-[clamp(40px,6vw,52px)]"
    >
      <slot name="icon">
        <!-- Default icon: kartu (KTP/BPJS) -->
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          class="w-[clamp(20px,3vw,26px)] h-[clamp(20px,3vw,26px)]"
        >
          <rect x="2" y="5" width="20" height="14" rx="2" />
          <line x1="2" y1="10" x2="22" y2="10" />
        </svg>
      </slot>
    </div>

    <!-- Text content -->
    <div class="flex-1 min-w-0">
      <div class="text-[clamp(15px,2.5vw,19px)] font-medium leading-tight">
        {{ title }}
      </div>
      <div
        v-if="subtitle"
        class="text-[clamp(11px,1.6vw,14px)] mt-1 text-white/85"
      >
        {{ subtitle }}
      </div>
      <span
        v-if="tag"
        class="inline-block mt-[8px] px-3 py-1 rounded-tag text-[clamp(9px,1.1vw,11px)] font-medium bg-white/20"
      >
        {{ tag }}
      </span>
    </div>

    <!-- Loading spinner / chevron -->
    <div class="flex-shrink-0">
      <svg
        v-if="loading"
        class="animate-spin h-5 w-5 text-white/80"
        viewBox="0 0 24 24"
      >
        <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3" fill="none" stroke-dasharray="40" stroke-dashoffset="20" />
      </svg>
      <svg
        v-else
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="2.5"
        stroke-linecap="round"
        stroke-linejoin="round"
        class="w-[clamp(18px,2.5vw,22px)] h-[clamp(18px,2.5vw,22px)] text-white/70"
      >
        <polyline points="9 18 15 12 9 6" />
      </svg>
    </div>
  </button>
</template>
