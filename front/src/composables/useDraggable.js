// Универсальный drag+snap-to-edge композабл для плавающих виджетов
// (паттерн chat heads/Shimeji): свободное перетаскивание указателем, при
// отпускании — прилипание к ближайшему горизонтальному краю экрана,
// позиция персистится в localStorage. Чистые функции вынесены наружу —
// тестируются без монтирования компонента/DOM.
import { onBeforeUnmount, ref } from 'vue'
import { storageGetJSON, storageSetJSON } from '@/utils/storage.js'

// Зажимает {x,y} внутри вьюпорта с отступом margin; bottomInset — доп. запас
// снизу (мобильная нижняя навигация + safe-area).
export function clampToViewport(x, y, w, h, margin, vw, vh, bottomInset = 0) {
  const maxX = Math.max(margin, vw - w - margin)
  const maxY = Math.max(margin, vh - h - margin - bottomInset)
  return {
    x: Math.min(Math.max(x, margin), maxX),
    y: Math.min(Math.max(y, margin), maxY),
  }
}

// Ближайший горизонтальный край: сравниваем центр виджета с серединой экрана.
export function snapX(x, w, margin, vw) {
  const center = x + w / 2
  return center < vw / 2 ? margin : Math.max(margin, vw - w - margin)
}

export function cornerPosition(corner, w, h, margin, vw, vh, bottomInset = 0) {
  const [vert, horiz] = corner.split('-')
  const x = horiz === 'left' ? margin : Math.max(margin, vw - w - margin)
  const y = vert === 'top' ? margin : Math.max(margin, vh - h - margin - bottomInset)
  return { x, y }
}

// У правого края нижняя зона может быть занята другим плавающим элементом
// (FAB мини-хаба) — прижатый вправо виджет поднимаем выше этой зоны, чтобы
// они не перекрывали и не блокировали друг друга.
export function avoidRightBottom(pos, w, h, margin, vw, vh, bottomInset, reserve) {
  if (!reserve) return pos
  const rightEdge = Math.max(margin, vw - w - margin)
  if (pos.x < rightEdge - 1) return pos
  const maxY = vh - h - margin - bottomInset - reserve
  return pos.y > maxY ? { ...pos, y: Math.max(margin, maxY) } : pos
}

const DRAG_THRESHOLD = 4

/**
 * @param {object} opts
 * @param {string} opts.storageKey — ключ localStorage для персистентной позиции.
 * @param {{w:number,h:number}} opts.size — размер виджета в px.
 * @param {string} [opts.defaultCorner='bottom-left'] — угол по умолчанию.
 * @param {number} [opts.margin=16] — отступ от краёв экрана.
 * @param {number|Function} [opts.bottomInset=0] — доп. запас снизу (px или
 *   функция — пересчитывается на каждый clamp, переживает resize/поворот).
 * @param {number|Function} [opts.rightBottomReserve=0] — высота «запретной»
 *   зоны в правом нижнем углу (напр., под FAB мини-хаба).
 */
export function useDraggable({
  storageKey, size, defaultCorner = 'bottom-left', margin = 16,
  bottomInset = 0, rightBottomReserve = 0,
}) {
  const viewportSize = () => ({
    vw: typeof window !== 'undefined' ? window.innerWidth : 1024,
    vh: typeof window !== 'undefined' ? window.innerHeight : 768,
  })
  const resolvedBottomInset = () => (typeof bottomInset === 'function' ? bottomInset() : bottomInset)
  const resolvedReserve = () => (typeof rightBottomReserve === 'function' ? rightBottomReserve() : rightBottomReserve)

  function constrain(x, y) {
    const { vw, vh } = viewportSize()
    const bi = resolvedBottomInset()
    const p = clampToViewport(x, y, size.w, size.h, margin, vw, vh, bi)
    return avoidRightBottom(p, size.w, size.h, margin, vw, vh, bi, resolvedReserve())
  }

  function loadInitial() {
    const { vw, vh } = viewportSize()
    const saved = storageGetJSON(storageKey, null)
    if (saved && Number.isFinite(saved.x) && Number.isFinite(saved.y)) {
      return constrain(saved.x, saved.y)
    }
    const p = cornerPosition(defaultCorner, size.w, size.h, margin, vw, vh, resolvedBottomInset())
    return constrain(p.x, p.y)
  }

  const pos = ref(loadInitial())
  const dragging = ref(false)
  let dragged = false
  let start = { x: 0, y: 0, px: 0, py: 0 }
  let rafId = 0
  let lastMove = null

  function onPointerDown(e) {
    if (e.button != null && e.button !== 0) return
    dragging.value = true
    dragged = false
    start = { x: pos.value.x, y: pos.value.y, px: e.clientX, py: e.clientY }
    // Capture — быстрый палец за пределами элемента не теряет события.
    try { e.currentTarget?.setPointerCapture?.(e.pointerId) } catch { /* noop */ }
    window.addEventListener('pointermove', onPointerMove)
    window.addEventListener('pointerup', onPointerUp)
    window.addEventListener('pointercancel', onPointerUp)
  }

  // Позиция пишется раз в кадр (rAF): на тач/стилусе pointermove идёт чаще
  // кадра, каждое событие рендерить незачем.
  function applyMove() {
    rafId = 0
    if (!dragging.value || !lastMove) return
    const dx = lastMove.px - start.px
    const dy = lastMove.py - start.py
    if (!dragged && (Math.abs(dx) > DRAG_THRESHOLD || Math.abs(dy) > DRAG_THRESHOLD)) dragged = true
    pos.value = constrain(start.x + dx, start.y + dy)
  }

  function onPointerMove(e) {
    if (!dragging.value) return
    lastMove = { px: e.clientX, py: e.clientY }
    if (!rafId) rafId = requestAnimationFrame(applyMove)
  }

  function endDrag() {
    dragging.value = false
    if (rafId) { cancelAnimationFrame(rafId); rafId = 0; applyMoveSync() }
    lastMove = null
    window.removeEventListener('pointermove', onPointerMove)
    window.removeEventListener('pointerup', onPointerUp)
    window.removeEventListener('pointercancel', onPointerUp)
    if (dragged) {
      const { vw } = viewportSize()
      const snapped = { ...pos.value, x: snapX(pos.value.x, size.w, margin, vw) }
      pos.value = constrain(snapped.x, snapped.y)
    }
    storageSetJSON(storageKey, pos.value)
  }

  function applyMoveSync() {
    if (!lastMove) return
    const dx = lastMove.px - start.px
    const dy = lastMove.py - start.py
    if (!dragged && (Math.abs(dx) > DRAG_THRESHOLD || Math.abs(dy) > DRAG_THRESHOLD)) dragged = true
    pos.value = constrain(start.x + dx, start.y + dy)
  }

  function onPointerUp() {
    endDrag()
  }

  // Пересобрать позицию к ближайшему краю при повороте экрана/ресайзе.
  function onResize() {
    const { vw } = viewportSize()
    pos.value = constrain(snapX(pos.value.x, size.w, margin, vw), pos.value.y)
  }
  if (typeof window !== 'undefined') window.addEventListener('resize', onResize)

  onBeforeUnmount(() => {
    if (rafId) cancelAnimationFrame(rafId)
    window.removeEventListener('pointermove', onPointerMove)
    window.removeEventListener('pointerup', onPointerUp)
    window.removeEventListener('pointercancel', onPointerUp)
    window.removeEventListener('resize', onResize)
  })

  return {
    pos,
    dragging,
    onPointerDown,
    // После pointerup синхронно недоступно — компонент читает флаг в click,
    // который на pointer-устройствах всегда всплывает следом за pointerup.
    wasDragged: () => dragged,
  }
}
