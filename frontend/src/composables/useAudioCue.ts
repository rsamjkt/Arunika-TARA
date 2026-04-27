// useAudioCue — synth audio cue (tap, success, error) via Web Audio API.
//
// Tidak butuh asset file — semua sound di-synthesize dari oscillator.
// Cocok untuk kiosk: ringan, no network, customizable per environment.
//
// Trigger lewat composable:
//   const audio = useAudioCue()
//   audio.tap()        → soft click ~50ms
//   audio.success()    → bell chime ~600ms (do-mi-sol)
//   audio.error()      → low buzz ~300ms
//
// Master switch + volume di-baca dari useBrandingStore.
import { useBrandingStore } from '../stores/branding'

let audioContext: AudioContext | null = null

function getContext(): AudioContext | null {
  if (audioContext) return audioContext
  try {
    const Ctx = window.AudioContext || (window as any).webkitAudioContext
    if (!Ctx) return null
    audioContext = new Ctx()
    return audioContext
  } catch {
    return null
  }
}

function playTone(opts: {
  freq: number
  durationMs: number
  type?: OscillatorType
  volume: number
  attack?: number
  decay?: number
}) {
  const ctx = getContext()
  if (!ctx) return
  const t = ctx.currentTime
  const osc = ctx.createOscillator()
  const gain = ctx.createGain()
  osc.type = opts.type ?? 'sine'
  osc.frequency.setValueAtTime(opts.freq, t)

  // Envelope: quick attack, smooth decay (no clicks)
  const attack = opts.attack ?? 0.005
  const decay = opts.decay ?? opts.durationMs / 1000
  gain.gain.setValueAtTime(0, t)
  gain.gain.linearRampToValueAtTime(opts.volume, t + attack)
  gain.gain.exponentialRampToValueAtTime(0.0001, t + attack + decay)

  osc.connect(gain).connect(ctx.destination)
  osc.start(t)
  osc.stop(t + attack + decay + 0.05)
}

function playSequence(notes: Array<{ freq: number; offsetMs: number; durMs: number }>, volume: number) {
  notes.forEach((n) => {
    setTimeout(() => playTone({
      freq: n.freq,
      durationMs: n.durMs,
      volume,
      type: 'sine',
    }), n.offsetMs)
  })
}

export function useAudioCue() {
  const branding = useBrandingStore()

  const enabled = () => branding.audioEnabled
  const vol = () => branding.audioVolume

  return {
    /** Soft click — tap response feedback */
    tap() {
      if (!enabled()) return
      playTone({ freq: 1200, durationMs: 40, volume: vol() * 0.3, attack: 0.002 })
    },

    /** Success chime — registrasi/SEP berhasil, tiket cetak */
    success() {
      if (!enabled()) return
      playSequence(
        [
          { freq: 523, offsetMs: 0, durMs: 150 },     // C5
          { freq: 659, offsetMs: 100, durMs: 150 },   // E5
          { freq: 784, offsetMs: 200, durMs: 250 },   // G5
        ],
        vol() * 0.5,
      )
    },

    /** Error tone — gagal, alert modal muncul */
    error() {
      if (!enabled()) return
      playTone({ freq: 220, durationMs: 250, volume: vol() * 0.4, type: 'sawtooth' })
    },

    /** Notification — booking confirmed, fingerprint detected, dll */
    notify() {
      if (!enabled()) return
      playTone({ freq: 880, durationMs: 120, volume: vol() * 0.4 })
    },
  }
}
