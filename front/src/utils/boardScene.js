/* Сцена доски: модель объектов холста, геометрия и рендер.
   Зеркало back-go/board/internal/domain/scene.go — сервер по этим же полям
   считает текст надписей для поиска и строит SVG при выгрузке.

   Цвет объекта хранится КЛЮЧОМ палитры («ink», «red»…), а не значением: canvas
   не понимает var(--tag-*), поэтому ключ разворачивается в реальный цвет через
   getComputedStyle в момент рисования. Так доска остаётся в теме приложения и
   сама перекрашивается в тёмном режиме. */

export const OBJ = {
  path: 'path',
  comment: 'comment',
  line: 'line',
  arrow: 'arrow',
  rect: 'rect',
  ellipse: 'ellipse',
  diamond: 'diamond',
  text: 'text',
  sticky: 'sticky',
  image: 'image',
}

// Слой по умолчанию: сцены первой версии не знали про слои, поэтому весь их
// холст переезжает сюда.
export const BASE_LAYER = 'base'

export function newLayer(name = 'Слой') {
  return { id: newId(), name, visible: true, locked: false }
}

// Палитра доски: ключ → CSS-переменная. Восемь цветов тегов + «чернила» и «мел»
// (нейтральные, читаемые на любом фоне холста).
export const SCENE_COLORS = [
  { key: 'ink', label: 'Чернила', token: '--color-text' },
  { key: 'red', label: 'Красный', token: '--tag-red-accent' },
  { key: 'orange', label: 'Оранжевый', token: '--tag-orange-accent' },
  { key: 'amber', label: 'Янтарный', token: '--tag-amber-accent' },
  { key: 'green', label: 'Зелёный', token: '--tag-green-accent' },
  { key: 'teal', label: 'Бирюзовый', token: '--tag-teal-accent' },
  { key: 'blue', label: 'Синий', token: '--tag-blue-accent' },
  { key: 'violet', label: 'Фиолетовый', token: '--tag-violet-accent' },
  { key: 'pink', label: 'Розовый', token: '--tag-pink-accent' },
]

const COLOR_TOKENS = Object.fromEntries(SCENE_COLORS.map((c) => [c.key, c.token]))

// Толщины пера и размеры надписей — общий словарь для тулбара и рендера.
export const STROKE_WIDTHS = [2, 4, 8, 16]
export const TEXT_SIZES = [14, 18, 24, 32, 48]

export const BACKGROUNDS = [
  { key: 'grid', label: 'Сетка' },
  { key: 'dots', label: 'Точки' },
  { key: 'plain', label: 'Чистый лист' },
]

// resolveColor — значение токена палитры; кэш сбрасывается при смене темы.
let colorCache = new Map()
let colorCacheKey = ''

export function resolveColor(key, fallbackToken = '--color-text') {
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
    version: 2,
    background: 'grid',
    layers: [{ id: BASE_LAYER, name: 'Слой 1', visible: true, locked: false }],
    objects: [],
  }
}

/** Нормализация сцены с сервера (или из файла): не доверяем структуре.
    Заодно поднимает сцены v1 до v2: добавляет базовый слой и приписывает к нему
    все объекты без слоя. */
export function normalizeScene(raw) {
  const scene = raw && typeof raw === 'object' ? raw : {}
  const objects = Array.isArray(scene.objects) ? scene.objects.filter((o) => o && o.type) : []

  let layers = Array.isArray(scene.layers) ? scene.layers.filter((l) => l && l.id) : []
  if (!layers.length) layers = [{ id: BASE_LAYER, name: 'Слой 1', visible: true, locked: false }]
  layers = layers.map((l) => ({
    id: String(l.id),
    name: l.name || 'Слой',
    visible: l.visible !== false,
    locked: !!l.locked,
  }))

  const known = new Set(layers.map((l) => l.id))
  const fallback = layers[0].id
  return {
    version: 2,
    background: BACKGROUNDS.some((b) => b.key === scene.background) ? scene.background : 'grid',
    layers,
    objects: objects.map((o) => (known.has(o.layer) ? o : { ...o, layer: fallback })),
  }
}

/** Объекты в порядке отрисовки: снизу вверх по слоям. onlyVisible — для рендера. */
export function orderedObjects(scene, { onlyVisible = true } = {}) {
  const s = normalizeScene(scene)
  const out = []
  for (const layer of s.layers) {
    if (onlyVisible && !layer.visible) continue
    for (const o of s.objects) {
      if (o.layer === layer.id) out.push(o)
    }
  }
  return out
}

/** Слои, доступные для правки: видимые и незаблокированные. */
export function editableLayerIds(scene) {
  return new Set(normalizeScene(scene).layers.filter((l) => l.visible && !l.locked).map((l) => l.id))
}

let seq = 0
export function newId() {
  seq += 1
  return `o${Date.now().toString(36)}${seq.toString(36)}`
}

// COMMENT_PIN — диаметр булавки комментария в единицах сцены.
export const COMMENT_PIN = 28

/** Прямоугольник объекта в координатах сцены. */
export function objectBounds(o) {
  switch (o.type) {
    case OBJ.comment:
      return { x: o.x, y: o.y, w: COMMENT_PIN, h: COMMENT_PIN }
    case OBJ.path: {
      const pts = o.points || []
      if (pts.length < 2) return { x: o.x || 0, y: o.y || 0, w: 0, h: 0 }
      let minX = Infinity; let minY = Infinity; let maxX = -Infinity; let maxY = -Infinity
      for (let i = 0; i + 1 < pts.length; i += 2) {
        minX = Math.min(minX, pts[i]); maxX = Math.max(maxX, pts[i])
        minY = Math.min(minY, pts[i + 1]); maxY = Math.max(maxY, pts[i + 1])
      }
      return { x: minX, y: minY, w: maxX - minX, h: maxY - minY }
    }
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

/** Общая рамка списка объектов (null — пусто). */
export function sceneBounds(objects) {
  if (!objects.length) return null
  let minX = Infinity; let minY = Infinity; let maxX = -Infinity; let maxY = -Infinity
  for (const o of objects) {
    const b = objectBounds(o)
    minX = Math.min(minX, b.x); minY = Math.min(minY, b.y)
    maxX = Math.max(maxX, b.x + b.w); maxY = Math.max(maxY, b.y + b.h)
  }
  return { x: minX, y: minY, w: maxX - minX, h: maxY - minY }
}

/** Попадание точки сцены в объект (с допуском tolerance в единицах сцены). */
export function hitTest(o, px, py, tolerance = 6) {
  if (o.type === OBJ.path) {
    const pts = o.points || []
    for (let i = 0; i + 3 < pts.length; i += 2) {
      if (distToSegment(px, py, pts[i], pts[i + 1], pts[i + 2], pts[i + 3]) <= tolerance + (o.width || 3) / 2) return true
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

/** Текст надписей, стикеров и комментариев — превью плитки и поиск. */
export function sceneText(scene) {
  return normalizeScene(scene).objects
    .map((o) => (o.text || '').trim())
    .filter(Boolean)
    .join('\n')
}

// ── Рендер ───────────────────────────────────────────────────────

/** Фон холста: сетка/точки/чистый лист в координатах экрана. */
export function drawBackground(ctx, { width, height, camera, background }) {
  ctx.save()
  ctx.fillStyle = resolveColor('__surface', '--color-surface')
  ctx.fillRect(0, 0, width, height)
  if (background !== 'plain') {
    const step = 32 * camera.scale
    if (step > 6) {
      const offsetX = ((-camera.x * camera.scale) % step + step) % step
      const offsetY = ((-camera.y * camera.scale) % step + step) % step
      ctx.strokeStyle = resolveColor('__grid', '--color-outline-variant')
      ctx.fillStyle = ctx.strokeStyle
      ctx.globalAlpha = 0.5
      ctx.lineWidth = 1
      for (let x = offsetX; x < width; x += step) {
        for (let y = offsetY; y < height; y += step) {
          if (background === 'dots') {
            ctx.fillRect(x, y, 1.5, 1.5)
          }
        }
        if (background === 'grid') {
          ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, height); ctx.stroke()
        }
      }
      if (background === 'grid') {
        for (let y = offsetY; y < height; y += step) {
          ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(width, y); ctx.stroke()
        }
      }
    }
  }
  ctx.restore()
}

/** Отрисовать один объект (ctx уже в координатах сцены). images — Map<src, Image>. */
export function drawObject(ctx, o, images) {
  const stroke = resolveColor(o.color || 'ink')
  const width = o.width || 3
  ctx.save()
  ctx.globalAlpha = o.opacity != null && o.opacity > 0 ? o.opacity : 1
  ctx.strokeStyle = stroke
  ctx.lineWidth = width
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'

  switch (o.type) {
    case OBJ.path: {
      const pts = o.points || []
      if (pts.length >= 4) {
        ctx.beginPath()
        ctx.moveTo(pts[0], pts[1])
        for (let i = 2; i + 1 < pts.length; i += 2) ctx.lineTo(pts[i], pts[i + 1])
        ctx.stroke()
      }
      break
    }
    case OBJ.line:
    case OBJ.arrow: {
      ctx.beginPath()
      ctx.moveTo(o.x, o.y)
      ctx.lineTo(o.x2, o.y2)
      ctx.stroke()
      if (o.type === OBJ.arrow) drawArrowHead(ctx, o, stroke, width)
      break
    }
    case OBJ.rect:
      fillAndStroke(ctx, o, () => roundRect(ctx, o.x, o.y, o.w, o.h, 8))
      break
    case OBJ.ellipse:
      fillAndStroke(ctx, o, () => {
        ctx.beginPath()
        ctx.ellipse(o.x + o.w / 2, o.y + o.h / 2, Math.abs(o.w / 2), Math.abs(o.h / 2), 0, 0, Math.PI * 2)
      })
      break
    case OBJ.diamond:
      fillAndStroke(ctx, o, () => {
        ctx.beginPath()
        ctx.moveTo(o.x + o.w / 2, o.y)
        ctx.lineTo(o.x + o.w, o.y + o.h / 2)
        ctx.lineTo(o.x + o.w / 2, o.y + o.h)
        ctx.lineTo(o.x, o.y + o.h / 2)
        ctx.closePath()
      })
      break
    case OBJ.sticky: {
      const tint = resolveColor(o.color || 'amber')
      ctx.globalAlpha = 0.85
      ctx.fillStyle = tint
      roundRect(ctx, o.x, o.y, o.w, o.h, 6)
      ctx.fill()
      ctx.globalAlpha = 1
      drawText(ctx, o.text, o.x + 12, o.y + 26, 16, resolveColor('ink'), o.w - 24)
      break
    }
    case OBJ.text:
      drawText(ctx, o.text, o.x, o.y, o.size || 18, stroke)
      break
    case OBJ.comment: {
      const tint = resolveColor(o.resolved ? 'green' : (o.color || 'amber'))
      const r = COMMENT_PIN / 2
      ctx.globalAlpha = o.resolved ? 0.55 : 1
      ctx.fillStyle = tint
      ctx.beginPath()
      // Капля: круг с «хвостиком» вниз-влево, как булавка комментария.
      ctx.arc(o.x + r, o.y + r, r, 0, Math.PI * 2)
      ctx.moveTo(o.x + r - 5, o.y + COMMENT_PIN - 3)
      ctx.lineTo(o.x + r, o.y + COMMENT_PIN + 7)
      ctx.lineTo(o.x + r + 5, o.y + COMMENT_PIN - 3)
      ctx.closePath()
      ctx.fill()
      const count = 1 + (Array.isArray(o.replies) ? o.replies.length : 0)
      ctx.fillStyle = resolveColor('chalk', '--color-surface')
      ctx.font = 'bold 14px Inter, system-ui, sans-serif'
      ctx.textAlign = 'center'
      ctx.textBaseline = 'middle'
      ctx.fillText(String(count), o.x + r, o.y + r)
      ctx.textAlign = 'start'
      break
    }
    case OBJ.image: {
      const img = images?.get(o.src)
      if (img?.complete && img.naturalWidth) {
        ctx.drawImage(img, o.x, o.y, o.w, o.h)
      } else {
        ctx.strokeStyle = resolveColor('__ph', '--color-outline-variant')
        ctx.strokeRect(o.x, o.y, o.w, o.h)
      }
      break
    }
    default:
      break
  }
  ctx.restore()
}

function fillAndStroke(ctx, o, path) {
  path()
  if (o.fill) {
    ctx.save()
    ctx.globalAlpha = (o.opacity || 1) * 0.35
    ctx.fillStyle = resolveColor(o.fill)
    ctx.fill()
    ctx.restore()
  }
  ctx.stroke()
}

function roundRect(ctx, x, y, w, h, r) {
  const rr = Math.min(r, Math.abs(w) / 2, Math.abs(h) / 2)
  ctx.beginPath()
  ctx.moveTo(x + rr, y)
  ctx.arcTo(x + w, y, x + w, y + h, rr)
  ctx.arcTo(x + w, y + h, x, y + h, rr)
  ctx.arcTo(x, y + h, x, y, rr)
  ctx.arcTo(x, y, x + w, y, rr)
  ctx.closePath()
}

function drawArrowHead(ctx, o, color, width) {
  const angle = Math.atan2(o.y2 - o.y, o.x2 - o.x)
  const size = Math.max(10, width * 3)
  ctx.save()
  ctx.fillStyle = color
  ctx.beginPath()
  ctx.moveTo(o.x2, o.y2)
  ctx.lineTo(o.x2 - size * Math.cos(angle - Math.PI / 7), o.y2 - size * Math.sin(angle - Math.PI / 7))
  ctx.lineTo(o.x2 - size * Math.cos(angle + Math.PI / 7), o.y2 - size * Math.sin(angle + Math.PI / 7))
  ctx.closePath()
  ctx.fill()
  ctx.restore()
}

function drawText(ctx, text, x, y, size, color, maxWidth) {
  const lines = String(text || '').split('\n')
  ctx.save()
  ctx.fillStyle = color
  ctx.font = `${size}px Inter, system-ui, sans-serif`
  ctx.textBaseline = 'alphabetic'
  let line = 0
  for (const raw of lines) {
    for (const part of maxWidth ? wrapLine(ctx, raw, maxWidth) : [raw]) {
      ctx.fillText(part, x, y + line * size * 1.35)
      line += 1
    }
  }
  ctx.restore()
}

function wrapLine(ctx, text, maxWidth) {
  const words = String(text).split(' ')
  const out = []
  let cur = ''
  for (const w of words) {
    const probe = cur ? `${cur} ${w}` : w
    if (ctx.measureText(probe).width > maxWidth && cur) {
      out.push(cur)
      cur = w
    } else {
      cur = probe
    }
  }
  out.push(cur)
  return out
}

/** Полная отрисовка сцены в контекст (общая для холста, превью и экспорта). */
export function renderScene(ctx, scene, { width, height, camera, images, background = true }) {
  const s = normalizeScene(scene)
  if (background) drawBackground(ctx, { width, height, camera, background: s.background })
  ctx.save()
  ctx.scale(camera.scale, camera.scale)
  ctx.translate(-camera.x, -camera.y)
  for (const o of orderedObjects(s)) drawObject(ctx, o, images)
  ctx.restore()
}

/** PNG-миниатюра доски для плитки списка (Blob или null, если рисовать нечего). */
export async function renderPreview(scene, { width = 640, height = 400, images } = {}) {
  if (typeof document === 'undefined') return null
  const s = normalizeScene(scene)
  if (!s.objects.length) return null
  const b = sceneBounds(s.objects) || { x: 0, y: 0, w: width, h: height }
  const pad = 24
  const scale = Math.min(width / (b.w + pad * 2 || width), height / (b.h + pad * 2 || height), 2)
  const canvas = document.createElement('canvas')
  canvas.width = width
  canvas.height = height
  const ctx = canvas.getContext('2d')
  if (!ctx) return null
  const camera = { x: b.x - pad, y: b.y - pad, scale }
  renderScene(ctx, s, { width, height, camera, images })
  return new Promise((resolve) => canvas.toBlob(resolve, 'image/png', 0.85))
}
