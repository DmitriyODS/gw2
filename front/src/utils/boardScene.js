/* Сцена доски: модель объектов холста и геометрия. Рендер живёт рядом, в
   utils/boardRender.js — здесь только то, что описывает содержимое.
   Зеркало back-go/board/internal/domain/scene.go — сервер по этим же полям
   считает текст надписей для поиска и строит SVG при выгрузке.

   Цвет объекта хранится КЛЮЧОМ палитры («ink», «red»…), а не значением: canvas
   не понимает var(--tag-*), поэтому ключ разворачивается в реальный цвет через
   getComputedStyle в момент рисования. Так доска остаётся в теме приложения и
   сама перекрашивается в тёмном режиме. ИСКЛЮЧЕНИЕ — произвольный цвет из
   пипетки и палитры-круга: он хранится значением «#rrggbb» и теме не следует
   (человек выбрал именно этот оттенок, а не роль в оформлении).

   Устройство сцены — по образцу Figma: плоский список объектов, где связи
   выражены МЕТКАМИ, а не вложенностью (`layer`, `frame`, `group`, `bool`).
   Вложенные контейнеры пришлось бы обходить рекурсивно в каждом месте — от
   выделения до выгрузки, — а меткой набор собирается одним фильтром. */
import { normalizeEffects, normalizeGradient } from '@/utils/boardPaint.js'
import { moveNodes, nodesOf, outlinePoints, scaleNodes, vectorOutline } from '@/utils/boardVector.js'

export const OBJ = {
  path: 'path',
  vector: 'vector',
  comment: 'comment',
  line: 'line',
  arrow: 'arrow',
  rect: 'rect',
  ellipse: 'ellipse',
  diamond: 'diamond',
  polygon: 'polygon',
  text: 'text',
  sticky: 'sticky',
  image: 'image',
}

// Слой по умолчанию: сцены первой версии не знали про слои, поэтому весь их
// холст переезжает сюда.
export const BASE_LAYER = 'base'

export function newLayer(name = 'Слой') {
  return {
    id: newId(), name, visible: true, locked: false,
    opacity: 1, blend: 'normal', clip: false, mask: null, effects: null,
  }
}

/* Палитра доски: ключ → CSS-переменная. Восемь цветов тегов, «чернила» и белый.
   Белый задан ЗНАЧЕНИЕМ, а не токеном: это цвет краски (нужен белым и в тёмной
   теме — им рисуют по тёмному фону и по картинкам), а не роль в оформлении. */
export const SCENE_COLORS = [
  { key: 'ink', label: 'Чернила', token: '--color-text' },
  { key: 'white', label: 'Белый', value: '#ffffff' },
  { key: 'red', label: 'Красный', token: '--tag-red-accent' },
  { key: 'orange', label: 'Оранжевый', token: '--tag-orange-accent' },
  { key: 'amber', label: 'Янтарный', token: '--tag-amber-accent' },
  { key: 'green', label: 'Зелёный', token: '--tag-green-accent' },
  { key: 'teal', label: 'Бирюзовый', token: '--tag-teal-accent' },
  { key: 'blue', label: 'Синий', token: '--tag-blue-accent' },
  { key: 'violet', label: 'Фиолетовый', token: '--tag-violet-accent' },
  { key: 'pink', label: 'Розовый', token: '--tag-pink-accent' },
]

const COLOR_TOKENS = Object.fromEntries(SCENE_COLORS.filter((c) => c.token).map((c) => [c.key, c.token]))
const COLOR_VALUES = Object.fromEntries(SCENE_COLORS.filter((c) => c.value).map((c) => [c.key, c.value]))

/* Палитра ФАЙЛОВ: зеркало domain.SceneColors (сервер теми же значениями рисует
   SVG). Тема приложения тут ни при чём — файл уезжает наружу, а «чернила» в
   тёмной теме светлые, и растр на белом листе выходил почти пустым. */
export const EXPORT_COLORS = {
  ink: '#1f2430',
  white: '#ffffff',
  chalk: '#f8fafc',
  red: '#e05252',
  orange: '#e07a3c',
  amber: '#d9a520',
  green: '#3fa45b',
  teal: '#2b9b9b',
  blue: '#3b74d6',
  violet: '#7a5cd6',
  pink: '#d65c9b',
  __ph: '#c7ccd6', // рамка не загрузившейся картинки
}

/* Режимы наложения слоя: значения canvas globalCompositeOperation и CSS
   mix-blend-mode совпадают, поэтому один список годится и холсту, и SVG. */
export const BLEND_MODES = [
  { key: 'normal', label: 'Обычный' },
  { key: 'multiply', label: 'Умножение' },
  { key: 'screen', label: 'Экран' },
  { key: 'overlay', label: 'Перекрытие' },
  { key: 'darken', label: 'Затемнение' },
  { key: 'lighten', label: 'Осветление' },
  { key: 'color-dodge', label: 'Осветление основы' },
  { key: 'color-burn', label: 'Затемнение основы' },
  { key: 'hard-light', label: 'Жёсткий свет' },
  { key: 'soft-light', label: 'Мягкий свет' },
  { key: 'difference', label: 'Разница' },
  { key: 'exclusion', label: 'Исключение' },
  { key: 'hue', label: 'Цветовой тон' },
  { key: 'saturation', label: 'Насыщенность' },
  { key: 'color', label: 'Цветность' },
  { key: 'luminosity', label: 'Яркость' },
]

const BLEND_KEYS = new Set(BLEND_MODES.map((b) => b.key))

/* Булевы операции над фигурами — как в Figma: участники группы остаются
   отдельными объектами и правятся дальше, а вместе дают одну форму. Операция
   считается КОМПОЗИТОМ при отрисовке: пересечение произвольных контуров иначе
   требует своей геометрической библиотеки, а результат обязан совпадать с тем,
   что видно на холсте. */
export const BOOL_OPS = [
  { key: 'union', label: 'Объединить', icon: 'join_full', composite: 'source-over' },
  { key: 'subtract', label: 'Вычесть', icon: 'join_inner', composite: 'destination-out' },
  { key: 'intersect', label: 'Пересечь', icon: 'join_left', composite: 'destination-in' },
  { key: 'exclude', label: 'Исключить', icon: 'join_right', composite: 'xor' },
]

export const BOOL_COMPOSITE = Object.fromEntries(BOOL_OPS.map((b) => [b.key, b.composite]))
const BOOL_KEYS = new Set(BOOL_OPS.map((b) => b.key))

// Толщины пера, размеры ластика и надписей — общий словарь для тулбара и рендера.
export const STROKE_WIDTHS = [2, 4, 8, 16]
export const ERASER_SIZES = [8, 16, 32, 64, 128]
export const TEXT_SIZES = [14, 18, 24, 32, 48]

export const BACKGROUNDS = [
  { key: 'grid', label: 'Сетка' },
  { key: 'dots', label: 'Точки' },
  { key: 'plain', label: 'Чистый лист' },
]

// Кадры анимации: границы частоты и предел числа кадров (сцена целиком уезжает
// в одном JSON, поэтому потолок нужен — иначе сохранение раздувается).
export const FPS_RANGE = { min: 1, max: 60, def: 12 }
export const MAX_FRAMES = 240

/** Произвольный цвет — «#rgb»/«#rrggbb», а не ключ палитры. */
export function isHexColor(value) {
  return typeof value === 'string' && /^#([0-9a-f]{3}|[0-9a-f]{6})$/i.test(value)
}

// resolveColor — значение токена палитры; кэш сбрасывается при смене темы.
let colorCache = new Map()
let colorCacheKey = ''

export function resolveColor(key, fallbackToken = '--color-text') {
  if (isHexColor(key)) return key
  if (COLOR_VALUES[key]) return COLOR_VALUES[key]
  if (typeof window === 'undefined') return '#000'
  const root = document.documentElement
  const themeKey = `${root.dataset.dark || ''}|${root.dataset.theme || ''}`
  if (themeKey !== colorCacheKey) {
    colorCache = new Map()
    colorCacheKey = themeKey
  }
  if (colorCache.has(key)) return colorCache.get(key)
  const token = COLOR_TOKENS[key] || fallbackToken
  const value = getComputedStyle(root).getPropertyValue(token).trim() || '#000'
  colorCache.set(key, value)
  return value
}

/** Сброс кэша цветов — зовётся при переключении темы. */
export function invalidateColors() {
  colorCache = new Map()
  colorCacheKey = ''
}

export function emptyScene() {
  return {
    version: 4,
    background: 'grid',
    layers: [{ id: BASE_LAYER, name: 'Слой 1', visible: true, locked: false, opacity: 1, blend: 'normal', clip: false, mask: null, effects: null }],
    objects: [],
    animation: null,
  }
}

function normalizeMask(mask) {
  if (!mask || !Array.isArray(mask.points) || mask.points.length < 6) return null
  return { points: mask.points.map(Number), invert: !!mask.invert }
}

function clamp01(value, fallback = 1) {
  const n = Number(value)
  if (!Number.isFinite(n)) return fallback
  return Math.min(1, Math.max(0, n))
}

/** Нормализация анимации: пустой список кадров означает «доска не анимирована». */
function normalizeAnimation(raw) {
  const frames = Array.isArray(raw?.frames) ? raw.frames.filter((f) => f && f.id) : []
  if (!frames.length) return null
  const fps = Math.min(FPS_RANGE.max, Math.max(FPS_RANGE.min, Math.round(Number(raw.fps) || FPS_RANGE.def)))
  return {
    fps,
    frames: frames.slice(0, MAX_FRAMES).map((f, i) => ({ id: String(f.id), name: f.name || `Кадр ${i + 1}` })),
  }
}

/* Нормализация объекта: приводим к порядку то, от чего зависит отрисовка.
   Полей, которых нет, НЕ дописываем: сцена уезжает на сервер целиком, и лишние
   ключи у каждого объекта заметно её раздувают. */
function normalizeObject(o, { layer, frame, boolCounts }) {
  const next = { ...o, layer }
  if (frame) next.frame = frame
  else delete next.frame

  const gradient = normalizeGradient(o.gradient)
  if (gradient) next.gradient = gradient
  else delete next.gradient

  const effects = normalizeEffects(o.effects)
  if (effects) next.effects = effects
  else delete next.effects

  // Булева группа из одного участника смысла не имеет — метку снимаем.
  if (o.bool && boolCounts.get(o.bool) > 1) {
    next.bool = String(o.bool)
    next.boolOp = BOOL_KEYS.has(o.boolOp) ? o.boolOp : 'union'
  } else {
    delete next.bool
    delete next.boolOp
  }

  if (o.hidden) next.hidden = true
  else delete next.hidden
  if (o.locked) next.locked = true
  else delete next.locked
  if (o.maskObject) next.maskObject = true
  else delete next.maskObject

  if (o.type === OBJ.vector) next.nodes = nodesOf(o)
  return next
}

/** Нормализация сцены с сервера (или из файла): не доверяем структуре.
    Заодно поднимает сцены прежних версий: добавляет базовый слой, приписывает
    к нему объекты без слоя и достраивает свойства слоя (прозрачность,
    наложение, обтравка, маска, эффекты), которых прежние сцены не знали. */
export function normalizeScene(raw) {
  const scene = raw && typeof raw === 'object' ? raw : {}
  const objects = Array.isArray(scene.objects) ? scene.objects.filter((o) => o && o.type) : []

  let layers = Array.isArray(scene.layers) ? scene.layers.filter((l) => l && l.id) : []
  if (!layers.length) layers = [{ id: BASE_LAYER, name: 'Слой 1' }]
  layers = layers.map((l) => ({
    id: String(l.id),
    name: l.name || 'Слой',
    visible: l.visible !== false,
    locked: !!l.locked,
    opacity: clamp01(l.opacity),
    blend: BLEND_KEYS.has(l.blend) ? l.blend : 'normal',
    clip: !!l.clip,
    mask: normalizeMask(l.mask),
    effects: normalizeEffects(l.effects),
  }))
  // Нижний слой обтравлять не по чему — иначе он исчез бы целиком.
  if (layers[0].clip) layers[0] = { ...layers[0], clip: false }

  const animation = normalizeAnimation(scene.animation)
  const frames = new Set((animation?.frames || []).map((f) => f.id))
  const known = new Set(layers.map((l) => l.id))
  const fallback = layers[0].id

  const boolCounts = new Map()
  for (const o of objects) {
    if (o.bool) boolCounts.set(o.bool, (boolCounts.get(o.bool) || 0) + 1)
  }

  return {
    version: 4,
    background: BACKGROUNDS.some((b) => b.key === scene.background) ? scene.background : 'grid',
    layers,
    // Объект с чужим кадром (кадр удалили) становится общим для всей сцены,
    // а не пропадает: терять нарисованное нельзя.
    objects: objects.map((o) => normalizeObject(o, {
      layer: known.has(o.layer) ? o.layer : fallback,
      frame: o.frame && frames.has(o.frame) ? o.frame : undefined,
      boolCounts,
    })),
    animation,
  }
}

/** Объекты в порядке отрисовки: снизу вверх по слоям. onlyVisible — для рендера
    (скрытые слои и скрытые объекты пропускаются). frame — кадр анимации:
    объекты чужих кадров пропускаются, объекты без кадра видны всегда. */
export function orderedObjects(scene, { onlyVisible = true, frame = null } = {}) {
  const s = normalizeScene(scene)
  const out = []
  for (const layer of s.layers) {
    if (onlyVisible && !layer.visible) continue
    for (const o of s.objects) {
      if (o.layer !== layer.id) continue
      if (onlyVisible && o.hidden) continue
      if (frame && o.frame && o.frame !== frame) continue
      out.push(o)
    }
  }
  return out
}

/** Объекты кадра (включая общие для всей сцены) — таймлайн и экспорт видео. */
export function objectsForFrame(scene, frame) {
  return normalizeScene(scene).objects.filter((o) => !frame || !o.frame || o.frame === frame)
}

/** Слои, доступные для правки: видимые и незаблокированные. */
export function editableLayerIds(scene) {
  return new Set(normalizeScene(scene).layers.filter((l) => l.visible && !l.locked).map((l) => l.id))
}

/** Можно ли трогать объект: его слой доступен, сам он не скрыт и не заперт. */
export function isObjectEditable(o, editableLayers) {
  return editableLayers.has(o.layer) && !o.hidden && !o.locked
}

let seq = 0
export function newId() {
  seq += 1
  return `o${Date.now().toString(36)}${seq.toString(36)}`
}

// COMMENT_PIN — диаметр булавки комментария в единицах сцены.
export const COMMENT_PIN = 28

function pointsBounds(pts, o) {
  if (pts.length < 2) return { x: o.x || 0, y: o.y || 0, w: 0, h: 0 }
  let minX = Infinity; let minY = Infinity; let maxX = -Infinity; let maxY = -Infinity
  for (let i = 0; i + 1 < pts.length; i += 2) {
    minX = Math.min(minX, pts[i]); maxX = Math.max(maxX, pts[i])
    minY = Math.min(minY, pts[i + 1]); maxY = Math.max(maxY, pts[i + 1])
  }
  return { x: minX, y: minY, w: maxX - minX, h: maxY - minY }
}

/** Прямоугольник объекта в координатах сцены — БЕЗ учёта поворота (локальная
    рамка, в которой объект и нарисован). Габарит повёрнутого даёт objectAABB. */
export function objectBounds(o) {
  switch (o.type) {
    case OBJ.comment:
      return { x: o.x, y: o.y, w: COMMENT_PIN, h: COMMENT_PIN }
    case OBJ.vector:
      return pointsBounds(vectorOutline(o), o)
    case OBJ.path:
      return pointsBounds(o.points || [], o)
    case OBJ.line:
    case OBJ.arrow: {
      const x = Math.min(o.x, o.x2); const y = Math.min(o.y, o.y2)
      return { x, y, w: Math.abs(o.x2 - o.x), h: Math.abs(o.y2 - o.y) }
    }
    case OBJ.text: {
      const size = o.size || 18
      const lines = String(o.text || '').split('\n')
      const w = Math.max(...lines.map((l) => l.length), 1) * size * 0.58
      return { x: o.x, y: o.y - size, w, h: lines.length * size * 1.35 }
    }
    default:
      return { x: o.x, y: o.y, w: o.w || 0, h: o.h || 0 }
  }
}

/** Центр локальной рамки — вокруг него объект и поворачивается. */
export function objectCenter(o) {
  const b = objectBounds(o)
  return { x: b.x + b.w / 2, y: b.y + b.h / 2 }
}

export function rotatePoint(px, py, cx, cy, deg) {
  if (!deg) return { x: px, y: py }
  const rad = (deg * Math.PI) / 180
  const cos = Math.cos(rad); const sin = Math.sin(rad)
  const dx = px - cx; const dy = py - cy
  return { x: cx + dx * cos - dy * sin, y: cy + dx * sin + dy * cos }
}

/** Габарит объекта с учётом поворота: рамка выделения, выгрузка, лассо. */
export function objectAABB(o) {
  const b = objectBounds(o)
  if (!o.angle) return b
  const c = { x: b.x + b.w / 2, y: b.y + b.h / 2 }
  const corners = [[b.x, b.y], [b.x + b.w, b.y], [b.x + b.w, b.y + b.h], [b.x, b.y + b.h]]
    .map(([x, y]) => rotatePoint(x, y, c.x, c.y, o.angle))
  const xs = corners.map((p) => p.x); const ys = corners.map((p) => p.y)
  return {
    x: Math.min(...xs), y: Math.min(...ys),
    w: Math.max(...xs) - Math.min(...xs), h: Math.max(...ys) - Math.min(...ys),
  }
}

/** Общая рамка списка объектов (null — пусто). */
export function sceneBounds(objects) {
  if (!objects.length) return null
  let minX = Infinity; let minY = Infinity; let maxX = -Infinity; let maxY = -Infinity
  for (const o of objects) {
    const b = objectAABB(o)
    minX = Math.min(minX, b.x); minY = Math.min(minY, b.y)
    maxX = Math.max(maxX, b.x + b.w); maxY = Math.max(maxY, b.y + b.h)
  }
  return { x: minX, y: minY, w: maxX - minX, h: maxY - minY }
}

/** Попадание точки сцены в объект (с допуском tolerance в единицах сцены). */
export function hitTest(o, px, py, tolerance = 6) {
  // Повёрнутый объект проверяем в его собственных координатах: точку крутим в
  // обратную сторону, дальше всё как обычно.
  if (o.angle) {
    const c = objectCenter(o)
    const p = rotatePoint(px, py, c.x, c.y, -o.angle)
    px = p.x; py = p.y
  }
  if (o.type === OBJ.path || o.type === OBJ.vector) {
    const pts = o.type === OBJ.vector ? vectorOutline(o) : (o.points || [])
    const reach = tolerance + (o.width || 3) / 2
    if ((o.filled || o.closed) && pointInPolygon(pts, px, py)) return true
    for (let i = 0; i + 3 < pts.length; i += 2) {
      if (distToSegment(px, py, pts[i], pts[i + 1], pts[i + 2], pts[i + 3]) <= reach) return true
    }
    return false
  }
  if (o.type === OBJ.line || o.type === OBJ.arrow) {
    return distToSegment(px, py, o.x, o.y, o.x2, o.y2) <= tolerance + (o.width || 3) / 2
  }
  const b = objectBounds(o)
  return px >= b.x - tolerance && px <= b.x + b.w + tolerance
    && py >= b.y - tolerance && py <= b.y + b.h + tolerance
}

/** Точка внутри полигона (points — плоский список пар) — лассо и маски. */
export function pointInPolygon(points, px, py) {
  let inside = false
  const n = points.length / 2
  for (let i = 0, j = n - 1; i < n; j = i, i += 1) {
    const xi = points[i * 2]; const yi = points[i * 2 + 1]
    const xj = points[j * 2]; const yj = points[j * 2 + 1]
    if ((yi > py) !== (yj > py) && px < ((xj - xi) * (py - yi)) / (yj - yi) + xi) inside = !inside
  }
  return inside
}

function distToSegment(px, py, x1, y1, x2, y2) {
  const dx = x2 - x1; const dy = y2 - y1
  const len2 = dx * dx + dy * dy
  if (!len2) return Math.hypot(px - x1, py - y1)
  let t = ((px - x1) * dx + (py - y1) * dy) / len2
  t = Math.max(0, Math.min(1, t))
  return Math.hypot(px - (x1 + t * dx), py - (y1 + t * dy))
}

/** Сдвиг объекта (перетаскивание). */
export function moveObject(o, dx, dy) {
  if (o.type === OBJ.vector) return { ...o, nodes: moveNodes(nodesOf(o), dx, dy) }
  if (o.type === OBJ.path) {
    const pts = [...(o.points || [])]
    for (let i = 0; i + 1 < pts.length; i += 2) { pts[i] += dx; pts[i + 1] += dy }
    return { ...o, points: pts }
  }
  if (o.type === OBJ.line || o.type === OBJ.arrow) {
    return { ...o, x: o.x + dx, y: o.y + dy, x2: o.x2 + dx, y2: o.y2 + dy }
  }
  return { ...o, x: o.x + dx, y: o.y + dy }
}

/** Масштабирование объекта из его рамки в новую (ручки выделения). */
export function scaleObject(o, from, to) {
  const sx = from.w ? to.w / from.w : 1
  const sy = from.h ? to.h / from.h : 1
  const mapX = (x) => to.x + (x - from.x) * sx
  const mapY = (y) => to.y + (y - from.y) * sy
  if (o.type === OBJ.vector) return { ...o, nodes: scaleNodes(nodesOf(o), mapX, mapY) }
  if (o.type === OBJ.path) {
    const pts = [...(o.points || [])]
    for (let i = 0; i + 1 < pts.length; i += 2) { pts[i] = mapX(pts[i]); pts[i + 1] = mapY(pts[i + 1]) }
    return { ...o, points: pts }
  }
  if (o.type === OBJ.line || o.type === OBJ.arrow) {
    return { ...o, x: mapX(o.x), y: mapY(o.y), x2: mapX(o.x2), y2: mapY(o.y2) }
  }
  if (o.type === OBJ.text) {
    return { ...o, x: mapX(o.x), y: mapY(o.y), size: Math.max(8, Math.round((o.size || 18) * sy)) }
  }
  return { ...o, x: mapX(o.x), y: mapY(o.y), w: (o.w || 0) * sx, h: (o.h || 0) * sy }
}

/** Поворот объекта вокруг ОБЩЕГО центра выделения: сам угол плюс перенос
    центра — иначе группа объектов крутилась бы каждый вокруг себя. */
export function rotateObject(o, deg, pivot) {
  const c = objectCenter(o)
  const moved = rotatePoint(c.x, c.y, pivot.x, pivot.y, deg)
  const shifted = moveObject(o, moved.x - c.x, moved.y - c.y)
  return { ...shifted, angle: normalizeAngle((o.angle || 0) + deg) }
}

export function normalizeAngle(deg) {
  const n = Math.round(((deg % 360) + 360) % 360 * 10) / 10
  return n === 0 ? 0 : n
}

/** Текст надписей, стикеров и комментариев — превью плитки и поиск. */
export function sceneText(scene) {
  return normalizeScene(scene).objects
    .map((o) => (o.text || '').trim())
    .filter(Boolean)
    .join('\n')
}

/** Участники булевой группы в порядке отрисовки; первый — база операции. */
export function boolMembers(objects, id) {
  return objects.filter((o) => o.bool === id)
}

/** Вершины много-/звезды в её рамке (зеркало серверного polygonPoints). */
export function polygonPoints(o) {
  const sides = Math.max(3, Math.min(24, Math.round(o.sides || 5)))
  const cx = o.x + o.w / 2; const cy = o.y + o.h / 2
  const rx = o.w / 2; const ry = o.h / 2
  const inner = Math.min(0.9, Math.max(0.1, o.inner || 0.5))
  const count = o.star ? sides * 2 : sides
  const out = []
  for (let i = 0; i < count; i += 1) {
    const angle = (Math.PI * 2 * i) / count - Math.PI / 2
    const k = o.star && i % 2 ? inner : 1
    out.push(cx + Math.cos(angle) * rx * k, cy + Math.sin(angle) * ry * k)
  }
  return out
}

/** Полилиния контура объекта — булевы операции, текст по контуру, привязки. */
export function objectOutline(o) {
  return outlinePoints(o, { polygonPoints })
}

// ── Дерево слоёв ─────────────────────────────────────────────────

// Названия и значки типов: ими подписаны строки объектов в панели слоёв.
export const OBJ_LABELS = {
  path: 'Штрих',
  vector: 'Контур',
  line: 'Линия',
  arrow: 'Стрелка',
  rect: 'Прямоугольник',
  ellipse: 'Овал',
  diamond: 'Ромб',
  polygon: 'Многоугольник',
  text: 'Надпись',
  sticky: 'Заметка',
  image: 'Картинка',
  comment: 'Комментарий',
}

export const OBJ_ICONS = {
  path: 'draw',
  vector: 'polyline',
  line: 'horizontal_rule',
  arrow: 'arrow_right_alt',
  rect: 'crop_square',
  ellipse: 'circle',
  diamond: 'change_history',
  polygon: 'pentagon',
  text: 'title',
  sticky: 'sticky_note_2',
  image: 'image',
  comment: 'add_comment',
}

/** Подпись объекта в дереве слоёв: свой текст, иначе — название типа. */
export function objectLabel(o) {
  const text = String(o.text || '').trim().split('\n')[0]
  if (text) return text.length > 28 ? `${text.slice(0, 28)}…` : text
  if (o.erase) return o.filled ? 'Стёртая область' : 'Ластик'
  if (o.type === OBJ.polygon && o.star) return 'Звезда'
  return OBJ_LABELS[o.type] || 'Объект'
}

export function objectIcon(o) {
  if (o.erase) return 'ink_eraser'
  if (o.maskObject) return 'photo_filter'
  return OBJ_ICONS[o.type] || 'category'
}
