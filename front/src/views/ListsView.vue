<template>
  <div class="admin-page">
    <header class="admin-sticky">
      <div class="page-head">
        <div class="page-head-text">
          <h1 class="page-head-title">Списки</h1>
          <div class="page-head-meta">
            <span class="meta-stat">
              <span class="material-symbols-outlined">apartment</span>
              <strong>{{ departments.length }}</strong> отд.
            </span>
            <span class="meta-stat">
              <span class="material-symbols-outlined">category</span>
              <strong>{{ unitTypes.length }}</strong> типов
            </span>
            <span class="meta-stat">
              <span class="material-symbols-outlined">flag</span>
              <strong>{{ stages.length }}</strong> этапов
            </span>
          </div>
        </div>
        <button
          v-if="canEdit && effectiveCompanyId != null"
          class="btn-filled desktop-only"
          @click="addItem(tab)"
        >
          <span class="material-symbols-outlined">add</span>
          <span>Добавить</span>
        </button>
      </div>

      <div class="admin-toolbar">
        <SegmentedTabs
          v-model="tab"
          :tabs="tabsForUi"
          full-width
        />
      </div>
    </header>

    <div ref="bodyRef" class="admin-body">
      <!-- Системный администратор без выбранной компании — placeholder. -->
      <div
        v-if="!effectiveCompanyId && auth.isRootAdmin"
        class="placeholder"
      >
        <div class="placeholder-icon">
          <span class="material-symbols-outlined">domain</span>
        </div>
        <h3>Выберите компанию</h3>
        <p>
          Списки ведутся внутри компании. Откройте «Сменить компанию»
          в боковом меню, чтобы продолжить.
        </p>
      </div>

      <template v-else>
        <!-- ===== Мобильное представление: компактные строки ===== -->
        <div v-if="isMobile" class="pane">
          <div class="pane-hint">
            <span class="material-symbols-outlined">info</span>
            {{ mobileHint }}
          </div>

          <div v-if="!currentItems.length" class="m-empty">
            <div class="empty-icon-circle">
              <span class="material-symbols-outlined">{{ mobileEmptyIcon }}</span>
            </div>
            <h3>{{ mobileEmptyTitle }}</h3>
          </div>

          <ul v-else class="m-list">
            <li
              v-for="item in currentItems"
              :key="item.id"
              class="m-row"
              :class="{ editing: isEditingMobile(item) }"
            >
              <span v-if="tab !== 'stages'" class="row-ico" data-tone="primary">
                <span class="material-symbols-outlined">{{ tabIcon }}</span>
              </span>
              <span
                v-else
                :class="['stage-chip', `tag-${stageDisplayColor(item)}`]"
              />

              <div class="m-row-body">
                <template v-if="isEditingMobile(item)">
                  <input
                    :ref="bindEditInput"
                    :value="rowEditValue(rowPrefix, item)"
                    class="row-input"
                    :placeholder="mobilePlaceholder"
                    @click.stop
                    @input="setRowEditValue($event.target.value)"
                    @keyup.enter="commitMobile(item)"
                    @keyup.escape="cancelRow"
                  />
                  <div v-if="tab === 'stages'" class="color-dots" @click.stop>
                    <button
                      v-for="c in STAGE_COLORS"
                      :key="c"
                      type="button"
                      :class="['color-dot', `tag-${c}`, { selected: editColor === c }]"
                      :title="c"
                      @click.stop="editColor = c"
                    />
                  </div>
                </template>
                <template v-else>
                  <span class="row-name">{{ item.name }}</span>
                  <div v-if="tab === 'stages'" class="m-row-meta">
                    <span :class="['mini-tag', `tag-${item.color}`]">{{ item.color }}</span>
                    <span class="order-badge">#{{ item.order }}</span>
                  </div>
                </template>
              </div>

              <div class="m-row-actions" @click.stop>
                <template v-if="isEditingMobile(item)">
                  <button class="icon-btn primary" :title="item.__draft ? 'Создать' : 'Сохранить'" @click="commitMobile(item)">
                    <span class="material-symbols-outlined">check</span>
                  </button>
                  <button class="icon-btn" title="Отмена" @click="cancelRow">
                    <span class="material-symbols-outlined">close</span>
                  </button>
                </template>
                <template v-else-if="canEdit">
                  <button class="icon-btn" title="Редактировать" @click="startEditMobile(item)">
                    <span class="material-symbols-outlined">edit</span>
                  </button>
                  <button class="icon-btn danger" :title="`Удалить`" @click="askDelete(tab, item)">
                    <span class="material-symbols-outlined">delete</span>
                  </button>
                </template>
              </div>
            </li>
          </ul>
        </div>

        <div v-else-if="tab === 'departments'" class="pane">
          <div class="pane-hint">
            <span class="material-symbols-outlined">info</span>
            Группировка сотрудников и сегмент в статистике.
          </div>

          <AppDataTable
            :value="displayDepartments"
            empty-message="Отделов пока нет"
          >
            <Column header="" style="width: 60px">
              <template #body>
                <span class="row-ico" data-tone="primary">
                  <span class="material-symbols-outlined">apartment</span>
                </span>
              </template>
            </Column>
            <Column header="Название">
              <template #body="{ data }">
                <template v-if="isEditingRow('d', data)">
                  <input
                    :ref="bindEditInput"
                    :value="rowEditValue('d', data)"
                    class="row-input"
                    placeholder="Название отдела"
                    @click.stop
                    @input="setRowEditValue($event.target.value)"
                    @keyup.enter="commitRowEdit('departments', data)"
                    @keyup.escape="cancelRow"
                  />
                </template>
                <span v-else class="row-name">{{ data.name }}</span>
              </template>
            </Column>
            <Column header="" style="width: 220px" body-style="text-align: right">
              <template #body="{ data }">
                <div class="row-actions" @click.stop>
                  <template v-if="isEditingRow('d', data)">
                    <button class="pill-btn primary" @click="commitRowEdit('departments', data)">
                      <span class="material-symbols-outlined">check</span>
                      {{ data.__draft ? 'Создать' : 'Сохранить' }}
                    </button>
                    <button class="icon-btn" title="Отмена" @click="cancelRow">
                      <span class="material-symbols-outlined">close</span>
                    </button>
                  </template>
                  <template v-else-if="canEdit">
                    <button class="icon-btn" title="Редактировать" @click="startEdit('d', data)">
                      <span class="material-symbols-outlined">edit</span>
                    </button>
                    <button class="icon-btn danger" title="Удалить" @click="askDelete('departments', data)">
                      <span class="material-symbols-outlined">delete</span>
                    </button>
                  </template>
                </div>
              </template>
            </Column>
          </AppDataTable>
        </div>

        <div v-else-if="tab === 'unit-types'" class="pane">
          <div class="pane-hint">
            <span class="material-symbols-outlined">info</span>
            Категории работы — встреча, дизайн, написание кода и т. п.
          </div>

          <AppDataTable
            :value="displayUnitTypes"
            empty-message="Типов юнитов пока нет"
          >
            <Column header="" style="width: 60px">
              <template #body>
                <span class="row-ico" data-tone="primary">
                  <span class="material-symbols-outlined">category</span>
                </span>
              </template>
            </Column>
            <Column header="Название">
              <template #body="{ data }">
                <template v-if="isEditingRow('u', data)">
                  <input
                    :ref="bindEditInput"
                    :value="rowEditValue('u', data)"
                    class="row-input"
                    placeholder="Название типа"
                    @click.stop
                    @input="setRowEditValue($event.target.value)"
                    @keyup.enter="commitRowEdit('unit-types', data)"
                    @keyup.escape="cancelRow"
                  />
                </template>
                <span v-else class="row-name">{{ data.name }}</span>
              </template>
            </Column>
            <Column header="" style="width: 220px" body-style="text-align: right">
              <template #body="{ data }">
                <div class="row-actions" @click.stop>
                  <template v-if="isEditingRow('u', data)">
                    <button class="pill-btn primary" @click="commitRowEdit('unit-types', data)">
                      <span class="material-symbols-outlined">check</span>
                      {{ data.__draft ? 'Создать' : 'Сохранить' }}
                    </button>
                    <button class="icon-btn" title="Отмена" @click="cancelRow">
                      <span class="material-symbols-outlined">close</span>
                    </button>
                  </template>
                  <template v-else-if="canEdit">
                    <button class="icon-btn" title="Редактировать" @click="startEdit('u', data)">
                      <span class="material-symbols-outlined">edit</span>
                    </button>
                    <button class="icon-btn danger" title="Удалить" @click="askDelete('unit-types', data)">
                      <span class="material-symbols-outlined">delete</span>
                    </button>
                  </template>
                </div>
              </template>
            </Column>
          </AppDataTable>
        </div>

        <div v-else class="pane">
          <div class="pane-hint">
            <span class="material-symbols-outlined">info</span>
            Колонки канбан-режима задач. Порядок — drag-and-drop.
          </div>

          <AppDataTable
            :value="displayStages"
            empty-message="Этапов пока нет"
            @row-reorder="onStageReorder"
          >
            <Column row-reorder :reorderable-column="false" header="" style="width: 36px" />
            <Column header="" style="width: 50px">
              <template #body="{ data }">
                <span :class="['stage-chip', `tag-${stageDisplayColor(data)}`]" />
              </template>
            </Column>
            <Column header="Этап">
              <template #body="{ data }">
                <template v-if="isEditingRow('s', data)">
                  <input
                    :ref="bindEditInput"
                    :value="rowEditValue('s', data)"
                    class="row-input"
                    placeholder="Название этапа"
                    @click.stop
                    @input="setRowEditValue($event.target.value)"
                    @keyup.enter="commitStageEdit(data)"
                    @keyup.escape="cancelRow"
                  />
                </template>
                <span v-else class="row-name">{{ data.name }}</span>
              </template>
            </Column>
            <Column header="Цвет" style="width: 260px">
              <template #body="{ data }">
                <div
                  v-if="isEditingRow('s', data)"
                  class="color-dots"
                  @click.stop
                  @mousedown.stop
                >
                  <button
                    v-for="c in STAGE_COLORS"
                    :key="c"
                    type="button"
                    :class="['color-dot', `tag-${c}`, { selected: editColor === c }]"
                    :title="c"
                    draggable="false"
                    @dragstart.prevent.stop
                    @mousedown.stop
                    @click.stop="editColor = c"
                  />
                </div>
                <span v-else :class="['mini-tag', `tag-${data.color}`]">{{ data.color }}</span>
              </template>
            </Column>
            <Column header="Порядок" style="width: 100px">
              <template #body="{ data }">
                <span v-if="!data.__draft" class="order-badge">#{{ data.order }}</span>
              </template>
            </Column>
            <Column header="" style="width: 220px" body-style="text-align: right">
              <template #body="{ data }">
                <div class="row-actions" @click.stop>
                  <template v-if="isEditingRow('s', data)">
                    <button class="pill-btn primary" @click="commitStageEdit(data)">
                      <span class="material-symbols-outlined">check</span>
                      {{ data.__draft ? 'Создать' : 'Сохранить' }}
                    </button>
                    <button class="icon-btn" title="Отмена" @click="cancelRow">
                      <span class="material-symbols-outlined">close</span>
                    </button>
                  </template>
                  <template v-else-if="canEdit">
                    <button class="icon-btn" title="Редактировать" @click="startEdit('s', data)">
                      <span class="material-symbols-outlined">edit</span>
                    </button>
                    <button class="icon-btn danger" title="Удалить" @click="askDelete('stages', data)">
                      <span class="material-symbols-outlined">delete</span>
                    </button>
                  </template>
                </div>
              </template>
            </Column>
          </AppDataTable>
        </div>
      </template>
    </div>

    <ConfirmDialog
      :visible="deleteDlg.open"
      :header="`Удалить «${deleteDlg.item?.name}»?`"
      :message="deleteMessage"
      confirm-label="Удалить"
      danger-confirm
      @confirm="doDelete"
      @cancel="deleteDlg.open = false"
    />

    <AppFab
      :visible="canEdit && effectiveCompanyId != null"
      icon="add"
      label="Добавить"
      :collapsed="isCompact"
      :aria-label="`Добавить в ${tab}`"
      @click="addItem(tab)"
    />
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted } from 'vue'
import Column from 'primevue/column'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import AppDataTable from '@/components/common/AppDataTable.vue'
import AppFab from '@/components/common/AppFab.vue'
import SegmentedTabs from '@/components/common/SegmentedTabs.vue'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { useScrollCollapse } from '@/composables/useScrollCollapse.js'

const { isMobile } = useBreakpoint()
const bodyRef = ref(null)
const { isCompact } = useScrollCollapse(bodyRef)
import { useAuthStore } from '@/stores/auth.js'
import { useCompaniesStore } from '@/stores/companies.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { usePermission } from '@/composables/usePermission.js'
import {
  getDepartments, createDepartment, updateDepartment, deleteDepartment,
} from '@/api/departments.js'
import {
  getUnitTypes, createUnitType, updateUnitType, deleteUnitType,
} from '@/api/unitTypes.js'
import {
  getStages, createStage, updateStage, deleteStage, reorderStages, STAGE_COLORS,
} from '@/api/stages.js'

// STAGE_COLORS используется в шаблоне (auto-imported в setup-биндинги).

const auth = useAuthStore()
const companies = useCompaniesStore()
const notif = useNotificationsStore()
const { isAtLeast, ROLES } = usePermission()

const canEdit = computed(() => isAtLeast(ROLES.MANAGER))

const tabs = [
  { key: 'departments', label: 'Отделы', icon: 'apartment' },
  { key: 'unit-types', label: 'Типы юнитов', icon: 'category' },
  { key: 'stages', label: 'Этапы', icon: 'flag' },
]
const tab = ref('departments')

const tabsForUi = computed(() => tabs.map(t => ({
  value: t.key,
  label: t.label,
  icon: t.icon,
  badge: counts.value[t.key] || null,
})))

/* ---------- helpers для мобильного представления ---------- */
const rowPrefix = computed(() =>
  tab.value === 'departments' ? 'd' : tab.value === 'unit-types' ? 'u' : 's'
)

const currentItems = computed(() => {
  if (tab.value === 'departments') return displayDepartments.value
  if (tab.value === 'unit-types') return displayUnitTypes.value
  return displayStages.value
})

const tabIcon = computed(() => {
  if (tab.value === 'departments') return 'apartment'
  if (tab.value === 'unit-types') return 'category'
  return 'flag'
})

const mobileHint = computed(() => {
  if (tab.value === 'departments') return 'Группировка сотрудников и сегмент в статистике.'
  if (tab.value === 'unit-types') return 'Категории работы — встреча, дизайн, написание кода и т. п.'
  return 'Колонки канбан-режима задач.'
})

const mobilePlaceholder = computed(() => {
  if (tab.value === 'departments') return 'Название отдела'
  if (tab.value === 'unit-types') return 'Название типа'
  return 'Название этапа'
})

const mobileEmptyTitle = computed(() => {
  if (tab.value === 'departments') return 'Отделов пока нет'
  if (tab.value === 'unit-types') return 'Типов юнитов пока нет'
  return 'Этапов пока нет'
})

const mobileEmptyIcon = computed(() => tabIcon.value)

function isEditingMobile(item) {
  return isEditingRow(rowPrefix.value, item)
}

function startEditMobile(item) {
  startEdit(rowPrefix.value, item)
}

function commitMobile(item) {
  if (tab.value === 'stages') commitStageEdit(item)
  else commitRowEdit(tab.value, item)
}

const departments = ref([])
const unitTypes = ref([])
const stages = ref([])

const counts = computed(() => ({
  'departments': departments.value.length,
  'unit-types': unitTypes.value.length,
  'stages': stages.value.length,
}))

const effectiveCompanyId = computed(() => companies.effectiveCompanyId)

/* ---------- state редактирования / добавления ----------
   Создание и редактирование делается прямо в строке таблицы.
   Для создания добавляется временный объект {__draft: true} в начало списка. */
const editing = ref(null) // 'd-<id>' | 'u-<id>' | 's-<id>' | 'd-draft' | 'u-draft' | 's-draft'
const editName = ref('')
const editColor = ref('blue')
const draftActive = ref(null) // 'departments' | 'unit-types' | 'stages' | null

function isEditingRow(prefix, data) {
  if (data.__draft) return editing.value === `${prefix}-draft`
  return editing.value === `${prefix}-${data.id}`
}
function rowEditValue(prefix, data) {
  return editing.value === `${prefix}-${data.__draft ? 'draft' : data.id}` ? editName.value : data.name
}
function setRowEditValue(v) { editName.value = v }

function bindEditInput(el) {
  if (el && el.focus) nextTick(() => el.focus())
}

function stageDisplayColor(s) {
  return isEditingRow('s', s) ? editColor.value : s.color
}

const draftItem = computed(() => {
  if (!draftActive.value) return null
  if (draftActive.value === 'stages') {
    return { id: '__draft__', __draft: true, name: '', color: 'blue', order: 0 }
  }
  return { id: '__draft__', __draft: true, name: '' }
})

const displayDepartments = computed(() => {
  if (draftActive.value === 'departments') return [draftItem.value, ...departments.value]
  return departments.value
})
const displayUnitTypes = computed(() => {
  if (draftActive.value === 'unit-types') return [draftItem.value, ...unitTypes.value]
  return unitTypes.value
})
const displayStages = computed(() => {
  if (draftActive.value === 'stages') return [draftItem.value, ...stages.value]
  return stages.value
})

watch(tab, () => {
  cancelRow()
})

/* ---------- delete dialog ---------- */
const deleteDlg = ref({ open: false, kind: null, item: null })
const deleteMessage = computed(() => {
  const k = deleteDlg.value.kind
  if (k === 'departments') return 'Отдел и его привязки к задачам будут удалены.'
  if (k === 'unit-types') return 'ВНИМАНИЕ: удаление каскадно удалит ВСЕ юниты этого типа во всей компании.'
  if (k === 'stages') return 'Этап будет удалён. Карточки задач, прикреплённые к нему, останутся без этапа.'
  return ''
})

onMounted(loadAll)
watch(effectiveCompanyId, loadAll)

async function loadAll() {
  if (effectiveCompanyId.value == null) {
    departments.value = []; unitTypes.value = []; stages.value = []
    return
  }
  try {
    const [d, u, s] = await Promise.all([getDepartments(), getUnitTypes(), getStages()])
    departments.value = d || []
    unitTypes.value = u || []
    stages.value = s || []
  } catch (e) {
    notif.error(e?.message || 'Не удалось загрузить списки')
  }
}

function addItem(kind) {
  cancelRow()
  draftActive.value = kind
  editName.value = ''
  editColor.value = 'blue'
  const prefix = kind === 'departments' ? 'd' : kind === 'unit-types' ? 'u' : 's'
  editing.value = `${prefix}-draft`
}

function cancelRow() {
  editing.value = null
  editName.value = ''
  draftActive.value = null
}

function startEdit(prefix, item) {
  draftActive.value = null
  editing.value = `${prefix}-${item.id}`
  editName.value = item.name
  if (prefix === 's') editColor.value = item.color
}

async function commitRowEdit(kind, item) {
  const name = editName.value.trim()
  if (!name) return
  try {
    if (item.__draft) {
      // create
      if (kind === 'departments') {
        const d = await createDepartment({ name })
        departments.value.push(d)
      } else {
        const u = await createUnitType({ name })
        unitTypes.value.push(u)
      }
      notif.success('Добавлено')
    } else {
      // update
      if (name === item.name) { cancelRow(); return }
      if (kind === 'departments') {
        const upd = await updateDepartment(item.id, { name })
        _replace(departments.value, upd)
      } else {
        const upd = await updateUnitType(item.id, { name })
        _replace(unitTypes.value, upd)
      }
      notif.success('Сохранено')
    }
    cancelRow()
  } catch (e) {
    notif.error(e?.message || 'Не удалось сохранить')
  }
}

async function commitStageEdit(item) {
  const name = editName.value.trim()
  if (!name) return
  try {
    if (item.__draft) {
      const s = await createStage({ name, color: editColor.value })
      stages.value.push(s)
      notif.success('Добавлено')
    } else {
      const upd = await updateStage(item.id, { name, color: editColor.value })
      _replace(stages.value, upd)
      notif.success('Сохранено')
    }
    cancelRow()
  } catch (e) {
    notif.error(e?.message || 'Не удалось сохранить')
  }
}

function _replace(arr, item) {
  const idx = arr.findIndex(x => x.id === item.id)
  if (idx >= 0) arr.splice(idx, 1, item)
}

function askDelete(kind, item) {
  deleteDlg.value = { open: true, kind, item }
}

async function doDelete() {
  const { kind, item } = deleteDlg.value
  if (!item) return
  try {
    if (kind === 'departments') {
      await deleteDepartment(item.id)
      departments.value = departments.value.filter(x => x.id !== item.id)
    } else if (kind === 'unit-types') {
      await deleteUnitType(item.id)
      unitTypes.value = unitTypes.value.filter(x => x.id !== item.id)
    } else if (kind === 'stages') {
      await deleteStage(item.id)
      stages.value = stages.value.filter(x => x.id !== item.id)
    }
    notif.success('Удалено')
    deleteDlg.value.open = false
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить')
  }
}

/* ---------- DnD для этапов (через PrimeVue row-reorder) ---------- */
async function onStageReorder(e) {
  // При активном draft событие не корректно — порядок включает draft.
  if (draftActive.value === 'stages') return
  const reordered = e.value
  stages.value = reordered
  try {
    const upd = await reorderStages(reordered.map(s => s.id))
    stages.value = upd
  } catch (err) {
    notif.error(err?.message || 'Не удалось применить порядок')
    loadAll()
  }
}

</script>

<style scoped>
/* ============ Шапка ============ */
.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}
.page-head-text { min-width: 0; }
.page-head-title {
  margin: 0 0 6px;
  font-size: 24px;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--color-text);
}
.page-head-meta {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 13px;
  color: var(--color-text-dim);
}
.page-head-meta .meta-stat {
  background: var(--color-surface-high);
  color: var(--color-text);
}

/* ============ Placeholder ============ */
.placeholder {
  padding: 56px 20px;
  text-align: center;
  background: var(--color-surface-high);
  border-radius: var(--radius-xl);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}
.placeholder-icon {
  width: 84px;
  height: 84px;
  border-radius: var(--radius-xl);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  display: grid;
  place-items: center;
}
.placeholder-icon .material-symbols-outlined { font-size: 40px; }
.placeholder h3 { margin: 0; color: var(--color-text); font-size: 18px; font-weight: 700; }
.placeholder p { margin: 0; color: var(--color-text-dim); font-size: 14px; max-width: 420px; }

/* ============ Pane ============ */
.pane {
  display: flex;
  flex-direction: column;
  gap: 14px;
  height: 100%;
  min-height: 0;
}

.pane-hint {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-radius: var(--radius-full);
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
  font-size: 12.5px;
  font-weight: 500;
  align-self: flex-start;
  max-width: 100%;
}
.pane-hint .material-symbols-outlined { font-size: 16px; opacity: 0.8; }

/* ============ Ячейки таблицы ============ */
.row-ico {
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md);
  display: grid;
  place-items: center;
  flex-shrink: 0;
}
.row-ico[data-tone="primary"] {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}
.row-ico .material-symbols-outlined { font-size: 20px; }

.row-name {
  font-weight: 600;
  color: var(--color-text);
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.row-input {
  border: none;
  outline: none;
  background: var(--color-surface);
  font: inherit;
  font-weight: 600;
  font-size: 14px;
  color: var(--color-text);
  padding: 8px 14px;
  border-radius: var(--radius-full);
  min-width: 0;
  width: 100%;
  box-shadow: inset 0 0 0 2px var(--color-primary);
}

.row-actions {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  justify-content: flex-end;
}

/* ============ Кнопки ============ */
.pill-btn {
  appearance: none;
  border: none;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: var(--radius-full);
  font: inherit;
  font-size: 12.5px;
  font-weight: 600;
  transition: background .14s, color .14s, box-shadow .14s;
}
.pill-btn.primary {
  background: var(--color-primary);
  color: var(--color-on-primary);
}
.pill-btn.primary:hover {
  background: var(--color-primary-hover);
  box-shadow: var(--shadow-sm);
}
.pill-btn .material-symbols-outlined { font-size: 16px; }

.icon-btn {
  appearance: none;
  border: none;
  background: transparent;
  width: 34px;
  height: 34px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  color: var(--color-text-dim);
  cursor: pointer;
  transition: background .14s, color .14s;
}
.icon-btn:hover {
  background: var(--color-surface-high);
  color: var(--color-text);
}
.icon-btn.danger:hover {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}
.icon-btn .material-symbols-outlined { font-size: 18px; }

.btn-filled {
  appearance: none;
  border: none;
  cursor: pointer;
  background: var(--color-primary);
  color: var(--color-on-primary);
  border-radius: var(--radius-full);
  padding: 10px 18px;
  font: inherit;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  box-shadow: var(--shadow-sm);
  transition: background .14s, box-shadow .14s;
}
.btn-filled:hover { background: var(--color-primary-hover); }
.btn-filled .material-symbols-outlined { font-size: 18px; }

/* ============ Этапы: chip + порядок + цвет ============ */
.stage-chip {
  width: 24px;
  height: 24px;
  border-radius: var(--radius-full);
  background: var(--tag-surface);
  border: 2.5px solid var(--tag-accent, var(--tag-border));
  flex-shrink: 0;
  display: inline-block;
}
.stage-chip.tag-red    { --tag-surface: var(--tag-red-surface);    --tag-border: var(--tag-red-border);    --tag-accent: var(--tag-red-accent); }
.stage-chip.tag-orange { --tag-surface: var(--tag-orange-surface); --tag-border: var(--tag-orange-border); --tag-accent: var(--tag-orange-accent); }
.stage-chip.tag-amber  { --tag-surface: var(--tag-amber-surface);  --tag-border: var(--tag-amber-border);  --tag-accent: var(--tag-amber-accent); }
.stage-chip.tag-green  { --tag-surface: var(--tag-green-surface);  --tag-border: var(--tag-green-border);  --tag-accent: var(--tag-green-accent); }
.stage-chip.tag-teal   { --tag-surface: var(--tag-teal-surface);   --tag-border: var(--tag-teal-border);   --tag-accent: var(--tag-teal-accent); }
.stage-chip.tag-blue   { --tag-surface: var(--tag-blue-surface);   --tag-border: var(--tag-blue-border);   --tag-accent: var(--tag-blue-accent); }
.stage-chip.tag-violet { --tag-surface: var(--tag-violet-surface); --tag-border: var(--tag-violet-border); --tag-accent: var(--tag-violet-accent); }
.stage-chip.tag-pink   { --tag-surface: var(--tag-pink-surface);   --tag-border: var(--tag-pink-border);   --tag-accent: var(--tag-pink-accent); }

.mini-tag {
  padding: 3px 12px;
  border-radius: var(--radius-full);
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  background: var(--tag-surface);
  color: var(--tag-accent);
  border: 1px solid var(--tag-border);
}
.mini-tag.tag-red    { --tag-surface: var(--tag-red-surface);    --tag-border: var(--tag-red-border);    --tag-accent: var(--tag-red-accent); }
.mini-tag.tag-orange { --tag-surface: var(--tag-orange-surface); --tag-border: var(--tag-orange-border); --tag-accent: var(--tag-orange-accent); }
.mini-tag.tag-amber  { --tag-surface: var(--tag-amber-surface);  --tag-border: var(--tag-amber-border);  --tag-accent: var(--tag-amber-accent); }
.mini-tag.tag-green  { --tag-surface: var(--tag-green-surface);  --tag-border: var(--tag-green-border);  --tag-accent: var(--tag-green-accent); }
.mini-tag.tag-teal   { --tag-surface: var(--tag-teal-surface);   --tag-border: var(--tag-teal-border);   --tag-accent: var(--tag-teal-accent); }
.mini-tag.tag-blue   { --tag-surface: var(--tag-blue-surface);   --tag-border: var(--tag-blue-border);   --tag-accent: var(--tag-blue-accent); }
.mini-tag.tag-violet { --tag-surface: var(--tag-violet-surface); --tag-border: var(--tag-violet-border); --tag-accent: var(--tag-violet-accent); }
.mini-tag.tag-pink   { --tag-surface: var(--tag-pink-surface);   --tag-border: var(--tag-pink-border);   --tag-accent: var(--tag-pink-accent); }

.order-badge {
  font-size: 12px;
  font-weight: 700;
  color: var(--color-text-dim);
  background: var(--color-surface-high);
  padding: 3px 10px;
  border-radius: var(--radius-full);
  font-variant-numeric: tabular-nums;
}

.color-dots {
  display: inline-flex;
  gap: 4px;
  flex-wrap: wrap;
}
.color-dot {
  appearance: none;
  border: 2.5px solid var(--tag-accent, var(--tag-border));
  background: var(--tag-surface);
  width: 22px;
  height: 22px;
  border-radius: 50%;
  cursor: pointer;
  transition: transform .12s, box-shadow .12s;
  padding: 0;
}
.color-dot:hover { transform: scale(1.12); }
.color-dot.selected {
  box-shadow: 0 0 0 2px var(--color-surface), 0 0 0 4px var(--color-primary);
  transform: scale(1.12);
}
.color-dot.tag-red    { --tag-surface: var(--tag-red-surface);    --tag-border: var(--tag-red-border);    --tag-accent: var(--tag-red-accent); }
.color-dot.tag-orange { --tag-surface: var(--tag-orange-surface); --tag-border: var(--tag-orange-border); --tag-accent: var(--tag-orange-accent); }
.color-dot.tag-amber  { --tag-surface: var(--tag-amber-surface);  --tag-border: var(--tag-amber-border);  --tag-accent: var(--tag-amber-accent); }
.color-dot.tag-green  { --tag-surface: var(--tag-green-surface);  --tag-border: var(--tag-green-border);  --tag-accent: var(--tag-green-accent); }
.color-dot.tag-teal   { --tag-surface: var(--tag-teal-surface);   --tag-border: var(--tag-teal-border);   --tag-accent: var(--tag-teal-accent); }
.color-dot.tag-blue   { --tag-surface: var(--tag-blue-surface);   --tag-border: var(--tag-blue-border);   --tag-accent: var(--tag-blue-accent); }
.color-dot.tag-violet { --tag-surface: var(--tag-violet-surface); --tag-border: var(--tag-violet-border); --tag-accent: var(--tag-violet-accent); }
.color-dot.tag-pink   { --tag-surface: var(--tag-pink-surface);   --tag-border: var(--tag-pink-border);   --tag-accent: var(--tag-pink-accent); }

/* ===== Мобильные списковые карточки ===== */
.m-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.m-row {
  display: flex;
  align-items: center;
  gap: 12px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg);
  padding: 12px 14px;
  min-height: 60px;
}

.m-row.editing {
  border-color: var(--color-primary);
  background: var(--color-surface-high);
}

.m-row .row-ico {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  flex-shrink: 0;
}

.m-row .stage-chip {
  width: 28px;
  height: 28px;
  flex-shrink: 0;
}

.m-row-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.m-row-body .row-name {
  white-space: normal;
  word-break: break-word;
}

.m-row-meta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.m-row-actions {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}

.m-row-actions .icon-btn {
  width: 40px;
  height: 40px;
  background: var(--color-surface-high);
}
.m-row-actions .icon-btn.primary {
  background: var(--color-primary);
  color: var(--color-on-primary);
}
.m-row-actions .icon-btn.primary:hover {
  background: var(--color-primary-hover);
}

.m-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 48px 20px;
  text-align: center;
}
.empty-icon-circle {
  width: 80px;
  height: 80px;
  border-radius: var(--radius-xl);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  display: grid;
  place-items: center;
}
.empty-icon-circle .material-symbols-outlined { font-size: 36px; }
.m-empty h3 { margin: 0; color: var(--color-text); font-size: 16px; font-weight: 700; }

@media (max-width: 768px) {
  .hide-narrow { display: none; }
  .desktop-only { display: none; }

  .page-head-title { font-size: 20px; }
  .page-head-meta { font-size: 12px; }
  .pane-hint { font-size: 12px; padding: 6px 12px; }
}
</style>
