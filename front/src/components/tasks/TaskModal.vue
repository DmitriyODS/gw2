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
      <!-- Мобильные вкладки -->
      <div v-if="isMobile" class="mobile-tabs">
        <button
          class="mobile-tab-btn"
          :class="{ active: mobileTab === 'details' }"
          @click="mobileTab = 'details'"
        >
          <span class="material-symbols-outlined">info</span>
          Детали
        </button>
        <button
          class="mobile-tab-btn"
          :class="{ active: mobileTab === 'units' }"
          @click="mobileTab = 'units'"
        >
          <span class="material-symbols-outlined">timer</span>
          Юниты
        </button>
        <button class="mobile-close-btn" @click="$emit('close')">
          <span class="material-symbols-outlined">close</span>
        </button>
      </div>

      <!-- Левая панель (детали задачи) -->
      <div class="task-left" :class="{ hidden: isMobile && mobileTab !== 'details' }">
        <div class="task-top-row">
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
            <div class="color-wrapper">
              <button
                class="icon-btn-round"
                :class="{ active: showColorPicker }"
                @click="showColorPicker = !showColorPicker"
                title="Цвет задачи"
              >
                <span class="material-symbols-outlined">palette</span>
              </button>
              <div v-if="showColorPicker" class="color-popover" @click.stop>
                <TaskColorPicker :model-value="task.color || null" @select="handleSetColor" />
              </div>
            </div>
            <button v-if="canEditTask" class="icon-btn-round" @click="showEditForm = true" title="Редактировать">
              <span class="material-symbols-outlined">edit</span>
            </button>
            <button v-if="canDeleteTask" class="icon-btn-round danger" @click="confirmDelete" title="Удалить">
              <span class="material-symbols-outlined">delete</span>
            </button>
          </div>
        </div>

        <h2 class="task-title">{{ task.name }}</h2>

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
        <div v-if="task.link_yougile" class="field-box">
          <div class="field-label">
            <span class="material-symbols-outlined field-label-icon">link</span>
            YouGile
          </div>
          <div class="field-value yougile-value">
            <span class="yougile-url">{{ task.link_yougile }}</span>
            <button class="action-btn-round" @click="copyLink" title="Скопировать ссылку">
              <span class="material-symbols-outlined">content_copy</span>
            </button>
            <a :href="task.link_yougile" target="_blank" class="action-btn-round" title="Открыть в новой вкладке">
              <span class="material-symbols-outlined">open_in_new</span>
            </a>
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

        <!-- Нижние кнопки -->
        <div class="task-bottom-actions">
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

      <!-- Правая панель (юниты) -->
      <div class="task-right" :class="{ hidden: isMobile && mobileTab !== 'units' }">
        <div class="units-header">
          <h3 class="units-title">Юниты</h3>
          <div class="units-header-actions">
            <button
              v-if="canStartUnit"
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
          />
          <div v-if="units.length === 0" class="no-units">
            <span class="material-symbols-outlined">hourglass_empty</span>
            Юнитов пока нет
          </div>
        </div>
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
    <Dialog
      v-if="confirmDialog.visible"
      :visible="confirmDialog.visible"
      @update:visible="confirmDialog.visible = false"
      modal
      :header="confirmDialog.title"
      style="width: 380px"
    >
      <p class="confirm-text">{{ confirmDialog.message }}</p>
      <template #footer>
        <button class="btn-secondary" @click="confirmDialog.visible = false">Отмена</button>
        <button
          class="btn-danger"
          @click="confirmDialog.onConfirm(); confirmDialog.visible = false"
          :disabled="actionLoading"
        >
          {{ confirmDialog.confirmLabel || 'Подтвердить' }}
        </button>
      </template>
    </Dialog>
  </Dialog>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import Dialog from 'primevue/dialog'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import UnitListItem from '@/components/tasks/UnitListItem.vue'
import TaskForm from '@/components/tasks/TaskForm.vue'
import TaskColorPicker from '@/components/tasks/TaskColorPicker.vue'
import StartUnitModal from '@/components/units/StartUnitModal.vue'
import UnitEditModal from '@/components/units/UnitEditModal.vue'
import { getUnits, deleteUnit } from '@/api/units.js'
import { deleteTask, archiveTask, restoreTask, toggleFavorite as apiFavorite, updateTask } from '@/api/tasks.js'
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
      : 'padding:0; overflow:hidden; border-radius:20px'
  }
}))

const units = ref([])
const unitsLoading = ref(false)
const showEditForm = ref(false)
const showStartUnit = ref(false)
const showColorPicker = ref(false)
const editingUnit = ref(null)
const actionLoading = ref(false)

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

onMounted(() => {
  loadUnits()
})

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

function confirmDeleteUnit(unit) {
  confirmDialog.value = {
    visible: true,
    title: 'Удалить юнит',
    message: `Вы уверены, что хотите удалить юнит "${unit.name}"?`,
    confirmLabel: 'Удалить',
    onConfirm: () => handleDeleteUnit(unit.id)
  }
}

async function handleDeleteUnit(unitId) {
  try {
    await deleteUnit(unitId)
    units.value = units.value.filter(u => u.id !== unitId)
    notifications.success('Юнит удалён')
    if (unitsStore.activeUnit?.id === unitId) {
      unitsStore.clearActiveUnit()
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
  tasksStore.patchTask({ id: props.task.id, has_units: true })
}

async function handleToggleFavorite() {
  const next = !props.task.is_favorite
  tasksStore.patchTask({ id: props.task.id, is_favorite: next })
  try {
    await apiFavorite(props.task.id)
  } catch (e) {
    tasksStore.patchTask({ id: props.task.id, is_favorite: !next })
    notifications.error(e?.message || 'Не удалось изменить избранное')
  }
}

async function handleSetColor(color) {
  showColorPicker.value = false
  const prev = props.task.color ?? null
  if (prev === color) return
  tasksStore.patchTask({ id: props.task.id, color })
  try {
    await updateTask(props.task.id, { color })
  } catch (e) {
    tasksStore.patchTask({ id: props.task.id, color: prev })
    notifications.error(e?.message || 'Не удалось изменить цвет')
  }
}
</script>

<style scoped>
.task-modal-body {
  display: flex;
  min-height: 520px;
}

/* ─── Левая панель ─── */
.task-left {
  width: 45%;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 24px 24px 24px 28px;
  background: var(--color-surface);
  border-right: 1px solid var(--gw-border);
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
  height: 30px;
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

.color-wrapper {
  position: relative;
  display: flex;
}

.color-popover {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  z-index: 60;
  background: var(--color-surface);
  border: 1px solid var(--gw-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  padding: 10px;
  width: 132px;
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
  display: flex;
  flex-direction: column;
  background: var(--color-bg);
  padding: 20px;
  gap: 14px;
}

.units-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.units-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--gw-text);
  margin: 0;
}

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

/* ── Мобильный layout ── */
.mobile-tabs {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 10px 12px;
  background: var(--color-surface);
  border-bottom: 1px solid var(--gw-border);
  flex-shrink: 0;
}

.mobile-tab-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 12px;
  border: none;
  border-radius: 8px;
  background: transparent;
  color: var(--gw-text-secondary);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.mobile-tab-btn.active {
  background: var(--gw-primary);
  color: var(--color-on-primary);
}

.mobile-tab-btn .material-symbols-outlined {
  font-size: 18px;
}

.mobile-close-btn {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: 1px solid var(--gw-border);
  background: transparent;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--gw-text-secondary);
  flex-shrink: 0;
  margin-left: auto;
  transition: background 0.12s;
}

.mobile-close-btn:active {
  background: var(--gw-bg);
}

.mobile-close-btn .material-symbols-outlined {
  font-size: 20px;
}

.mobile-layout {
  flex-direction: column;
  min-height: unset;
  height: 100%;
}

.mobile-layout .task-left {
  width: 100%;
  border-right: none;
  border-bottom: none;
  overflow-y: auto;
  flex: 1;
}

.mobile-layout .task-right {
  flex: 1;
  overflow-y: auto;
}

.hidden {
  display: none !important;
}
</style>
