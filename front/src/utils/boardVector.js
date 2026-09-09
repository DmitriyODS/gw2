/* Векторные контуры доски: узлы Безье, обход контура и превращение любой
   фигуры в контур. Отдельный модуль, потому что этим пользуются трое:
   перо (правка узлов), булевы операции (им нужен общий контур) и текст по
   контуру (ему нужна длина и точка на ней).

   Узел — `{ x, y, ix, iy, ox, oy }`: сама точка и две направляющие в АБСОЛЮТНЫХ
   координатах сцены. Абсолютные, а не относительные, потому что объект двигают
   и масштабируют теми же формулами, что и остальные точки холста, — иначе
   каждое преобразование пришлось бы дублировать для ручек. */

// FLATTEN_STEPS — на сколько отрезков дробится один сегмент кривой при обходе
// контура. Больше не нужно: контур меряется для попаданий и текста, а не для
// печати.
const FLATTEN_STEPS = 16

export function newNode(x, y) {
  return { x, y, ix: x, iy: y, ox: x, oy: y }
}

/** Есть ли у узла оттянутая направляющая (иначе сегмент — прямая). */
export function hasHandles(node) {
  return node.ix !== node.x || node.iy !== node.y || node.ox !== node.x || node.oy !== node.y
}

/** Узлы объекта — с достройкой отсутствующих направляющих. */
export function nodesOf(o) {
  return (o.nodes || []).map((n) => ({
    x: n.x, y: n.y,
    ix: n.ix ?? n.x, iy: n.iy ?? n.y,
    ox: n.ox ?? n.x, oy: n.oy ?? n.y,
  }))
}

/** Контур вектора в контексте (ctx уже в координатах сцены). */
export function vectorShape(ctx, o) {
  const nodes = nodesOf(o)
  if (nodes.length < 2) return false
  ctx.beginPath()
  ctx.moveTo(nodes[0].x, nodes[0].y)
  for (let i = 0; i + 1 < nodes.length; i += 1) {
    const a = nodes[i]; const b = nodes[i + 1]
    ctx.bezierCurveTo(a.ox, a.oy, b.ix, b.iy, b.x, b.y)
  }
  if (o.closed) {
    const a = nodes[nodes.length - 1]; const b = nodes[0]
    ctx.bezierCurveTo(a.ox, a.oy, b.ix, b.iy, b.x, b.y)
    ctx.closePath()
  }
  return true
}

function bezierPoint(a, b, t) {
  const mt = 1 - t
  const w0 = mt * mt * mt
  const w1 = 3 * mt * mt * t
  const w2 = 3 * mt * t * t
  const w3 = t * t * t
  return {
    x: w0 * a.x + w1 * a.ox + w2 * b.ix + w3 * b.x,
    y: w0 * a.y + w1 * a.oy + w2 * b.iy + w3 * b.y,
  }
}

/** Полилиния вектора: плоский массив [x0,y0,x1,y1,…]. */
export function vectorOutline(o, steps = FLATTEN_STEPS) {
  const nodes = nodesOf(o)
  if (!nodes.length) return []
  const out = [nodes[0].x, nodes[0].y]
  const pairs = []
  for (let i = 0; i + 1 < nodes.length; i += 1) pairs.push([nodes[i], nodes[i + 1]])
  if (o.closed && nodes.length > 2) pairs.push([nodes[nodes.length - 1], nodes[0]])
  for (const [a, b] of pairs) {
    const straight = !hasHandles(a) && !hasHandles(b)
    if (straight) {
      out.push(b.x, b.y)
      continue
    }
    for (let s = 1; s <= steps; s += 1) {
      const p = bezierPoint(a, b, s / steps)
      out.push(p.x, p.y)
    }
  }
  return out
}

/** Полилиния ЛЮБОГО объекта — общий язык булевых операций и текста по контуру.
    Для объектов без собственной геометрии (надпись, картинка) — их рамка. */
export function outlinePoints(o, shapes = {}) {
  const { polygonPoints } = shapes
  switch (o.type) {
    case 'vector':
      return vectorOutline(o)
    case 'path':
      return [...(o.points || [])]
    case 'ellipse': {
      const cx = o.x + o.w / 2; const cy = o.y + o.h / 2
      const out = []
      for (let i = 0; i < 48; i += 1) {
        const a = (Math.PI * 2 * i) / 48
        out.push(cx + Math.cos(a) * (o.w / 2), cy + Math.sin(a) * (o.h / 2))
      }
      return out
    }
    case 'diamond':
      return [o.x + o.w / 2, o.y, o.x + o.w, o.y + o.h / 2, o.x + o.w / 2, o.y + o.h, o.x, o.y + o.h / 2]
    case 'polygon':
      return polygonPoints ? polygonPoints(o) : []
    case 'line':
    case 'arrow':
      return [o.x, o.y, o.x2, o.y2]
    default:
      return [o.x, o.y, o.x + (o.w || 0), o.y, o.x + (o.w || 0), o.y + (o.h || 0), o.x, o.y + (o.h || 0)]
  }
}

/** Длина полилинии и накопленные длины — для текста по контуру. */
export function outlineMetrics(points) {
  const steps = [0]
  let total = 0
  for (let i = 2; i + 1 < points.length; i += 2) {
    total += Math.hypot(points[i] - points[i - 2], points[i + 1] - points[i - 1])
    steps.push(total)
  }
  return { total, steps }
}

/** Точка на полилинии по пройденной длине: координаты и наклон касательной. */
export function pointAtLength(points, metrics, length) {
  const { steps, total } = metrics
  if (!points.length) return null
  const target = Math.max(0, Math.min(total, length))
  let i = 1
  while (i < steps.length && steps[i] < target) i += 1
  const segStart = (i - 1) * 2
  const segEnd = i * 2
  if (segEnd + 1 >= points.length) {
    return { x: points[points.length - 2], y: points[points.length - 1], angle: 0 }
  }
  const segLen = steps[i] - steps[i - 1] || 1
  const t = (target - steps[i - 1]) / segLen
  const x0 = points[segStart]; const y0 = points[segStart + 1]
  const x1 = points[segEnd]; const y1 = points[segEnd + 1]
  return { x: x0 + (x1 - x0) * t, y: y0 + (y1 - y0) * t, angle: Math.atan2(y1 - y0, x1 - x0) }
}

/** Сдвиг узлов вместе с направляющими. */
export function moveNodes(nodes, dx, dy) {
  return nodes.map((n) => ({
    x: n.x + dx, y: n.y + dy,
    ix: (n.ix ?? n.x) + dx, iy: (n.iy ?? n.y) + dy,
    ox: (n.ox ?? n.x) + dx, oy: (n.oy ?? n.y) + dy,
  }))
}

/** Масштабирование узлов из одной рамки в другую. */
export function scaleNodes(nodes, mapX, mapY) {
  return nodes.map((n) => ({
    x: mapX(n.x), y: mapY(n.y),
    ix: mapX(n.ix ?? n.x), iy: mapY(n.iy ?? n.y),
    ox: mapX(n.ox ?? n.x), oy: mapY(n.oy ?? n.y),
  }))
}

/** Что под курсором в режиме правки: узел, его направляющая или ничего. */
export function nodeHit(o, px, py, tolerance) {
  const nodes = nodesOf(o)
  for (let i = nodes.length - 1; i >= 0; i -= 1) {
    const n = nodes[i]
    if (Math.hypot(n.ox - px, n.oy - py) <= tolerance && hasHandles(n)) return { index: i, part: 'out' }
    if (Math.hypot(n.ix - px, n.iy - py) <= tolerance && hasHandles(n)) return { index: i, part: 'in' }
    if (Math.hypot(n.x - px, n.y - py) <= tolerance) return { index: i, part: 'node' }
  }
  return null
}

/** Двинуть узел или его направляющую. mirror — вести вторую ручку зеркально
    (обычное поведение пера; Alt его ломает и даёт угол). */
export function moveNode(o, index, part, x, y, { mirror = true } = {}) {
  const nodes = nodesOf(o)
  const n = nodes[index]
  if (!n) return o
  if (part === 'node') {
    const dx = x - n.x; const dy = y - n.y
    nodes[index] = { x, y, ix: n.ix + dx, iy: n.iy + dy, ox: n.ox + dx, oy: n.oy + dy }
  } else if (part === 'out') {
    nodes[index] = { ...n, ox: x, oy: y, ...(mirror ? { ix: 2 * n.x - x, iy: 2 * n.y - y } : {}) }
  } else {
    nodes[index] = { ...n, ix: x, iy: y, ...(mirror ? { ox: 2 * n.x - x, oy: 2 * n.y - y } : {}) }
  }
  return { ...o, nodes }
}

/** Добавить узел на ближайшем сегменте (клик по контуру). */
export function insertNode(o, px, py) {
  const nodes = nodesOf(o)
  if (nodes.length < 2) return o
  let best = { dist: Infinity, index: 0, point: null }
  const pairs = []
  for (let i = 0; i + 1 < nodes.length; i += 1) pairs.push([i, i + 1])
  if (o.closed && nodes.length > 2) pairs.push([nodes.length - 1, 0])
  for (const [ai, bi] of pairs) {
    const a = nodes[ai]; const b = nodes[bi]
    for (let s = 1; s < FLATTEN_STEPS; s += 1) {
      const p = bezierPoint(a, b, s / FLATTEN_STEPS)
      const dist = Math.hypot(p.x - px, p.y - py)
      if (dist < best.dist) best = { dist, index: ai + 1, point: p }
    }
  }
  if (!best.point) return o
  nodes.splice(best.index, 0, newNode(best.point.x, best.point.y))
  return { ...o, nodes }
}

/** Удалить узел; контур из двух узлов не разбираем — он перестанет быть линией. */
export function removeNode(o, index) {
  const nodes = nodesOf(o)
  if (nodes.length <= 2) return o
  nodes.splice(index, 1)
  return { ...o, nodes }
}

/** Сгладить или заострить узел: направляющие тянутся к соседям либо убираются. */
export function toggleNodeSmooth(o, index) {
  const nodes = nodesOf(o)
  const n = nodes[index]
  if (!n) return o
  if (hasHandles(n)) {
    nodes[index] = newNode(n.x, n.y)
    return { ...o, nodes }
  }
  const prev = nodes[index - 1] || nodes[nodes.length - 1] || n
  const next = nodes[index + 1] || nodes[0] || n
  // Направляющие вдоль отрезка «предыдущий → следующий»: так узел становится
  // гладким, а форма кривой почти не меняется.
  const dx = (next.x - prev.x) / 4
  const dy = (next.y - prev.y) / 4
  nodes[index] = { x: n.x, y: n.y, ix: n.x - dx, iy: n.y - dy, ox: n.x + dx, oy: n.y + dy }
  return { ...o, nodes }
}

/** Фигура → векторный контур: «превратить в кривые» перед правкой узлов. */
export function shapeToVector(o, shapes = {}) {
  const points = outlinePoints(o, shapes)
  const nodes = []
  for (let i = 0; i + 1 < points.length; i += 2) nodes.push(newNode(points[i], points[i + 1]))
  return {
    id: o.id, type: 'vector', layer: o.layer, frame: o.frame, group: o.group,
    nodes, closed: o.type !== 'line' && o.type !== 'arrow' && o.type !== 'path' ? true : !!o.closed,
    color: o.color, fill: o.fill, gradient: o.gradient, width: o.width, opacity: o.opacity,
    effects: o.effects, angle: o.angle, filled: o.filled ?? !!o.fill, solid: o.solid,
  }
}
