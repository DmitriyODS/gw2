import { ref, onMounted, onBeforeUnmount } from 'vue'

/* Поведение мобильного FAB: прячется при прокрутке вниз, появляется при
 * прокрутке вверх или когда прокрутка остановилась. Слушает scroll в
 * capture-фазе на document — ловит прокрутку ЛЮБОГО внутреннего контейнера
 * (scroll-события не всплывают), без привязки к конкретному элементу. */
export function useFabOnScroll({ idleMs = 700 } = {}) {
  const fabVisible = ref(true)
  const lastTops = new WeakMap()
  let idleTimer = null

  function onScroll(e) {
    const el = e.target === document ? document.scrollingElement : e.target
    if (!el || !(el instanceof Element)) return
    const top = el.scrollTop
    const prev = lastTops.get(el) ?? top
    if (top > prev + 4) fabVisible.value = false
    else if (top < prev - 4) fabVisible.value = true
    lastTops.set(el, top)
    clearTimeout(idleTimer)
    idleTimer = setTimeout(() => { fabVisible.value = true }, idleMs)
  }

  onMounted(() => document.addEventListener('scroll', onScroll, { capture: true, passive: true }))
  onBeforeUnmount(() => {
    document.removeEventListener('scroll', onScroll, { capture: true })
    clearTimeout(idleTimer)
  })

  return { fabVisible }
}
