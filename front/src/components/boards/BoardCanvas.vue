<script setup>
/* Холст доски: свой движок поверх <canvas> — перо и маркер, фигуры, стрелки,
   надписи, стикеры и картинки на бесконечном полотне с зумом и панорамой.

   Почему свой, а не библиотека: сцена — обычный JSON (его же понимает сервер:
   ищет по надписям и строит SVG при выгрузке), а рисование живёт в токенах
   темы — цвета берутся из --tag-*, поэтому доска сама перекрашивается в тёмном
   режиме. Правки идут наружу событием update:scene, а история (undo/redo)
   держится здесь: снимок сцены на каждый завершённый жест. */
import { computed, onBeforeUnmount, onMounted, ref, shallowRef, watch } from 'vue'
import Textarea from 'primevue/textarea'
import ContextMenu from '@/components/common/ContextMenu.vue'
import {
  COMMENT_PIN, OBJ, drawObject, editableLayerIds, hitTest, moveObject, newId,
  normalizeScene, objectBounds, orderedObjects, renderScene, resolveColor, scaleObject,
} from '@/utils/boardScene.js'

const props = defineProps({
  scene: { type: Object, required: true },
  tool: { type: String, default: 'select' },
  color: { type: String, default: 'ink' },
  fill: { type: String, default: '' },
  width: { type: Number, default: 4 },
  textSize: { type: Number, default: 18 },
  readOnly: { type: Boolean, default: false },
  // Слой, в который попадают новые объекты (панель слоёв редактора).
  activeLayer: { type: String, default: '' },
  // Кто я — подписываю комментарии.
  me: { type: Object, default: () => ({}) },
  // Курсоры соавторов: [{ user_id, fio, x, y }].
  peers: { type: Array, default: () => [] },
})

/* update:scene — новая сцена целиком (её сохраняет редактор), ops — только
   изменившиеся объекты: их редактор рассылает соавторам, чтобы параллельные
   правки не затирали друг друга целой сценой. */
const emit = defineEmits(['update:scene', 'ops', 'pointer-move', 'select-change', 'comment-open'])

// STICKY_SIZE — сторона липкой заметки в единицах сцены (квадрат, как в Miro).
const STICKY_SIZE = 180

const host = ref(null)
const canvas = ref(null)
const editor = ref(null)          // textarea инлайн-ввода надписи

const camera = ref({ x: -80, y: -60, scale: 1 })
const selectedIds = ref([])
const editing = ref(null)         // { id, x, y, value, isNew }
// Контекстное меню холста: пункты зависят от того, попали ли по объекту.
const menu = ref({ visible: false, x: 0, y: 0, onObject: false })
const images = shallowRef(new Map())

let ctx = null
let dpr = 1
let frame = 0
let resizeObs = null

// Активный жест: рисование, перетаскивание, рамка выделения или панорама.
let gesture = null
// Пинч двумя пальцами: расстояние и центр на старте.
let pinch = null

const objects = computed(() => normalizeScene(props.scene).objects)
// Порядок отрисовки: снизу вверх по слоям (скрытые слои не рисуются).
const visibleObjects = computed(() => orderedObjects(props.scene))
// Трогать можно только объекты видимых и незаблокированных слоёв.
const editableLayers = computed(() => editableLayerIds(props.scene))
const canEdit = computed(() => !props.readOnly)

// Слой для новых объектов: заданный редактором либо первый доступный.
function targetLayer() {
  const layers = normalizeScene(props.scene).layers
  if (props.activeLayer && editableLayers.value.has(props.activeLayer)) return props.activeLayer
  return (layers.find((l) => l.visible && !l.locked) || layers[0]).id
}

/** Объект под точкой — сверху вниз, только по доступным слоям. */
function objectAt(px, py, tolerance = 8) {
  const list = visibleObjects.value.filter((o) => editableLayers.value.has(o.layer))
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
  if (frame) return
  frame = requestAnimationFrame(() => {
    frame = 0
    draw()
  })
}

function draw() {
  const el = canvas.value
  if (!el || !ctx) return
  const { width, height } = el.getBoundingClientRect()
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
  ctx.clearRect(0, 0, width, height)

  renderScene(ctx, props.scene, { width, height, camera: camera.value, images: images.value })

  // Незавершённый объект текущего жеста рисуем поверх — он ещё не в сцене.
  if (gesture?.draft) {
    ctx.save()
    ctx.scale(camera.value.scale, camera.value.scale)
    ctx.translate(-camera.value.x, -camera.value.y)
    drawObject(ctx, gesture.draft, images.value)
    ctx.restore()
  }
  drawSelection(ctx)
  drawMarquee(ctx)
  drawPeers(ctx)
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
    const b = objectBounds(o)
    const p = toScreen({ x: b.x, y: b.y })
    c.strokeRect(p.x - 4, p.y - 4, b.w * camera.value.scale + 8, b.h * camera.value.scale + 8)
  }
  // Ручка масштабирования — в правом нижнем углу общей рамки выделения.
  const box = selectionBox()
  if (box) {
    const p = toScreen({ x: box.x + box.w, y: box.y + box.h })
    c.setLineDash([])
    c.fillStyle = accent
    c.fillRect(p.x - 4, p.y - 4, 10, 10)
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

function selectionBox() {
  const picked = objects.value.filter((o) => selectedIds.value.includes(o.id))
  if (!picked.length) return null
  let minX = Infinity; let minY = Infinity; let maxX = -Infinity; let maxY = -Infinity
  for (const o of picked) {
    const b = objectBounds(o)
    minX = Math.min(minX, b.x); minY = Math.min(minY, b.y)
    maxX = Math.max(maxX, b.x + b.w); maxY = Math.max(maxY, b.y + b.h)
  }
  return { x: minX, y: minY, w: maxX - minX, h: maxY - minY }
}

// ── Изменение сцены ──────────────────────────────────────────────

function commit(objectsNext, { background, layers, ops } = {}) {
  emit('update:scene', {
    ...normalizeScene(props.scene),
    ...(background ? { background } : {}),
    ...(layers ? { layers } : {}),
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
  setSelection(objects.value.map((o) => o.id))
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

// ── Указатель ────────────────────────────────────────────────────

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
    case 'eraser':
      gesture = { kind: 'erase' }
      eraseAt(pt)
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
    case 'pen':
    case 'marker':
      gesture = {
        kind: 'draw',
        draft: {
          id: newId(), type: OBJ.path, layer: targetLayer(), x: pt.x, y: pt.y,
          points: [pt.x, pt.y], color: props.color,
          width: props.tool === 'marker' ? props.width * 3 : props.width,
          opacity: props.tool === 'marker' ? 0.35 : 1,
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
    // Ручка масштабирования в правом нижнем углу рамки.
    const handle = toScreen({ x: box.x + box.w, y: box.y + box.h })
    const screen = toScreen(pt)
    if (Math.abs(screen.x - handle.x) < 10 && Math.abs(screen.y - handle.y) < 10) {
      gesture = { kind: 'scale', from: box, origin: { ...box }, snapshot: selectedObjects() }
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
    id: newId(), type: OBJ.sticky, layer: targetLayer(),
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
    id: newId(), type: OBJ.comment, layer: targetLayer(),
    x: pt.x - COMMENT_PIN / 2, y: pt.y - COMMENT_PIN / 2,
    color: 'amber', text: '', resolved: false, replies: [],
    author_id: props.me?.id ?? null, author: props.me?.fio || '',
    created_at: new Date().toISOString(),
  }
  addObject(o)
  setSelection([o.id])
  emit('comment-open', o)
}

function startShape(pt) {
  const type = { line: OBJ.line, arrow: OBJ.arrow, rect: OBJ.rect, ellipse: OBJ.ellipse, diamond: OBJ.diamond }[props.tool]
  if (!type) return
  const base = { id: newId(), type, layer: targetLayer(), color: props.color, width: props.width, fill: props.fill }
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
  emit('pointer-move', pt)
  if (!gesture) return

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
      const dx = pt.x - gesture.last.x
      const dy = pt.y - gesture.last.y
      gesture.last = pt
      commit(objects.value.map((o) => (selectedIds.value.includes(o.id) ? moveObject(o, dx, dy) : o)))
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
    setSelection(objects.value.filter((o) => intersects(objectBounds(o), box)).map((o) => o.id))
  }
  gesture = null
  requestDraw()
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

function onDoubleClick(e) {
  if (!canEdit.value) return
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
  startTextEditing(pt)
}

function startTextEditing(pt) {
  const o = {
    id: newId(), type: OBJ.text, layer: targetLayer(), x: pt.x, y: pt.y,
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
  const hit = [...objects.value].reverse().find((o) => hitTest(o, pt.x, pt.y, 8 / camera.value.scale))
  if (hit && !selectedIds.value.includes(hit.id)) setSelection([hit.id])
  if (!hit) setSelection([])
  menu.value = { visible: true, x: e.clientX, y: e.clientY, onObject: !!hit }
  requestDraw()
}

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
      { label: 'Вписать в экран', icon: 'fit_screen', action: 'fit' },
    ]
  }
  const many = selectedIds.value.length > 1
  const picked = objects.value.filter((o) => selectedIds.value.includes(o.id))
  const grouped = picked.some((o) => o.group)
  const layers = normalizeScene(props.scene).layers
  return [
    { label: many ? 'Дублировать выделенное' : 'Дублировать', icon: 'content_copy', action: 'duplicate' },
    ...(many ? [{ label: 'Сгруппировать', icon: 'join_inner', action: 'group' }] : []),
    ...(grouped ? [{ label: 'Разгруппировать', icon: 'call_split', action: 'ungroup' }] : []),
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

function onMenuSelect(action) {
  if (action.startsWith('layer:')) {
    moveSelectionToLayer(action.slice(6))
    requestDraw()
    return
  }
  switch (action) {
    case 'group': groupSelected(); break
    case 'ungroup': ungroupSelected(); break
    case 'delete': removeSelected(); break
    case 'duplicate': duplicateSelected(); break
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
  if (e.key === 'Delete' || e.key === 'Backspace') {
    if (selectedIds.value.length) {
      e.preventDefault()
      removeSelected()
    }
  }
  if (e.key === 'Escape') setSelection([])
  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'a') {
    e.preventDefault()
    setSelection(objects.value.map((o) => o.id))
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
  if (!objects.value.length) return null
  let minX = Infinity; let minY = Infinity; let maxX = -Infinity; let maxY = -Infinity
  for (const o of objects.value) {
    const b = objectBounds(o)
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
    id: newId(), type: OBJ.image, src,
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
  ctx = canvas.value.getContext('2d')
  resize()
  syncImages()
  resizeObs = new ResizeObserver(resize)
  resizeObs.observe(host.value)
  window.addEventListener('keydown', onKeyDown)
})

onBeforeUnmount(() => {
  resizeObs?.disconnect()
  window.removeEventListener('keydown', onKeyDown)
  if (frame) cancelAnimationFrame(frame)
})

watch(() => props.scene, () => {
  syncImages()
  requestDraw()
}, { deep: true })

watch(() => props.peers, requestDraw, { deep: true })
watch(() => props.tool, () => {
  if (props.tool !== 'select') setSelection([])
  requestDraw()
})
</script>

<template>
  <div ref="host" class="board-canvas" :class="{ 'is-readonly': readOnly }">
    <canvas
      ref="canvas"
      class="board-canvas__surface"
      :data-tool="tool"
      @pointerdown="onPointerDown"
      @pointermove="onPointerMove"
      @pointerup="onPointerUp"
      @pointercancel="onPointerUp"
      @pointerleave="onPointerUp"
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
.board-canvas__surface[data-tool='eraser'] { cursor: cell; }
.board-canvas__surface[data-tool='text'] { cursor: text; }
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
