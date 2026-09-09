<script setup>
/* Экран доски: холст на всю площадь, плавающий тулбар, автосохранение сцены и
   совместное рисование. Миниатюра плитки снимается с самого холста после
   паузы в рисовании — список досок выглядит как галерея эскизов.

   Режим кадров (лента внизу) превращает доску в покадровую анимацию: объекты
   получают метку кадра, «калька» показывает соседние, а готовое уезжает
   видеофайлом. Всё это — свойства той же сцены, поэтому отдельного хранилища
   у анимации нет. */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import InputText from 'primevue/inputtext'
import AppPage from '@/components/ui/AppPage.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import BoardCanvas from '@/components/boards/BoardCanvas.vue'
import BoardToolbar from '@/components/boards/BoardToolbar.vue'
import CommentPopup from '@/components/boards/CommentPopup.vue'
import ExportAnimationDialog from '@/components/boards/ExportAnimationDialog.vue'
import FrameTimeline from '@/components/boards/FrameTimeline.vue'
import LayersPanel from '@/components/boards/LayersPanel.vue'
import PropertiesPanel from '@/components/boards/PropertiesPanel.vue'
import ShareDialog from '@/components/boards/ShareDialog.vue'
import * as api from '@/api/boards.js'
import { FPS_RANGE, emptyScene, newId, normalizeScene } from '@/utils/boardScene.js'
import { sceneToPreview } from '@/utils/boardExport.js'
import { BOARD_EXPORT_ITEMS, boardExportFormat, useBoardDownload } from '@/composables/useBoardDownload.js'
import { useBoardCollab } from '@/composables/useBoardCollab.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { useAuthStore } from '@/stores/auth.js'
import { useBoardsStore } from '@/stores/boards.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const route = useRoute()
const router = useRouter()
const boards = useBoardsStore()
const auth = useAuthStore()
const notify = useNotificationsStore()
const { downloadBoard } = useBoardDownload()

const boardId = computed(() => Number(route.params.id))

const board = ref(null)
const scene = ref(emptyScene())
const title = ref('')
const loading = ref(true)
const saving = ref(false)
const shareOpen = ref(false)

const canvasRef = ref(null)
const fileInput = ref(null)

const tool = ref('pencil')
const color = ref('ink')
const fill = ref('')
const strokeWidth = ref(4)
const opacity = ref(1)
const textSize = ref(18)
const eraserSize = ref(32)
const eraseMode = ref('pixel')
const polygonSides = ref(5)
const polygonStar = ref(false)
const selection = ref([])
const activeLayer = ref('')
const layersOpen = ref(false)
const propsOpen = ref(true)

// Режим кадров: лента, текущий кадр, глубина кальки и проигрывание.
const framesOpen = ref(false)
const currentFrame = ref('')
const onionDepth = ref(1)
const playing = ref(false)
const exportingVideo = ref(false)
const exportProgress = ref(0)
const exportOpen = ref(false)

// Открытое обсуждение: сам комментарий и точка на экране, где стоит булавка.
const activeComment = ref(null)
const commentAnchor = ref({ x: 0, y: 0 })
const exportMenu = ref({ visible: false, x: 0, y: 0 })
const boardMenu = ref({ visible: false, x: 0, y: 0 })
const { isMobile } = useBreakpoint()

// История: снимки сцены до правки. Хранится здесь, а не в сторе — она нужна
// только открытому редактору и не переживает выход из доски.
const undoStack = ref([])
const redoStack = ref([])
const HISTORY_LIMIT = 60

let saveTimer = null
let previewTimer = null
let drawingUntil = 0
let dirty = false

const canEdit = computed(() => !board.value || board.value.my_access !== 'view')
const me = computed(() => ({ id: auth.userId, fio: auth.user?.fio || '' }))
const zoom = computed(() => canvasRef.value?.camera?.scale || 1)
const background = computed(() => normalizeScene(scene.value).background)
const frames = computed(() => normalizeScene(scene.value).animation?.frames || [])
// Панель свойств и дерево слоёв показывают ТЕ ЖЕ объекты, что выделены на
// холсте: выделение живёт в холсте, наружу приходит списком id.
const selectedObjects = computed(() => {
  const ids = new Set(selection.value)
  return normalizeScene(scene.value).objects.filter((o) => ids.has(o.id))
})

/* Свойства показываются САМИ, как только на холсте что-то выделено, и уходят
   вместе с выделением. Закрытая крестиком панель молчит до следующего нажатия
   кнопки «Свойства»: иначе она возвращалась бы на каждый клик по холсту. */
const showProps = computed(() => propsOpen.value && selection.value.length > 0)

/* Калька: предыдущие кадры бледнеют с удалением от текущего. Во время
   проигрывания её нет — иначе ролик смотрится смазанным. */
const onion = computed(() => {
  if (!framesOpen.value || playing.value || !onionDepth.value || !currentFrame.value) return []
  const list = frames.value
  const index = list.findIndex((f) => f.id === currentFrame.value)
  const out = []
  for (let back = onionDepth.value; back >= 1; back -= 1) {
    const ghost = list[index - back]
    if (ghost) out.push({ frame: ghost.id, alpha: 0.35 / back })
  }
  return out
})

const { others: peers, start: startCollab, sendCursor, sendScene, sendOps } = useBoardCollab({
  boardId,
  canEdit,
  isDrawing: () => Date.now() < drawingUntil,
  getScene: () => scene.value,
  getTitle: () => title.value,
  onRemoteOps: applyRemoteOps,
  onRemoteScene: (remote) => { scene.value = normalizeScene(remote) },
  onRemoteTitle: (remote) => { if (document.activeElement?.dataset?.boardTitle == null) title.value = remote },
})

async function load() {
  loading.value = true
  try {
    const data = await api.getBoard(boardId.value)
    board.value = data
    title.value = data.title || ''
    scene.value = normalizeScene(data.scene)
    activeLayer.value = scene.value.layers[scene.value.layers.length - 1].id
    // Доска с кадрами открывается в режиме анимации на первом кадре: иначе
    // холст показал бы только «общие» объекты, а кадры казались бы потерянными.
    if (frames.value.length) {
      framesOpen.value = true
      currentFrame.value = frames.value[0].id
    }
    startCollab()
  } catch {
    notify.error('Не удалось открыть доску')
    router.replace('/boards')
  } finally {
    loading.value = false
  }
}


// ── Правки и сохранение ──────────────────────────────────────────

function onSceneUpdate(next) {
  if (!canEdit.value) return
  pushHistory(scene.value)
  scene.value = normalizeScene(next)
  drawingUntil = Date.now() + 400
  dirty = true
  scheduleSave()
  schedulePreview()
}

/* Операции холста уходят соавторам адресно: правка одного объекта не трогает
   остальную сцену, поэтому одновременная работа не затирает чужие штрихи. */
function onCanvasOps(ops) {
  if (!canEdit.value) return
  sendOps(ops)
}

/** Применить операции соавтора к своей сцене. */
function applyRemoteOps(ops) {
  const current = normalizeScene(scene.value)
  const byId = new Map(current.objects.map((o) => [o.id, o]))
  for (const op of ops) {
    if (op.kind === 'remove') {
      for (const id of op.ids || []) byId.delete(id)
    } else {
      for (const o of op.objects || []) byId.set(o.id, o)
    }
  }
  scene.value = normalizeScene({ ...current, objects: [...byId.values()] })
  dirty = true
  scheduleSave()
}

/** Слои правит панель — они часть сцены, поэтому едут тем же путём. */
function onLayersUpdate(layers) {
  onSceneUpdate({ ...normalizeScene(scene.value), layers })
  sendScene() // порядок и видимость слоёв — свойство всей сцены, не объекта
}

/** Объекты правит дерево слоёв: порядок, видимость, блокировка, слой. */
function onObjectsUpdate(objects) {
  onSceneUpdate({ ...normalizeScene(scene.value), objects })
  sendScene() // порядок объектов — свойство всей сцены, не одного объекта
}

/** Лента кадров меняет и кадры, и объекты — это правка всей сцены. */
function onFramesUpdate(next) {
  onSceneUpdate(next)
  sendScene()
}

function pushHistory(snapshot) {
  undoStack.value.push(JSON.stringify(snapshot))
  if (undoStack.value.length > HISTORY_LIMIT) undoStack.value.shift()
  redoStack.value = []
}

function undo() {
  const prev = undoStack.value.pop()
  if (!prev) return
  redoStack.value.push(JSON.stringify(scene.value))
  scene.value = normalizeScene(JSON.parse(prev))
  dirty = true
  scheduleSave()
  sendScene()
}

function redo() {
  const next = redoStack.value.pop()
  if (!next) return
  undoStack.value.push(JSON.stringify(scene.value))
  scene.value = normalizeScene(JSON.parse(next))
  dirty = true
  scheduleSave()
  sendScene()
}

function scheduleSave() {
  clearTimeout(saveTimer)
  saveTimer = setTimeout(save, 900)
}

async function save() {
  if (!dirty || !canEdit.value) return
  saving.value = true
  try {
    const updated = await api.updateBoard(boardId.value, { title: title.value, scene: scene.value })
    dirty = false
    boards.applyBoardSocket('updated', updated)
  } catch {
    notify.error('Не удалось сохранить доску')
  } finally {
    saving.value = false
  }
}

// Превью — тяжеловато для каждого штриха, поэтому снимаем после паузы.
function schedulePreview() {
  clearTimeout(previewTimer)
  previewTimer = setTimeout(async () => {
    if (!canEdit.value) return
    try {
      const blob = await sceneToPreview(scene.value)
      if (blob) await api.uploadPreview(boardId.value, blob)
    } catch { /* превью не критично */ }
  }, 4000)
}

// ── Комментарии ──────────────────────────────────────────────────

function onCommentOpen(comment) {
  activeComment.value = comment
  const cam = canvasRef.value?.camera
  const rect = canvasRef.value?.$el?.getBoundingClientRect?.()
  if (cam && rect) {
    // Попап ставим рядом с булавкой в координатах контейнера холста.
    commentAnchor.value = {
      x: (comment.x - cam.x) * cam.scale + 36,
      y: (comment.y - cam.y) * cam.scale,
    }
  }
}

function onCommentUpdate(next) {
  const current = normalizeScene(scene.value)
  onSceneUpdate({
    ...current,
    objects: current.objects.map((o) => (o.id === next.id ? next : o)),
  })
  sendOps([{ kind: 'upsert', objects: [next] }])
  activeComment.value = next
}

function onCommentDelete(comment) {
  const current = normalizeScene(scene.value)
  onSceneUpdate({ ...current, objects: current.objects.filter((o) => o.id !== comment.id) })
  sendOps([{ kind: 'remove', ids: [comment.id] }])
  activeComment.value = null
}

function setBackground(key) {
  onSceneUpdate({ ...normalizeScene(scene.value), background: key })
}

// ── Кадры ────────────────────────────────────────────────────────

/* Первое включение заводит один кадр, а нарисованное раньше остаётся ОБЩИМ
   (видно во всех кадрах) — это фон анимации. Так включение режима ничего не
   прячет и легко отменяется. */
function toggleFrames() {
  if (framesOpen.value) {
    framesOpen.value = false
    currentFrame.value = ''
    return
  }
  if (!frames.value.length) {
    if (!canEdit.value) return
    const first = { id: newId(), name: 'Кадр 1' }
    onSceneUpdate({ ...normalizeScene(scene.value), animation: { fps: FPS_RANGE.def, frames: [first] } })
    sendScene()
    currentFrame.value = first.id
  } else if (!currentFrame.value) {
    currentFrame.value = frames.value[0].id
  }
  framesOpen.value = true
}

function openAnimationExport() {
  if (!frames.value.length) {
    notify.warn('Сначала добавьте кадры анимации')
    return
  }
  exportOpen.value = true
}

/** Собрать ролик или гифку по настройкам диалога. */
async function exportAnimation({ format, ...options }) {
  exportingVideo.value = true
  exportProgress.value = 0
  try {
    await downloadBoard({ id: boardId.value, title: title.value }, format, scene.value, {
      ...options,
      onProgress: (value) => { exportProgress.value = value },
    })
    exportOpen.value = false
  } finally {
    exportingVideo.value = false
  }
}

// ── Картинки ─────────────────────────────────────────────────────

function pickImage() {
  fileInput.value?.click()
}

async function onImagePicked(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  try {
    const { path } = await api.uploadImage(boardId.value, file)
    const img = new Image()
    img.onload = () => canvasRef.value?.placeImage(path, img.naturalWidth, img.naturalHeight)
    img.onerror = () => canvasRef.value?.placeImage(path)
    img.src = path
  } catch {
    notify.error('Не удалось загрузить картинку')
  }
}

// ── Выгрузка ─────────────────────────────────────────────────────

/* Те же действия, что в ряду кнопок на широком экране. Фон — подменю: три
   варианта отдельными пунктами заняли бы половину списка. */
const boardMenuItems = computed(() => [
  { label: 'Отменить', icon: 'undo', action: 'undo', disabled: !undoStack.value.length },
  { label: 'Повторить', icon: 'redo', action: 'redo', disabled: !redoStack.value.length },
  { divider: true },
  {
    label: 'Фон',
    icon: 'grid_4x4',
    children: [
      { label: 'Сетка', icon: 'grid_4x4', action: 'bg:grid' },
      { label: 'Точки', icon: 'blur_on', action: 'bg:dots' },
      { label: 'Чистый', icon: 'crop_portrait', action: 'bg:plain' },
    ],
  },
  { label: layersOpen.value ? 'Скрыть слои' : 'Слои', icon: 'layers', action: 'layers' },
  { label: propsOpen.value ? 'Скрыть свойства' : 'Свойства', icon: 'tune', action: 'props' },
  { label: framesOpen.value ? 'Скрыть кадры' : 'Кадры анимации', icon: 'animation', action: 'frames' },
  { divider: true },
  { label: 'Скачать', icon: 'download', action: 'export' },
  ...(canEdit.value ? [{ label: 'Поделиться', icon: 'share', action: 'share' }] : []),
])

function openBoardMenu(e) {
  const r = e.currentTarget.getBoundingClientRect()
  boardMenu.value = { visible: true, x: r.right, y: r.bottom + 4 }
}

function onBoardMenu(action) {
  boardMenu.value.visible = false
  if (action === 'undo') undo()
  else if (action === 'redo') redo()
  else if (action.startsWith('bg:')) setBackground(action.slice(3))
  else if (action === 'layers') layersOpen.value = !layersOpen.value
  else if (action === 'props') propsOpen.value = !propsOpen.value
  else if (action === 'frames') toggleFrames()
  else if (action === 'share') shareOpen.value = true
  // Экспорт — своё меню; открываем его там же, где стояло это.
  else if (action === 'export') {
    exportMenu.value = { visible: true, x: boardMenu.value.x, y: boardMenu.value.y }
  }
}

function openExportMenu(e) {
  exportMenu.value = { visible: true, x: e.clientX, y: e.clientY }
}

// Сцену передаём свою: открытая доска правится прямо сейчас, и на сервере
// может лежать состояние до последнего автосохранения.
function exportBoard(action) {
  const format = boardExportFormat(action)
  if (format === 'animation') {
    openAnimationExport()
    return
  }
  if (format) downloadBoard({ id: boardId.value, title: title.value }, format, scene.value)
}

// ── Клавиатура ───────────────────────────────────────────────────

function onKeyDown(e) {
  if (!(e.ctrlKey || e.metaKey)) return
  const key = e.key.toLowerCase()
  if (key === 'z' && !e.shiftKey) { e.preventDefault(); undo() }
  else if ((key === 'z' && e.shiftKey) || key === 'y') { e.preventDefault(); redo() }
  else if (key === 's') { e.preventDefault(); save() }
}

// ── Жизненный цикл ───────────────────────────────────────────────

onMounted(() => {
  load()
  window.addEventListener('keydown', onKeyDown)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeyDown)
  clearTimeout(saveTimer)
  clearTimeout(previewTimer)
  save()
})

// Пока что-то выделено, палитра и толщина меняют сами объекты — это ожидаемое
// поведение любого редактора (иначе цвет применился бы только к следующему).
watch(color, (v) => { if (selection.value.length) canvasRef.value?.applyStyle({ color: v }) })
watch(fill, (v) => { if (selection.value.length) canvasRef.value?.applyStyle({ fill: v }) })
watch(strokeWidth, (v) => { if (selection.value.length) canvasRef.value?.applyStyle({ width: v }) })
watch(textSize, (v) => { if (selection.value.length) canvasRef.value?.applyStyle({ size: v }) })
watch(opacity, (v) => { if (selection.value.length) canvasRef.value?.applyStyle({ opacity: v }) })

// Кадр мог исчезнуть (удалил соавтор или сам) — переходим на первый.
watch(frames, (list) => {
  if (!list.length) {
    framesOpen.value = false
    currentFrame.value = ''
  } else if (!list.some((f) => f.id === currentFrame.value)) {
    currentFrame.value = list[0].id
  }
})

watch(boardId, () => { if (boardId.value) load() })
watch(title, () => {
  if (!canEdit.value || loading.value) return
  dirty = true
  scheduleSave()
})
</script>

<template>
  <!-- bare: доска сама себе фон — панель раздела под холстом не нужна.
       headless: шапку с названием, соавторами и инструментами рисует редактор.
       scroll=false: холст занимает всё тело и прокрутки не имеет. -->
  <AppPage class="be" bare headless flush :scroll="false">
    <header class="be-head">
      <button type="button" class="be-back" title="К доскам" aria-label="К доскам" @click="router.push('/boards')">
        <span class="material-symbols-outlined">arrow_back</span>
      </button>

      <InputText
        v-model="title"
        data-board-title
        class="be-title"
        placeholder="Название доски"
        maxlength="300"
        :disabled="!canEdit"
      />

      <div class="be-peers">
        <span
          v-for="p in peers"
          :key="p.user_id"
          class="be-peer"
          :style="{ background: `var(--tag-${p.color}-accent)` }"
          :title="p.fio"
        >{{ (p.fio || '?').charAt(0) }}</span>
      </div>

      <span v-if="saving" class="be-state">Сохраняем…</span>
      <span v-else-if="!canEdit" class="be-state">Только просмотр</span>

      <!-- На телефоне ряд из семи кнопок не помещается: он переносился второй
           строкой, а поле названия схлопывалось в кружок. Поэтому всё, кроме
           отмены/повтора, уходит в меню. -->
      <button
        v-if="isMobile"
        type="button"
        class="be-btn"
        title="Действия с доской"
        aria-label="Действия с доской"
        @click="openBoardMenu"
      >
        <span class="material-symbols-outlined">more_vert</span>
      </button>

      <div v-else class="be-actions">
        <button type="button" class="be-btn" title="Отменить" aria-label="Отменить" :disabled="!undoStack.length" @click="undo">
          <span class="material-symbols-outlined">undo</span>
        </button>
        <button type="button" class="be-btn" title="Повторить" aria-label="Повторить" :disabled="!redoStack.length" @click="redo">
          <span class="material-symbols-outlined">redo</span>
        </button>
        <button
          v-for="bg in [{ key: 'grid', icon: 'grid_4x4' }, { key: 'dots', icon: 'blur_on' }, { key: 'plain', icon: 'crop_portrait' }]"
          :key="bg.key"
          type="button"
          class="be-btn"
          :class="{ 'is-active': background === bg.key }"
          :title="`Фон: ${bg.key}`"
          :disabled="!canEdit"
          @click="setBackground(bg.key)"
        >
          <span class="material-symbols-outlined">{{ bg.icon }}</span>
        </button>
        <button
          type="button"
          class="be-btn"
          :class="{ 'is-active': layersOpen }"
          title="Слои"
          aria-label="Слои"
          @click="layersOpen = !layersOpen"
        >
          <span class="material-symbols-outlined">layers</span>
        </button>
        <button
          type="button"
          class="be-btn"
          :class="{ 'is-active': propsOpen }"
          title="Свойства выделенного"
          aria-label="Свойства выделенного"
          @click="propsOpen = !propsOpen"
        >
          <span class="material-symbols-outlined">tune</span>
        </button>
        <button
          type="button"
          class="be-btn"
          :class="{ 'is-active': framesOpen }"
          title="Кадры анимации"
          aria-label="Кадры анимации"
          :disabled="!canEdit && !frames.length"
          @click="toggleFrames"
        >
          <span class="material-symbols-outlined">animation</span>
        </button>
        <button type="button" class="be-btn" title="Скачать" aria-label="Скачать" @click="openExportMenu">
          <span class="material-symbols-outlined">download</span>
        </button>
        <button
          v-if="canEdit"
          type="button"
          class="be-btn"
          title="Поделиться"
          aria-label="Поделиться"
          @click="shareOpen = true"
        >
          <span class="material-symbols-outlined">share</span>
        </button>
      </div>
    </header>

    <div class="be-body">
      <BrandLoader v-if="loading" :size="64" class="be-loader" />
      <template v-else>
        <BoardCanvas
          ref="canvasRef"
          :scene="scene"
          :tool="tool"
          :color="color"
          :fill="fill"
          :width="strokeWidth"
          :opacity="opacity"
          :text-size="textSize"
          :eraser-size="eraserSize"
          :erase-mode="eraseMode"
          :polygon-sides="polygonSides"
          :polygon-star="polygonStar"
          :read-only="!canEdit"
          :peers="peers"
          :active-layer="activeLayer"
          :frame="framesOpen ? currentFrame : ''"
          :onion="onion"
          :me="me"
          @update:scene="onSceneUpdate"
          @ops="onCanvasOps"
          @pointer-move="sendCursor"
          @select-change="(ids) => (selection = ids)"
          @comment-open="onCommentOpen"
          @pick-color="(hex) => (color = hex)"
          @request-tool="(key) => (tool = key)"
        />

        <CommentPopup
          :comment="activeComment"
          :anchor="commentAnchor"
          :me="me"
          :read-only="!canEdit"
          @update="onCommentUpdate"
          @delete="onCommentDelete"
          @close="activeComment = null"
        />

        <!-- Панели правого края: слои постоянные, свойства приходят с
             выделением. Каждая — самостоятельная колонка, поэтому открытая
             панель слоёв просто сдвигается левее, а не делит место. -->
        <LayersPanel
          v-if="layersOpen"
          class="be-layers"
          :class="{ 'has-props': showProps }"
          :scene="scene"
          :active-layer="activeLayer"
          :selection="selection"
          :read-only="!canEdit"
          @update:layers="onLayersUpdate"
          @update:objects="onObjectsUpdate"
          @update:active-layer="(id) => (activeLayer = id)"
          @select="(ids) => canvasRef?.setSelection(ids)"
          @close="layersOpen = false"
        />

        <PropertiesPanel
          v-if="showProps"
          class="be-props"
          :selection="selectedObjects"
          :read-only="!canEdit"
          @apply="(patch) => canvasRef?.applyStyle(patch)"
          @align="(kind) => canvasRef?.alignSelection(kind)"
          @bool="(op) => canvasRef?.booleanSelection(op)"
          @bool-release="canvasRef?.releaseBoolean()"
          @mask="canvasRef?.toggleMaskObject()"
          @order="(front) => canvasRef?.reorderSelected(front)"
          @to-vector="canvasRef?.convertToVector()"
          @text-path="canvasRef?.attachTextToPath()"
          @close="propsOpen = false"
        />

        <div v-if="framesOpen && frames.length" class="be-frames">
          <FrameTimeline
            :scene="scene"
            :frame="currentFrame"
            :onion-depth="onionDepth"
            :read-only="!canEdit"
            :exporting="exportingVideo"
            @update:scene="onFramesUpdate"
            @update:frame="(id) => (currentFrame = id)"
            @update:onion-depth="(v) => (onionDepth = v)"
            @update:playing="(v) => (playing = v)"
            @export="openAnimationExport"
            @close="toggleFrames"
          />
        </div>

        <div
          v-if="canEdit"
          class="be-toolbar"
          :class="{ 'has-frames': framesOpen && frames.length, 'has-side': showProps || layersOpen }"
        >
          <BoardToolbar
            v-model:tool="tool"
            v-model:color="color"
            v-model:fill="fill"
            v-model:width="strokeWidth"
            v-model:opacity="opacity"
            v-model:text-size="textSize"
            v-model:eraser-size="eraserSize"
            v-model:erase-mode="eraseMode"
            v-model:polygon-sides="polygonSides"
            v-model:polygon-star="polygonStar"
            :zoom="zoom"
            :has-selection="!!selection.length"
            @zoom-in="canvasRef?.zoomIn()"
            @zoom-out="canvasRef?.zoomOut()"
            @fit="canvasRef?.fitToContent()"
            @add-image="pickImage"
            @delete-selected="canvasRef?.removeSelected()"
          />
        </div>
      </template>
    </div>

    <input ref="fileInput" type="file" accept="image/*" hidden @change="onImagePicked" />

    <ContextMenu
      :visible="boardMenu.visible"
      :x="boardMenu.x"
      :y="boardMenu.y"
      :items="boardMenuItems"
      @select="onBoardMenu"
      @close="boardMenu.visible = false"
    />

    <ContextMenu
      :visible="exportMenu.visible"
      :x="exportMenu.x"
      :y="exportMenu.y"
      :items="BOARD_EXPORT_ITEMS"
      @select="exportBoard"
      @close="exportMenu.visible = false"
    />

    <ExportAnimationDialog
      v-model="exportOpen"
      :frames="frames"
      :fps="normalizeScene(scene).animation?.fps || 12"
      :busy="exportingVideo"
      :progress="exportProgress"
      @export="exportAnimation"
    />

    <ShareDialog v-if="board" v-model="shareOpen" subject-type="board" :subject-id="board.id" />
  </AppPage>
</template>

<style scoped>
/* Каркас — AppPage bare: панели под холстом нет, поля и зазор задаёт сам
   редактор (холсту нужны свои, а не общие поля раздела). */
.be :deep(.page-body) {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px;
}

.be-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border: 1px solid var(--glass-edge);
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
}

.be-title {
  flex: 1;
  min-width: 0;
  border: none;
  background: transparent;
  font-size: 1.05rem;
  font-weight: 600;
}

.be-state {
  font-size: 12px;
  color: var(--color-text-muted);
  white-space: nowrap;
}

.be-peers { display: flex; gap: 4px; }

.be-peer {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 26px;
  max-width: 26px;
  min-height: 26px;
  max-height: 26px;
  border-radius: 50%;
  color: var(--color-on-primary);
  font-size: 12px;
  font-weight: 600;
}

.be-actions { display: flex; gap: 2px; }

.be-back,
.be-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 34px;
  max-width: 34px;
  min-height: 34px;
  max-height: 34px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
}

.be-back:hover,
.be-btn:hover:not(:disabled):not(.is-active) { background: var(--color-surface-variant); }
.be-btn:disabled { opacity: 0.4; cursor: default; }
.be-btn.is-active,
.be-btn.is-active:hover { background: var(--color-primary); color: var(--color-on-primary); }
.be-btn .material-symbols-outlined,
.be-back .material-symbols-outlined { font-size: 20px; }

.be-body {
  position: relative;
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  /* Панели редактора считают своё место по ширине ОКНА раздела. */
  container-type: inline-size;
}

.be-props,
.be-layers {
  position: absolute;
  top: 12px;
  bottom: 12px;
  right: 12px;
}

/* Свойства всегда у самого края; слои уступают им место, когда те открыты. */
.be-layers.has-props { right: 292px; }

/* Узкое окно: обе панели рядом не помещаются — слои уходят под свойства и
   прокручиваются вместе с ними. Считаем по ширине ОКНА (@container), а не
   экрана: доска живёт в окне рабочего стола. */
@container (max-width: 780px) {
  .be-layers.has-props {
    right: 12px;
    bottom: auto;
    max-height: 40%;
  }

  .be-props { top: auto; height: 55%; }
  /* В узком окне панель занимает низ целиком — двигать инструменты некуда. */
  .be-toolbar.has-side { padding-right: 12px; }
}

.be-frames {
  position: absolute;
  left: 12px;
  right: 12px;
  bottom: 12px;
  pointer-events: none;
}

.be-toolbar {
  /* Центрируем флексом, БЕЗ transform: трансформированный предок образует
     backdrop root, и размытие панели перестало бы захватывать холст. */
  position: absolute;
  left: 0;
  right: 0;
  bottom: 12px;
  display: flex;
  justify-content: center;
  padding: 0 12px;
  /* Клики мимо панели должны доходить до холста. */
  pointer-events: none;
}

/* Лента кадров занимает низ, панели — правый край: инструменты уступают им
   место, иначе панель накрывала бы половину ряда кнопок. */
.be-toolbar.has-frames { bottom: 168px; }
.be-toolbar.has-side { padding-right: 300px; }

.be-loader { margin: auto; }

@media (max-width: 768px) {
  .be :deep(.page-body) { padding: 4px; gap: 4px; }
  /* Шапка — одна строка: назад, название и меню. Ряд кнопок сюда не помещался
     и переносился второй строкой, отбирая место у названия. */
  .be-head { flex-wrap: nowrap; }
  .be-title { min-width: 80px; }
  /* Соавторов и статус в узкой шапке не показываем — они есть в самой доске. */
  .be-peers, .be-state { display: none; }
  .be-toolbar.has-frames { bottom: 156px; }
}
</style>
