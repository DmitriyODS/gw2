<template>
  <AppDialog
    :model-value="modelValue"
    :title="title"
    :icon="editing ? 'edit_calendar' : 'event'"
    size="lg"
    :busy="saving"
    @update:model-value="onClose"
  >
    <div class="ce-grid">
      <!-- Встроенное обязательное поле: дата и время записи -->
      <div class="ce-cell ce-cell-date">
        <div class="ce-label">Дата и время<span class="ce-req">*</span></div>
        <DatePicker
          v-if="editing"
          v-model="eventAt"
          show-time hour-format="24" :step-minute="5"
          show-button-bar date-format="dd.mm.yy" placeholder="Выберите"
          class="ce-date"
        />
        <span v-else class="ce-date-val">{{ formattedEventAt }}</span>
      </div>

      <!-- Пользовательские поля (с учётом условной видимости) -->
      <div
        v-for="f in visibleFields"
        :key="f.id"
        class="ce-cell"
        :style="cellStyle(f)"
      >
        <div class="ce-label">{{ f.label }}</div>
        <FieldInput
          v-if="editing"
          :field="f"
          :model-value="form[String(f.id)] ?? null"
          :upload="uploadFile"
          @update:model-value="form[String(f.id)] = $event"
        />
        <FieldValue v-else :field="f" :value="entry?.data?.[String(f.id)] ?? null" />
      </div>
    </div>

    <template #footer>
      <template v-if="editing">
        <button class="btn-text" :disabled="saving" @click="cancelEdit">Отмена</button>
        <button class="btn-filled" :disabled="saving" @click="save">
          <span v-if="saving" class="material-symbols-outlined spin">progress_activity</span>
          {{ isNew ? 'Создать' : 'Сохранить' }}
        </button>
      </template>
      <template v-else>
        <button v-if="!readonly && !isNew" class="btn-text danger" @click="confirmDelete = true">
          <span class="material-symbols-outlined">delete</span> Удалить
        </button>
        <span class="ce-foot-spacer" />
        <button class="btn-text" @click="onClose(false)">Закрыть</button>
        <button v-if="!readonly" class="btn-filled" @click="editing = true">
          <span class="material-symbols-outlined">edit</span> Редактировать
        </button>
      </template>
    </template>

    <ConfirmDialog
      :visible="confirmDelete"
      header="Удалить запись?"
      message="Запись будет удалена безвозвратно."
      confirm-label="Удалить" danger-confirm
      @confirm="doDelete" @cancel="confirmDelete = false"
    />
  </AppDialog>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import DatePicker from 'primevue/datepicker'
import AppDialog from '@/components/common/AppDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import FieldInput from '@/components/common/FieldInput.vue'
import FieldValue from '@/components/common/FieldValue.vue'
import { uploadFile } from '@/api/calendars.js'
import { useCalendarsStore } from '@/stores/calendars.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { isFieldVisible } from '@/utils/calendarFields.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  calendar: { type: Object, default: null },
  entry: { type: Object, default: null },     // null → создание новой записи
  readonly: { type: Boolean, default: false }, // публичный просмотр по ссылке
  defaultDate: { type: [Date, String, Number], default: null }, // предзаполнение дня
})
const emit = defineEmits(['update:modelValue', 'saved'])

const store = useCalendarsStore()
const notif = useNotificationsStore()

const editing = ref(false)
const saving = ref(false)
const form = reactive({})
const eventAt = ref(null)
const isNew = ref(false)
const confirmDelete = ref(false)

watch(() => props.modelValue, (open) => {
  if (!open) return
  isNew.value = !props.entry && !props.readonly
  editing.value = isNew.value
  for (const k of Object.keys(form)) delete form[k]
  const data = props.entry?.data || {}
  for (const [k, v] of Object.entries(data)) form[k] = v
  eventAt.value = resolveInitialDate()
})

function resolveInitialDate() {
  if (props.entry?.event_at) {
    const d = new Date(props.entry.event_at)
    return isNaN(d.getTime()) ? null : d
  }
  if (props.defaultDate != null) {
    const d = new Date(props.defaultDate)
    return isNaN(d.getTime()) ? new Date() : d
  }
  return new Date()
}

const visibleFields = computed(() =>
  (props.calendar?.fields || []).filter((f) => isFieldVisible(f, form)),
)

const formattedEventAt = computed(() => {
  const v = props.entry?.event_at
  if (!v) return '—'
  const d = new Date(v)
  if (isNaN(d.getTime())) return '—'
  const pad = (n) => String(n).padStart(2, '0')
  return `${pad(d.getDate())}.${pad(d.getMonth() + 1)}.${d.getFullYear()} ${pad(d.getHours())}:${pad(d.getMinutes())}`
})

const title = computed(() =>
  isNew.value ? 'Новая запись' : (editing.value ? 'Редактирование записи' : 'Запись'),
)

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
  if (!(eventAt.value instanceof Date) || isNaN(eventAt.value.getTime())) {
    notif.error('Укажите дату и время записи')
    return
  }
  saving.value = true
  try {
    // Сохраняем значения только видимых полей — скрытые условием очищаются.
    const visibleIds = new Set(visibleFields.value.map((f) => String(f.id)))
    const data = {}
    for (const [k, v] of Object.entries(form)) {
      if (visibleIds.has(k) && v != null && v !== '') data[k] = v
    }
    const at = new Date(eventAt.value)
    at.setSeconds(0, 0)
    const iso = at.toISOString()
    if (isNew.value) {
      await store.createEntry(iso, data)
      notif.success('Запись добавлена')
    } else {
      await store.updateEntry(props.entry.id, iso, data)
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

async function doDelete() {
  confirmDelete.value = false
  if (!props.entry) return
  try {
    await store.deleteEntry(props.entry.id)
    notif.success('Запись удалена')
    emit('saved')
    emit('update:modelValue', false)
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить запись')
  }
}
</script>

<style scoped>
.ce-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}
.ce-cell { min-width: 0; display: flex; flex-direction: column; gap: 6px; }
.ce-cell-date { grid-column: span 3; }
.ce-label { font-size: 12px; font-weight: 600; color: var(--color-text-dim); text-transform: uppercase; letter-spacing: 0.02em; }
.ce-req { color: var(--color-error); margin-left: 2px; }
.ce-date { width: 100%; max-width: 280px; }
.ce-date :deep(.p-datepicker) { width: 100%; }
.ce-date-val { font-size: 15px; font-weight: 600; color: var(--color-text); }

.btn-text {
  border: none; background: none; cursor: pointer;
  padding: 10px 16px; border-radius: var(--radius-full);
  color: var(--color-text-dim); font-weight: 600; font-size: 14px;
}
.btn-text:hover { background: var(--color-surface-high); color: var(--color-text); }
.btn-text.danger { color: var(--color-error); }
.btn-text.danger:hover { background: var(--color-error-container, var(--color-surface-high)); color: var(--color-error); }
.ce-foot-spacer { flex: 1; }
.btn-filled {
  display: inline-flex; align-items: center; gap: 6px;
  border: none; cursor: pointer;
  padding: 10px 18px; border-radius: var(--radius-full);
  background: var(--color-primary); color: var(--color-on-primary);
  font-weight: 600; font-size: 14px;
}
.btn-filled:disabled { opacity: 0.6; cursor: default; }
.spin { animation: cespin 1s linear infinite; }
@keyframes cespin { to { transform: rotate(360deg); } }

@media (max-width: 640px) {
  .ce-grid { grid-template-columns: 1fr; }
  .ce-cell { grid-column: span 1 !important; }
}
</style>
