import { ref, watch, onBeforeUnmount } from 'vue'

/**
 * Подбор раскладки сетки плиток фиксированной пропорции (3:4) так, чтобы они
 * максимально заполняли контейнер, не растягиваясь и не обрезаясь. Возвращает
 * число колонок и ширину плитки в px (как в Meet/Zoom — перебор колонок,
 * выбор варианта с наибольшей плиткой). ResizeObserver следит за контейнером.
 *
 * @param containerRef ref на DOM-контейнер сетки
 * @param count        ref/computed с числом плиток
 * @param opts.aspect  отношение ширина/высота плитки (3/4 по умолчанию)
 * @param opts.gap     зазор между плитками в px
 */
export function useTileGrid(containerRef, count, opts = {}) {
  const { aspect = 3 / 4, gap = 8 } = opts
  const cols = ref(1)
  const tilePx = ref(0)
  let ro = null

  function recompute() {
    const el = containerRef.value
    const n = count.value || 0
    if (!el || n < 1) return
    const cw = el.clientWidth
    const ch = el.clientHeight
    if (cw <= 0 || ch <= 0) return

    let best = { c: 1, w: 0 }
    for (let c = 1; c <= n; c++) {
      const r = Math.ceil(n / c)
      let w = (cw - (c - 1) * gap) / c
      let h = w / aspect
      if (h * r + (r - 1) * gap > ch) {
        h = (ch - (r - 1) * gap) / r
        w = h * aspect
      }
      if (w > best.w) best = { c, w }
    }
    cols.value = best.c
    tilePx.value = Math.max(0, Math.floor(best.w))
  }

  watch(containerRef, (el) => {
    ro?.disconnect()
    ro = null
    if (el) {
      ro = new ResizeObserver(recompute)
      ro.observe(el)
      recompute()
    }
  }, { flush: 'post' })

  watch(count, recompute, { flush: 'post' })

  onBeforeUnmount(() => ro?.disconnect())

  return { cols, tilePx }
}
