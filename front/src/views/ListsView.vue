<template>
  <div class="lists-view">
    <header class="lists-header">
      <div class="lists-title-row">
        <h1 class="lists-title">Списки</h1>
        <CompanySelect v-if="auth.isRootAdmin" />
      </div>
      <p class="lists-subtitle">
        Справочники компании: отделы, типы юнитов и этапы задач.
        У каждой компании — свой набор.
      </p>
    </header>

    <nav class="lists-tabs" role="tablist">
      <button
        v-for="t in tabs"
        :key="t.key"
        class="tab"
        :class="{ active: tab === t.key }"
        role="tab"
        @click="tab = t.key"
      >
        <span class="material-symbols-outlined">{{ t.icon }}</span>
        <span>{{ t.label }}</span>
        <span class="tab-count">{{ counts[t.key] }}</span>
      </button>
    </nav>

    <div v-if="!effectiveCompanyId && auth.isRootAdmin" class="placeholder">
      <div class="placeholder-icon"><span class="material-symbols-outlined">domain</span></div>
      <h3>Выберите компанию</h3>
      <p>Списки ведутся внутри компании. Выберите её в селекторе сверху, чтобы продолжить.</p>
    </div>

    <section v-else class="lists-pane">
      <transition name="pane-swap" mode="out-in">
        <!-- Отделы -->
        <div v-if="tab === 'departments'" key="departments" class="pane">
          <div class="pane-toolbar">
            <p class="hint">Используются для группировки сотрудников и в статистике.</p>
            <button v-if="canEdit" class="btn-filled" @click="addItem('departments')">
              <span class="material-symbols-outlined">add</span> Добавить
            </button>
          </div>
          <div class="rows">
            <div v-if="adding === 'departments'" class="row editing">
              <span class="material-symbols-outlined row-icon">apartment</span>
              <input
                v-model="newName" class="row-input" placeholder="Название отдела"
                autofocus @keyup.enter="saveNew('departments')" @keyup.escape="cancelAdd"
              />
              <button class="icon-btn success" @click="saveNew('departments')" title="Сохранить">
                <span class="material-symbols-outlined">check</span>
              </button>
              <button class="icon-btn" @click="cancelAdd" title="Отмена">
                <span class="material-symbols-outlined">close</span>
              </button>
            </div>
            <ListRow
              v-for="d in departments" :key="d.id"
              :item="d" icon="apartment"
              :editing="editing === `d-${d.id}`"
              :can-edit="canEdit"
              @start="startEdit('d', d)"
              @save="saveEdit('departments', d, $event)"
              @cancel="editing = null"
              @delete="askDelete('departments', d)"
            />
            <Empty
              v-if="!departments.length && adding !== 'departments'"
              icon="apartment" title="Отделов пока нет"
              text="Создайте первый — он появится в фильтрах задач и статистике."
            />
          </div>
        </div>

        <!-- Типы юнитов -->
        <div v-else-if="tab === 'unit-types'" key="unit-types" class="pane">
          <div class="pane-toolbar">
            <p class="hint">Категории работы — встреча, дизайн, написание кода и т. п.</p>
            <button v-if="canEdit" class="btn-filled" @click="addItem('unit-types')">
              <span class="material-symbols-outlined">add</span> Добавить
            </button>
          </div>
          <div class="rows">
            <div v-if="adding === 'unit-types'" class="row editing">
              <span class="material-symbols-outlined row-icon">category</span>
              <input
                v-model="newName" class="row-input" placeholder="Название типа"
                autofocus @keyup.enter="saveNew('unit-types')" @keyup.escape="cancelAdd"
              />
              <button class="icon-btn success" @click="saveNew('unit-types')" title="Сохранить">
                <span class="material-symbols-outlined">check</span>
              </button>
              <button class="icon-btn" @click="cancelAdd" title="Отмена">
                <span class="material-symbols-outlined">close</span>
              </button>
            </div>
            <ListRow
              v-for="ut in unitTypes" :key="ut.id"
              :item="ut" icon="category"
              :editing="editing === `u-${ut.id}`"
              :can-edit="canEdit"
              @start="startEdit('u', ut)"
              @save="saveEdit('unit-types', ut, $event)"
              @cancel="editing = null"
              @delete="askDelete('unit-types', ut)"
            />
            <Empty
              v-if="!unitTypes.length && adding !== 'unit-types'"
              icon="category" title="Типов юнитов пока нет"
              text="Без типов нельзя создавать юниты. Добавьте хотя бы один."
            />
          </div>
        </div>

        <!-- Этапы -->
        <div v-else key="stages" class="pane">
          <div class="pane-toolbar">
            <p class="hint">
              Колонки канбан-режима задач. Порядок здесь определяет порядок колонок.
            </p>
            <button v-if="canEdit" class="btn-filled" @click="addItem('stages')">
              <span class="material-symbols-outlined">add</span> Добавить
            </button>
          </div>

          <div v-if="adding === 'stages'" class="row editing stage-add">
            <span class="material-symbols-outlined row-icon">flag</span>
            <input
              v-model="newName" class="row-input" placeholder="Название этапа"
              autofocus @keyup.enter="saveNew('stages')" @keyup.escape="cancelAdd"
            />
            <ColorDots v-model="newColor" />
            <button class="icon-btn success" @click="saveNew('stages')" title="Сохранить">
              <span class="material-symbols-outlined">check</span>
            </button>
            <button class="icon-btn" @click="cancelAdd" title="Отмена">
              <span class="material-symbols-outlined">close</span>
            </button>
          </div>

          <div class="rows" @dragover.prevent>
            <div
              v-for="(s, idx) in stages"
              :key="s.id"
              :draggable="canEdit && editing !== `s-${s.id}`"
              :class="['drag-slot', { 'drop-before': dragOverIdx === idx && dragFromIdx > idx,
                                     'drop-after':  dragOverIdx === idx && dragFromIdx < idx,
                                     'dragging': dragFromIdx === idx }]"
              @dragstart="onDragStart(idx, $event)"
              @dragover.prevent="dragOverIdx = idx"
              @dragend="onDragEnd"
              @drop="onDrop(idx)"
            >
              <StageRow
                :stage="s"
                :editing="editing === `s-${s.id}`"
                :can-edit="canEdit"
                @start="startEdit('s', s)"
                @save="saveStage(s, $event)"
                @cancel="editing = null"
                @delete="askDelete('stages', s)"
              />
            </div>
          </div>

          <Empty
            v-if="!stages.length && adding !== 'stages'"
            icon="flag" title="Этапов пока нет"
            text="Добавьте, например: «К работе», «В работе», «На проверке», «Готово»."
          />
        </div>
      </transition>
    </section>

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
import { ref, computed, watch, onMounted, h } from 'vue'
import CompanySelect from '@/components/common/CompanySelect.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
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

const departments = ref([])
const unitTypes = ref([])
const stages = ref([])

const counts = computed(() => ({
  'departments': departments.value.length,
  'unit-types': unitTypes.value.length,
  'stages': stages.value.length,
}))

const effectiveCompanyId = computed(() => companies.effectiveCompanyId)

const adding = ref(null)
const newName = ref('')
const newColor = ref('blue')
const editing = ref(null)

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
  adding.value = kind
  newName.value = ''
  newColor.value = 'blue'
  editing.value = null
}

function cancelAdd() {
  adding.value = null
  newName.value = ''
}

async function saveNew(kind) {
  const name = newName.value.trim()
  if (!name) return
  try {
    if (kind === 'departments') {
      const d = await createDepartment({ name })
      departments.value.push(d)
    } else if (kind === 'unit-types') {
      const u = await createUnitType({ name })
      unitTypes.value.push(u)
    } else if (kind === 'stages') {
      const s = await createStage({ name, color: newColor.value })
      stages.value.push(s)
    }
    notif.success('Добавлено')
    cancelAdd()
  } catch (e) {
    notif.error(e?.message || 'Не удалось создать')
  }
}

function startEdit(prefix, item) {
  editing.value = `${prefix}-${item.id}`
}

async function saveEdit(kind, item, name) {
  const trimmed = name.trim()
  if (!trimmed || trimmed === item.name) { editing.value = null; return }
  try {
    if (kind === 'departments') {
      const upd = await updateDepartment(item.id, { name: trimmed })
      _replace(departments.value, upd)
    } else {
      const upd = await updateUnitType(item.id, { name: trimmed })
      _replace(unitTypes.value, upd)
    }
    editing.value = null
    notif.success('Сохранено')
  } catch (e) {
    notif.error(e?.message || 'Не удалось сохранить')
  }
}

async function saveStage(item, payload) {
  const trimmed = payload.name.trim()
  if (!trimmed) { editing.value = null; return }
  try {
    const upd = await updateStage(item.id, { name: trimmed, color: payload.color })
    _replace(stages.value, upd)
    editing.value = null
    notif.success('Сохранено')
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

// Нативный HTML5 DnD — без сторонних либ (см. PLAN_V3.md, Этап 3).
const dragFromIdx = ref(-1)
const dragOverIdx = ref(-1)

function onDragStart(idx, ev) {
  if (!canEdit.value) return
  dragFromIdx.value = idx
  try { ev.dataTransfer.effectAllowed = 'move' } catch {}
}

function onDragEnd() {
  dragFromIdx.value = -1
  dragOverIdx.value = -1
}

async function onDrop(toIdx) {
  const from = dragFromIdx.value
  onDragEnd()
  if (from < 0 || from === toIdx) return
  const next = [...stages.value]
  const [moved] = next.splice(from, 1)
  next.splice(toIdx, 0, moved)
  stages.value = next
  try {
    const upd = await reorderStages(next.map(s => s.id))
    stages.value = upd
  } catch (e) {
    notif.error(e?.message || 'Не удалось применить порядок')
    loadAll()
  }
}

// Inline-компоненты
const Empty = {
  props: ['icon', 'title', 'text'],
  setup(p) {
    return () => h('div', { class: 'empty' }, [
      h('div', { class: 'empty-icon' }, [
        h('span', { class: 'material-symbols-outlined' }, p.icon),
      ]),
      h('h4', p.title),
      h('p', p.text),
    ])
  },
}

const ListRow = {
  props: {
    item: Object, icon: String, editing: Boolean, canEdit: Boolean,
  },
  emits: ['start', 'save', 'cancel', 'delete'],
  setup(props, { emit }) {
    const editName = ref('')
    watch(() => props.editing, (v) => {
      if (v) editName.value = props.item.name
    })
    return () => props.editing
      ? h('div', { class: 'row editing' }, [
          h('span', { class: 'material-symbols-outlined row-icon' }, props.icon),
          h('input', {
            class: 'row-input',
            value: editName.value,
            onInput: e => editName.value = e.target.value,
            onKeyup: (e) => {
              if (e.key === 'Enter') emit('save', editName.value)
              else if (e.key === 'Escape') emit('cancel')
            },
          }),
          h('button', {
            class: 'icon-btn success', title: 'Сохранить',
            onClick: () => emit('save', editName.value),
          }, [h('span', { class: 'material-symbols-outlined' }, 'check')]),
          h('button', {
            class: 'icon-btn', title: 'Отмена', onClick: () => emit('cancel'),
          }, [h('span', { class: 'material-symbols-outlined' }, 'close')]),
        ])
      : h('div', { class: 'row' }, [
          h('span', { class: 'material-symbols-outlined row-icon' }, props.icon),
          h('span', { class: 'row-name' }, props.item.name),
          props.canEdit ? h('div', { class: 'row-actions' }, [
            h('button', {
              class: 'icon-btn', title: 'Редактировать', onClick: () => emit('start'),
            }, [h('span', { class: 'material-symbols-outlined' }, 'edit')]),
            h('button', {
              class: 'icon-btn danger', title: 'Удалить', onClick: () => emit('delete'),
            }, [h('span', { class: 'material-symbols-outlined' }, 'delete')]),
          ]) : null,
        ])
  },
}

const ColorDots = {
  props: ['modelValue'],
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () => h('div', { class: 'color-dots' },
      STAGE_COLORS.map(c => h('button', {
        type: 'button',
        class: ['color-dot', `tag-${c}`, { selected: props.modelValue === c }],
        title: c,
        onClick: () => emit('update:modelValue', c),
      })))
  },
}

const StageRow = {
  props: { stage: Object, editing: Boolean, canEdit: Boolean },
  emits: ['start', 'save', 'cancel', 'delete'],
  setup(props, { emit }) {
    const editName = ref('')
    const editColor = ref('blue')
    watch(() => props.editing, (v) => {
      if (v) { editName.value = props.stage.name; editColor.value = props.stage.color }
    })
    return () => props.editing
      ? h('div', { class: 'row editing' }, [
          h('span', { class: ['stage-dot', `tag-${editColor.value}`] }),
          h('input', {
            class: 'row-input', value: editName.value,
            onInput: e => editName.value = e.target.value,
            onKeyup: (e) => {
              if (e.key === 'Enter') emit('save', { name: editName.value, color: editColor.value })
              else if (e.key === 'Escape') emit('cancel')
            },
          }),
          h(ColorDots, {
            modelValue: editColor.value,
            'onUpdate:modelValue': (v) => editColor.value = v,
          }),
          h('button', {
            class: 'icon-btn success', title: 'Сохранить',
            onClick: () => emit('save', { name: editName.value, color: editColor.value }),
          }, [h('span', { class: 'material-symbols-outlined' }, 'check')]),
          h('button', {
            class: 'icon-btn', title: 'Отмена', onClick: () => emit('cancel'),
          }, [h('span', { class: 'material-symbols-outlined' }, 'close')]),
        ])
      : h('div', { class: 'row stage-row' }, [
          props.canEdit ? h('span', {
            class: 'drag-grip material-symbols-outlined', title: 'Перетащить',
          }, 'drag_indicator') : null,
          h('span', { class: ['stage-dot', `tag-${props.stage.color}`] }),
          h('span', { class: 'row-name' }, props.stage.name),
          h('span', { class: 'order-badge' }, `#${props.stage.order}`),
          props.canEdit ? h('div', { class: 'row-actions' }, [
            h('button', {
              class: 'icon-btn', title: 'Редактировать', onClick: () => emit('start'),
            }, [h('span', { class: 'material-symbols-outlined' }, 'edit')]),
            h('button', {
              class: 'icon-btn danger', title: 'Удалить', onClick: () => emit('delete'),
            }, [h('span', { class: 'material-symbols-outlined' }, 'delete')]),
          ]) : null,
        ])
  },
}
</script>

<style scoped>
.lists-view {
  padding: 24px;
  max-width: 980px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.lists-header { display: flex; flex-direction: column; gap: 8px; }
.lists-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.lists-title { font-size: 28px; font-weight: 700; margin: 0; color: var(--color-on-surface); }
.lists-subtitle { margin: 0; font-size: 14px; color: var(--color-on-surface-variant); }

.lists-tabs {
  display: inline-flex;
  background: var(--color-surface-container);
  padding: 4px;
  border-radius: var(--radius-full, 999px);
  gap: 2px;
  width: max-content;
  max-width: 100%;
  overflow-x: auto;
}
.tab {
  appearance: none;
  border: none;
  background: transparent;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  font: inherit;
  font-weight: 600;
  color: var(--color-on-surface-variant);
  border-radius: var(--radius-full, 999px);
  transition: background .12s, color .12s;
  white-space: nowrap;
}
.tab .material-symbols-outlined { font-size: 18px; }
.tab:hover { color: var(--color-on-surface); }
.tab.active { background: var(--color-primary); color: var(--color-on-primary); }
.tab-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 22px;
  height: 22px;
  padding: 0 6px;
  border-radius: 999px;
  background: color-mix(in oklab, currentColor 18%, transparent);
  font-size: 11px;
  font-weight: 700;
}

.placeholder {
  padding: 48px 20px;
  text-align: center;
  background: var(--color-surface-container);
  border-radius: var(--radius-lg, 16px);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}
.placeholder-icon {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  display: grid;
  place-items: center;
}
.placeholder-icon .material-symbols-outlined { font-size: 32px; }
.placeholder h3 { margin: 0; color: var(--color-on-surface); }
.placeholder p { margin: 0; color: var(--color-on-surface-variant); font-size: 14px; }

.lists-pane {
  background: var(--color-surface);
  border: 1px solid var(--color-outline-variant);
  border-radius: var(--radius-lg, 16px);
  padding: 16px;
}

.pane-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 12px;
  margin-bottom: 12px;
  border-bottom: 1px dashed var(--color-outline-variant);
}
.hint { margin: 0; font-size: 13px; color: var(--color-on-surface-variant); }

.rows { display: flex; flex-direction: column; gap: 6px; }

.row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  background: var(--color-surface-container);
  border-radius: var(--radius-md, 12px);
}
.row.editing { background: var(--color-surface-high); }
.row.stage-add { background: var(--color-surface-high); }

.row-icon, .stage-dot {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border-radius: var(--radius-md, 12px);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 18px;
  flex: none;
}

.stage-dot {
  background: var(--tag-surface);
  border: 2px solid var(--tag-border);
}
.stage-dot.tag-red    { --tag-surface: var(--tag-red-surface);    --tag-border: var(--tag-red-border); }
.stage-dot.tag-orange { --tag-surface: var(--tag-orange-surface); --tag-border: var(--tag-orange-border); }
.stage-dot.tag-amber  { --tag-surface: var(--tag-amber-surface);  --tag-border: var(--tag-amber-border); }
.stage-dot.tag-green  { --tag-surface: var(--tag-green-surface);  --tag-border: var(--tag-green-border); }
.stage-dot.tag-teal   { --tag-surface: var(--tag-teal-surface);   --tag-border: var(--tag-teal-border); }
.stage-dot.tag-blue   { --tag-surface: var(--tag-blue-surface);   --tag-border: var(--tag-blue-border); }
.stage-dot.tag-violet { --tag-surface: var(--tag-violet-surface); --tag-border: var(--tag-violet-border); }
.stage-dot.tag-pink   { --tag-surface: var(--tag-pink-surface);   --tag-border: var(--tag-pink-border); }

.row-name { flex: 1; font-weight: 500; color: var(--color-on-surface); min-width: 0; }
.row-input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  font: inherit;
  color: var(--color-on-surface);
  border-bottom: 2px solid var(--color-primary);
  padding: 4px 0;
}

.row-actions { display: flex; gap: 2px; }
.icon-btn {
  appearance: none;
  border: none;
  background: transparent;
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  color: var(--color-on-surface-variant);
  cursor: pointer;
  transition: background .12s, color .12s;
}
.icon-btn:hover { background: var(--color-surface-high); color: var(--color-on-surface); }
.icon-btn.success:hover { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.icon-btn.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
.icon-btn .material-symbols-outlined { font-size: 18px; }

.color-dots { display: inline-flex; gap: 4px; flex-wrap: wrap; }
.color-dot {
  appearance: none;
  border: 2px solid var(--tag-border);
  background: var(--tag-surface);
  width: 22px;
  height: 22px;
  border-radius: 50%;
  cursor: pointer;
  transition: transform .12s, box-shadow .12s;
  padding: 0;
}
.color-dot:hover { transform: scale(1.1); }
.color-dot.selected {
  box-shadow: 0 0 0 2px var(--color-surface), 0 0 0 4px var(--color-primary);
  transform: scale(1.1);
}
.color-dot.tag-red    { --tag-surface: var(--tag-red-surface);    --tag-border: var(--tag-red-border); }
.color-dot.tag-orange { --tag-surface: var(--tag-orange-surface); --tag-border: var(--tag-orange-border); }
.color-dot.tag-amber  { --tag-surface: var(--tag-amber-surface);  --tag-border: var(--tag-amber-border); }
.color-dot.tag-green  { --tag-surface: var(--tag-green-surface);  --tag-border: var(--tag-green-border); }
.color-dot.tag-teal   { --tag-surface: var(--tag-teal-surface);   --tag-border: var(--tag-teal-border); }
.color-dot.tag-blue   { --tag-surface: var(--tag-blue-surface);   --tag-border: var(--tag-blue-border); }
.color-dot.tag-violet { --tag-surface: var(--tag-violet-surface); --tag-border: var(--tag-violet-border); }
.color-dot.tag-pink   { --tag-surface: var(--tag-pink-surface);   --tag-border: var(--tag-pink-border); }

.drag-grip {
  cursor: grab;
  color: var(--color-on-surface-variant);
  font-size: 18px;
  opacity: .55;
  transition: opacity .12s;
}
.drag-grip:active { cursor: grabbing; }
.row:hover .drag-grip { opacity: 1; }

.drag-slot { transition: transform .12s; }
.drag-slot[draggable="true"] { cursor: grab; }
.drag-slot[draggable="true"]:active { cursor: grabbing; }
.drag-slot.dragging { opacity: .4; }
.drag-slot.drop-before { box-shadow: inset 0 3px 0 var(--color-primary); border-radius: var(--radius-md, 12px); }
.drag-slot.drop-after  { box-shadow: inset 0 -3px 0 var(--color-primary); border-radius: var(--radius-md, 12px); }

.order-badge {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-on-surface-variant);
  background: var(--color-surface);
  padding: 2px 8px;
  border-radius: 999px;
}

.empty {
  text-align: center;
  padding: 28px 12px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}
.empty .empty-icon {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  display: grid;
  place-items: center;
}
.empty .empty-icon .material-symbols-outlined { font-size: 28px; }
.empty h4 { margin: 0; font-size: 16px; color: var(--color-on-surface); }
.empty p { margin: 0; font-size: 13px; color: var(--color-on-surface-variant); max-width: 360px; }

.btn-filled {
  appearance: none;
  border: none;
  cursor: pointer;
  background: var(--color-primary);
  color: var(--color-on-primary);
  border-radius: var(--radius-full, 999px);
  padding: 9px 16px;
  font: inherit;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.btn-filled:hover { filter: brightness(.94); }

.pane-swap-enter-active, .pane-swap-leave-active { transition: opacity .15s, transform .15s; }
.pane-swap-enter-from, .pane-swap-leave-to { opacity: 0; transform: translateX(8px); }

@media (max-width: 600px) {
  .lists-view { padding: 16px; }
  .pane-toolbar { flex-direction: column; align-items: stretch; gap: 10px; }
  .tab span:not(.material-symbols-outlined):not(.tab-count) { display: none; }
}
</style>
