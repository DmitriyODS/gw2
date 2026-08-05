<script setup>
/* Экран доски: холст на всю площадь, плавающий тулбар, автосохранение сцены и
   совместное рисование. Миниатюра плитки снимается с самого холста после
   паузы в рисовании — список досок выглядит как галерея эскизов. */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import InputText from 'primevue/inputtext'
import AppPage from '@/components/ui/AppPage.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import BoardCanvas from '@/components/boards/BoardCanvas.vue'
import BoardToolbar from '@/components/boards/BoardToolbar.vue'
import CommentPopup from '@/components/boards/CommentPopup.vue'
import LayersPanel from '@/components/boards/LayersPanel.vue'
import ShareDialog from '@/components/boards/ShareDialog.vue'
import * as api from '@/api/boards.js'
import { emptyScene, normalizeScene } from '@/utils/boardScene.js'
import { saveBlob, sceneToJpeg, sceneToPdf, sceneToPng } from '@/utils/boardExport.js'
import { useBoardCollab } from '@/composables/useBoardCollab.js'
import { useAuthStore } from '@/stores/auth.js'
import { useBoardsStore } from '@/stores/boards.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const route = useRoute()
const router = useRouter()
const boards = useBoardsStore()
const auth = useAuthStore()
const notify = useNotificationsStore()

const boardId = computed(() => Number(route.params.id))

const board = ref(null)
const scene = ref(emptyScene())
const title = ref('')
const loading = ref(true)
const saving = ref(false)
const shareOpen = ref(false)

const canvasRef = ref(null)
const fileInput = ref(null)

const tool = ref('pen')
const color = ref('ink')
const fill = ref('')
const strokeWidth = ref(4)
const textSize = ref(18)
const selection = ref([])
const activeLayer = ref('')
const layersOpen = ref(false)
// Открытое обсуждение: сам комментарий и точка на экране, где стоит булавка.
const activeComment = ref(null)
const commentAnchor = ref({ x: 0, y: 0 })
const exportMenu = ref({ visible: false, x: 0, y: 0 })

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
      const blob = await sceneToPng(scene.value, { scale: 0.6 })
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

const EXPORT_ITEMS = [
  { label: 'Картинка PNG', icon: 'image', action: 'png' },
  { label: 'Картинка JPG', icon: 'photo', action: 'jpg' },
  { label: 'Документ PDF', icon: 'picture_as_pdf', action: 'pdf' },
  { divider: true },
  { label: 'Вектор SVG', icon: 'draft', action: 'svg' },
  { label: 'Сцена JSON', icon: 'data_object', action: 'json' },
]

function openExportMenu(e) {
  exportMenu.value = { visible: true, x: e.clientX, y: e.clientY }
}

async function exportBoard(format) {
  const name = title.value || 'Доска'
  try {
    // Растр и PDF рисует холст (в них попадают и картинки сцены), svg/json —
    // сервер: он же кладёт в SVG сами файлы картинок.
    if (format === 'png' || format === 'jpg' || format === 'pdf') {
      const make = { png: sceneToPng, jpg: sceneToJpeg, pdf: sceneToPdf }[format]
      const blob = await make(scene.value)
      if (!blob) {
        notify.warn('На доске пока нечего сохранять')
        return
      }
      saveBlob(blob, `${name}.${format}`)
      return
    }
    saveBlob(await api.exportBoard(boardId.value, format), `${name}.${format}`)
  } catch {
    notify.error('Не удалось выгрузить доску')
  }
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

      <div class="be-actions">
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
          :text-size="textSize"
          :read-only="!canEdit"
          :peers="peers"
          :active-layer="activeLayer"
          :me="me"
          @update:scene="onSceneUpdate"
          @ops="onCanvasOps"
          @pointer-move="sendCursor"
          @select-change="(ids) => (selection = ids)"
          @comment-open="onCommentOpen"
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

        <LayersPanel
          v-if="layersOpen"
          class="be-layers"
          :scene="scene"
          :active-layer="activeLayer"
          :read-only="!canEdit"
          @update:layers="onLayersUpdate"
          @update:active-layer="(id) => (activeLayer = id)"
          @close="layersOpen = false"
        />

        <div v-if="canEdit" class="be-toolbar">
          <BoardToolbar
            v-model:tool="tool"
            v-model:color="color"
            v-model:fill="fill"
            v-model:width="strokeWidth"
            v-model:text-size="textSize"
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
      :visible="exportMenu.visible"
      :x="exportMenu.x"
      :y="exportMenu.y"
      :items="EXPORT_ITEMS"
      @select="exportBoard"
      @close="exportMenu.visible = false"
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
}

.be-layers {
  position: absolute;
  top: 12px;
  right: 12px;
  max-height: calc(100% - 24px);
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

.be-loader { margin: auto; }

@media (max-width: 768px) {
  .be :deep(.page-body) { padding: 4px; gap: 4px; }
  .be-actions { flex-wrap: wrap; justify-content: flex-end; }
}
</style>
