<!--
  DokterPicker — list radio dokter yang bertugas di poli pada hari ini.
  Default selected: index 0 (yang paling awal).

  Spec P-043:
    - Highlight pilihan pertama
    - Card per dokter dengan nama, jam praktik, sisa kuota
    - Touch target min-h clamp(52-72px) wajib
-->
<script setup>
import { computed } from 'vue'

const props = defineProps({
  // List dokter — JadwalDokter[]
  list: { type: Array, default: () => [] },
  // Selected kode dokter (parent control via v-model)
  modelValue: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue'])

const isSelected = (kdDokter) => props.modelValue === kdDokter

function pick(d) {
  if (!d.Aktif) return // dokter cuti
  emit('update:modelValue', d.KdDokter)
}

const empty = computed(() => props.list.length === 0)
</script>

<template>
  <div class="flex flex-col gap-[clamp(6px,1vw,8px)]">
    <p
      v-if="empty"
      class="text-[clamp(11px,1.5vw,13px)] text-text-muted text-center
             py-4 bg-bg rounded-card border border-border"
    >
      Belum ada jadwal dokter untuk poli ini hari ini.
    </p>
    <button
      v-for="d in list"
      :key="d.KdDokter"
      type="button"
      :disabled="!d.Aktif"
      :class="[
        'rounded-card border transition-colors text-left',
        'px-[clamp(12px,2vw,16px)] py-[clamp(10px,1.6vw,12px)]',
        'min-h-[clamp(52px,7vw,72px)]',
        'flex items-center gap-[clamp(10px,1.5vw,14px)]',
        isSelected(d.KdDokter)
          ? 'bg-blue-light border-blue text-blue-dark'
          : 'bg-surface border-border hover:border-border-strong',
        !d.Aktif && 'opacity-50 cursor-not-allowed line-through',
      ]"
      :aria-pressed="isSelected(d.KdDokter)"
      @click="pick(d)"
    >
      <!-- Radio indicator -->
      <span
        :class="[
          'w-[clamp(16px,2.2vw,20px)] h-[clamp(16px,2.2vw,20px)] rounded-full border-2 shrink-0',
          'flex items-center justify-center',
          isSelected(d.KdDokter) ? 'border-blue' : 'border-border-strong',
        ]"
      >
        <span
          v-if="isSelected(d.KdDokter)"
          class="w-[clamp(8px,1.2vw,10px)] h-[clamp(8px,1.2vw,10px)] rounded-full bg-blue"
        />
      </span>

      <div class="flex-1 min-w-0">
        <div class="text-[clamp(12px,1.7vw,14px)] font-medium leading-tight">
          {{ d.NmDokter || d.KdDokter }}
        </div>
        <div class="text-[clamp(10px,1.3vw,12px)] text-text-muted mt-1 leading-tight">
          <span v-if="d.JamMulai && d.JamSelesai">
            {{ d.JamMulai }}–{{ d.JamSelesai }}
          </span>
          <span v-if="d.Aktif && d.Kuota > 0" class="ml-2">
            Sisa kuota: <span class="font-medium text-text-primary">{{ d.Sisa }}</span>/{{ d.Kuota }}
          </span>
          <span v-if="!d.Aktif" class="text-rose-700 font-medium">Cuti hari ini</span>
        </div>
      </div>
    </button>
  </div>
</template>
