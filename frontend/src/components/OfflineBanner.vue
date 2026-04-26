<!--
  OfflineBanner — banner global yang muncul saat sistem offline
  (Khanza tidak reachable). Mounted di App.vue sebagai overlay
  fixed atas RouterView.

  Spec P-050:
    bg #FFFBEB (warning), dot kuning pulse,
    teks "Mode offline — antrian disimpan sementara"
-->
<script setup>
defineProps({
  visible: { type: Boolean, default: false },
})
</script>

<template>
  <Transition
    enter-active-class="transition-transform duration-200"
    enter-from-class="-translate-y-full"
    enter-to-class="translate-y-0"
    leave-active-class="transition-transform duration-150"
    leave-from-class="translate-y-0"
    leave-to-class="-translate-y-full"
  >
    <div
      v-if="visible"
      class="fixed top-0 left-0 right-0 z-30
             bg-warning-bg border-b border-warning-border
             flex items-center justify-center gap-2
             px-4 py-[clamp(6px,1vw,8px)]
             text-warning text-[clamp(10px,1.4vw,12px)] font-medium"
      role="status"
      aria-live="polite"
    >
      <span
        class="w-[clamp(6px,1vw,8px)] h-[clamp(6px,1vw,8px)] rounded-full
               bg-amber-500 animate-pulse flex-shrink-0"
      />
      <span>Mode offline — antrian disimpan sementara</span>
    </div>
  </Transition>
</template>
