<template>
  <div class="cm">
    <div class="cm-head">
      <h3 class="cm-title">Календари компании</h3>
      <p class="cm-sub">
        Создавайте календари и настраивайте поля их карточек. У каждой записи уже
        есть обязательная дата и время — остальные поля добавляете вы. Поле можно
        показывать только при определённом значении другого (условная видимость).
      </p>
    </div>

    <div class="cm-body">
      <!-- Список календарей -->
      <aside class="cm-side">
        <div class="cm-side-head">
          <span>Список</span>
          <button class="cm-add-circle" title="Новый календарь" @click="openCreate">
            <span class="material-symbols-outlined">add</span>
          </button>
        </div>
        <div class="cm-side-list">
          <div v-if="store.loadingList" class="cm-side-note">Загрузка…</div>
          <div v-else-if="!store.calendars.length" class="cm-side-note">Календарей пока нет</div>
          <button
            v-for="c in store.calendars"
            :key="c.id"
            class="cm-side-item"
            :class="{ active: c.id === editId }"
            @click="selectCalendar(c)"
          >
            <span class="material-symbols-outlined">calendar_month</span>
            <span class="cm-side-name">{{ c.name }}</span>
          </button>
        </div>
      </aside>

      <!-- Редактор выбранного календаря -->
      <section class="cm-detail">
        <template v-if="current">
          <div class="cm-detail-head">
            <input v-model="editName" class="ctl cm-name" placeholder="Название календаря" @input="dirty = true" />
            <button class="cm-icon-btn danger" title="Удалить календарь" @click="confirmDelete = true">
              <span class="material-symbols-outlined">delete</span>
            </button>
          </div>

          <div class="cm-fields">
            <div v-if="!editFields.length" class="cm-fields-empty">
              <span class="material-symbols-outlined">dashboard_customize</span>
              <p>В календаре пока нет полей (кроме даты и времени)</p>
              <button class="cm-btn-tonal" @click="openField(-1)">
                <span class="material-symbols-outlined">add</span> Добавить первое поле
              </button>
            </div>

            <ul v-else class="cm-field-rows">
              <li
                v-for="(f, i) in editFields"
                :key="f._k"
                class="cm-field-row"
                @click="openField(i)"
              >
                <span class="cm-reorder" @click.stop>
                  <button class="cm-reorder-btn" :disabled="i === 0" title="Выше" @click="move(i, -1)">
                    <span class="material-symbols-outlined">keyboard_arrow_up</span>
                  </button>
                  <button class="cm-reorder-btn" :disabled="i === editFields.length - 1" title="Ниже" @click="move(i, 1)">
                    <span class="material-symbols-outlined">keyboard_arrow_down</span>
                  </button>
                </span>

                <span class="cm-field-icon"><span class="material-symbols-outlined">{{ fieldIcon(f.type) }}</span></span>

                <span class="cm-field-main">
                  <span class="cm-field-label">{{ f.label || 'Без названия' }}</span>
                  <span class="cm-field-type">
                    {{ fieldLabel(f.type) }}
                    <template v-if="f.visible_field_id"> · условное</template>
                  </span>
                </span>

                <span class="cm-field-meta">
                  <span v-if="f.visible_field_id" class="cm-badge col" title="Показывается по условию">
                    <span class="material-symbols-outlined">rule</span>
                  </span>
                  <span class="cm-badge" :title="`Ширина ${f.col_span} · высота ${f.row_span}`">{{ f.col_span }}×{{ f.row_span }}</span>
                  <span class="cm-badge col" :class="{ off: !f.show_in_table }" :title="f.show_in_table ? 'Показывается в плитке/таблице' : 'Скрыто из плитки'">
                    <span class="material-symbols-outlined">{{ f.show_in_table ? 'visibility' : 'visibility_off' }}</span>
                  </span>
                </span>

                <span class="cm-field-actions" @click.stop>
                  <button class="cm-icon-btn sm" title="Настроить" @click="openField(i)">
                    <span class="material-symbols-outlined">tune</span>
                  </button>
                  <button class="cm-icon-btn sm danger" title="Удалить поле" @click="removeField(i)">
                    <span class="material-symbols-outlined">close</span>
                  </button>
                </span>
              </li>
            </ul>
          </div>

          <div class="cm-detail-foot">
            <button v-if="editFields.length" class="cm-btn-tonal" @click="openField(-1)">
              <span class="material-symbols-outlined">add</span> Добавить поле
            </button>
            <span v-if="dirty" class="cm-dirty"><span class="cm-dot" /> Есть несохранённые изменения</span>
            <span class="cm-foot-spacer" />
            <button class="cm-btn-primary" :disabled="saving || !dirty" @click="save">
              <span v-if="saving" class="material-symbols-outlined spin">progress_activity</span>
              Сохранить
            </button>
          </div>
        </template>

        <div v-else class="cm-detail-empty">
          <span class="material-symbols-outlined">calendar_month</span>
          <p>Выберите календарь слева или создайте новый</p>
        </div>
      </section>
    </div>

    <!-- Настройка поля -->
    <AppDialog
      v-model="fieldOpen"
      :title="fieldIndex === -1 ? 'Новое поле' : 'Настройка поля'"
      :icon="draft.type ? fieldIcon(draft.type) : 'add_box'"
      size="md"
      :actions="[{ kind: 'cancel', label: 'Отмена' }, { kind: 'confirm', label: 'Готово' }]"
      @cancel="fieldOpen = false"
      @confirm="applyField"
    >
      <div class="fd">
        <label class="fd-field">
          <span class="fd-label">Название</span>
          <input v-model="draft.label" class="ctl" placeholder="Например: Тема встречи" />
        </label>

        <label class="fd-field">
          <span class="fd-label">Тип</span>
          <Select
            v-model="draft.type" :options="typeOptions" option-label="label" option-value="value"
            @update:model-value="onDraftType"
          >
            <template #option="{ option }">
              <span class="fd-type-opt"><span class="material-symbols-outlined">{{ fieldIcon(option.value) }}</span>{{ option.label }}</span>
            </template>
          </Select>
        </label>

        <!-- Конфиг типа -->
        <label v-if="draft.type === 'text'" class="fd-check">
          <Checkbox v-model="draft.config.multiline" binary /> Многострочное поле
        </label>

        <label v-else-if="draft.type === 'number'" class="fd-field">
          <span class="fd-label">Шаблон номера (regex, необязательно)</span>
          <input v-model="draft.config.pattern" class="ctl" placeholder="напр. ^\d{3}-\d{4}$" />
          <span class="fd-hint">Если задан — значение должно совпадать с шаблоном.</span>
        </label>

        <div v-else-if="draft.type === 'select'" class="fd-field">
          <span class="fd-label">Варианты выбора</span>
          <div class="fd-options">
            <div v-for="(opt, oi) in draft.config.options" :key="oi" class="fd-option">
              <input :value="opt" class="ctl" placeholder="Вариант" @input="draft.config.options[oi] = $event.target.value" />
              <button class="cm-icon-btn sm danger" title="Удалить" @click="draft.config.options.splice(oi, 1)">
                <span class="material-symbols-outlined">close</span>
              </button>
            </div>
          </div>
          <button class="cm-btn-text" @click="draft.config.options.push('')">
            <span class="material-symbols-outlined">add</span> Добавить вариант
          </button>
          <label class="fd-check"><Checkbox v-model="draft.config.multiple" binary /> Разрешить выбор нескольких</label>
        </div>

        <div v-else-if="draft.type === 'datetime'" class="fd-field">
          <span class="fd-label">Что показывать</span>
          <div class="fd-inline">
            <label class="fd-check"><Checkbox v-model="draft.config.year" binary /> Год</label>
            <label class="fd-check"><Checkbox v-model="draft.config.month_day" binary /> Месяц и день</label>
            <label class="fd-check"><Checkbox v-model="draft.config.time" binary /> Время</label>
          </div>
        </div>

        <!-- Условная видимость -->
        <div class="fd-field fd-cond">
          <span class="fd-label">Условие показа поля</span>
          <Select
            v-model="draft.visible_field_id" :options="conditionOptions"
            option-label="label" option-value="value" show-clear placeholder="Показывать всегда"
            @update:model-value="onConditionField"
          />
          <span v-if="!conditionSourceFields.length" class="fd-hint">
            Условие можно повесить на другое поле-галочку или список выбора. Сначала добавьте такое поле и сохраните календарь.
          </span>
          <template v-else-if="draft.visible_field_id">
            <span class="fd-hint" v-if="conditionSource?.type === 'checkbox'">
              Поле будет видно, когда галочка «{{ conditionSource.label }}» отмечена.
            </span>
            <div v-else-if="conditionSource?.type === 'select'" class="fd-cond-val">
              <span class="fd-label">…когда выбрано значение</span>
              <Select
                v-model="draft.visible_value" :options="conditionSourceOptions"
                placeholder="Выберите вариант"
              />
            </div>
          </template>
        </div>

        <!-- Раскладка -->
        <div class="fd-grid">
          <div class="fd-field">
            <span class="fd-label">Ширина (колонок)</span>
            <div class="fd-seg">
              <button v-for="n in 3" :key="n" :class="{ active: draft.col_span === n }" @click="draft.col_span = n">{{ n }}</button>
            </div>
          </div>
          <div class="fd-field">
            <span class="fd-label">Высота (строк)</span>
            <div class="fd-seg">
              <button v-for="n in 3" :key="n" :class="{ active: draft.row_span === n }" @click="draft.row_span = n">{{ n }}</button>
            </div>
          </div>
        </div>

        <label class="fd-check"><Checkbox v-model="draft.show_in_table" binary /> Показывать в плитке и таблице</label>
        <label class="fd-check"><Checkbox v-model="draft.show_in_card" binary /> Показывать в карточке события</label>
      </div>
    </AppDialog>

    <!-- Создание календаря -->
    <AppDialog
      v-model="creating" title="Новый календарь" icon="add" :busy="saving"
      :actions="[{ kind: 'cancel', label: 'Отмена' }, { kind: 'confirm', label: 'Создать' }]"
      @cancel="creating = false" @confirm="doCreate"
    >
      <input v-model="newName" class="ctl" placeholder="Название календаря" @keyup.enter="doCreate" />
    </AppDialog>

    <ConfirmDialog
      :visible="confirmDelete"
      header="Удалить календарь?"
      message="Календарь и все его записи будут удалены безвозвратно."
      confirm-label="Удалить" danger-confirm
      @confirm="doDelete" @cancel="confirmDelete = false"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import Select from 'primevue/select'
import Checkbox from 'primevue/checkbox'
import AppDialog from '@/components/common/AppDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import * as api from '@/api/calendars.js'
import { useCalendarsStore } from '@/stores/calendars.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { FIELD_TYPES, defaultConfig, fieldIcon, fieldLabel, canBeCondition } from '@/utils/calendarFields.js'

const store = useCalendarsStore()
const notif = useNotificationsStore()

const typeOptions = FIELD_TYPES.map((f) => ({ label: f.label, value: f.value || f.type }))

const editId = ref(null)
const editName = ref('')
const editFields = ref([])
const dirty = ref(false)
const saving = ref(false)
const creating = ref(false)
const newName = ref('')
const confirmDelete = ref(false)

let keySeq = 0
const current = computed(() => store.calendars.find((c) => c.id === editId.value) || null)

function selectCalendar(c) {
  editId.value = c.id
  editName.value = c.name
  editFields.value = (c.fields || []).map((f) => normalizeField(f))
  dirty.value = false
}

function normalizeField(f) {
  return {
    _k: ++keySeq,
    id: f.id || 0,
    label: f.label || '',
    type: f.type || 'text',
    config: { ...defaultConfig(f.type || 'text'), ...(f.config || {}) },
    col_span: clampSpan(f.col_span),
    row_span: clampSpan(f.row_span),
    show_in_table: f.show_in_table !== false,
    show_in_card: f.show_in_card !== false,
    visible_field_id: f.visible_field_id ?? null,
    visible_value: f.visible_value ?? null,
  }
}
function clampSpan(v) { return Math.min(3, Math.max(1, v || 1)) }

function removeField(i) { editFields.value.splice(i, 1); dirty.value = true }
function move(i, dir) {
  const j = i + dir
  if (j < 0 || j >= editFields.value.length) return
  const arr = editFields.value
  ;[arr[i], arr[j]] = [arr[j], arr[i]]
  dirty.value = true
}

// ── Диалог настройки поля ──
const fieldOpen = ref(false)
const fieldIndex = ref(-1)
const draft = reactive({
  _k: 0, id: 0, label: '', type: 'text', config: {}, col_span: 1, row_span: 1,
  show_in_table: true, show_in_card: true, visible_field_id: null, visible_value: null,
})

// Поля-источники условия: сохранённые (id>0) галочки/списки, кроме редактируемого.
const conditionSourceFields = computed(() =>
  editFields.value.filter((f) => f.id > 0 && canBeCondition(f.type) && f._k !== draft._k),
)
const conditionOptions = computed(() =>
  conditionSourceFields.value.map((f) => ({ label: f.label || 'Без названия', value: f.id })),
)
const conditionSource = computed(() =>
  conditionSourceFields.value.find((f) => f.id === draft.visible_field_id) || null,
)
const conditionSourceOptions = computed(() => conditionSource.value?.config?.options || [])

function onConditionField(val) {
  if (!val) { draft.visible_field_id = null; draft.visible_value = null; return }
  draft.visible_field_id = val
  const src = conditionSourceFields.value.find((f) => f.id === val)
  draft.visible_value = src?.type === 'checkbox' ? 'true' : (draft.visible_value || null)
}

function openField(i) {
  fieldIndex.value = i
  const src = i === -1
    ? { id: 0, label: '', type: 'text', config: defaultConfig('text'), col_span: 1, row_span: 1, show_in_table: true, show_in_card: true, visible_field_id: null, visible_value: null }
    : editFields.value[i]
  Object.assign(draft, {
    _k: src._k ?? 0,
    id: src.id || 0,
    label: src.label || '',
    type: src.type || 'text',
    config: deepConfig(src.type, src.config),
    col_span: clampSpan(src.col_span),
    row_span: clampSpan(src.row_span),
    show_in_table: src.show_in_table !== false,
    show_in_card: src.show_in_card !== false,
    visible_field_id: src.visible_field_id ?? null,
    visible_value: src.visible_value ?? null,
  })
  fieldOpen.value = true
}
function deepConfig(type, config) {
  const base = { ...defaultConfig(type), ...(config || {}) }
  if (Array.isArray(base.options)) base.options = [...base.options]
  return base
}
function onDraftType() { Object.assign(draft, { config: defaultConfig(draft.type) }) }

function applyField() {
  if (!draft.label.trim()) { notif.error('Укажите название поля'); return }
  // Поле не может быть источником условия само для себя — sanity на тип источника.
  let visibleFieldId = draft.visible_field_id ?? null
  let visibleValue = draft.visible_value ?? null
  if (visibleFieldId) {
    const src = conditionSourceFields.value.find((f) => f.id === visibleFieldId)
    if (!src) { visibleFieldId = null; visibleValue = null }
    else if (src.type === 'checkbox') visibleValue = 'true'
    else if (src.type === 'select' && !visibleValue) {
      notif.error('Выберите значение для условия показа'); return
    }
  } else {
    visibleValue = null
  }
  const field = {
    _k: draft._k || ++keySeq,
    id: draft.id || 0,
    label: draft.label.trim(),
    type: draft.type,
    config: cleanConfig(draft),
    col_span: draft.col_span,
    row_span: draft.row_span,
    show_in_table: draft.show_in_table,
    show_in_card: draft.show_in_card,
    visible_field_id: visibleFieldId,
    visible_value: visibleValue,
  }
  if (fieldIndex.value === -1) editFields.value.push(field)
  else editFields.value[fieldIndex.value] = field
  dirty.value = true
  fieldOpen.value = false
}
function cleanConfig(f) {
  if (f.type === 'select') {
    return { options: (f.config.options || []).map((s) => s.trim()).filter(Boolean), multiple: !!f.config.multiple }
  }
  if (f.type === 'number') return { pattern: (f.config.pattern || '').trim() }
  if (f.type === 'text') return { multiline: !!f.config.multiline }
  if (f.type === 'datetime') return { year: !!f.config.year, month_day: !!f.config.month_day, time: !!f.config.time }
  return {}
}

// ── Сохранение / CRUD календаря ──
async function save() {
  if (!editName.value.trim()) { notif.error('Укажите название календаря'); return }
  for (const f of editFields.value) {
    if (!f.label.trim()) { notif.error('У каждого поля должно быть название'); return }
  }
  saving.value = true
  try {
    if (editName.value.trim() !== current.value.name) {
      await api.updateCalendar(editId.value, editName.value.trim())
    }
    await api.replaceFields(editId.value, editFields.value.map((f) => ({
      id: f.id || 0, label: f.label.trim(), type: f.type, config: f.config,
      col_span: f.col_span, row_span: f.row_span, show_in_table: f.show_in_table, show_in_card: f.show_in_card,
      visible_field_id: f.visible_field_id ?? null, visible_value: f.visible_value ?? null,
    })))
    notif.success('Календарь сохранён')
    await store.fetchCalendars()
    const again = store.calendars.find((c) => c.id === editId.value)
    if (again) selectCalendar(again)
  } catch (e) {
    notif.error(e?.message || 'Не удалось сохранить календарь')
  } finally {
    saving.value = false
  }
}

function openCreate() { newName.value = ''; creating.value = true }
async function doCreate() {
  if (!newName.value.trim()) return
  saving.value = true
  try {
    const cal = await api.createCalendar(newName.value.trim())
    creating.value = false
    await store.fetchCalendars()
    const created = store.calendars.find((c) => c.id === cal.id)
    if (created) selectCalendar(created)
  } catch (e) {
    notif.error(e?.message || 'Не удалось создать календарь')
  } finally {
    saving.value = false
  }
}

async function doDelete() {
  confirmDelete.value = false
  try {
    await api.deleteCalendar(editId.value)
    editId.value = null
    await store.fetchCalendars()
    notif.success('Календарь удалён')
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить календарь')
  }
}

onMounted(() => store.fetchCalendars())
</script>

<style scoped>
.cm { flex: 1; min-height: 0; width: 100%; display: flex; flex-direction: column; gap: 16px; }
.ctl { width: 100%; padding: 10px 12px; font: inherit; appearance: none; }

.cm-head { flex: none; display: flex; flex-direction: column; gap: 4px; }
.cm-title { margin: 0; font-size: 17px; font-weight: 700; color: var(--color-text); }
.cm-sub { margin: 0; font-size: 13px; color: var(--color-text-dim); max-width: 640px; }

.cm-body { flex: 1; min-height: 0; display: flex; gap: 16px; align-items: stretch; }

.cm-side {
  width: 240px; flex-shrink: 0;
  display: flex; flex-direction: column; min-height: 0;
  border: 1px solid var(--color-outline-dim); border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg); overflow: hidden;
}
.cm-side-head {
  flex: none;
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 14px; font-weight: 700; font-size: 14px; color: var(--color-text);
  border-bottom: 1px solid var(--color-outline-dim);
}
.cm-add-circle {
  width: 30px; height: 30px; display: grid; place-items: center;
  border: none; border-radius: var(--radius-full);
  background: var(--color-primary); color: var(--color-on-primary); cursor: pointer;
}
.cm-add-circle:hover { filter: brightness(1.05); }
.cm-add-circle .material-symbols-outlined { font-size: 20px; }
.cm-side-list { flex: 1; min-height: 0; overflow-y: auto; padding: 8px; display: flex; flex-direction: column; gap: 2px; }
.cm-side-note { padding: 20px 12px; color: var(--color-text-dim); font-size: 13px; text-align: center; }
.cm-side-item {
  display: flex; align-items: center; gap: 10px; width: 100%;
  padding: 10px 12px; border: none; background: none; cursor: pointer;
  border-radius: var(--radius-md); color: var(--color-text-dim);
  text-align: left; font-size: 14px; font-weight: 600;
}
.cm-side-item:hover { background: var(--glass-hover-bg); color: var(--color-text); }
.cm-side-item.active { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.cm-side-item .material-symbols-outlined { font-size: 20px; flex-shrink: 0; }
.cm-side-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.cm-detail {
  flex: 1; min-width: 0; min-height: 0;
  display: flex; flex-direction: column;
  border: 1px solid var(--color-outline-dim); border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg); overflow: hidden;
}
.cm-detail-head {
  flex: none; display: flex; align-items: center; gap: 10px;
  padding: 12px 14px; border-bottom: 1px solid var(--color-outline-dim);
}
.cm-name { flex: 1; font-weight: 700; }

.cm-fields { flex: 1; min-height: 0; overflow-y: auto; padding: 10px 12px; }
.cm-field-rows { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.cm-field-row {
  display: flex; align-items: center; gap: 12px;
  padding: 10px 12px; cursor: pointer;
  border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md);
  background: var(--acrylic-card-bg);
  transition: border-color .12s, background .12s;
}
.cm-field-row:hover { background: var(--glass-hover-bg); border-color: var(--color-outline); }

.cm-reorder { display: flex; flex-direction: column; gap: 2px; flex-shrink: 0; }
.cm-reorder-btn {
  width: 24px; height: 18px; display: grid; place-items: center;
  border: none; border-radius: var(--radius-sm); background: transparent;
  color: var(--color-text-dim); cursor: pointer;
}
.cm-reorder-btn:disabled { opacity: 0.25; cursor: default; }
.cm-reorder-btn:not(:disabled):hover { background: var(--color-surface-low); color: var(--color-text); }
.cm-reorder-btn .material-symbols-outlined { font-size: 18px; }

.cm-field-icon {
  width: 36px; height: 36px; flex-shrink: 0; display: grid; place-items: center;
  border-radius: var(--radius-md); background: var(--color-primary-container); color: var(--color-on-primary-container);
}
.cm-field-icon .material-symbols-outlined { font-size: 20px; }

.cm-field-main { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.cm-field-label { font-size: 14px; font-weight: 600; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.cm-field-type { font-size: 12px; color: var(--color-text-dim); }

.cm-field-meta { display: flex; align-items: center; gap: 6px; flex-shrink: 0; }
.cm-badge {
  display: inline-flex; align-items: center; justify-content: center; gap: 2px;
  min-width: 34px; height: 24px; padding: 0 8px;
  border-radius: var(--radius-full); background: var(--color-surface-low);
  color: var(--color-text-dim); font-size: 12px; font-weight: 700;
}
.cm-badge.col { min-width: 24px; padding: 0 6px; }
.cm-badge.col .material-symbols-outlined { font-size: 16px; }
.cm-badge.col.off { color: var(--color-error); }

.cm-field-actions { display: flex; align-items: center; gap: 4px; flex-shrink: 0; }

.cm-fields-empty {
  height: 100%; display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 10px; color: var(--color-text-dim);
}
.cm-fields-empty .material-symbols-outlined { font-size: 44px; }
.cm-fields-empty p { margin: 0; font-size: 14px; text-align: center; }

.cm-detail-foot {
  flex: none; display: flex; align-items: center; gap: 12px;
  padding: 12px 14px; border-top: 1px solid var(--color-outline-dim);
}
.cm-foot-spacer { flex: 1; }
.cm-dirty { display: inline-flex; align-items: center; gap: 6px; font-size: 12px; color: var(--color-text-dim); }
.cm-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--color-warning, var(--color-primary)); }

.cm-detail-empty {
  flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 8px; color: var(--color-text-dim);
}
.cm-detail-empty .material-symbols-outlined { font-size: 48px; }
.cm-detail-empty p { margin: 0; font-size: 14px; }

/* ── Диалог поля ── */
.fd { display: flex; flex-direction: column; gap: 16px; }
.fd-field { display: flex; flex-direction: column; gap: 6px; }
.fd-label { font-size: 13px; font-weight: 600; color: var(--color-text-dim); }
.fd-hint { font-size: 12px; color: var(--color-text-dim); }
.fd :deep(.p-select) { width: 100%; }
.fd-type-opt { display: inline-flex; align-items: center; gap: 8px; }
.fd-type-opt .material-symbols-outlined { font-size: 18px; color: var(--color-text-dim); }
.fd-check { display: inline-flex; align-items: center; gap: 8px; font-size: 14px; color: var(--color-text); cursor: pointer; }
.fd-inline { display: flex; flex-wrap: wrap; gap: 16px; }
.fd-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.fd-options { display: flex; flex-direction: column; gap: 8px; }
.fd-option { display: flex; align-items: center; gap: 8px; }
.fd-cond { padding: 12px; border: 1px dashed var(--color-outline-dim); border-radius: var(--radius-md); }
.fd-cond-val { display: flex; flex-direction: column; gap: 6px; margin-top: 8px; }

.fd-seg { display: inline-flex; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); overflow: hidden; width: max-content; }
.fd-seg button {
  width: 44px; height: 38px; border: none; background: var(--acrylic-card-bg);
  color: var(--color-text-dim); cursor: pointer; font-weight: 700; font-size: 14px;
  border-right: 1px solid var(--color-outline-dim);
}
.fd-seg button:last-child { border-right: none; }
.fd-seg button.active { background: var(--grad-primary); color: var(--color-on-primary); }

/* ── Кнопки ── */
.cm-icon-btn {
  width: 36px; height: 36px; flex-shrink: 0; display: grid; place-items: center;
  border: none; border-radius: var(--radius-full);
  background: var(--color-surface-low); color: var(--color-text-dim); cursor: pointer;
}
.cm-icon-btn:hover { background: var(--color-surface-high); color: var(--color-text); }
.cm-icon-btn.sm { width: 32px; height: 32px; }
.cm-icon-btn.sm .material-symbols-outlined { font-size: 18px; }
.cm-icon-btn.danger { color: var(--color-error); }

.cm-btn-primary {
  display: inline-flex; align-items: center; gap: 6px; height: 38px; padding: 0 18px;
  border: none; border-radius: var(--radius-full);
  background: var(--color-primary); color: var(--color-on-primary);
  font-weight: 600; font-size: 14px; cursor: pointer;
}
.cm-btn-primary:disabled { opacity: 0.5; cursor: default; }
.cm-btn-tonal {
  display: inline-flex; align-items: center; gap: 6px; height: 38px; padding: 0 16px;
  border: none; border-radius: var(--radius-full);
  background: var(--color-primary-container); color: var(--color-on-primary-container);
  font-weight: 600; font-size: 14px; cursor: pointer;
}
.cm-btn-text {
  display: inline-flex; align-items: center; gap: 4px; align-self: flex-start;
  border: none; background: none; cursor: pointer; color: var(--color-primary); font-weight: 600; font-size: 14px;
}
.spin { animation: cmspin 1s linear infinite; }
@keyframes cmspin { to { transform: rotate(360deg); } }

@media (max-width: 768px) {
  .cm-body { flex-direction: column; }
  .cm-side { width: 100%; max-height: 30vh; }
  .fd-grid { grid-template-columns: 1fr; }
}
</style>
