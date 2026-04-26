// useClock - reactive jam digital live untuk header kiosk.
//
// Update setiap detik (precision detik cukup untuk display jam digital).
// Auto-cleanup di onUnmounted.
import { onMounted, onUnmounted, ref } from 'vue'

const HARI = ['Min', 'Sen', 'Sel', 'Rab', 'Kam', 'Jum', 'Sab']
const BULAN = [
  'Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun',
  'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des',
]

const pad = (n: number) => String(n).padStart(2, '0')

function formatTime(d: Date): string {
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

function formatDate(d: Date): string {
  return `${HARI[d.getDay()]}, ${pad(d.getDate())} ${BULAN[d.getMonth()]} ${d.getFullYear()}`
}

export interface Clock {
  time: ReturnType<typeof ref<string>>
  date: ReturnType<typeof ref<string>>
}

export function useClock() {
  const now = new Date()
  const time = ref(formatTime(now))
  const date = ref(formatDate(now))

  let intervalID: ReturnType<typeof setInterval> | null = null

  onMounted(() => {
    intervalID = setInterval(() => {
      const d = new Date()
      time.value = formatTime(d)
      date.value = formatDate(d)
    }, 1000)
  })

  onUnmounted(() => {
    if (intervalID) clearInterval(intervalID)
  })

  return { time, date }
}
