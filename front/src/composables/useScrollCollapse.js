import { ref, onMounted, onBeforeUnmount } from 'vue'

/* Отслеживает скролл указанного элемента (через ref) и возвращает
   реактивный `isCompact`: true при заметном скролле вниз, false у верха
   или при скролле вверх. Используется для схлопывания шапки/FAB. */
export function useScrollCollapse(targetRef, {
  enterThreshold = 40,
  upDelta = 4,
  downDelta = 8,
  resetTop = 24,
} = {}) {
  const isCompact = ref(false)
  let lastScrollTop = 0
  let el = null

  function onScroll() {
    if (!el) return
    const st = el.scrollTop
    if (st > lastScrollTop + downDelta && st > enterThreshold) isCompact.value = true
    else if (st < lastScrollTop - upDelta || st < resetTop) isCompact.value = false
    lastScrollTop = st
  }

  onMounted(() => {
    el = targetRef.value
    if (!el) return
    el.addEventListener('scroll', onScroll, { passive: true })
  })

  onBeforeUnmount(() => {
    if (el) el.removeEventListener('scroll', onScroll)
  })

  return { isCompact }
}
