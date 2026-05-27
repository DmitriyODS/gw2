<template>
  <div class="tasks-view">
    <header class="tasks-header">
      <!-- Панель инструментов -->
      <div class="tasks-toolbar">
        <div class="search-wrapper">
          <span class="material-symbols-outlined search-icon">search</span>
          <input
            v-model="searchQuery"
            class="search-input"
            placeholder="Поиск по названию задачи…"
            @input="onSearch"
          />
          <button v-if="searchQuery" class="search-clear" @click="clearSearch" title="Очистить">
            <span class="material-symbols-outlined">close</span>
          </button>
        </div>

        <!-- Переключатель вида: сетка / список -->
        <div class="view-toggle" role="group" aria-label="Вид отображения">
          <button
            class="view-toggle-btn"
            :class="{ active: viewMode === 'grid' }"
            @click="setViewMode('grid')"
            title="Карточки"
          >
            <span class="material-symbols-outlined">grid_view</span>
          </button>
          <button
            class="view-toggle-btn"
            :class="{ active: viewMode === 'list' }"
            @click="setViewMode('list')"
            title="Список"
          >
            <span class="material-symbols-outlined">view_list</span>
          </button>
        </div>

        <!-- Мобильные иконки сортировки/фильтров -->
        <button class="btn-icon mobile-only" @click="showSortSheet = true" title="Сортировка">
          <span class="material-symbols-outlined">sort</span>
        </button>
        <button
          class="btn-icon mobile-only"
          :class="{ 'has-dot': hasActiveFilters }"
          @click="showMobileFilters = true"
          title="Фильтры"
        >
          <span class="material-symbols-outlined">tune</span>
        </button>

        <!-- Кнопка «Добавить» — десктоп -->
        <button
          v-if="canCreateTask"
          data-tutorial="task-add-btn"
          class="btn-add desktop-only"
          @click="showCreateTask = true"
        >
          <span class="material-symbols-outlined">add</span>
          <span class="btn-add-label">Добавить</span>
        </button>
      </div>

      <!-- Сегментированные вкладки -->
      <div class="tasks-tabs">
        <button
          v-for="tab in tabs"
          :key="tab.value"
          :data-tutorial="`tab-${tab.value}`"
          class="tab-btn"
          :class="{ active: tasksStore.filters.tab === tab.value }"
          @click="tasksStore.setTab(tab.value)"
        >
          <span class="material-symbols-outlined">{{ tab.icon }}</span>
          <span class="tab-label">{{ tab.label }}</span>
        </button>
      </div>
    </header>

    <div class="tasks-body">
      <TaskFilters :mobile-visible="showMobileFilters" @close="showMobileFilters = false" />

      <main ref="cardsAreaRef" class="cards-area" @scroll="onCardsScroll">
        <div v-if="tasksStore.loading" class="state-block">
          <ProgressSpinner />
        </div>
        <template v-else>
          <div v-if="tasksStore.error" class="state-block empty-state error-state">
            <span class="material-symbols-outlined">error_outline</span>
            <p>{{ tasksStore.error }}</p>
            <button class="btn-retry" @click="tasksStore.fetchTasks()">Повторить</button>
          </div>
          <div v-else-if="tasksStore.tasks.length === 0" class="state-block empty-state">
            <span class="empty-icon material-symbols-outlined">{{ emptyIcon }}</span>
            <p class="empty-title">{{ emptyTitle }}</p>
            <p class="empty-sub">{{ emptySub }}</p>
            <button v-if="canCreateTask && tasksStore.filters.tab === 'active'" class="btn-add" @click="showCreateTask = true">
              <span class="material-symbols-outlined">add</span>
              Создать задачу
            </button>
          </div>
          <div v-else :class="viewMode === 'grid' ? 'cards-grid' : 'cards-list'">
            <TaskCard
              v-for="task in tasksStore.tasks"
              :key="task.id"
              :task="task"
              :view="viewMode"
              @click="openTask(task)"
              @toggle-favorite="toggleFavorite"
              @set-color="setColor"
              @start-unit="onStartUnit"
              @stop-unit="onStopUnit"
            />
          </div>

          <div v-if="tasksStore.total > tasksStore.filters.per_page" class="pagination">
            <button
              class="page-btn"
              :disabled="tasksStore.filters.page === 1"
              @click="tasksStore.setFilter('page', tasksStore.filters.page - 1)"
            >
              <span class="material-symbols-outlined">chevron_left</span>
            </button>
            <span class="page-info">{{ tasksStore.filters.page }} / {{ totalPages }}</span>
            <button
              class="page-btn"
              :disabled="tasksStore.tasks.length < tasksStore.filters.per_page"
              @click="tasksStore.setFilter('page', tasksStore.filters.page + 1)"
            >
              <span class="material-symbols-outlined">chevron_right</span>
            </button>
          </div>
        </template>
      </main>
    </div>

    <!-- FAB создания — мобильный -->
    <Teleport to="body">
      <button
        v-if="canCreateTask"
        class="fab"
        :class="{ 'fab--hidden': !fabVisible }"
        @click="showCreateTask = true"
        aria-label="Добавить задачу"
      >
        <span class="material-symbols-outlined">add</span>
      </button>
    </Teleport>

    <SortSheet :visible="showSortSheet" @close="showSortSheet = false" />

    <TaskModal
      v-if="tasksStore.activeTask"
      :task="tasksStore.activeTask"
      @close="tasksStore.closeTask()"
    />

    <TaskForm
      v-if="showCreateTask"
      :task="null"
      @close="showCreateTask = false"
      @saved="onTaskCreated"
    />

    <!-- Быстрый старт юнита прямо с карточки -->
    <StartUnitModal
      v-if="startUnitTaskId != null"
      :task-id="startUnitTaskId"
      @close="startUnitTaskId = null"
      @started="startUnitTaskId = null"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTasksStore } from '@/stores/tasks.js'
import { useUnitsStore } from '@/stores/units.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import { toggleFavorite as apiFavorite, setTaskColor } from '@/api/tasks.js'
import TaskCard from '@/components/tasks/TaskCard.vue'
import TaskFilters from '@/components/tasks/TaskFilters.vue'
import TaskModal from '@/components/tasks/TaskModal.vue'
import TaskForm from '@/components/tasks/TaskForm.vue'
import SortSheet from '@/components/tasks/SortSheet.vue'
import StartUnitModal from '@/components/units/StartUnitModal.vue'
import ProgressSpinner from 'primevue/progressspinner'

const VIEW_KEY = 'gw2_tasks_view'

const route = useRoute()
const router = useRouter()
const tasksStore = useTasksStore()
const unitsStore = useUnitsStore()
const notif = useNotificationsStore()
const { isAtLeast } = usePermission()

const showCreateTask = ref(false)
const searchQuery = ref(tasksStore.filters.search)
const showMobileFilters = ref(false)
const showSortSheet = ref(false)
const startUnitTaskId = ref(null)

const viewMode = ref(localStorage.getItem(VIEW_KEY) === 'list' ? 'list' : 'grid')
function setViewMode(mode) {
  viewMode.value = mode
  try { localStorage.setItem(VIEW_KEY, mode) } catch {}
}

const cardsAreaRef = ref(null)
const fabVisible = ref(true)
let lastScrollTop = 0

function onCardsScroll() {
  const el = cardsAreaRef.value
  if (!el) return
  const st = el.scrollTop
  fabVisible.value = st < lastScrollTop || st < 60
  lastScrollTop = st
}

const canCreateTask = computed(() => isAtLeast(ROLES.EMPLOYEE))
const totalPages = computed(() => Math.ceil(tasksStore.total / tasksStore.filters.per_page))

const hasActiveFilters = computed(() => {
  const f = tasksStore.filters
  return f.sort !== 'last_activity'
    || f.dept_id != null
    || f.has_units != null
    || f.period_preset != null
    || f.received_from
    || f.received_to
})

const tabs = [
  { value: 'active', label: 'Активные', icon: 'checklist' },
  { value: 'favorites', label: 'Избранное', icon: 'star' },
  { value: 'archive', label: 'Архив', icon: 'inventory_2' }
]

const emptyMeta = {
  active: { icon: 'task_alt', title: 'Активных задач нет', sub: 'Создайте новую задачу или измените фильтры.' },
  favorites: { icon: 'star', title: 'В избранном пусто', sub: 'Отметьте задачу звёздочкой, чтобы она появилась здесь.' },
  archive: { icon: 'inventory_2', title: 'Архив пуст', sub: 'Завершённые задачи будут храниться здесь.' }
}
const emptyIcon = computed(() => emptyMeta[tasksStore.filters.tab]?.icon ?? 'inbox')
const emptyTitle = computed(() => (searchQuery.value ? 'Ничего не найдено' : emptyMeta[tasksStore.filters.tab]?.title ?? 'Задач не найдено'))
const emptySub = computed(() => (searchQuery.value ? 'Попробуйте изменить запрос или сбросить фильтры.' : emptyMeta[tasksStore.filters.tab]?.sub ?? ''))

let searchTimeout = null

function onSearch() {
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    tasksStore.setFilter('search', searchQuery.value)
  }, 400)
}

function clearSearch() {
  searchQuery.value = ''
  tasksStore.setFilter('search', '')
}

async function openTask(task) {
  try {
    const { getTask } = await import('@/api/tasks.js')
    const full = await getTask(task.id)
    tasksStore.openTask(full)
  } catch {
    tasksStore.openTask(task)
  }
}

async function toggleFavorite(task) {
  try {
    await apiFavorite(task.id)
    tasksStore.setFavorite(task.id, !task.is_favorite)
  } catch (e) {
    notif.error(e.message || 'Ошибка')
  }
}

async function setColor({ task, color }) {
  const prev = task.color ?? null
  tasksStore.patchTask({ id: task.id, color })
  try {
    await setTaskColor(task.id, color)
  } catch (e) {
    tasksStore.patchTask({ id: task.id, color: prev })
    notif.error(e.message || 'Не удалось изменить цвет')
  }
}

function onStartUnit(task) {
  startUnitTaskId.value = task.id
}

async function onStopUnit() {
  try {
    await unitsStore.stop()
    notif.success('Юнит остановлен')
  } catch (e) {
    notif.error(e.message || 'Не удалось остановить юнит')
  }
}

function onTaskCreated(task) {
  showCreateTask.value = false
  tasksStore.upsertTask(task)
  tasksStore.fetchTasks({ silent: true }).catch(() => {})
}

onMounted(async () => {
  try {
    await tasksStore.fetchTasks()
  } catch (e) {
    notif.error(e.message || 'Не удалось загрузить задачи')
  }
  try {
    await unitsStore.fetchActiveUnit()
  } catch {}
  const openId = route.query.open
  if (openId) {
    openTask({ id: Number(openId) })
    router.replace({ path: '/tasks' })
  }
})
</script>

<style scoped>
.tasks-view {
  display: flex;
  flex-direction: column;
  height: 100%;
}

/* ─── Шапка ─── */
.tasks-header {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px 24px;
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-outline-dim);
  flex-shrink: 0;
}

.tasks-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
}

.search-wrapper {
  flex: 1;
  min-width: 0;
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 12px;
  font-size: 20px;
  color: var(--color-text-dim);
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 10px 38px 10px 40px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  font-size: 14px;
  background: var(--color-surface-low);
  color: var(--color-text);
  outline: none;
  transition: border-color 0.15s, background 0.15s, box-shadow 0.15s;
}

.search-input:focus {
  border-color: var(--color-primary);
  background: var(--color-surface);
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--color-primary) 16%, transparent);
}

.search-input::placeholder {
  color: var(--color-text-dim);
}

.search-clear {
  position: absolute;
  right: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.search-clear:hover {
  background: var(--color-surface-highest);
  color: var(--color-text);
}

.search-clear .material-symbols-outlined {
  font-size: 18px;
}

/* Переключатель вида */
.view-toggle {
  display: flex;
  background: var(--color-surface-high);
  border-radius: var(--radius-full);
  padding: 3px;
  gap: 2px;
  flex-shrink: 0;
}

.view-toggle-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 32px;
  border: none;
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  transition: background 0.18s, color 0.18s;
}

.view-toggle-btn:hover {
  color: var(--color-text);
}

.view-toggle-btn.active {
  background: var(--color-surface);
  color: var(--color-primary);
  box-shadow: var(--shadow-sm);
}

.view-toggle-btn .material-symbols-outlined {
  font-size: 20px;
}

/* Кнопки-иконки (мобильные) */
.btn-icon {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-outline-dim);
  background: var(--color-surface-low);
  color: var(--color-text);
  cursor: pointer;
  flex-shrink: 0;
  transition: background 0.15s, color 0.15s;
}

.btn-icon:active {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.btn-icon .material-symbols-outlined {
  font-size: 22px;
}

.btn-icon.has-dot::after {
  content: '';
  position: absolute;
  top: 8px;
  right: 8px;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-primary);
  border: 2px solid var(--color-surface);
}

/* Кнопка «Добавить» */
.btn-add {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--color-primary);
  color: var(--color-on-primary);
  border: none;
  border-radius: var(--radius-full);
  padding: 10px 20px;
  font-size: 14px;
  font-weight: 650;
  cursor: pointer;
  transition: background 0.15s, box-shadow 0.15s;
  white-space: nowrap;
  flex-shrink: 0;
  box-shadow: var(--shadow-sm);
}

.btn-add:hover {
  background: var(--color-primary-hover);
  box-shadow: var(--shadow-md);
}

.btn-add .material-symbols-outlined {
  font-size: 20px;
}

/* Сегментированные вкладки */
.tasks-tabs {
  display: inline-flex;
  align-self: flex-start;
  gap: 2px;
  background: var(--color-surface-high);
  border-radius: var(--radius-full);
  padding: 4px;
}

.tab-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 18px;
  border: none;
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-text-dim);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.18s, color 0.18s;
}

.tab-btn .material-symbols-outlined {
  font-size: 18px;
}

.tab-btn:hover:not(.active) {
  color: var(--color-text);
}

.tab-btn.active {
  background: var(--color-surface);
  color: var(--color-primary);
  box-shadow: var(--shadow-sm);
}

/* ─── Тело ─── */
.tasks-body {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.cards-area {
  flex: 1;
  overflow-y: auto;
  padding: 22px 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.cards-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(248px, 1fr));
  gap: 16px;
}

.cards-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

/* Состояния */
.state-block {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 48px;
}

.empty-state {
  flex-direction: column;
  gap: 8px;
  color: var(--color-text-dim);
  text-align: center;
  margin: auto;
}

.empty-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 88px;
  height: 88px;
  border-radius: 50%;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 44px;
  margin-bottom: 8px;
}

.empty-title {
  margin: 0;
  font-size: 17px;
  font-weight: 650;
  color: var(--color-text);
}

.empty-sub {
  margin: 0 0 8px;
  font-size: 14px;
  max-width: 320px;
}

.error-state .empty-icon,
.error-state .material-symbols-outlined {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
  font-size: 48px;
}

.btn-retry {
  margin-top: 4px;
  padding: 9px 22px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: var(--color-surface);
  color: var(--color-text);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-retry:hover {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: var(--color-on-primary);
}

/* Пагинация */
.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 8px 0 4px;
}

.page-btn {
  width: 40px;
  height: 40px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: var(--color-surface);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text);
  transition: background 0.15s, border-color 0.15s;
}

.page-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.page-btn:not(:disabled):hover {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: var(--color-on-primary);
}

.page-btn .material-symbols-outlined {
  font-size: 22px;
}

.page-info {
  min-width: 48px;
  text-align: center;
  font-size: 14px;
  font-weight: 650;
  color: var(--color-text);
}

/* Видимость по платформе */
.mobile-only {
  display: none;
}

/* ─── Мобильная адаптивность ─── */
@media (max-width: 768px) {
  .tasks-header {
    padding: 10px 12px;
    gap: 10px;
  }

  .desktop-only {
    display: none;
  }

  .mobile-only {
    display: flex;
  }

  .cards-grid {
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: 12px;
  }

  .cards-area {
    padding: 14px 12px;
    padding-bottom: calc(60px + 80px + env(safe-area-inset-bottom, 0px));
  }

  .tab-btn {
    flex: 1;
    justify-content: center;
    padding: 8px 10px;
  }

  .tasks-tabs {
    align-self: stretch;
    display: flex;
  }

  .tab-label {
    font-size: 13px;
  }

  .fab {
    position: fixed;
    right: 16px;
    bottom: calc(60px + 16px + env(safe-area-inset-bottom, 0px));
    width: 56px;
    height: 56px;
    border-radius: 50%;
    border: none;
    background: var(--color-primary);
    color: var(--color-on-primary);
    box-shadow: 0 4px 16px color-mix(in oklch, var(--color-primary) 50%, transparent);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 150;
    transition: transform 0.28s cubic-bezier(0.4, 0, 0.2, 1),
                opacity 0.28s cubic-bezier(0.4, 0, 0.2, 1),
                background 0.15s;
  }

  .fab:active {
    background: var(--color-primary-hover);
  }

  .fab .material-symbols-outlined {
    font-size: 26px;
  }

  .fab--hidden {
    transform: translateY(calc(100% + 24px));
    opacity: 0;
    pointer-events: none;
  }
}

@media (min-width: 769px) {
  .fab {
    display: none;
  }
}

@media (max-width: 480px) {
  .cards-grid {
    grid-template-columns: 1fr;
  }

  .tab-label {
    display: none;
  }

  .tab-btn {
    padding: 9px;
  }
}
</style>
