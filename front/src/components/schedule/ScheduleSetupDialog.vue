<template>
  <AppDialog
    :model-value="modelValue"
    title="Настройки расписания"
    size="lg"
    :busy="saving"
    @update:model-value="close"
  >
    <AppTabs
      v-model="tab"
      :tabs="[
        { value: 'cycle', label: 'Цикл', icon: 'repeat' },
        { value: 'categories', label: 'Категории', icon: 'palette' },
        { value: 'fields', label: 'Поля', icon: 'view_list' },
      ]"
    />

    <!-- Цикл: его длина и названия недель. -->
    <AppStack v-if="tab === 'cycle'" :gap="14" class="ss-pane">
      <div class="ss-field">
        <label class="ss-label">Название</label>
        <input v-model="form.name" class="ctl" type="text" maxlength="120" />
      </div>

      <div class="ss-row">
        <div class="ss-field">
          <label class="ss-label">Недель в цикле</label>
          <select v-model.number="form.cycle_weeks" class="ctl">
            <option v-for="n in MAX_CYCLE_WEEKS" :key="n" :value="n">{{ n }}</option>
          </select>
        </div>
        <div class="ss-field">
          <label class="ss-label">Первая неделя цикла</label>
          <input v-model="form.cycle_anchor" class="ctl" type="date" />
        </div>
      </div>

      <!-- Названия недель: «Числитель» и «Знаменатель» при цикле из двух,
           «А/Б/В» при трёх. Пустое поле подписывается «Неделя N». -->
      <div v-if="form.cycle_weeks > 1" class="ss-field">
        <label class="ss-label">Как называть недели</label>
        <div class="ss-labels">
          <input
            v-for="n in form.cycle_weeks"
            :key="n"
            v-model="form.week_labels[n - 1]"
            class="ctl"
            type="text"
            maxlength="40"
            :placeholder="`Неделя ${n}`"
          />
        </div>
        <AppButton
          v-if="form.cycle_weeks === 2"
          variant="text"
          label="Числитель и знаменатель"
          @click="form.week_labels = ['Числитель', 'Знаменатель']"
        />
      </div>

      <div class="ss-field">
        <label class="ss-label">Часовой пояс</label>
        <select v-model="form.timezone" class="ctl">
          <option v-for="tz in TIMEZONES" :key="tz.value" :value="tz.value">{{ tz.label }}</option>
        </select>
        <span class="ss-hint">По нему считаются «сегодня» и линия «сейчас».</span>
      </div>

      <AppInfoBar
        v-if="cycleShrinks"
        tone="warning"
        message="Цикл станет короче: занятия, привязанные к исчезающим неделям, станут еженедельными."
      />
    </AppStack>

    <!-- Категории: они же раскраска блоков на шкале. -->
    <AppStack v-else-if="tab === 'categories'" :gap="10" class="ss-pane">
      <EmptyState
        v-if="!categories.length"
        size="sm"
        icon="palette"
        title="Категорий нет"
        subtitle="Категория красит занятие и собирает похожие вместе."
      />
      <div v-for="c in categories" :key="c.id" class="ss-cat">
        <input
          :value="c.name"
          class="ctl"
          type="text"
          maxlength="80"
          @change="renameCategory(c, $event.target.value)"
        />
        <ColorSwatchPicker
          :model-value="c.color"
          @update:model-value="recolorCategory(c, $event)"
        />
        <AppButton
          variant="icon"
          icon="delete"
          tone="danger"
          label="Удалить категорию"
          @click="dropCategory(c)"
        />
      </div>
      <div class="ss-cat-new">
        <input
          v-model="newCategory"
          class="ctl"
          type="text"
          maxlength="80"
          placeholder="Например, Лекция"
          @keyup.enter="addCategory"
        />
        <AppButton variant="glass" icon="add" label="Добавить" @click="addCategory" />
      </div>
    </AppStack>

    <!-- Поля карточки занятия. -->
    <AppStack v-else :gap="10" class="ss-pane">
      <EmptyState
        v-if="!fields.length"
        size="sm"
        icon="view_list"
        title="Своих полей нет"
        subtitle="Поле показывает у занятия что-то ещё: аудиторию, преподавателя, ссылку."
      />
      <div v-for="(f, i) in fields" :key="f.key" class="ss-field-row">
        <input v-model="f.label" class="ctl ss-field-label" type="text" maxlength="120" placeholder="Название поля" />
        <select v-model="f.type" class="ctl ss-field-type">
          <option v-for="t in FIELD_TYPES" :key="t.value" :value="t.value">{{ t.label }}</option>
        </select>
        <AppSwitch v-model="f.show_on_block" label="На шкале" />
        <div class="ss-field-tools">
          <AppButton variant="icon" icon="arrow_upward" label="Выше" :disabled="i === 0" @click="moveField(i, -1)" />
          <AppButton variant="icon" icon="arrow_downward" label="Ниже" :disabled="i === fields.length - 1" @click="moveField(i, 1)" />
          <AppButton variant="icon" icon="delete" tone="danger" label="Удалить поле" @click="dropField(i)" />
        </div>
        <!-- Варианты списка задаются построчно: их правит тот же человек, что
             и заводит поле, и отдельный экран здесь только мешает. -->
        <textarea
          v-if="f.type === 'select'"
          v-model="f.optionsText"
          class="ctl ss-field-options"
          rows="2"
          placeholder="Варианты, по одному на строку"
        />
      </div>
      <AppButton variant="glass" icon="add" label="Добавить поле" @click="addField" />
    </AppStack>

    <span v-if="error" class="ss-error">{{ error }}</span>

    <template #footer>
      <AppButton
        variant="glass"
        tone="danger"
        icon="delete"
        label="Удалить расписание"
        :disabled="saving"
        @click="confirmDelete = true"
      />
      <span class="ss-spacer" />
      <AppButton label="Отмена" :disabled="saving" @click="close(false)" />
      <AppButton variant="filled" label="Сохранить" :loading="saving" @click="save" />
    </template>

    <ConfirmDialog
      :visible="confirmDelete"
      header="Удалить расписание?"
      message="Занятия, категории и выданные ссылки исчезнут безвозвратно."
      confirm-label="Удалить"
      danger-confirm
      @confirm="removeSchedule"
      @cancel="confirmDelete = false"
    />
  </AppDialog>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppSwitch from '@/components/ui/AppSwitch.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import ColorSwatchPicker from '@/components/common/ColorSwatchPicker.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { MAX_CYCLE_WEEKS, dateKey } from '@/utils/scheduleCycle.js'
import { TASK_COLOR_IDS } from '@/utils/taskColors.js'

// Палитра категорий — та же восьмёрка цветов-тегов (домен: domain.CategoryColors).
const CATEGORY_COLORS = TASK_COLOR_IDS

// Набор типов — ядро настраиваемых записей БЕЗ файловых: расписание файлов
// не держит вовсе.
const FIELD_TYPES = [
  { value: 'text', label: 'Текст' },
  { value: 'textarea', label: 'Длинный текст' },
  { value: 'number', label: 'Число' },
  { value: 'select', label: 'Список' },
  { value: 'checkbox', label: 'Галочка' },
  { value: 'link', label: 'Ссылка' },
  { value: 'phone', label: 'Телефон' },
  { value: 'email', label: 'Почта' },
  { value: 'datetime', label: 'Дата' },
  { value: 'regex', label: 'Текст по шаблону' },
]

const TIMEZONES = [
  { value: 'Europe/Kaliningrad', label: 'Калининград (МСК−1)' },
  { value: 'Europe/Moscow', label: 'Москва (МСК)' },
  { value: 'Europe/Samara', label: 'Самара (МСК+1)' },
  { value: 'Asia/Yekaterinburg', label: 'Екатеринбург (МСК+2)' },
  { value: 'Asia/Omsk', label: 'Омск (МСК+3)' },
  { value: 'Asia/Krasnoyarsk', label: 'Красноярск (МСК+4)' },
  { value: 'Asia/Irkutsk', label: 'Иркутск (МСК+5)' },
  { value: 'Asia/Yakutsk', label: 'Якутск (МСК+6)' },
  { value: 'Asia/Vladivostok', label: 'Владивосток (МСК+7)' },
  { value: 'Asia/Magadan', label: 'Магадан (МСК+8)' },
  { value: 'Asia/Kamchatka', label: 'Камчатка (МСК+9)' },
]

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  schedule: { type: Object, required: true },
  // Операции раздела: диалог ждёт их и показывает отказ сервера.
  submit: { type: Function, required: true },
  saveFields: { type: Function, required: true },
  addCategoryFn: { type: Function, required: true },
  updateCategoryFn: { type: Function, required: true },
  removeCategoryFn: { type: Function, required: true },
  removeFn: { type: Function, required: true },
})

const emit = defineEmits(['update:modelValue'])

const tab = ref('cycle')
const saving = ref(false)
const error = ref('')
const confirmDelete = ref(false)
const newCategory = ref('')

const form = reactive({
  name: '', cycle_weeks: 1, cycle_anchor: '', week_labels: [], timezone: 'Europe/Moscow',
})
const fields = ref([])

const categories = computed(() => props.schedule?.categories || [])
const cycleShrinks = computed(() => form.cycle_weeks < (props.schedule?.cycle_weeks || 1))

let fieldKeySeq = 0

watch(() => [props.modelValue, props.schedule], () => {
  if (!props.modelValue || !props.schedule) return
  error.value = ''
  tab.value = 'cycle'
  form.name = props.schedule.name || ''
  form.cycle_weeks = props.schedule.cycle_weeks || 1
  form.cycle_anchor = props.schedule.cycle_anchor || dateKey(new Date())
  form.week_labels = [...(props.schedule.week_labels || [])]
  form.timezone = props.schedule.timezone || 'Europe/Moscow'
  fields.value = (props.schedule.fields || []).map((f) => ({
    key: `f${(fieldKeySeq += 1)}`,
    id: f.id,
    label: f.label,
    type: f.type,
    config: { ...(f.config || {}) },
    col_span: f.col_span || 1,
    row_span: f.row_span || 1,
    show_on_block: !!f.show_on_block,
    show_in_card: f.show_in_card !== false,
    optionsText: (f.config?.options || []).join('\n'),
  }))
}, { immediate: true })

function close(value = false) {
  if (saving.value) return
  emit('update:modelValue', value)
}

// ── Категории: правятся сразу, по одной — так же, как справочники в реестрах.
async function addCategory() {
  const name = newCategory.value.trim()
  if (!name) return
  try {
    await props.addCategoryFn({ name, color: CATEGORY_COLORS[categories.value.length % CATEGORY_COLORS.length] })
    newCategory.value = ''
  } catch (e) {
    error.value = e?.message || 'Не удалось добавить категорию'
  }
}

async function renameCategory(category, name) {
  if (!name.trim() || name === category.name) return
  try {
    await props.updateCategoryFn(category.id, { name: name.trim(), color: category.color })
  } catch (e) {
    error.value = e?.message || 'Не удалось переименовать категорию'
  }
}

async function recolorCategory(category, color) {
  try {
    await props.updateCategoryFn(category.id, { name: category.name, color })
  } catch (e) {
    error.value = e?.message || 'Не удалось изменить цвет'
  }
}

async function dropCategory(category) {
  try {
    await props.removeCategoryFn(category.id)
  } catch (e) {
    error.value = e?.message || 'Не удалось удалить категорию'
  }
}

// ── Поля: правятся набором и сохраняются вместе с настройками.
function addField() {
  fields.value.push({
    key: `f${(fieldKeySeq += 1)}`, id: 0, label: '', type: 'text', config: {},
    col_span: 1, row_span: 1, show_on_block: false, show_in_card: true, optionsText: '',
  })
}

function dropField(index) { fields.value.splice(index, 1) }

function moveField(index, delta) {
  const next = index + delta
  if (next < 0 || next >= fields.value.length) return
  const [field] = fields.value.splice(index, 1)
  fields.value.splice(next, 0, field)
}

async function save() {
  saving.value = true
  error.value = ''
  try {
    await props.submit({
      name: form.name.trim(),
      cycle_weeks: form.cycle_weeks,
      cycle_anchor: form.cycle_anchor,
      week_labels: form.week_labels.slice(0, form.cycle_weeks),
      timezone: form.timezone,
    })
    await props.saveFields(fields.value
      .filter((f) => f.label.trim())
      .map((f) => ({
        id: f.id || 0,
        label: f.label.trim(),
        type: f.type,
        config: f.type === 'select'
          ? { ...f.config, options: f.optionsText.split('\n').map((s) => s.trim()).filter(Boolean) }
          : f.config,
        col_span: f.col_span,
        row_span: f.row_span,
        show_on_block: f.show_on_block,
        show_in_card: f.show_in_card,
      })))
    close(false)
  } catch (e) {
    error.value = e?.message || 'Не удалось сохранить настройки'
  } finally {
    saving.value = false
  }
}

async function removeSchedule() {
  confirmDelete.value = false
  saving.value = true
  try {
    await props.removeFn()
    close(false)
  } catch (e) {
    error.value = e?.message || 'Не удалось удалить расписание'
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.ss-pane { padding-top: 14px; }
.ss-field { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.ss-label { font-size: 12px; color: var(--color-on-surface-variant); }
.ss-hint { font-size: 12px; color: var(--color-on-surface-variant); }
.ss-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(180px, 100%), 1fr));
  gap: 10px;
}
.ss-labels {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(140px, 100%), 1fr));
  gap: 8px;
}

.ss-cat, .ss-cat-new {
  display: flex;
  align-items: center;
  gap: 8px;
}
.ss-cat .ctl, .ss-cat-new .ctl { flex: 1; min-width: 0; }

.ss-field-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 150px) auto auto;
  align-items: center;
  gap: 8px;
  padding: 8px;
  border-radius: var(--radius-sm);
  background: var(--color-surface-container-lowest);
}
.ss-field-options { grid-column: 1 / -1; }
.ss-field-tools { display: flex; gap: 2px; }
@media (max-width: 640px) {
  .ss-field-row { grid-template-columns: minmax(0, 1fr) auto; }
  .ss-field-type { grid-column: 1 / -1; }
}

.ss-error { display: block; padding-top: 10px; font-size: 13px; color: var(--color-error); }
.ss-spacer { flex: 1; }
</style>
