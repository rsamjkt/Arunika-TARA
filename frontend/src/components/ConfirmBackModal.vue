<!--
  ConfirmBackModal — modal sopan untuk konfirmasi action "back/reset state".
  Reusable di mana saja yang butuh "yakin batalkan?".

  Bahasa: tone netral & mengayomi (calm UX), bukan urgent/scary.
-->
<script setup>
import { PhWarningCircle } from '@phosphor-icons/vue'
import { useAudioCue } from '../composables/useAudioCue'

defineProps({
  visible: { type: Boolean, default: false },
  title: { type: String, required: true },
  message: { type: String, default: '' },
  confirmLabel: { type: String, default: 'Ya, kembali' },
  cancelLabel: { type: String, default: 'Tidak, lanjut isi' },
})
const emit = defineEmits(['confirm', 'cancel'])
const audio = useAudioCue()

function onConfirm() {
  audio.tap()
  emit('confirm')
}
function onCancel() {
  audio.tap()
  emit('cancel')
}
</script>

<template>
  <Transition name="fade">
    <div
      v-if="visible"
      class="fixed inset-0 z-[100] bg-black/40 backdrop-blur-sm
             flex items-center justify-center p-[clamp(16px,3vw,24px)]"
      role="dialog"
      aria-modal="true"
    >
      <div
        class="bg-surface rounded-card shadow-xl max-w-[480px] w-full
               p-[clamp(20px,3vw,28px)] flex flex-col gap-[clamp(12px,1.8vw,16px)]"
      >
        <div class="flex items-center gap-[clamp(10px,1.5vw,14px)]">
          <div class="bg-amber-50 text-amber-700 rounded-full p-[clamp(10px,1.5vw,14px)] flex-shrink-0">
            <PhWarningCircle :size="32" weight="duotone" />
          </div>
          <h2 class="text-[clamp(16px,2.2vw,20px)] font-medium text-text-primary leading-tight">
            {{ title }}
          </h2>
        </div>
        <p class="text-[clamp(14px,1.7vw,16px)] text-text-secondary leading-relaxed">
          {{ message }}
        </p>

        <div class="flex flex-col sm:flex-row gap-[clamp(8px,1.2vw,10px)] mt-[clamp(8px,1.2vw,12px)]">
          <button
            type="button"
            class="flex-1 bg-surface border border-border text-text-primary
                   font-medium rounded-btn
                   px-[clamp(16px,2.5vw,20px)] py-[clamp(12px,1.8vw,16px)]
                   text-[clamp(14px,1.8vw,16px)]
                   min-h-[clamp(56px,7vw,64px)]
                   hover:border-border-strong"
            @click="onCancel"
          >
            {{ cancelLabel }}
          </button>
          <button
            type="button"
            class="flex-1 text-white font-medium rounded-btn
                   px-[clamp(16px,2.5vw,20px)] py-[clamp(12px,1.8vw,16px)]
                   text-[clamp(14px,1.8vw,16px)]
                   min-h-[clamp(56px,7vw,64px)]
                   hover:opacity-90 active:opacity-80"
            style="background-color: var(--color-primary, #1B4FD8)"
            @click="onConfirm"
          >
            {{ confirmLabel }}
          </button>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 200ms ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
