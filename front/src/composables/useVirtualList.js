/**
 * Оконный рендеринг длинной ленты: в DOM только строки у экрана (плюс запас
 * overscan), остальное заменяют распорки по измеренным высотам.
 *
 * Высоты строк заранее неизвестны (текст, картинки, реакции), поэтому строка
 * рисуется с оценкой, а после показа её замеряет ResizeObserver. Замер строки
 * ВЫШЕ видимой области сдвигает прокрутку на ту же разницу — иначе лента
 * прыгала бы под пальцем. Браузерный scroll anchoring при этом надо выключить
 * (`overflow-anchor: none` у контейнера), иначе поправка сложится дважды.
 *
 * Режим «прилип к низу» (лента чата): пока человек внизу, рост строк не
 * сдвигает его вверх — прокрутка остаётся у последнего сообщения.
 *
 * @param {object} opts
 * @param {import('vue').Ref<HTMLElement|null>} opts.container — прокручиваемый элемент.
 * @param {import('vue').Ref<HTMLElement|null>} opts.list — обёртка строк внутри него.
 * @param {import('vue').Ref<Array<string|number>>} opts.keys — ключи строк по порядку.
 * @param {(index: number) => number} opts.estimate — оценка высоты незамеренной строки.
 * @param {number} [opts.overscan] — запас над и под экраном, px.
 */
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { HeightIndex } from '@/utils/heightIndex.js'

const STICK_EPS = 4

export function useVirtualList({ container, list, keys, estimate, overscan = 800 }) {
  const index = new HeightIndex(estimate)
  // Растёт при любом изменении высот или состава — по нему пересчитывается всё.
  const version = ref(0)
  const scrollTop = ref(0)
  const viewport = ref(0)
  const listTop = ref(0)
  let stick = false

  let keyIndex = new Map()
  const elKey = new WeakMap()
  const els = new Map()
  const refFns = new Map()

  watch(keys, (k) => {
    index.setKeys(k)
    index.prune()
    keyIndex = new Map(k.map((key, i) => [key, i]))
    for (const key of refFns.keys()) if (!keyIndex.has(key)) refFns.delete(key)
    version.value++
  }, { immediate: true })

  function viewTop() {
    return scrollTop.value - listTop.value
  }

  const start = computed(() => {
    version.value
    if (!keys.value.length) return 0
    return index.indexAt(Math.max(0, viewTop() - overscan))
  })

  const end = computed(() => {
    version.value
    const n = keys.value.length
    if (!n) return 0
    const bottom = viewTop() + (viewport.value || 1000) + overscan
    return Math.min(n, index.indexAt(Math.max(0, bottom)) + 1)
  })

  const total = computed(() => {
    version.value
    return index.total
  })

  /** Смещение строки от начала обёртки (реактивно). */
  function offsetOf(i) {
    version.value
    return index.offsetOf(i)
  }

  function readGeometry() {
    const c = container.value
    if (!c) return
    scrollTop.value = c.scrollTop
    viewport.value = c.clientHeight
    const l = list.value
    if (l) listTop.value = l.getBoundingClientRect().top - c.getBoundingClientRect().top + c.scrollTop
  }

  /** Звать из обработчика scroll контейнера. */
  function onScroll() {
    const c = container.value
    if (!c) return
    readGeometry()
    stick = c.scrollHeight - c.scrollTop - c.clientHeight < STICK_EPS
  }

  function onResize(entries) {
    const c = container.value
    let anchor = 0
    let changed = false
    const top = c ? c.scrollTop - listTop.value : 0
    for (const entry of entries) {
      const el = entry.target
      if (el === c) {
        viewport.value = c.clientHeight
        continue
      }
      const key = elKey.get(el)
      const i = keyIndex.get(key)
      if (i === undefined) continue
      const h = entry.borderBoxSize?.[0]?.blockSize ?? el.offsetHeight
      // Ноль — окно свёрнуто (display: none): прежний замер вернее.
      if (!h) continue
      const rowTop = index.offsetOf(i)
      const delta = index.set(key, h, i)
      if (!delta) continue
      changed = true
      if (rowTop < top) anchor += delta
    }
    if (!changed) return
    version.value++
    if (!c) return
    if (stick) c.scrollTop = c.scrollHeight
    else if (anchor) c.scrollTop += anchor
  }

  const ro = typeof ResizeObserver !== 'undefined' ? new ResizeObserver(onResize) : null

  watch(container, (c, prev) => {
    if (prev) ro?.unobserve(prev)
    if (c) {
      ro?.observe(c)
      readGeometry()
    }
  }, { immediate: true })
  // Обёртка появляется позже контейнера (лента загрузилась) — её отступ от
  // верха контейнера нужен расчёту видимых строк сразу, а не с первой прокрутки.
  watch(list, (l) => { if (l) readGeometry() }, { flush: 'post' })

  /** Функциональный ref строки: `:ref="measureRef(key)"` на корне строки. */
  function measureRef(key) {
    let fn = refFns.get(key)
    if (!fn) {
      fn = (target) => {
        // Корень компонента (ref на компоненте отдаёт его экземпляр).
        const el = target?.$el ?? target
        const prev = els.get(key)
        if (prev && prev !== el) {
          ro?.unobserve(prev)
          els.delete(key)
        }
        if (el instanceof Element) {
          els.set(key, el)
          elKey.set(el, key)
          ro?.observe(el)
        }
      }
      refFns.set(key, fn)
    }
    return fn
  }

  /** Прокрутить к строке; align — 'start' | 'center'. */
  function scrollToIndex(i, { align = 'start', offset = 0, behavior } = {}) {
    const c = container.value
    if (!c) return
    stick = false
    let top = listTop.value + index.offsetOf(i) + offset
    if (align === 'center') {
      top -= (c.clientHeight - index.heightOf(keys.value[i], i)) / 2
    }
    c.scrollTo({ top: Math.max(0, top), behavior })
    if (behavior !== 'smooth') readGeometry()
  }

  /** Вниз и остаться там, пока строки у низа замеряются. */
  async function scrollToBottom({ behavior } = {}) {
    const c = container.value
    if (!c) return
    stick = behavior !== 'smooth'
    c.scrollTo({ top: c.scrollHeight, behavior })
    readGeometry()
    if (!stick) return
    // Нижние строки только что пришли с оценкой: после их отрисовки высота
    // ленты уточнится, и низ надо догнать ещё раз.
    await nextTick()
    if (stick && container.value) container.value.scrollTop = container.value.scrollHeight
  }

  /** Строка с ключом отрисована? Если нет — прокрутить к ней и дождаться. */
  async function reveal(key, align = 'center') {
    const i = keyIndex.get(key)
    if (i === undefined) return null
    if (!els.has(key)) {
      scrollToIndex(i, { align })
      await nextTick()
    }
    return els.get(key) || null
  }

  onBeforeUnmount(() => ro?.disconnect())

  return {
    start, end, total, offsetOf, measureRef, onScroll,
    scrollToIndex, scrollToBottom, reveal, indexOfKey: (key) => keyIndex.get(key),
  }
}
