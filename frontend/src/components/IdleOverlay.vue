<!--
  IdleOverlay — fullscreen overlay yang muncul di 10 detik terakhir
  sebelum auto-reset. Sentuh layar mana saja → composable yang panggil
  reset() (dispatch lewat parent component yang tracking idle).

  Visual: gelap-translucent, message besar, countdown angka dengan
  prominent typography.
-->
<script setup>
import { I18N } from '../constants/i18n'

defineProps({
  secondsLeft: { type: Number, required: true },
  visible: { type: Boolean, default: false },
})
</script>

<template>
  <Transition
    enter-active-class="transition-opacity duration-200"
    enter-from-class="opacity-0"
    enter-to-class="opacity-100"
    leave-active-class="transition-opacity duration-200"
    leave-from-class="opacity-100"
    leave-to-class="opacity-0"
  >
    <div
      v-if="visible"
      class="fixed inset-0 z-50 flex flex-col items-center justify-center gap-6
             bg-black/70 backdrop-blur-sm text-white text-center px-6"
      role="alertdialog"
      aria-live="polite"
    >
      <!-- Big countdown number -->
      <div class="text-[clamp(80px,18vw,160px)] font-medium leading-none">
        {{ secondsLeft }}
      </div>

      <!-- Message — context aware: pasien sedang di flow + tidak interaksi -->
      <p class="text-[clamp(20px,3vw,28px)] font-semibold">
        {{ I18N.idle.title }}
      </p>
      <p class="text-[clamp(14px,2vw,18px)] text-white/85">
        {{ I18N.idle.sub }}
      </p>
      <p class="text-[clamp(12px,1.6vw,15px)] text-white/70 mt-2">
        {{ I18N.idle.tap }}
      </p>
    </div>
  </Transition>
</template>
