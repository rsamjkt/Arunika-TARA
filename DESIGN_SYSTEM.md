# Design System — APM Kiosk UI

> Memory untuk semua keputusan desain Vue 3 + Tailwind CSS.
> Referensi: airport check-in kiosk aesthetic — bersih, putih, presisi.

## Prinsip Desain

1. **Satu aksi utama per layar** — satu CTA besar, satu arah
2. **Warna minimal** — satu aksen biru, tiga semantik (hijau/kuning/merah)
3. **Responsif lewat `clamp()`** — bekerja di 15" hingga 32" monitor
4. **Touch target minimum** — `min-height: clamp(52px, 7vw, 72px)`
5. **Bahasa Indonesia sederhana** — tidak ada istilah teknis BPJS untuk pasien

## Design Tokens (Tailwind Config)

```js
// tailwind.config.js
module.exports = {
  theme: {
    extend: {
      colors: {
        // Primary — satu-satunya aksen
        blue: {
          DEFAULT: '#1B4FD8',
          light: '#EEF2FF',
          dark: '#1E40AF',
          hover: '#1648C5',
        },
        // Semantik
        success: { DEFAULT: '#065F46', bg: '#ECFDF5', border: '#6EE7B7' },
        warning: { DEFAULT: '#92400E', bg: '#FFFBEB', border: '#FCD34D' },
        danger:  { DEFAULT: '#991B1B', bg: '#FEF2F2', border: '#FCA5A5' },
        // Neutral
        surface: '#FFFFFF',
        bg:      '#F5F6F8',
        border:  '#E4E6EA',
        'border-strong': '#CDD1D9',
        text: {
          primary:   '#0E1117',
          secondary: '#4B5563',
          muted:     '#9CA3AF',
        },
      },
      borderRadius: {
        kiosk: '14px',
        card:  '12px',
        btn:   '10px',
        tag:   '9999px',
      },
      fontSize: {
        // SEMUA menggunakan clamp() untuk responsivitas
        // Format: [min, preferred (vw), max]
        'kiosk-hero':   ['clamp(16px,2.5vw,22px)', { lineHeight: '1.3' }],
        'kiosk-title':  ['clamp(13px,2vw,17px)',   { lineHeight: '1.4' }],
        'kiosk-body':   ['clamp(11px,1.6vw,14px)', { lineHeight: '1.6' }],
        'kiosk-label':  ['clamp(10px,1.3vw,12px)', { lineHeight: '1.5' }],
        'kiosk-micro':  ['clamp(9px,1.1vw,11px)',  { lineHeight: '1.4' }],
        'kiosk-number': ['clamp(44px,7vw,60px)',   { lineHeight: '1'   }],
      },
    },
  },
}
```

## Komponen Wajib (src/components/)

### `<BigButton>` — Tombol utama kiosk

```vue
<!-- src/components/BigButton.vue -->
<template>
  <button
    :class="[
      'w-full rounded-kiosk border transition-opacity active:opacity-85',
      'flex items-center gap-[clamp(10px,2vw,16px)]',
      'p-[clamp(14px,2.5vw,20px)]',
      variant === 'primary'
        ? 'bg-blue text-white border-blue'
        : 'bg-surface text-text-primary border-border',
    ]"
    :disabled="loading"
    @click="$emit('click')"
  >
    <div
      v-if="icon"
      :class="[
        'rounded-[10px] flex items-center justify-center flex-shrink-0',
        'w-[clamp(40px,6vw,52px)] h-[clamp(40px,6vw,52px)]',
        variant === 'primary' ? 'bg-white/20' : 'bg-blue-light',
      ]"
    >
      <component :is="icon" class="w-[clamp(18px,2.8vw,24px)] h-[clamp(18px,2.8vw,24px)]" />
    </div>
    <div class="text-left">
      <div class="text-kiosk-title font-medium">{{ title }}</div>
      <div v-if="subtitle" class="text-kiosk-label mt-1 opacity-70">{{ subtitle }}</div>
      <span v-if="tag" class="inline-block mt-[6px] px-3 py-1 rounded-tag text-kiosk-micro font-medium bg-white/20">
        {{ tag }}
      </span>
    </div>
    <div v-if="loading" class="ml-auto">
      <Spinner class="w-5 h-5" />
    </div>
  </button>
</template>
```

### `<PatientCard>` — Kartu info pasien

```vue
<!-- src/components/PatientCard.vue -->
<template>
  <div class="bg-surface border border-border rounded-card p-[clamp(12px,2vw,16px)]">
    <!-- Header: pill status + tanggal -->
    <div class="flex items-center justify-between mb-[clamp(8px,1.5vw,12px)]">
      <StatusPill :type="statusType" :label="statusLabel" />
      <span class="text-kiosk-micro text-text-muted">{{ dateLabel }}</span>
    </div>
    
    <!-- Nama & nomor kartu -->
    <div class="text-kiosk-hero font-medium text-text-primary">{{ nama }}</div>
    <div class="font-mono text-kiosk-label text-text-muted mt-1 tracking-wide">
      {{ formatKartu(noKartu) }}
    </div>
    
    <hr class="border-border my-[clamp(8px,1.2vw,10px)]">
    
    <!-- Key-value rows -->
    <div v-for="row in details" :key="row.key" class="flex justify-between items-start gap-2 py-[3px]">
      <span class="text-kiosk-label text-text-muted">{{ row.key }}</span>
      <span :class="['text-kiosk-body font-medium text-right', row.accent ? 'text-blue' : 'text-text-primary']">
        {{ row.value }}
      </span>
    </div>
  </div>
</template>
```

### `<NumPad>` — Keypad input angka

```vue
<!-- src/components/NumPad.vue -->
<!-- Min touch target: clamp(52px, 7vw, 72px) -->
<template>
  <div class="grid grid-cols-3 gap-[clamp(5px,1vw,8px)]">
    <button
      v-for="key in keys"
      :key="key"
      :class="[
        'rounded-kiosk border text-center font-medium transition-colors',
        'min-h-[clamp(52px,7vw,72px)]',
        keyClass(key),
      ]"
      @click="handleKey(key)"
    >
      {{ key === 'del' ? '←' : key === 'go' ? 'Cari' : key }}
    </button>
  </div>
</template>
```

### `<StatusPill>` — Badge status

```vue
<!-- 4 varian: success, info, warning, danger -->
<template>
  <span :class="['inline-flex items-center gap-[5px] px-[10px] py-1 rounded-tag text-kiosk-micro font-medium', classes]">
    <span class="w-[5px] h-[5px] rounded-full" :class="dotClass"></span>
    {{ label }}
  </span>
</template>
```

### `<DetectionLoader>` — Animasi saat Smart Detector berjalan

```vue
<!-- Progress ring + step list yang update realtime -->
<template>
  <div class="flex flex-col gap-[clamp(8px,1.5vw,12px)]">
    <!-- Spinner ring -->
    <div class="flex flex-col items-center gap-2 py-[clamp(10px,2vw,16px)]">
      <SpinRing class="w-[clamp(48px,7vw,60px)] h-[clamp(48px,7vw,60px)]" />
      <p class="text-kiosk-title font-medium text-text-primary">{{ title }}</p>
      <p class="text-kiosk-label text-text-muted">{{ subtitle }}</p>
    </div>
    
    <!-- Step list -->
    <div class="flex flex-col gap-[5px]">
      <DetectionStep
        v-for="step in steps"
        :key="step.id"
        :label="step.label"
        :state="step.state"  <!-- 'done' | 'active' | 'wait' -->
      />
    </div>
  </div>
</template>
```

### `<FingerprintWidget>` — Layar verifikasi sidik jari

```vue
<!-- Animasi pulse ring + instruksi besar -->
<template>
  <div class="flex flex-col items-center gap-[clamp(12px,2vw,16px)] py-[clamp(16px,3vw,24px)]">
    <!-- Animasi ring -->
    <div class="relative w-[clamp(80px,12vw,110px)] h-[clamp(80px,12vw,110px)]">
      <FPRing :state="state" />  <!-- idle | scanning | success | failed -->
    </div>
    
    <!-- Instruksi -->
    <p class="text-kiosk-hero font-medium text-text-primary text-center">
      {{ stateMessage }}
    </p>
    <p class="text-kiosk-body text-text-muted text-center">
      {{ stateSubtitle }}
    </p>
    
    <!-- Timeout progress -->
    <div v-if="state === 'scanning'" class="w-full bg-border rounded-full h-1">
      <div class="bg-blue h-1 rounded-full transition-all" :style="{ width: progressPct + '%' }"></div>
    </div>
  </div>
</template>
```

## Responsivitas — Aturan Ketat

### Breakpoint Strategy

APM berjalan fullscreen di monitor kiosk. Tidak ada mobile breakpoint — hanya fluid scaling:

```css
/* SELALU gunakan clamp() untuk ukuran yang bervariasi */
/* Format: clamp(minimum, preferred-vw, maximum) */

/* Font sizes */
font-size: clamp(14px, 2.2vw, 18px);   /* hero text */
font-size: clamp(11px, 1.6vw, 14px);   /* body */
font-size: clamp(9px, 1.2vw, 12px);    /* label/caption */

/* Padding & gap */
padding: clamp(12px, 2.5vw, 20px);
gap: clamp(8px, 1.5vw, 12px);

/* Touch targets (WAJIB min 52px) */
min-height: clamp(52px, 7vw, 72px);
min-width: clamp(52px, 7vw, 72px);

/* Nomor tiket besar */
font-size: clamp(44px, 8vw, 64px);
```

### Grid Layout

```html
<!-- 2 kolom untuk tombol sekunder -->
<div class="grid grid-cols-2 gap-[clamp(8px,1.5vw,12px)]">...</div>

<!-- Full width untuk tombol primary -->
<button class="w-full">...</button>
```

## Pinia Stores

### `usePatientStore` — State deteksi pasien

```ts
// src/stores/patient.ts
export const usePatientStore = defineStore('patient', {
  state: () => ({
    input: '',                    // Raw input user
    peserta: null as Peserta | null,
    detectionResult: null as DetectionResult | null,
    isDetecting: false,
    detectSteps: [] as DetectStep[],
    error: null as string | null,
  }),
  actions: {
    async detect(input: string) {
      this.isDetecting = true
      this.input = input
      // Call Wails binding → Go Detector.Detect()
      const result = await DetectPatient(input)
      this.detectionResult = result
      this.isDetecting = false
    },
    reset() {
      this.$reset()
    }
  }
})
```

### `useAntrianStore` — State antrian

```ts
export const useAntrianStore = defineStore('antrian', {
  state: () => ({
    lastTicket: null as Ticket | null,
    counters: {} as Record<string, number>,
  }),
  actions: {
    async ambilAntrian(jenis: AntrianJenis, subJenis?: string) {
      const ticket = await CreateAntrian({ jenis, subJenis })
      this.lastTicket = ticket
      return ticket
    }
  }
})
```

## Screen Navigation Map

```
HomeScreen
├── [BPJS] → InputScreen → DetectScreen → ResultScreen (MJKN/Kontrol/Baru/Nonaktif)
│                                              └── [Konfirmasi] → TicketScreen
├── [Umum] → SearchPatientScreen → RegistrasiScreen → TicketScreen
└── [Antrian] → AntrianScreen → TicketScreen

TicketScreen → [auto 10 detik] → HomeScreen
HomeScreen → [60 detik idle] → HomeScreen (reset)

AdminScreen → [terpisah, akses via PIN]
```

## Wails IPC Binding Conventions

```go
// Semua method yang di-expose ke Vue harus:
// 1. Mulai dengan huruf kapital (exported)
// 2. Return (data, error) — error selalu jadi nilai kedua
// 3. Ada di struct App di cmd/apm/app.go

// app.go
func (a *App) DetectPatient(input string) (*domain.DetectionResult, error) {
    return a.detector.Detect(a.ctx, domain.PatientInput{Identifier: input})
}

func (a *App) CreateAntrian(req domain.AntrianRequest) (*domain.Ticket, error) {
    return a.antrianService.Create(a.ctx, req)
}
```

```ts
// frontend: auto-generated bindings oleh Wails
import { DetectPatient, CreateAntrian } from '../wailsjs/go/main/App'

// Usage di Vue:
const result = await DetectPatient(input.value)
```

