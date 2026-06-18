<template>
  <AppDialog
    :model-value="modelValue"
    :title="title"
    :icon="editing ? 'edit_note' : 'description'"
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
        <RegistryFieldInput
          v-if="editing"
          :field="f"
          :model-value="form[String(f.id)] ?? null"
          @update:model-value="form[String(f.id)] = $event"
        />
        <RegistryFieldValue v-else :field="f" :value="record?.data?.[String(f.id)] ?? null" />
      </div>
    </div>
    <p v-else class="rec-empty">В этом реестре пока нет полей.</p>

    <template #footer>
      <template v-if="editing">
        <button class="btn-text" :disabled="saving" @click="cancelEdit">Отмена</button>
        <button class="btn-filled" :disabled="saving" @click="save">
          <span v-if="saving" class="material-symbols-outlined spin">progress_activity</span>
          {{ isNew ? 'Создать' : 'Сохранить' }}
        </button>
      </template>
      <template v-else>
        <button class="btn-text" @click="onClose(false)">Закрыть</button>
        <button v-if="!readonly" class="btn-filled" @click="editing = true">
          <span class="material-symbols-outlined">edit</span> Редактировать
        </button>
      </template>
    </template>
  </AppDialog>
</template>

<script setup>
import { reactive, ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import RegistryFieldInput from './RegistryFieldInput.vue'
import RegistryFieldValue from './RegistryFieldValue.vue'
import { useRegistriesStore } from '@/stores/registries.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  registry: { type: Object, default: null },
  record: { type: Object, default: null }, // null → создание новой записи
  readonly: { type: Boolean, default: false }, // публичный просмотр по ссылке
})
const emit = defineEmits(['update:modelValue', 'saved'])

const store = useRegistriesStore()
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

async function save() {
  saving.value = true
  try {
    const data = { ...form }
    if (isNew.value) {
      await store.createRecord(data)
      notif.success('Запись добавлена')
    } else {
      await store.updateRecord(props.record.id, data)
      notif.success('Запись сохранена')
    }
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

.btn-text {
  border: none; background: none; cursor: pointer;
  padding: 10px 16px; border-radius: var(--radius-full);
  color: var(--color-text-dim); font-weight: 600; font-size: 14px;
}
.btn-text:hover { background: var(--color-surface-high); color: var(--color-text); }
.btn-filled {
  display: inline-flex; align-items: center; gap: 6px;
  border: none; cursor: pointer;
  padding: 10px 18px; border-radius: var(--radius-full);
  background: var(--color-primary); color: var(--color-on-primary);
  font-weight: 600; font-size: 14px;
}
.btn-filled:disabled { opacity: 0.6; cursor: default; }
.spin { animation: rspin 1s linear infinite; }
@keyframes rspin { to { transform: rotate(360deg); } }
</style>
