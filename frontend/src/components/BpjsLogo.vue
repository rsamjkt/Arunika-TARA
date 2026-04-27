<!--
  BpjsLogo — komponen visual logo BPJS Kesehatan untuk flow Pasien BPJS.

  Default: SVG inline dengan warna brand BPJS (hijau-tosca #00A99D + plus medis)
  + text "BPJS Kesehatan" — readable di kiosk dari jarak 1-2 meter.

  Bisa di-override via [branding] bpjs_logo_path di config.toml kalau RS
  punya file logo resmi (PNG/SVG dari brosur BPJS daerah). Lihat
  useBrandingStore.bpjsLogoDataURL.

  Props:
    size: "sm" | "md" | "lg" — preset ukuran (default md)
    variant: "full" | "icon" | "text" — full = logo + text, icon = perisai saja, text = "BPJS Kesehatan" saja
    inverse: bool — true = warna putih (untuk background gelap mis. hero biru)

  Pakai:
    <BpjsLogo />                                   <!-- default md, full -->
    <BpjsLogo size="lg" inverse />                 <!-- besar, putih (hero BPJS) -->
    <BpjsLogo variant="icon" />                    <!-- perisai aja untuk badge -->
-->
<script setup>
import { computed } from 'vue'
import { useBrandingStore } from '../stores/branding'

const props = defineProps({
  size: { type: String, default: 'md' },           // sm | md | lg
  variant: { type: String, default: 'full' },      // full | icon | text
  inverse: { type: Boolean, default: false },
})
const branding = useBrandingStore()

const sizes = {
  sm: { icon: 32, text: '14px', spacing: 'gap-2' },
  md: { icon: 48, text: '18px', spacing: 'gap-3' },
  lg: { icon: 72, text: '26px', spacing: 'gap-4' },
}
const cur = computed(() => sizes[props.size] || sizes.md)

// Color palette BPJS Kesehatan (resmi):
//   Hijau-tosca utama: #00A99D
//   Hijau gelap secondary: #007A6F
//   Putih untuk inverse di hero biru
const fill = computed(() => props.inverse ? '#FFFFFF' : '#00A99D')
const fillSecondary = computed(() => props.inverse ? '#FFFFFF' : '#007A6F')
const textColor = computed(() => props.inverse ? '#FFFFFF' : '#0E1117')
const subColor = computed(() => props.inverse ? 'rgba(255,255,255,0.85)' : '#5C6470')

// Kalau RS pasok logo BPJS sendiri lewat config, prefer itu
const customLogoUrl = computed(() => branding.data?.bpjs_logo_data_url || '')
</script>

<template>
  <!-- Custom logo override dari config — tampilkan langsung -->
  <img
    v-if="customLogoUrl"
    :src="customLogoUrl"
    alt="Logo BPJS Kesehatan"
    :style="{ height: cur.icon + 'px', width: 'auto' }"
    class="object-contain"
  />

  <!-- Default SVG inline + text -->
  <div v-else class="flex items-center" :class="cur.spacing">
    <!-- Icon perisai BPJS-style -->
    <svg
      v-if="variant !== 'text'"
      :width="cur.icon"
      :height="cur.icon"
      viewBox="0 0 100 100"
      xmlns="http://www.w3.org/2000/svg"
      aria-label="Logo BPJS Kesehatan"
      role="img"
    >
      <!-- Outer rounded square (BPJS branding rounded shield) -->
      <rect x="6" y="6" width="88" height="88" rx="20" :fill="fill" />

      <!-- Inner: lingkaran putih dengan plus medis -->
      <circle cx="50" cy="42" r="18" fill="#FFFFFF" />
      <!-- Plus / cross medical -->
      <rect x="46" y="30" width="8" height="24" rx="2" :fill="fill" />
      <rect x="38" y="38" width="24" height="8" rx="2" :fill="fill" />

      <!-- Text "BPJS" di bawah -->
      <text
        x="50"
        y="80"
        text-anchor="middle"
        font-family="Arial, Helvetica, sans-serif"
        font-weight="800"
        font-size="14"
        fill="#FFFFFF"
        letter-spacing="0.5"
      >BPJS</text>
    </svg>

    <!-- Text "BPJS Kesehatan" -->
    <div v-if="variant !== 'icon'" class="flex flex-col leading-tight">
      <span
        :style="{ fontSize: cur.text, color: textColor }"
        class="font-bold tracking-tight"
      >
        BPJS Kesehatan
      </span>
      <span
        v-if="size !== 'sm'"
        :style="{ fontSize: 'calc(' + cur.text + ' * 0.55)', color: subColor }"
        class="font-medium tracking-wide"
      >
        Jaminan Kesehatan Nasional
      </span>
    </div>
  </div>
</template>
