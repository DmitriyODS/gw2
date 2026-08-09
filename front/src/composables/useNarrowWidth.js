import { onBeforeUnmount, onMounted, ref } from 'vue'

/**
 * «Тесно ли» блоку по СОБСТВЕННОМУ размеру.
 *
 * Раздел живёт окном рабочего стола, поэтому раскладку решает размер ПАНЕЛИ,
 * а не экрана — медиазапросы про неё ничего не знают. Две тонкости, ради
 * которых это общий хук, а не три строки на месте:
 *
 * 1. Меряем ВНЕШНИЙ размер (border-box). Тесная раскладка обычно сама меняет
 *    поля блока и его полосу прокрутки, а `contentRect` их вычитает — на
 *    границе получались вечные качели «тесно ↔ просторно».
 * 2. Порог с гистерезисом: обратно раскладка переключается только с запасом,
 *    поэтому дребезг в пару пикселей ничего не переключает.
 *
 * Нулевой размер пропускаем: так выглядит скрытый блок (свёрнутое окно), и
 * о раскладке он не говорит ничего.
 */
const HYSTERESIS = 24

function useTightSize(elRef, threshold, axis) {
  const tight = ref(false)
  const at = () => (typeof threshold === 'function' ? threshold() : threshold)
  let ro = null

  onMounted(() => {
    const el = elRef.value
    if (typeof ResizeObserver === 'undefined' || !el) {
      // Движок без ResizeObserver: судим по экрану — грубо, но лучше, чем ничего.
      if (typeof window === 'undefined') return
      tight.value = (axis === 'height' ? window.innerHeight : window.innerWidth) <= 768
      return
    }
    ro = new ResizeObserver(([entry]) => {
      const box = entry.borderBoxSize?.[0]
      const size = axis === 'height'
        ? box?.blockSize ?? entry.target.offsetHeight
        : box?.inlineSize ?? entry.target.offsetWidth
      if (!size) return
      const next = size < at() + (tight.value ? HYSTERESIS : 0)
      if (next !== tight.value) tight.value = next
    })
    ro.observe(el)
  })

  onBeforeUnmount(() => ro?.disconnect())

  return tight
}

/**
 * @param {import('vue').Ref<HTMLElement|null>} elRef — измеряемый блок.
 * @param {number|(() => number)} threshold — порог тесноты по ширине, px.
 * @returns {import('vue').Ref<boolean>}
 */
export function useNarrowWidth(elRef, threshold) {
  return useTightSize(elRef, threshold, 'width')
}

/**
 * То же по ВЫСОТЕ — для экранной клавиатуры: она срезает у панели половину
 * высоты, и раскладка, рассчитанная на полный экран, перестаёт быть пригодной.
 *
 * @param {import('vue').Ref<HTMLElement|null>} elRef — измеряемый блок.
 * @param {number|(() => number)} threshold — порог тесноты по высоте, px.
 * @returns {import('vue').Ref<boolean>}
 */
export function useShortHeight(elRef, threshold) {
  return useTightSize(elRef, threshold, 'height')
}
