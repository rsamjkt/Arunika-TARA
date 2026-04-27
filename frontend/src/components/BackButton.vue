<!--
  BackButton — tombol "Kembali" konsisten di kiri-bawah semua screen.

  Single-root template (Vue 3 attribute inheritance friendly).

  Props:
    needsConfirm: kalau true, tampilkan ConfirmBackModal sebelum execute
    confirmTitle, confirmMessage: copywriting modal
    label: text label (default "Kembali")

  Usage:
    <BackButton @click="back" />
    <BackButton :needs-confirm="hasUnsavedData" @click="back" />
-->
<script setup>
import { ref } from 'vue'
import { PhCaretLeft } from '@phosphor-icons/vue'
import ConfirmBackModal from './ConfirmBackModal.vue'
import { useAudioCue } from '../composables/useAudioCue'

const props = defineProps({
  needsConfirm: { type: Boolean, default: false },
  confirmTitle: { type: String, default: 'Yakin kembali ke layar sebelumnya?' },
  confirmMessage: {
    type: String,
    default: 'Data yang sudah Anda isi akan dihapus dari layar ini. Anda bisa mulai lagi dari awal.',
  },
  label: { type: String, default: 'Kembali' },
})
const emit = defineEmits(['click'])
const audio = useAudioCue()

const showModal = ref(false)

function handleTap(event) {
  // Stop event bubbling supaya parent container click tidak ikut trigger
  if (event && event.stopPropagation) event.stopPropagation()
  audio.tap()
  if (props.needsConfirm) {
    showModal.value = true
  } else {
    emit('click')
  }
}
function confirmBack() {
  showModal.value = false
  emit('click')
}
function cancelBack() {
  showModal.value = false
}
</script>

<template>
  <div class="inline-block">
    <button
      type="button"
      class="bg-surface border border-border text-text-primary
             font-medium rounded-btn cursor-pointer
             min-h-[clamp(60px,8vw,72px)]
             min-w-[clamp(140px,18vw,200px)]
             px-[clamp(16px,2.5vw,24px)] py-[clamp(12px,1.8vw,16px)]
             text-[clamp(14px,1.9vw,17px)]
             hover:border-border-strong hover:bg-bg
             active:bg-border
             flex items-center gap-3 shadow-sm
             relative z-10"
      aria-label="Kembali ke layar sebelumnya"
      @click="handleTap"
    >
      <PhCaretLeft :size="24" weight="bold" />
      <span>{{ label }}</span>
    </button>

    <ConfirmBackModal
      :visible="showModal"
      :title="confirmTitle"
      :message="confirmMessage"
      @confirm="confirmBack"
      @cancel="cancelBack"
    />
  </div>
</template>
