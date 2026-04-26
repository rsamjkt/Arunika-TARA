<!--
  ActionTile — button kotak dengan icon + judul untuk grid action di
  admin panel. Variant 'danger' untuk Reset/destructive actions.
-->
<script setup>
defineProps({
  title: { type: String, required: true },
  subtitle: { type: String, default: '' },
  variant: {
    type: String, default: 'default',
    validator: (v) => ['default', 'danger', 'success'].includes(v),
  },
  loading: { type: Boolean, default: false },
})
defineEmits(['click'])

const iconBg = {
  default: 'bg-blue-light text-blue',
  danger: 'bg-rose-50 text-rose-700',
  success: 'bg-emerald-50 text-emerald-700',
}
</script>

<template>
  <button
    type="button"
    :disabled="loading"
    class="bg-surface border border-border rounded-card text-left
           p-[clamp(12px,2vw,16px)] flex items-center gap-3
           hover:border-border-strong active:bg-bg
           min-h-[clamp(64px,9vw,84px)]
           disabled:opacity-60 disabled:cursor-not-allowed
           transition-colors"
    @click="$emit('click', $event)"
  >
    <div
      :class="[
        'rounded-[10px] flex items-center justify-center flex-shrink-0',
        'w-[clamp(36px,5vw,44px)] h-[clamp(36px,5vw,44px)]',
        iconBg[variant],
      ]"
    >
      <slot name="icon" />
    </div>
    <div class="flex-1 min-w-0">
      <div class="text-[clamp(12px,1.6vw,14px)] font-medium text-text-primary leading-tight">
        {{ title }}
      </div>
      <div
        v-if="subtitle"
        class="text-[clamp(10px,1.3vw,12px)] text-text-muted mt-1 leading-tight"
      >
        {{ subtitle }}
      </div>
    </div>
    <svg
      v-if="loading"
      class="animate-spin w-4 h-4 text-text-muted"
      viewBox="0 0 24 24" fill="none"
    >
      <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3"
              fill="none" stroke-dasharray="40" stroke-dashoffset="20" />
    </svg>
  </button>
</template>
