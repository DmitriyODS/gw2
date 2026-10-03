/**
 * Горячие клавиши раздела, а не всего приложения.
 *
 * Разделы живут окнами и остаются смонтированными в фоне, поэтому слушатель
 * `keydown` на window у каждого из них слышит ВСЁ: Delete в списке задач
 * удалял бы выделенное на доске в соседнем окне, Ctrl+Z — отменял бы её
 * правку. Клавиша считается своей, если последнее нажатие указателя или фокус
 * пришлись внутрь области (холст фокуса не берёт, поэтому одного фокуса мало).
 * Набор текста в полях ввода сочетаниями раздела не считается.
 *
 * @param {() => HTMLElement|null|undefined} getRoot — корень области.
 * @param {(e: KeyboardEvent) => void} handler
 */
import { onBeforeUnmount, onMounted } from 'vue'

export function isTypingTarget(el) {
  if (!el) return false
  return el.tagName === 'INPUT' || el.tagName === 'TEXTAREA' || el.tagName === 'SELECT' || !!el.isContentEditable
}

export function useScopedHotkeys(getRoot, handler) {
  let armed = false

  function inside(target) {
    const root = getRoot()
    return !!root && target instanceof Node && root.contains(target)
  }

  const onPointerDown = (e) => { armed = inside(e.target) }
  const onFocusIn = (e) => { armed = inside(e.target) }
  const onKeyDown = (e) => {
    if (!armed || isTypingTarget(e.target)) return
    if (e.target !== document.body && e.target !== document.documentElement && !inside(e.target)) return
    handler(e)
  }

  onMounted(() => {
    document.addEventListener('pointerdown', onPointerDown, true)
    document.addEventListener('focusin', onFocusIn, true)
    window.addEventListener('keydown', onKeyDown)
  })
  onBeforeUnmount(() => {
    document.removeEventListener('pointerdown', onPointerDown, true)
    document.removeEventListener('focusin', onFocusIn, true)
    window.removeEventListener('keydown', onKeyDown)
  })

  return {
    /** Считать область активной сразу (только что открытый раздел). */
    arm: () => { armed = true },
    /** Работают ли сейчас с этой областью (для событий вроде paste). */
    isActive: () => armed,
  }
}
