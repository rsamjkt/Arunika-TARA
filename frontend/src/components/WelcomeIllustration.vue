<!--
  WelcomeIllustration — abstract geometric illustration untuk welcome
  banner HomeScreen. Style: flat minimal, ala Phosphor Duotone +
  unDraw geometric. Hindari menggambar figur manusia (susah bikin
  rapi inline) — fokus ke ikon medis + kios geometry.

  Color follow `--color-primary` via currentColor. Background blob
  pakai opacity rendah supaya menyatu dengan card welcome banner.

  Untuk REPLACE dengan asset unDraw resmi:
  1. Download SVG dari https://undraw.co (theme medis: "Doctors",
     "Medicine", "Welcome", "Hospital")
  2. Customize warna di unDraw editor → match primary RS
  3. Replace seluruh template <svg>...</svg>

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
    viewBox="0 0 400 400"
    xmlns="http://www.w3.org/2000/svg"
    class="select-none"
    role="img"
    aria-label="Ilustrasi anjungan pasien mandiri"
  >
    <!-- Background soft circle -->
    <circle cx="200" cy="200" r="180" fill="currentColor" opacity="0.06" />

    <!-- Ground shadow -->
    <ellipse cx="200" cy="350" rx="140" ry="10" fill="currentColor" opacity="0.10" />

    <!-- KIOSK MAIN BODY — centered, friendly rounded -->
    <!-- Stand / pole -->
    <rect x="180" y="290" width="40" height="60" rx="4"
      fill="currentColor" opacity="0.85" />
    <!-- Base -->
    <ellipse cx="200" cy="348" rx="60" ry="6" fill="currentColor" opacity="0.95" />

    <!-- Kiosk display frame (tablet-style) -->
    <rect x="110" y="100" width="180" height="200" rx="20"
      fill="white" stroke="currentColor" stroke-width="4" />

    <!-- Screen (inset) -->
    <rect x="125" y="115" width="150" height="160" rx="10"
      fill="currentColor" opacity="0.95" />

    <!-- T.A.R.A logo on screen — big friendly T -->
    <g transform="translate(200 175)">
      <circle r="42" fill="white" opacity="0.15" />
      <text
        x="0"
        y="18"
        text-anchor="middle"
        font-family="Inter, system-ui, sans-serif"
        font-size="64"
        font-weight="900"
        fill="white"
        letter-spacing="-2"
      >T</text>
    </g>

    <!-- Status indicator dot on screen -->
    <circle cx="260" cy="130" r="4" fill="#34D399" />

    <!-- Speaker grille hint at bottom of screen -->
    <rect x="170" y="248" width="60" height="3" rx="1.5" fill="white" opacity="0.4" />
    <rect x="170" y="256" width="60" height="3" rx="1.5" fill="white" opacity="0.3" />

    <!-- FLOATING ACCENT ICONS — medical badges around kiosk -->

    <!-- Plus medical cross — kiri atas -->
    <g transform="translate(70 90)">
      <circle r="26" fill="white" stroke="currentColor" stroke-width="3" />
      <rect x="-3" y="-12" width="6" height="24" rx="2" fill="currentColor" />
      <rect x="-12" y="-3" width="24" height="6" rx="2" fill="currentColor" />
    </g>

    <!-- Heart pulse — kanan atas -->
    <g transform="translate(330 100)">
      <circle r="26" fill="white" stroke="currentColor" stroke-width="3" />
      <path
        d="M 0 -2 C -8 -10 -18 -4 -18 4 C -18 12 0 18 0 18 C 0 18 18 12 18 4 C 18 -4 8 -10 0 -2 Z"
        fill="currentColor"
        opacity="0.85"
      />
    </g>

    <!-- ID Card icon — kiri bawah -->
    <g transform="translate(60 250)">
      <rect x="-26" y="-18" width="52" height="36" rx="6"
        fill="white" stroke="currentColor" stroke-width="3" />
      <circle cx="-10" cy="-2" r="6" fill="currentColor" opacity="0.5" />
      <rect x="0" y="-6" width="20" height="3" rx="1.5" fill="currentColor" opacity="0.4" />
      <rect x="0" y="0" width="14" height="3" rx="1.5" fill="currentColor" opacity="0.4" />
      <rect x="-22" y="10" width="44" height="3" rx="1.5" fill="currentColor" opacity="0.3" />
    </g>

    <!-- Antrian / ticket icon — kanan bawah -->
    <g transform="translate(340 250)">
      <rect x="-22" y="-22" width="44" height="44" rx="6"
        fill="white" stroke="currentColor" stroke-width="3" />
      <text
        x="0"
        y="6"
        text-anchor="middle"
        font-family="Inter, system-ui, sans-serif"
        font-size="20"
        font-weight="800"
        fill="currentColor"
      >A1</text>
    </g>

    <!-- Sparkle dots - decorative -->
    <circle cx="120" cy="60" r="3" fill="currentColor" opacity="0.5" />
    <circle cx="280" cy="60" r="4" fill="currentColor" opacity="0.4" />
    <circle cx="100" cy="320" r="3" fill="currentColor" opacity="0.5" />
    <circle cx="305" cy="320" r="4" fill="currentColor" opacity="0.4" />
    <circle cx="40" cy="180" r="3" fill="currentColor" opacity="0.4" />
    <circle cx="365" cy="190" r="3" fill="currentColor" opacity="0.4" />
  </svg>
</template>
