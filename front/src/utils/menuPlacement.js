// Размещение плавающих меню/поповеров так, чтобы они не обрезались краем экрана
// и ужимались (со скроллом), если выше/шире вьюпорта. Возвращает готовые
// значения для inline-стиля: { left, top, width, maxHeight }.

// fitToViewport — меню у произвольной точки (курсор/тап). Прижимает верх-лево к
// (x, y), но не даёт вылезти за край; высота ограничивается вьюпортом.
export function fitToViewport(el, { x, y, pad = 8 } = {}) {
  const vw = window.innerWidth
  const vh = window.innerHeight
  const width = Math.min(el.offsetWidth, vw - 2 * pad)
  const maxHeight = vh - 2 * pad
  const height = Math.min(el.scrollHeight, maxHeight)

  let left = x
  let top = y
  if (left + width > vw - pad) left = vw - width - pad
  if (top + height > vh - pad) top = vh - height - pad
  left = Math.max(pad, left)
  top = Math.max(pad, top)
  return { left, top, width, maxHeight }
}

// placeByAnchor — меню, привязанное к элементу-триггеру (его rect). По умолчанию
// раскрывается ВНИЗ, но флипается ВВЕРХ, если снизу мало места; выбирается
// сторона с бóльшим запасом, высота ограничивается запасом этой стороны.
// align: 'right' — правый край меню к правому краю триггера; 'left' — наоборот.
export function placeByAnchor(el, rect, { gap = 6, pad = 8, align = 'right' } = {}) {
  const vw = window.innerWidth
  const vh = window.innerHeight
  const width = Math.min(el.offsetWidth, vw - 2 * pad)
  const natural = el.scrollHeight

  const spaceBelow = vh - rect.bottom - gap - pad
  const spaceAbove = rect.top - gap - pad

  let top
  let maxHeight
  if (natural <= spaceBelow || spaceBelow >= spaceAbove) {
    // Вниз (помещается либо снизу больше места).
    top = rect.bottom + gap
    maxHeight = spaceBelow
  } else {
    // Вверх.
    maxHeight = spaceAbove
    top = rect.top - gap - Math.min(natural, maxHeight)
  }

  let left = align === 'right' ? rect.right - width : rect.left
  left = Math.min(Math.max(pad, left), vw - width - pad)
  top = Math.max(pad, top)
  return { left, top, width, maxHeight: Math.max(0, maxHeight) }
}
