/**
 * Геометрия окон рабочего стола: чистые функции без DOM — прилипание к краям,
 * зажим в рабочей области, каскадная раскладка новых окон.
 *
 * area — рабочая область {x, y, w, h}: экран минус панель задач.
 * rect — прямоугольник окна {x, y, w, h} в координатах вьюпорта.
 */

// Толщина «липкой» полосы у края экрана и размер угловой зоны (четверти).
const EDGE = 6
const CORNER = 160

/**
 * Зона прилипания под указателем: 'max' (верх), 'left'/'right' (половины),
 * 'tl'/'tr'/'bl'/'br' (четверти) либо null.
 */
export function snapZoneAt(px, py, area) {
  const left = px <= area.x + EDGE
  const right = px >= area.x + area.w - EDGE
  const top = py <= area.y + EDGE
  const bottom = py >= area.y + area.h - EDGE
  const nearTop = py <= area.y + CORNER
  const nearBottom = py >= area.y + area.h - CORNER
  const nearLeft = px <= area.x + CORNER
  const nearRight = px >= area.x + area.w - CORNER

  if (left && nearTop) return 'tl'
  if (left && nearBottom) return 'bl'
  if (right && nearTop) return 'tr'
  if (right && nearBottom) return 'br'
  if (left) return 'left'
  if (right) return 'right'
  if (top && nearLeft) return 'tl'
  if (top && nearRight) return 'tr'
  if (top) return 'max'
  if (bottom && nearLeft) return 'bl'
  if (bottom && nearRight) return 'br'
  return null
}

/** Прямоугольник зоны прилипания в рабочей области. */
export function rectForZone(zone, area) {
  const halfW = Math.round(area.w / 2)
  const halfH = Math.round(area.h / 2)
  switch (zone) {
    case 'max': return { x: area.x, y: area.y, w: area.w, h: area.h }
    case 'left': return { x: area.x, y: area.y, w: halfW, h: area.h }
    case 'right': return { x: area.x + area.w - halfW, y: area.y, w: halfW, h: area.h }
    case 'tl': return { x: area.x, y: area.y, w: halfW, h: halfH }
    case 'tr': return { x: area.x + area.w - halfW, y: area.y, w: halfW, h: halfH }
    case 'bl': return { x: area.x, y: area.y + area.h - halfH, w: halfW, h: halfH }
    case 'br': return { x: area.x + area.w - halfW, y: area.y + area.h - halfH, w: halfW, h: halfH }
    default: return null
  }
}

/** Размер окна не больше рабочей области и не меньше минимального. */
export function clampSize(rect, area, min = { w: 360, h: 260 }) {
  return {
    ...rect,
    w: Math.max(min.w, Math.min(rect.w, area.w)),
    h: Math.max(min.h, Math.min(rect.h, area.h)),
  }
}

/**
 * Держит окно в пределах рабочей области: целиком по вертикали (заголовок не
 * должен уезжать под панель задач или за верх экрана) и хотя бы частью по
 * горизонтали — как в настольных ОС.
 */
export function clampPosition(rect, area, keepVisible = 120) {
  const maxX = area.x + area.w - keepVisible
  const minX = area.x - rect.w + keepVisible
  return {
    ...rect,
    x: Math.round(Math.min(Math.max(rect.x, minX), maxX)),
    y: Math.round(Math.min(Math.max(rect.y, area.y), area.y + area.h - 48)),
  }
}

/** Каскад новых окон по центру рабочей области со сдвигом. */
export function cascadeRect(index, size, area) {
  const base = clampSize({ x: 0, y: 0, ...size }, area)
  const step = 34
  const shift = (index % 6) * step
  const x = area.x + Math.max(0, Math.round((area.w - base.w) / 2) - 60) + shift
  const y = area.y + Math.max(0, Math.round((area.h - base.h) / 2) - 40) + shift
  return clampPosition({ ...base, x, y }, area)
}

/**
 * Пересчёт геометрии окна после изменения размеров экрана: сохраняет прижатые
 * зоны и держит «нормальные» окна в новой рабочей области.
 */
export function refitRect(rect, area, min) {
  return clampPosition(clampSize(rect, area, min), area)
}
