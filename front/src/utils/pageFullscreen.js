/* Полноэкранный режим САМОГО БРАУЗЕРА (как F11) — не путать с развёрнутым
   окном рабочего стола (`desktop.fullscreen`): там окно занимает экран внутри
   страницы, здесь страница занимает монитор.

   Префиксы нужны Safari (webkit) и старым Edge (ms): вызов уходит первому
   методу, который есть. */

const el = () => document.documentElement

export function isPageFullscreen() {
  if (typeof document === 'undefined') return false
  return !!(document.fullscreenElement || document.webkitFullscreenElement || document.msFullscreenElement)
}

export function pageFullscreenSupported() {
  if (typeof document === 'undefined') return false
  const d = document
  const e = el()
  return !!(e.requestFullscreen || e.webkitRequestFullscreen || e.msRequestFullscreen)
    && !!(d.exitFullscreen || d.webkitExitFullscreen || d.msExitFullscreen)
}

/** Развернуть/свернуть. Промис — потому что requestFullscreen асинхронный. */
export function togglePageFullscreen() {
  try {
    if (isPageFullscreen()) {
      const exit = document.exitFullscreen || document.webkitExitFullscreen || document.msExitFullscreen
      return Promise.resolve(exit?.call(document))
    }
    const e = el()
    const request = e.requestFullscreen || e.webkitRequestFullscreen || e.msRequestFullscreen
    return Promise.resolve(request?.call(e))
  } catch {
    // Браузер может отказать (нет пользовательского жеста, политика iframe) —
    // это не повод ронять интерфейс: кнопка просто ничего не сделает.
    return Promise.resolve()
  }
}

/** Подписка на смену режима (F11 и Esc меняют его мимо нашей кнопки). */
export function onPageFullscreenChange(handler) {
  document.addEventListener('fullscreenchange', handler)
  document.addEventListener('webkitfullscreenchange', handler)
  return () => {
    document.removeEventListener('fullscreenchange', handler)
    document.removeEventListener('webkitfullscreenchange', handler)
  }
}
