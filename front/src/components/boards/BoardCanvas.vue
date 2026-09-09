<script setup>
/* Холст доски: свой движок поверх <canvas> — кисти и перо, фигуры, стрелки,
   надписи, стикеры и картинки на бесконечном полотне с зумом и панорамой,
   плюс приёмы графических редакторов: пиксельный ластик, лассо, пипетка,
   заливка, поворот, выравнивание и покадровая анимация.

   Почему свой, а не библиотека: сцена — обычный JSON (его же понимает сервер:
   ищет по надписям и строит SVG при выгрузке), а рисование живёт в токенах
   темы — цвета берутся из --tag-*, поэтому доска сама перекрашивается в тёмном
   режиме. Правки идут наружу событием update:scene, а история (undo/redo)
   держится в редакторе: снимок сцены на каждый завершённый жест. */
import { computed, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import Textarea from 'primevue/textarea'
import ContextMenu from '@/components/common/ContextMenu.vue'
import { useThemeStore } from '@/stores/theme.js'
import {
  BOOL_OPS, COMMENT_PIN, OBJ, editableLayerIds, hitTest, invalidateColors, isObjectEditable, moveObject,
  newId, normalizeScene, objectAABB, objectBounds, objectCenter, orderedObjects, pointInPolygon,
  polygonPoints, resolveColor, rotateObject, rotatePoint, scaleObject,
} from '@/utils/boardScene.js'
import { drawObject, renderScene } from '@/utils/boardRender.js'
import {
  insertNode, moveNode, newNode, nodeHit, nodesOf, removeNode, shapeToVector, toggleNodeSmooth,
} from '@/utils/boardVector.js'

const props = defineProps({
  scene: { type: Object, required: true },
  tool: { type: String, default: 'select' },
  color: { type: String, default: 'ink' },
  fill: { type: String, default: '' },
  width: { type: Number, default: 4 },
  opacity: { type: Number, default: 1 },
  textSize: { type: Number, default: 18 },
  // Ластик: размер пятна и режим — стирать пиксели или удалять объекты целиком.
  eraserSize: { type: Number, default: 32 },
  eraseMode: { type: String, default: 'pixel' },
  // Многоугольник: число сторон и «звезда» (вершины через одну — внутрь).
  polygonSides: { type: Number, default: 5 },
  polygonStar: { type: Boolean, default: false },
  readOnly: { type: Boolean, default: false },
  // Слой, в который попадают новые объекты (панель слоёв редактора).
  activeLayer: { type: String, default: '' },
  // Кадр анимации: новые объекты попадают в него, чужие кадры не показываются.
  frame: { type: String, default: '' },
  // «Калька»: соседние кадры полупрозрачно — [{ frame, alpha }].
  onion: { type: Array, default: () => [] },
  // Кто я — подписываю комментарии.
  me: { type: Object, default: () => ({}) },
  // Курсоры соавторов: [{ user_id, fio, x, y }].
  peers: { type: Array, default: () => [] },
})

/* update:scene — новая сцена целиком (её сохраняет редактор), ops — только
   изменившиеся объекты: их редактор рассылает соавторам, чтобы параллельные
   правки не затирали друг друга целой сценой. */
const emit = defineEmits([
  'update:scene', 'ops', 'pointer-move', 'select-change', 'comment-open', 'pick-color', 'request-tool',
])

// STICKY_SIZE — сторона липкой заметки в единицах сцены (квадрат, как в Miro).
const STICKY_SIZE = 180
// Ручка поворота висит над рамкой выделения — на этом отступе в пикселях.
const ROTATE_OFFSET = 26
// Привязка при перетаскивании срабатывает на этом расстоянии (в пикселях
// экрана — иначе на мелком масштабе объекты «прилипали» бы издалека).
const SNAP_PX = 6
// Радиус попадания по узлу и ручке контура, тоже в пикселях экрана.
const NODE_PX = 7

const host = ref(null)
const canvas = ref(null)
const editor = ref(null)          // textarea инлайн-ввода надписи

const camera = ref({ x: -80, y: -60, scale: 1 })
const selectedIds = ref([])
const editing = ref(null)         // { id, x, y, value, isNew }
// Контекстное меню холста: пункты зависят от того, попали ли по объекту.
const menu = ref({ visible: false, x: 0, y: 0, onObject: false })
const images = shallowRef(new Map())
// Позиция указателя в координатах сцены — для круга кисти и ластика.
const hover = ref(null)
// Незаконченная кривая: точки, поставленные кликами, и текущий курсор.
const pending = ref(null)
// Перо Безье: узлы будущего контура, пока его рисуют.
const penDraft = ref(null)
// Правка узлов: id контура, с которым сейчас работают, и выбранный узел.
const nodeEdit = ref({ id: '', index: -1 })
// Подсказки-привязки при перетаскивании: отрезки в координатах сцены.
const guides = ref([])

let ctx = null
let dpr = 1
let frameId = 0
let resizeObs = null

// Активный жест: рисование, перетаскивание, рамка выделения или панорама.
let gesture = null
// Пинч двумя пальцами: расстояние и центр на старте.
let pinch = null
// Последний контур лассо — из него делается маска слоя и «стереть внутри».
let lastLasso = null
// Свой буфер обмена: системный требует разрешений и не носит наши объекты.
let clipboard = []

const objects = computed(() => normalizeScene(props.scene).objects)
// Порядок отрисовки: снизу вверх по слоям (скрытые слои и чужие кадры — нет).
const visibleObjects = computed(() => orderedObjects(props.scene, { frame: props.frame || null }))
// Трогать можно только объекты видимых и незаблокированных слоёв.
const editableLayers = computed(() => editableLayerIds(props.scene))
const canEdit = computed(() => !props.readOnly)
// Круг под курсором показывают инструменты с «пятном»: кисти и ластик.
const brushTools = new Set(['pencil', 'marker', 'brush'])
const showBrushCursor = computed(() => canEdit.value
  && (brushTools.has(props.tool) || (props.tool === 'eraser' && props.eraseMode === 'pixel')))

// Слой для новых объектов: заданный редактором либо первый доступный.
function targetLayer() {
  const layers = normalizeScene(props.scene).layers
  if (props.activeLayer && editableLayers.value.has(props.activeLayer)) return props.activeLayer
  return (layers.find((l) => l.visible && !l.locked) || layers[0]).id
}

// Поля, общие для любого нового объекта: слой и кадр анимации.
function baseFields() {
  return props.frame ? { layer: targetLayer(), frame: props.frame } : { layer: targetLayer() }
}

/** Объект под точкой — сверху вниз, только по доступным слоям. */
function objectAt(px, py, tolerance = 8) {
  const list = visibleObjects.value.filter((o) => isObjectEditable(o, editableLayers.value))
  for (let i = list.length - 1; i >= 0; i--) {
    if (hitTest(list[i], px, py, tolerance / camera.value.scale)) return list[i]
  }
  return null
}

/** Выделение с учётом групп: клик по объекту группы берёт всю группу. */
function selectionFor(o) {
  if (!o) return []
  if (!o.group) return [o.id]
  return objects.value.filter((x) => x.group === o.group).map((x) => x.id)
}

// ── Координаты ───────────────────────────────────────────────────

function toScene(clientX, clientY) {
  const rect = canvas.value.getBoundingClientRect()
  return {
    x: (clientX - rect.left) / camera.value.scale + camera.value.x,
    y: (clientY - rect.top) / camera.value.scale + camera.value.y,
  }
}

function toScreen(pt) {
  return {
    x: (pt.x - camera.value.x) * camera.value.scale,
    y: (pt.y - camera.value.y) * camera.value.scale,
  }
}

// ── Отрисовка ────────────────────────────────────────────────────

function requestDraw() {
  if (frameId) return
  frameId = requestAnimationFrame(() => {
    frameId = 0
    draw()
  })
}

function draw() {
  const el = canvas.value
  if (!el || !ctx) return
  const { width, height } = el.getBoundingClientRect()
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, width, height)

  renderScene(ctx, props.scene, {
    width, height, camera: camera.value, images: images.value,
    frame: props.frame || null, onion: props.onion,
  })

  // Незавершённый объект текущего жеста рисуем поверх — он ещё не в сцене.
  if (gesture?.draft) {
    ctx.save()
    ctx.scale(camera.value.scale, camera.value.scale)
    ctx.translate(-camera.value.x, -camera.value.y)
    // Стирающий штрих на холсте показываем «мелом»: destination-out поверх
    // готовой картинки выел бы и фон, и соседние слои.
    drawObject(ctx, gesture.draft.erase
      ? { ...gesture.draft, erase: false, color: '__ghost', opacity: 0.35 }
      : gesture.draft, images.value)
    ctx.restore()
  }
  drawPending(ctx)
  drawPen(ctx)
  drawNodes(ctx)
  drawGuides(ctx)
  drawSelection(ctx)
  drawMarquee(ctx)
  drawLasso(ctx)
  drawBrushCursor(ctx)
  drawPeers(ctx)
}

/** Незаконченная кривая: поставленные точки и «резинка» до курсора. */
function drawPending(c) {
  if (!pending.value) return
  const pts = pending.value.points
  c.save()
  c.strokeStyle = resolveColor(props.color)
  c.lineWidth = 1.5
  c.setLineDash([5, 4])
  c.beginPath()
  const first = toScreen({ x: pts[0], y: pts[1] })
  c.moveTo(first.x, first.y)
  for (let i = 2; i + 1 < pts.length; i += 2) {
    const p = toScreen({ x: pts[i], y: pts[i + 1] })
    c.lineTo(p.x, p.y)
  }
  if (hover.value) {
    const p = toScreen(hover.value)
    c.lineTo(p.x, p.y)
  }
  c.stroke()
  c.setLineDash([])
  c.fillStyle = resolveColor('__accent', '--color-primary')
  for (let i = 0; i + 1 < pts.length; i += 2) {
    const p = toScreen({ x: pts[i], y: pts[i + 1] })
    c.fillRect(p.x - 3, p.y - 3, 6, 6)
  }
  c.restore()
}

/** Перо Безье: поставленные узлы, их направляющие и «резинка» до курсора. */
function drawPen(c) {
  const draft = penDraft.value
  if (!draft?.nodes.length) return
  const accent = resolveColor('__accent', '--color-primary')
  c.save()
  c.strokeStyle = accent
  c.fillStyle = accent
  c.lineWidth = 1.5

  // Сам контур строим тем же кодом, что и готовый объект, — предпросмотр не
  // должен отличаться от результата.
  c.save()
  c.scale(camera.value.scale, camera.value.scale)
  c.translate(-camera.value.x, -camera.value.y)
  drawObject(c, { type: OBJ.vector, nodes: draft.nodes, color: props.color, width: props.width }, images.value)
  c.restore()

  c.setLineDash([5, 4])
  const last = draft.nodes[draft.nodes.length - 1]
  if (hover.value && last) {
    const a = toScreen({ x: last.x, y: last.y })
    const b = toScreen(hover.value)
    c.beginPath()
    c.moveTo(a.x, a.y)
    c.lineTo(b.x, b.y)
    c.stroke()
  }
  c.setLineDash([])
  for (const n of draft.nodes) {
    const p = toScreen(n)
    c.fillRect(p.x - 3, p.y - 3, 6, 6)
    for (const h of [{ x: n.ix, y: n.iy }, { x: n.ox, y: n.oy }]) {
      if (h.x === n.x && h.y === n.y) continue
      const hp = toScreen(h)
      c.beginPath(); c.moveTo(p.x, p.y); c.lineTo(hp.x, hp.y); c.stroke()
      c.beginPath(); c.arc(hp.x, hp.y, 3, 0, Math.PI * 2); c.fill()
    }
  }
  c.restore()
}

/** Правка узлов выбранного контура: сами узлы и их направляющие. */
function drawNodes(c) {
  const target = editingVector()
  if (!target) return
  const accent = resolveColor('__accent', '--color-primary')
  const surface = resolveColor('chalk', '--color-surface')
  c.save()
  c.strokeStyle = accent
  c.lineWidth = 1.5
  nodesOf(target).forEach((n, i) => {
    const p = toScreen(n)
    for (const h of [{ x: n.ix, y: n.iy }, { x: n.ox, y: n.oy }]) {
      if (h.x === n.x && h.y === n.y) continue
      const hp = toScreen(h)
      c.beginPath(); c.moveTo(p.x, p.y); c.lineTo(hp.x, hp.y); c.stroke()
      c.fillStyle = surface
      c.beginPath(); c.arc(hp.x, hp.y, 4, 0, Math.PI * 2); c.fill(); c.stroke()
    }
    c.fillStyle = i === nodeEdit.value.index ? accent : surface
    c.fillRect(p.x - 4, p.y - 4, 8, 8)
    c.strokeRect(p.x - 4, p.y - 4, 8, 8)
  })
  c.restore()
}

/** Направляющие привязки: тонкие линии по совпавшим краям и центрам. */
function drawGuides(c) {
  if (!guides.value.length) return
  c.save()
  c.strokeStyle = resolveColor('red', '--tag-red-accent')
  c.lineWidth = 1
  c.setLineDash([4, 4])
  for (const g of guides.value) {
    const a = toScreen({ x: g.x1, y: g.y1 })
    const b = toScreen({ x: g.x2, y: g.y2 })
    c.beginPath(); c.moveTo(a.x, a.y); c.lineTo(b.x, b.y); c.stroke()
  }
  c.restore()
}

function drawSelection(c) {
  if (!selectedIds.value.length) return
  const accent = resolveColor('__accent', '--color-primary')
  c.save()
  c.strokeStyle = accent
  c.lineWidth = 1.5
  c.setLineDash([6, 4])
  for (const o of objects.value) {
    if (!selectedIds.value.includes(o.id)) continue
    // Повёрнутый объект обводим по его собственной рамке — иначе рамка не
    // совпадает с тем, что видно на холсте.
    const b = objectBounds(o)
    const corners = [[b.x, b.y], [b.x + b.w, b.y], [b.x + b.w, b.y + b.h], [b.x, b.y + b.h]]
    const center = objectCenter(o)
    c.beginPath()
    corners.forEach(([x, y], i) => {
      const p = toScreen(o.angle ? rotatePoint(x, y, center.x, center.y, o.angle) : { x, y })
      if (i === 0) c.moveTo(p.x, p.y)
      else c.lineTo(p.x, p.y)
    })
    c.closePath()
    c.stroke()
  }
  const box = selectionBox()
  if (box) {
    c.setLineDash([])
    c.fillStyle = accent
    // Ручка масштабирования — в правом нижнем углу общей рамки выделения.
    const handle = toScreen({ x: box.x + box.w, y: box.y + box.h })
    c.fillRect(handle.x - 4, handle.y - 4, 10, 10)
    // Ручка поворота — над серединой верхней грани.
    const top = toScreen({ x: box.x + box.w / 2, y: box.y })
    c.beginPath()
    c.moveTo(top.x, top.y)
    c.lineTo(top.x, top.y - ROTATE_OFFSET)
    c.stroke()
    c.beginPath()
    c.arc(top.x, top.y - ROTATE_OFFSET, 6, 0, Math.PI * 2)
    c.fill()
  }
  c.restore()
}

function drawMarquee(c) {
  if (gesture?.kind !== 'marquee') return
  const a = toScreen(gesture.from)
  const b = toScreen(gesture.to)
  c.save()
  c.strokeStyle = resolveColor('__accent', '--color-primary')
  c.fillStyle = resolveColor('__accent', '--color-primary')
  c.globalAlpha = 0.12
  c.fillRect(Math.min(a.x, b.x), Math.min(a.y, b.y), Math.abs(b.x - a.x), Math.abs(b.y - a.y))
  c.globalAlpha = 1
  c.setLineDash([4, 4])
  c.strokeRect(Math.min(a.x, b.x), Math.min(a.y, b.y), Math.abs(b.x - a.x), Math.abs(b.y - a.y))
  c.restore()
}

function drawLasso(c) {
  if (gesture?.kind !== 'lasso' || gesture.points.length < 4) return
  c.save()
  c.strokeStyle = resolveColor('__accent', '--color-primary')
  c.fillStyle = resolveColor('__accent', '--color-primary')
  c.lineWidth = 1.5
  c.setLineDash([5, 4])
  c.beginPath()
  const start = toScreen({ x: gesture.points[0], y: gesture.points[1] })
  c.moveTo(start.x, start.y)
  for (let i = 2; i + 1 < gesture.points.length; i += 2) {
    const p = toScreen({ x: gesture.points[i], y: gesture.points[i + 1] })
    c.lineTo(p.x, p.y)
  }
  c.closePath()
  c.globalAlpha = 0.1
  c.fill()
  c.globalAlpha = 1
  c.stroke()
  c.restore()
}

/** Круг кисти или ластика под курсором: показывает реальный размер пятна. */
function drawBrushCursor(c) {
  if (!showBrushCursor.value || !hover.value || gesture?.kind === 'pan') return
  const size = props.tool === 'eraser' ? props.eraserSize : brushWidth()
  const p = toScreen(hover.value)
  c.save()
  c.strokeStyle = resolveColor('__accent', '--color-primary')
  c.lineWidth = 1
  c.globalAlpha = 0.8
  c.beginPath()
  c.arc(p.x, p.y, Math.max(2, (size * camera.value.scale) / 2), 0, Math.PI * 2)
  c.stroke()
  c.restore()
}

function drawPeers(c) {
  if (!props.peers.length) return
  c.save()
  for (const peer of props.peers) {
    if (peer.x == null) continue
    const p = toScreen(peer)
    const tint = resolveColor(peerColor(peer.user_id))
    c.fillStyle = tint
    c.beginPath()
    c.arc(p.x, p.y, 5, 0, Math.PI * 2)
    c.fill()
    if (peer.fio) {
      c.font = '12px Inter, system-ui, sans-serif'
      const label = peer.fio.split(' ')[0]
      const w = c.measureText(label).width + 12
      c.globalAlpha = 0.9
      c.fillRect(p.x + 8, p.y - 10, w, 20)
      c.globalAlpha = 1
      c.fillStyle = resolveColor('chalk', '--color-surface')
      c.fillText(label, p.x + 14, p.y + 4)
    }
  }
  c.restore()
}

// Цвет курсора соавтора — детерминированно от его id (у каждого свой).
const PEER_COLORS = ['blue', 'green', 'violet', 'orange', 'teal', 'pink']
function peerColor(id) {
  return PEER_COLORS[Math.abs(Number(id) || 0) % PEER_COLORS.length]
}

/** Общая рамка выделения — с учётом поворота объектов. */
function selectionBox() {
  const picked = objects.value.filter((o) => selectedIds.value.includes(o.id))
  if (!picked.length) return null
  let minX = Infinity; let minY = Infinity; let maxX = -Infinity; let maxY = -Infinity
  for (const o of picked) {
    const b = objectAABB(o)
    minX = Math.min(minX, b.x); minY = Math.min(minY, b.y)
    maxX = Math.max(maxX, b.x + b.w); maxY = Math.max(maxY, b.y + b.h)
  }
  return { x: minX, y: minY, w: maxX - minX, h: maxY - minY }
}

// ── Изменение сцены ──────────────────────────────────────────────

function commit(objectsNext, { background, layers, animation, ops } = {}) {
  emit('update:scene', {
    ...normalizeScene(props.scene),
    ...(background ? { background } : {}),
    ...(layers ? { layers } : {}),
    ...(animation !== undefined ? { animation } : {}),
    objects: objectsNext,
  })
  if (ops?.length) emit('ops', ops)
}

// upsert/remove — операции для соавторов: адресные, поэтому одновременная
// работа над разными объектами не приводит к затиранию чужих правок.
const upsertOp = (list) => ({ kind: 'upsert', objects: list })
const removeOp = (ids) => ({ kind: 'remove', ids })

function addObject(o) {
  commit([...objects.value, o], { ops: [upsertOp([o])] })
}

function updateObject(id, patch) {
  const next = objects.value.map((o) => (o.id === id ? { ...o, ...patch } : o))
  commit(next, { ops: [upsertOp(next.filter((o) => o.id === id))] })
}

function removeSelected() {
  if (!selectedIds.value.length) return
  const ids = [...selectedIds.value]
  commit(objects.value.filter((o) => !ids.includes(o.id)), { ops: [removeOp(ids)] })
  setSelection([])
}

/** Сгруппировать выделенное: клик по любому объекту берёт всю группу. */
function groupSelected() {
  if (selectedIds.value.length < 2) return
  const group = newId()
  const next = objects.value.map((o) => (selectedIds.value.includes(o.id) ? { ...o, group } : o))
  commit(next, { ops: [upsertOp(next.filter((o) => selectedIds.value.includes(o.id)))] })
}

function ungroupSelected() {
  const next = objects.value.map((o) => {
    if (!selectedIds.value.includes(o.id) || !o.group) return o
    const { group, ...rest } = o
    return rest
  })
  commit(next, { ops: [upsertOp(next.filter((o) => selectedIds.value.includes(o.id)))] })
}

/** Перенести выделенное в другой слой (drag в панели слоёв не нужен). */
function moveSelectionToLayer(layerId) {
  if (!selectedIds.value.length) return
  const next = objects.value.map((o) => (selectedIds.value.includes(o.id) ? { ...o, layer: layerId } : o))
  commit(next, { ops: [upsertOp(next.filter((o) => selectedIds.value.includes(o.id)))] })
}

/** Дубликат выделенного со сдвигом — как «копировать/вставить» на месте. */
function duplicateSelected() {
  const picked = objects.value.filter((o) => selectedIds.value.includes(o.id))
  if (!picked.length) return
  const copies = picked.map((o) => ({ ...moveObject(o, 24, 24), id: newId() }))
  commit([...objects.value, ...copies], { ops: [upsertOp(copies)] })
  setSelection(copies.map((o) => o.id))
}

/** Порядок отрисовки = порядок в массиве: наверх — в конец, вниз — в начало. */
function reorderSelected(toFront) {
  if (!selectedIds.value.length) return
  const picked = objects.value.filter((o) => selectedIds.value.includes(o.id))
  const rest = objects.value.filter((o) => !selectedIds.value.includes(o.id))
  commit(toFront ? [...rest, ...picked] : [...picked, ...rest], { ops: [upsertOp(picked)] })
}

function selectAll() {
  setSelection(visibleObjects.value.map((o) => o.id))
  requestDraw()
}

function setSelection(ids) {
  selectedIds.value = ids
  emit('select-change', ids)
}

/** Выделенные объекты — наружу (панель свойств тулбара). */
function selectedObjects() {
  return objects.value.filter((o) => selectedIds.value.includes(o.id))
}

// ── Контуры: перо и правка узлов ─────────────────────────────────

/** Контур, узлы которого сейчас правят (инструмент «Узлы» + выделение). */
function editingVector() {
  if (props.tool !== 'nodes') return null
  const picked = objects.value.find((o) => o.id === nodeEdit.value.id && o.type === OBJ.vector)
  if (picked) return picked
  const selected = selectedObjects().find((o) => o.type === OBJ.vector)
  return selected || null
}

function penHit(draft, pt) {
  const tol = NODE_PX / camera.value.scale
  const first = draft.nodes[0]
  return first && draft.nodes.length >= 2 && Math.hypot(first.x - pt.x, first.y - pt.y) <= tol
}

function startPen(pt, e) {
  const draft = penDraft.value
  if (!draft) {
    penDraft.value = { nodes: [newNode(pt.x, pt.y)] }
    gesture = { kind: 'pen-handle', index: 0 }
    return
  }
  // Клик по первой точке замыкает контур — как в векторных редакторах.
  if (penHit(draft, pt)) {
    finishPen(true)
    return
  }
  draft.nodes.push(newNode(pt.x, pt.y))
  gesture = { kind: 'pen-handle', index: draft.nodes.length - 1, alt: e.altKey }
}

/** Завершить контур: замкнутый заливается, открытый остаётся линией. */
function finishPen(closed) {
  const draft = penDraft.value
  penDraft.value = null
  if (!draft || draft.nodes.length < 2) return
  const o = {
    id: newId(), type: OBJ.vector, ...baseFields(),
    nodes: draft.nodes, closed,
    color: props.color, width: props.width, opacity: props.opacity,
    ...(props.fill ? { fill: props.fill, filled: closed } : {}),
  }
  addObject(o)
  setSelection([o.id])
  nodeEdit.value = { id: o.id, index: -1 }
  requestDraw()
}

function startNodeEdit(pt, e) {
  const target = editingVector()
  if (!target) {
    // Инструмент включён, а контур не выбран — выбираем как обычно.
    startSelect(pt, e)
    return
  }
  const tol = NODE_PX / camera.value.scale
  const hit = nodeHit(target, pt.x, pt.y, tol)
  if (hit) {
    nodeEdit.value = { id: target.id, index: hit.index }
    // Alt по узлу переключает «гладкий ↔ угловой», как в Illustrator.
    if (e.altKey && hit.part === 'node') {
      updateObject(target.id, toggleNodeSmooth(target, hit.index))
      return
    }
    gesture = { kind: 'node', id: target.id, index: hit.index, part: hit.part, alt: e.altKey }
    return
  }
  // Клик по самому контуру добавляет узел, мимо — выбираем другой объект.
  if (hitTest(target, pt.x, pt.y, tol)) {
    updateObject(target.id, insertNode(target, pt.x, pt.y))
    return
  }
  startSelect(pt, e)
  const picked = selectedObjects().find((o) => o.type === OBJ.vector)
  nodeEdit.value = { id: picked?.id || '', index: -1 }
}

/** Превратить фигуры выделения в векторные контуры — «в кривые». */
function convertToVector() {
  const picked = selectedObjects().filter((o) => o.type !== OBJ.vector && o.type !== OBJ.text
    && o.type !== OBJ.image && o.type !== OBJ.comment && o.type !== OBJ.sticky)
  if (!picked.length) return
  const byId = new Map(picked.map((o) => [o.id, shapeToVector(o, { polygonPoints })]))
  const next = objects.value.map((o) => byId.get(o.id) || o)
  commit(next, { ops: [upsertOp([...byId.values()])] })
  requestDraw()
}

// ── Булевы операции и маски ──────────────────────────────────────

/* Булева группа — как в Figma: участники остаются отдельными объектами, а
   операция считается при отрисовке. Нижний объект выделения становится базой:
   именно из него вычитают и с ним пересекают. */
function booleanSelection(op) {
  const picked = objects.value.filter((o) => selectedIds.value.includes(o.id))
  if (picked.length < 2) return
  const id = newId()
  const byId = new Map(picked.map((o, i) => [o.id, { ...o, bool: id, boolOp: i === 0 ? 'union' : op }]))
  const next = objects.value.map((o) => byId.get(o.id) || o)
  commit(next, { ops: [upsertOp([...byId.values()])] })
  requestDraw()
}

/** Разобрать булеву группу обратно на фигуры. */
function releaseBoolean() {
  const picked = selectedObjects().filter((o) => o.bool)
  if (!picked.length) return
  const ids = new Set(picked.map((o) => o.bool))
  const changed = []
  const next = objects.value.map((o) => {
    if (!o.bool || !ids.has(o.bool)) return o
    const { bool, boolOp, ...rest } = o
    changed.push(rest)
    return rest
  })
  commit(next, { ops: [upsertOp(changed)] })
  requestDraw()
}

/* Объект-маска: нижний объект выделения показывает всё, что лежит над ним в
   слое, только внутри своей формы (в Figma это «использовать как маску»). */
function toggleMaskObject() {
  const picked = objects.value.filter((o) => selectedIds.value.includes(o.id))
  if (!picked.length) return
  const target = picked[0]
  const next = objects.value.map((o) => (o.id === target.id ? { ...o, maskObject: !o.maskObject } : o))
  commit(next, { ops: [upsertOp(next.filter((o) => o.id === target.id))] })
  requestDraw()
}

/** Пустить надпись по выделенному контуру (и снять привязку обратно). */
function attachTextToPath() {
  const picked = selectedObjects()
  const text = picked.find((o) => o.type === OBJ.text)
  const path = picked.find((o) => o.type === OBJ.vector || o.type === OBJ.path
    || o.type === OBJ.ellipse || o.type === OBJ.rect || o.type === OBJ.polygon)
  if (!text) return
  const patch = text.pathRef ? { pathRef: undefined, pathOffset: undefined } : { pathRef: path?.id, pathOffset: 0 }
  if (!text.pathRef && !path) return
  updateObject(text.id, patch)
  requestDraw()
}

// ── Привязка при перетаскивании ──────────────────────────────────

/* Сдвиг с примагничиванием к краям и центрам соседей — та же помощь, что
   даёт Figma. Считаем по ОБЩЕЙ рамке выделения: иначе группа объектов липла
   бы каждым своим краем и дёргалась. */
function snapMove(dx, dy) {
  const box = selectionBox()
  if (!box) return { dx, dy }
  const tol = SNAP_PX / camera.value.scale
  const moved = { x: box.x + dx, y: box.y + dy, w: box.w, h: box.h }
  const lines = []
  let bestX = null
  let bestY = null

  for (const other of visibleObjects.value) {
    if (selectedIds.value.includes(other.id)) continue
    const b = objectAABB(other)
    const xs = [[moved.x, b.x], [moved.x + moved.w / 2, b.x + b.w / 2], [moved.x + moved.w, b.x + b.w],
      [moved.x, b.x + b.w], [moved.x + moved.w, b.x]]
    for (const [from, to] of xs) {
      const d = to - from
      if (Math.abs(d) <= tol && (bestX === null || Math.abs(d) < Math.abs(bestX.d))) bestX = { d, at: to, other: b }
    }
    const ys = [[moved.y, b.y], [moved.y + moved.h / 2, b.y + b.h / 2], [moved.y + moved.h, b.y + b.h],
      [moved.y, b.y + b.h], [moved.y + moved.h, b.y]]
    for (const [from, to] of ys) {
      const d = to - from
      if (Math.abs(d) <= tol && (bestY === null || Math.abs(d) < Math.abs(bestY.d))) bestY = { d, at: to, other: b }
    }
  }

  if (bestX) {
    const top = Math.min(moved.y, bestX.other.y)
    const bottom = Math.max(moved.y + moved.h, bestX.other.y + bestX.other.h)
    lines.push({ x1: bestX.at, y1: top, x2: bestX.at, y2: bottom })
  }
  if (bestY) {
    const left = Math.min(moved.x, bestY.other.x)
    const right = Math.max(moved.x + moved.w, bestY.other.x + bestY.other.w)
    lines.push({ x1: left, y1: bestY.at, x2: right, y2: bestY.at })
  }
  guides.value = lines
  return { dx: dx + (bestX?.d || 0), dy: dy + (bestY?.d || 0) }
}

// ── Копирование и вставка ────────────────────────────────────────

function copySelection(cut = false) {
  const picked = selectedObjects()
  if (!picked.length) return
  clipboard = picked.map((o) => JSON.parse(JSON.stringify(o)))
  if (cut) removeSelected()
}

function pasteClipboard() {
  if (!clipboard.length) return
  // Вставляем в ТЕКУЩИЙ слой и кадр: буфер мог пережить и то, и другое.
  const copies = clipboard.map((o) => ({ ...moveObject(o, 24, 24), ...baseFields(), id: newId() }))
  commit([...objects.value, ...copies], { ops: [upsertOp(copies)] })
  setSelection(copies.map((o) => o.id))
  requestDraw()
}

// ── Выравнивание ─────────────────────────────────────────────────

/** Выравнивание и распределение выделенного по общей рамке (панель Illustrator). */
function alignSelection(kind) {
  const picked = selectedObjects()
  if (picked.length < 2) return
  const box = selectionBox()
  const byId = new Map()

  if (kind === 'distribute-h' || kind === 'distribute-v') {
    const horizontal = kind === 'distribute-h'
    const sorted = [...picked].sort((a, b) => (horizontal
      ? objectAABB(a).x - objectAABB(b).x
      : objectAABB(a).y - objectAABB(b).y))
    const sizes = sorted.reduce((sum, o) => sum + (horizontal ? objectAABB(o).w : objectAABB(o).h), 0)
    const gap = (Math.max(0, (horizontal ? box.w : box.h) - sizes)) / (sorted.length - 1)
    let cursor = horizontal ? box.x : box.y
    for (const o of sorted) {
      const b = objectAABB(o)
      byId.set(o.id, horizontal ? moveObject(o, cursor - b.x, 0) : moveObject(o, 0, cursor - b.y))
      cursor += (horizontal ? b.w : b.h) + gap
    }
  } else {
    for (const o of picked) {
      const b = objectAABB(o)
      const shift = {
        left: [box.x - b.x, 0],
        right: [box.x + box.w - (b.x + b.w), 0],
        hcenter: [box.x + box.w / 2 - (b.x + b.w / 2), 0],
        top: [0, box.y - b.y],
        bottom: [0, box.y + box.h - (b.y + b.h)],
        vcenter: [0, box.y + box.h / 2 - (b.y + b.h / 2)],
      }[kind]
      if (!shift) return
      byId.set(o.id, moveObject(o, shift[0], shift[1]))
    }
  }

  const next = objects.value.map((o) => byId.get(o.id) || o)
  commit(next, { ops: [upsertOp([...byId.values()])] })
  requestDraw()
}

// ── Лассо: маска слоя и стирание области ─────────────────────────

/** Маска активного слоя по последнему контуру лассо (обратная — «вырезать»). */
function maskFromLasso(invert = false) {
  if (!lastLasso || lastLasso.length < 6) return false
  const scene = normalizeScene(props.scene)
  const layers = scene.layers.map((l) => (l.id === targetLayer() ? { ...l, mask: { points: [...lastLasso], invert } } : l))
  commit(objects.value, { layers })
  return true
}

/** Снять маску активного слоя. */
function clearLayerMask() {
  const scene = normalizeScene(props.scene)
  commit(objects.value, { layers: scene.layers.map((l) => (l.id === targetLayer() ? { ...l, mask: null } : l)) })
}

/** Стереть внутри контура лассо — залитый стирающий объект в активном слое. */
function eraseInLasso() {
  if (!lastLasso || lastLasso.length < 6) return false
  const o = {
    id: newId(), type: OBJ.path, ...baseFields(),
    points: [...lastLasso], closed: true, filled: true, erase: true, width: 0,
  }
  addObject(o)
  setSelection([])
  requestDraw()
  return true
}

const hasLasso = () => !!(lastLasso && lastLasso.length >= 6)

// ── Указатель ────────────────────────────────────────────────────

/* Толщина кисти: у маркера пятно шире, у остальных — как в тулбаре. Ноль
   означает «без обводки» у фигур, но кисть с нулевой толщиной оставляла бы
   невидимый штрих — поэтому здесь минимум единица. */
function brushWidth() {
  const base = Math.max(1, props.width)
  return props.tool === 'marker' ? base * 3 : base
}

// Инструменты, которым нужен активный контур: они не сбрасывают выделение.
const KEEP_SELECTION = new Set(['select', 'lasso', 'nodes'])

function onPointerDown(e) {
  // Правая кнопка — только контекстное меню (иначе ПКМ начинала бы штрих).
  if (e.button === 2) return
  if (editing.value) commitEditing()
  canvas.value.setPointerCapture?.(e.pointerId)
  const pt = toScene(e.clientX, e.clientY)

  // Средняя кнопка, пробел и инструмент «рука» — всегда панорама.
  if (e.button === 1 || props.tool === 'pan' || e.shiftKey) {
    gesture = { kind: 'pan', from: { x: e.clientX, y: e.clientY }, camera: { ...camera.value } }
    return
  }
  if (!canEdit.value) {
    gesture = { kind: 'pan', from: { x: e.clientX, y: e.clientY }, camera: { ...camera.value } }
    return
  }

  switch (props.tool) {
    case 'select':
      startSelect(pt, e)
      break
    case 'lasso':
      gesture = { kind: 'lasso', points: [pt.x, pt.y], additive: e.ctrlKey || e.metaKey }
      break
    case 'eyedropper':
      pickColorAt(e.clientX, e.clientY)
      break
    case 'fill':
      fillAt(pt)
      break
    case 'eraser':
      if (props.eraseMode === 'object') {
        gesture = { kind: 'erase' }
        eraseAt(pt)
      } else {
        gesture = {
          kind: 'draw',
          draft: {
            id: newId(), type: OBJ.path, ...baseFields(), x: pt.x, y: pt.y,
            points: [pt.x, pt.y], erase: true, smooth: true, width: props.eraserSize,
          },
        }
      }
      break
    case 'text':
      startTextEditing(pt)
      break
    case 'sticky':
      addSticky(pt)
      break
    case 'comment':
      addComment(pt)
      break
    case 'curve':
      addCurvePoint(pt)
      break
    case 'pen':
      startPen(pt, e)
      break
    case 'nodes':
      startNodeEdit(pt, e)
      break
    case 'pencil':
    case 'brush':
    case 'marker':
      gesture = {
        kind: 'draw',
        draft: {
          id: newId(), type: OBJ.path, ...baseFields(), x: pt.x, y: pt.y,
          points: [pt.x, pt.y], color: props.color,
          width: brushWidth(),
          smooth: props.tool === 'brush',
          opacity: props.tool === 'marker' ? 0.35 : props.opacity,
        },
      }
      break
    default:
      startShape(pt)
      break
  }
  requestDraw()
}

function startSelect(pt, e) {
  const box = selectionBox()
  if (box) {
    const screen = toScreen(pt)
    const handle = toScreen({ x: box.x + box.w, y: box.y + box.h })
    if (Math.abs(screen.x - handle.x) < 10 && Math.abs(screen.y - handle.y) < 10) {
      gesture = { kind: 'scale', from: box, origin: { ...box }, snapshot: selectedObjects() }
      return
    }
    const top = toScreen({ x: box.x + box.w / 2, y: box.y })
    if (Math.hypot(screen.x - top.x, screen.y - (top.y - ROTATE_OFFSET)) < 12) {
      const pivot = { x: box.x + box.w / 2, y: box.y + box.h / 2 }
      gesture = {
        kind: 'rotate', pivot, snapshot: selectedObjects(),
        start: Math.atan2(pt.y - pivot.y, pt.x - pivot.x),
      }
      return
    }
  }
  const hit = objectAt(pt.x, pt.y)
  if (hit) {
    const group = selectionFor(hit)
    const ids = e.ctrlKey || e.metaKey
      ? (selectedIds.value.includes(hit.id)
        ? selectedIds.value.filter((id) => !group.includes(id))
        : [...selectedIds.value, ...group])
      : (selectedIds.value.includes(hit.id) ? selectedIds.value : group)
    setSelection(ids)
    gesture = { kind: 'move', last: pt }
    return
  }
  setSelection([])
  gesture = { kind: 'marquee', from: pt, to: pt }
}

/** Липкая заметка: сразу открывает ввод — стикер без текста бесполезен. */
function addSticky(pt) {
  const o = {
    id: newId(), type: OBJ.sticky, ...baseFields(),
    x: pt.x, y: pt.y, w: STICKY_SIZE, h: STICKY_SIZE,
    color: props.color === 'ink' ? 'amber' : props.color, text: '',
  }
  addObject(o)
  setSelection([o.id])
  openEditor(o, { x: o.x + 10, y: o.y + 10 })
}

/** Булавка комментария: текст пишется в попапе редактора (там же и ответы). */
function addComment(pt) {
  const o = {
    id: newId(), type: OBJ.comment, ...baseFields(),
    x: pt.x - COMMENT_PIN / 2, y: pt.y - COMMENT_PIN / 2,
    color: 'amber', text: '', resolved: false, replies: [],
    author_id: props.me?.id ?? null, author: props.me?.fio || '',
    created_at: new Date().toISOString(),
  }
  addObject(o)
  setSelection([o.id])
  emit('comment-open', o)
}

/** Кривая по точкам: клик ставит узел, клик по первому узлу замыкает контур. */
function addCurvePoint(pt) {
  if (!pending.value) {
    pending.value = { points: [pt.x, pt.y] }
    return
  }
  const pts = pending.value.points
  const near = Math.hypot(pts[0] - pt.x, pts[1] - pt.y) < 10 / camera.value.scale
  if (near && pts.length >= 6) {
    finishCurve(true)
    return
  }
  pts.push(pt.x, pt.y)
}

function finishCurve(closed = false) {
  const state = pending.value
  pending.value = null
  if (!state) return
  const points = [...state.points]
  // Двойной клик «завершить» — это два обычных клика, и второй уже поставил
  // точку там же: без чистки в контуре оставался бы дубль последнего узла.
  if (points.length >= 4
    && points[points.length - 2] === points[points.length - 4]
    && points[points.length - 1] === points[points.length - 3]) {
    points.length -= 2
  }
  if (points.length < 4) return
  addObject({
    id: newId(), type: OBJ.path, ...baseFields(),
    points, color: props.color, width: props.width, opacity: props.opacity,
    smooth: true, closed, filled: closed && !!props.fill, ...(props.fill ? { fill: props.fill } : {}),
  })
  requestDraw()
}

function startShape(pt) {
  const type = {
    line: OBJ.line, arrow: OBJ.arrow, rect: OBJ.rect,
    ellipse: OBJ.ellipse, diamond: OBJ.diamond, polygon: OBJ.polygon,
  }[props.tool]
  if (!type) return
  const base = {
    id: newId(), type, ...baseFields(), color: props.color, width: props.width,
    fill: props.fill, opacity: props.opacity,
    ...(type === OBJ.polygon ? { sides: props.polygonSides, star: props.polygonStar } : {}),
  }
  gesture = {
    kind: 'shape',
    draft: type === OBJ.line || type === OBJ.arrow
      ? { ...base, x: pt.x, y: pt.y, x2: pt.x, y2: pt.y }
      : { ...base, x: pt.x, y: pt.y, w: 0, h: 0 },
    origin: pt,
  }
}

function onPointerMove(e) {
  const pt = toScene(e.clientX, e.clientY)
  hover.value = pt
  emit('pointer-move', pt)
  if (!gesture) {
    // Круг кисти и «резинка» кривой ходят за курсором и без нажатия.
    if (showBrushCursor.value || pending.value) requestDraw()
    return
  }

  switch (gesture.kind) {
    case 'pan': {
      const dx = (e.clientX - gesture.from.x) / camera.value.scale
      const dy = (e.clientY - gesture.from.y) / camera.value.scale
      camera.value = { ...camera.value, x: gesture.camera.x - dx, y: gesture.camera.y - dy }
      break
    }
    case 'draw':
      gesture.draft.points.push(pt.x, pt.y)
      break
    case 'lasso':
      gesture.points.push(pt.x, pt.y)
      break
    case 'shape': {
      const d = gesture.draft
      if (d.type === OBJ.line || d.type === OBJ.arrow) {
        gesture.draft = { ...d, x2: pt.x, y2: pt.y }
      } else {
        gesture.draft = {
          ...d,
          x: Math.min(gesture.origin.x, pt.x),
          y: Math.min(gesture.origin.y, pt.y),
          w: Math.abs(pt.x - gesture.origin.x),
          h: Math.abs(pt.y - gesture.origin.y),
        }
      }
      break
    }
    case 'move': {
      const raw = { dx: pt.x - gesture.last.x, dy: pt.y - gesture.last.y }
      // Привязку выключает Alt — иногда объект нужно поставить именно «мимо».
      const snapped = e.altKey ? raw : snapMove(raw.dx, raw.dy)
      if (e.altKey) guides.value = []
      gesture.last = { x: pt.x + (snapped.dx - raw.dx), y: pt.y + (snapped.dy - raw.dy) }
      commit(objects.value.map((o) => (selectedIds.value.includes(o.id)
        ? moveObject(o, snapped.dx, snapped.dy) : o)))
      break
    }
    case 'pen-handle': {
      const draft = penDraft.value
      if (!draft) break
      // Тянем направляющую поставленного узла: Alt ломает симметрию.
      draft.nodes = moveNode({ nodes: draft.nodes }, gesture.index, 'out', pt.x, pt.y,
        { mirror: !e.altKey }).nodes
      break
    }
    case 'node': {
      const target = objects.value.find((o) => o.id === gesture.id)
      if (!target) break
      updateObject(target.id, moveNode(target, gesture.index, gesture.part, pt.x, pt.y,
        { mirror: !(e.altKey || gesture.alt) }))
      break
    }
    case 'scale': {
      const from = gesture.origin
      let w = Math.max(8, pt.x - from.x)
      let h = Math.max(8, pt.y - from.y)
      // Ctrl (⌘) — пропорционально: картинки и фигуры не «плывут» по форме.
      // Ведём по большему сдвигу, чтобы жест ощущался как обычная тяга угла.
      if (e.ctrlKey || e.metaKey) {
        const k = Math.max(w / from.w, h / from.h)
        w = Math.max(8, from.w * k)
        h = Math.max(8, from.h * k)
      }
      const to = { x: from.x, y: from.y, w, h }
      const byId = new Map(gesture.snapshot.map((o) => [o.id, o]))
      commit(objects.value.map((o) => (byId.has(o.id) ? scaleObject(byId.get(o.id), from, to) : o)))
      break
    }
    case 'rotate': {
      const angle = Math.atan2(pt.y - gesture.pivot.y, pt.x - gesture.pivot.x)
      let deg = ((angle - gesture.start) * 180) / Math.PI
      // Alt держит шаг в 15° — ровные повороты без прицеливания.
      if (e.altKey) deg = Math.round(deg / 15) * 15
      const byId = new Map(gesture.snapshot.map((o) => [o.id, o]))
      commit(objects.value.map((o) => (byId.has(o.id) ? rotateObject(byId.get(o.id), deg, gesture.pivot) : o)))
      break
    }
    case 'marquee':
      gesture.to = pt
      break
    case 'erase':
      eraseAt(pt)
      break
    default:
      break
  }
  requestDraw()
}

function onPointerLeave(e) {
  onPointerUp(e)
  // Круг кисти не должен «залипать» у края холста, когда указатель уже ушёл.
  hover.value = null
  requestDraw()
}

function onPointerUp() {
  if (!gesture) return
  if (gesture.kind === 'draw' && gesture.draft.points.length >= 4) {
    addObject(gesture.draft)
  }
  if (gesture.kind === 'shape') {
    const d = gesture.draft
    const big = d.type === OBJ.line || d.type === OBJ.arrow
      ? Math.hypot(d.x2 - d.x, d.y2 - d.y) > 4
      : d.w > 4 && d.h > 4
    if (big) addObject(d)
  }
  if (gesture.kind === 'marquee') {
    const box = {
      x: Math.min(gesture.from.x, gesture.to.x), y: Math.min(gesture.from.y, gesture.to.y),
      w: Math.abs(gesture.to.x - gesture.from.x), h: Math.abs(gesture.to.y - gesture.from.y),
    }
    setSelection(visibleObjects.value.filter((o) => intersects(objectAABB(o), box)).map((o) => o.id))
  }
  if (gesture.kind === 'lasso') finishLasso(gesture)
  if (gesture.kind === 'move') guides.value = []
  gesture = null
  requestDraw()
}

/** Свободное выделение: берём объекты, чей центр попал внутрь контура. */
function finishLasso(state) {
  if (state.points.length < 6) return
  lastLasso = [...state.points]
  const inside = visibleObjects.value.filter((o) => {
    const b = objectAABB(o)
    return pointInPolygon(state.points, b.x + b.w / 2, b.y + b.h / 2)
  }).map((o) => o.id)
  setSelection(state.additive ? [...new Set([...selectedIds.value, ...inside])] : inside)
}

function intersects(a, b) {
  return a.x < b.x + b.w && a.x + a.w > b.x && a.y < b.y + b.h && a.y + a.h > b.y
}

function eraseAt(pt) {
  const hit = objectAt(pt.x, pt.y, 10)
  if (!hit) return
  const ids = selectionFor(hit)
  commit(objects.value.filter((o) => !ids.includes(o.id)), { ops: [removeOp(ids)] })
}

/** Пипетка: цвет пикселя прямо с холста (включая картинки и наложения). */
function pickColorAt(clientX, clientY) {
  const rect = canvas.value.getBoundingClientRect()
  try {
    const data = ctx.getImageData(
      Math.round((clientX - rect.left) * dpr),
      Math.round((clientY - rect.top) * dpr), 1, 1,
    ).data
    const hex = `#${[data[0], data[1], data[2]].map((v) => v.toString(16).padStart(2, '0')).join('')}`
    emit('pick-color', hex)
  } catch { /* холст «испорчен» чужой картинкой — молча выходим */ }
}

/** Заливка: клик по фигуре красит её текущим цветом (у кривой — замыкает). */
function fillAt(pt) {
  const hit = objectAt(pt.x, pt.y)
  if (!hit) return
  const paint = props.fill || props.color
  if (hit.type === OBJ.path) updateObject(hit.id, { fill: paint, filled: true, closed: true, solid: true })
  else if (hit.type === OBJ.text || hit.type === OBJ.image) updateObject(hit.id, { color: paint })
  else updateObject(hit.id, { fill: paint, solid: true })
  setSelection([hit.id])
}

function onDoubleClick(e) {
  if (!canEdit.value) return
  if (pending.value) {
    finishCurve(false)
    return
  }
  if (penDraft.value) {
    finishPen(false)
    return
  }
  // Двойной клик по контуру — правка его узлов, как в Figma.
  const at = toScene(e.clientX, e.clientY)
  const dblTarget = objectAt(at.x, at.y)
  if (dblTarget?.type === OBJ.vector) {
    nodeEdit.value = { id: dblTarget.id, index: -1 }
    emit('request-tool', 'nodes')
    return
  }
  const pt = toScene(e.clientX, e.clientY)
  const hit = objectAt(pt.x, pt.y)
  if (hit?.type === OBJ.comment) {
    emit('comment-open', hit)
    return
  }
  if (hit && (hit.type === OBJ.text || hit.type === OBJ.sticky)) {
    const anchor = hit.type === OBJ.text ? { x: hit.x, y: hit.y - (hit.size || 18) } : { x: hit.x + 8, y: hit.y + 8 }
    editing.value = { id: hit.id, value: hit.text || '', ...toScreen(anchor), isNew: false }
    focusEditor()
    return
  }
  if (props.tool === 'select' || props.tool === 'text') startTextEditing(pt)
}

function startTextEditing(pt) {
  const o = {
    id: newId(), type: OBJ.text, ...baseFields(), x: pt.x, y: pt.y,
    text: '', size: props.textSize, color: props.color,
  }
  addObject(o)
  openEditor(o, { x: o.x, y: o.y - o.size })
}

// openEditor — инлайн-ввод текста поверх холста в точке объекта.
function openEditor(o, anchor) {
  editing.value = { id: o.id, value: o.text || '', ...toScreen(anchor), isNew: !o.text }
  focusEditor()
}

function focusEditor() {
  requestAnimationFrame(() => {
    // PrimeVue-компонент: настоящее поле лежит в $el.
    const el = editor.value?.$el || editor.value
    el?.focus?.()
    el?.select?.()
  })
}

function commitEditing() {
  const state = editing.value
  editing.value = null
  if (!state) return
  const text = state.value.trim()
  if (!text) {
    // Пустую надпись не храним — иначе холст копит невидимый мусор. Стикер без
    // текста остаётся: он сам по себе объект (цветной листок).
    const target = objects.value.find((o) => o.id === state.id)
    if (target?.type === OBJ.text) commit(objects.value.filter((o) => o.id !== state.id))
    return
  }
  updateObject(state.id, { text })
}

// ── Зум и клавиатура ─────────────────────────────────────────────

function onContextMenu(e) {
  e.preventDefault()
  if (editing.value) commitEditing()
  const pt = toScene(e.clientX, e.clientY)
  const hit = [...visibleObjects.value].reverse().find((o) => hitTest(o, pt.x, pt.y, 8 / camera.value.scale))
  if (hit && !selectedIds.value.includes(hit.id)) setSelection([hit.id])
  if (!hit) setSelection([])
  menu.value = { visible: true, x: e.clientX, y: e.clientY, onObject: !!hit }
  requestDraw()
}

const ALIGN_ITEMS = [
  { label: 'По левому краю', icon: 'align_horizontal_left', action: 'align:left' },
  { label: 'По центру', icon: 'align_horizontal_center', action: 'align:hcenter' },
  { label: 'По правому краю', icon: 'align_horizontal_right', action: 'align:right' },
  { label: 'По верху', icon: 'align_vertical_top', action: 'align:top' },
  { label: 'По середине', icon: 'align_vertical_center', action: 'align:vcenter' },
  { label: 'По низу', icon: 'align_vertical_bottom', action: 'align:bottom' },
  { divider: true },
  { label: 'Разложить по горизонтали', icon: 'horizontal_distribute', action: 'align:distribute-h' },
  { label: 'Разложить по вертикали', icon: 'vertical_distribute', action: 'align:distribute-v' },
]

const menuItems = computed(() => {
  if (!canEdit.value) {
    return [
      { label: 'Вписать в экран', icon: 'fit_screen', action: 'fit' },
      { label: 'Приблизить', icon: 'zoom_in', action: 'zoom-in' },
      { label: 'Отдалить', icon: 'zoom_out', action: 'zoom-out' },
    ]
  }
  if (!menu.value.onObject) {
    return [
      { label: 'Выделить всё', icon: 'select_all', action: 'select-all' },
      ...(clipboard.length ? [{ label: 'Вставить', icon: 'content_paste', action: 'paste' }] : []),
      ...(hasLasso() ? [{
        label: 'Выделение лассо',
        icon: 'gesture',
        children: [
          { label: 'Стереть внутри', icon: 'ink_eraser', action: 'lasso-erase' },
          { label: 'Маска слоя', icon: 'photo_filter', action: 'lasso-mask' },
          { label: 'Обратная маска', icon: 'hide_image', action: 'lasso-mask-invert' },
          { label: 'Снять маску слоя', icon: 'layers_clear', action: 'mask-clear' },
        ],
      }] : []),
      { label: 'Вписать в экран', icon: 'fit_screen', action: 'fit' },
    ]
  }
  const many = selectedIds.value.length > 1
  const picked = objects.value.filter((o) => selectedIds.value.includes(o.id))
  const grouped = picked.some((o) => o.group)
  const rotated = picked.some((o) => o.angle)
  const boolean = picked.some((o) => o.bool)
  const masked = picked.some((o) => o.maskObject)
  const shapes = picked.filter((o) => ![OBJ.text, OBJ.image, OBJ.comment, OBJ.sticky, OBJ.vector].includes(o.type))
  const text = picked.find((o) => o.type === OBJ.text)
  const layers = normalizeScene(props.scene).layers
  return [
    { label: 'Копировать', icon: 'content_copy', action: 'copy' },
    { label: 'Вырезать', icon: 'content_cut', action: 'cut' },
    { label: many ? 'Дублировать выделенное' : 'Дублировать', icon: 'library_add', action: 'duplicate' },
    ...(many ? [{ label: 'Сгруппировать', icon: 'join_inner', action: 'group' }] : []),
    ...(grouped ? [{ label: 'Разгруппировать', icon: 'call_split', action: 'ungroup' }] : []),
    ...(many ? [{ label: 'Выровнять', icon: 'align_horizontal_center', children: ALIGN_ITEMS }] : []),
    ...(many ? [{
      label: 'Объединить фигуры',
      icon: 'join_full',
      children: [
        ...BOOL_OPS.map((b) => ({ label: b.label, icon: b.icon, action: `bool:${b.key}` })),
        ...(boolean ? [{ divider: true }, { label: 'Разъединить', icon: 'call_split', action: 'bool-release' }] : []),
      ],
    }] : []),
    ...(boolean && !many ? [{ label: 'Разъединить фигуры', icon: 'call_split', action: 'bool-release' }] : []),
    {
      label: masked ? 'Снять маску с объекта' : 'Использовать как маску',
      icon: 'photo_filter',
      action: 'mask-object',
    },
    ...(shapes.length ? [{ label: 'Превратить в кривые', icon: 'polyline', action: 'to-vector' }] : []),
    ...(text ? [{
      label: text.pathRef ? 'Снять текст с контура' : 'Пустить текст по контуру',
      icon: 'text_rotation_none',
      action: 'text-path',
    }] : []),
    {
      label: 'Повернуть',
      icon: 'rotate_right',
      children: [
        { label: 'На 90° вправо', icon: 'rotate_right', action: 'rotate:90' },
        { label: 'На 90° влево', icon: 'rotate_left', action: 'rotate:-90' },
        { label: 'На 180°', icon: 'flip_camera_android', action: 'rotate:180' },
        ...(rotated ? [{ label: 'Сбросить поворот', icon: 'restart_alt', action: 'rotate:reset' }] : []),
      ],
    },
    ...(layers.length > 1 ? [{
      label: 'Перенести в слой',
      icon: 'layers',
      children: layers.map((l) => ({ label: l.name, icon: 'layers', action: `layer:${l.id}` })),
    }] : []),
    { label: 'На передний план', icon: 'flip_to_front', action: 'to-front' },
    { label: 'На задний план', icon: 'flip_to_back', action: 'to-back' },
    { divider: true },
    { label: many ? 'Удалить выделенное' : 'Удалить', icon: 'delete', danger: true, action: 'delete' },
  ]
})

/** Поворот выделенного на фиксированный угол (меню и клавиши). */
function rotateSelection(deg) {
  const picked = selectedObjects()
  if (!picked.length) return
  const box = selectionBox()
  const pivot = { x: box.x + box.w / 2, y: box.y + box.h / 2 }
  const byId = new Map(picked.map((o) => [o.id, deg === null ? { ...o, angle: 0 } : rotateObject(o, deg, pivot)]))
  const next = objects.value.map((o) => byId.get(o.id) || o)
  commit(next, { ops: [upsertOp([...byId.values()])] })
}

function onMenuSelect(action) {
  if (action.startsWith('bool:')) {
    booleanSelection(action.slice(5))
    return
  }
  if (action.startsWith('layer:')) {
    moveSelectionToLayer(action.slice(6))
    requestDraw()
    return
  }
  if (action.startsWith('align:')) {
    alignSelection(action.slice(6))
    return
  }
  if (action.startsWith('rotate:')) {
    const value = action.slice(7)
    rotateSelection(value === 'reset' ? null : Number(value))
    requestDraw()
    return
  }
  switch (action) {
    case 'group': groupSelected(); break
    case 'ungroup': ungroupSelected(); break
    case 'delete': removeSelected(); break
    case 'duplicate': duplicateSelected(); break
    case 'copy': copySelection(false); break
    case 'cut': copySelection(true); break
    case 'paste': pasteClipboard(); break
    case 'bool-release': releaseBoolean(); break
    case 'mask-object': toggleMaskObject(); break
    case 'to-vector': convertToVector(); break
    case 'text-path': attachTextToPath(); break
    case 'lasso-erase': eraseInLasso(); break
    case 'lasso-mask': maskFromLasso(false); break
    case 'lasso-mask-invert': maskFromLasso(true); break
    case 'mask-clear': clearLayerMask(); break
    case 'to-front': reorderSelected(true); break
    case 'to-back': reorderSelected(false); break
    case 'select-all': selectAll(); break
    case 'fit': fitToContent(); break
    case 'zoom-in': zoomIn(); break
    case 'zoom-out': zoomOut(); break
    default: break
  }
  requestDraw()
}

function onWheel(e) {
  e.preventDefault()
  if (e.ctrlKey || e.metaKey || !e.shiftKey) {
    zoomAt(e.clientX, e.clientY, e.deltaY < 0 ? 1.1 : 1 / 1.1)
  } else {
    camera.value = { ...camera.value, x: camera.value.x + e.deltaY / camera.value.scale }
  }
  requestDraw()
}

function zoomAt(clientX, clientY, factor) {
  const before = toScene(clientX, clientY)
  const scale = Math.min(4, Math.max(0.2, camera.value.scale * factor))
  camera.value = { ...camera.value, scale }
  const after = toScene(clientX, clientY)
  camera.value = {
    ...camera.value,
    x: camera.value.x + (before.x - after.x),
    y: camera.value.y + (before.y - after.y),
  }
}

function onKeyDown(e) {
  if (editing.value) return
  if (!canEdit.value) return
  const target = e.target
  if (target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)) return
  const combo = e.ctrlKey || e.metaKey
  if (e.key === 'Escape') {
    if (pending.value) { pending.value = null; requestDraw() }
    if (penDraft.value) { penDraft.value = null; requestDraw() }
    setSelection([])
  }
  if (e.key === 'Enter' && (pending.value || penDraft.value)) {
    e.preventDefault()
    if (pending.value) finishCurve(false)
    else finishPen(false)
  }
  if (e.key === 'Delete' || e.key === 'Backspace') {
    // В правке узлов Delete убирает узел, а не весь контур.
    const target = props.tool === 'nodes' && nodeEdit.value.index >= 0 ? editingVector() : null
    if (target) {
      e.preventDefault()
      updateObject(target.id, removeNode(target, nodeEdit.value.index))
      nodeEdit.value = { ...nodeEdit.value, index: -1 }
      requestDraw()
    } else if (selectedIds.value.length) {
      e.preventDefault()
      removeSelected()
    }
  }
  if (!combo && selectedIds.value.length && e.key.startsWith('Arrow')) {
    // Стрелки двигают выделенное: шаг 1, с Shift — 10 (привычка редакторов).
    e.preventDefault()
    const step = e.shiftKey ? 10 : 1
    const [dx, dy] = {
      ArrowLeft: [-step, 0], ArrowRight: [step, 0], ArrowUp: [0, -step], ArrowDown: [0, step],
    }[e.key] || [0, 0]
    const moved = selectedObjects().map((o) => moveObject(o, dx, dy))
    const byId = new Map(moved.map((o) => [o.id, o]))
    commit(objects.value.map((o) => byId.get(o.id) || o), { ops: [upsertOp(moved)] })
    requestDraw()
  }
  if (!combo) return
  const key = e.key.toLowerCase()
  if (key === 'a') {
    e.preventDefault()
    selectAll()
  } else if (key === 'c') {
    copySelection(false)
  } else if (key === 'x') {
    e.preventDefault()
    copySelection(true)
  } else if (key === 'v') {
    pasteClipboard()
  } else if (key === 'd') {
    e.preventDefault()
    duplicateSelected()
    requestDraw()
  } else if (key === 'g') {
    e.preventDefault()
    if (e.shiftKey) ungroupSelected()
    else groupSelected()
    requestDraw()
  }
}

// ── Тач: пинч-зум и панорама двумя пальцами ──────────────────────

function onTouchStart(e) {
  if (e.touches.length !== 2) return
  gesture = null
  pinch = pinchState(e.touches)
}

function onTouchMove(e) {
  if (e.touches.length !== 2 || !pinch) return
  e.preventDefault()
  const next = pinchState(e.touches)
  zoomAt(next.cx, next.cy, next.dist / pinch.dist)
  camera.value = {
    ...camera.value,
    x: camera.value.x - (next.cx - pinch.cx) / camera.value.scale,
    y: camera.value.y - (next.cy - pinch.cy) / camera.value.scale,
  }
  pinch = next
  requestDraw()
}

function onTouchEnd(e) {
  if (e.touches.length < 2) pinch = null
}

function pinchState(touches) {
  const [a, b] = [touches[0], touches[1]]
  return {
    dist: Math.hypot(b.clientX - a.clientX, b.clientY - a.clientY) || 1,
    cx: (a.clientX + b.clientX) / 2,
    cy: (a.clientY + b.clientY) / 2,
  }
}

// ── Картинки ─────────────────────────────────────────────────────

/** Подгрузка картинок сцены: рендер ждёт их через onload → requestDraw. */
function syncImages() {
  const map = images.value
  let added = false
  for (const o of objects.value) {
    if (o.type !== OBJ.image || !o.src || map.has(o.src)) continue
    const img = new Image()
    img.crossOrigin = 'anonymous'
    img.onload = requestDraw
    img.src = o.src
    map.set(o.src, img)
    added = true
  }
  if (added) images.value = new Map(map)
}

// ── Публичный интерфейс для родителя ─────────────────────────────

function zoomIn() { zoomAt(centerX(), centerY(), 1.2); requestDraw() }
function zoomOut() { zoomAt(centerX(), centerY(), 1 / 1.2); requestDraw() }
function resetZoom() { camera.value = { ...camera.value, scale: 1 }; requestDraw() }

function centerX() {
  const rect = canvas.value?.getBoundingClientRect()
  return (rect?.left || 0) + (rect?.width || 0) / 2
}
function centerY() {
  const rect = canvas.value?.getBoundingClientRect()
  return (rect?.top || 0) + (rect?.height || 0) / 2
}

/** Вписать всё нарисованное в экран (кнопка «по размеру»). */
function fitToContent() {
  const box = selectionBox() || contentBox()
  if (!box || !canvas.value) return
  const rect = canvas.value.getBoundingClientRect()
  const scale = Math.min(4, Math.max(0.2, Math.min(rect.width / (box.w + 80), rect.height / (box.h + 80))))
  camera.value = {
    scale,
    x: box.x + box.w / 2 - rect.width / (2 * scale),
    y: box.y + box.h / 2 - rect.height / (2 * scale),
  }
  requestDraw()
}

function contentBox() {
  const list = visibleObjects.value
  if (!list.length) return null
  let minX = Infinity; let minY = Infinity; let maxX = -Infinity; let maxY = -Infinity
  for (const o of list) {
    const b = objectAABB(o)
    minX = Math.min(minX, b.x); minY = Math.min(minY, b.y)
    maxX = Math.max(maxX, b.x + b.w); maxY = Math.max(maxY, b.y + b.h)
  }
  return { x: minX, y: minY, w: maxX - minX, h: maxY - minY }
}

/** Вставить картинку по центру видимой области (после загрузки на сервер). */
function placeImage(src, naturalWidth = 320, naturalHeight = 240) {
  const rect = canvas.value.getBoundingClientRect()
  const center = toScene(rect.left + rect.width / 2, rect.top + rect.height / 2)
  const max = 480
  const k = Math.min(1, max / Math.max(naturalWidth, naturalHeight))
  addObject({
    id: newId(), type: OBJ.image, src, ...baseFields(),
    x: center.x - (naturalWidth * k) / 2, y: center.y - (naturalHeight * k) / 2,
    w: naturalWidth * k, h: naturalHeight * k,
  })
}

/** Применить цвет/толщину к выделенным объектам (панель свойств). */
function applyStyle(patch) {
  if (!selectedIds.value.length) return
  commit(objects.value.map((o) => (selectedIds.value.includes(o.id) ? { ...o, ...patch } : o)))
}

defineExpose({
  zoomIn, zoomOut, resetZoom, fitToContent, placeImage, applyStyle,
  removeSelected, duplicateSelected, selectAll, selectedObjects, camera,
  groupSelected, ungroupSelected, moveSelectionToLayer, setSelection,
  alignSelection, rotateSelection, copySelection, pasteClipboard,
  maskFromLasso, clearLayerMask, eraseInLasso, hasLasso,
  booleanSelection, releaseBoolean, toggleMaskObject, convertToVector, attachTextToPath,
  reorderSelected,
})

// ── Жизненный цикл ───────────────────────────────────────────────

function resize() {
  const el = canvas.value
  if (!el) return
  const rect = el.getBoundingClientRect()
  dpr = window.devicePixelRatio || 1
  el.width = Math.max(1, Math.round(rect.width * dpr))
  el.height = Math.max(1, Math.round(rect.height * dpr))
  requestDraw()
}

onMounted(() => {
  ctx = canvas.value.getContext('2d', { willReadFrequently: true })
  resize()
  syncImages()
  resizeObs = new ResizeObserver(resize)
  resizeObs.observe(host.value)
  window.addEventListener('keydown', onKeyDown)
  // Окно рабочего стола позиционируется transform'ом с CSS-переходом (сессия
  // восстанавливается на перезагрузке асинхронно, геометрия окна доезжает до
  // места уже ПОСЛЕ первого mount) — ResizeObserver реагирует только на смену
  // логического размера host, а не на смену transform/devicePixelRatio по
  // ходу переезда. Пока размер не пришёл в норму, координаты клика (toScene)
  // и буфер канваса (dpr) считаются по ещё не устоявшемуся rect — отсюда
  // смещение штриха относительно курсора, которое снимает только следующий
  // ResizeObserver (например, реальный ресайз при разворачивании в полный
  // экран). Досчитываем сами через короткий таймаут после монтирования.
  window.addEventListener('resize', resize)
  setTimeout(resize, 250)
})

onBeforeUnmount(() => {
  resizeObs?.disconnect()
  window.removeEventListener('keydown', onKeyDown)
  window.removeEventListener('resize', resize)
  if (frameId) cancelAnimationFrame(frameId)
})

watch(() => props.scene, () => {
  syncImages()
  requestDraw()
}, { deep: true })

// Смена темы не трогает props.scene — без отдельного watcher'а холст держал
// цвета предыдущей темы (в т.ч. фон-сетку/точки) до первой правки сцены.
const theme = useThemeStore()
watch(() => theme.dark, () => {
  invalidateColors()
  requestDraw()
})

watch(() => props.peers, requestDraw, { deep: true })
watch(() => [props.frame, props.onion], requestDraw, { deep: true })
watch(() => props.tool, () => {
  if (!KEEP_SELECTION.has(props.tool)) setSelection([])
  // Незаконченный контур принадлежит своему инструменту — уходя, закрываем его.
  if (props.tool !== 'curve' && pending.value) finishCurve(false)
  if (props.tool !== 'pen' && penDraft.value) finishPen(false)
  if (props.tool !== 'nodes') nodeEdit.value = { id: '', index: -1 }
  requestDraw()
})
</script>

<template>
  <!-- data-no-ptr: вертикальный жест здесь — это штрих, а не «обновить
       страницу». Без него оттяжка сверху вниз при рисовании поднимала индикатор
       обновления сенсорных каркасов и могла перезагрузить доску (см.
       components/common/PullToRefresh.vue). -->
  <div ref="host" class="board-canvas" :class="{ 'is-readonly': readOnly }" data-no-ptr>
    <canvas
      ref="canvas"
      class="board-canvas__surface"
      :data-tool="tool"
      :class="{ 'is-brush': showBrushCursor }"
      @pointerdown="onPointerDown"
      @pointermove="onPointerMove"
      @pointerup="onPointerUp"
      @pointercancel="onPointerUp"
      @pointerleave="onPointerLeave"
      @dblclick="onDoubleClick"
      @wheel="onWheel"
      @touchstart="onTouchStart"
      @touchmove="onTouchMove"
      @touchend="onTouchEnd"
      @contextmenu="onContextMenu"
    />

    <!-- Инлайн-ввод надписи: поле поверх холста ровно в точке ввода. -->
    <Textarea
      v-if="editing"
      ref="editor"
      v-model="editing.value"
      class="board-canvas__editor"
      :style="{ left: `${editing.x}px`, top: `${editing.y}px`, fontSize: `${textSize * camera.scale}px` }"
      rows="1"
      auto-resize
      @blur="commitEditing"
      @keydown.esc.prevent="commitEditing"
      @keydown.enter.exact.prevent="commitEditing"
    />

    <ContextMenu
      :visible="menu.visible"
      :x="menu.x"
      :y="menu.y"
      :items="menuItems"
      @select="onMenuSelect"
      @close="menu.visible = false"
    />
  </div>
</template>

<style scoped>
.board-canvas {
  position: relative;
  width: 100%;
  height: 100%;
  overflow: hidden;
  border-radius: var(--radius-lg);
  background: var(--color-surface);
}

.board-canvas__surface {
  display: block;
  width: 100%;
  height: 100%;
  touch-action: none;
  cursor: crosshair;
}

.board-canvas__surface[data-tool='select'] { cursor: default; }
.board-canvas__surface[data-tool='pan'] { cursor: grab; }
.board-canvas__surface[data-tool='text'] { cursor: text; }
.board-canvas__surface[data-tool='eyedropper'] { cursor: copy; }
.board-canvas__surface[data-tool='pen'] { cursor: crosshair; }
.board-canvas__surface[data-tool='nodes'] { cursor: default; }
/* У кисти и ластика свой круг на холсте — системный курсор только мешает. */
.board-canvas__surface.is-brush { cursor: none; }
.is-readonly .board-canvas__surface { cursor: grab; }

.board-canvas__editor {
  position: absolute;
  z-index: 2;
  min-width: 120px;
  padding: 2px 6px;
  border: 1px solid var(--color-primary);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  color: var(--color-text);
  font-family: inherit;
  line-height: 1.35;
  resize: none;
  outline: none;
}
</style>
