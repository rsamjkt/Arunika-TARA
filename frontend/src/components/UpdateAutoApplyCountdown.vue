<!--
  UpdateAutoApplyCountdown — modal full-screen 30 detik countdown saat
  cfg.update.auto_apply=true. User bisa Cancel supaya update tidak
  diterapkan otomatis (admin tetap bisa apply manual nanti).

  Props:
    visible : boolean
    seconds : initial countdown (default 30)
    version : versi target update (display)

  Emits:
    cancel  : user cancel
    timeout : countdown habis
-->
<script setup>
import { ref, watch, onUnmounted } from 'vue'
import { PhCloudArrowDown, PhX } from '@phosphor-icons/vue'

const props = defineProps({
  visible: { type: Boolean, default: false },
  seconds: { type: Number, default: 30 },
  version: { type: String, default: '' },
})

const emit = defineEmits(['cancel', 'timeout'])

const remaining = ref(props.seconds)
let timerId = null

function startCountdown() {
  remaining.value = props.seconds
  clearTimer()
  timerId = setInterval(() => {
    remaining.value -= 1
    if (remaining.value <= 0) {
      clearTimer()
      emit('timeout')
    }
  }, 1000)
}

function clearTimer() {
  if (timerId) {
    clearInterval(timerId)
    timerId = null
  }
}

watch(() => props.visible, (v) => {
  if (v) startCountdown()
  else clearTimer()
}, { immediate: true })

onUnmounted(clearTimer)

function onCancel() {
  clearTimer()
  emit('cancel')
}
</script>

<template>
  <Teleport to="body">
    <div
      v-if="visible"
      class="fixed inset-0 z-[100] flex items-center justify-center
             p-[clamp(20px,4vw,40px)]
             bg-black/85 backdrop-blur-md"
      role="dialog"
      aria-modal="true"
    >
      <div
        class="bg-surface rounded-card shadow-2xl
               max-w-[600px] w-full
               px-[clamp(28px,4vw,40px)] py-[clamp(32px,5vw,48px)]
               flex flex-col items-center gap-[clamp(18px,2.5vw,24px)] text-center"
      >
        <div
          class="rounded-full flex items-center justify-center bg-blue-light text-blue
                 w-[clamp(80px,11vw,108px)] h-[clamp(80px,11vw,108px)]"
        >
          <PhCloudArrowDown :size="64" weight="fill" />
        </div>

        <h2 class="text-[clamp(22px,3vw,28px)] font-bold text-text-primary leading-tight">
          Update tersedia: {{ version }}
        </h2>
        <p class="text-[clamp(13px,1.7vw,15px)] text-text-secondary leading-relaxed">
          Kiosk akan restart untuk menerapkan update dalam
          <span class="font-bold text-blue">{{ remaining }} detik</span>.
          Tekan <strong>Tunda</strong> untuk update nanti via panel admin.
        </p>

        <!-- Countdown ring -->
        <div class="relative w-[clamp(96px,13vw,128px)] h-[clamp(96px,13vw,128px)]">
          <svg viewBox="0 0 100 100" class="w-full h-full -rotate-90">
            <circle cx="50" cy="50" r="46" fill="none" stroke="var(--color-border, #e5e7eb)" stroke-width="6" />
            <circle
              cx="50" cy="50" r="46" fill="none"
              stroke="var(--color-primary, #1B4FD8)" stroke-width="6"
              stroke-linecap="round"
              :stroke-dasharray="2 * Math.PI * 46"
              :stroke-dashoffset="(2 * Math.PI * 46) * (1 - remaining / seconds)"
              class="transition-[stroke-dashoffset] duration-1000 ease-linear"
            />
          </svg>
          <div
            class="absolute inset-0 flex items-center justify-center
                   text-[clamp(28px,4vw,36px)] font-bold text-text-primary"
          >
            {{ remaining }}
          </div>
        </div>

        <button
          type="button"
          class="rounded-btn bg-surface text-text-primary border-2 border-border
                 hover:border-border-strong active:bg-bg
                 px-[clamp(20px,3vw,28px)] py-[clamp(12px,1.8vw,16px)]
                 text-[clamp(14px,1.9vw,16px)] font-semibold
                 min-h-[clamp(52px,7vw,64px)]
                 flex items-center gap-2"
          @click="onCancel"
        >
          <PhX :size="20" weight="bold" />
          Tunda update
        </button>
      </div>
    </div>
  </Teleport>
</template>
