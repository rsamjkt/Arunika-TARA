<!--
  NumPad — keypad input angka kiosk.

  Layout 3x4:
    1 2 3
    4 5 6
    7 8 9
    [hapus] 0 [cari]

  Touch target wajib min-h clamp(52px, 7vw, 72px) per DESIGN_SYSTEM.md.
  Tombol "cari": bg #1B4FD8 (primary). Tombol "hapus": bg #F5F6F8.
-->
<script setup>
defineProps({
  // Disable cari kalau input belum cukup (validasi minimal di parent)
  canSubmit: { type: Boolean, default: true },
  loading: { type: Boolean, default: false },
})
const emit = defineEmits(['digit', 'delete', 'submit'])

// Layout dideklarasi sebagai array supaya gampang di-render & test
const keys = [
  ['1', '2', '3'],
  ['4', '5', '6'],
  ['7', '8', '9'],
  ['del', '0', 'go'],
]

function handleKey(key) {
  if (key === 'del') return emit('delete')
  if (key === 'go') return emit('submit')
  emit('digit', key)
}

function keyClass(key) {
  if (key === 'go') {
    return 'bg-blue text-white hover:bg-blue-hover active:opacity-85 disabled:opacity-50'
  }
  if (key === 'del') {
    return 'bg-bg text-text-primary hover:bg-border active:opacity-90'
  }
  // Digit buttons
  return 'bg-surface text-text-primary border border-border hover:border-border-strong active:bg-bg'
}
</script>

<template>
  <div
    class="grid grid-cols-3 gap-[clamp(6px,1.2vw,10px)]"
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
          'min-h-[clamp(52px,7vw,72px)]',
          'text-[clamp(20px,3vw,28px)]',
          'flex items-center justify-center',
          'disabled:cursor-not-allowed',
          keyClass(key),
        ]"
        :disabled="(key === 'go' && (!canSubmit || loading)) || loading"
        :aria-label="key === 'del' ? 'Hapus' : key === 'go' ? 'Cari' : `Angka ${key}`"
        @click="handleKey(key)"
      >
        <!-- Hapus: backspace icon -->
        <svg
          v-if="key === 'del'"
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          class="w-[clamp(20px,3vw,26px)] h-[clamp(20px,3vw,26px)]"
        >
          <path d="M21 4H8L1 12l7 8h13a2 2 0 0 0 2-2V6a2 2 0 0 0-2-2z" />
          <line x1="18" y1="9" x2="12" y2="15" />
          <line x1="12" y1="9" x2="18" y2="15" />
        </svg>

        <!-- Cari: search icon + text -->
        <span
          v-else-if="key === 'go'"
          class="flex items-center gap-2"
        >
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
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <span class="text-[clamp(13px,1.8vw,16px)] font-medium">Cari</span>
        </span>

        <!-- Digit -->
        <span v-else>{{ key }}</span>
      </button>
    </template>
  </div>
</template>
