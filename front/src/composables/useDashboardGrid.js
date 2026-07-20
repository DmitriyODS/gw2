import { ref, reactive, watch, nextTick, onBeforeUnmount } from 'vue'

/**
 * Кастомизируемая дашборд-сетка карточек с masonry-упаковкой: перетаскивание
 * для смены мест, изменение ширины (снап к колонкам) и высоты (снап к шагу),
 * персист раскладки в localStorage.
 *
 * Раскладкой управляют:
 *   order    — порядок id карточек (через CSS `order` grid-элементов);
 *   spans    — ширина карточки в колонках (1..cols) через `grid-column: span N`;
 *   heights  — минимальная высота (px) при ресайзе по вертикали;
 *   rowSpans — число грид-строк по факту высоты (masonry, измеряется RO).
 *
 * Карточка в разметке должна нести `data-card-id` (для hit-теста при drag),
 * `:style="cardStyle(id)"` и `:ref="el => observeCard(id, el)"` (замер высоты).
 * Ручку перетаскивания и грипы ресайза родитель вешает на `startDrag(e, id)` /
 * `startResize(e, id, gridEl, dir)` (dir: 'x' | 'y' | 'xy') через pointerdown.
 *
 * @param storageKey ключ localStorage для раскладки
 * @param cols       число колонок сетки (снап-шаг ресайза)
 */
// Шаг «липкой» привязки высоты и минимальная высота карточки при ресайзе.
const HEIGHT_STEP = 24
const MIN_HEIGHT = 140

// Masonry-упаковка: высота каждой карточки квантуется в строки-спаны сетки.
// ROW_UNIT — высота строки грид-трека (синхронно с `grid-auto-rows` в CSS),
// VGAP — вертикальный зазор между карточками (закладывается в спан, т.к. в CSS
// row-gap = 0). Мелкий ROW_UNIT ⇒ плотная упаковка колонок независимо друг
// от друга: высокая карточка слева не растягивает строку, справа встают стопкой.
const ROW_UNIT = 4
const VGAP = 20

export function useDashboardGrid({ storageKey, cols = 4 }) {
  const order = ref([])
  const spans = reactive({})
  const heights = reactive({}) // id → минимальная высота в px (не задано ⇒ авто по контенту)
  const rowSpans = reactive({}) // id → число грид-строк (измеряется по факту)
  const dragId = ref(null)
  const resizeId = ref(null)
  const dragOffset = reactive({ x: 0, y: 0 })

  const cardEls = new Map()
  let ro = null

  let defaults = []

  const clamp = (s) => Math.min(cols, Math.max(1, Math.round(Number(s)) || 1))

  // Синхронизация с реально присутствующими карточками (часть — условные):
  // сохраняем пользовательский порядок/ширины/высоты, дописываем новые, чистим ушедшие.
  function sync(items) {
    let saved = null
    try { saved = JSON.parse(localStorage.getItem(storageKey) || 'null') } catch { /* noop */ }
    const ids = items.map((i) => i.id)
    const savedOrder = Array.isArray(saved?.order) ? saved.order.filter((id) => ids.includes(id)) : []
    order.value = [...savedOrder, ...ids.filter((id) => !savedOrder.includes(id))]
    for (const it of items) {
      if (spans[it.id] == null) spans[it.id] = clamp(saved?.spans?.[it.id] ?? it.span ?? cols)
      const h = saved?.heights?.[it.id]
      if (h != null && heights[it.id] == null) heights[it.id] = Math.max(MIN_HEIGHT, Math.round(h))
    }
    for (const key of Object.keys(spans)) {
      if (!ids.includes(key)) delete spans[key]
    }
    for (const key of Object.keys(heights)) {
      if (!ids.includes(key)) delete heights[key]
    }
    defaults = items.map((i) => ({ id: i.id, span: clamp(i.span ?? cols) }))
  }

  function persist() {
    try {
      localStorage.setItem(storageKey, JSON.stringify({
        order: order.value,
        spans: { ...spans },
        heights: { ...heights },
      }))
    } catch { /* noop */ }
  }
  watch([order, spans, heights], persist, { deep: true })

  function reset() {
    order.value = defaults.map((d) => d.id)
    for (const d of defaults) spans[d.id] = d.span
    for (const key of Object.keys(heights)) delete heights[key] // высота — снова авто
  }

  // Блокировка выделения текста на время drag/resize (иначе pointer-таскание
  // выделяет текст под курсором).
  function lockSelect(lock) {
    const s = document.body?.style
    if (!s) return
    s.userSelect = lock ? 'none' : ''
    s.webkitUserSelect = lock ? 'none' : ''
  }

  // FLIP-анимация: плавно перегоняем элемент из прежнего положения (first) в
  // текущее. Транзишн снимается по завершении, чтобы не мешать drag-следованию.
  function flipMove(el, first) {
    const last = el.getBoundingClientRect()
    const dx = first.left - last.left
    const dy = first.top - last.top
    if (!dx && !dy) return
    el.style.transition = 'none'
    el.style.transform = `translate(${dx}px, ${dy}px)`
    void el.offsetWidth // форс-reflow: зафиксировать стартовую точку
    requestAnimationFrame(() => {
      el.style.transition = 'transform 0.24s cubic-bezier(0.2, 0, 0, 1)'
      el.style.transform = ''
      const done = () => {
        el.style.transition = ''
        el.removeEventListener('transitionend', done)
      }
      el.addEventListener('transitionend', done)
    })
  }
  function captureRects(exceptId) {
    const map = new Map()
    for (const [id, el] of cardEls) {
      if (id !== exceptId) map.set(id, el.getBoundingClientRect())
    }
    return map
  }
  function animateOthers(first) {
    nextTick(() => {
      for (const [id, el] of cardEls) {
        if (id === dragId.value) continue
        const f = first.get(id)
        if (f) flipMove(el, f)
      }
    })
  }

  // ── Перетаскивание (смена мест) ──────────────────────────────────
  // Карточка «плавает» под курсором (translate от своего слота). После каждой
  // перестановки слот пересчитывается, чтобы карточка ОСТАЛАСЬ под курсором —
  // без рывков к сетке. Антидребезг от «мигания» на границе блоков: с одной
  // целью — одна перестановка (lastOverId) + короткий кулдаун.
  let grab = { x: 0, y: 0 }        // смещение курсора внутри карточки при захвате
  let slot = { x: 0, y: 0 }        // позиция слота карточки (без transform)
  let lastPointer = { x: 0, y: 0 }
  let lastOverId = null
  let lastSwapAt = 0

  function positionDragged(px, py) {
    dragOffset.x = px - grab.x - slot.x
    dragOffset.y = py - grab.y - slot.y
  }

  function startDrag(e, id) {
    if (e.button != null && e.button !== 0) return
    e.preventDefault()
    // Сброс остатков незавершённой FLIP-анимации — иначе следование за
    // курсором унаследует transition и будет «запаздывать».
    const el = cardEls.get(id)
    if (el) { el.style.transition = ''; el.style.transform = '' }
    const r = el ? el.getBoundingClientRect() : { left: e.clientX, top: e.clientY }
    slot = { x: r.left, y: r.top }
    grab = { x: e.clientX - r.left, y: e.clientY - r.top }
    lastPointer = { x: e.clientX, y: e.clientY }
    lastOverId = null
    lastSwapAt = 0
    dragId.value = id
    dragOffset.x = 0
    dragOffset.y = 0
    lockSelect(true)
    window.addEventListener('pointermove', onDragMove)
    window.addEventListener('pointerup', endDrag, { once: true })
  }
  function onDragMove(e) {
    if (!dragId.value) return
    lastPointer = { x: e.clientX, y: e.clientY }
    positionDragged(e.clientX, e.clientY)

    // Перетаскиваемая карточка прозрачна для указателя (см. cardStyle) —
    // elementFromPoint вернёт карточку под ней.
    const target = document.elementFromPoint(e.clientX, e.clientY)?.closest?.('[data-card-id]')
    const tid = target?.getAttribute('data-card-id')
    if (!tid || tid === dragId.value) { lastOverId = null; return }
    if (tid === lastOverId) return                 // с этой целью уже переставились
    const now = performance.now()
    if (now - lastSwapAt < 90) return              // кулдаун против дребезга

    const arr = order.value.slice()
    const from = arr.indexOf(dragId.value)
    const to = arr.indexOf(tid)
    if (from === -1 || to === -1) return

    lastOverId = tid
    lastSwapAt = now
    const first = captureRects(dragId.value)       // позиции соседей ДО перестановки
    arr.splice(from, 1)
    arr.splice(to, 0, dragId.value)
    order.value = arr
    animateOthers(first)                           // соседи плавно расступаются

    // Слот перетаскиваемой сместился — пересчитаем, чтобы она осталась под
    // курсором (translate теперь = 0-смещение; вычитаем текущий transform).
    const el = cardEls.get(dragId.value)
    const offX = dragOffset.x
    const offY = dragOffset.y
    nextTick(() => {
      if (!el || dragId.value == null) return
      const rr = el.getBoundingClientRect()
      slot = { x: rr.left - offX, y: rr.top - offY }
      positionDragged(lastPointer.x, lastPointer.y)
    })
  }
  function endDrag() {
    const id = dragId.value
    const el = id ? cardEls.get(id) : null
    const first = el ? el.getBoundingClientRect() : null
    dragId.value = null
    dragOffset.x = 0
    dragOffset.y = 0
    lastOverId = null
    lockSelect(false)
    window.removeEventListener('pointermove', onDragMove)
    // Плавно «опускаем» карточку из-под курсора в её слот.
    if (el && first) nextTick(() => flipMove(el, first))
  }

  // ── Изменение размера: ширина — снап к колонкам, высота — к шагу ──
  // dir: 'x' — только ширина, 'y' — только высота, 'xy' — обе (угол).
  let resizeCtx = null
  function startResize(e, id, gridEl, dir = 'xy') {
    if (e.button != null && e.button !== 0) return
    e.stopPropagation()
    e.preventDefault()
    if (!gridEl) return
    const rect = gridEl.getBoundingClientRect()
    const gap = parseFloat(getComputedStyle(gridEl).columnGap) || 0
    const unit = (rect.width - gap * (cols - 1)) / cols + gap
    const card = e.target?.closest?.('[data-card-id]')
    resizeId.value = id
    resizeCtx = {
      startX: e.clientX,
      startY: e.clientY,
      startSpan: spans[id],
      startHeight: heights[id] ?? (card ? card.offsetHeight : MIN_HEIGHT),
      unit,
      dir,
    }
    lockSelect(true)
    window.addEventListener('pointermove', onResizeMove)
    window.addEventListener('pointerup', endResize, { once: true })
  }
  function onResizeMove(e) {
    if (!resizeCtx) return
    if (resizeCtx.dir !== 'y') {
      const dCols = Math.round((e.clientX - resizeCtx.startX) / resizeCtx.unit)
      spans[resizeId.value] = clamp(resizeCtx.startSpan + dCols)
    }
    if (resizeCtx.dir !== 'x') {
      const rawH = resizeCtx.startHeight + (e.clientY - resizeCtx.startY)
      heights[resizeId.value] = Math.max(MIN_HEIGHT, Math.round(rawH / HEIGHT_STEP) * HEIGHT_STEP)
    }
  }
  function endResize() {
    resizeId.value = null
    resizeCtx = null
    lockSelect(false)
    window.removeEventListener('pointermove', onResizeMove)
  }

  // ── Masonry: измерение фактической высоты карточки → число грид-строк ──
  function computeRowSpan(id) {
    const el = cardEls.get(id)
    if (!el) return
    const span = Math.max(1, Math.ceil((el.offsetHeight + VGAP) / ROW_UNIT))
    if (rowSpans[id] !== span) rowSpans[id] = span
  }
  // Ref-колбэк карточки: подписываемся на изменения её размера. Смена ширины
  // (span) или высоты (min-height) меняет offsetHeight ⇒ пересчёт строк-спана.
  // Установка grid-row высоту контента не трогает (align-items:start) — петли нет.
  function observeCard(id, el) {
    const prev = cardEls.get(id)
    if (el && prev === el) return
    if (prev && ro) ro.unobserve(prev)
    if (el) {
      el.__cardId = id
      cardEls.set(id, el)
      if (!ro) {
        ro = new ResizeObserver((entries) => {
          for (const en of entries) if (en.target.__cardId) computeRowSpan(en.target.__cardId)
        })
      }
      ro.observe(el)
      computeRowSpan(id) // сразу, чтобы не мигало наложением до первого RO-колбэка
    } else {
      cardEls.delete(id)
    }
  }

  function cardStyle(id) {
    const s = { order: order.value.indexOf(id), gridColumn: `span ${spans[id] || 1}` }
    if (rowSpans[id]) s.gridRow = `span ${rowSpans[id]}`
    // Высота — floor через min-height: контент не обрезается, а карточка растёт
    // (offsetHeight ↑ ⇒ строк-спан ↑), соседи по колонке пакуются ниже.
    if (heights[id] != null) s.minHeight = `${heights[id]}px`
    if (dragId.value === id) {
      s.transform = `translate(${dragOffset.x}px, ${dragOffset.y}px)`
      s.zIndex = 40
      s.pointerEvents = 'none'
    }
    return s
  }

  onBeforeUnmount(() => {
    window.removeEventListener('pointermove', onDragMove)
    window.removeEventListener('pointermove', onResizeMove)
    ro?.disconnect()
  })

  return { order, spans, dragId, resizeId, sync, reset, startDrag, startResize, observeCard, cardStyle, cols }
}
