<template>
  <article
    class="task-card"
    :class="[`view-${view}`, { favorite: task.is_favorite, archived: task.is_archived, colored: !!task.color, running: isRunningHere }]"
    :style="cardStyle"
    @click.stop="$emit('click', task)"
    @contextmenu.prevent="onContextMenu"
  >
    <!-- Цветовая полоса слева (если задан цвет-тег) -->
    <span v-if="task.color" class="color-stripe" aria-hidden="true" />

    <div class="card-main">
      <div class="card-header">
        <span
          v-if="task.department?.name"
          class="chip-tint chip-tint--primary dept-badge"
          :title="task.department.name"
        >
          <span class="material-symbols-outlined">domain</span>
          <span class="dept-badge-name">{{ task.department.name }}</span>
        </span>

        <div class="card-actions">
          <a
            v-if="task.yougile_task_id && task.link_yougile"
            class="yg-badge"
            :href="task.link_yougile"
            target="_blank"
            rel="noopener"
            @click.stop
            title="Открыть карточку в YouGile"
          >
            <span class="material-symbols-outlined">sync_alt</span>
          </a>
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
        </div>
      </div>

      <h3 class="task-name">{{ task.name }}</h3>

      <div class="task-meta">
        <span
          v-if="task.stage"
          class="chip-tint stage-chip"
          :style="stageChipStyle"
          :title="`Этап: ${task.stage.name}`"
        >
          <span class="stage-dot" :style="stageDotStyle" />
          {{ task.stage.name }}
        </span>
        <span
          v-for="tg in task.tags || []"
          :key="tg.id"
          class="chip-tint stage-chip"
          :style="tagChipStyle(tg)"
          :title="`Тег: ${tg.name}`"
        >
          <span class="material-symbols-outlined tag-chip-icon">sell</span>
          {{ tg.name }}
        </span>
        <span
          v-if="deadlineInfo"
          class="chip-tint"
          :class="deadlineChipClass"
          :title="`Срок: ${formatDate(task.deadline)}`"
        >
          <span class="material-symbols-outlined">{{ deadlineInfo.icon }}</span>
          {{ deadlineInfo.label }}
        </span>
        <span class="chip-tint meta-date" :title="`Поступила: ${formatDate(task.received_at)}`">
          <span class="material-symbols-outlined">calendar_today</span>
          {{ formatDate(task.received_at) }}
        </span>
      </div>

      <div class="card-footer">
        <div class="footer-left">
          <button
            v-if="!task.is_archived"
            class="btn-soft-success work-btn"
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
import { computed } from 'vue'
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

const emit = defineEmits(['click', 'toggle-favorite', 'start-unit', 'stop-unit', 'context-menu'])

function onContextMenu(e) {
  emit('context-menu', { x: e.clientX, y: e.clientY, task: props.task })
}

const unitsStore = useUnitsStore()

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

// Чип тега — та же палитра токенов, что и этап.
function tagChipStyle(tag) {
  if (!tag?.color) return {}
  return {
    background: `var(--tag-${tag.color}-surface)`,
    color: `var(--tag-${tag.color}-accent)`,
    borderColor: `var(--tag-${tag.color}-border)`,
  }
}

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

// Тон чипа срока — через глобальные модификаторы .chip-tint--*.
const deadlineChipClass = computed(() => {
  const level = deadlineInfo.value?.level
  if (level === 'overdue') return 'chip-tint--error'
  if (level === 'soon') return 'chip-tint--warning'
  return ''
})

function onWorkClick() {
  if (isRunningHere.value) emit('stop-unit', props.task)
  else emit('start-unit', props.task)
}

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' })
}
</script>

<style scoped>
/* Стеклянная карточка в потоке: почти без теней, лёгкий подъём на hover. */
.task-card {
  position: relative;
  background: var(--acrylic-card-bg);
  border: 1px solid var(--acrylic-border);
  border-radius: 18px;
  cursor: pointer;
  overflow: hidden;
  transition: box-shadow 0.18s ease, transform 0.14s ease, border-color 0.18s ease;
}

.task-card:hover {
  box-shadow: var(--shadow-sm);
  transform: translateY(-2px);
  border-color: color-mix(in oklch, var(--color-primary) 35%, var(--acrylic-border));
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

/* Чип отдела-заказчика — глобальный .chip-tint--primary, здесь только обрезка. */
.dept-badge {
  min-width: 0;
  padding-left: 8px;
}

.dept-badge .material-symbols-outlined {
  font-size: 14px;
}

.dept-badge-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* margin-left: auto — экшены прижаты вправо и без чипа отдела. */
.card-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
  margin-left: auto;
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

.yg-badge {
  display: inline-flex; align-items: center; justify-content: center;
  width: 26px; height: 26px; border-radius: var(--radius-full);
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  text-decoration: none;
  transition: background 0.15s;
}
.yg-badge:hover { background: color-mix(in oklch, var(--color-tertiary-container) 88%, black); }
.yg-badge .material-symbols-outlined { font-size: 16px; }

.task-name {
  font-size: 15px;
  font-weight: 700;
  color: var(--color-text);
  margin: 0;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.task-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.task-meta .chip-tint .material-symbols-outlined {
  font-size: 14px;
}

/* Дата поступления — приглушённая, без плашки. */
.meta-date {
  background: transparent;
  padding-left: 2px;
  padding-right: 2px;
  font-weight: 500;
}

.stage-chip {
  border: 1px solid transparent;
}

.tag-chip-icon { font-size: 13px; }

.stage-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
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

/* «В работу» — глобальный .btn-soft-success; запущенный юнит остаётся
   в secondary-акценте юнитов (плашка активного юнита, ринг карточки). */
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

.task-card.view-list .task-meta .meta-date {
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
  box-shadow: var(--shadow-sm);
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
    box-shadow: none;
    border-color: var(--acrylic-border);
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
  .task-meta .chip-tint {
    font-size: 11.5px;
    padding: 3px 8px;
  }

  .task-meta .chip-tint .material-symbols-outlined {
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
