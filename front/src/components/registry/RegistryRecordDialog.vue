<template>
  <AppDialog
    :model-value="modelValue"
    :title="title"
    size="lg"
    :busy="saving"
    @update:model-value="onClose"
  >
    <div v-if="registry?.fields?.length" class="rec-grid">
      <div
        v-for="f in registry.fields"
        :key="f.id"
        class="rec-cell"
        :style="cellStyle(f)"
      >
        <div class="rec-label">{{ f.label }}</div>
        <FieldInput
          v-if="editing"
          :field="f"
          :model-value="form[String(f.id)] ?? null"
          :upload="upload"
          @update:model-value="form[String(f.id)] = $event"
        />
        <FieldValue v-else :field="f" :value="record?.data?.[String(f.id)] ?? null" />
      </div>
    </div>
    <p v-else class="rec-empty">В этом реестре пока нет полей.</p>

    <template #footer>
      <template v-if="editing">
        <AppButton variant="text" label="Отмена" :disabled="saving" @click="cancelEdit" />
        <AppButton
          variant="filled"
          :label="isNew ? 'Создать' : 'Сохранить'"
          :loading="saving"
          @click="submit"
        />
      </template>
      <template v-else>
        <AppButton variant="text" label="Закрыть" @click="onClose(false)" />
        <AppButton
          v-if="!readonly"
          variant="filled"
          icon="edit"
          label="Редактировать"
          @click="editing = true"
        />
      </template>
    </template>
  </AppDialog>
</template>

<script setup>
/* Карточка записи: просмотр и правка. Сохранение и загрузку файлов делает
   вызывающий (`save`/`upload`) — раздел ходит своими ручками, публичная
   страница по ссылке своими, а сама карточка про это ничего не знает и потому
   годится обеим. */
import { reactive, ref, watch } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import FieldInput from '@/components/common/FieldInput.vue'
import FieldValue from '@/components/common/FieldValue.vue'
import { uploadFile } from '@/api/registries.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  registry: { type: Object, default: null },
  record: { type: Object, default: null }, // null → создание новой записи
  readonly: { type: Boolean, default: false }, // просмотр по ссылке уровня view
  /** (data, record|null) => Promise — сохранение; record === null → создание. */
  save: { type: Function, default: null },
  /** (file) => Promise<{path,name,mime,size,thumb?}> — загрузка файла поля. */
  upload: { type: Function, default: uploadFile },
})
const emit = defineEmits(['update:modelValue', 'saved'])

const notif = useNotificationsStore()

const editing = ref(false)
const saving = ref(false)
const form = reactive({})
const isNew = ref(false)

watch(() => props.modelValue, (open) => {
  if (!open) return
  isNew.value = !props.record && !props.readonly
  editing.value = isNew.value // в readonly — всегда просмотр
  for (const k of Object.keys(form)) delete form[k]
  const data = props.record?.data || {}
  for (const [k, v] of Object.entries(data)) form[k] = v
})

const title = ref('')
watch([() => props.modelValue, () => props.record, editing], () => {
  title.value = isNew.value ? 'Новая запись' : (editing.value ? 'Редактирование записи' : 'Запись')
})

function cellStyle(f) {
  return {
    gridColumn: `span ${Math.min(3, Math.max(1, f.col_span || 1))}`,
    gridRow: `span ${Math.max(1, f.row_span || 1)}`,
  }
}

function onClose(v) {
  if (v) return
  emit('update:modelValue', false)
}
function cancelEdit() {
  if (isNew.value) { emit('update:modelValue', false); return }
  editing.value = false
}

async function submit() {
  if (!props.save) return
  saving.value = true
  try {
    await props.save({ ...form }, isNew.value ? null : props.record)
    notif.success(isNew.value ? 'Запись добавлена' : 'Запись сохранена')
    emit('saved')
    emit('update:modelValue', false)
  } catch (e) {
    notif.error(e?.message || 'Не удалось сохранить запись')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.rec-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}
.rec-cell { min-width: 0; display: flex; flex-direction: column; gap: 6px; }
.rec-label { font-size: 12px; font-weight: 600; color: var(--color-text-dim); text-transform: uppercase; letter-spacing: 0.02em; }
.rec-empty { color: var(--color-text-dim); text-align: center; padding: 24px 0; }

@media (max-width: 640px) {
  .rec-grid { grid-template-columns: 1fr; }
  .rec-cell { grid-column: span 1 !important; }
}

</style>
