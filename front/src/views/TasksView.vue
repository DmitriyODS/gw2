<template>
  <div class="tasks-view">
    <header class="tasks-header" :class="{ 'is-compact': isCompact }">
      <!-- Панель инструментов -->
      <div class="tasks-toolbar">
        <SearchField
          :model-value="searchQuery"
          placeholder="Поиск по названию задачи…"
          hotkey
          @update:model-value="onSearchInput"
        />

        <!-- Переключатель вида: сетка / список / канбан (только десктоп) -->
        <div class="desktop-only" role="group" aria-label="Вид отображения">
          <SegmentedTabs
            variant="glass"
            :model-value="viewMode"
            :tabs="viewTabs"
            @update:model-value="setViewMode"
          />
        </div>

        <!-- Мобильные иконки сортировки/фильтров -->
        <button class="btn-icon mobile-only" @click="showSortSheet = true" title="Сортировка" aria-label="Сортировка">
          <span class="material-symbols-outlined">sort</span>
        </button>
        <button
          class="btn-icon mobile-only"
          :class="{ 'has-dot': hasActiveFilters }"
          @click="showMobileFilters = true"
          title="Фильтры"
          aria-label="Фильтры"
        >
          <span class="material-symbols-outlined">tune</span>
        </button>

        <!-- Кнопка «Из YouGile» — десктоп, видна только если интеграция доступна -->
        <button
          v-if="canCreateTask && yougileAvailable"
          class="btn-glass desktop-only"
          @click="showImportYg = true"
          title="Импортировать карточку из YouGile"
        >
          <span class="material-symbols-outlined">sync_alt</span>
          Из YouGile
        </button>

        <!-- Кнопка «Добавить» — десктоп -->
        <button
          v-if="canCreateTask"
          data-tutorial="task-add-btn"
          class="btn-grad desktop-only"
          @click="showCreateTask = true"
        >
          <span class="material-symbols-outlined">add</span>
          Добавить
        </button>
      </div>

      <!-- Вкладки — только мобильный: на десктопе ими управляет рейка фильтров -->
      <SegmentedTabs
        v-if="isMobile"
        :model-value="tasksStore.filters.tab"
        :tabs="tabs"
        :full-width="isMobile"
        :dense="isCompact"
        @update:model-value="tasksStore.setTab($event)"
      />
    </header>

    <div class="tasks-body">
      <!-- Рут-админ без выбранной компании -->
      <EmptyState
        v-if="auth.isSuperAdmin && !companiesStore.effectiveCompanyId"
        class="tasks-empty"
        icon="domain"
        title="Выберите компанию"
        subtitle="Задачи ведутся в рамках компании. Выберите её в боковом меню."
      />

      <template v-else>
      <TaskFilters :mobile-visible="showMobileFilters" @close="showMobileFilters = false" />

      <main
        ref="cardsAreaRef"
        class="cards-area"
        :class="{ 'cards-area--board': viewMode === 'board' }"
      >
        <div v-if="tasksStore.loading" class="state-block">
          <ProgressSpinner />
        </div>
        <template v-else>
          <EmptyState
            v-if="tasksStore.error"
            class="tasks-empty"
            icon="error_outline"
            tone="error"
            :subtitle="tasksStore.error"
          >
            <button class="btn-retry" @click="tasksStore.fetchTasks()">Повторить</button>
          </EmptyState>
          <EmptyState
            v-else-if="tasksStore.tasks.length === 0"
            class="tasks-empty"
            :icon="emptyIcon"
            :title="emptyTitle"
            :subtitle="emptySub"
          >
            <button v-if="canCreateTask && tasksStore.filters.tab === 'active'" class="btn-grad" @click="showCreateTask = true">
              <span class="material-symbols-outlined">add</span>
              Создать задачу
            </button>
          </EmptyState>
          <TaskKanban
            v-else-if="viewMode === 'board'"
            @open-task="openTask"
            @toggle-favorite="toggleFavorite"
            @start-unit="onStartUnit"
            @stop-unit="onStopUnit"
            @context-menu="openTaskContextMenu"
          />
          <div v-else :class="viewMode === 'grid' ? 'cards-grid' : 'cards-list'">
            <TaskCard
              v-for="task in tasksStore.tasks"
              :key="task.id"
              v-memo="[task, viewMode, unitsStore.activeUnit?.id]"
              :task="task"
              :view="viewMode"
              @click="openTask(task)"
              @toggle-favorite="toggleFavorite"
              @start-unit="onStartUnit"
              @stop-unit="onStopUnit"
              @context-menu="openTaskContextMenu"
            />
          </div>

          <div v-if="viewMode !== 'board' && tasksStore.total > tasksStore.filters.per_page" class="pagination">
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
      </template>
    </div>

    <AppFab
      :visible="canCreateTask"
      icon="add"
      label="Создать"
      :collapsed="isCompact"
      aria-label="Создать задачу"
      @click="showCreateTask = true"
    />

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

    <ImportFromYougileDialog
      :visible="showImportYg"
      @close="showImportYg = false"
      @imported="onYgImported"
    />

    <!-- Быстрый старт юнита прямо с карточки -->
    <StartUnitModal
      v-if="startUnitTaskId != null"
      :task-id="startUnitTaskId"
      @close="startUnitTaskId = null"
      @started="startUnitTaskId = null"
    />

    <!-- Редактирование задачи из контекстного меню -->
    <TaskForm
      v-if="editingTask"
      :task="editingTask"
      @close="editingTask = null"
      @saved="onTaskEditedFromCtx"
    />

    <!-- Контекстное меню по ПКМ на карточке задачи -->
    <TaskContextMenu
      :visible="taskCtxMenu.visible"
      :x="taskCtxMenu.x"
      :y="taskCtxMenu.y"
      :can-edit="taskCtxCanEdit"
      :is-archived="!!taskCtxMenu.task?.is_archived"
      :is-running="taskCtxIsRunning"
      :color="taskCtxMenu.task?.color || ''"
      :tags="tasksStore.tags"
      :task-tag-ids="taskCtxTagIds"
      @close="taskCtxMenu.visible = false"
      @action="onTaskCtxAction"
      @color="onTaskCtxColor"
      @toggle-tag="onTaskCtxToggleTag"
    />

    <!-- Диалог отправки задачи в чат -->
    <SendTaskDialog
      ref="sendTaskDialogRef"
      v-model="sendTaskOpen"
      :task="sendTaskSource"
      @confirm="onSendTaskConfirm"
    />

    <!-- Подтверждение архивации из контекстного меню -->
    <ConfirmDialog
      :visible="archiveConfirm.visible"
      header="Завершить задачу"
      :message="archiveConfirm.message"
      confirm-label="Завершить"
      @confirm="doArchiveTask"
      @cancel="archiveConfirm.visible = false"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTasksStore } from '@/stores/tasks.js'
import { useUnitsStore } from '@/stores/units.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import { useAuthStore } from '@/stores/auth.js'
import { useCompaniesStore } from '@/stores/companies.js'
import { toggleFavorite as apiFavorite, setTaskColor, archiveTask as apiArchiveTask, getTask } from '@/api/tasks.js'
import { useMessengerStore } from '@/stores/messenger.js'
import TaskCard from '@/components/tasks/TaskCard.vue'
import TaskFilters from '@/components/tasks/TaskFilters.vue'
import TaskModal from '@/components/tasks/TaskModal.vue'
import TaskForm from '@/components/tasks/TaskForm.vue'
import ImportFromYougileDialog from '@/components/tasks/ImportFromYougileDialog.vue'
import { useYougileStore } from '@/stores/yougile.js'
import TaskKanban from '@/components/tasks/TaskKanban.vue'
import SortSheet from '@/components/tasks/SortSheet.vue'
import StartUnitModal from '@/components/units/StartUnitModal.vue'
import TaskContextMenu from '@/components/tasks/TaskContextMenu.vue'
import SendTaskDialog from '@/components/tasks/SendTaskDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import AppFab from '@/components/common/AppFab.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SegmentedTabs from '@/components/common/SegmentedTabs.vue'
import SearchField from '@/components/common/SearchField.vue'
import ProgressSpinner from 'primevue/progressspinner'
import { useCompanySettings } from '@/composables/useCompanySettings.js'
import { useScrollCollapse } from '@/composables/useScrollCollapse.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { storageGet, storageSet } from '@/utils/storage.js'

const VIEW_KEY = 'gw2_tasks_view'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const companiesStore = useCompaniesStore()
const tasksStore = useTasksStore()
const unitsStore = useUnitsStore()
const notif = useNotificationsStore()
const { isAtLeast } = usePermission()

const showCreateTask = ref(false)
const showImportYg = ref(false)

const yougileStore = useYougileStore()
const yougileAvailable = computed(() => yougileStore.isAvailable)
const searchQuery = ref(tasksStore.filters.search)
const showMobileFilters = ref(false)
const showSortSheet = ref(false)
const startUnitTaskId = ref(null)

const { usesStages } = useCompanySettings()

// Канбан доступен только если у компании включены этапы и мы не в архиве.
const canShowKanban = computed(() =>
  usesStages.value && tasksStore.filters.tab !== 'archive'
)

const _saved = storageGet(VIEW_KEY, '')
const viewMode = ref(_saved === 'list' || _saved === 'board' ? _saved : 'grid')

function setViewMode(mode) {
  viewMode.value = mode
  storageSet(VIEW_KEY, mode)
}

// Если перешли в архив, а активный режим — канбан, переключаемся на сетку.
watch(canShowKanban, (v) => {
  if (!v && viewMode.value === 'board') viewMode.value = 'grid'
})

// Табы переключателя вида (icon-only, glass-вариант SegmentedTabs).
const viewTabs = computed(() => [
  { value: 'grid', icon: 'grid_view' },
  { value: 'list', icon: 'view_list' },
  ...(canShowKanban.value ? [{ value: 'board', icon: 'view_kanban' }] : []),
])

// Канбан показывает все задачи сразу (без пагинации) — каждая колонка
// прокручивается отдельно. В сетке/списке возвращаем стандартный шаг 30,
// чтобы не грузить лишнее. immediate: true — синхронизировать состояние
// фильтра при первичном монтировании с восстановленным viewMode.
const PER_PAGE_GRID = 30
const PER_PAGE_BOARD = 1000
watch(viewMode, (m) => {
  const target = m === 'board' ? PER_PAGE_BOARD : PER_PAGE_GRID
  if (tasksStore.filters.per_page !== target) {
    tasksStore.setFilter('per_page', target)
  }
}, { immediate: true })

const cardsAreaRef = ref(null)
const { isCompact } = useScrollCollapse(cardsAreaRef)
const { isMobile } = useBreakpoint()

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
    || f.created_by_me
})

const tabs = [
  { value: 'active', label: 'Активные', icon: 'checklist', tutorial: 'tab-active' },
  { value: 'favorites', label: 'Избранное', icon: 'star', tutorial: 'tab-favorites' },
  { value: 'archive', label: 'Архив', icon: 'inventory_2', tutorial: 'tab-archive' },
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
let initialFetchDone = false

function onSearchInput(value) {
  searchQuery.value = value
  clearTimeout(searchTimeout)
  // Очистку применяем сразу (крестик/Esc), набор текста — с дебаунсом.
  if (!value) {
    tasksStore.setFilter('search', '')
    return
  }
  searchTimeout = setTimeout(() => {
    tasksStore.setFilter('search', searchQuery.value)
  }, 400)
}

async function openTask(task) {
  try {
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

/* ── Контекстное меню по ПКМ на карточке задачи ─────────────────── */
const taskCtxMenu = ref({ visible: false, x: 0, y: 0, task: null })
const editingTask = ref(null)
const sendTaskOpen = ref(false)
const sendTaskSource = ref(null)
const sendTaskDialogRef = ref(null)
const archiveConfirm = ref({ visible: false, taskId: null, message: '' })
const messengerStore = useMessengerStore()

const taskCtxCanEdit = computed(() => {
  const t = taskCtxMenu.value.task
  if (!t) return false
  // Минимальная проверка прав. Серверная всё равно решающая, но в меню
  // незачем светить «Изменить» и «В архив» рядовому сотруднику без прав.
  if (auth.user?.id === t.responsible?.id || auth.user?.id === t.responsible_user_id) return true
  return isAtLeast(ROLES.MANAGER)
})

const taskCtxIsRunning = computed(() => {
  const t = taskCtxMenu.value.task
  return !!t && unitsStore.activeUnit?.task_id === t.id
})

function openTaskContextMenu({ x, y, task }) {
  taskCtxMenu.value = { visible: true, x, y, task }
  tasksStore.fetchTags() // лениво: справочник для секции «Теги»
}

// Отмеченные теги — из АКТУАЛЬНОЙ задачи стора (patchTask обновляет её после
// каждого toggle, снапшот в taskCtxMenu устаревает).
const taskCtxTagIds = computed(() => {
  const id = taskCtxMenu.value.task?.id
  const task = id != null ? (tasksStore.taskById.get(id) || taskCtxMenu.value.task) : null
  return (task?.tags || []).map((t) => t.id)
})

function onTaskCtxToggleTag(tagId) {
  const task = taskCtxMenu.value.task
  if (!task) return
  tasksStore.toggleTaskTag(task.id, tagId).catch((e) => {
    notif.error(e?.message || 'Не удалось изменить теги')
  })
}

function onTaskCtxColor(colorId) {
  const task = taskCtxMenu.value.task
  if (!task) return
  // API снимает цвет по null; '' приходит от кнопки «Без цвета» в палитре.
  setColor({ task, color: colorId || null })
}

function onTaskCtxAction(action) {
  const task = taskCtxMenu.value.task
  if (!task) return
  if (action === 'open') openTask(task)
  else if (action === 'edit') startEditTask(task)
  else if (action === 'start-unit') onStartUnit(task)
  else if (action === 'stop-unit') onStopUnit()
  else if (action === 'send') startSendTask(task)
  else if (action === 'archive') askArchiveTask(task)
}

async function startEditTask(task) {
  // TaskForm ожидает полный объект — подтянем свежий, чтобы поля (описание,
  // вложения, ответственный) точно были.
  try {
    editingTask.value = await getTask(task.id)
  } catch {
    editingTask.value = task
  }
}

function onTaskEditedFromCtx(task) {
  editingTask.value = null
  tasksStore.upsertTask(task)
}

function startSendTask(task) {
  sendTaskSource.value = task
  sendTaskOpen.value = true
}

async function onSendTaskConfirm({ user, text }) {
  try {
    const convId = await messengerStore.openWith(user.id)
    await messengerStore.send(convId, {
      text: text || null,
      attachment_ids: [],
      reply_to_id: null,
      task_id: sendTaskSource.value?.id || null,
    })
    notif.success(`Задача отправлена: ${user.fio}`)
    sendTaskOpen.value = false
    sendTaskSource.value = null
  } catch (e) {
    notif.error(e?.message || 'Не удалось отправить задачу')
  } finally {
    sendTaskDialogRef.value?.stopSending()
  }
}

function askArchiveTask(task) {
  archiveConfirm.value = {
    visible: true,
    taskId: task.id,
    message: `Завершить задачу "${task.name}"? Задача будет перемещена в архив.`,
  }
}

async function doArchiveTask() {
  const id = archiveConfirm.value.taskId
  archiveConfirm.value.visible = false
  if (id == null) return
  try {
    const result = await apiArchiveTask(id)
    tasksStore.archiveTask(id, result?.archived_at)
    notif.success('Задача завершена и перемещена в архив')
  } catch (e) {
    if (e?.status === 409) {
      notif.error('Нельзя архивировать задачу с активным юнитом')
    } else {
      notif.error(e?.message || 'Не удалось завершить задачу')
    }
  }
}

function onTaskCreated(task) {
  showCreateTask.value = false
  tasksStore.upsertTask(task)
  tasksStore.fetchTasks({ silent: true }).catch(() => {})
  openTask(task)
}

function onYgImported(task) {
  showImportYg.value = false
  tasksStore.upsertTask(task)
  tasksStore.fetchTasks({ silent: true }).catch(() => {})
  openTask(task)
}

function consumeOpenQuery() {
  // Источника два: canonical `/tasks/:id` (params.id) и legacy `/tasks?open=…`.
  // Второй вариант — для утреннего брифинга Грувика/уведомлений/совместимости.
  const openId = route.params.id || route.query.open
  if (!openId) return
  openTask({ id: Number(openId) })
  // Сворачиваем URL обратно к /tasks, чтобы повторный клик на ту же задачу
  // (или history.back) снова открыл модалку.
  router.replace({ path: '/tasks' })
}

onMounted(() => {
  initialFetchDone = true
  // Первичная загрузка задач. Watch на viewMode с immediate:true дёргает
  // setFilter только если per_page реально меняется (для board-режима);
  // при дефолтном grid условие ложно — поэтому fetch нужен здесь явно.
  tasksStore.fetchTasks().catch(() => {})
  // Карточку из canonical-ссылки /tasks/:id открываем немедленно — не ждём
  // активный юнит и статус YouGile, иначе по deep-link карточка появляется
  // с лишней задержкой (на медленной сети — спустя десятки секунд).
  consumeOpenQuery()
  unitsStore.fetchActiveUnit().catch(() => {})
  // Статус YouGile подгружаем фоном — нужен только для показа/скрытия кнопок.
  yougileStore.refreshStatus().catch(() => {})
})

/* Если пользователь уже на /tasks и кликнул задачу в утреннем брифинге,
   роутер делает push с тем же path и другим query — компонент НЕ пересоздаётся,
   onMounted не повторяется. Поэтому слушаем сам query.open и реагируем здесь. */
watch(() => route.query.open, (v) => {
  if (v) consumeOpenQuery()
})
// То же и для canonical-маршрута: если перейти с `/tasks/5` на `/tasks/8`
// уже находясь на `/tasks/:id`, компонент не пересоздаётся.
watch(() => route.params.id, (v) => {
  if (v) consumeOpenQuery()
})

// Рут-админ переключил компанию — перезагружаем задачи.
watch(() => companiesStore.effectiveCompanyId, () => {
  if (!initialFetchDone) return
  tasksStore.fetchTasks().catch(() => {})
})
</script>

<style scoped>
.tasks-view {
  display: flex;
  flex-direction: column;
  height: 100%;
}

/* ─── Шапка ───
   На десктопе — прозрачный тулбар поверх «сияния» страницы: поле поиска,
   переключатель вида и кнопки — самостоятельные стеклянные элементы. */
.tasks-header {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 20px 24px 4px;
  flex-shrink: 0;
  transition: padding 0.22s cubic-bezier(0.4, 0, 0.2, 1),
              gap 0.22s cubic-bezier(0.4, 0, 0.2, 1);
}

.tasks-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
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
  min-height: 0;
}

/* В режиме канбана общий вертикальный скролл выключаем — прокрутка живёт
   внутри каждой колонки. Дочерний TaskKanban растягивается на всю высоту
   области, чтобы колонкам было где скроллиться. */
.cards-area--board { overflow-y: hidden; }
.cards-area--board > .kanban { flex: 1; min-height: 0; }

/* Колонки канбана — тот же стеклянный язык, что и карточки/рейка. */
.cards-area--board :deep(.kanban-col) {
  background: var(--acrylic-card-bg);
  border-color: var(--acrylic-border);
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

.tasks-empty {
  margin: auto;
}

.btn-retry {
  margin-top: 4px;
  padding: 9px 22px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: var(--acrylic-card-bg);
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
  background: var(--acrylic-card-bg);
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
  /* На мобильном шапка остаётся плотной плашкой (как раньше). */
  .tasks-header {
    padding: 10px 12px 8px;
    gap: 10px;
    background: var(--acrylic-card-bg);
    border-bottom: 1px solid var(--color-outline-dim);
  }

  /* Компактный режим при скролле вниз — экономим вертикальное место,
     контент не «дёргается»: шапка сжимается плавно. */
  .tasks-header.is-compact {
    padding-top: 6px;
    padding-bottom: 4px;
    gap: 6px;
  }

  .desktop-only {
    display: none;
  }

  .mobile-only {
    display: inline-flex;
  }

  .btn-icon {
    width: 44px;
    height: 44px;
    background: var(--color-surface-high);
    border-color: transparent;
  }

  .btn-icon.has-dot::after {
    border-color: var(--color-surface-high);
  }

  .cards-grid {
    grid-template-columns: 1fr;
    gap: 10px;
  }

  .cards-area {
    /* Резервируем место под нижнюю навигацию (64px) + extended FAB (~72px вместе
       с отступом). safe-area-inset-bottom — для iPhone home indicator. */
    padding: 14px 12px;
    padding-bottom: calc(64px + 96px + env(safe-area-inset-bottom, 0px));
  }

}

@media (max-width: 480px) {
  .tasks-header {
    padding: 8px 10px 6px;
  }
}
</style>
