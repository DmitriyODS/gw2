<template>
  <article
    class="task-card"
    :class="[`view-${view}`, { favorite: task.is_favorite, archived: task.is_archived, colored: !!task.color, running: isRunningHere }]"
    :style="cardStyle"
    @click.stop="$emit('click', task)"
  >
    <!-- Цветовая полоса слева (если задан цвет-тег) -->
    <span v-if="task.color" class="color-stripe" aria-hidden="true" />

    <div class="card-main">
      <div class="card-header">
        <span class="dept-badge" :title="task.department?.name || '—'">
          <span class="material-symbols-outlined">apartment</span>
          {{ task.department?.name || '—' }}
        </span>

        <div class="card-actions">
          <button
            ref="colorBtnRef"
            class="card-action-btn"
            :class="{ active: showColors }"
            title="Цвет задачи"
            @click.stop="showColors = !showColors"
          >
            <span class="material-symbols-outlined">palette</span>
          </button>
          <button
            class="card-action-btn favorite-btn"
            :class="{ 'is-fav': task.is_favorite }"
            @click.stop="$emit('toggle-favorite', task)"
            :title="task.is_favorite ? 'Убрать из избранного' : 'Добавить в избранное'"
          >
            <span class="material-symbols-outlined" :class="{ filled: task.is_favorite }">
              {{ task.is_favorite ? 'favorite' : 'favorite_border' }}
            </span>
          </button>
          <TaskColorPopover
            v-model="showColors"
            :anchor="colorBtnRef"
            :value="task.color || null"
            @select="onSelectColor"
          />
        </div>
      </div>

      <h3 class="task-name">{{ task.name }}</h3>

      <div class="task-meta">
        <span
          v-if="task.stage"
          class="meta-chip stage-chip"
          :style="stageChipStyle"
          :title="`Этап: ${task.stage.name}`"
        >
          <span class="stage-dot" :style="stageDotStyle" />
          {{ task.stage.name }}
        </span>
        <span
          v-if="deadlineInfo"
          class="meta-chip"
          :class="`deadline-${deadlineInfo.level}`"
          :title="`Срок: ${formatDate(task.deadline)}`"
        >
          <span class="material-symbols-outlined">{{ deadlineInfo.icon }}</span>
          {{ deadlineInfo.label }}
        </span>
        <span class="meta-chip muted" :title="`Поступила: ${formatDate(task.received_at)}`">
          <span class="material-symbols-outlined">inbox</span>
          {{ formatDate(task.received_at) }}
        </span>
      </div>

      <div class="card-footer">
        <div class="footer-left">
          <button
            v-if="!task.is_archived"
            class="work-btn"
            :class="{ 'is-running': isRunningHere }"
            @click.stop="onWorkClick"
            :title="isRunningHere ? 'Остановить юнит' : 'Начать юнит'"
          >
            <span class="material-symbols-outlined">{{ isRunningHere ? 'stop' : 'play_arrow' }}</span>
            <span class="work-btn-label">{{ isRunningHere ? 'Стоп' : 'В работу' }}</span>
          </button>
          <span v-else-if="task.has_units" class="units-indicator" title="По задаче есть юниты">
            <span class="material-symbols-outlined">timer</span>
          </span>
        </div>

        <span
          v-if="task.responsible"
          class="responsible-ava"
          :title="`Ответственный: ${task.responsible.fio}`"
        >
          <img
            :src="task.responsible.avatar_path ? `/uploads/${task.responsible.avatar_path}` : `/api/users/${task.responsible.id}/identicon`"
            :alt="task.responsible.fio"
          />
        </span>

        <div v-if="task.active_users?.length" class="active-users">
          <span
            v-for="user in task.active_users.slice(0, 4)"
            :key="user.id"
            class="active-avatar"
            :title="user.fio"
          >
            <img
              :src="user.avatar_path ? `/uploads/${user.avatar_path}` : `/api/users/${user.id}/identicon`"
              :alt="user.fio"
            />
          </span>
          <span v-if="task.active_users.length > 4" class="active-avatar active-avatar-more">
            +{{ task.active_users.length - 4 }}
          </span>
        </div>
      </div>
    </div>
  </article>
</template>

<script setup>
import { ref, computed } from 'vue'
import TaskColorPopover from '@/components/tasks/TaskColorPopover.vue'
import { cardColorStyle } from '@/utils/taskColors.js'
import { useUnitsStore } from '@/stores/units.js'

const props = defineProps({
  task: {
    type: Object,
    required: true
  },
  view: {
    type: String,
    default: 'grid' // 'grid' | 'list'
  }
})

const emit = defineEmits(['click', 'toggle-favorite', 'set-color', 'start-unit', 'stop-unit'])

const unitsStore = useUnitsStore()

const showColors = ref(false)
const colorBtnRef = ref(null)

const cardStyle = computed(() => cardColorStyle(props.task.color))

const stageChipStyle = computed(() => {
  const color = props.task.stage?.color
  if (!color) return {}
  return {
    background: `var(--tag-${color}-surface)`,
    color: `var(--tag-${color}-accent)`,
    borderColor: `var(--tag-${color}-border)`,
  }
})

const stageDotStyle = computed(() => {
  const color = props.task.stage?.color
  if (!color) return {}
  return { background: `var(--tag-${color}-accent)` }
})

const isRunningHere = computed(() => unitsStore.activeUnit?.task_id === props.task.id)

// Подсказка по сроку: только для не-архивных задач с дедлайном.
const deadlineInfo = computed(() => {
  if (props.task.is_archived || !props.task.deadline) return null
  const d = new Date(props.task.deadline)
  if (Number.isNaN(d.getTime())) return null
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  d.setHours(0, 0, 0, 0)
  const diff = Math.round((d - today) / 86400000)
  if (diff < 0) return { level: 'overdue', icon: 'warning', label: 'Просрочено' }
  if (diff === 0) return { level: 'soon', icon: 'today', label: 'Сегодня' }
  if (diff <= 2) return { level: 'soon', icon: 'event', label: `Через ${diff} дн.` }
  return { level: 'normal', icon: 'event', label: formatDate(props.task.deadline) }
})

function onWorkClick() {
  if (isRunningHere.value) emit('stop-unit', props.task)
  else emit('start-unit', props.task)
}

function onSelectColor(color) {
  if ((props.task.color || null) === color) return
  emit('set-color', { task: props.task, color })
}

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' })
}
</script>

<style scoped>
.task-card {
  position: relative;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg);
  cursor: pointer;
  overflow: hidden;
  transition: box-shadow 0.18s ease, transform 0.14s ease, border-color 0.18s ease;
}

.task-card:hover {
  box-shadow: var(--shadow-lg);
  transform: translateY(-3px);
  border-color: color-mix(in oklch, var(--color-primary) 35%, var(--color-outline-dim));
}

.task-card:active {
  transform: translateY(-1px);
}

/* Запущенный юнит — мягкая подсветка рамки акцентом */
.task-card.running {
  border-color: color-mix(in oklch, var(--color-secondary) 55%, var(--color-outline-dim));
  box-shadow: 0 0 0 1px color-mix(in oklch, var(--color-secondary) 40%, transparent);
}

/* Окрашенная карточка — пастельный фон выбранного тега */
.task-card.colored {
  background: var(--card-tag-surface);
  border-color: var(--card-tag-border);
}

.task-card.archived {
  background: var(--color-surface-high);
  border-color: var(--color-outline-dim);
  opacity: 0.82;
}

.color-stripe {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 4px;
  background: var(--card-tag-accent, var(--color-primary));
}

.card-main {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 16px;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.dept-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  border-radius: var(--radius-full);
  padding: 3px 10px 3px 8px;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.dept-badge .material-symbols-outlined {
  font-size: 14px;
  flex-shrink: 0;
}

.card-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.card-action-btn {
  background: none;
  border: none;
  cursor: pointer;
  padding: 5px;
  color: var(--color-text-dim);
  border-radius: var(--radius-full);
  display: flex;
  align-items: center;
  transition: color 0.15s, background 0.15s;
}

.card-action-btn:hover,
.card-action-btn.active {
  color: var(--color-primary);
  background: color-mix(in oklch, var(--color-primary) 12%, transparent);
}

.card-action-btn .material-symbols-outlined {
  font-size: 20px;
}

.favorite-btn:hover,
.favorite-btn.is-fav {
  color: var(--color-error);
  background: color-mix(in oklch, var(--color-error) 12%, transparent);
}

.favorite-btn .material-symbols-outlined.filled {
  font-variation-settings: 'FILL' 1;
}

.task-name {
  font-size: 15px;
  font-weight: 650;
  color: var(--color-text);
  margin: 0;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.task-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.meta-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  font-weight: 600;
  padding: 3px 9px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  color: var(--color-text-dim);
  white-space: nowrap;
}

.meta-chip .material-symbols-outlined {
  font-size: 14px;
}

.meta-chip.muted {
  background: transparent;
  padding-left: 2px;
  font-weight: 500;
}

.deadline-overdue {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}

.deadline-soon {
  background: var(--color-warning-container);
  color: var(--color-on-warning-container);
}

.deadline-normal {
  background: var(--color-surface-high);
  color: var(--color-text-dim);
}

.stage-chip {
  border: 1px solid transparent;
}

.stage-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
}

.responsible-ava {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  overflow: hidden;
  flex-shrink: 0;
  box-shadow: 0 0 0 2px var(--color-surface), 0 0 0 3px var(--color-outline-dim);
  margin-right: 6px;
}

.task-card.colored .responsible-ava {
  box-shadow: 0 0 0 2px var(--card-tag-surface), 0 0 0 3px var(--card-tag-border);
}

.responsible-ava img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 2px;
  min-height: 30px;
}

.footer-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.work-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  border: none;
  cursor: pointer;
  border-radius: var(--radius-full);
  padding: 5px 12px 5px 9px;
  font-size: 12px;
  font-weight: 650;
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
  transition: background 0.15s, transform 0.1s;
}

.work-btn:hover {
  background: color-mix(in oklch, var(--color-secondary) 28%, var(--color-secondary-container));
}

.work-btn:active {
  transform: scale(0.96);
}

.work-btn .material-symbols-outlined {
  font-size: 18px;
}

.work-btn.is-running {
  background: var(--color-secondary);
  color: var(--color-on-secondary);
}

.work-btn.is-running:hover {
  background: var(--color-secondary-hover);
}

.units-indicator {
  display: inline-flex;
  align-items: center;
  color: var(--color-secondary);
}

.units-indicator .material-symbols-outlined {
  font-size: 18px;
}

.active-users {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.active-avatar {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  border: 2px solid var(--color-surface);
  overflow: hidden;
  margin-left: -6px;
  flex-shrink: 0;
  box-shadow: 0 0 0 1px var(--color-outline-dim);
}

.task-card.colored .active-avatar {
  border-color: var(--card-tag-surface);
}

.active-avatar:first-child {
  margin-left: 0;
}

.active-avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.active-avatar-more {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 9px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: visible;
}

/* ═══════════════ Режим списка (компактная строка) ═══════════════ */
.task-card.view-list .card-main {
  flex-direction: row;
  align-items: center;
  gap: 14px;
  padding: 12px 16px;
}

.task-card.view-list .card-header {
  order: 2;
  flex-direction: row-reverse;
  width: auto;
  flex-shrink: 0;
}

.task-card.view-list .dept-badge {
  display: none;
}

.task-card.view-list .task-name {
  order: 1;
  flex: 1;
  -webkit-line-clamp: 1;
  font-size: 14px;
}

.task-card.view-list .task-meta {
  order: 1;
  flex-wrap: nowrap;
  flex-shrink: 0;
}

.task-card.view-list .task-meta .meta-chip.muted {
  display: none;
}

.task-card.view-list .card-footer {
  order: 1;
  width: auto;
  margin-top: 0;
  flex-shrink: 0;
  min-height: 0;
}

.task-card.view-list .work-btn-label {
  display: none;
}

.task-card.view-list .work-btn {
  padding: 6px;
}

.task-card.view-list:hover {
  transform: none;
  box-shadow: var(--shadow-md);
}

@media (max-width: 600px) {
  .task-card.view-list .card-main {
    gap: 10px;
    padding: 12px 14px;
  }
  .task-card.view-list .task-meta {
    display: none;
  }
}

/* ═══════════════ Мобильная адаптация карточки ═══════════════ */
@media (max-width: 768px) {
  /* Hover-эффект — лишний на тач-устройствах, на тапе у нас уже :active. */
  .task-card:hover {
    transform: none;
    box-shadow: var(--shadow-sm);
    border-color: var(--color-outline-dim);
  }

  .task-card:active {
    transform: scale(0.985);
    box-shadow: var(--shadow-md);
  }

  .card-main {
    padding: 14px 14px 12px;
    gap: 8px;
  }

  .task-name {
    font-size: 14.5px;
    -webkit-line-clamp: 2;
    line-height: 1.35;
  }

  /* Тач-зоны action-кнопок — минимум 40×40 px. */
  .card-action-btn {
    padding: 8px;
    min-width: 40px;
    min-height: 40px;
    justify-content: center;
  }

  .card-action-btn .material-symbols-outlined {
    font-size: 22px;
  }

  .dept-badge {
    font-size: 11.5px;
    padding: 3px 9px 3px 7px;
  }

  /* Чипы метаданных чуть компактнее — на узких экранах хорошо умещаются в ряд. */
  .meta-chip {
    font-size: 11.5px;
    padding: 3px 8px;
  }

  .meta-chip .material-symbols-outlined {
    font-size: 13px;
  }

  /* «В работу» — крупная тач-зона, без сжатия. */
  .work-btn {
    padding: 8px 14px 8px 10px;
    font-size: 13px;
    min-height: 36px;
  }

  .work-btn .material-symbols-outlined {
    font-size: 20px;
  }

  /* Аватарки чуть крупнее — лучше различимы на маленьких экранах. */
  .responsible-ava {
    width: 28px;
    height: 28px;
    margin-right: 4px;
  }

  .active-avatar {
    width: 26px;
    height: 26px;
  }
}

@media (max-width: 360px) {
  .card-main {
    padding: 12px 12px 10px;
  }

  .dept-badge {
    max-width: 60%;
  }

  /* На самых узких экранах — скрываем подпись «В работу», остаётся круглая
     кнопка-играть. Чтение названия задачи важнее. */
  .work-btn-label {
    display: none;
  }

  .work-btn {
    padding: 8px;
    border-radius: 50%;
    min-width: 36px;
  }
}
</style>
