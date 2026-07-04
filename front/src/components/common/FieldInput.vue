<template>
  <div class="fi">
    <!-- Текст -->
    <textarea
      v-if="field.type === 'text' && field.config?.multiline"
      class="ctl" rows="3" :value="modelValue || ''"
      @input="emit('update:modelValue', $event.target.value)"
    />
    <input
      v-else-if="field.type === 'text'"
      class="ctl" type="text" :value="modelValue || ''"
      @input="emit('update:modelValue', $event.target.value)"
    />

    <!-- Число -->
    <template v-else-if="field.type === 'number'">
      <input
        class="ctl" type="text" inputmode="decimal" :value="modelValue ?? ''"
        :placeholder="field.config?.pattern ? `Шаблон: ${field.config.pattern}` : ''"
        @input="emit('update:modelValue', $event.target.value)"
      />
    </template>

    <!-- Галочка -->
    <label v-else-if="field.type === 'checkbox'" class="fi-check">
      <Checkbox :model-value="!!modelValue" binary @update:model-value="emit('update:modelValue', $event)" />
      <span>{{ field.label }}</span>
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
      :show-time="!!field.config?.time"
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

const props = defineProps({
  field: { type: Object, required: true },
  modelValue: { default: null },
  /* Загрузчик файла своего раздела: async (file) => метаданные { path, name, … } */
  upload: { type: Function, required: true },
})
const emit = defineEmits(['update:modelValue'])

const options = computed(() => props.field.config?.options || [])

// ── Дата ──
const cfg = computed(() => props.field.config || {})
const isTimeOnly = computed(() => !!cfg.value.time && !cfg.value.month_day && !cfg.value.year)
const dateView = computed(() => (cfg.value.year && !cfg.value.month_day ? 'year' : 'date'))
const dateFormat = computed(() => (cfg.value.year && !cfg.value.month_day ? 'yy' : 'dd.mm.yy'))
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
