import { ref } from 'vue'

const MOBILE_BP = 768

/* Один общий matchMedia на всё приложение: раньше у каждого потребителя был
   свой слушатель resize, и на телефоне каждое появление клавиатуры прогоняло
   их все. Медиазапрос срабатывает только при пересечении порога. */
const mq = typeof window !== 'undefined' && typeof window.matchMedia === 'function'
  ? window.matchMedia(`(max-width: ${MOBILE_BP}px)`)
  : null
const isMobile = ref(mq ? mq.matches : false)

if (mq?.addEventListener) mq.addEventListener('change', (e) => { isMobile.value = e.matches })
else mq?.addListener?.((e) => { isMobile.value = e.matches }) // старый Safari

export function useBreakpoint() {
  return { isMobile }
}
