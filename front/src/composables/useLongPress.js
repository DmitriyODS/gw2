/**
 * Долгое нажатие — контекстное меню на тач-экране (аналог ПКМ).
 *
 * Пальцем правой кнопки нет, а меню плитке и кнопке панели задач нужно; жест
 * один и тот же в двух местах, поэтому живёт общим composable'ом.
 * Смещение пальца больше порога считаем прокруткой и жест отменяем.
 */
import { ref } from 'vue'

const MOVE_TOLERANCE = 10

export function useLongPress(onLongPress, { delay = 480 } = {}) {
  let timer = null
  let origin = null
  const fired = ref(false)

  function cancel() {
    clearTimeout(timer)
    timer = null
    origin = null
  }

  function start(payload, e) {
    cancel()
    fired.value = false
    origin = { x: e.clientX, y: e.clientY }
    timer = setTimeout(() => {
      timer = null
      fired.value = true
      // Короткая отдача: пользователь понимает, что жест сработал, ещё до меню.
      navigator.vibrate?.(12)
      onLongPress(payload, e)
    }, delay)
  }

  function move(e) {
    if (!timer || !origin) return
    if (Math.hypot(e.clientX - origin.x, e.clientY - origin.y) > MOVE_TOLERANCE) cancel()
  }

  /** Клик после сработавшего жеста — уже отработанное действие: гасим его. */
  function consumed() {
    if (!fired.value) return false
    fired.value = false
    return true
  }

  return { start, move, cancel, consumed }
}
