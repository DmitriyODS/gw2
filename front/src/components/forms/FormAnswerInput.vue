<template>
  <div class="fa">
    <!-- Пояснительный блок: ответа не требует, только текст и картинка. -->
    <template v-if="question.type === 'note'">
      <p v-if="question.config?.text" class="fa-note">{{ question.config.text }}</p>
    </template>

    <!-- Короткий ответ и абзац. -->
    <InputText
      v-else-if="question.type === 'short_text'"
      :model-value="value || ''"
      class="fa-wide"
      :disabled="disabled"
      placeholder="Ваш ответ"
      maxlength="1000"
      @update:model-value="emitValue($event)"
    />
    <Textarea
      v-else-if="question.type === 'paragraph'"
      :model-value="value || ''"
      class="fa-wide"
      :disabled="disabled"
      placeholder="Ваш ответ"
      rows="4"
      auto-resize
      maxlength="10000"
      @update:model-value="emitValue($event)"
    />

    <!-- Выпадающий список. -->
    <Select
      v-else-if="question.type === 'dropdown'"
      :model-value="value || null"
      class="fa-wide"
      :options="options"
      :disabled="disabled"
      placeholder="Выберите вариант"
      show-clear
      @update:model-value="emitValue($event || '')"
    />

    <!-- Один из списка и несколько из списка: один и тот же список строк,
         различается только значок состояния и правило выбора. Голых input тут
         нет намеренно — строка ядра ведёт себя одинаково на тач и мыши. -->
    <AppStack v-else-if="isChoice(question.type)" :gap="6">
      <AppRow
        v-for="opt in options"
        :key="opt"
        :title="opt"
        :icon="choiceIcon(opt)"
        dense
        clickable
        :disabled="disabled"
        :selected="isPicked(opt)"
        @click="pick(opt)"
      />
      <!-- «Другое»: свой вариант вписывается строкой и хранится как обычное
           значение — сервер принимает его, только если автор включил флажок. -->
      <div v-if="question.config?.other" class="fa-other">
        <AppRow
          title="Другое"
          :icon="choiceIcon(otherValue, true)"
          dense
          clickable
          :disabled="disabled"
          :selected="otherPicked"
          @click="pickOther"
        />
        <InputText
          v-model="otherText"
          class="fa-wide"
          :disabled="disabled"
          placeholder="Свой вариант"
          maxlength="500"
          @update:model-value="onOtherText"
        />
      </div>
    </AppStack>

    <!-- «Запись»: варианты с местами. Занятые не выбрать, остаток виден сразу —
         человек не должен узнавать о нехватке мест уже после отправки. -->
    <AppStack v-else-if="question.type === 'booking'" :gap="6">
      <AppRow
        v-for="slot in slots"
        :key="slot.option"
        :title="slot.option"
        :hint="slotHint(slot)"
        :icon="value === slot.option ? 'radio_button_checked' : 'radio_button_unchecked'"
        dense
        clickable
        :disabled="disabled || (!slot.left && value !== slot.option)"
        :tone="slot.left ? 'neutral' : 'danger'"
        :selected="value === slot.option"
        @click="pickSlot(slot)"
      >
        <AppChip
          size="sm"
          :tone="slot.left ? 'success' : 'error'"
          :label="slot.left ? `${slot.left} из ${slot.total}` : 'Мест нет'"
        />
      </AppRow>
    </AppStack>

    <!-- Линейная шкала. -->
    <div v-else-if="question.type === 'scale'" class="fa-scale">
      <span v-if="question.config?.min_label" class="fa-scale-label">
        {{ question.config.min_label }}
      </span>
      <div class="fa-scale-row">
        <AppButton
          v-for="n in scaleValues"
          :key="n"
          size="sm"
          :variant="Number(value) === n ? 'filled' : 'glass'"
          :label="String(n)"
          :disabled="disabled"
          @click="emitValue(Number(value) === n ? null : n)"
        />
      </div>
      <span v-if="question.config?.max_label" class="fa-scale-label">
        {{ question.config.max_label }}
      </span>
    </div>

    <!-- Оценка звёздами. -->
    <div v-else-if="question.type === 'rating'" class="fa-rating">
      <AppButton
        v-for="n in ratingValues"
        :key="n"
        variant="icon"
        size="sm"
        :icon="Number(value) >= n ? 'star' : 'star_outline'"
        :tone="Number(value) >= n ? 'primary' : 'neutral'"
        :disabled="disabled"
        :aria-label="`Оценка ${n}`"
        @click="emitValue(Number(value) === n ? null : n)"
      />
      <span v-if="value" class="fa-rating-value">{{ value }} из {{ ratingValues.length }}</span>
    </div>

    <!-- Сетки: строки против столбцов. Прокручивается сама — таблица шире
         панели не должна тащить за собой весь раздел. -->
    <div v-else-if="isGrid(question.type)" class="fa-grid-wrap">
      <table class="fa-grid">
        <thead>
          <tr>
            <th />
            <th v-for="col in gridCols" :key="col" class="fa-grid-col">{{ col }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in gridRows" :key="row">
            <th class="fa-grid-row">{{ row }}</th>
            <td v-for="col in gridCols" :key="col">
              <AppButton
                variant="icon"
                size="sm"
                :icon="gridIcon(row, col)"
                :tone="gridPicked(row, col) ? 'primary' : 'neutral'"
                :disabled="disabled"
                :aria-label="`${row} — ${col}`"
                @click="toggleGrid(row, col)"
              />
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Дата и время. -->
    <DatePicker
      v-else-if="question.type === 'date'"
      :model-value="dateValue"
      class="fa-date"
      :disabled="disabled"
      :show-time="!!question.config?.with_time"
      hour-format="24"
      date-format="dd.mm.yy"
      show-icon
      icon-display="input"
      placeholder="Выберите дату"
      @update:model-value="onDate"
    />
    <TimePicker
      v-else-if="question.type === 'time'"
      :model-value="value || null"
      :disabled="disabled"
      clearable
      @update:model-value="emitValue($event || '')"
    />

    <!-- Файлы. -->
    <AppStack v-else-if="question.type === 'file'" :gap="8">
      <AppRow
        v-for="(file, i) in files"
        :key="file.path || i"
        :title="file.name || 'Файл'"
        :hint="fileSize(file.size)"
        icon="draft"
        dense
      >
        <AppButton
          variant="icon" size="sm" tone="danger" icon="close"
          :disabled="disabled"
          title="Убрать файл" aria-label="Убрать файл"
          @click="removeFile(i)"
        />
      </AppRow>
      <AppButton
        v-if="files.length < maxFiles"
        variant="glass"
        icon="upload"
        :label="uploading ? `Загрузка… ${uploadPercent}%` : 'Приложить файл'"
        :disabled="disabled || uploading"
        :loading="uploading"
        @click="pickFile"
      />
      <small class="fa-hint">
        До {{ maxFiles }} файлов, каждый не больше {{ Math.round(maxSize / 1024 / 1024) }} МБ
      </small>
    </AppStack>

    <small v-if="error" class="fa-error">{{ error }}</small>
  </div>
</template>

<script setup>
/* Ввод ответа на один вопрос — общий для раздела и публичной страницы формы.

   Компонент ничего не знает про то, куда уедет ответ: значение приходит и
   уходит через v-model, загрузка файла — через переданную функцию upload
   (в разделе она чанковая, по внешней ссылке — одним запросом). */
import { computed, ref, watch } from 'vue'
import DatePicker from 'primevue/datepicker'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import Textarea from 'primevue/textarea'
import AppButton from '@/components/ui/AppButton.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import TimePicker from '@/components/common/TimePicker.vue'
import { bookingSlots, fileCount, fileLimit, isChoice, isGrid } from '@/utils/formFields.js'

const props = defineProps({
  question: { type: Object, required: true },
  modelValue: { type: [String, Number, Array, Object, null], default: null },
  disabled: { type: Boolean, default: false },
  error: { type: String, default: '' },
  /** upload(file) → Promise<{path, name, mime, size}>. */
  upload: { type: Function, default: null },
  /** Занятые места «Записи»: {вариант: занято} — считает сервер. */
  taken: { type: Object, default: () => ({}) },
})
const emit = defineEmits(['update:modelValue', 'error'])

const value = computed(() => props.modelValue)

function emitValue(v) {
  emit('update:modelValue', v)
}

// ── Варианты ──
const options = computed(() => props.question.config?.options || [])

// Свой вариант хранится тем же значением, что и обычный, — отличаем его по
// отсутствию в списке вариантов автора.
const otherText = ref('')
const otherValue = computed(() => otherText.value.trim())

watch(() => props.modelValue, (v) => {
  if (props.question.type === 'checkbox') {
    const extra = (Array.isArray(v) ? v : []).find((x) => !options.value.includes(x))
    if (extra && extra !== otherText.value) otherText.value = extra
    return
  }
  if (typeof v === 'string' && v && !options.value.includes(v)) otherText.value = v
}, { immediate: true })

const otherPicked = computed(() => {
  if (props.question.type === 'checkbox') {
    return (Array.isArray(value.value) ? value.value : []).some((x) => !options.value.includes(x))
  }
  return !!value.value && !options.value.includes(value.value)
})

function isPicked(opt) {
  if (props.question.type === 'checkbox') {
    return (Array.isArray(value.value) ? value.value : []).includes(opt)
  }
  return value.value === opt
}

function choiceIcon(opt, other = false) {
  const picked = other ? otherPicked.value : isPicked(opt)
  if (props.question.type === 'checkbox') return picked ? 'check_box' : 'check_box_outline_blank'
  return picked ? 'radio_button_checked' : 'radio_button_unchecked'
}

function pick(opt) {
  if (props.disabled) return
  if (props.question.type === 'checkbox') {
    const list = Array.isArray(value.value) ? [...value.value] : []
    const i = list.indexOf(opt)
    if (i === -1) list.push(opt)
    else list.splice(i, 1)
    emitValue(list)
    return
  }
  // Повторный выбор снимает ответ: необязательный вопрос иначе не отменить.
  emitValue(value.value === opt ? '' : opt)
}

function pickOther() {
  if (props.disabled || !otherValue.value) return
  pick(otherValue.value)
}

function onOtherText(text) {
  const next = String(text || '').trim()
  if (props.question.type === 'checkbox') {
    const list = (Array.isArray(value.value) ? value.value : []).filter((x) => options.value.includes(x))
    if (next) list.push(next)
    emitValue(list)
    return
  }
  if (otherPicked.value || !value.value) emitValue(next)
}

// ── Запись (места) ──
const slots = computed(() => bookingSlots(props.question, props.taken))

function slotHint(slot) {
  if (!slot.total) return 'Мест не задано'
  return slot.left ? `Осталось ${slot.left} из ${slot.total}` : 'Свободных мест не осталось'
}

function pickSlot(slot) {
  if (props.disabled) return
  // Занятый вариант выбрать нельзя, но уже выбранный можно снять.
  if (!slot.left && value.value !== slot.option) return
  emitValue(value.value === slot.option ? '' : slot.option)
}

// ── Шкала и оценка ──
const scaleValues = computed(() => {
  const cfg = props.question.config || {}
  const min = Number(cfg.min) === 0 ? 0 : 1
  const max = Math.min(10, Math.max(min + 1, Number(cfg.max) || 5))
  return Array.from({ length: max - min + 1 }, (_, i) => min + i)
})

const ratingValues = computed(() => {
  const max = Math.min(10, Math.max(3, Number(props.question.config?.max) || 5))
  return Array.from({ length: max }, (_, i) => i + 1)
})

// ── Сетки ──
const gridRows = computed(() => props.question.config?.rows || [])
const gridCols = computed(() => props.question.config?.cols || [])

function gridPicked(row, col) {
  const cell = (value.value || {})[row]
  if (props.question.type === 'grid_checkbox') return Array.isArray(cell) && cell.includes(col)
  return cell === col
}

function gridIcon(row, col) {
  const picked = gridPicked(row, col)
  if (props.question.type === 'grid_checkbox') return picked ? 'check_box' : 'check_box_outline_blank'
  return picked ? 'radio_button_checked' : 'radio_button_unchecked'
}

function toggleGrid(row, col) {
  if (props.disabled) return
  const next = { ...(value.value || {}) }
  if (props.question.type === 'grid_checkbox') {
    const list = Array.isArray(next[row]) ? [...next[row]] : []
    const i = list.indexOf(col)
    if (i === -1) list.push(col)
    else list.splice(i, 1)
    if (list.length) next[row] = list
    else delete next[row]
  } else if (next[row] === col) {
    delete next[row]
  } else {
    next[row] = col
  }
  emitValue(next)
}

// ── Дата ──
const dateValue = computed(() => {
  if (!value.value) return null
  const d = new Date(String(value.value))
  return Number.isNaN(d.getTime()) ? null : d
})

function onDate(d) {
  if (!d) {
    emitValue('')
    return
  }
  const pad = (n) => String(n).padStart(2, '0')
  const date = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
  emitValue(props.question.config?.with_time
    ? `${date}T${pad(d.getHours())}:${pad(d.getMinutes())}`
    : date)
}

// ── Файлы ──
const files = computed(() => (Array.isArray(value.value) ? value.value : []))
const maxFiles = computed(() => fileCount(props.question))
const maxSize = computed(() => fileLimit(props.question))
const uploading = ref(false)
const uploadPercent = ref(0)

function pickFile() {
  const input = document.createElement('input')
  input.type = 'file'
  input.onchange = () => {
    const file = input.files?.[0]
    if (file) sendFile(file)
  }
  input.click()
}

async function sendFile(file) {
  if (!props.upload) return
  if (file.size > maxSize.value) {
    emit('error', `Файл больше ${Math.round(maxSize.value / 1024 / 1024)} МБ`)
    return
  }
  uploading.value = true
  uploadPercent.value = 0
  try {
    const saved = await props.upload(file, (p) => { uploadPercent.value = Math.round(p * 100) })
    emitValue([...files.value, saved])
  } catch (e) {
    emit('error', e?.message || 'Не удалось загрузить файл')
  } finally {
    uploading.value = false
  }
}

function removeFile(i) {
  const list = [...files.value]
  list.splice(i, 1)
  emitValue(list)
}

function fileSize(bytes) {
  if (!bytes) return ''
  const mb = bytes / 1024 / 1024
  return mb >= 1 ? `${mb.toFixed(1)} МБ` : `${Math.max(1, Math.round(bytes / 1024))} КБ`
}
</script>

<style scoped>
.fa { display: flex; flex-direction: column; gap: 10px; min-width: 0; }
.fa-wide { width: 100%; }
.fa-note { margin: 0; font-size: 14px; color: var(--color-text-dim); overflow-wrap: anywhere; }
.fa-other { display: flex; flex-direction: column; gap: 6px; }
.fa-hint { font-size: 12px; color: var(--color-text-dim); }
.fa-error { font-size: 12px; color: var(--color-error); }

.fa-scale { display: flex; flex-direction: column; gap: 6px; }
.fa-scale-row { display: flex; gap: 6px; flex-wrap: wrap; }
.fa-scale-label { font-size: 12px; color: var(--color-text-dim); }

.fa-rating { display: flex; align-items: center; gap: 4px; flex-wrap: wrap; }
.fa-rating-value { margin-left: 6px; font-size: 13px; color: var(--color-text-dim); }

/* Сетка прокручивается внутри себя: раздел горизонтальной прокрутки иметь не
   должен, а столбцов у вопроса бывает много. */
.fa-grid-wrap { overflow-x: auto; }

.fa-grid { border-collapse: collapse; font-size: 13px; }
.fa-grid th { font-weight: 500; color: var(--color-text-dim); padding: 6px 10px; }
.fa-grid-row { text-align: left; max-width: 220px; overflow-wrap: anywhere; }
.fa-grid-col { white-space: nowrap; }
.fa-grid td { text-align: center; padding: 2px 6px; }
.fa-grid tbody tr:nth-child(odd) { background: var(--color-surface-low); }

.fa-date { width: 100%; max-width: 280px; }
</style>
