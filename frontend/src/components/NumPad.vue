<!--
  NumPad — keypad input angka kiosk (post QW2 redesign).

  Layout 4x3 (3-wide, 4-baris):
    1   2   3
    4   5   6
    7   8   9
    HAPUS   0   CARI

  Perbaikan QW2:
    - Tombol "HAPUS" pakai TEXT label (bukan ikon backspace ambigu)
    - Tombol "CARI" tetap col-span-1 tapi diperbesar font + ikon Phosphor magnifying-glass
    - Tombol angka diperbesar minimum 60px (sebelumnya 52px) untuk lansia friendly
    - Gap antar tombol diperlebar 10px (sebelumnya 6px) — easier finger separation
    - Audio cue tap saat tekan
-->
<script setup>
import { PhMagnifyingGlass } from '@phosphor-icons/vue'
import { useAudioCue } from '../composables/useAudioCue'

defineProps({
  canSubmit: { type: Boolean, default: true },
  loading: { type: Boolean, default: false },
})
const emit = defineEmits(['digit', 'delete', 'submit'])
const audio = useAudioCue()

const keys = [
  ['1', '2', '3'],
  ['4', '5', '6'],
  ['7', '8', '9'],
  ['del', '0', 'go'],
]

function handleKey(key) {
  audio.tap()
  if (key === 'del') return emit('delete')
  if (key === 'go') return emit('submit')
  emit('digit', key)
}

function keyClass(key) {
  if (key === 'go') {
    return 'text-white hover:opacity-90 active:opacity-80 disabled:opacity-50'
  }
  if (key === 'del') {
    return 'bg-amber-50 text-amber-800 hover:bg-amber-100 active:bg-amber-200 border border-amber-200'
  }
  return 'bg-surface text-text-primary border border-border hover:border-border-strong active:bg-bg'
}
function keyStyle(key) {
  if (key === 'go') {
    return { backgroundColor: 'var(--color-primary, #1B4FD8)' }
  }
  return {}
}
</script>

<template>
  <div
    class="grid grid-cols-3 gap-[clamp(10px,1.5vw,14px)]"
    role="group"
    aria-label="Keypad angka"
  >
    <template v-for="(row, ri) in keys" :key="ri">
      <button
        v-for="key in row"
        :key="key"
        type="button"
        :class="[
          'rounded-kiosk text-center font-medium transition-colors',
          'min-h-[clamp(60px,8vw,80px)]',
          'text-[clamp(22px,3.2vw,30px)]',
          'flex items-center justify-center gap-2',
          'disabled:cursor-not-allowed',
          keyClass(key),
        ]"
        :style="keyStyle(key)"
        :disabled="(key === 'go' && (!canSubmit || loading)) || loading"
        :aria-label="key === 'del' ? 'Hapus angka terakhir' : key === 'go' ? 'Cari pasien' : `Angka ${key}`"
        @click="handleKey(key)"
      >
        <!-- HAPUS — text label, bukan ikon -->
        <span
          v-if="key === 'del'"
          class="text-[clamp(15px,2vw,18px)] font-semibold tracking-wide"
        >
          HAPUS
        </span>

        <!-- CARI — ikon Phosphor + text label besar -->
        <template v-else-if="key === 'go'">
          <PhMagnifyingGlass :size="22" weight="bold" />
          <span class="text-[clamp(16px,2.2vw,20px)] font-semibold">CARI</span>
        </template>

        <!-- Digit angka -->
        <span v-else>{{ key }}</span>
      </button>
    </template>
  </div>
</template>
