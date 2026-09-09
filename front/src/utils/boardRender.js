/* Рендер доски: фон, объекты, слои и кадры. Модель живёт в boardScene.js —
   здесь только отрисовка, поэтому её можно менять, не трогая формат сцены.

   Слой рисуется ГРУППАМИ через offscreen-буферы, когда у него есть своё
   свойство на всё содержимое (прозрачность, наложение, обтравка, маска, эффект)
   или внутри есть то, что складывается композитом: булева группа, объект-маска
   и стирающие штрихи. Обычный слой идёт коротким путём прямо на холст —
   иначе каждая доска платила бы за возможности, которыми не пользуется. */
import {
  BOOL_COMPOSITE, COMMENT_PIN, OBJ, isHexColor, normalizeScene, objectBounds, objectCenter,
  objectOutline, polygonPoints, resolveColor,
} from '@/utils/boardScene.js'
import {
  applyEffects, cssFilter, hasEffects, makeGradient,
} from '@/utils/boardPaint.js'
import { outlineMetrics, pointAtLength, vectorShape } from '@/utils/boardVector.js'

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
      ctx.strokeStyle = resolveColor('__grid', '--color-outline-dim')
      ctx.fillStyle = ctx.strokeStyle
      ctx.globalAlpha = 0.5
      ctx.lineWidth = 1
      for (let x = offsetX; x < width; x += step) {
        for (let y = offsetY; y < height; y += step) {
          if (background === 'dots') ctx.fillRect(x, y, 1.5, 1.5)
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

/** Контур свободной кривой: ломаная либо сглаженная (квадратичные сегменты
    через середины отрезков — так рисуют «кисть» все редакторы). */
export function pathShape(ctx, o) {
  const pts = o.points || []
  if (pts.length < 4) return false
  ctx.beginPath()
  ctx.moveTo(pts[0], pts[1])
  if (o.smooth && pts.length >= 6) {
    for (let i = 2; i + 3 < pts.length; i += 2) {
      const mx = (pts[i] + pts[i + 2]) / 2
      const my = (pts[i + 1] + pts[i + 3]) / 2
      ctx.quadraticCurveTo(pts[i], pts[i + 1], mx, my)
    }
    ctx.lineTo(pts[pts.length - 2], pts[pts.length - 1])
  } else {
    for (let i = 2; i + 1 < pts.length; i += 2) ctx.lineTo(pts[i], pts[i + 1])
  }
  if (o.closed) ctx.closePath()
  return true
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

/* Текст по контуру: буквы расставляются вдоль полилинии опорного объекта и
   поворачиваются по касательной. Опорный контур передаётся ГОТОВОЙ полилинией
   (её считает вызывающий по своей сцене) — рендер не ищет объекты сам. */
function drawTextOnPath(ctx, o, outline, size, color) {
  if (!outline || outline.length < 4) return
  const metrics = outlineMetrics(outline)
  ctx.save()
  ctx.fillStyle = color
  ctx.font = `${size}px Inter, system-ui, sans-serif`
  ctx.textBaseline = 'alphabetic'
  ctx.textAlign = 'center'
  const chars = [...String(o.text || '')]
  const widths = chars.map((ch) => ctx.measureText(ch).width)
  const total = widths.reduce((sum, w) => sum + w, 0)
  const start = Math.max(0, ((o.pathOffset || 0) / 100) * metrics.total)
  // Текст длиннее контура просто обрывается — тянуть буквы было бы враньём.
  let cursor = start
  chars.forEach((ch, i) => {
    const at = pointAtLength(outline, metrics, cursor + widths[i] / 2)
    if (!at || cursor > metrics.total) return
    ctx.save()
    ctx.translate(at.x, at.y)
    ctx.rotate(at.angle)
    ctx.fillText(ch, 0, -(o.pathSide === 'inside' ? -size * 0.8 : 0))
    ctx.restore()
    cursor += widths[i]
  })
  ctx.restore()
  return total
}

/** Заливка объекта: градиент, если задан, иначе цвет (или ничего). */
function fillPaint(ctx, o, col, colors) {
  if (o.gradient) {
    const box = objectBounds(o)
    const paint = makeGradient(ctx, o.gradient, box, (key) => col(key))
    if (paint) return paint
  }
  if (!o.fill) return null
  return col(o.fill)
}

function fillAndStroke(ctx, o, col, colors, path) {
  path()
  const paint = fillPaint(ctx, o, col, colors)
  if (paint) {
    ctx.save()
    // Обычная заливка полупрозрачна (за фигурой виден холст), «сплошная»
    // приходит из ведра и градиента — их человек задал намеренно.
    ctx.globalAlpha = ctx.globalAlpha * (o.solid || o.gradient ? 1 : 0.35)
    ctx.fillStyle = paint
    ctx.fill()
    ctx.restore()
  }
  if ((o.width ?? 3) > 0) ctx.stroke()
}

/** Отрисовать один объект (ctx уже в координатах сцены). images — Map<src, Image>.
    colors — готовая палитра (EXPORT_COLORS для файлов); без неё цвета берутся
    из темы приложения, как на живом холсте. outline — полилиния опорного
    контура для текста по контуру. */
export function drawObject(ctx, o, images, colors, outline) {
  const col = (key, token) => (isHexColor(key) ? key : (colors?.[key] || resolveColor(key, token)))
  const stroke = col(o.color || 'ink')
  const width = o.width ?? 3
  ctx.save()
  if (o.angle) {
    const c = objectCenter(o)
    ctx.translate(c.x, c.y)
    ctx.rotate((o.angle * Math.PI) / 180)
    ctx.translate(-c.x, -c.y)
  }
  // Стирающий штрих выедает то, что нарисовано в слое ниже него: обычный путь,
  // но в режиме destination-out (пиксельный ластик и «стереть внутри лассо»).
  if (o.erase) {
    ctx.globalCompositeOperation = 'destination-out'
    ctx.globalAlpha = 1
  } else {
    ctx.globalAlpha = o.opacity != null && o.opacity > 0 ? o.opacity : 1
    if (o.effects) applyEffects(ctx, o.effects, (key) => col(key))
  }
  ctx.strokeStyle = stroke
  ctx.lineWidth = width
  ctx.lineCap = 'round'
  ctx.lineJoin = 'round'

  switch (o.type) {
    case OBJ.path: {
      if (!pathShape(ctx, o)) break
      if (o.erase) {
        // Заливка — «стереть область» (лассо), обводка — сам ластик.
        if (o.filled) ctx.fill()
        else ctx.stroke()
        break
      }
      if (o.filled || o.gradient) {
        const paint = fillPaint(ctx, o, col, colors) || stroke
        ctx.save()
        ctx.globalAlpha = ctx.globalAlpha * (o.solid || o.gradient ? 1 : 0.35)
        ctx.fillStyle = paint
        ctx.fill()
        ctx.restore()
      }
      if (width > 0) ctx.stroke()
      break
    }
    case OBJ.vector: {
      if (!vectorShape(ctx, o)) break
      if (o.filled || o.gradient) {
        const paint = fillPaint(ctx, o, col, colors) || stroke
        ctx.save()
        ctx.globalAlpha = ctx.globalAlpha * (o.solid || o.gradient ? 1 : 0.35)
        ctx.fillStyle = paint
        ctx.fill()
        ctx.restore()
      }
      if (width > 0) ctx.stroke()
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
      fillAndStroke(ctx, o, col, colors, () => roundRect(ctx, o.x, o.y, o.w, o.h, o.radius ?? 8))
      break
    case OBJ.ellipse:
      fillAndStroke(ctx, o, col, colors, () => {
        ctx.beginPath()
        ctx.ellipse(o.x + o.w / 2, o.y + o.h / 2, Math.abs(o.w / 2), Math.abs(o.h / 2), 0, 0, Math.PI * 2)
      })
      break
    case OBJ.diamond:
      fillAndStroke(ctx, o, col, colors, () => {
        ctx.beginPath()
        ctx.moveTo(o.x + o.w / 2, o.y)
        ctx.lineTo(o.x + o.w, o.y + o.h / 2)
        ctx.lineTo(o.x + o.w / 2, o.y + o.h)
        ctx.lineTo(o.x, o.y + o.h / 2)
        ctx.closePath()
      })
      break
    case OBJ.polygon:
      fillAndStroke(ctx, o, col, colors, () => {
        const pts = polygonPoints(o)
        ctx.beginPath()
        ctx.moveTo(pts[0], pts[1])
        for (let i = 2; i + 1 < pts.length; i += 2) ctx.lineTo(pts[i], pts[i + 1])
        ctx.closePath()
      })
      break
    case OBJ.sticky: {
      const tint = col(o.color || 'amber')
      ctx.globalAlpha = 0.85
      ctx.fillStyle = fillPaint(ctx, o, col, colors) || tint
      roundRect(ctx, o.x, o.y, o.w, o.h, o.radius ?? 6)
      ctx.fill()
      ctx.globalAlpha = 1
      drawText(ctx, o.text, o.x + 12, o.y + 26, 16, col('ink'), o.w - 24)
      break
    }
    case OBJ.text: {
      const size = o.size || 18
      if (o.pathRef && outline) drawTextOnPath(ctx, o, outline, size, stroke)
      else drawText(ctx, o.text, o.x, o.y, size, stroke)
      break
    }
    case OBJ.comment: {
      const tint = col(o.resolved ? 'green' : (o.color || 'amber'))
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
      ctx.fillStyle = col('chalk', '--color-surface')
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
        if (o.radius) {
          // Скруглённая картинка: обрезаем по той же рамке, что и у фигуры.
          ctx.save()
          roundRect(ctx, o.x, o.y, o.w, o.h, o.radius)
          ctx.clip()
          ctx.drawImage(img, o.x, o.y, o.w, o.h)
          ctx.restore()
        } else {
          ctx.drawImage(img, o.x, o.y, o.w, o.h)
        }
      } else {
        ctx.strokeStyle = col('__ph', '--color-outline-variant')
        ctx.strokeRect(o.x, o.y, o.w, o.h)
      }
      break
    }
    default:
      break
  }
  ctx.restore()
}

/* Буферы слоёв переиспользуются пулом: иначе каждый кадр рисования выделял бы
   по несколько полноэкранных растров. */
const bufferPool = []

function takeBuffer(width, height) {
  const canvas = bufferPool.pop() || (typeof document === 'undefined' ? null : document.createElement('canvas'))
  if (!canvas) return null
  if (canvas.width !== width || canvas.height !== height) {
    canvas.width = width
    canvas.height = height
  } else {
    canvas.getContext('2d')?.clearRect(0, 0, width, height)
  }
  return canvas
}

function freeBuffer(canvas) {
  if (canvas && bufferPool.length < 6) bufferPool.push(canvas)
}

/** Буфер размером с холст, уже настроенный на мировые координаты. */
function makeBuffer(env) {
  const canvas = takeBuffer(env.width, env.height)
  const ctx = canvas?.getContext('2d')
  if (!ctx) return null
  ctx.save()
  if (env.matrix) ctx.setTransform(env.matrix)
  ctx.scale(env.camera.scale, env.camera.scale)
  ctx.translate(-env.camera.x, -env.camera.y)
  return { canvas, ctx }
}

/** Свести готовый буфер в целевой контекст (в экранных координатах). */
function blit(dst, canvas, { alpha = 1, blend = 'source-over', filter = '' } = {}) {
  dst.save()
  dst.setTransform(1, 0, 0, 1, 0, 0)
  dst.globalAlpha = alpha
  dst.globalCompositeOperation = blend
  if (filter && 'filter' in dst) dst.filter = filter
  dst.drawImage(canvas, 0, 0)
  dst.restore()
}

/* Булева группа: первый участник — база, остальные складываются с ней своим
   композитом. Стиль берётся у базы: результат операции — ОДНА фигура, и своих
   цветов у вычитаемого быть не может. */
function drawBoolGroup(ctx, members, env) {
  const base = members[0]
  const buf = makeBuffer(env)
  if (!buf) return
  drawObject(buf.ctx, { ...base, opacity: 1, effects: null }, env.images, env.colors, env.outlineOf?.(base))
  for (const m of members.slice(1)) {
    buf.ctx.save()
    buf.ctx.globalCompositeOperation = BOOL_COMPOSITE[m.boolOp] || 'source-over'
    drawObject(buf.ctx, {
      ...m,
      color: base.color, fill: base.fill, gradient: base.gradient,
      width: base.width, radius: m.radius, opacity: 1, effects: null, angle: m.angle,
    }, env.images, env.colors)
    buf.ctx.restore()
  }
  buf.ctx.restore()
  blit(ctx, buf.canvas, {
    alpha: base.opacity != null && base.opacity > 0 ? base.opacity : 1,
    filter: cssFilter(base.effects),
  })
  freeBuffer(buf.canvas)
}

/** Объекты слоя по порядку: булевы группы рисуются целиком на месте базы. */
function drawItems(ctx, items, env) {
  const done = new Set()
  for (const o of items) {
    if (done.has(o.id)) continue
    if (o.bool) {
      const members = items.filter((m) => m.bool === o.bool)
      members.forEach((m) => done.add(m.id))
      drawBoolGroup(ctx, members, env)
      continue
    }
    drawObject(ctx, o, env.images, env.colors, env.outlineOf?.(o))
  }
}

/* Объект-маска (как в Figma «использовать как маску»): показывает всё, что
   лежит НАД ним в слое, только внутри своей формы — до следующей маски. */
function splitByMasks(items) {
  const sections = []
  let current = { mask: null, items: [] }
  for (const o of items) {
    if (o.maskObject) {
      sections.push(current)
      current = { mask: o, items: [] }
      continue
    }
    current.items.push(o)
  }
  sections.push(current)
  return sections.filter((s) => s.mask || s.items.length)
}

function drawMaskedSection(ctx, section, env) {
  const content = makeBuffer(env)
  if (!content) return
  drawItems(content.ctx, section.items, env)
  content.ctx.restore()

  const maskBuf = makeBuffer(env)
  if (maskBuf) {
    drawObject(maskBuf.ctx, { ...section.mask, opacity: 1, maskObject: false }, env.images, env.colors)
    maskBuf.ctx.restore()
    blit(content.ctx, maskBuf.canvas, { blend: 'destination-in' })
    freeBuffer(maskBuf.canvas)
  }
  blit(ctx, content.canvas)
  freeBuffer(content.canvas)
}

// Слою нужен свой буфер, только если он что-то делает со всем содержимым.
function layerNeedsBuffer(layer, objects) {
  return layer.opacity < 1 || layer.blend !== 'normal' || layer.clip || !!layer.mask
    || hasEffects(layer.effects) || objects.some((o) => o.erase || o.maskObject)
}

function drawLayerObjects(ctx, layer, objects, env) {
  const items = objects.filter((o) => !o.hidden)
  const paint = () => {
    for (const section of splitByMasks(items)) {
      if (section.mask) drawMaskedSection(ctx, section, env)
      else drawItems(ctx, section.items, env)
    }
  }
  if (!layer.mask) {
    paint()
    return
  }
  ctx.save()
  const pts = layer.mask.points
  ctx.beginPath()
  ctx.moveTo(pts[0], pts[1])
  for (let i = 2; i + 1 < pts.length; i += 2) ctx.lineTo(pts[i], pts[i + 1])
  ctx.closePath()
  if (!layer.mask.invert) ctx.clip()
  paint()
  // Обратная маска: содержимое нарисовано целиком, а полигон из него выеден.
  if (layer.mask.invert) {
    ctx.globalCompositeOperation = 'destination-out'
    ctx.fill()
  }
  ctx.restore()
}

/** Отрисовка кадра сцены: слои снизу вверх с их прозрачностью, наложением,
    масками и обтравкой. Сцена приходит УЖЕ нормализованной (renderScene делает
    это один раз на кадр — на большой доске лишний проход заметен). */
function renderFrame(ctx, s, { camera, images, colors, frame, alpha = 1 }) {
  const target = ctx.canvas
  const matrix = ctx.getTransform?.()
  const byId = new Map(s.objects.map((o) => [o.id, o]))
  const env = {
    images, colors, camera, matrix, width: target.width, height: target.height,
    // Текст по контуру: полилиния опорного объекта берётся из этой же сцены.
    outlineOf: (o) => (o.pathRef && byId.has(o.pathRef) ? outlineOf(byId.get(o.pathRef)) : null),
  }

  const compose = (dst, canvas, opacity, blend, extraAlpha = 1, filter = '') => {
    blit(dst, canvas, { alpha: opacity * extraAlpha, blend, filter })
  }

  /* Группа обтравки: базовый слой плюс идущие над ним слои с clip. Она
     собирается в своём буфере, потому что обтравленные слои видны только по
     альфе базы, а на холсте этой альфы уже не найти. */
  let group = null

  const flushGroup = () => {
    if (!group) return
    compose(ctx, group.canvas, group.opacity, group.blend, alpha, group.filter)
    freeBuffer(group.canvas)
    freeBuffer(group.base)
    group = null
  }

  s.layers.forEach((layer, index) => {
    if (!layer.visible) {
      if (!layer.clip) flushGroup()
      return
    }
    const objects = s.objects.filter((o) => o.layer === layer.id && (!frame || !o.frame || o.frame === frame))
    if (!layer.clip) flushGroup()

    /* Полупрозрачный кадр «кальки» тоже идёт через буфер: у объектов своя
       прозрачность, и на общем холсте она затирала бы прозрачность призрака
       (drawObject задаёт globalAlpha сам). */
    const buffered = alpha < 1 || layerNeedsBuffer(layer, objects)
    const nextClips = !!s.layers[index + 1]?.clip
    if (!buffered && !group && !nextClips) {
      ctx.save()
      ctx.scale(camera.scale, camera.scale)
      ctx.translate(-camera.x, -camera.y)
      drawLayerObjects(ctx, layer, objects, env)
      ctx.restore()
      return
    }

    const buf = makeBuffer(env)
    if (!buf) return
    drawLayerObjects(buf.ctx, layer, objects, env)
    buf.ctx.restore()

    if (layer.clip && group) {
      blit(buf.ctx, group.base, { blend: 'destination-in' })
    }

    const filter = cssFilter(layer.effects)
    if (group) {
      compose(group.ctx, buf.canvas, layer.opacity, layer.blend, 1, filter)
      freeBuffer(buf.canvas)
      return
    }

    if (nextClips) {
      // Слой открывает группу: его альфу держим отдельной копией — буфер
      // группы к приходу обтравленных слоёв уже смешан с ними.
      const base = takeBuffer(env.width, env.height)
      const bctx = base?.getContext('2d')
      if (bctx) {
        bctx.save()
        bctx.setTransform(1, 0, 0, 1, 0, 0)
        bctx.drawImage(buf.canvas, 0, 0)
        bctx.restore()
      }
      group = {
        canvas: buf.canvas, ctx: buf.ctx, base,
        opacity: layer.opacity, blend: layer.blend, filter,
      }
      return
    }

    compose(ctx, buf.canvas, layer.opacity, layer.blend, alpha, filter)
    freeBuffer(buf.canvas)
  })
  flushGroup()
}

/* Полилинии опорных контуров кэшируются по самому объекту: текст по контуру
   перерисовывается каждый кадр, а разложение кривых в ломаную не бесплатно.
   Ключ — объект, а не id: правка создаёт новый объект, и кэш обновляется сам. */
const outlineCache = new WeakMap()

function outlineOf(o) {
  const cached = outlineCache.get(o)
  if (cached) return cached
  const points = objectOutline(o)
  outlineCache.set(o, points)
  return points
}

/** Полная отрисовка сцены в контекст (общая для холста, превью и экспорта).
    frame — кадр анимации, onion — «калька»: соседние кадры полупрозрачно. */
export function renderScene(ctx, scene, {
  width, height, camera, images, background = true, colors, frame = null, onion = [],
}) {
  const s = normalizeScene(scene)
  if (background) drawBackground(ctx, { width, height, camera, background: s.background })
  for (const ghost of onion) {
    renderFrame(ctx, s, { camera, images, colors, frame: ghost.frame, alpha: ghost.alpha })
  }
  renderFrame(ctx, s, { camera, images, colors, frame })
}
