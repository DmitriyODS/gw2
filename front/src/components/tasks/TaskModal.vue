<template>
  <Dialog
    :visible="true"
    @update:visible="$emit('close')"
    modal
    :closable="false"
    :style="dialogStyle"
    :pt="dialogPt"
  >
    <div class="task-modal-body" :class="{ 'mobile-layout': isMobile }">
      <!-- ─── Мобильная шапка: M3 Top App Bar ─── -->
      <header v-if="isMobile" class="mobile-topbar">
        <button class="topbar-icon-btn" @click="$emit('close')" aria-label="Закрыть">
          <span class="material-symbols-outlined">arrow_back</span>
        </button>
        <div class="topbar-title-wrap">
          <span class="topbar-eyebrow">Задача №{{ task.id }}</span>
          <span class="topbar-title">{{ task.name }}</span>
        </div>
        <button
          class="topbar-icon-btn"
          :class="{ 'is-fav': task.is_favorite }"
          @click="handleToggleFavorite"
          :aria-label="task.is_favorite ? 'Убрать из избранного' : 'Добавить в избранное'"
        >
          <span class="material-symbols-outlined" :class="{ filled: task.is_favorite }">
            {{ task.is_favorite ? 'favorite' : 'favorite_border' }}
          </span>
        </button>
        <button
          class="topbar-icon-btn"
          :class="{ active: showMobileMenu }"
          @click="showMobileMenu = !showMobileMenu"
          aria-label="Дополнительно"
        >
          <span class="material-symbols-outlined">more_vert</span>
        </button>

        <TaskColorPopover
          v-model="showColorPicker"
          :anchor="colorBtnRef"
          :value="task.color || null"
          @select="handleSetColor"
        />
      </header>

      <!-- Overflow-меню действий (вынесено в body через Teleport, чтобы не
           обрезалось overflow:hidden у PrimeVue Dialog content). -->
      <Teleport to="body">
        <div v-if="isMobile && showMobileMenu" class="mobile-menu-backdrop" @click="showMobileMenu = false" />
        <Transition name="mobile-menu">
          <div v-if="isMobile && showMobileMenu" class="mobile-menu" @click.stop>
            <button class="mm-item" @click="onMobileMenuAction('copy-link')">
              <span class="material-symbols-outlined">link</span>
              Скопировать ссылку
            </button>
            <button class="mm-item" ref="colorBtnRef" @click="onMobileMenuAction('color')">
              <span class="material-symbols-outlined">palette</span>
              Цвет задачи
            </button>
            <button v-if="canEditTask" class="mm-item" @click="onMobileMenuAction('edit')">
              <span class="material-symbols-outlined">edit</span>
              Редактировать
            </button>
            <button v-if="canDeleteTask" class="mm-item danger" @click="onMobileMenuAction('delete')">
              <span class="material-symbols-outlined">delete</span>
              Удалить
            </button>
          </div>
        </Transition>
      </Teleport>

      <!-- ─── Мобильные табы (3 шт., без дубля закрытия) ─── -->
      <div v-if="isMobile" class="mobile-tabs" role="tablist">
        <button
          class="mobile-tab-btn"
          :class="{ active: mobileTab === 'details' }"
          role="tab"
          :aria-selected="mobileTab === 'details'"
          @click="mobileTab = 'details'"
        >
          <span class="material-symbols-outlined">info</span>
          Детали
        </button>
        <button
          class="mobile-tab-btn"
          :class="{ active: mobileTab === 'units' && rightTab === 'units' }"
          role="tab"
          :aria-selected="mobileTab === 'units' && rightTab === 'units'"
          @click="mobileTab = 'units'; rightTab = 'units'"
        >
          <span class="material-symbols-outlined">timer</span>
          Юниты
        </button>
        <button
          class="mobile-tab-btn"
          :class="{ active: mobileTab === 'units' && rightTab === 'comments' }"
          role="tab"
          :aria-selected="mobileTab === 'units' && rightTab === 'comments'"
          @click="mobileTab = 'units'; rightTab = 'comments'"
        >
          <span class="material-symbols-outlined">forum</span>
          Комментарии
        </button>
      </div>

      <!-- Левая панель (детали задачи) -->
      <div class="task-left" :class="{ hidden: isMobile && mobileTab !== 'details' }">
        <!-- На мобильном эти actions уехали в Top App Bar — здесь они скрыты. -->
        <div v-if="!isMobile" class="task-top-row">
          <span class="task-id-label">№{{ task.id }}</span>
          <div class="task-top-actions">
            <button
              class="icon-btn-round favorite"
              :class="{ 'is-fav': task.is_favorite }"
              @click="handleToggleFavorite"
              :title="task.is_favorite ? 'Убрать из избранного' : 'Добавить в избранное'"
            >
              <span class="material-symbols-outlined" :class="{ filled: task.is_favorite }">
                {{ task.is_favorite ? 'favorite' : 'favorite_border' }}
              </span>
            </button>
            <button
              class="icon-btn-round"
              @click="copySelfLink"
              title="Скопировать ссылку на задачу"
            >
              <span class="material-symbols-outlined">link</span>
            </button>
            <button
              ref="colorBtnRef"
              class="icon-btn-round"
              :class="{ active: showColorPicker }"
              @click="showColorPicker = !showColorPicker"
              title="Цвет задачи"
            >
              <span class="material-symbols-outlined">palette</span>
            </button>
            <TaskColorPopover
              v-model="showColorPicker"
              :anchor="colorBtnRef"
              :value="task.color || null"
              @select="handleSetColor"
            />
            <button v-if="canEditTask" class="icon-btn-round" @click="showEditForm = true" title="Редактировать">
              <span class="material-symbols-outlined">edit</span>
            </button>
            <button v-if="canDeleteTask" class="icon-btn-round danger" @click="confirmDelete" title="Удалить">
              <span class="material-symbols-outlined">delete</span>
            </button>
          </div>
        </div>

        <!-- Название задачи: на мобильном — уже в Top App Bar, повторно не показываем. -->
        <h2 v-if="!isMobile" class="task-title">{{ task.name }}</h2>

        <!-- Заказчик -->
        <div class="field-box">
          <div class="field-label">Заказчик</div>
          <div class="field-value">{{ task.department?.name || '—' }}</div>
        </div>

        <!-- Даты в ряд -->
        <div class="fields-row">
          <div class="field-box half">
            <div class="field-label">Дата поступления</div>
            <div class="field-value with-icon">
              <span class="material-symbols-outlined field-icon">calendar_today</span>
              {{ formatDate(task.received_at) }}
            </div>
          </div>
          <div class="field-box half">
            <div class="field-label">Дата создания</div>
            <div class="field-value with-icon">
              <span class="material-symbols-outlined field-icon">calendar_today</span>
              {{ formatDateTime(task.created_at) }}
            </div>
          </div>
        </div>

        <!-- YouGile -->
        <div v-if="usesYougile && (task.link_yougile || yougileAvailable)" class="field-box">
          <div class="field-label">
            <span class="material-symbols-outlined field-label-icon">link</span>
            YouGile
          </div>
          <!-- Связь есть -->
          <div v-if="task.link_yougile" class="field-value yougile-value">
            <span class="yougile-url">{{ task.link_yougile }}</span>
            <button class="action-btn-round" @click="copyLink" title="Скопировать ссылку">
              <span class="material-symbols-outlined">content_copy</span>
            </button>
            <a :href="task.link_yougile" target="_blank" class="action-btn-round" title="Открыть в новой вкладке">
              <span class="material-symbols-outlined">open_in_new</span>
            </a>
            <button v-if="yougileAvailable" class="action-btn-round danger"
                    :disabled="ygBusy" @click="onUnlinkYg" title="Отвязать от YouGile">
              <span class="material-symbols-outlined">link_off</span>
            </button>
          </div>
          <!-- Связи нет, но интеграция доступна — предлагаем создать -->
          <div v-else-if="yougileAvailable" class="field-value yougile-empty">
            <span class="text-dim">Карточка не привязана</span>
            <button class="btn-tonal" :disabled="ygBusy" @click="onExportYg">
              <span class="material-symbols-outlined">cloud_upload</span>
              {{ ygBusy ? 'Создаём…' : 'Создать в YouGile' }}
            </button>
          </div>
        </div>

        <!-- Ответственный (read-only — редактируется в режиме «Редактировать») -->
        <div class="field-box">
          <div class="field-label">Ответственный</div>
          <div class="field-value responsible-value">
            <template v-if="responsibleDisplay">
              <img :src="responsibleAvatar" class="responsible-avatar" alt="" />
              <span class="responsible-name">{{ responsibleDisplay.fio }}</span>
            </template>
            <template v-else>
              <span class="material-symbols-outlined field-icon">person_off</span>
              <span class="text-dim">Не назначен</span>
            </template>
          </div>
        </div>

        <!-- Этап (read-only — редактируется в режиме «Редактировать») -->
        <div v-if="usesStages" class="field-box">
          <div class="field-label">Этап</div>
          <div class="field-value">
            <template v-if="currentStage">{{ currentStage.name }}</template>
            <span v-else class="text-dim">Без этапа</span>
          </div>
        </div>

        <!-- Теги (правятся прямо здесь — набор общий для компании) -->
        <div class="field-box">
          <div class="field-label tm-tags-label">
            Теги
            <button
              class="tm-tags-edit"
              type="button"
              :title="tagsEditing ? 'Готово' : 'Изменить теги'"
              :aria-label="tagsEditing ? 'Готово' : 'Изменить теги'"
              @click="toggleTagsEditing"
            >
              <span class="material-symbols-outlined">{{ tagsEditing ? 'check' : 'add' }}</span>
            </button>
          </div>
          <div v-if="!tagsEditing" class="field-value tm-tags-value">
            <span
              v-for="tg in task.tags || []"
              :key="tg.id"
              class="tm-tag-chip"
              :style="{ background: `var(--tag-${tg.color}-surface)`, color: `var(--tag-${tg.color}-accent)` }"
            >
              <span class="material-symbols-outlined tm-tag-icon">sell</span>
              {{ tg.name }}
            </span>
            <span v-if="!(task.tags || []).length" class="text-dim">Без тегов</span>
          </div>
          <div v-else class="tm-tags-editor">
            <button
              v-for="t in tasksStore.tags"
              :key="t.id"
              class="tm-tag-chip tm-tag-pick"
              :class="{ active: taskTagIds.includes(t.id) }"
              :style="{ background: `var(--tag-${t.color}-surface)`, color: `var(--tag-${t.color}-accent)` }"
              type="button"
              @click="onToggleTag(t.id)"
            >
              <span class="material-symbols-outlined tm-tag-icon">
                {{ taskTagIds.includes(t.id) ? 'check_box' : 'check_box_outline_blank' }}
              </span>
              {{ t.name }}
            </button>
            <span v-if="!tasksStore.tags.length" class="text-dim">
              Тегов пока нет — их создаёт менеджер в панели фильтров задач
            </span>
          </div>
        </div>

        <!-- Дедлайн -->
        <div v-if="task.deadline" class="field-box">
          <div class="field-label">Дедлайн</div>
          <div class="field-value with-icon" :class="{ overdue: isOverdue }">
            <span class="material-symbols-outlined field-icon">calendar_today</span>
            {{ formatDate(task.deadline) }}
          </div>
        </div>

        <!-- Создатель -->
        <div class="field-box">
          <div class="field-label">Создатель задачи</div>
          <div class="field-value">{{ task.author?.fio || '—' }}</div>
        </div>

        <!-- Работали над задачей -->
        <TaskContributors :task-id="task.id" />

        <!-- Нижние кнопки — только на десктопе. На мобильном они вынесены
             в sticky bottom action bar (см. ниже). -->
        <div v-if="!isMobile" class="task-bottom-actions">
          <button
            v-if="!task.is_archived && canEditTask"
            class="btn-full pill primary-btn"
            @click="confirmArchive"
            :disabled="actionLoading"
          >
            <span class="material-symbols-outlined">check_circle</span>
            Завершить задачу
          </button>
          <button
            v-if="task.is_archived && canEditTask"
            class="btn-full pill accent-btn"
            @click="handleRestore"
            :disabled="actionLoading"
          >
            <span class="material-symbols-outlined">unarchive</span>
            Вернуть из архива
          </button>
        </div>
      </div>

      <!-- Правая панель (юниты / комментарии) -->
      <div class="task-right" :class="{ hidden: isMobile && mobileTab === 'details' }">
        <!-- units-header — только на десктопе. На мобильном:
             - переключение юниты/комментарии — в верхних табах
             - «Начать юнит» — в sticky bottom action bar
             - закрытие — в Top App Bar -->
        <div v-if="!isMobile" class="units-header">
          <div class="right-tabs">
            <button
              class="right-tab"
              :class="{ active: rightTab === 'units' }"
              @click="rightTab = 'units'"
            >
              <span class="material-symbols-outlined">timer</span>
              Юниты
            </button>
            <button
              class="right-tab"
              :class="{ active: rightTab === 'comments' }"
              @click="rightTab = 'comments'"
            >
              <span class="material-symbols-outlined">forum</span>
              Комментарии
            </button>
          </div>
          <div class="units-header-actions">
            <button
              v-if="rightTab === 'units' && canStartUnit"
              class="btn-start-unit pill"
              @click="showStartUnit = true"
            >
              <span class="material-symbols-outlined">add</span>
              Начать юнит
            </button>
            <button class="btn-close-round" @click="$emit('close')" title="Закрыть">
              <span class="material-symbols-outlined">close</span>
            </button>
          </div>
        </div>

        <template v-if="rightTab === 'units'">
          <div v-if="unitsLoading" class="units-loading">
            <span class="material-symbols-outlined spinning">progress_activity</span>
            Загрузка юнитов...
          </div>
          <div v-else class="units-list">
            <UnitListItem
              v-for="unit in units"
              :key="unit.id"
              :unit="unit"
              @edit="openEditUnit"
              @delete="confirmDeleteUnit"
              @clone="confirmCloneUnit"
            />
            <div v-if="units.length === 0" class="no-units">
              <span class="material-symbols-outlined">hourglass_empty</span>
              Юнитов пока нет
            </div>
          </div>
        </template>
        <TaskComments v-else :task-id="task.id" />
      </div>

      <!-- ─── Sticky bottom action bar (только мобильный) ─── -->
      <div v-if="isMobile && mobileBottomAction" class="mobile-bottom-bar">
        <button
          class="bottom-action-btn"
          :class="mobileBottomAction.cls"
          :disabled="actionLoading"
          @click="mobileBottomAction.onClick"
        >
          <span class="material-symbols-outlined">{{ mobileBottomAction.icon }}</span>
          {{ mobileBottomAction.label }}
        </button>
      </div>
    </div>

    <!-- Вложенные модалки -->
    <TaskForm
      v-if="showEditForm"
      :task="task"
      @close="showEditForm = false"
      @saved="onTaskSaved"
    />

    <StartUnitModal
      v-if="showStartUnit"
      :task-id="task.id"
      @close="showStartUnit = false"
      @started="onUnitStarted"
    />

    <UnitEditModal
      v-if="editingUnit"
      :unit="editingUnit"
      @close="editingUnit = null"
      @saved="loadUnits"
    />

    <!-- Диалог подтверждения -->
    <AppDialog
      v-if="confirmDialog.visible"
      v-model="confirmDialog.visible"
      :tone="confirmDialog.tone || 'danger'"
      :icon="confirmDialog.icon || 'warning'"
      size="sm"
      :title="confirmDialog.title"
      :subtitle="confirmDialog.message"
      :actions="[
        { kind: 'cancel', label: 'Отмена' },
        { kind: 'confirm', label: confirmDialog.confirmLabel || 'Подтвердить', disabled: actionLoading },
      ]"
      @confirm="confirmDialog.onConfirm(); confirmDialog.visible = false"
    />
  </Dialog>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import Dialog from 'primevue/dialog'
import AppDialog from '@/components/common/AppDialog.vue'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { getSocket } from '@/socket/index.js'
import UnitListItem from '@/components/tasks/UnitListItem.vue'
import TaskForm from '@/components/tasks/TaskForm.vue'
import TaskColorPopover from '@/components/tasks/TaskColorPopover.vue'
import TaskComments from '@/components/tasks/TaskComments.vue'
import TaskContributors from '@/components/tasks/TaskContributors.vue'
import { useCompanySettings } from '@/composables/useCompanySettings.js'
import { getStages } from '@/api/stages.js'
import { getDirectoryUser } from '@/api/users.js'
import StartUnitModal from '@/components/units/StartUnitModal.vue'
import UnitEditModal from '@/components/units/UnitEditModal.vue'
import { getUnits, deleteUnit, createUnit } from '@/api/units.js'
import { deleteTask, archiveTask, restoreTask, toggleFavorite as apiFavorite, setTaskColor } from '@/api/tasks.js'
import { exportYougileTask, unlinkYougileTask } from '@/api/yougile.js'
import { useYougileStore } from '@/stores/yougile.js'
import { useTasksStore } from '@/stores/tasks.js'
import { useUnitsStore } from '@/stores/units.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import { useAuthStore } from '@/stores/auth.js'

const props = defineProps({
  task: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['close'])

const tasksStore = useTasksStore()
const unitsStore = useUnitsStore()
const notifications = useNotificationsStore()
const authStore = useAuthStore()
const { isAtLeast } = usePermission()
const { isMobile } = useBreakpoint()

const mobileTab = ref('details')
const rightTab = ref('units')
const showMobileMenu = ref(false)
const stages = ref([])
const { usesYougile, usesStages } = useCompanySettings()

const yougileStore = useYougileStore()
const yougileAvailable = computed(() => yougileStore.isAvailable)
const ygBusy = ref(false)

async function onExportYg() {
  ygBusy.value = true
  try {
    const updated = await exportYougileTask({ gw_task_id: props.task.id })
    tasksStore.upsertTask(updated)
    notifications.success('Карточка создана в YouGile')
  } catch (e) {
    notifications.error(e?.data?.message || e?.message || 'Не удалось создать в YouGile')
  } finally {
    ygBusy.value = false
  }
}

async function onUnlinkYg() {
  ygBusy.value = true
  try {
    const updated = await unlinkYougileTask(props.task.id)
    tasksStore.upsertTask(updated)
    notifications.success('Связь с YouGile разорвана')
  } catch (e) {
    notifications.error(e?.data?.message || e?.message || 'Не удалось отвязать')
  } finally {
    ygBusy.value = false
  }
}

const dialogStyle = computed(() => {
  if (isMobile.value) {
    return 'width: 100vw; max-width: 100vw; height: 100dvh; max-height: 100dvh; border-radius: 0; margin: 0'
  }
  return 'width: 960px; max-width: 95vw; border-radius: 20px'
})

const dialogPt = computed(() => ({
  header: { style: 'display:none' },
  content: {
    style: isMobile.value
      ? 'padding:0; overflow:hidden; border-radius:0; height:100%; display:flex; flex-direction:column'
      : 'padding:0; overflow:hidden; border-radius:20px; display:flex; flex-direction:column'
  }
}))

const units = ref([])
const unitsLoading = ref(false)
const showEditForm = ref(false)
const showStartUnit = ref(false)
const showColorPicker = ref(false)
const editingUnit = ref(null)
const actionLoading = ref(false)
const colorBtnRef = ref(null)

const confirmDialog = ref({
  visible: false,
  title: '',
  message: '',
  confirmLabel: '',
  onConfirm: () => {}
})

const isOwnTask = computed(() => props.task.author_id === authStore.user?.id)

const canEditTask = computed(() => isAtLeast(ROLES.EMPLOYEE))

const canDeleteTask = computed(() => isAtLeast(ROLES.EMPLOYEE))

const canStartUnit = computed(() => {
  if (props.task.is_archived) return false
  if (unitsStore.activeUnit) return false
  return isAtLeast(ROLES.EMPLOYEE)
})

const isOverdue = computed(() => {
  if (!props.task.deadline) return false
  return new Date(props.task.deadline) < new Date()
})

const responsibleDirectory = ref(null)
const responsibleDisplay = computed(() => {
  const t = props.task
  if (!t.responsible_user_id) return null
  if (t.responsible && t.responsible.fio) return t.responsible
  return responsibleDirectory.value
})
const responsibleAvatar = computed(() => {
  const u = responsibleDisplay.value
  if (!u) return ''
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
})
watch(
  () => props.task.responsible_user_id,
  async (id) => {
    if (!id) { responsibleDirectory.value = null; return }
    if (props.task.responsible?.fio) { responsibleDirectory.value = null; return }
    try { responsibleDirectory.value = await getDirectoryUser(id) }
    catch { responsibleDirectory.value = null }
  },
  { immediate: true },
)

const currentStage = computed(() => {
  const sid = props.task.stage_id
  if (!sid) return null
  return stages.value.find((s) => s.id === sid) || null
})

/* ── Теги: правка прямо из модалки (мультивыбор чипами) ── */
const tagsEditing = ref(false)

const taskTagIds = computed(() => {
  // Актуальные теги — из стора (patchTask обновляет после каждого toggle).
  const fresh = tasksStore.taskById.get(props.task.id)
  return ((fresh || props.task).tags || []).map((t) => t.id)
})

function toggleTagsEditing() {
  tagsEditing.value = !tagsEditing.value
  if (tagsEditing.value) tasksStore.fetchTags()
}

function onToggleTag(tagId) {
  tasksStore.toggleTaskTag(props.task.id, tagId).catch(() => {
    notifications.error('Не удалось изменить теги')
  })
}

/* Действие в sticky-баре зависит от активной мобильной вкладки.
   - Детали: «Завершить задачу» (или «Вернуть из архива»)
   - Юниты: «Начать юнит» (если разрешено и нет активного)
   - Чат: нет sticky-кнопки (у комментариев свой инпут) */
const mobileBottomAction = computed(() => {
  if (mobileTab.value === 'details') {
    if (props.task.is_archived && canEditTask.value) {
      return {
        icon: 'unarchive',
        label: 'Вернуть из архива',
        cls: 'tone-tertiary',
        onClick: handleRestore,
      }
    }
    if (!props.task.is_archived && canEditTask.value) {
      return {
        icon: 'check_circle',
        label: 'Завершить задачу',
        cls: 'tone-primary',
        onClick: confirmArchive,
      }
    }
    return null
  }
  if (mobileTab.value === 'units' && rightTab.value === 'units' && canStartUnit.value) {
    return {
      icon: 'play_arrow',
      label: 'Начать юнит',
      cls: 'tone-secondary',
      onClick: () => { showStartUnit.value = true },
    }
  }
  return null
})

function onMobileMenuAction(action) {
  showMobileMenu.value = false
  if (action === 'color') {
    // Открываем popover чуть позже — после закрытия меню, чтобы anchor
    // успел смонтироваться/закрепиться.
    setTimeout(() => { showColorPicker.value = true }, 50)
  } else if (action === 'edit') {
    showEditForm.value = true
  } else if (action === 'delete') {
    confirmDelete()
  } else if (action === 'copy-link') {
    copySelfLink()
  }
}

onMounted(() => {
  loadUnits()
  if (usesStages.value) loadStages()
  subscribeUnitEvents()
})

onBeforeUnmount(() => unsubscribeUnitEvents())

/* Список юнитов живёт локально и без подписки устаревает: остановив юнит из
   ActiveUnitModal/карточки, пользователь открывал редактирование со старым
   объектом (datetime_end=null) — и поле «окончание» не показывалось. */
const unitHandlers = {
  'unit:started': (unit) => {
    if (unit.task_id !== props.task.id) return
    if (!units.value.some((u) => u.id === unit.id)) units.value.unshift(unit)
  },
  'unit:stopped': ({ unit_id, task_id, datetime_end }) => {
    if (task_id !== props.task.id) return
    const u = units.value.find((x) => x.id === unit_id)
    if (u && datetime_end) u.datetime_end = datetime_end
  },
  'unit:updated': (data) => {
    const u = units.value.find((x) => x.id === data.unit_id)
    if (u) Object.assign(u, data)
  },
  'unit:deleted': ({ unit_id }) => {
    const idx = units.value.findIndex((x) => x.id === unit_id)
    if (idx !== -1) units.value.splice(idx, 1)
  },
}

function subscribeUnitEvents() {
  const socket = getSocket()
  if (!socket) return
  for (const [event, handler] of Object.entries(unitHandlers)) socket.on(event, handler)
}

function unsubscribeUnitEvents() {
  const socket = getSocket()
  if (!socket) return
  for (const [event, handler] of Object.entries(unitHandlers)) socket.off(event, handler)
}

async function loadStages() {
  try {
    const data = await getStages()
    stages.value = Array.isArray(data) ? data : (data.items ?? [])
  } catch {
    stages.value = []
  }
}

async function loadUnits() {
  unitsLoading.value = true
  try {
    const data = await getUnits(props.task.id)
    units.value = Array.isArray(data) ? data : (data.units ?? data.items ?? [])
  } catch {
    units.value = []
  } finally {
    unitsLoading.value = false
  }
}

function formatDate(d) {
  if (!d) return '—'
  return new Date(d).toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' })
}

function formatDateTime(d) {
  if (!d) return '—'
  return new Date(d).toLocaleString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit' }).replace(',', ' г.,')
}

async function copyLink() {
  try {
    await navigator.clipboard.writeText(props.task.link_yougile)
    notifications.success('Ссылка скопирована')
  } catch {
    notifications.error('Не удалось скопировать ссылку')
  }
}

async function copySelfLink() {
  // Canonical-ссылка на задачу внутри GW. Используем абсолютный URL, чтобы её
  // можно было сразу скинуть в мессенджер/почту.
  const url = `${window.location.origin}/tasks/${props.task.id}`
  try {
    await navigator.clipboard.writeText(url)
    notifications.success('Ссылка на задачу скопирована')
  } catch {
    notifications.error('Не удалось скопировать ссылку')
  }
}

function confirmDelete() {
  confirmDialog.value = {
    visible: true,
    title: 'Удалить задачу',
    message: `Вы уверены, что хотите удалить задачу "${props.task.name}"? Это действие нельзя отменить.`,
    confirmLabel: 'Удалить',
    onConfirm: handleDelete
  }
}

async function handleDelete() {
  actionLoading.value = true
  try {
    await deleteTask(props.task.id)
    tasksStore.removeTask(props.task.id)
    notifications.success('Задача удалена')
    emit('close')
  } catch (e) {
    notifications.error(e?.message || 'Не удалось удалить задачу')
  } finally {
    actionLoading.value = false
  }
}

function confirmArchive() {
  confirmDialog.value = {
    visible: true,
    title: 'Завершить задачу',
    message: `Завершить задачу "${props.task.name}"? Задача будет перемещена в архив.`,
    confirmLabel: 'Завершить',
    onConfirm: handleArchive
  }
}

async function handleArchive() {
  actionLoading.value = true
  try {
    const result = await archiveTask(props.task.id)
    tasksStore.archiveTask(props.task.id, result?.archived_at)
    notifications.success('Задача завершена и перемещена в архив')
    emit('close')
  } catch (e) {
    if (e?.status === 409) {
      notifications.error('Нельзя архивировать задачу с активным юнитом')
    } else {
      notifications.error(e?.message || 'Не удалось завершить задачу')
    }
  } finally {
    actionLoading.value = false
  }
}

async function handleRestore() {
  actionLoading.value = true
  try {
    await restoreTask(props.task.id)
    tasksStore.restoreTask(props.task.id)
    notifications.success('Задача возвращена из архива')
    emit('close')
  } catch (e) {
    notifications.error(e?.message || 'Не удалось вернуть задачу')
  } finally {
    actionLoading.value = false
  }
}

function openEditUnit(unit) {
  editingUnit.value = unit
}

function confirmCloneUnit(unit) {
  confirmDialog.value = {
    visible: true,
    tone: 'primary',
    icon: 'restart_alt',
    title: 'Создать новый юнит',
    message: `Начать новый юнит «${unit.name}» с тем же типом? Учёт времени пойдёт заново.`,
    confirmLabel: 'Создать',
    onConfirm: () => cloneUnit(unit),
  }
}

// Создать новый юнит от существующего — на себя, с тем же названием и типом.
// Сервер хранит инвариант «1 активный юнит»: при наличии активного вернёт 409.
async function cloneUnit(unit) {
  if (unitsStore.activeUnit) {
    notifications.error('У вас уже есть активный юнит')
    return
  }
  try {
    const newUnit = await createUnit(props.task.id, {
      name: unit.name,
      unit_type_id: unit.unit_type_id ?? unit.unit_type?.id,
    })
    unitsStore.startUnit(newUnit)
    notifications.success('Юнит запущен')
    loadUnits()
  } catch (e) {
    if (e?.status === 409) {
      notifications.error('У вас уже есть активный юнит')
    } else {
      notifications.error(e?.message || 'Не удалось запустить юнит')
    }
  }
}

function confirmDeleteUnit(unit) {
  confirmDialog.value = {
    visible: true,
    title: 'Удалить юнит',
    message: `Вы уверены, что хотите удалить юнит "${unit.name}"?`,
    confirmLabel: 'Удалить',
    onConfirm: () => handleDeleteUnit(unit)
  }
}

async function handleDeleteUnit(unit) {
  try {
    await deleteUnit(unit.id)
    units.value = units.value.filter(u => u.id !== unit.id)
    notifications.success('Юнит удалён')
    if (unitsStore.activeUnit?.id === unit.id) {
      unitsStore.clearActiveUnit()
    }
    // Сразу актуализируем карточку: убираем аватарку, если удалён активный
    // юнит, и снимаем индикатор юнитов, если их больше не осталось.
    if (!unit.datetime_end && unit.user_id != null) {
      tasksStore.removeActiveUser(props.task.id, unit.user_id)
    }
    if (units.value.length === 0) {
      tasksStore.patchTask({ id: props.task.id, has_units: false })
    }
  } catch (e) {
    notifications.error(e?.message || 'Не удалось удалить юнит')
  }
}

function onTaskSaved(updatedTask) {
  tasksStore.upsertTask(updatedTask)
  showEditForm.value = false
}

function onUnitStarted() {
  showStartUnit.value = false
  loadUnits()
  // Карточку задачи (индикатор юнита + аватарку) обновляет unitsStore.startUnit.
}

async function handleToggleFavorite() {
  try {
    await apiFavorite(props.task.id)
    tasksStore.setFavorite(props.task.id, !props.task.is_favorite)
  } catch (e) {
    notifications.error(e?.message || 'Не удалось изменить избранное')
  }
}

async function handleSetColor(color) {
  showColorPicker.value = false
  const prev = props.task.color ?? null
  if (prev === color) return
  tasksStore.patchTask({ id: props.task.id, color })
  try {
    await setTaskColor(props.task.id, color)
  } catch (e) {
    tasksStore.patchTask({ id: props.task.id, color: prev })
    notifications.error(e?.message || 'Не удалось изменить цвет')
  }
}
</script>

<style scoped>
.task-modal-body {
  display: flex;
  flex: 1;
  min-height: 520px;
  max-height: 85dvh;
  overflow: hidden;
}

/* ─── Левая панель ─── */
.task-left {
  width: 45%;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 24px 24px 24px 28px;
  background: transparent; /* фон даёт акриловый .p-dialog */
  border-right: 1px solid var(--gw-border);
  overflow-y: auto;
  min-height: 0;
}

.task-top-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.task-id-label {
  font-size: 13px;
  color: var(--gw-text-secondary);
  font-variant-numeric: tabular-nums;
}

.task-top-actions {
  display: flex;
  gap: 6px;
}

.icon-btn-round {
  width: 30px;
  height: 30px; min-height: 0;
  border-radius: 50%;
  border: 1px solid var(--gw-border);
  background: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--gw-text-secondary);
  transition: background 0.12s, color 0.12s, border-color 0.12s;
}

.icon-btn-round:hover {
  background: var(--gw-primary-light);
  color: var(--gw-primary);
  border-color: var(--gw-primary);
}

.icon-btn-round.danger:hover {
  background: var(--color-error-container);
  color: var(--color-error);
  border-color: color-mix(in oklch, var(--color-error) 40%, var(--color-outline-dim));
}

.icon-btn-round.active {
  background: var(--gw-primary-light);
  color: var(--gw-primary);
  border-color: var(--gw-primary);
}

.icon-btn-round.favorite:hover {
  background: var(--color-error-container);
  color: var(--color-error);
  border-color: color-mix(in oklch, var(--color-error) 40%, var(--color-outline-dim));
}

.icon-btn-round.favorite.is-fav {
  border-color: color-mix(in oklch, var(--color-error) 40%, var(--color-outline-dim));
}

.icon-btn-round .material-symbols-outlined {
  font-size: 16px;
}

.icon-btn-round .material-symbols-outlined.filled {
  color: var(--color-error);
  font-variation-settings: 'FILL' 1;
}

.task-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--gw-text);
  margin: 0;
  line-height: 1.4;
  text-decoration: underline;
}

/* Поля */
.field-box {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.fields-row {
  display: flex;
  gap: 10px;
}

.field-box.half {
  flex: 1;
  min-width: 0;
}

.field-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--gw-primary);
  display: flex;
  align-items: center;
  gap: 4px;
}

.field-label-icon {
  font-size: 14px;
}

/* ── Теги ── */
.tm-tags-label { justify-content: space-between; }

.tm-tags-edit {
  border: none;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  display: inline-flex;
  padding: 2px;
  border-radius: var(--radius-full);
}
.tm-tags-edit:hover { color: var(--color-text); }
.tm-tags-edit .material-symbols-outlined { font-size: 16px; }

.tm-tags-value,
.tm-tags-editor {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}

.tm-tag-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border: 1px solid transparent;
  border-radius: var(--radius-full);
  padding: 4px 10px;
  font-size: 12px;
  font-weight: 600;
}
.tm-tag-icon { font-size: 13px; }

.tm-tag-pick {
  font: inherit;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  opacity: 0.55;
  transition: opacity 0.12s;
}
.tm-tag-pick.active { opacity: 1; outline: 2px solid currentColor; outline-offset: -2px; }

.field-value {
  border: 1px solid var(--gw-border);
  border-radius: 8px;
  padding: 7px 10px;
  font-size: 13px;
  color: var(--gw-text);
  background: var(--color-surface);
  min-height: 34px;
  display: flex;
  align-items: center;
}

.field-value.with-icon {
  gap: 6px;
}

.responsible-value {
  gap: 10px;
}

.responsible-avatar {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

.responsible-name {
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.text-dim {
  color: var(--gw-text-secondary);
}

.field-icon {
  font-size: 15px;
  color: var(--gw-text-secondary);
  flex-shrink: 0;
}

.field-value.overdue {
  color: var(--color-error);
  font-weight: 600;
  border-color: color-mix(in oklch, var(--color-error) 40%, var(--color-outline-dim));
}

.yougile-value {
  gap: 6px;
  overflow: hidden;
}

.yougile-url {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--gw-primary);
  font-size: 12px;
}

.action-btn-round {
  width: 26px;
  height: 26px;
  /* min-height перебивает глобальный мобильный tap-target у button/a —
     без него кнопки растягиваются в овалы. */
  min-height: 26px;
  aspect-ratio: 1;
  border-radius: 50%;
  border: 1px solid var(--gw-border);
  background: none;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--gw-text-secondary);
  text-decoration: none;
  flex-shrink: 0;
  transition: background 0.12s, color 0.12s;
}

.action-btn-round:hover {
  background: var(--gw-primary-light);
  color: var(--gw-primary);
}

.action-btn-round.danger { color: var(--color-error); border-color: var(--color-error); }
.action-btn-round.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); }

.yougile-empty {
  display: flex; gap: 10px; align-items: center; flex-wrap: wrap;
}
.btn-tonal {
  display: inline-flex; align-items: center; gap: 6px;
  height: 32px; padding: 0 14px; border-radius: 16px;
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
  border: none; cursor: pointer; font: inherit; font-weight: 600;
}
.btn-tonal:hover:not(:disabled) {
  background: color-mix(in oklch, var(--color-secondary-container) 92%, black);
}
.btn-tonal:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-tonal .material-symbols-outlined { font-size: 18px; }

.action-btn-round .material-symbols-outlined {
  font-size: 14px;
}

/* Нижние кнопки */
.task-bottom-actions {
  margin-top: auto;
  padding-top: 16px;
  border-top: 1px solid var(--gw-border);
}

.btn-full {
  width: 100%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px 20px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  border: none;
  transition: opacity 0.12s;
}

.btn-full:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-full:not(:disabled):hover {
  opacity: 0.88;
}

.btn-full .material-symbols-outlined {
  font-size: 18px;
}

.pill {
  border-radius: 999px;
}

.primary-btn {
  background: var(--gw-primary);
  color: var(--color-on-primary);
}

.accent-btn {
  background: var(--gw-accent);
  color: var(--color-on-secondary);
}

/* ─── Правая панель ─── */
.task-right {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: color-mix(in oklch, var(--color-surface-low) 55%, transparent);
  padding: 20px;
  gap: 14px;
  overflow: hidden;
}

.units-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  flex-shrink: 0;
}

.units-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--gw-text);
  margin: 0;
}

.right-tabs {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px;
  background: var(--color-surface-high);
  border-radius: var(--radius-full, 999px);
}
.right-tab {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 14px;
  background: transparent;
  border: none;
  border-radius: var(--radius-full, 999px);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-on-surface-variant);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.right-tab:hover { color: var(--color-on-surface); }
.right-tab.active {
  background: var(--color-primary);
  color: var(--color-on-primary);
}
.right-tab .material-symbols-outlined { font-size: 16px; }

.w-full { width: 100%; }

.units-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.btn-start-unit {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--gw-primary);
  color: var(--color-on-primary);
  border: none;
  padding: 8px 16px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  border-radius: var(--radius-full);
  transition: background 0.12s;
}

.btn-start-unit:hover {
  background: var(--gw-primary-hover);
}

.btn-start-unit .material-symbols-outlined {
  font-size: 16px;
}

.btn-close-round {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  border: 1px solid var(--gw-border);
  background: var(--color-surface);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--gw-text-secondary);
  transition: background 0.12s, color 0.12s;
  flex-shrink: 0;
}

.btn-close-round:hover {
  background: var(--gw-primary-light);
  color: var(--gw-primary);
}

.btn-close-round .material-symbols-outlined {
  font-size: 20px;
}

.units-loading {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--gw-text-secondary);
  font-size: 13px;
  padding: 20px 0;
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.units-list {
  display: flex;
  flex-direction: column;
  overflow-y: auto;
  flex: 1;
  min-height: 0;
  padding-right: 4px;
  margin-right: -4px;
}

.no-units {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 32px 0;
  color: var(--gw-text-secondary);
  font-size: 13px;
}

.no-units .material-symbols-outlined {
  font-size: 36px;
  opacity: 0.4;
}

/* Confirm dialog */
.confirm-text {
  font-size: 14px;
  color: var(--gw-text);
  margin: 0;
  line-height: 1.5;
}

.btn-secondary {
  background: transparent;
  border: 1px solid var(--gw-border);
  border-radius: 8px;
  padding: 8px 18px;
  font-size: 14px;
  color: var(--gw-text);
  cursor: pointer;
}

.btn-secondary:hover {
  background: var(--gw-bg);
}

.btn-danger {
  background: var(--color-error);
  border: none;
  border-radius: 8px;
  padding: 8px 18px;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-on-error);
  cursor: pointer;
  transition: background 0.12s;
}

.btn-danger:hover:not(:disabled) {
  background: var(--color-error-hover);
}

.btn-danger:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* ═══════════════ Мобильный layout (M3) ═══════════════ */
.mobile-layout {
  flex-direction: column;
  min-height: unset;
  max-height: unset;
  height: 100%;
}

.mobile-layout .task-left {
  width: 100%;
  border-right: none;
  border-bottom: none;
  overflow-y: auto;
  flex: 1;
  min-height: 0;
  padding: 16px 16px 24px;
  gap: 14px;
}

.mobile-layout .task-right {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  padding: 12px 16px 16px;
  gap: 10px;
}

.hidden {
  display: none !important;
}

/* ── Top App Bar (M3) ── */
.mobile-topbar {
  position: relative;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 4px 6px 4px;
  padding-top: calc(6px + env(safe-area-inset-top, 0px));
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border-bottom: 1px solid var(--color-outline-dim);
  flex-shrink: 0;
  min-height: 56px;
}

.topbar-icon-btn {
  width: 44px;
  height: 44px;
  flex-shrink: 0;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
  display: grid;
  place-items: center;
  transition: background 0.15s, color 0.15s;
}

.topbar-icon-btn:active {
  background: color-mix(in oklch, var(--color-primary) 16%, transparent);
}

.topbar-icon-btn.active {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.topbar-icon-btn .material-symbols-outlined {
  font-size: 24px;
}

.topbar-icon-btn.is-fav {
  color: var(--color-error);
}

.topbar-icon-btn .material-symbols-outlined.filled {
  font-variation-settings: 'FILL' 1;
}

.topbar-title-wrap {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0;
  padding: 0 2px;
  line-height: 1.15;
}

.topbar-eyebrow {
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text-dim);
  text-transform: uppercase;
  letter-spacing: 0.4px;
  font-variant-numeric: tabular-nums;
}

.topbar-title {
  font-size: 15px;
  font-weight: 650;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

/* стиль .mobile-menu вынесен в global <style> (Teleport уносит его в body,
   scoped-стили туда не доедут) */

/* ── Tabs (M3 secondary tabs) ── */
.mobile-tabs {
  display: flex;
  align-items: stretch;
  gap: 0;
  padding: 0;
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border-bottom: 1px solid var(--color-outline-dim);
  flex-shrink: 0;
}

.mobile-tab-btn {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 10px 8px 11px;
  border: none;
  background: transparent;
  color: var(--color-text-dim);
  font-size: 12.5px;
  font-weight: 600;
  cursor: pointer;
  position: relative;
  min-height: 56px;
  transition: color 0.18s;
}

.mobile-tab-btn:active {
  background: color-mix(in oklch, var(--color-primary) 10%, transparent);
}

.mobile-tab-btn.active {
  color: var(--color-primary);
}

/* M3 indicator: подчёркивающая полоска снизу под активной вкладкой. */
.mobile-tab-btn.active::after {
  content: '';
  position: absolute;
  left: 50%;
  bottom: 0;
  transform: translateX(-50%);
  width: 60%;
  height: 3px;
  background: var(--color-primary);
  border-radius: 3px 3px 0 0;
}

.mobile-tab-btn .material-symbols-outlined {
  font-size: 22px;
}

/* ── Sticky bottom action bar ── */
.mobile-bottom-bar {
  flex-shrink: 0;
  padding: 12px 16px calc(12px + env(safe-area-inset-bottom, 0px));
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border-top: 1px solid var(--color-outline-dim);
  display: flex;
  gap: 8px;
}

.bottom-action-btn {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 14px 18px;
  border: none;
  border-radius: var(--radius-full);
  font-size: 15px;
  font-weight: 650;
  cursor: pointer;
  min-height: 52px;
  transition: background 0.15s, transform 0.1s, opacity 0.15s;
}

.bottom-action-btn:active {
  transform: scale(0.98);
}

.bottom-action-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.bottom-action-btn .material-symbols-outlined {
  font-size: 22px;
}

.bottom-action-btn.tone-primary {
  background: var(--color-primary);
  color: var(--color-on-primary);
  box-shadow: var(--shadow-sm);
}

.bottom-action-btn.tone-tertiary {
  background: var(--color-tertiary);
  color: var(--color-on-tertiary);
}

.bottom-action-btn.tone-secondary {
  background: var(--color-secondary);
  color: var(--color-on-secondary);
}

/* ── Подкрутка полей в мобильном (более плотный M3-вид) ── */
.mobile-layout .field-label {
  font-size: 12px;
  color: var(--color-text-dim);
  text-transform: uppercase;
  letter-spacing: 0.4px;
  font-weight: 700;
}

.mobile-layout .field-value {
  font-size: 14px;
  padding: 10px 12px;
  min-height: 44px;
  border-radius: var(--radius-md);
  background: var(--color-surface-high);
  border-color: transparent;
}

.mobile-layout .fields-row {
  flex-direction: column;
  gap: 14px;
}

.mobile-layout .field-box.half {
  width: 100%;
}

.mobile-layout :deep(.p-select) {
  background: var(--color-surface-high);
  border-color: transparent;
}

.mobile-layout :deep(.p-select-label) {
  padding: 12px 12px;
  font-size: 14px;
}
</style>

<!-- Global (un-scoped) — нужно, чтобы стили доехали до Teleport.to=body. -->
<style>
.mobile-menu {
  position: fixed;
  top: calc(56px + env(safe-area-inset-top, 0px) + 6px);
  right: 8px;
  z-index: 10001;
  min-width: 220px;
  padding: 6px;
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border-radius: var(--radius-lg, 16px);
  box-shadow: var(--shadow-lg, 0 12px 32px rgba(0, 0, 0, 0.18));
  border: 1px solid var(--color-outline-dim);
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.mobile-menu-backdrop {
  position: fixed;
  inset: 0;
  z-index: 10000;
  background: transparent;
}

.mobile-menu .mm-item {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 14px;
  border: none;
  border-radius: var(--radius-md, 12px);
  background: transparent;
  color: var(--color-text);
  font: inherit;
  font-size: 14.5px;
  font-weight: 500;
  cursor: pointer;
  text-align: left;
  min-height: 48px;
  transition: background 0.12s;
}

.mobile-menu .mm-item:active {
  background: color-mix(in oklch, var(--color-primary) 14%, transparent);
}

.mobile-menu .mm-item.danger {
  color: var(--color-error);
}

.mobile-menu .mm-item.danger:active {
  background: color-mix(in oklch, var(--color-error) 14%, transparent);
}

.mobile-menu .mm-item .material-symbols-outlined {
  font-size: 22px;
}

.mobile-menu-enter-active,
.mobile-menu-leave-active {
  transition: opacity 0.16s ease, transform 0.18s cubic-bezier(0.4, 0, 0.2, 1);
  transform-origin: top right;
}

.mobile-menu-enter-from,
.mobile-menu-leave-to {
  opacity: 0;
  transform: scale(0.95) translateY(-4px);
}
</style>
