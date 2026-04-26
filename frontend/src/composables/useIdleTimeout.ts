// useIdleTimeout — auto-reset kiosk ke home setelah idle.
//
// Pakai di HomeScreen dan setiap screen flow registrasi. Listen
// mousemove/touchstart/keydown di window — setiap event reset timer.
//
// Dua callback:
//   onCountdown(secondsLeft) - dipanggil setiap detik di 10 detik terakhir
//   onTimeout()              - dipanggil saat timer habis
//
// Caller (component) bertanggung jawab menjalankan navigation/reset
// di onTimeout. Composable ini hanya track timing.
import { onMounted, onUnmounted, ref } from 'vue'

export interface IdleOptions {
  totalSeconds: number
  countdownThreshold: number // mulai countdown N detik sebelum timeout
  onCountdown?: (secondsLeft: number) => void
  onTimeout: () => void
}

export function useIdleTimeout(opts: IdleOptions) {
  const isCountingDown = ref(false)
  const secondsLeft = ref(opts.totalSeconds)

  let intervalID: ReturnType<typeof setInterval> | null = null
  let lastActivity = Date.now()

  const reset = () => {
    lastActivity = Date.now()
    if (isCountingDown.value) {
      isCountingDown.value = false
      secondsLeft.value = opts.totalSeconds
    }
  }

  const tick = () => {
    const elapsedMs = Date.now() - lastActivity
    const remainingSec = Math.max(0, opts.totalSeconds - Math.floor(elapsedMs / 1000))
    secondsLeft.value = remainingSec

    if (remainingSec === 0) {
      stop()
      opts.onTimeout()
      return
    }
    if (remainingSec <= opts.countdownThreshold) {
      if (!isCountingDown.value) {
        isCountingDown.value = true
      }
      opts.onCountdown?.(remainingSec)
    }
  }

  const events: Array<keyof WindowEventMap> = ['mousemove', 'touchstart', 'keydown', 'click']

  const start = () => {
    if (intervalID) return
    lastActivity = Date.now()
    intervalID = setInterval(tick, 1000)
    events.forEach((e) => window.addEventListener(e, reset, { passive: true }))
  }

  const stop = () => {
    if (intervalID) {
      clearInterval(intervalID)
      intervalID = null
    }
    events.forEach((e) => window.removeEventListener(e, reset))
  }

  onMounted(start)
  onUnmounted(stop)

  return { isCountingDown, secondsLeft, reset, stop, start }
}
