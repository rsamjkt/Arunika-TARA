<!--
  StatusList — daftar komponen sistem dengan StatusPill di kanan.
  Dipakai admin panel section "Status komponen".
-->
<script setup>
import StatusPill from './StatusPill.vue'

defineProps({
  // [{ label, status: 'online'|'offline'|'warning'|'idle', detail?: string }]
  items: { type: Array, required: true },
})

const pillVariant = (status) => {
  switch (status) {
    case 'online': return 'success'
    case 'offline': return 'danger'
    case 'warning': return 'warning'
    default: return 'info'
  }
}
const pillLabel = (status, detail) => {
  if (detail) return detail
  switch (status) {
    case 'online': return 'Online'
    case 'offline': return 'Offline'
    case 'warning': return 'Perhatian'
    default: return 'Tidak diketahui'
  }
}
</script>

<template>
  <div class="bg-surface border border-border rounded-card divide-y divide-border">
    <div
      v-for="item in items"
      :key="item.label"
      class="flex items-center justify-between gap-3
             px-[clamp(12px,2vw,16px)] py-[clamp(10px,1.5vw,12px)]"
    >
      <span class="text-[clamp(11px,1.5vw,13px)] text-text-primary">
        {{ item.label }}
      </span>
      <StatusPill
        :label="pillLabel(item.status, item.detail)"
        :variant="pillVariant(item.status)"
      />
    </div>
  </div>
</template>
