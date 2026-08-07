<script setup>
/* Публичный просмотр доски по ссылке-коду: без авторизации. Режим ссылки
   решает сервер — view открывает холст только на чтение, edit разрешает
   рисовать (правки уходят PUT'ом по тому же коду). */
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import BrandLoader from '@/components/common/BrandLoader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import BoardCanvas from '@/components/boards/BoardCanvas.vue'
import BoardToolbar from '@/components/boards/BoardToolbar.vue'
import { getSharedBoard, updateSharedBoard } from '@/api/boards.js'
import { emptyScene, normalizeScene } from '@/utils/boardScene.js'

const route = useRoute()
const code = computed(() => String(route.params.code || ''))

const board = ref(null)
const scene = ref(emptyScene())
const loading = ref(true)
const failed = ref(false)
const saving = ref(false)

const canvasRef = ref(null)
const tool = ref('pen')
const color = ref('ink')
const fill = ref('')
const strokeWidth = ref(4)
const textSize = ref(18)
const selection = ref([])

let saveTimer = null
let dirty = false

const canEdit = computed(() => board.value?.my_access === 'edit')
const zoom = computed(() => canvasRef.value?.camera?.scale || 1)

async function load() {
  loading.value = true
  try {
    const data = await getSharedBoard(code.value)
    board.value = data
    scene.value = normalizeScene(data.scene)
  } catch {
    failed.value = true
  } finally {
    loading.value = false
  }
}

function onSceneUpdate(next) {
  if (!canEdit.value) return
  scene.value = next
  dirty = true
  clearTimeout(saveTimer)
  saveTimer = setTimeout(save, 900)
}

async function save() {
  if (!dirty || !canEdit.value) return
  saving.value = true
  try {
    await updateSharedBoard(code.value, { scene: scene.value })
    dirty = false
  } catch {
    // Троттлинг анонимных правок на сервере — просто пробуем позже.
    saveTimer = setTimeout(save, 3000)
  } finally {
    saving.value = false
  }
}

onMounted(load)
onBeforeUnmount(() => {
  clearTimeout(saveTimer)
  save()
})
</script>

<template>
  <div class="sb">
    <header class="sb-head">
      <span class="material-symbols-outlined sb-logo">gesture</span>
      <h1 class="sb-title">{{ board?.title || 'Доска' }}</h1>
      <span class="sb-state">
        <template v-if="saving">Сохраняем…</template>
        <template v-else-if="canEdit">Можно рисовать</template>
        <template v-else-if="board">Только просмотр</template>
      </span>
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
          :text-size="textSize"
          :read-only="!canEdit"
          @update:scene="onSceneUpdate"
          @select-change="(ids) => (selection = ids)"
        />
        <div v-if="canEdit" class="sb-toolbar">
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
.sb-title { flex: 1; min-width: 0; margin: 0; font-size: 1.05rem; font-weight: 600; }
.sb-state { font-size: 12px; color: var(--color-text-muted); }

.sb-body {
  position: relative;
  flex: 1;
  min-height: 0;
  display: flex;
  align-items: center;
  justify-content: center;
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
</style>
