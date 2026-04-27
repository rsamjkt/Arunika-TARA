// Pinia store untuk branding RS — di-load sekali di App.vue mount,
// applied ke document.documentElement sebagai CSS custom properties.
//
// Usage di komponen:
//   const branding = useBrandingStore()
//   branding.hospitalName  → tampilkan di header
//   branding.logoDataURL   → <img :src="branding.logoDataURL">
//
// CSS variables yang di-set:
//   --color-primary, --color-primary-dark, --color-accent
// → Tailwind class `bg-[var(--color-primary)]`, `text-[var(--color-primary)]`
//   tetap dipakai untuk theming dynamic.
import { defineStore } from 'pinia'
import { apmService } from '../services/apm'
import type { Branding } from '../services/apm'

const DEFAULTS = {
  hospital_name: 'Rumah Sakit',
  hospital_tagline: 'Anjungan Pasien Mandiri',
  logo_path: '',
  logo_data_url: '',
  primary_color: '#1B4FD8',
  primary_color_dark: '',
  accent_color: '',
  audio_enabled: true,
  audio_volume: 0.6,
}

interface BrandingState {
  data: Branding
  loaded: boolean
}

export const useBrandingStore = defineStore('branding', {
  state: (): BrandingState => ({
    data: DEFAULTS as Branding,
    loaded: false,
  }),

  getters: {
    hospitalName: (s) => s.data.hospital_name || DEFAULTS.hospital_name,
    hospitalTagline: (s) => s.data.hospital_tagline || DEFAULTS.hospital_tagline,
    logoDataURL: (s) => s.data.logo_data_url || '',
    primaryColor: (s) => s.data.primary_color || DEFAULTS.primary_color,
    audioEnabled: (s) => s.data.audio_enabled !== false,
    audioVolume: (s) => s.data.audio_volume || DEFAULTS.audio_volume,
  },

  actions: {
    /**
     * load — fetch branding dari Go backend, lalu apply CSS variables
     * ke document.documentElement. Dipanggil sekali di App.vue onMounted.
     */
    async load() {
      try {
        const b = await apmService.getBranding()
        this.data = b
        this.loaded = true
        this.applyCSSVariables()
      } catch (e) {
        // Fallback ke default — kiosk tetap jalan dengan biru korporat
        this.applyCSSVariables()
      }
    },

    applyCSSVariables() {
      const root = document.documentElement
      const primary = this.data.primary_color || DEFAULTS.primary_color
      const primaryDark = this.data.primary_color_dark || darkenHex(primary, 12)
      const accent = this.data.accent_color || lightenHex(primary, 40)

      root.style.setProperty('--color-primary', primary)
      root.style.setProperty('--color-primary-dark', primaryDark)
      root.style.setProperty('--color-primary-light', lightenHex(primary, 45))
      root.style.setProperty('--color-accent', accent)
    },
  },
})

// Helper: darken hex color by N percent (simple linear)
function darkenHex(hex: string, percent: number): string {
  const c = hexToRgb(hex)
  if (!c) return hex
  const factor = 1 - percent / 100
  return rgbToHex(
    Math.max(0, Math.round(c.r * factor)),
    Math.max(0, Math.round(c.g * factor)),
    Math.max(0, Math.round(c.b * factor)),
  )
}

function lightenHex(hex: string, percent: number): string {
  const c = hexToRgb(hex)
  if (!c) return hex
  const factor = percent / 100
  return rgbToHex(
    Math.min(255, Math.round(c.r + (255 - c.r) * factor)),
    Math.min(255, Math.round(c.g + (255 - c.g) * factor)),
    Math.min(255, Math.round(c.b + (255 - c.b) * factor)),
  )
}

function hexToRgb(hex: string): { r: number; g: number; b: number } | null {
  const m = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex.trim())
  if (!m) return null
  return { r: parseInt(m[1], 16), g: parseInt(m[2], 16), b: parseInt(m[3], 16) }
}

function rgbToHex(r: number, g: number, b: number): string {
  const h = (n: number) => n.toString(16).padStart(2, '0')
  return `#${h(r)}${h(g)}${h(b)}`.toUpperCase()
}
