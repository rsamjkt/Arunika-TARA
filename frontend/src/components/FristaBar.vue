<!--
  FristaBar — banner status reader Frista. Selalu visible di
  InputScreen supaya pasien tahu opsi tap kartu sebagai alternatif
  ngetik manual.

  Spec DESIGN_SYSTEM.md:
    available: bg #ECFDF5, border #6EE7B7, dot hijau pulse
    not-available: dot abu, teks "Frista tidak terhubung"
-->
<script setup>
defineProps({
  available: { type: Boolean, required: true },
})

// Teks default — bisa di-override via slot kalau ingin custom message
</script>

<template>
  <div
    :class="[
      'rounded-card border flex items-center gap-[clamp(8px,1.5vw,12px)]',
      'px-[clamp(12px,2vw,16px)] py-[clamp(8px,1.4vw,12px)]',
      available
        ? 'bg-success-bg border-success-border'
        : 'bg-bg border-border',
    ]"
    role="status"
    aria-live="polite"
  >
    <span
      :class="[
        'w-[clamp(8px,1.2vw,10px)] h-[clamp(8px,1.2vw,10px)] rounded-full flex-shrink-0',
        available ? 'bg-emerald-500 animate-pulse' : 'bg-text-muted',
      ]"
    />
    <p
      :class="[
        'text-[clamp(11px,1.5vw,13px)] leading-snug flex-1',
        available ? 'text-success' : 'text-text-secondary',
      ]"
    >
      <slot>
        <template v-if="available">
          <span class="font-medium">Frista aktif</span> — tempel kartu BPJS atau KTP untuk isi otomatis
        </template>
        <template v-else>
          <span class="font-medium">Frista tidak terhubung</span> — silakan ketik nomor secara manual
        </template>
      </slot>
    </p>
  </div>
</template>
