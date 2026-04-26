<!--
  AlertModal — dialog error/info reusable.

  Variant:
    error   → header merah, icon warning, ghost btn "Tutup"
    warning → header kuning
    info    → header biru
    success → header hijau

  Pakai:
    <AlertModal :visible="showErr" variant="error" title="Gagal"
                :message="err" primary-label="Coba lagi"
                @primary="retry" @close="showErr=false" />
-->
<script setup>
defineProps({
  visible: { type: Boolean, default: false },
  variant: {
    type: String, default: 'error',
    validator: (v) => ['error', 'warning', 'info', 'success'].includes(v),
  },
  title: { type: String, required: true },
  message: { type: String, default: '' },
  primaryLabel: { type: String, default: 'Coba lagi' },
  closeLabel: { type: String, default: 'Tutup' },
})
defineEmits(['primary', 'close'])

const accentBg = {
  error: 'bg-danger-bg', warning: 'bg-warning-bg',
  info: 'bg-blue-light', success: 'bg-success-bg',
}
const accentText = {
  error: 'text-danger', warning: 'text-warning',
  info: 'text-blue-dark', success: 'text-success',
}
</script>

<template>
  <Transition
    enter-active-class="transition-opacity duration-200"
    enter-from-class="opacity-0" enter-to-class="opacity-100"
    leave-active-class="transition-opacity duration-150"
    leave-from-class="opacity-100" leave-to-class="opacity-0"
  >
    <div
      v-if="visible"
      class="fixed inset-0 z-40 flex items-center justify-center p-4
             bg-black/50 backdrop-blur-sm"
      role="alertdialog"
      aria-modal="true"
    >
      <div
        class="bg-surface rounded-card max-w-[440px] w-full overflow-hidden
               shadow-xl"
        @click.stop
      >
        <!-- Icon block -->
        <div :class="['flex items-center justify-center py-[clamp(20px,3vw,32px)]', accentBg[variant]]">
          <div :class="['rounded-full p-[clamp(10px,1.5vw,14px)]', accentText[variant]]">
            <svg
              v-if="variant === 'error' || variant === 'warning'"
              xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
              stroke="currentColor" stroke-width="2" stroke-linecap="round"
              stroke-linejoin="round" class="w-[clamp(32px,5vw,42px)] h-[clamp(32px,5vw,42px)]"
            >
              <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
              <line x1="12" y1="9" x2="12" y2="13" />
              <line x1="12" y1="17" x2="12.01" y2="17" />
            </svg>
            <svg
              v-else-if="variant === 'success'"
              xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
              stroke="currentColor" stroke-width="2" stroke-linecap="round"
              stroke-linejoin="round" class="w-[clamp(32px,5vw,42px)] h-[clamp(32px,5vw,42px)]"
            >
              <polyline points="20 6 9 17 4 12" />
            </svg>
            <svg
              v-else
              xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
              stroke="currentColor" stroke-width="2" stroke-linecap="round"
              stroke-linejoin="round" class="w-[clamp(32px,5vw,42px)] h-[clamp(32px,5vw,42px)]"
            >
              <circle cx="12" cy="12" r="10" />
              <line x1="12" y1="8" x2="12" y2="12" />
              <line x1="12" y1="16" x2="12.01" y2="16" />
            </svg>
          </div>
        </div>

        <!-- Body -->
        <div class="p-[clamp(18px,2.5vw,24px)] flex flex-col gap-3 text-center">
          <h2 class="text-[clamp(16px,2.4vw,19px)] font-medium text-text-primary">
            {{ title }}
          </h2>
          <p
            v-if="message"
            class="text-[clamp(11px,1.5vw,13px)] text-text-secondary leading-relaxed"
          >
            {{ message }}
          </p>
        </div>

        <!-- Actions -->
        <div class="flex flex-col gap-2 px-[clamp(18px,2.5vw,24px)] pb-[clamp(18px,2.5vw,24px)]">
          <button
            type="button"
            class="w-full rounded-btn bg-blue text-white font-medium
                   px-4 py-[clamp(10px,1.6vw,12px)] text-[clamp(13px,1.8vw,15px)]
                   active:opacity-85"
            @click="$emit('primary')"
          >
            {{ primaryLabel }}
          </button>
          <button
            type="button"
            class="w-full rounded-btn bg-surface text-text-secondary border border-border
                   px-4 py-[clamp(8px,1.4vw,10px)] text-[clamp(12px,1.6vw,14px)]
                   active:bg-bg"
            @click="$emit('close')"
          >
            {{ closeLabel }}
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>
