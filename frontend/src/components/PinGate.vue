<!--
  PinGate — full-screen overlay PIN entry untuk akses admin panel.
  Validasi backend (apmService.verifyAdminPIN). Sukses → emit 'unlock'.

  Layout: kotak dialog tengah dengan title, 6-digit display dots,
  inline numpad. Wrong PIN → shake + clear input.
-->
<script setup>
import { ref } from 'vue'
import { apmService } from '../services/apm'

const emit = defineEmits(['unlock'])

const pin = ref('')
const MAX = 6
const error = ref(false)
const verifying = ref(false)

async function append(d) {
  if (verifying.value || pin.value.length >= MAX) return
  pin.value += d
  if (pin.value.length >= 4) {
    // Auto-verify pada 4 digit pertama (kalau gagal, user bisa lanjut sampai 6)
    await tryVerify()
  }
}

function del() {
  if (verifying.value) return
  pin.value = pin.value.slice(0, -1)
  error.value = false
}

async function tryVerify() {
  if (verifying.value || pin.value.length < 4) return
  verifying.value = true
  try {
    const ok = await apmService.verifyAdminPIN(pin.value)
    if (ok) {
      emit('unlock')
    } else {
      error.value = true
      // Shake animation via class toggle
      setTimeout(() => {
        pin.value = ''
        error.value = false
      }, 500)
    }
  } catch {
    error.value = true
  } finally {
    verifying.value = false
  }
}

const keys = [
  ['1', '2', '3'],
  ['4', '5', '6'],
  ['7', '8', '9'],
  ['', '0', 'del'],
]
</script>

<template>
  <div
    class="fixed inset-0 z-50 flex items-center justify-center
           bg-black/60 backdrop-blur-sm p-4"
  >
    <div class="bg-surface rounded-card max-w-[400px] w-full p-[clamp(20px,3vw,32px)]">
      <h2 class="text-center text-[clamp(15px,2.2vw,18px)] font-medium text-text-primary mb-2">
        Akses Admin
      </h2>
      <p class="text-center text-[clamp(11px,1.5vw,13px)] text-text-muted mb-6">
        Masukkan PIN untuk lanjut
      </p>

      <!-- PIN dots display -->
      <div
        :class="[
          'flex justify-center gap-3 mb-6',
          error && 'animate-shake',
        ]"
      >
        <span
          v-for="i in MAX"
          :key="i"
          :class="[
            'w-[clamp(12px,2vw,16px)] h-[clamp(12px,2vw,16px)] rounded-full transition-colors',
            i <= pin.length
              ? error ? 'bg-rose-500' : 'bg-blue'
              : 'bg-border',
          ]"
        />
      </div>

      <!-- Numpad -->
      <div class="grid grid-cols-3 gap-2">
        <template v-for="(row, ri) in keys" :key="ri">
          <button
            v-for="(key, ki) in row"
            :key="ri + '-' + ki"
            type="button"
            :class="[
              'rounded-btn border text-center font-medium transition-colors',
              'min-h-[clamp(44px,6vw,56px)] text-[clamp(16px,2.4vw,20px)]',
              key === '' && 'invisible',
              key === 'del'
                ? 'bg-bg border-border text-text-secondary'
                : key
                  ? 'bg-surface border-border hover:border-border-strong active:bg-bg'
                  : '',
            ]"
            :disabled="key === '' || verifying"
            @click="key === 'del' ? del() : key && append(key)"
          >
            <span v-if="key === 'del'" aria-label="Hapus">←</span>
            <span v-else-if="key">{{ key }}</span>
          </button>
        </template>
      </div>

      <!-- Hint -->
      <p
        v-if="error"
        class="text-center text-[clamp(10px,1.4vw,12px)] text-rose-700 mt-4"
      >
        PIN salah, silakan coba lagi
      </p>
    </div>
  </div>
</template>

<style scoped>
@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-8px); }
  75% { transform: translateX(8px); }
}
.animate-shake {
  animation: shake 300ms ease-in-out;
}
</style>
