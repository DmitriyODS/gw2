<template>
  <Teleport to="body">
    <div class="tfw" :class="{ minimized, narrow }" :style="winStyle">
      <!-- Шапка — за неё перетаскиваем -->
      <header class="tfw-head" @pointerdown="onDragStart">
        <span class="material-symbols-outlined tfw-grip">drag_indicator</span>
        <div class="tfw-title-wrap">
          <span class="tfw-eyebrow">Задача №{{ taskId }}</span>
          <span class="tfw-title">{{ task?.name || taskName || `Задача #${taskId}` }}</span>
        </div>
        <button class="tfw-icon-btn" :title="minimized ? 'Развернуть' : 'Свернуть'" @click="minimized = !minimized">
          <span class="material-symbols-outlined">{{ minimized ? 'expand_content' : 'remove' }}</span>
        </button>
        <button class="tfw-icon-btn" title="Закрыть" @click="$emit('close')">
          <span class="material-symbols-outlined">close</span>
        </button>
      </header>

      <div v-show="!minimized" class="tfw-cols">
        <div v-if="loading" class="tfw-loading tfw-fill">
          <span class="material-symbols-outlined spinning">progress_activity</span>
          Загрузка задачи…
        </div>

        <template v-else>
          <!-- Левая панель: информация о задаче -->
          <div class="tfw-left">
            <span class="tfw-badge" :class="task?.is_archived ? 'archived' : 'active'">
              <span class="material-symbols-outlined">{{ task?.is_archived ? 'inventory_2' : 'play_circle' }}</span>
              {{ task?.is_archived ? 'В архиве' : 'В работе' }}
            </span>

            <div class="field-box">
              <div class="field-label">Заказчик</div>
              <div class="field-value">{{ task?.department?.name || '—' }}</div>
            </div>

            <div class="fields-row">
              <div class="field-box half">
                <div class="field-label">Поступила</div>
                <div class="field-value with-icon">
                  <span class="material-symbols-outlined field-icon">calendar_today</span>
                  {{ fmtDate(task?.received_at) }}
                </div>
              </div>
              <div class="field-box half">
                <div class="field-label">Создана</div>
                <div class="field-value with-icon">
                  <span class="material-symbols-outlined field-icon">calendar_today</span>
                  {{ fmtDate(task?.created_at) }}
                </div>
              </div>
            </div>

            <div class="field-box">
              <div class="field-label">Ответственный</div>
              <div class="field-value responsible-value">
                <template v-if="task?.responsible">
                  <img :src="avatarOf(task.responsible)" class="responsible-avatar" alt="" />
                  <span class="responsible-name">{{ task.responsible.fio }}</span>
                </template>
                <span v-else class="text-dim">Не назначен</span>
              </div>
            </div>

            <div v-if="task?.deadline" class="field-box">
              <div class="field-label">Дедлайн</div>
              <div class="field-value with-icon">
                <span class="material-symbols-outlined field-icon">calendar_today</span>
                {{ fmtDate(task.deadline) }}
              </div>
            </div>

            <div class="field-box">
              <div class="field-label">Создатель задачи</div>
              <div class="field-value">{{ task?.author?.fio || '—' }}</div>
            </div>
          </div>

          <!-- Правая панель: комментарии / юниты -->
          <div class="tfw-right">
            <div class="tfw-tabs">
              <button class="tfw-tab" :class="{ active: tab === 'comments' }" @click="tab = 'comments'">
                <span class="material-symbols-outlined">forum</span> Комментарии
              </button>
              <button class="tfw-tab" :class="{ active: tab === 'units' }" @click="openUnits">
                <span class="material-symbols-outlined">timer</span> Юниты
              </button>
            </div>

            <div class="tfw-tab-body">
              <TaskComments v-if="tab === 'comments'" :task-id="taskId" />
              <template v-else>
                <div v-if="unitsLoading" class="tfw-loading">
                  <span class="material-symbols-outlined spinning">progress_activity</span>
                  Загрузка юнитов…
                </div>
                <div v-else-if="!units.length" class="tfw-empty">
                  <span class="material-symbols-outlined">hourglass_empty</span>
                  Юнитов пока нет
                </div>
                <ul v-else class="tfw-units">
                  <li v-for="u in units" :key="u.id" class="tfw-unit">
                    <div class="tfw-unit-main">
                      <span class="tfw-unit-name">{{ u.name }}</span>
                      <span class="tfw-unit-dur">{{ unitDuration(u) }}</span>
                    </div>
                    <div class="tfw-unit-meta">
                      <span v-if="u.unit_type?.name" class="tfw-unit-type">{{ u.unit_type.name }}</span>
                      <span>{{ fmtDate(u.datetime_start) }}</span>
                      <span v-if="u.user?.fio">· {{ u.user.fio }}</span>
                      <span v-if="!u.datetime_end" class="tfw-unit-live">в работе</span>
                    </div>
                  </li>
                </ul>
              </template>
            </div>
          </div>
        </template>
      </div>

      <!-- Уголок изменения размера -->
      <div v-show="!minimized" class="tfw-resize" @pointerdown="onResizeStart" title="Изменить размер"></div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import TaskComments from '@/components/tasks/TaskComments.vue'
import { getTask } from '@/api/tasks.js'
import { getUnits } from '@/api/units.js'

const props = defineProps({
  taskId: { type: Number, required: true },
  taskName: { type: String, default: '' },
})
defineEmits(['close'])

const MIN_W = 480
const MIN_H = 340

const task = ref(null)
const loading = ref(true)
const tab = ref('comments')
const units = ref([])
const unitsLoading = ref(false)
const unitsLoaded = ref(false)
const minimized = ref(false)

const pos = ref(null)
const size = ref({ w: 760, h: 560 })
const narrow = computed(() => size.value.w < 600)

onMounted(() => {
  const w = Math.min(760, window.innerWidth - 32)
  const h = Math.min(560, window.innerHeight - 120)
  size.value = { w: Math.max(MIN_W, w), h: Math.max(MIN_H, h) }
  pos.value = { left: Math.max(8, window.innerWidth - size.value.w - 24), top: 72 }
  load()
})

const winStyle = computed(() => {
  const s = {}
  if (pos.value) {
    s.left = `${pos.value.left}px`
    s.top = `${pos.value.top}px`
    s.right = 'auto'
    s.bottom = 'auto'
  }
  s.width = `${size.value.w}px`
  if (!minimized.value) s.height = `${size.value.h}px`
  return s
})

async function load() {
  loading.value = true
  try {
    task.value = await getTask(props.taskId)
  } catch {
    task.value = null
  } finally {
    loading.value = false
  }
}

function openUnits() {
  tab.value = 'units'
  if (unitsLoaded.value) return
  unitsLoaded.value = true
  loadUnits()
}

async function loadUnits() {
  unitsLoading.value = true
  try {
    const data = await getUnits(props.taskId)
    units.value = Array.isArray(data) ? data : (data.units ?? data.items ?? [])
  } catch {
    units.value = []
  } finally {
    unitsLoading.value = false
  }
}

function avatarOf(a) {
  if (!a) return ''
  return a.avatar_path ? `/uploads/${a.avatar_path}` : `/api/users/${a.id}/identicon`
}

function fmtDate(d) {
  return d ? new Date(d).toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' }) : '—'
}

function unitDuration(u) {
  const end = u.datetime_end ? new Date(u.datetime_end) : new Date()
  const mins = Math.max(0, Math.round((end - new Date(u.datetime_start)) / 60000))
  const h = Math.floor(mins / 60)
  const m = mins % 60
  if (h && m) return `${h} ч ${m} мин`
  if (h) return `${h} ч`
  return `${m} мин`
}

/* ── Перетаскивание ──────────────────────────────────────────── */
let dragging = false
let dragOffset = { x: 0, y: 0 }

function onDragStart(e) {
  if (e.target.closest('button')) return
  const el = e.currentTarget.closest('.tfw')
  if (!el) return
  const rect = el.getBoundingClientRect()
  dragOffset = { x: e.clientX - rect.left, y: e.clientY - rect.top }
  pos.value = { left: rect.left, top: rect.top }
  dragging = true
  window.addEventListener('pointermove', onDragMove)
  window.addEventListener('pointerup', onDragEnd)
  e.preventDefault()
}

function onDragMove(e) {
  if (!dragging) return
  const el = document.querySelector('.tfw')
  const w = el?.offsetWidth || size.value.w
  const h = el?.offsetHeight || size.value.h
  const left = Math.max(8, Math.min(e.clientX - dragOffset.x, window.innerWidth - w - 8))
  const top = Math.max(8, Math.min(e.clientY - dragOffset.y, window.innerHeight - h - 8))
  pos.value = { left, top }
}

function onDragEnd() {
  dragging = false
  window.removeEventListener('pointermove', onDragMove)
  window.removeEventListener('pointerup', onDragEnd)
}

/* ── Изменение размера ───────────────────────────────────────── */
let resizing = false
let resizeStart = { x: 0, y: 0, w: 0, h: 0 }

function onResizeStart(e) {
  resizing = true
  resizeStart = { x: e.clientX, y: e.clientY, w: size.value.w, h: size.value.h }
  window.addEventListener('pointermove', onResizeMove)
  window.addEventListener('pointerup', onResizeEnd)
  e.preventDefault()
  e.stopPropagation()
}

function onResizeMove(e) {
  if (!resizing) return
  const left = pos.value?.left ?? 0
  const top = pos.value?.top ?? 0
  const w = Math.max(MIN_W, Math.min(resizeStart.w + (e.clientX - resizeStart.x), window.innerWidth - left - 8))
  const h = Math.max(MIN_H, Math.min(resizeStart.h + (e.clientY - resizeStart.y), window.innerHeight - top - 8))
  size.value = { w, h }
}

function onResizeEnd() {
  resizing = false
  window.removeEventListener('pointermove', onResizeMove)
  window.removeEventListener('pointerup', onResizeEnd)
}

onBeforeUnmount(() => {
  onDragEnd()
  onResizeEnd()
})
</script>

<style scoped>
.tfw {
  position: fixed;
  z-index: 10040;
  max-width: 96vw;
  max-height: 92vh;
  display: flex;
  flex-direction: column;
  background: var(--acrylic-bg);
  backdrop-filter: var(--acrylic-blur);
  -webkit-backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  border-radius: 18px;
  box-shadow: var(--shadow-xl);
  overflow: hidden;
}
.tfw.minimized { height: auto !important; }

.tfw-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 10px 10px 6px;
  background: var(--color-surface-high);
  border-bottom: 1px solid var(--color-outline-dim);
  cursor: grab;
  touch-action: none;
  user-select: none;
  flex-shrink: 0;
}
.tfw.minimized .tfw-head { border-bottom: none; }
.tfw-head:active { cursor: grabbing; }
.tfw-grip { color: var(--color-on-surface-variant); font-size: 20px; }
.tfw-title-wrap { display: flex; flex-direction: column; min-width: 0; flex: 1; }
.tfw-eyebrow { font-size: 10.5px; color: var(--color-on-surface-variant); font-weight: 600; }
.tfw-title {
  font-size: 14px;
  font-weight: 650;
  color: var(--color-on-surface);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tfw-icon-btn {
  flex-shrink: 0;
  width: 30px;
  height: 30px;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--color-on-surface-variant);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.tfw-icon-btn:hover { background: var(--color-surface); color: var(--color-on-surface); }
.tfw-icon-btn .material-symbols-outlined { font-size: 18px; }

/* ─── Две колонки ─── */
.tfw-cols {
  display: flex;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
.tfw.narrow .tfw-cols { flex-direction: column; }

.tfw-left {
  width: 44%;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px 16px;
  background: transparent; /* фон даёт акриловый корень окна */
  border-right: 1px solid var(--color-outline-dim);
  overflow-y: auto;
  min-height: 0;
}
.tfw.narrow .tfw-left {
  width: 100%;
  flex: 0 0 auto;
  max-height: 42%;
  border-right: none;
  border-bottom: 1px solid var(--color-outline-dim);
}

.tfw-right {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px 16px;
  background: var(--color-bg);
  overflow: hidden;
}

.tfw-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  align-self: flex-start;
  font-size: 11.5px;
  font-weight: 650;
  padding: 3px 10px;
  border-radius: var(--radius-full, 999px);
}
.tfw-badge .material-symbols-outlined { font-size: 15px; }
.tfw-badge.active { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.tfw-badge.archived { background: var(--color-surface-high); color: var(--color-on-surface-variant); }

/* Поля задачи — как в карточке задачи */
.field-box { display: flex; flex-direction: column; gap: 4px; }
.fields-row { display: flex; gap: 10px; }
.field-box.half { flex: 1; min-width: 0; }
.field-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-primary);
}
.field-value {
  border: 1px solid var(--color-outline-dim);
  border-radius: 8px;
  padding: 7px 10px;
  font-size: 13px;
  color: var(--color-on-surface);
  background: var(--color-surface);
  min-height: 34px;
  display: flex;
  align-items: center;
}
.field-value.with-icon { gap: 6px; }
.responsible-value { gap: 10px; }
.responsible-avatar { width: 24px; height: 24px; border-radius: 50%; object-fit: cover; flex-shrink: 0; }
.responsible-name { font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.text-dim { color: var(--color-on-surface-variant); }
.field-icon { font-size: 15px; color: var(--color-on-surface-variant); flex-shrink: 0; }

/* Табы / контент справа */
.tfw-tabs {
  display: inline-flex;
  background: var(--color-surface-high);
  border-radius: var(--radius-full, 999px);
  padding: 4px;
  align-self: flex-start;
  flex-shrink: 0;
}
.tfw-tab {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  border: none;
  background: none;
  font-size: 12.5px;
  font-weight: 600;
  padding: 6px 14px;
  border-radius: var(--radius-full, 999px);
  cursor: pointer;
  color: var(--color-on-surface-variant);
}
.tfw-tab.active { background: var(--color-primary); color: var(--color-on-primary); }
.tfw-tab .material-symbols-outlined { font-size: 16px; }

.tfw-tab-body { display: flex; flex-direction: column; min-height: 0; flex: 1; }

.tfw-loading, .tfw-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 24px 0;
  color: var(--color-on-surface-variant);
  font-size: 13px;
}
.tfw-fill { flex: 1; justify-content: center; }
.tfw-empty .material-symbols-outlined { font-size: 30px; opacity: 0.55; }

.tfw-units { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; overflow-y: auto; }
.tfw-unit { background: var(--color-surface-high); border-radius: 10px; padding: 8px 12px; }
.tfw-unit-main { display: flex; justify-content: space-between; gap: 8px; align-items: baseline; }
.tfw-unit-name { font-size: 13px; font-weight: 600; color: var(--color-on-surface); }
.tfw-unit-dur { font-size: 12px; font-weight: 600; color: var(--color-primary); flex-shrink: 0; }
.tfw-unit-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 3px;
  font-size: 11.5px;
  color: var(--color-on-surface-variant);
}
.tfw-unit-type {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
  border-radius: var(--radius-full, 999px);
  padding: 1px 8px;
}
.tfw-unit-live { color: var(--color-primary); font-weight: 650; }

/* Уголок ресайза — диагональные риски в правом нижнем углу */
.tfw-resize {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 18px;
  height: 18px;
  cursor: nwse-resize;
  touch-action: none;
  z-index: 1;
  background:
    linear-gradient(135deg, transparent 0 6px, var(--color-on-surface-variant) 6px 8px, transparent 8px 10px,
      var(--color-on-surface-variant) 10px 12px, transparent 12px 100%);
  opacity: 0.5;
  border-bottom-right-radius: 16px;
}
.tfw-resize:hover { opacity: 0.9; }

.spinning { animation: tfwspin 1s linear infinite; }
@keyframes tfwspin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
</style>
