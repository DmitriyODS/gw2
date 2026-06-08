<script setup>
import { ref, computed, onMounted } from 'vue'
import { useTasksStore } from '@/stores/tasks.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { getStages } from '@/api/stages.js'
import TaskCard from '@/components/tasks/TaskCard.vue'

const emit = defineEmits(['open-task', 'toggle-favorite', 'set-color', 'start-unit', 'stop-unit', 'context-menu'])

const tasks = useTasksStore()
const notify = useNotificationsStore()

const stages = ref([])
const draggingId = ref(null)
const hoverStageId = ref(undefined) // undefined — не над колонкой, null — над «Без этапа»

const columns = computed(() => {
  const noStage = { id: null, name: 'Без этапа', color: 'blue' }
  return [noStage, ...stages.value]
})

function tasksOf(stageId) {
  return tasks.tasks.filter((t) => (t.stage_id ?? null) === (stageId ?? null))
}

function colHeaderStyle(stage) {
  if (!stage?.color || stage.id == null) {
    return {
      background: 'var(--color-surface-high)',
      color: 'var(--color-on-surface-variant)',
    }
  }
  return {
    background: `var(--tag-${stage.color}-surface)`,
    color: `var(--tag-${stage.color}-accent)`,
  }
}

function onDragStart(e, task) {
  draggingId.value = task.id
  // FF требует setData, иначе drag не активируется.
  try { e.dataTransfer.setData('text/plain', String(task.id)) } catch {}
  e.dataTransfer.effectAllowed = 'move'
}

function onDragEnd() {
  draggingId.value = null
  hoverStageId.value = undefined
}

function onDragOver(e) {
  e.preventDefault()
  e.dataTransfer.dropEffect = 'move'
}

function onDragEnter(stageId) {
  hoverStageId.value = stageId
}

function onDragLeave(stageId) {
  if (hoverStageId.value === stageId) hoverStageId.value = undefined
}

async function onDrop(e, stageId) {
  e.preventDefault()
  hoverStageId.value = undefined
  const id = draggingId.value
  draggingId.value = null
  if (id == null) return
  const task = tasks.tasks.find((t) => t.id === id)
  if (!task) return
  if ((task.stage_id ?? null) === (stageId ?? null)) return
  try {
    await tasks.dragMoveStage(id, stageId)
  } catch (e) {
    notify.error(e?.message || 'Не удалось переместить')
  }
}

async function load() {
  try {
    const data = await getStages()
    stages.value = Array.isArray(data) ? data : (data.items ?? [])
  } catch {
    stages.value = []
  }
}

onMounted(load)
</script>

<template>
  <div class="kanban">
    <div
      v-for="col in columns"
      :key="col.id ?? 'none'"
      class="kanban-col"
      :class="{ 'drag-over': hoverStageId === col.id && hoverStageId !== undefined }"
      @dragover.prevent="onDragOver"
      @dragenter="onDragEnter(col.id)"
      @dragleave="onDragLeave(col.id)"
      @drop="onDrop($event, col.id)"
    >
      <div class="kanban-col-head" :style="colHeaderStyle(col)">
        <span class="kanban-col-name">{{ col.name }}</span>
        <span class="kanban-col-count">{{ tasksOf(col.id).length }}</span>
      </div>
      <div class="kanban-col-body">
        <div
          v-for="t in tasksOf(col.id)"
          :key="t.id"
          class="kanban-card-wrap"
          :class="{ dragging: draggingId === t.id }"
          draggable="true"
          @dragstart="onDragStart($event, t)"
          @dragend="onDragEnd"
        >
          <TaskCard
            :task="t"
            view="grid"
            @click="emit('open-task', t)"
            @toggle-favorite="emit('toggle-favorite', $event)"
            @set-color="emit('set-color', $event)"
            @start-unit="emit('start-unit', $event)"
            @stop-unit="emit('stop-unit', $event)"
            @context-menu="emit('context-menu', $event)"
          />
        </div>
        <div v-if="!tasksOf(col.id).length" class="kanban-col-empty">
          Пусто
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.kanban {
  display: flex;
  gap: 14px;
  align-items: stretch;
  overflow-x: auto;
  min-height: 0;
  padding-bottom: 4px;
}

.kanban-col {
  flex: 0 0 280px;
  display: flex;
  flex-direction: column;
  background: var(--color-surface-low);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg, 16px);
  overflow: hidden;
  transition: border-color 0.15s, background 0.15s;
}
.kanban-col.drag-over {
  border-color: var(--color-primary);
  background: color-mix(in oklab, var(--color-primary) 6%, var(--color-surface-low));
}

.kanban-col-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  font-weight: 700;
  font-size: 13px;
  flex-shrink: 0;
}
.kanban-col-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.kanban-col-count {
  background: color-mix(in oklab, currentColor 18%, transparent);
  padding: 2px 9px;
  border-radius: var(--radius-full, 999px);
  font-size: 12px;
}

.kanban-col-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 10px;
  flex: 1;
  min-height: 80px;
  overflow-y: auto;
}

.kanban-card-wrap { cursor: grab; }
.kanban-card-wrap.dragging { opacity: 0.5; cursor: grabbing; }

.kanban-col-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  font-size: 12px;
  color: var(--color-on-surface-variant);
  opacity: 0.7;
}
</style>
