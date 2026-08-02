<template>
  <AppPage title="Задачи" :commands="commands" flush :scroll="false" @command="onCommand">
    <template #subhead="{ narrow }">
      <SearchField
        :model-value="searchQuery"
        placeholder="Поиск по названию задачи…"
        hotkey
        :collapsible="false"
        @update:model-value="onSearchInput"
      />

      <!-- Вид (сетка/список/канбан) — только когда панель широкая: в узкой
           канбан всё равно нечитаем, а место нужно поиску. -->
      <AppTabs
        v-if="!narrow"
        variant="tint"
        dense
        :model-value="viewMode"
        :tabs="viewTabs"
        @update:model-value="setViewMode"
      />

      <!-- Узкая панель: фильтры и сортировка живут в шторках, а не в рейке. -->
      <AppTabs
        v-if="narrow"
        :model-value="tasksStore.filters.tab"
        :tabs="tabs"
        full-width
        dense
        @update:model-value="tasksStore.setTab($event)"
      />
    </template>

    <template #default="{ narrow }">
    <!-- Режим отпуска: создание/редактирование задач и юниты закрыты -->
    <AppInfoBar
      v-if="onVacation"
      class="vacation-banner"
      tone="warning"
      icon="beach_access"
      message="Вы в отпуске — создание и редактирование задач недоступно."
    >
      <template #actions>
        <AppButton
          size="sm"
          icon="person"
          label="Аккаунт"
          @click="router.push('/settings?section=account')"
        />
      </template>
    </AppInfoBar>

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
          <BrandLoader />
        </div>
        <template v-else>
          <EmptyState
            v-if="tasksStore.error"
            class="tasks-empty"
            icon="error_outline"
            tone="error"
            :subtitle="tasksStore.error"
          >
            <AppButton label="Повторить" @click="tasksStore.fetchTasks()" />
          </EmptyState>
          <EmptyState
            v-else-if="tasksStore.tasks.length === 0"
            class="tasks-empty"
            :icon="emptyIcon"
            :title="emptyTitle"
            :subtitle="emptySub"
          >
            <AppButton
              v-if="canCreateTask && tasksStore.filters.tab === 'active'"
              variant="filled"
              icon="add"
              label="Создать задачу"
              @click="showCreateTask = true"
            />
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
            <AppButton
              variant="icon"
              size="sm"
              icon="chevron_left"
              aria-label="Предыдущая страница"
              :disabled="tasksStore.filters.page === 1"
              @click="tasksStore.setFilter('page', tasksStore.filters.page - 1)"
            />
            <span class="page-info">{{ tasksStore.filters.page }} / {{ totalPages }}</span>
            <AppButton
              variant="icon"
              size="sm"
              icon="chevron_right"
              aria-label="Следующая страница"
              :disabled="tasksStore.tasks.length < tasksStore.filters.per_page"
              @click="tasksStore.setFilter('page', tasksStore.filters.page + 1)"
            />
          </div>
        </template>
      </main>
      </template>
    </div>

    <SortSheet :visible="showSortSheet" @close="showSortSheet = false" />

    <TaskModal
      v-if="tasksStore.activeTask"
      :task="tasksStore.activeTask"
      @close="tasksStore.closeTask()"
    />

    <TaskForm
      v-if="showCreateTask"
      :task="null"
      :preset-name="createPresetName"
      @close="closeCreateTask"
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

    <!-- Ссылка на задачу, которой нет доступа: задача — сущность компании,
         поэтому постороннему говорим прямо, а своему из другой компании
         предлагаем переключиться. -->
    <AppDialog
      :model-value="!!linkError"
      tone="warning"
      size="sm"
      :title="linkError?.title || ''"
      @update:model-value="linkError = null"
    >
      <p class="tasks-linkerr">{{ linkError?.message }}</p>
      <div class="tasks-linkerr-actions">
        <button class="btn-glass" @click="linkError = null">Закрыть</button>
        <button v-if="linkError?.companyId" class="btn-grad" :disabled="switching" @click="switchAndOpen">
          <span class="material-symbols-outlined">swap_horiz</span>
          {{ switching ? 'Переключаем…' : 'Переключить компанию' }}
        </button>
      </div>
    </AppDialog>

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
    </template>
  </AppPage>
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
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AppPage from '@/components/ui/AppPage.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import SearchField from '@/components/common/SearchField.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
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
// Отказ по ссылке на задачу: { title, message, companyId?, taskId? }.
const linkError = ref(null)
const switching = ref(false)
// Название, с которым открыта форма создания (команда «создай задачу …» из
// строки поиска рабочего стола).
const createPresetName = ref('')
const showImportYg = ref(false)

const yougileStore = useYougileStore()
const yougileAvailable = computed(() => yougileStore.isAvailable)
const searchQuery = ref(tasksStore.filters.search)
const showMobileFilters = ref(false)
const showSortSheet = ref(false)

/* Команды раздела. Сортировка и фильтры нужны только узкой панели: широкой
   служит рейка фильтров слева, где они и живут. */
const commands = computed(() => [
  ...(canCreateTask.value
    ? [{ key: 'create', label: 'Добавить', icon: 'add', variant: 'filled', primary: true, fab: true }]
    : []),
  { key: 'sort', label: 'Сортировка', icon: 'sort', hidden: !isMobile.value },
  { key: 'filters', label: 'Фильтры', icon: 'tune', hidden: !isMobile.value },
  ...(canCreateTask.value && yougileAvailable.value
    ? [{ key: 'yougile', label: 'Импорт из YouGile', icon: 'sync_alt' }]
    : []),
])

function onCommand(key) {
  if (key === 'create') showCreateTask.value = true
  else if (key === 'sort') showSortSheet.value = true
  else if (key === 'filters') showMobileFilters.value = true
  else if (key === 'yougile') showImportYg.value = true
}
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

const onVacation = computed(() => !!auth.user?.on_vacation)
const canCreateTask = computed(() => isAtLeast(ROLES.EMPLOYEE) && !onVacation.value)
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
  if (onVacation.value) {
    notif.warn('Вы в отпуске — юниты подождут возвращения')
    return
  }
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
  else if (action === 'copy-link') copyTaskLink(task)
  else if (action === 'archive') askArchiveTask(task)
}

async function copyTaskLink(task) {
  try {
    await navigator.clipboard.writeText(`${location.origin}/tasks/${task.id}`)
    notif.success('Ссылка на задачу скопирована')
  } catch {
    notif.error('Не удалось скопировать ссылку')
  }
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

function closeCreateTask() {
  showCreateTask.value = false
  createPresetName.value = ''
}

function onTaskCreated(task) {
  closeCreateTask()
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

/* Форма создания с готовым названием: `/tasks?new=1&title=…` — команда
   «создай задачу …» из строки поиска рабочего стола. */
function consumeCreateQuery() {
  if (!route.query.new) return
  if (!canCreateTask.value) return
  createPresetName.value = String(route.query.title || '')
  showCreateTask.value = true
  router.replace({ path: '/tasks' }).catch(() => {})
}

function consumeOpenQuery() {
  // Источника два: canonical `/tasks/:id` (params.id) и legacy `/tasks?open=…`.
  // Второй вариант — для утреннего брифинга Грувика/уведомлений/совместимости.
  const openId = route.params.id || route.query.open
  if (!openId) return
  openTaskByLink(Number(openId))
  // Сворачиваем URL обратно к /tasks, чтобы повторный клик на ту же задачу
  // (или history.back) снова открыл модалку.
  router.replace({ path: '/tasks' })
}

/* Переход по ссылке на задачу. В отличие от клика по карточке, здесь заранее
   ничего не известно: задача может быть чужой компании — тогда вместо пустой
   карточки показываем, чего не хватает. */
async function openTaskByLink(id) {
  try {
    tasksStore.openTask(await getTask(id))
  } catch (e) {
    if (e?.error === 'TASK_OTHER_COMPANY') {
      linkError.value = {
        title: 'Задача другой компании',
        message: 'Эта задача принадлежит другой вашей компании. Переключитесь на неё, чтобы открыть.',
        companyId: e.company_id ?? null,
        taskId: id,
      }
    } else if (e?.status === 403) {
      linkError.value = {
        title: 'Доступ ограничен',
        message: 'Задача принадлежит компании, в которой вы не состоите. Попросите доступ у её администратора.',
      }
    } else {
      linkError.value = { title: 'Задача не найдена', message: 'Возможно, её удалили или ссылка неверна.' }
    }
  }
}

async function switchAndOpen() {
  const { companyId, taskId } = linkError.value || {}
  if (!companyId || switching.value) return
  switching.value = true
  try {
    await auth.switchCompany(companyId)
    linkError.value = null
    await openTaskByLink(taskId)
  } catch (e) {
    notif.error(e?.message || 'Не удалось переключить компанию')
  } finally {
    switching.value = false
  }
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
  consumeCreateQuery()
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
watch(() => route.query.new, (v) => {
  if (v) consumeCreateQuery()
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
/* Отказ по ссылке на задачу */
.tasks-linkerr {
  margin: 0;
  font-size: 14px;
  line-height: 1.5;
  color: var(--color-text-dim);
}

.tasks-linkerr-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 18px;
  flex-wrap: wrap;
}

/* Полоса режима отпуска: вид у неё общий, здесь — только место в раскладке. */
.vacation-banner { flex-shrink: 0; margin: 8px 16px 0; }

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

/* Пагинация */
.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 8px 0 4px;
}

.page-info {
  min-width: 48px;
  text-align: center;
  font-size: 14px;
  font-weight: 650;
  color: var(--color-text);
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
