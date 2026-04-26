<!--
  Placeholder screen — dipakai untuk route yang belum di-implement
  (P-042..P-046). Render header dengan judul + tombol kembali.
-->
<script setup>
import { useRouter } from 'vue-router'
import { I18N } from '../constants/i18n'

defineProps({
  title: { type: String, required: true },
  phase: { type: String, default: '' },
})

const router = useRouter()
</script>

<template>
  <main class="min-h-screen bg-bg flex flex-col">
    <header
      class="bg-surface border-b border-border flex items-center
             px-[clamp(16px,3vw,28px)] py-[clamp(10px,1.8vw,16px)]
             gap-[clamp(8px,1.5vw,14px)]"
    >
      <button
        type="button"
        class="text-text-secondary hover:text-text-primary
               px-3 py-1 rounded-btn flex items-center gap-1
               text-[clamp(12px,1.6vw,14px)]"
        @click="router.push({ name: 'home' })"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
          class="w-4 h-4"
        >
          <polyline points="15 18 9 12 15 6" />
        </svg>
        {{ I18N.common.back }}
      </button>
      <h1 class="text-[clamp(15px,2.2vw,18px)] font-medium text-text-primary">
        {{ title }}
      </h1>
    </header>

    <section class="flex-1 flex items-center justify-center p-8 text-center">
      <div>
        <p class="text-[clamp(14px,2vw,18px)] text-text-secondary">
          Layar ini akan tersedia di
          <span class="font-mono font-medium text-blue">{{ phase }}</span>
        </p>
        <p class="text-[clamp(11px,1.5vw,13px)] text-text-muted mt-2">
          P-041 (Home Screen) sudah selesai. Lanjut ke phase berikutnya
          untuk implementasi screen ini.
        </p>
      </div>
    </section>
  </main>
</template>
