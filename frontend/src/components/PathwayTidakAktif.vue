<!--
  PathwayTidakAktif — Smart Detector pathway untuk PatientType.TidakAktif.

  Pasien BPJS-nya tidak aktif (PVCH 0 atau VClaim balikkan status non-aktif).
  Kiosk tetap bantu pasien dgn rute alternatif:
    1. Daftar sebagai pasien umum (in-house, langsung billing).
    2. Hubungi petugas (operator nanti aktivasi BPJS manual / arahkan).

  Props: -
  Events:
    daftar-umum — parent reset & navigate ke /input?mode=umum.
    hubungi-petugas — sementara no-op visual; nanti bisa emit event ke
                      Wails App (audible alert ke meja petugas).
-->
<script setup>
defineEmits(['daftar-umum', 'hubungi-petugas'])
</script>

<template>
  <section class="flex flex-col gap-[clamp(10px,1.8vw,14px)]">
    <!-- Info bar danger -->
    <div
      class="rounded-card border bg-danger-bg text-danger border-danger-border
             p-[clamp(12px,2vw,16px)] flex items-start gap-2"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
        stroke="currentColor" stroke-width="2.2" stroke-linecap="round"
        stroke-linejoin="round" class="w-5 h-5 mt-[2px] shrink-0"
      >
        <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
        <line x1="12" y1="9" x2="12" y2="13" />
        <line x1="12" y1="17" x2="12.01" y2="17" />
      </svg>
      <div class="flex-1">
        <p class="text-[clamp(12px,1.6vw,14px)] font-medium leading-snug">
          Status BPJS Anda saat ini tidak aktif
        </p>
        <p class="text-[clamp(11px,1.5vw,13px)] leading-snug mt-1 text-danger/90">
          Hubungi BPJS Kesehatan terdekat untuk reaktivasi, atau daftar
          sebagai pasien umum untuk pelayanan hari ini.
        </p>
      </div>
    </div>

    <!-- CTA primary: daftar umum -->
    <button
      type="button"
      :class="[
        'w-full rounded-kiosk transition-opacity active:opacity-85',
        'bg-blue text-white border border-blue',
        'px-[clamp(14px,2.5vw,20px)] py-[clamp(14px,2.5vw,20px)]',
        'text-[clamp(14px,2vw,17px)] font-medium',
        'flex items-center justify-between gap-3',
        'min-h-[clamp(56px,8vw,72px)]',
      ]"
      @click="$emit('daftar-umum')"
    >
      <span class="flex items-center gap-2">
        <svg
          xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
          stroke="currentColor" stroke-width="2.2" stroke-linecap="round"
          stroke-linejoin="round" class="w-5 h-5 shrink-0"
        >
          <circle cx="12" cy="8" r="4" />
          <path d="M4 21v-2a4 4 0 0 1 4-4h8a4 4 0 0 1 4 4v2" />
        </svg>
        Daftar sebagai pasien umum
      </span>
      <svg
        xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
        stroke="currentColor" stroke-width="2.5" stroke-linecap="round"
        stroke-linejoin="round" class="w-5 h-5 shrink-0"
      >
        <polyline points="9 18 15 12 9 6" />
      </svg>
    </button>

    <!-- Ghost: hubungi petugas -->
    <button
      type="button"
      class="w-full rounded-kiosk transition-colors
             bg-surface text-text-secondary border border-border
             hover:border-border-strong active:bg-bg
             px-[clamp(12px,2vw,16px)] py-[clamp(12px,2vw,16px)]
             text-[clamp(12px,1.7vw,14px)] font-medium
             min-h-[clamp(48px,7vw,60px)]
             flex items-center justify-center gap-2"
      @click="$emit('hubungi-petugas')"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none"
        stroke="currentColor" stroke-width="2" stroke-linecap="round"
        stroke-linejoin="round" class="w-4 h-4"
      >
        <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z" />
      </svg>
      Hubungi petugas untuk bantuan
    </button>
  </section>
</template>
