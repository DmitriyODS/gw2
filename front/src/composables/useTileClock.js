import { onBeforeUnmount, onMounted, ref } from 'vue'

/* Общие часы живых плиток: один интервал на всё приложение вместо таймера у
   каждой плитки (два-три десятка разнесённых во времени таймеров будили
   процессор непрерывно).

   Часы идут, только пока хоть одна плитка реально на экране: скрытый `v-show`
   стартовый экран телефона, свёрнутая группа, прокрученная за край плитка и
   скрытая вкладка их останавливают. Видимость меряет один общий
   IntersectionObserver — элемент с display:none для него не пересекается. */

export const TILE_PERIOD = 5000

const beat = ref(0)
const shownByEl = new WeakMap()
const visible = new Set()
let timer = null
let observer = null
let listening = false

function sync() {
  const run = visible.size > 0 && !document.hidden
  if (run && !timer) timer = setInterval(() => { beat.value++ }, TILE_PERIOD)
  else if (!run && timer) {
    clearInterval(timer)
    timer = null
  }
}

function ensureObserver() {
  if (observer) return observer
  observer = new IntersectionObserver((entries) => {
    for (const { target, isIntersecting } of entries) {
      const shown = shownByEl.get(target)
      if (!shown) continue
      shown.value = isIntersecting
      if (isIntersecting) visible.add(target)
      else visible.delete(target)
    }
    sync()
  })
  if (!listening) {
    listening = true
    document.addEventListener('visibilitychange', sync)
  }
  return observer
}

/**
 * @param {import('vue').Ref<HTMLElement|null>} elRef — корень плитки.
 * @returns {{ beat: import('vue').Ref<number>, shown: import('vue').Ref<boolean> }}
 *   `beat` растёт раз в TILE_PERIOD, пока идут часы; `shown` — плитка на экране.
 */
export function useTileClock(elRef) {
  const shown = ref(false)

  onMounted(() => {
    const el = elRef.value
    if (!el) return
    shownByEl.set(el, shown)
    ensureObserver().observe(el)
  })

  onBeforeUnmount(() => {
    const el = elRef.value
    if (!el || !observer) return
    observer.unobserve(el)
    shownByEl.delete(el)
    visible.delete(el)
    sync()
  })

  return { beat, shown }
}
