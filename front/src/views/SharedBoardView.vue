<script setup>
/* Публичный просмотр доски по ссылке-коду: без авторизации. Режим ссылки
   решает сервер — view открывает холст только на чтение, edit разрешает
   рисовать (правки уходят PUT'ом по тому же коду). */
import { computed, onBeforeUnmount, onMounted, ref, shallowRef } from 'vue'
import { useRoute } from 'vue-router'
import AppButton from '@/components/ui/AppButton.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import BoardCanvas from '@/components/boards/BoardCanvas.vue'
import BoardToolbar from '@/components/boards/BoardToolbar.vue'
import CommentPopup from '@/components/boards/CommentPopup.vue'
import FrameTimeline from '@/components/boards/FrameTimeline.vue'
import { getSharedBoard, updateSharedBoard } from '@/api/boards.js'
import { emptyScene, normalizeScene } from '@/utils/boardScene.js'
import { useSceneHistory } from '@/composables/useSceneHistory.js'
import { useScopedHotkeys } from '@/composables/useScopedHotkeys.js'

const route = useRoute()
const code = computed(() => String(route.params.code || ''))

const board = ref(null)
// Сцена неизменяема (новый объект на каждую правку) — глубокие прокси не нужны.
const scene = shallowRef(emptyScene())
const loading = ref(true)
const failed = ref(false)
const saving = ref(false)

const canvasRef = ref(null)
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
// Анимированную доску гость смотрит покадрово — с проигрыванием и без правок.
const currentFrame = ref('')

let saveTimer = null
let dirty = false
let revision = 0
let saveInFlight = null
let lastGesture = 0
const history = useSceneHistory()

/* Обсуждения гость читает, но не ведёт: имени у него нет. Ветку ищем в живой
   сцене по id — так видны и свежие правки. */
const commentId = ref('')
const commentAnchor = ref({ x: 0, y: 0 })
const activeComment = computed(() => (commentId.value
  ? normalizeScene(scene.value).objects.find((o) => o.id === commentId.value) || null
  : null))

function openComment(comment) {
  commentId.value = comment.id
  const cam = canvasRef.value?.camera
  if (cam) commentAnchor.value = { x: (comment.x - cam.x) * cam.scale + 36, y: (comment.y - cam.y) * cam.scale }
}

const canEdit = computed(() => board.value?.my_access === 'edit')
const zoom = computed(() => canvasRef.value?.camera?.scale || 1)
const frames = computed(() => normalizeScene(scene.value).animation?.frames || [])

async function load() {
  loading.value = true
  try {
    const data = await getSharedBoard(code.value)
    board.value = data
    scene.value = normalizeScene(data.scene)
    if (frames.value.length) currentFrame.value = frames.value[0].id
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}

function onSceneUpdate(next, meta) {
  if (!canEdit.value) return
  // Жест холста — один шаг отмены (см. редактор).
  const gesture = meta?.gesture || 0
  if (!gesture || gesture !== lastGesture) history.push(scene.value)
  lastGesture = gesture
  setScene(next)
}

function setScene(next) {
  scene.value = normalizeScene(next)
  dirty = true
  revision += 1
  clearTimeout(saveTimer)
  saveTimer = setTimeout(save, 900)
}

function undo() {
  const prev = history.undo(scene.value)
  if (prev && canEdit.value) setScene(prev)
}

function redo() {
  const next = history.redo(scene.value)
  if (next && canEdit.value) setScene(next)
}

// Сохранения строго по одному, чистым — только если после снимка не рисовали.
async function save() {
  if (saveInFlight) {
    await saveInFlight
    return save()
  }
  if (!dirty || !canEdit.value) return
  const snapshot = revision
  saving.value = true
  saveInFlight = updateSharedBoard(code.value, { scene: scene.value })
    .then(() => { if (revision === snapshot) dirty = false })
    .catch(() => {
      // Троттлинг анонимных правок на сервере — просто пробуем позже.
      clearTimeout(saveTimer)
      saveTimer = setTimeout(save, 3000)
    })
    .finally(() => {
      saving.value = false
      saveInFlight = null
    })
  await saveInFlight
}

const root = ref(null)
const { arm } = useScopedHotkeys(() => root.value, (e) => {
  if (!(e.ctrlKey || e.metaKey) || !canEdit.value) return
  const key = e.key.toLowerCase()
  if (key === 'z' && !e.shiftKey) { e.preventDefault(); undo() }
  else if ((key === 'z' && e.shiftKey) || key === 'y') { e.preventDefault(); redo() }
})

onMounted(() => {
  load()
  arm()
})
onBeforeUnmount(() => {
  clearTimeout(saveTimer)
  save()
})
</script>

<template>
  <div ref="root" class="sb" data-board-scope>
    <header class="sb-head">
      <span class="material-symbols-outlined sb-logo">gesture</span>
      <h1 class="sb-title">{{ board?.title || 'Доска' }}</h1>
      <span class="sb-state">
        <template v-if="saving">Сохраняем…</template>
        <template v-else-if="canEdit">Можно рисовать</template>
        <template v-else-if="board">Только просмотр</template>
      </span>
      <template v-if="canEdit">
        <AppButton variant="icon" icon="undo" label="Отменить" title="Отменить" :disabled="!history.canUndo.value" @click="undo" />
        <AppButton variant="icon" icon="redo" label="Повторить" title="Повторить" :disabled="!history.canRedo.value" @click="redo" />
      </template>
    </header>

    <div class="sb-body">
      <BrandLoader v-if="loading" :size="64" />
      <EmptyState
        v-else-if="failed"
        icon="link_off"
        tone="error"
        title="Ссылка не действует"
        subtitle="Возможно, доступ по ней отозвали или доску удалили."
      />
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
          :frame="currentFrame"
          :read-only="!canEdit"
          @update:scene="onSceneUpdate"
          @select-change="(ids) => (selection = ids)"
          @pick-color="(hex) => (color = hex)"
          @request-tool="(key) => (tool = key)"
          @comment-open="openComment"
        />

        <CommentPopup
          :comment="activeComment"
          :anchor="commentAnchor"
          read-only
          @close="commentId = ''"
        />

        <div v-if="frames.length" class="sb-frames">
          <FrameTimeline
            :scene="scene"
            :frame="currentFrame"
            :onion-depth="0"
            read-only
            @update:scene="onSceneUpdate"
            @update:frame="(id) => (currentFrame = id)"
            @close="currentFrame = frames[0].id"
          />
        </div>
        <div v-if="canEdit" class="sb-toolbar" :class="{ 'has-frames': frames.length }">
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
            :images="false"
            :comments="false"
            @zoom-in="canvasRef?.zoomIn()"
            @zoom-out="canvasRef?.zoomOut()"
            @fit="canvasRef?.fitToContent()"
            @delete-selected="canvasRef?.removeSelected()"
          />
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.sb {
  display: flex;
  flex-direction: column;
  height: 100vh;
  height: 100dvh;
  gap: 8px;
  padding: 8px;
}

.sb-head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border: 1px solid var(--glass-edge);
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
}

.sb-logo { color: var(--color-primary); }
.sb-title {
  flex: 1;
  min-width: 0;
  margin: 0;
  font-size: 1.05rem;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.sb-state { font-size: 12px; color: var(--color-text-dim); }

.sb-body {
  position: relative;
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.sb-frames {
  position: absolute;
  left: 12px;
  right: 12px;
  bottom: 12px;
  pointer-events: none;
}

.sb-toolbar {
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

/* Лента кадров занимает низ — панель инструментов уступает ей место. */
.sb-toolbar.has-frames { bottom: 168px; }
</style>
