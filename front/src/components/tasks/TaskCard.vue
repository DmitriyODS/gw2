<template>
  <article
    class="task-card glass-hover"
    :class="[`view-${view}`, { favorite: task.is_favorite, archived: task.is_archived, colored: !!task.color, running: isRunningHere }]"
    :style="cardStyle"
    @click.stop="$emit('click', task)"
    @contextmenu.prevent="onContextMenu"
  >
    <div class="card-main">
      <div class="card-top">
        <h3 class="task-name">{{ task.name }}</h3>
        <div class="card-actions">
          <a
            v-if="task.yougile_task_id && task.link_yougile"
            class="card-action-btn"
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

      <!-- Одна строка меты: этап + теги (2 + «+N») + срок. Монохромное стекло,
           цвет — только точками; строка не растёт, чипы усекаются. -->
      <div v-if="task.stage || visibleTags.length || deadlineInfo" class="task-meta">
        <span v-if="task.stage" class="meta-chip" :title="`Этап: ${task.stage.name}`">
          <span class="meta-dot" :style="stageDotStyle" />
          <span class="meta-chip-name">{{ task.stage.name }}</span>
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
        <span
          v-for="tg in visibleTags"
          :key="tg.id"
          class="meta-chip"
          :title="`Тег: ${tg.name}`"
        >
          <span class="meta-dot" :style="tagDotStyle(tg)" />
          <span class="meta-chip-name">{{ tg.name }}</span>
        </span>
        <span v-if="hiddenTags.length" class="meta-chip meta-chip-more" :title="hiddenTagsTitle">
          +{{ hiddenTags.length }}
        </span>
      </div>

      <div v-if="task.department?.name || task.active_users?.length || (task.is_archived && task.has_units)" class="card-footer">
        <span v-if="task.department?.name" class="card-dept" :title="`Отдел: ${task.department.name}`">
          {{ task.department.name }}
        </span>
        <span v-if="task.is_archived && task.has_units" class="units-indicator" title="По задаче есть юниты">
          <span class="material-symbols-outlined">timer</span>
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

const emit = defineEmits(['click', 'toggle-favorite', 'context-menu'])

function onContextMenu(e) {
  emit('context-menu', { x: e.clientX, y: e.clientY, task: props.task })
}

const unitsStore = useUnitsStore()

const cardStyle = computed(() => cardColorStyle(props.task.color))

const stageDotStyle = computed(() => {
  const color = props.task.stage?.color
  if (!color) return { background: 'var(--color-text-dim)' }
  return { background: `var(--tag-${color}-accent)` }
})

// Теги на карточке не раздувают её: видимы первые 2, остальные — счётчиком,
// цвет тега — точкой (монохромные чипы, полный список в тултипе и модалке).
const MAX_VISIBLE_TAGS = 2
const visibleTags = computed(() => (props.task.tags || []).slice(0, MAX_VISIBLE_TAGS))
const hiddenTags = computed(() => (props.task.tags || []).slice(MAX_VISIBLE_TAGS))
const hiddenTagsTitle = computed(() => hiddenTags.value.map((t) => t.name).join(', '))

function tagDotStyle(tag) {
  if (!tag?.color) return { background: 'var(--color-text-dim)' }
  return { background: `var(--tag-${tag.color}-accent)` }
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

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' })
}
</script>

<style scoped>
/* Стеклянная карточка в потоке: монохром, «иней», лёгкий подъём на hover. */
.task-card {
  position: relative;
  background: var(--acrylic-card-bg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  border: 1px solid var(--acrylic-border);
  border-radius: 18px;
  cursor: pointer;
  overflow: hidden;
}
/* Hover — глобальное «запотевание» .glass-hover (main.css). */

/* Запущенный юнит — мягкая подсветка рамки акцентом */
.task-card.running {
  border-color: color-mix(in oklch, var(--color-secondary) 55%, var(--color-outline-dim));
  box-shadow: var(--glass-edge), 0 0 0 1px color-mix(in oklch, var(--color-secondary) 40%, transparent);
}

/* Окрашенная карточка — личный цвет как стеклянный градиент из САМОГО
   цвета тега (общий иней сверху гасил насыщенность): пастельно, но ярко. */
.task-card.colored {
  background: var(--card-tag-surface);
  background: linear-gradient(155deg,
    color-mix(in oklch, var(--card-tag-surface) 92%, transparent),
    color-mix(in oklch, var(--card-tag-surface) 70%, transparent) 50%,
    color-mix(in oklch, var(--card-tag-surface) 85%, transparent));
  border-color: var(--card-tag-border);
}

.task-card.archived {
  background: var(--color-surface-high);
  background: var(--glass-bg);
  border-color: var(--color-outline-dim);
  opacity: 0.82;
}

.card-main {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px 16px;
}

.card-top {
  display: flex;
  align-items: flex-start;
  gap: 6px;
}

.task-name {
  flex: 1;
  min-width: 0;
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

.card-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
  margin-top: -2px;
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
  text-decoration: none;
  transition: color 0.15s, background 0.15s;
}

.card-action-btn:hover {
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

/* Мета: монохромные стеклянные чипы. Чипы НЕ сжимаются и не режутся —
   целиком переносятся; максимум две строки, лишние скрыты (полный
   состав — в тултипах и модалке). 54px = две строки чипов 24px + gap. */
.task-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  overflow: hidden;
  min-width: 0;
  max-height: 54px;
}

.meta-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 24px;
  padding: 0 9px;
  border-radius: var(--radius-full);
  background: var(--color-surface);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text-dim);
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
  flex-shrink: 0;
  max-width: 100%;
}

.meta-chip .material-symbols-outlined {
  font-size: 14px;
  flex-shrink: 0;
}

.meta-chip-name {
  overflow: hidden;
  text-overflow: ellipsis;
}

.meta-chip-more {
  flex-shrink: 0;
  font-weight: 700;
}

.meta-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

/* Срок: спокойный монохром; просрочка — единственный цветовой сигнал. */
.deadline-overdue { color: var(--color-error); }
.deadline-soon { color: var(--color-text); }

.card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 24px;
}

/* Отдел — приглушённая подпись без плашки. */
.card-dept {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
  color: var(--color-text-dim);
}

.units-indicator {
  display: inline-flex;
  align-items: center;
  color: var(--color-text-dim);
}

.units-indicator .material-symbols-outlined {
  font-size: 18px;
}

.active-users {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  margin-left: auto;
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

.task-card.view-list .card-top {
  flex: 1;
  min-width: 0;
  align-items: center;
  order: 1;
}

.task-card.view-list .task-name {
  -webkit-line-clamp: 1;
  font-size: 14px;
}

.task-card.view-list .card-actions {
  order: 3;
  margin-top: 0;
}

.task-card.view-list .task-meta {
  order: 2;
  flex-shrink: 0;
  max-width: 320px;
}

.task-card.view-list .card-footer {
  order: 2;
  margin-top: 0;
  min-height: 0;
  flex-shrink: 0;
}

.task-card.view-list .card-dept {
  display: none;
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
  .task-card:active {
    transform: scale(0.985);
    box-shadow: var(--glass-edge), var(--shadow-md);
  }

  .card-main {
    padding: 12px 14px;
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

  .meta-chip {
    font-size: 11.5px;
    padding: 3px 8px;
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
}
</style>
