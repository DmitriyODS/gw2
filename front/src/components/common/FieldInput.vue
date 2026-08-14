<template>
  <div class="fi">
    <!-- Текст. Длинный — самостоятельный тип; флаг config.multiline остаётся у
         полей, заведённых до его появления. -->
    <textarea
      v-if="field.type === 'textarea' || (field.type === 'text' && field.config?.multiline)"
      class="ctl" rows="3" :value="modelValue || ''"
      @input="emit('update:modelValue', $event.target.value)"
    />
    <input
      v-else-if="field.type === 'text'"
      class="ctl" type="text" :value="modelValue || ''"
      @input="emit('update:modelValue', $event.target.value)"
    />

    <!-- Телефон: разделители человек ставит как привык, храним «+цифры». -->
    <input
      v-else-if="field.type === 'phone'"
      class="ctl" type="tel" inputmode="tel" placeholder="+7 (900) 000-00-00"
      :value="modelValue || ''"
      @input="emit('update:modelValue', $event.target.value)"
      @blur="emit('update:modelValue', normalizePhone($event.target.value))"
    />

    <!-- Почта -->
    <template v-else-if="field.type === 'email'">
      <input
        class="ctl" type="email" inputmode="email" placeholder="name@example.com"
        :value="modelValue || ''"
        @input="emit('update:modelValue', $event.target.value)"
      />
      <span v-if="emailError" class="fi-error">{{ emailError }}</span>
    </template>

    <!-- Текст по шаблону: проверку задаёт составитель реестра регуляркой. -->
    <template v-else-if="field.type === 'regex'">
      <input
        class="ctl" type="text" :placeholder="cfg.hint || ''"
        :value="modelValue || ''"
        @input="emit('update:modelValue', $event.target.value)"
      />
      <span v-if="regexError" class="fi-error">{{ regexError }}</span>
    </template>

    <!-- Число: буквы не набираются вовсе (иначе они доезжали до базы и роняли
         сортировку по этому полю), границы и подсказка — из настроек поля. -->
    <template v-else-if="field.type === 'number'">
      <input
        class="ctl" type="text" inputmode="decimal" :value="modelValue ?? ''"
        :placeholder="numberHint"
        @input="onNumberInput"
        @blur="onNumberBlur"
      />
      <span v-if="numberError" class="fi-error">{{ numberError }}</span>
    </template>

    <!-- Наличие: пока галочка снята — позиция на месте. Отметили «забрали» —
         рядом появляется дата, до которой её не будет. -->
    <div v-else-if="field.type === 'stock'" class="fi-stock">
      <label class="fi-check">
        <Checkbox :model-value="!!modelValue?.taken" binary @update:model-value="setTaken" />
        <span>Забрали</span>
      </label>
      <div v-if="modelValue?.taken" class="fi-stock-until">
        <span class="fi-stock-label">До какой даты</span>
        <DatePicker
          :model-value="stockUntil"
          date-format="dd.mm.yy"
          show-button-bar
          placeholder="Выберите"
          @update:model-value="setUntil"
        />
      </div>
      <span v-else class="fi-stock-hint">Позиция в наличии.</span>
    </div>

    <!-- Флажок: надписи для установленного и снятого задаёт составитель. -->
    <label v-else-if="field.type === 'checkbox'" class="fi-check">
      <Checkbox :model-value="!!modelValue" binary @update:model-value="emit('update:modelValue', $event)" />
      <span>{{ checkboxText(field, !!modelValue) }}</span>
    </label>

    <!-- Список -->
    <MultiSelect
      v-else-if="field.type === 'select' && field.config?.multiple"
      :model-value="Array.isArray(modelValue) ? modelValue : []"
      :options="options" filter display="chip" placeholder="Выберите"
      @update:model-value="emit('update:modelValue', $event)"
    />
    <Select
      v-else-if="field.type === 'select'"
      :model-value="modelValue || null"
      :options="options" show-clear placeholder="Выберите"
      @update:model-value="emit('update:modelValue', $event)"
    />

    <!-- Ссылка -->
    <input
      v-else-if="field.type === 'link'"
      class="ctl" type="url" placeholder="https://…" :value="modelValue || ''"
      @input="emit('update:modelValue', $event.target.value)"
    />

    <!-- Дата/время -->
    <DatePicker
      v-else-if="field.type === 'datetime'"
      :model-value="dateValue"
      :show-time="showTime"
      :show-seconds="parts.seconds"
      :time-only="isTimeOnly"
      :view="dateView"
      :date-format="dateFormat"
      show-button-bar hour-format="24"
      placeholder="Выберите"
      @update:model-value="onDate"
    />

    <!-- Картинка / Файл -->
    <div v-else-if="field.type === 'image' || field.type === 'file'" class="fi-file">
      <div v-if="modelValue?.path" class="fi-file-cur">
        <FieldValue :field="field" :value="modelValue" />
        <button class="fi-file-rm" title="Убрать" @click="emit('update:modelValue', null)">
          <span class="material-symbols-outlined">close</span>
        </button>
      </div>
      <label class="fi-upload" :class="{ busy: uploading }">
        <input type="file" :accept="field.type === 'image' ? 'image/*' : undefined" hidden @change="onFile" />
        <span class="material-symbols-outlined">{{ uploading ? 'hourglass_top' : 'upload' }}</span>
        {{ uploading ? 'Загрузка…' : (modelValue?.path ? 'Заменить' : 'Загрузить') }}
      </label>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import Select from 'primevue/select'
import MultiSelect from 'primevue/multiselect'
import Checkbox from 'primevue/checkbox'
import DatePicker from 'primevue/datepicker'
import FieldValue from './FieldValue.vue'
import { useNotificationsStore } from '@/stores/notifications.js'
import { compressImage } from '@/utils/imageCompress.js'
import { dayString, parseDay } from '@/utils/dates.js'
import { checkboxText, dateParts, normalizePhone } from '@/utils/registryFields.js'

const props = defineProps({
  field: { type: Object, required: true },
  modelValue: { default: null },
  /* Загрузчик файла своего раздела: async (file) => метаданные { path, name, … } */
  upload: { type: Function, required: true },
})
const emit = defineEmits(['update:modelValue'])

const options = computed(() => props.field.config?.options || [])

const cfg = computed(() => props.field.config || {})

/* ── Число ──
   Поле принимает ТОЛЬКО число: буквы, попавшие сюда, доезжали до базы и потом
   роняли сортировку по этой колонке (Postgres не приводит «уточнить» к numeric).
   Поэтому нечисловой ввод не набирается вовсе, а границы min/max проверяются
   при потере фокуса — сервер их всё равно перепроверит. */
const NUMBER_CHARS = /^[+-]?[0-9]*[.,]?[0-9]*$/
const numberError = ref('')

const numberHint = computed(() => {
  if (cfg.value.pattern) return `Шаблон: ${cfg.value.pattern}`
  const { min, max } = cfg.value
  const has = (v) => v !== '' && v != null
  if (has(min) && has(max)) return `от ${min} до ${max}`
  if (has(min)) return `от ${min}`
  if (has(max)) return `до ${max}`
  return ''
})

function onNumberInput(e) {
  const raw = e.target.value.replace(',', '.')
  // Недопустимый символ откатываем, вернув в поле прежнее значение.
  if (!NUMBER_CHARS.test(raw)) {
    e.target.value = props.modelValue ?? ''
    numberError.value = 'Только число'
    return
  }
  numberError.value = ''
  emit('update:modelValue', raw)
}

/* ── Почта и шаблон ──
   Проверяем на лету, но НЕ мешаем набирать: недописанный адрес — это ещё не
   ошибка человека, а половина ввода. Отказ, если что, придёт с сервера. */
const emailError = computed(() => {
  const s = String(props.modelValue ?? '').trim()
  if (!s) return ''
  return /^[^@\s]+@[^@\s.]+(\.[^@\s.]+)+$/.test(s) ? '' : 'Непохоже на адрес почты'
})

const regexError = computed(() => {
  const s = String(props.modelValue ?? '').trim()
  const pattern = cfg.value.pattern
  if (!s || !pattern) return ''
  try {
    // Кривой шаблон — недосмотр составителя реестра, а не заполняющего: на нём
    // поле ведёт себя как обычный текст.
    return new RegExp(pattern).test(s) ? '' : (cfg.value.hint || 'Не соответствует шаблону')
  } catch {
    return ''
  }
})

function onNumberBlur() {
  const s = String(props.modelValue ?? '').trim()
  if (s === '') { numberError.value = ''; return }
  const n = Number(s)
  const { min, max } = cfg.value
  if (!Number.isFinite(n)) numberError.value = 'Только число'
  else if (min !== '' && min != null && n < Number(min)) numberError.value = `Не меньше ${min}`
  else if (max !== '' && max != null && n > Number(max)) numberError.value = `Не больше ${max}`
  else numberError.value = ''
}

/* ── Наличие ──
   Вернули позицию — значение снимаем целиком: «в наличии» не хранится, поэтому
   в записи не остаётся ни забытой даты, ни следа прошлой выдачи. */
function setTaken(on) {
  emit('update:modelValue', on ? { taken: true, until: props.modelValue?.until || '' } : null)
}

// Дата возврата хранится строкой YYYY-MM-DD — её же ждёт сервер.
const stockUntil = computed(() => parseDay(props.modelValue?.until))
function setUntil(date) {
  emit('update:modelValue', { taken: true, until: dayString(date) })
}

// ── Дата ──
// Части включаются по одной; dateParts понимает и прежнюю тройку
// year/month_day/time, поэтому поля календарей продолжают работать как были.
const parts = computed(() => dateParts(cfg.value))
const showTime = computed(() => parts.value.hours || parts.value.minutes || parts.value.seconds)
const hasDate = computed(() => parts.value.day || parts.value.month || parts.value.year)
const isTimeOnly = computed(() => showTime.value && !hasDate.value)
const yearOnly = computed(() => parts.value.year && !parts.value.day && !parts.value.month)
const dateView = computed(() => {
  if (yearOnly.value) return 'year'
  return parts.value.month && !parts.value.day ? 'month' : 'date'
})
const dateFormat = computed(() => {
  if (yearOnly.value) return 'yy'
  if (parts.value.month && !parts.value.day) return 'mm.yy'
  return parts.value.year ? 'dd.mm.yy' : 'dd.mm'
})
const dateValue = computed(() => {
  if (!props.modelValue) return null
  const d = new Date(props.modelValue)
  return isNaN(d.getTime()) ? null : d
})
function onDate(d) {
  emit('update:modelValue', d instanceof Date ? d.toISOString() : null)
}

// ── Файл ──
const uploading = ref(false)
async function onFile(e) {
  const picked = e.target.files?.[0]
  if (!picked) return
  uploading.value = true
  try {
    const file = props.field.type === 'image' ? await compressImage(picked) : picked
    const meta = await props.upload(file)
    emit('update:modelValue', meta)
  } catch {
    useNotificationsStore().error('Не удалось загрузить файл')
  } finally {
    uploading.value = false
    e.target.value = ''
  }
}
</script>

<style scoped>
.fi { width: 100%; }
/* Глобальный input.ctl задаёт фон/рамку, но не padding — добавляем. */
.fi .ctl { width: 100%; padding: 10px 12px; font: inherit; appearance: none; }
.fi textarea.ctl { resize: vertical; }
.fi :deep(.p-select),
.fi :deep(.p-multiselect),
.fi :deep(.p-datepicker) { width: 100%; }

.fi-check { display: inline-flex; align-items: center; gap: 8px; cursor: pointer; }
.fi-error { display: block; margin-top: 4px; font-size: 12px; color: var(--color-error); }

.fi-stock { display: flex; flex-direction: column; gap: 8px; align-items: flex-start; }
.fi-stock-until { display: flex; flex-direction: column; gap: 4px; width: 100%; }
.fi-stock-label { font-size: 12px; color: var(--color-text-dim); }
.fi-stock-hint { font-size: 12px; color: var(--color-text-dim); }

.fi-file { display: flex; flex-direction: column; gap: 8px; align-items: flex-start; max-width: 100%; }
.fi-file-cur { display: flex; align-items: flex-start; gap: 8px; max-width: 100%; min-width: 0; }
.fi-file-cur :deep(.fv-value) { min-width: 0; }
.fi-file-rm {
  width: 28px; height: 28px; flex-shrink: 0;
  display: grid; place-items: center;
  border: none; border-radius: var(--radius-full);
  background: var(--color-surface-low); color: var(--color-error);
  cursor: pointer;
}
.fi-file-rm:hover { background: var(--color-error-container, var(--color-surface-high)); }
.fi-upload {
  display: inline-flex; align-items: center; gap: 8px;
  padding: 8px 14px; border-radius: var(--radius-full);
  background: var(--color-primary-container); color: var(--color-on-primary-container);
  cursor: pointer; font-size: 14px; font-weight: 600;
}
.fi-upload.busy { opacity: 0.6; pointer-events: none; }
</style>
