<!--
  UpdateProgressModal — full-screen progress saat download/apply update.

  Props:
    visible    : boolean
    phase      : 'download' | 'apply' | 'restart'
    downloaded : number   (bytes)
    total      : number   (bytes)
    error      : string | null

  Tidak emit cancel — proses update tidak boleh di-interrupt.
-->
<script setup>
import { computed } from 'vue'
import { PhCloudArrowDown, PhCheckCircle, PhWarningCircle } from '@phosphor-icons/vue'

const props = defineProps({
  visible: { type: Boolean, default: false },
  phase: { type: String, default: 'download' },
  downloaded: { type: Number, default: 0 },
  total: { type: Number, default: 0 },
  error: { type: String, default: null },
  appliedVersion: { type: String, default: '' },
})

const percent = computed(() => {
  if (!props.total) return 0
  return Math.min(100, Math.round((props.downloaded / props.total) * 100))
})

const phaseLabel = computed(() => {
  if (props.error) return 'Update gagal'
  if (props.appliedVersion) return 'Update berhasil — kiosk akan restart'
  switch (props.phase) {
    case 'download': return 'Mengunduh update...'
    case 'apply': return 'Menerapkan update...'
    case 'restart': return 'Restart kiosk...'
    default: return 'Memproses update...'
  }
})

function formatMB(bytes) {
  if (!bytes) return '0 MB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
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
               max-w-[640px] w-full
               px-[clamp(28px,4vw,40px)] py-[clamp(32px,5vw,48px)]
               flex flex-col items-center gap-[clamp(16px,2.5vw,24px)] text-center"
      >
        <!-- Icon -->
        <div
          class="rounded-full flex items-center justify-center
                 w-[clamp(80px,11vw,108px)] h-[clamp(80px,11vw,108px)]"
          :class="{
            'bg-blue-light text-blue': !error && !appliedVersion,
            'bg-emerald-50 text-emerald-600': appliedVersion && !error,
            'bg-red-50 text-red-600': error,
          }"
        >
          <PhCheckCircle v-if="appliedVersion && !error" :size="64" weight="fill" />
          <PhWarningCircle v-else-if="error" :size="64" weight="fill" />
          <PhCloudArrowDown v-else :size="64" weight="fill" />
        </div>

        <!-- Title -->
        <h2 class="text-[clamp(22px,3vw,28px)] font-bold text-text-primary leading-tight">
          {{ phaseLabel }}
        </h2>

        <!-- Subtitle/error message -->
        <p
          v-if="error"
          class="text-[clamp(13px,1.7vw,15px)] text-red-700 leading-relaxed max-w-md"
        >
          {{ error }}
        </p>
        <p
          v-else-if="appliedVersion"
          class="text-[clamp(13px,1.7vw,15px)] text-text-secondary leading-relaxed"
        >
          Versi <span class="font-bold">{{ appliedVersion }}</span> sudah terpasang.
          Kiosk akan restart otomatis dalam beberapa detik.
        </p>
        <p
          v-else-if="phase === 'download'"
          class="text-[clamp(13px,1.7vw,15px)] text-text-secondary"
        >
          {{ formatMB(downloaded) }} / {{ formatMB(total) }}
          <span class="text-text-muted">— {{ percent }}%</span>
        </p>
        <p
          v-else
          class="text-[clamp(13px,1.7vw,15px)] text-text-secondary"
        >
          Mohon jangan matikan kiosk...
        </p>

        <!-- Progress bar (download only) -->
        <div
          v-if="phase === 'download' && !error && !appliedVersion"
          class="w-full bg-bg rounded-full h-3 overflow-hidden"
        >
          <div
            class="h-full transition-all duration-200"
            :style="{
              width: percent + '%',
              backgroundColor: 'var(--color-primary, #1B4FD8)',
            }"
          />
        </div>

        <!-- Apply phase: indeterminate spinner -->
        <div
          v-if="phase === 'apply' && !error && !appliedVersion"
          class="flex justify-center"
        >
          <svg class="animate-spin w-8 h-8 text-blue" viewBox="0 0 24 24" fill="none">
            <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-width="3"
                    fill="none" stroke-dasharray="40" stroke-dashoffset="20" />
          </svg>
        </div>
      </div>
    </div>
  </Teleport>
</template>
