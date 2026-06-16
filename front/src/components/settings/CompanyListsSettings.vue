<template>
  <div class="lists-settings">
    <!-- Списки (отделы/типы/этапы) скоупятся активной компанией сессии на
         бэке (tasksvc, requireCompanyScope из токена). Поэтому редактировать
         их можно только когда эта компания — активная (как и YouGile). -->
    <div v-if="companyId !== auth.companyId" class="note">
      Чтобы настроить списки этой компании, переключитесь на неё в боковой панели —
      отделы, типы юнитов и этапы привязаны к активной компании сессии.
    </div>

    <template v-else>
      <div class="lists-toolbar">
        <SegmentedTabs v-model="tab" :tabs="tabsForUi" />
        <button v-if="canEdit" class="btn-filled" @click="addItem(tab)">
          <span class="material-symbols-outlined">add</span>
          <span>Добавить</span>
        </button>
      </div>

      <p class="pane-hint">
        <span class="material-symbols-outlined">info</span>
        {{ hint }}
      </p>

      <!-- Отделы -->
      <AppDataTable v-if="tab === 'departments'" :value="displayDepartments" empty-message="Отделов пока нет">
        <Column header="" style="width: 56px">
          <template #body>
            <span class="row-ico"><span class="material-symbols-outlined">apartment</span></span>
          </template>
        </Column>
        <Column header="Название">
          <template #body="{ data }">
            <input
              v-if="isEditingRow('d', data)"
              :ref="bindEditInput"
              :value="rowEditValue('d', data)"
              class="row-input"
              placeholder="Название отдела"
              @click.stop
              @input="setRowEditValue($event.target.value)"
              @keyup.enter="commitRowEdit('departments', data)"
              @keyup.escape="cancelRow"
            />
            <span v-else class="row-name">{{ data.name }}</span>
          </template>
        </Column>
        <Column header="" style="width: 180px" body-style="text-align: right">
          <template #body="{ data }">
            <div class="row-actions" @click.stop>
              <template v-if="isEditingRow('d', data)">
                <button class="pill-btn" @click="commitRowEdit('departments', data)">
                  <span class="material-symbols-outlined">check</span>{{ data.__draft ? 'Создать' : 'Сохранить' }}
                </button>
                <button class="icon-btn" title="Отмена" @click="cancelRow"><span class="material-symbols-outlined">close</span></button>
              </template>
              <template v-else-if="canEdit">
                <button class="icon-btn" title="Редактировать" @click="startEdit('d', data)"><span class="material-symbols-outlined">edit</span></button>
                <button class="icon-btn danger" title="Удалить" @click="askDelete('departments', data)"><span class="material-symbols-outlined">delete</span></button>
              </template>
            </div>
          </template>
        </Column>
      </AppDataTable>

      <!-- Типы юнитов -->
      <AppDataTable v-else-if="tab === 'unit-types'" :value="displayUnitTypes" empty-message="Типов юнитов пока нет">
        <Column header="" style="width: 56px">
          <template #body>
            <span class="row-ico"><span class="material-symbols-outlined">category</span></span>
          </template>
        </Column>
        <Column header="Название">
          <template #body="{ data }">
            <input
              v-if="isEditingRow('u', data)"
              :ref="bindEditInput"
              :value="rowEditValue('u', data)"
              class="row-input"
              placeholder="Название типа"
              @click.stop
              @input="setRowEditValue($event.target.value)"
              @keyup.enter="commitRowEdit('unit-types', data)"
              @keyup.escape="cancelRow"
            />
            <span v-else class="row-name">{{ data.name }}</span>
          </template>
        </Column>
        <Column header="" style="width: 180px" body-style="text-align: right">
          <template #body="{ data }">
            <div class="row-actions" @click.stop>
              <template v-if="isEditingRow('u', data)">
                <button class="pill-btn" @click="commitRowEdit('unit-types', data)">
                  <span class="material-symbols-outlined">check</span>{{ data.__draft ? 'Создать' : 'Сохранить' }}
                </button>
                <button class="icon-btn" title="Отмена" @click="cancelRow"><span class="material-symbols-outlined">close</span></button>
              </template>
              <template v-else-if="canEdit">
                <button class="icon-btn" title="Редактировать" @click="startEdit('u', data)"><span class="material-symbols-outlined">edit</span></button>
                <button class="icon-btn danger" title="Удалить" @click="askDelete('unit-types', data)"><span class="material-symbols-outlined">delete</span></button>
              </template>
            </div>
          </template>
        </Column>
      </AppDataTable>

      <!-- Этапы -->
      <AppDataTable v-else :value="displayStages" empty-message="Этапов пока нет" @row-reorder="onStageReorder">
        <Column row-reorder :reorderable-column="false" header="" style="width: 36px" />
        <Column header="" style="width: 48px">
          <template #body="{ data }">
            <span :class="['stage-chip', `tag-${stageDisplayColor(data)}`]" />
          </template>
        </Column>
        <Column header="Этап">
          <template #body="{ data }">
            <input
              v-if="isEditingRow('s', data)"
              :ref="bindEditInput"
              :value="rowEditValue('s', data)"
              class="row-input"
              placeholder="Название этапа"
              @click.stop
              @input="setRowEditValue($event.target.value)"
              @keyup.enter="commitStageEdit(data)"
              @keyup.escape="cancelRow"
            />
            <span v-else class="row-name">{{ data.name }}</span>
          </template>
        </Column>
        <Column header="Цвет" style="width: 240px">
          <template #body="{ data }">
            <div v-if="isEditingRow('s', data)" class="color-dots" @click.stop @mousedown.stop>
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
        <Column header="Порядок" style="width: 92px">
          <template #body="{ data }">
            <span v-if="!data.__draft" class="order-badge">#{{ data.order }}</span>
          </template>
        </Column>
        <Column header="" style="width: 180px" body-style="text-align: right">
          <template #body="{ data }">
            <div class="row-actions" @click.stop>
              <template v-if="isEditingRow('s', data)">
                <button class="pill-btn" @click="commitStageEdit(data)">
                  <span class="material-symbols-outlined">check</span>{{ data.__draft ? 'Создать' : 'Сохранить' }}
                </button>
                <button class="icon-btn" title="Отмена" @click="cancelRow"><span class="material-symbols-outlined">close</span></button>
              </template>
              <template v-else-if="canEdit">
                <button class="icon-btn" title="Редактировать" @click="startEdit('s', data)"><span class="material-symbols-outlined">edit</span></button>
                <button class="icon-btn danger" title="Удалить" @click="askDelete('stages', data)"><span class="material-symbols-outlined">delete</span></button>
              </template>
            </div>
          </template>
        </Column>
      </AppDataTable>
    </template>

    <ConfirmDialog
      :visible="deleteDlg.open"
      :header="`Удалить «${deleteDlg.item?.name}»?`"
      :message="deleteMessage"
      confirm-label="Удалить"
      danger-confirm
      @confirm="doDelete"
      @cancel="deleteDlg.open = false"
    />
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted } from 'vue'
import Column from 'primevue/column'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import AppDataTable from '@/components/common/AppDataTable.vue'
import SegmentedTabs from '@/components/common/SegmentedTabs.vue'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { usePermission } from '@/composables/usePermission.js'
import { getDepartments, createDepartment, updateDepartment, deleteDepartment } from '@/api/departments.js'
import { getUnitTypes, createUnitType, updateUnitType, deleteUnitType } from '@/api/unitTypes.js'
import { getStages, createStage, updateStage, deleteStage, reorderStages, STAGE_COLORS } from '@/api/stages.js'

const props = defineProps({ companyId: { type: Number, required: true } })

const auth = useAuthStore()
const notif = useNotificationsStore()
const { isAtLeast, ROLES } = usePermission()

const canEdit = computed(() => isAtLeast(ROLES.MANAGER))
const isActive = computed(() => props.companyId === auth.companyId)

const tabs = [
  { key: 'departments', label: 'Отделы', icon: 'apartment' },
  { key: 'unit-types', label: 'Типы юнитов', icon: 'category' },
  { key: 'stages', label: 'Этапы', icon: 'flag' },
]
const tab = ref('departments')
const tabsForUi = computed(() => tabs.map(t => ({
  value: t.key, label: t.label, icon: t.icon, badge: counts.value[t.key] || null,
})))

const hint = computed(() => {
  if (tab.value === 'departments') return 'Группировка сотрудников и сегмент в статистике.'
  if (tab.value === 'unit-types') return 'Категории работы — встреча, дизайн, написание кода и т. п.'
  return 'Колонки канбан-режима задач. Порядок — перетаскиванием.'
})

const departments = ref([])
const unitTypes = ref([])
const stages = ref([])
const counts = computed(() => ({
  'departments': departments.value.length,
  'unit-types': unitTypes.value.length,
  'stages': stages.value.length,
}))

const editing = ref(null)
const editName = ref('')
const editColor = ref('blue')
const draftActive = ref(null)

function isEditingRow(prefix, data) {
  if (data.__draft) return editing.value === `${prefix}-draft`
  return editing.value === `${prefix}-${data.id}`
}
function rowEditValue(prefix, data) {
  return editing.value === `${prefix}-${data.__draft ? 'draft' : data.id}` ? editName.value : data.name
}
function setRowEditValue(v) { editName.value = v }
function bindEditInput(el) { if (el && el.focus) nextTick(() => el.focus()) }
function stageDisplayColor(s) { return isEditingRow('s', s) ? editColor.value : s.color }

const draftItem = computed(() => {
  if (!draftActive.value) return null
  if (draftActive.value === 'stages') return { id: '__draft__', __draft: true, name: '', color: 'blue', order: 0 }
  return { id: '__draft__', __draft: true, name: '' }
})
const displayDepartments = computed(() =>
  draftActive.value === 'departments' ? [draftItem.value, ...departments.value] : departments.value)
const displayUnitTypes = computed(() =>
  draftActive.value === 'unit-types' ? [draftItem.value, ...unitTypes.value] : unitTypes.value)
const displayStages = computed(() =>
  draftActive.value === 'stages' ? [draftItem.value, ...stages.value] : stages.value)

watch(tab, cancelRow)

const deleteDlg = ref({ open: false, kind: null, item: null })
const deleteMessage = computed(() => {
  const k = deleteDlg.value.kind
  if (k === 'departments') return 'Отдел и его привязки к задачам будут удалены.'
  if (k === 'unit-types') return 'ВНИМАНИЕ: удаление каскадно удалит ВСЕ юниты этого типа во всей компании.'
  if (k === 'stages') return 'Этап будет удалён. Карточки задач, прикреплённые к нему, останутся без этапа.'
  return ''
})

onMounted(loadAll)
// Перезагружаем при смене активной компании (редактирование доступно только активной).
watch(() => auth.companyId, loadAll)

async function loadAll() {
  if (!isActive.value) {
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
function cancelRow() { editing.value = null; editName.value = ''; draftActive.value = null }
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
      if (kind === 'departments') departments.value.push(await createDepartment({ name }))
      else unitTypes.value.push(await createUnitType({ name }))
      notif.success('Добавлено')
    } else {
      if (name === item.name) { cancelRow(); return }
      if (kind === 'departments') _replace(departments.value, await updateDepartment(item.id, { name }))
      else _replace(unitTypes.value, await updateUnitType(item.id, { name }))
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
      stages.value.push(await createStage({ name, color: editColor.value }))
      notif.success('Добавлено')
    } else {
      _replace(stages.value, await updateStage(item.id, { name, color: editColor.value }))
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

function askDelete(kind, item) { deleteDlg.value = { open: true, kind, item } }
async function doDelete() {
  const { kind, item } = deleteDlg.value
  if (!item) return
  try {
    if (kind === 'departments') { await deleteDepartment(item.id); departments.value = departments.value.filter(x => x.id !== item.id) }
    else if (kind === 'unit-types') { await deleteUnitType(item.id); unitTypes.value = unitTypes.value.filter(x => x.id !== item.id) }
    else if (kind === 'stages') { await deleteStage(item.id); stages.value = stages.value.filter(x => x.id !== item.id) }
    notif.success('Удалено')
    deleteDlg.value.open = false
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить')
  }
}

async function onStageReorder(e) {
  if (draftActive.value === 'stages') return
  const reordered = e.value
  stages.value = reordered
  try {
    stages.value = await reorderStages(reordered.map(s => s.id))
  } catch (err) {
    notif.error(err?.message || 'Не удалось применить порядок')
    loadAll()
  }
}
</script>

<style scoped>
.lists-settings { display: flex; flex-direction: column; gap: 14px; }

.note {
  padding: 12px 14px; border-radius: var(--radius-md, 12px);
  background: var(--color-surface-high); color: var(--color-text-dim); font-size: 13px; line-height: 1.5;
}

.lists-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }

.pane-hint {
  display: inline-flex; align-items: center; gap: 8px; margin: 0; padding: 8px 14px;
  border-radius: var(--radius-full); background: var(--color-secondary-container);
  color: var(--color-on-secondary-container); font-size: 12.5px; font-weight: 500;
  align-self: flex-start; max-width: 100%;
}
.pane-hint .material-symbols-outlined { font-size: 16px; opacity: 0.8; }

.row-ico {
  width: 36px; height: 36px; border-radius: var(--radius-md); display: grid; place-items: center; flex-shrink: 0;
  background: var(--color-primary-container); color: var(--color-on-primary-container);
}
.row-ico .material-symbols-outlined { font-size: 20px; }

.row-name { font-weight: 600; color: var(--color-text); font-size: 14px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.row-input {
  border: none; outline: none; background: var(--color-surface); font: inherit; font-weight: 600; font-size: 14px;
  color: var(--color-text); padding: 8px 14px; border-radius: var(--radius-full); width: 100%; box-sizing: border-box;
  box-shadow: inset 0 0 0 2px var(--color-primary);
}

.row-actions { display: inline-flex; align-items: center; gap: 6px; justify-content: flex-end; }

.pill-btn {
  appearance: none; border: none; cursor: pointer; display: inline-flex; align-items: center; gap: 6px;
  padding: 8px 14px; border-radius: var(--radius-full); font: inherit; font-size: 12.5px; font-weight: 600;
  background: var(--color-primary); color: var(--color-on-primary);
}
.pill-btn:hover { background: var(--color-primary-hover); box-shadow: var(--shadow-sm); }
.pill-btn .material-symbols-outlined { font-size: 16px; }

.icon-btn {
  appearance: none; border: none; background: transparent; width: 34px; height: 34px; display: grid;
  place-items: center; border-radius: 50%; color: var(--color-text-dim); cursor: pointer;
  transition: background .14s, color .14s;
}
.icon-btn:hover { background: var(--color-surface-high); color: var(--color-text); }
.icon-btn.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
.icon-btn .material-symbols-outlined { font-size: 18px; }

.btn-filled {
  appearance: none; border: none; cursor: pointer; background: var(--color-primary); color: var(--color-on-primary);
  border-radius: var(--radius-full); padding: 9px 16px; font: inherit; font-weight: 600;
  display: inline-flex; align-items: center; gap: 6px;
}
.btn-filled:hover { background: var(--color-primary-hover); }
.btn-filled .material-symbols-outlined { font-size: 18px; }

.stage-chip {
  width: 24px; height: 24px; border-radius: var(--radius-full); background: var(--tag-surface);
  border: 2.5px solid var(--tag-accent, var(--tag-border)); flex-shrink: 0; display: inline-block;
}
.stage-chip.tag-red { --tag-surface: var(--tag-red-surface); --tag-border: var(--tag-red-border); --tag-accent: var(--tag-red-accent); }
.stage-chip.tag-orange { --tag-surface: var(--tag-orange-surface); --tag-border: var(--tag-orange-border); --tag-accent: var(--tag-orange-accent); }
.stage-chip.tag-amber { --tag-surface: var(--tag-amber-surface); --tag-border: var(--tag-amber-border); --tag-accent: var(--tag-amber-accent); }
.stage-chip.tag-green { --tag-surface: var(--tag-green-surface); --tag-border: var(--tag-green-border); --tag-accent: var(--tag-green-accent); }
.stage-chip.tag-teal { --tag-surface: var(--tag-teal-surface); --tag-border: var(--tag-teal-border); --tag-accent: var(--tag-teal-accent); }
.stage-chip.tag-blue { --tag-surface: var(--tag-blue-surface); --tag-border: var(--tag-blue-border); --tag-accent: var(--tag-blue-accent); }
.stage-chip.tag-violet { --tag-surface: var(--tag-violet-surface); --tag-border: var(--tag-violet-border); --tag-accent: var(--tag-violet-accent); }
.stage-chip.tag-pink { --tag-surface: var(--tag-pink-surface); --tag-border: var(--tag-pink-border); --tag-accent: var(--tag-pink-accent); }

.mini-tag {
  padding: 3px 12px; border-radius: var(--radius-full); font-size: 10px; font-weight: 700;
  text-transform: uppercase; letter-spacing: 0.08em; background: var(--tag-surface);
  color: var(--tag-accent); border: 1px solid var(--tag-border);
}
.mini-tag.tag-red { --tag-surface: var(--tag-red-surface); --tag-border: var(--tag-red-border); --tag-accent: var(--tag-red-accent); }
.mini-tag.tag-orange { --tag-surface: var(--tag-orange-surface); --tag-border: var(--tag-orange-border); --tag-accent: var(--tag-orange-accent); }
.mini-tag.tag-amber { --tag-surface: var(--tag-amber-surface); --tag-border: var(--tag-amber-border); --tag-accent: var(--tag-amber-accent); }
.mini-tag.tag-green { --tag-surface: var(--tag-green-surface); --tag-border: var(--tag-green-border); --tag-accent: var(--tag-green-accent); }
.mini-tag.tag-teal { --tag-surface: var(--tag-teal-surface); --tag-border: var(--tag-teal-border); --tag-accent: var(--tag-teal-accent); }
.mini-tag.tag-blue { --tag-surface: var(--tag-blue-surface); --tag-border: var(--tag-blue-border); --tag-accent: var(--tag-blue-accent); }
.mini-tag.tag-violet { --tag-surface: var(--tag-violet-surface); --tag-border: var(--tag-violet-border); --tag-accent: var(--tag-violet-accent); }
.mini-tag.tag-pink { --tag-surface: var(--tag-pink-surface); --tag-border: var(--tag-pink-border); --tag-accent: var(--tag-pink-accent); }

.order-badge {
  font-size: 12px; font-weight: 700; color: var(--color-text-dim); background: var(--color-surface-high);
  padding: 3px 10px; border-radius: var(--radius-full); font-variant-numeric: tabular-nums;
}

.color-dots { display: inline-flex; gap: 4px; flex-wrap: wrap; }
.color-dot {
  appearance: none; border: 2.5px solid var(--tag-accent, var(--tag-border)); background: var(--tag-surface);
  width: 22px; height: 22px; border-radius: 50%; cursor: pointer; transition: transform .12s, box-shadow .12s; padding: 0;
}
.color-dot:hover { transform: scale(1.12); }
.color-dot.selected { box-shadow: 0 0 0 2px var(--color-surface), 0 0 0 4px var(--color-primary); transform: scale(1.12); }
.color-dot.tag-red { --tag-surface: var(--tag-red-surface); --tag-border: var(--tag-red-border); --tag-accent: var(--tag-red-accent); }
.color-dot.tag-orange { --tag-surface: var(--tag-orange-surface); --tag-border: var(--tag-orange-border); --tag-accent: var(--tag-orange-accent); }
.color-dot.tag-amber { --tag-surface: var(--tag-amber-surface); --tag-border: var(--tag-amber-border); --tag-accent: var(--tag-amber-accent); }
.color-dot.tag-green { --tag-surface: var(--tag-green-surface); --tag-border: var(--tag-green-border); --tag-accent: var(--tag-green-accent); }
.color-dot.tag-teal { --tag-surface: var(--tag-teal-surface); --tag-border: var(--tag-teal-border); --tag-accent: var(--tag-teal-accent); }
.color-dot.tag-blue { --tag-surface: var(--tag-blue-surface); --tag-border: var(--tag-blue-border); --tag-accent: var(--tag-blue-accent); }
.color-dot.tag-violet { --tag-surface: var(--tag-violet-surface); --tag-border: var(--tag-violet-border); --tag-accent: var(--tag-violet-accent); }
.color-dot.tag-pink { --tag-surface: var(--tag-pink-surface); --tag-border: var(--tag-pink-border); --tag-accent: var(--tag-pink-accent); }
</style>
