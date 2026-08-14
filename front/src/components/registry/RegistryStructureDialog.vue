<template>
  <AppDialog
    :model-value="modelValue"
    :title="registry ? 'Настройка реестра' : 'Новый реестр'"
    subtitle="Поля карточки, их порядок и ширина"
    size="lg"
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: registry ? 'Сохранить' : 'Создать', loading: busy, disabled: !canSave },
    ]"
    @update:model-value="$emit('update:modelValue', $event)"
    @cancel="close"
    @confirm="submit"
  >
    <AppStack :gap="14">
      <div class="rs-field">
        <span class="rs-field-label">Название реестра</span>
        <InputText v-model="name" placeholder="Например, «Оборудование»" maxlength="120" />
      </div>

      <AppSwitchRow
        v-model="accounting"
        title="Учётный реестр"
        hint="У записей появится выдача под ответственного: кому, до какого числа и история всех движений."
      />

      <AppRow
        v-if="selectOptions.length"
        title="Подразделы"
        hint="Варианты выбранного списка станут вкладками над таблицей."
        inline
      >
        <Select
          v-model="sectionFieldId"
          :options="sectionChoices"
          option-label="label"
          option-value="value"
          placeholder="Выключены"
        />
      </AppRow>

      <div class="rs-fields">
        <div class="rs-head">
          <span class="rs-head-title">Поля карточки</span>
          <span class="rs-head-hint">Перетащите за рукоятку, чтобы поменять порядок</span>
        </div>

        <!-- Карточка записи в четвертях: поле тянут за правый край, ширина
             прилипает к четвертям сетки. Это тот же col_span, что и в
             сегментах ниже, — просто видно, что получится. -->
        <div v-if="fields.length" ref="gridEl" class="rs-preview">
          <div
            v-for="(f, i) in fields"
            :key="f.key"
            class="rs-cell"
            :class="{ active: resizing === i }"
            :style="{ gridColumn: `span ${f.col_span}` }"
            :title="f.label || 'Без названия'"
          >
            <span class="material-symbols-outlined rs-cell-icon">{{ fieldIcon(f.type) }}</span>
            <span class="rs-cell-label">{{ f.label || 'Без названия' }}</span>
            <span
              class="rs-cell-grip"
              title="Потяните, чтобы изменить ширину"
              @pointerdown.stop.prevent="startResize(i, $event)"
            />
          </div>
        </div>

        <EmptyState
          v-if="!fields.length"
          size="sm"
          icon="table_rows"
          title="Полей пока нет"
          subtitle="Добавьте первое — из него сложится карточка записи."
        />

        <!-- Сетка в четыре четверти: ширина поля задаётся сегментами 1..4, а
             порядок — перетаскиванием. Каждое поле рисуется на своей строке
             редактора, чтобы правка не зависела от того, как оно легло. -->
        <ul v-else class="rs-list">
          <li
            v-for="(f, i) in fields"
            :key="f.key"
            class="rs-item"
            :class="{ dragging: dragIndex === i, over: overIndex === i }"
            @dragover.prevent="onDragOver(i)"
            @drop.prevent="onDrop(i)"
            @dragleave="onDragLeave(i)"
          >
            <button
              class="rs-grip"
              type="button"
              draggable="true"
              title="Перетащить"
              aria-label="Перетащить поле"
              @dragstart="onDragStart(i, $event)"
              @dragend="onDragEnd"
            >
              <span class="material-symbols-outlined">drag_indicator</span>
            </button>

            <div class="rs-body">
              <div class="rs-row">
                <InputText v-model="f.label" class="rs-label" placeholder="Название поля" maxlength="120" />
                <Select
                  v-model="f.type"
                  class="rs-type"
                  :options="typeOptions"
                  option-label="label"
                  option-value="value"
                  @change="onTypeChange(f)"
                />
                <AppButton
                  variant="icon"
                  size="sm"
                  tone="danger"
                  icon="delete"
                  title="Удалить поле"
                  aria-label="Удалить поле"
                  @click="removeField(i)"
                />
              </div>

              <div class="rs-row rs-row-tools">
                <!-- Ширина в карточке: сколько четвертей строки занимает поле. -->
                <span class="rs-tool-label">Ширина</span>
                <AppTabs
                  v-model="f.col_span"
                  :tabs="SPANS"
                  variant="tint"
                  dense
                />
                <AppSwitch v-model="f.show_in_table" label="В таблице" />
                <AppSwitch
                  v-if="isQrCapable(f.type)"
                  v-model="f.config.qr"
                  label="QR-код"
                />
              </div>

              <!-- Настройки конкретного типа. -->
              <div v-if="f.type === 'select'" class="rs-row rs-row-tools">
                <InputText
                  :model-value="(f.config.options || []).join(', ')"
                  class="rs-label"
                  placeholder="Варианты через запятую"
                  @update:model-value="setOptions(f, $event)"
                />
                <AppSwitch v-model="f.config.multiple" label="Несколько" />
              </div>

              <div v-else-if="f.type === 'checkbox'" class="rs-row rs-row-tools">
                <InputText v-model="f.config.on_label" class="rs-half" placeholder="Когда отмечен: Да" maxlength="40" />
                <InputText v-model="f.config.off_label" class="rs-half" placeholder="Когда снят: Нет" maxlength="40" />
              </div>

              <div v-else-if="f.type === 'number'" class="rs-row rs-row-tools">
                <InputText v-model="f.config.min" class="rs-num" placeholder="Минимум" />
                <InputText v-model="f.config.max" class="rs-num" placeholder="Максимум" />
                <InputText v-model="f.config.pattern" class="rs-label" placeholder="Шаблон (регулярка), необязательно" />
              </div>

              <div v-else-if="f.type === 'regex'" class="rs-row rs-row-tools">
                <InputText v-model="f.config.pattern" class="rs-label" placeholder="Регулярное выражение, например ^[A-Z]{2}-\d{3}$" />
                <InputText v-model="f.config.hint" class="rs-half" placeholder="Подсказка человеку" maxlength="80" />
              </div>

              <div v-else-if="f.type === 'datetime'" class="rs-row rs-row-tools">
                <span class="rs-tool-label">Показывать</span>
                <AppSwitch v-model="f.config.day" label="День" />
                <AppSwitch v-model="f.config.month" label="Месяц" />
                <AppSwitch v-model="f.config.year" label="Год" />
                <AppSwitch v-model="f.config.hours" label="Часы" />
                <AppSwitch v-model="f.config.minutes" label="Минуты" />
                <AppSwitch v-model="f.config.seconds" label="Секунды" />
              </div>
            </div>
          </li>
        </ul>

        <AppButton variant="glass" icon="add" label="Добавить поле" @click="addField" />
      </div>
    </AppStack>
  </AppDialog>
</template>

<script setup>
/* Создание реестра и правка его структуры — одна форма: набор полей, их
   порядок и ширина в карточке.

   Порядок меняется перетаскиванием за рукоятку (а не за всё поле: иначе нельзя
   было бы выделить текст в названии). Перетаскивается ИНДЕКС, а не элемент, и
   массив пересобирается целиком — так не бывает состояния «поле исчезло, но не
   вставилось», из-за которого прежний редактор терял поля при быстром броске. */
import { computed, ref, watch } from 'vue'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import AppButton from '@/components/ui/AppButton.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppSwitch from '@/components/ui/AppSwitch.vue'
import AppSwitchRow from '@/components/ui/AppSwitchRow.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { FIELD_TYPES, GRID_COLS, defaultConfig, fieldIcon, isQrCapable } from '@/utils/registryFields.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  /** null — создание нового реестра. */
  registry: { type: Object, default: null },
  /** save({ name, accounting, section_field_id, fields }) → Promise. */
  save: { type: Function, required: true },
})
const emit = defineEmits(['update:modelValue', 'error'])

const typeOptions = FIELD_TYPES.map((f) => ({ label: f.label, value: f.type }))
const SPANS = [
  { value: 1, label: '¼' },
  { value: 2, label: '½' },
  { value: 3, label: '¾' },
  { value: 4, label: '1' },
]

const name = ref('')
const accounting = ref(false)
const sectionFieldId = ref(null)
const fields = ref([])
const busy = ref(false)

// key — устойчивый ключ строки редактора: у новых полей id ещё нет, а без него
// Vue переиспользовал бы DOM соседней строки при перетаскивании.
let keySeq = 0
function toDraft(f) {
  return {
    key: `f${++keySeq}`,
    id: f?.id || 0,
    label: f?.label || '',
    type: f?.type || 'text',
    config: { ...defaultConfig(f?.type || 'text'), ...(f?.config || {}) },
    col_span: f?.col_span || 1,
    row_span: f?.row_span || 1,
    show_in_table: f?.show_in_table !== false,
  }
}

watch(() => props.modelValue, (open) => {
  if (!open) return
  name.value = props.registry?.name || ''
  accounting.value = !!props.registry?.accounting
  sectionFieldId.value = props.registry?.section_field_id || null
  fields.value = (props.registry?.fields || []).map(toDraft)
}, { immediate: true })

const canSave = computed(() =>
  name.value.trim().length > 0 && fields.value.every((f) => f.label.trim().length > 0))

// Источником подразделов может стать только списковое поле — остальным неоткуда
// взять набор вкладок.
const selectOptions = computed(() => fields.value.filter((f) => f.type === 'select' && f.id))
const sectionChoices = computed(() => [
  { label: 'Выключены', value: null },
  ...selectOptions.value.map((f) => ({ label: f.label || 'Без названия', value: f.id })),
])

function addField() {
  fields.value = [...fields.value, toDraft({ type: 'text' })]
}

function removeField(i) {
  const removed = fields.value[i]
  fields.value = fields.value.filter((_, idx) => idx !== i)
  // Поле-источник подразделов удалили — настройку выключаем сами, иначе она
  // ссылалась бы в никуда.
  if (removed?.id && sectionFieldId.value === removed.id) sectionFieldId.value = null
}

// Смена типа сбрасывает настройки прежнего: они относились к другому типу и в
// новом означали бы случайное.
function onTypeChange(f) {
  f.config = defaultConfig(f.type)
}

function setOptions(f, raw) {
  f.config.options = String(raw).split(',').map((s) => s.trim()).filter(Boolean)
}

/* ── Вытягивание ширины ──
   Ширину считаем от ФАКТИЧЕСКОЙ ширины сетки, а не от стартовой точки: доля
   четверти зависит от того, насколько широк диалог, и жёсткий шаг в пикселях
   врал бы на узком экране. Указатель захватываем (setPointerCapture) — иначе
   быстрый жест «убегает» из элемента и тяга обрывается на середине. */
const gridEl = ref(null)
const resizing = ref(null)

function startResize(index, e) {
  const grid = gridEl.value
  if (!grid) return
  const rect = grid.getBoundingClientRect()
  const step = rect.width / GRID_COLS
  resizing.value = index

  const move = (ev) => {
    const span = Math.round((ev.clientX - cellLeft(index, rect, step)) / step)
    fields.value[index].col_span = Math.min(GRID_COLS, Math.max(1, span))
  }
  const up = () => {
    resizing.value = null
    window.removeEventListener('pointermove', move)
    window.removeEventListener('pointerup', up)
  }
  window.addEventListener('pointermove', move)
  window.addEventListener('pointerup', up)
}

// Левый край поля в сетке: сумма ширин предыдущих полей с переносом по строкам.
function cellLeft(index, rect, step) {
  let col = 0
  for (let i = 0; i < index; i++) {
    const span = fields.value[i].col_span
    if (col + span > GRID_COLS) col = 0
    col += span
  }
  if (col + fields.value[index].col_span > GRID_COLS) col = 0
  return rect.left + col * step
}

// ── Перетаскивание ──
const dragIndex = ref(null)
const overIndex = ref(null)

function onDragStart(i, e) {
  dragIndex.value = i
  e.dataTransfer.effectAllowed = 'move'
  // Safari не начинает перетаскивание без данных в буфере.
  e.dataTransfer.setData('text/plain', String(i))
}

function onDragOver(i) {
  if (dragIndex.value !== null && i !== dragIndex.value) overIndex.value = i
}

function onDragLeave(i) {
  if (overIndex.value === i) overIndex.value = null
}

function onDrop(to) {
  const from = dragIndex.value
  onDragEnd()
  if (from === null || from === to) return
  const next = [...fields.value]
  const [moved] = next.splice(from, 1)
  next.splice(to, 0, moved)
  fields.value = next
}

function onDragEnd() {
  dragIndex.value = null
  overIndex.value = null
}

function close() {
  emit('update:modelValue', false)
}

async function submit() {
  if (!canSave.value || busy.value) return
  busy.value = true
  try {
    await props.save({
      name: name.value.trim(),
      accounting: accounting.value,
      section_field_id: sectionFieldId.value,
      fields: fields.value.map((f) => ({
        id: f.id,
        label: f.label.trim(),
        type: f.type,
        config: f.config,
        col_span: f.col_span,
        row_span: f.row_span,
        show_in_table: f.show_in_table,
      })),
    })
    close()
  } catch (e) {
    emit('error', e?.message || 'Не удалось сохранить реестр')
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.rs-field { display: flex; flex-direction: column; gap: 6px; }
.rs-field-label { font-size: 13px; color: var(--color-text-dim); }

.rs-fields { display: flex; flex-direction: column; gap: 10px; }

.rs-head { display: flex; align-items: baseline; gap: 10px; flex-wrap: wrap; }
.rs-head-title { font-size: 14px; font-weight: 600; }
.rs-head-hint { font-size: 12px; color: var(--color-text-dim); }

/* Предпросмотр карточки: та же сетка в четвертях, что и у записи. */
.rs-preview {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 6px;
  padding: 10px;
  border: 1px dashed var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
}

.rs-cell {
  position: relative;
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  height: 40px;
  padding: 0 14px 0 10px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-sm);
  background: var(--acrylic-card-bg);
  font-size: 12.5px;
}

.rs-cell.active { border-color: var(--color-primary); }
.rs-cell-icon { font-size: 16px; color: var(--color-text-dim); }

.rs-cell-label {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Рукоятка у правого края — за неё поле и тянут. Шире своей полоски, чтобы в
   неё попадали пальцем, а не только курсором. */
.rs-cell-grip {
  position: absolute;
  top: 0;
  right: 0;
  width: 12px;
  height: 100%;
  cursor: col-resize;
  touch-action: none;
}

.rs-cell-grip::after {
  content: '';
  position: absolute;
  top: 50%;
  right: 4px;
  width: 2px;
  height: 16px;
  border-radius: 1px;
  background: var(--color-outline-dim);
  transform: translateY(-50%);
}

.rs-cell:hover .rs-cell-grip::after,
.rs-cell.active .rs-cell-grip::after { background: var(--color-primary); }

.rs-list { display: flex; flex-direction: column; gap: 8px; margin: 0; padding: 0; list-style: none; }

.rs-item {
  display: flex;
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: var(--acrylic-card-bg);
}

.rs-item.dragging { opacity: 0.5; }
.rs-item.over { border-color: var(--color-primary); }

.rs-grip {
  display: flex;
  align-items: flex-start;
  padding: 4px 0 0;
  border: none;
  background: none;
  color: var(--color-text-dim);
  cursor: grab;
}

.rs-grip:active { cursor: grabbing; }

.rs-body { display: flex; flex: 1; min-width: 0; flex-direction: column; gap: 8px; }

.rs-row { display: flex; gap: 8px; align-items: center; }
.rs-row-tools { flex-wrap: wrap; }
.rs-tool-label { font-size: 12px; color: var(--color-text-dim); }

/* min-width: 0 обязателен — иначе поле не сожмётся уже своего содержимого и
   диалог поедет горизонтальной прокруткой. */
.rs-label { flex: 1; min-width: 0; }
.rs-type { width: 170px; flex: none; }
.rs-half { flex: 1; min-width: 0; }
.rs-num { width: 120px; flex: none; }

@container (max-width: 620px) {
  .rs-row { flex-wrap: wrap; }
  .rs-type { width: 100%; }
}

@media (max-width: 620px) {
  .rs-row { flex-wrap: wrap; }
  .rs-type { width: 100%; }
}
</style>
