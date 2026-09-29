import { ref, computed, unref, watch, onMounted, onUnmounted } from 'vue'

/**
 * Живой счётчик времени от момента `start` (ISO-строка/Date; ref/getter/значение).
 * Тикает раз в секунду, только пока `start` задан и вкладка видима (возврат на
 * вкладку сразу пересчитывает); чистит интервал при размонтировании.
 * Отдаёт и развёрнутый текст («1 ч 45 мин 32 сек»), и компактные часы («01:45:32»).
 *
 * Тик перерисовывает всё, что читает результат, — поэтому в больших панелях
 * счётчик держит отдельный маленький компонент (`ElapsedClock`).
 */
export function useElapsed(start) {
  const tick = ref(0)
  let timer = null

  const getStart = () => (typeof start === 'function' ? start() : unref(start))

  function sync() {
    const run = !!getStart() && !document.hidden
    if (run && !timer) {
      tick.value++
      timer = setInterval(() => { tick.value++ }, 1000)
    } else if (!run && timer) {
      clearInterval(timer)
      timer = null
    }
  }

  watch(getStart, sync)
  onMounted(() => {
    sync()
    document.addEventListener('visibilitychange', sync)
  })
  onUnmounted(() => {
    clearInterval(timer)
    timer = null
    document.removeEventListener('visibilitychange', sync)
  })

  const seconds = computed(() => {
    tick.value // зависимость для пересчёта раз в секунду
    const s = getStart()
    if (!s) return 0
    return Math.max(0, Math.floor((Date.now() - new Date(s)) / 1000))
  })

  const display = computed(() => formatDuration(seconds.value))
  const clock = computed(() => formatClock(seconds.value))

  return { seconds, display, clock }
}

function formatDuration(total) {
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  const ss = String(s).padStart(2, '0')
  const mm = String(m).padStart(2, '0')
  if (h > 0) return `${h} ч ${mm} мин ${ss} сек`
  if (m > 0) return `${m} мин ${ss} сек`
  return `${s} сек`
}

function formatClock(total) {
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  return `${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}
