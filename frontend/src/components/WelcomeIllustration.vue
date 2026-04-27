<!--
  WelcomeIllustration — vector ilustrasi medis untuk welcome banner
  HomeScreen. Style: ala unDraw (geometric clean lines, single accent
  color via currentColor + secondary, customizable via theme).

  Default: scene "registrasi pasien di kios" — petugas + kartu +
  stetoskop. Color follow `--color-primary` dari useBrandingStore.

  Untuk REPLACE dengan asset unDraw resmi:
  1. Download SVG dari https://undraw.co (pilih medis theme: "Doctors",
     "Medicine", "Online doctor", "Hospital", "Welcome", dll)
  2. Customize color di unDraw editor → match primary color RS Anda
  3. Replace seluruh template <svg>...</svg> di file ini dengan SVG
     hasil download
  4. Pastikan SVG punya viewBox yang sesuai + width/height responsive

  Props:
    size: "md" (240px) | "lg" (320px) | "xl" (400px)
-->
<script setup>
import { computed } from 'vue'

const props = defineProps({
  size: { type: String, default: 'lg' },
})

const sizes = { md: 240, lg: 320, xl: 400 }
const dim = computed(() => sizes[props.size] || sizes.lg)
</script>

<template>
  <svg
    :width="dim"
    :height="dim"
    viewBox="0 0 600 600"
    xmlns="http://www.w3.org/2000/svg"
    class="select-none"
    role="img"
    aria-label="Ilustrasi welcome — petugas RS dan pasien"
  >
    <!-- Background blob — soft circle accent -->
    <circle
      cx="300"
      cy="300"
      r="240"
      fill="currentColor"
      opacity="0.08"
    />
    <ellipse
      cx="300"
      cy="540"
      rx="220"
      ry="20"
      fill="currentColor"
      opacity="0.15"
    />

    <!-- Petugas RS standing — body silhouette -->
    <!-- Head -->
    <circle cx="220" cy="200" r="42" fill="#FFD7BB" />
    <!-- Hair / cap (medical scrub cap) -->
    <path
      d="M 178 195 Q 178 158 220 158 Q 262 158 262 195 L 262 200 L 178 200 Z"
      fill="currentColor"
      opacity="0.85"
    />
    <!-- Body / scrubs -->
    <path
      d="M 175 250 Q 175 240 195 235 L 245 235 Q 265 240 265 250 L 280 380 Q 280 395 265 395 L 175 395 Q 160 395 160 380 Z"
      fill="currentColor"
    />
    <!-- White lab coat collar V -->
    <path
      d="M 200 240 L 220 280 L 240 240 Z"
      fill="white"
      opacity="0.9"
    />
    <!-- Stethoscope -->
    <circle cx="220" cy="320" r="14" fill="none" stroke="#A0522D" stroke-width="3" />
    <path
      d="M 220 306 Q 220 270 195 268 M 220 306 Q 220 270 245 268"
      stroke="#A0522D"
      stroke-width="3"
      fill="none"
    />
    <!-- Arms -->
    <path
      d="M 175 250 L 145 320 Q 140 330 145 340 L 155 350"
      stroke="currentColor"
      stroke-width="20"
      fill="none"
      stroke-linecap="round"
    />
    <path
      d="M 265 250 L 295 320 Q 300 330 320 340 L 340 345"
      stroke="currentColor"
      stroke-width="20"
      fill="none"
      stroke-linecap="round"
    />
    <!-- Hand holding card -->
    <circle cx="340" cy="345" r="14" fill="#FFD7BB" />

    <!-- BPJS card icon (in petugas's hand, slight tilt) -->
    <g transform="rotate(-12 380 360)">
      <rect
        x="335"
        y="332"
        width="100"
        height="60"
        rx="8"
        fill="white"
        stroke="currentColor"
        stroke-width="3"
      />
      <!-- BPJS card details -->
      <rect x="345" y="342" width="30" height="20" rx="3" fill="currentColor" opacity="0.4" />
      <rect x="345" y="368" width="80" height="4" rx="2" fill="currentColor" opacity="0.3" />
      <rect x="345" y="376" width="50" height="4" rx="2" fill="currentColor" opacity="0.3" />
      <!-- "BPJS" text label -->
      <text
        x="395"
        y="356"
        font-family="Arial, sans-serif"
        font-size="10"
        font-weight="bold"
        fill="currentColor"
        opacity="0.7"
      >BPJS</text>
    </g>

    <!-- Kios / counter di kanan (pasien-side perspective) -->
    <rect
      x="430"
      y="350"
      width="120"
      height="180"
      rx="8"
      fill="white"
      stroke="currentColor"
      stroke-width="3"
    />
    <!-- Screen on kios -->
    <rect
      x="445"
      y="365"
      width="90"
      height="70"
      rx="4"
      fill="currentColor"
      opacity="0.85"
    />
    <!-- "T" letter mark (T.A.R.A) di screen -->
    <text
      x="490"
      y="412"
      text-anchor="middle"
      font-family="Arial, sans-serif"
      font-size="42"
      font-weight="900"
      fill="white"
    >T</text>
    <!-- Kios button strip -->
    <rect x="455" y="450" width="70" height="6" rx="3" fill="currentColor" opacity="0.3" />
    <rect x="455" y="465" width="70" height="6" rx="3" fill="currentColor" opacity="0.3" />
    <rect x="455" y="480" width="70" height="6" rx="3" fill="currentColor" opacity="0.3" />

    <!-- Plus medis (medical cross icon) — floating accent kiri atas -->
    <g transform="translate(85 130)">
      <circle r="28" fill="white" stroke="currentColor" stroke-width="3" />
      <rect x="-4" y="-14" width="8" height="28" rx="2" fill="currentColor" />
      <rect x="-14" y="-4" width="28" height="8" rx="2" fill="currentColor" />
    </g>

    <!-- Heart icon — floating accent kanan atas -->
    <g transform="translate(515 150)">
      <path
        d="M 0 6 C -10 -8 -28 -8 -28 6 C -28 18 0 32 0 32 C 0 32 28 18 28 6 C 28 -8 10 -8 0 6 Z"
        fill="currentColor"
        opacity="0.7"
      />
    </g>

    <!-- Sparkles — small dots accent -->
    <circle cx="120" cy="380" r="4" fill="currentColor" opacity="0.5" />
    <circle cx="155" cy="430" r="3" fill="currentColor" opacity="0.4" />
    <circle cx="100" cy="280" r="3" fill="currentColor" opacity="0.5" />
    <circle cx="490" cy="270" r="4" fill="currentColor" opacity="0.5" />
  </svg>
</template>
