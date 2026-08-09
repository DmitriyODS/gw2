import { onBeforeUnmount, onMounted, ref } from 'vue'

/**
 * «Тесно ли» блоку по СОБСТВЕННОЙ ширине.
 *
 * Раздел живёт окном рабочего стола, поэтому раскладку решает ширина ПАНЕЛИ,
 * а не экрана — медиазапросы про неё ничего не знают. Две тонкости, ради
 * которых это общий хук, а не три строки на месте:
 *
 * 1. Меряем ВНЕШНЮЮ ширину (border-box). Тесная раскладка обычно сама меняет
 *    поля блока и его полосу прокрутки, а `contentRect` их вычитает — на
 *    границе получались вечные качели «тесно ↔ просторно».
 * 2. Порог с гистерезисом: обратно раскладка переключается только с запасом,
 *    поэтому дребезг в пару пикселей ничего не переключает.
 *
 * Нулевую ширину пропускаем: так выглядит скрытый блок (свёрнутое окно), и
 * о раскладке она не говорит ничего.
 *
 * @param {import('vue').Ref<HTMLElement|null>} elRef — измеряемый блок.
 * @param {number|(() => number)} threshold — порог тесноты в пикселях.
 * @returns {import('vue').Ref<boolean>}
 */
const HYSTERESIS = 24

export function useNarrowWidth(elRef, threshold) {
  const narrow = ref(false)
  const at = () => (typeof threshold === 'function' ? threshold() : threshold)
  let ro = null

  onMounted(() => {
    const el = elRef.value
    if (typeof ResizeObserver === 'undefined' || !el) {
      // Движок без ResizeObserver: судим по экрану — грубо, но лучше, чем ничего.
      narrow.value = typeof window !== 'undefined' && window.innerWidth <= 768
      return
    }
    ro = new ResizeObserver(([entry]) => {
      const width = entry.borderBoxSize?.[0]?.inlineSize ?? entry.target.offsetWidth
      if (!width) return
      const next = width < at() + (narrow.value ? HYSTERESIS : 0)
      if (next !== narrow.value) narrow.value = next
    })
    ro.observe(el)
  })

  onBeforeUnmount(() => ro?.disconnect())

  return narrow
}
