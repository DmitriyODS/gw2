// Яндекс.Метрика. Счётчик инициализируется в index.html с defer:true —
// просмотры страниц SPA (включая самый первый) отправляет router.afterEach.
const COUNTER_ID = 110869595

// window.ym отсутствует на dev-хостах (гард в index.html) и при блокировщиках.
export function trackPageView(url, referer) {
  if (typeof window.ym !== 'function') return
  window.ym(COUNTER_ID, 'hit', url, referer ? { referer } : undefined)
}
