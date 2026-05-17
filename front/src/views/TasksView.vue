<template>
  <div class="tasks-view">
    <header class="tasks-header">
      <div class="tasks-header-top">
        <!-- Кнопка «Добавить» — только на десктопе -->
        <button
          v-if="canCreateTask"
          data-tutorial="task-add-btn"
          class="btn-primary btn-add-desktop"
          @click="showCreateTask = true"
        >
          <span class="material-symbols-outlined">add</span>
          <span class="btn-label">Добавить</span>
        </button>

        <div class="search-wrapper">
          <span class="material-symbols-outlined search-icon">search</span>
          <input
            v-model="searchQuery"
            class="search-input"
            placeholder="Поиск по названию задачи..."
            @input="onSearch"
          />
        </div>

        <!-- Кнопки только на мобильном -->
        <button class="btn-mobile-icon" @click="showSortSheet = true" title="Сортировка">
          <span class="material-symbols-outlined">sort</span>
        </button>
        <button class="btn-mobile-icon" @click="showMobileFilters = true" title="Фильтры">
          <span class="material-symbols-outlined">tune</span>
        </button>
      </div>

      <div class="tasks-header-tabs">
        <button
          v-for="tab in tabs"
          :key="tab.value"
          :data-tutorial="`tab-${tab.value}`"
          class="tab-btn"
          :class="{ active: tasksStore.filters.tab === tab.value }"
          @click="tasksStore.setTab(tab.value)"
        >
          {{ tab.label }}
        </button>
      </div>
    </header>

    <div class="tasks-body">
      <TaskFilters :mobile-visible="showMobileFilters" @close="showMobileFilters = false" />

      <main ref="cardsAreaRef" class="cards-area" @scroll="onCardsScroll">
        <div v-if="tasksStore.loading" class="loading-state">
          <ProgressSpinner />
        </div>
        <template v-else>
          <div v-if="tasksStore.error" class="empty-state error-state">
            <span class="material-symbols-outlined">error_outline</span>
            <p>{{ tasksStore.error }}</p>
            <button class="btn-retry" @click="tasksStore.fetchTasks()">Повторить</button>
          </div>
          <div v-else-if="tasksStore.tasks.length === 0" class="empty-state">
            <span class="material-symbols-outlined">inbox</span>
            <p>Задач не найдено</p>
          </div>
          <div v-else class="cards-grid">
            <TaskCard
              v-for="task in tasksStore.tasks"
              :key="task.id"
              :task="task"
              @click="openTask(task)"
              @toggle-favorite="toggleFavorite"
            />
          </div>
          <div
            v-if="tasksStore.total > tasksStore.filters.per_page"
            class="pagination"
          >
            <button
              class="page-btn"
              :disabled="tasksStore.filters.page === 1"
              @click="tasksStore.setFilter('page', tasksStore.filters.page - 1)"
            >
              <span class="material-symbols-outlined">chevron_left</span>
            </button>
            <span class="page-info">{{ tasksStore.filters.page }}</span>
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

    <!-- FAB создания задачи — только на мобильном -->
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

    <!-- Шторка сортировки -->
    <SortSheet :visible="showSortSheet" @close="showSortSheet = false" />

    <!-- Модалка просмотра задачи -->
    <TaskModal
      v-if="tasksStore.activeTask"
      :task="tasksStore.activeTask"
      @close="tasksStore.closeTask()"
    />

    <!-- Модалка создания задачи -->
    <TaskForm
      v-if="showCreateTask"
      :task="null"
      @close="showCreateTask = false"
      @saved="onTaskCreated"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useTasksStore } from '@/stores/tasks.js'
import { useUnitsStore } from '@/stores/units.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import { toggleFavorite as apiFavorite } from '@/api/tasks.js'
import TaskCard from '@/components/tasks/TaskCard.vue'
import TaskFilters from '@/components/tasks/TaskFilters.vue'
import TaskModal from '@/components/tasks/TaskModal.vue'
import TaskForm from '@/components/tasks/TaskForm.vue'
import SortSheet from '@/components/tasks/SortSheet.vue'
import ProgressSpinner from 'primevue/progressspinner'

const tasksStore = useTasksStore()
const unitsStore = useUnitsStore()
const notif = useNotificationsStore()
const { isAtLeast } = usePermission()

const showCreateTask = ref(false)
const searchQuery = ref('')
const showMobileFilters = ref(false)
const showSortSheet = ref(false)

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

const tabs = [
  { value: 'active', label: 'Активные' },
  { value: 'favorites', label: 'Избранное' },
  { value: 'archive', label: 'Архив' }
]

let searchTimeout = null

function onSearch() {
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    tasksStore.setFilter('search', searchQuery.value)
  }, 400)
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
    tasksStore.upsertTask({ id: task.id, is_favorite: !task.is_favorite })
  } catch (e) {
    notif.error(e.message || 'Ошибка')
  }
}

function onTaskCreated(task) {
  showCreateTask.value = false
  tasksStore.upsertTask(task)
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
})
</script>

<style scoped>
.tasks-view {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.tasks-header {
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  background: var(--gw-surface);
  border-bottom: 1px solid var(--gw-border);
  flex-shrink: 0;
}

.tasks-header-top {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
}

.tasks-header-tabs {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}

.btn-primary {
  display: flex;
  align-items: center;
  gap: 6px;
  background: var(--gw-primary);
  color: var(--color-on-primary);
  border: none;
  border-radius: 10px;
  padding: 9px 18px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s;
  white-space: nowrap;
  flex-shrink: 0;
}

.btn-primary:hover {
  background: var(--gw-primary-hover);
}

.btn-primary .material-symbols-outlined {
  font-size: 18px;
}

.search-wrapper {
  flex: 1;
  min-width: 200px;
  position: relative;
  display: flex;
  align-items: center;
}

.search-icon {
  position: absolute;
  left: 10px;
  font-size: 18px;
  color: var(--gw-text-secondary);
  pointer-events: none;
}

.search-input {
  width: 100%;
  padding: 8px 12px 8px 36px;
  border: 1px solid var(--gw-border);
  border-radius: 10px;
  font-size: 14px;
  background: var(--gw-bg);
  color: var(--gw-text);
  outline: none;
  transition: border-color 0.15s;
}

.search-input:focus {
  border-color: var(--gw-primary);
  background: var(--gw-surface);
}

.search-input::placeholder {
  color: var(--gw-text-secondary);
}

/* Мобильные иконки-кнопки — скрыты на десктопе */
.btn-mobile-icon {
  display: none;
}

.tab-btn {
  padding: 8px 16px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--gw-text-secondary);
  font-size: 14px;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.tab-btn.active {
  background: var(--gw-primary);
  color: var(--color-on-primary);
  font-weight: 600;
}

.tab-btn:hover:not(.active) {
  background: var(--gw-bg);
  color: var(--gw-text);
}

.tasks-body {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.cards-area {
  flex: 1;
  overflow-y: auto;
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.loading-state {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 60px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 60px;
  color: var(--gw-text-secondary);
  text-align: center;
}

.empty-state .material-symbols-outlined {
  font-size: 48px;
  color: var(--gw-border);
}

.empty-state p {
  margin: 0;
  font-size: 15px;
}

.error-state .material-symbols-outlined {
  color: var(--gw-danger);
}

.btn-retry {
  margin-top: 4px;
  padding: 8px 20px;
  border: 1px solid var(--gw-border);
  border-radius: 8px;
  background: var(--gw-surface);
  color: var(--gw-text);
  font-size: 14px;
  cursor: pointer;
  transition: background 0.15s;
}

.btn-retry:hover {
  background: var(--gw-primary);
  border-color: var(--gw-primary);
  color: var(--color-on-primary);
}

.cards-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 14px;
}

.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 8px 0;
}

.page-btn {
  width: 36px;
  height: 36px;
  border: 1px solid var(--gw-border);
  border-radius: 8px;
  background: var(--gw-surface);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--gw-text);
  transition: background 0.15s;
}

.page-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.page-btn:not(:disabled):hover {
  background: var(--gw-primary);
  border-color: var(--gw-primary);
  color: var(--color-on-primary);
}

.page-btn .material-symbols-outlined {
  font-size: 20px;
}

.page-info {
  min-width: 36px;
  text-align: center;
  font-size: 14px;
  font-weight: 600;
  color: var(--gw-text);
}

@media (max-width: 768px) {
  .cards-area {
    padding-bottom: calc(60px + 80px + env(safe-area-inset-bottom, 0px));
  }

  .tasks-header {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
    padding: 10px 12px;
  }

  /* Десктопная кнопка «Добавить» — скрыта на мобильном (есть FAB) */
  .btn-add-desktop {
    display: none;
  }

  /* Мобильные иконки — видны */
  .btn-mobile-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 40px;
    height: 40px;
    border-radius: 10px;
    border: 1px solid var(--gw-border);
    background: var(--gw-bg);
    color: var(--gw-text);
    cursor: pointer;
    flex-shrink: 0;
    transition: background 0.15s, color 0.15s;
  }

  .btn-mobile-icon:active {
    background: var(--gw-primary-light);
    color: var(--gw-primary);
  }

  .btn-mobile-icon .material-symbols-outlined {
    font-size: 20px;
  }

  .search-wrapper {
    max-width: unset;
  }

  /* На маленьких экранах карточки тоже уменьшаем */
  .cards-grid {
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  }

  /* FAB */
  .fab {
    position: fixed;
    right: 16px;
    bottom: calc(60px + 16px + env(safe-area-inset-bottom, 0px));
    width: 56px;
    height: 56px;
    border-radius: 50%;
    border: none;
    background: var(--gw-primary);
    color: var(--color-on-primary);
    box-shadow: 0 4px 14px color-mix(in oklch, var(--gw-primary) 50%, transparent);
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
    background: var(--gw-primary-hover);
  }

  .fab .material-symbols-outlined {
    font-size: 24px;
  }

  .fab--hidden {
    transform: translateY(calc(100% + 24px));
    opacity: 0;
    pointer-events: none;
  }
}

/* На десктопе FAB не нужен */
@media (min-width: 769px) {
  .fab {
    display: none;
  }
}

@media (max-width: 480px) {
  /* На телефонах — 1 колонка */
  .cards-grid {
    grid-template-columns: 1fr;
  }

  .cards-area {
    padding: 12px;
    padding-bottom: calc(60px + 12px + env(safe-area-inset-bottom, 0px));
  }
}
</style>
