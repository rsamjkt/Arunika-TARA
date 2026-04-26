<!--
  ProgressRing — SVG ring spinner untuk DetectScreen.

  Pakai SVG (BUKAN CSS gradient/border trick) supaya rendering crisp
  di semua DPI + bisa di-customize warna track & arc.

  Animasi: arc circle rotate via transform (continuous CW). Tidak
  pakai stroke-dasharray animation karena kurang halus di webview.
-->
<script setup>
defineProps({
  size: { type: Number, default: 60 }, // diameter px (sizing dilakukan di parent via wrapper class)
  strokeWidth: { type: Number, default: 5 },
  trackColor: { type: String, default: '#E4E6EA' },
  arcColor: { type: String, default: '#1B4FD8' },
  // Sweep angle: 75% of circle = 270deg untuk visual "loading"
  sweep: { type: Number, default: 0.25 },
})
</script>

<template>
  <svg
    :width="size"
    :height="size"
    :viewBox="`0 0 ${size} ${size}`"
    class="block"
    role="progressbar"
    aria-label="Memproses"
  >
    <!-- Track full circle -->
    <circle
      :cx="size / 2"
      :cy="size / 2"
      :r="size / 2 - strokeWidth"
      fill="none"
      :stroke="trackColor"
      :stroke-width="strokeWidth"
    />
    <!-- Arc rotating -->
    <circle
      :cx="size / 2"
      :cy="size / 2"
      :r="size / 2 - strokeWidth"
      fill="none"
      :stroke="arcColor"
      :stroke-width="strokeWidth"
      stroke-linecap="round"
      :stroke-dasharray="2 * Math.PI * (size / 2 - strokeWidth)"
      :stroke-dashoffset="2 * Math.PI * (size / 2 - strokeWidth) * sweep"
      :transform="`rotate(-90 ${size / 2} ${size / 2})`"
      class="spin-arc"
    />
  </svg>
</template>

<style scoped>
.spin-arc {
  transform-origin: center;
  animation: spin 1.1s linear infinite;
}
@keyframes spin {
  from { transform: rotate(-90deg); }
  to   { transform: rotate(270deg); }
}
</style>
