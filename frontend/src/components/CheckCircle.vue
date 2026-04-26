<!--
  CheckCircle — sukses indicator bulat dengan checkmark.

  Spec P-045:
    44-50px background #ECFDF5 border #6EE7B7
    SVG checkmark stroke #065F46
-->
<script setup>
defineProps({
  size: { type: Number, default: 50 },
})
</script>

<template>
  <div
    class="rounded-full bg-success-bg border border-success-border
           flex items-center justify-center mx-auto"
    :style="{ width: `clamp(${size - 6}px, 8vw, ${size}px)`, height: `clamp(${size - 6}px, 8vw, ${size}px)` }"
  >
    <svg
      xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
      stroke="#065F46" stroke-width="3" stroke-linecap="round"
      stroke-linejoin="round"
      class="w-[60%] h-[60%]"
    >
      <polyline points="20 6 9 17 4 12" />
    </svg>
  </div>
</template>
