<template>
  <AppDialog
    :model-value="modelValue"
    :title="title"
    :icon="isNew ? 'note_add' : 'event_note'"
    size="md"
    :busy="saving"
    @update:model-value="onClose"
  >
    <!-- Режим редактирования / создания -->
    <div v-if="editing" class="de">
      <label class="de-field">
        <span class="de-label">Название<span class="de-req">*</span></span>
        <input
          ref="titleInput"
          v-model="form.title"
          type="text"
          class="de-input"
          placeholder="Что нужно сделать?"
          maxlength="200"
          @keydown.enter.exact.prevent="save"
        />
      </label>

      <label class="de-field">
        <span class="de-label">Описание</span>
        <textarea
          v-model="form.description"
          class="de-input de-textarea"
          rows="3"
          placeholder="Подробности (необязательно)"
        />
      </label>

      <!-- Дата/время сворачиваются: быстрое создание — только название и описание -->
      <button type="button" class="de-more" @click="showDetails = !showDetails">
        <span class="material-symbols-outlined">{{ showDetails ? 'expand_less' : 'expand_more' }}</span>
        {{ showDetails ? 'Скрыть дату и время' : 'Дата и время' }}
        <span v-if="!showDetails" class="de-more-hint">{{ detailsHint }}</span>
      </button>

      <Transition name="de-reveal">
        <div v-if="showDetails" class="de-details">
          <label class="de-field">
            <span class="de-label">Дата<span class="de-req">*</span></span>
            <DatePicker
              v-model="form.date"
              date-format="dd.mm.yy" placeholder="Выберите день"
              show-button-bar class="de-date"
            />
          </label>
          <div class="de-times">
            <label class="de-field">
              <span class="de-label">Начало</span>
              <TimePicker v-model="form.start" placeholder="—" clearable />
            </label>
            <label class="de-field">
              <span class="de-label">Завершение</span>
              <TimePicker v-model="form.end" placeholder="—" clearable />
            </label>
          </div>
        </div>
      </Transition>
    </div>

    <!-- Режим просмотра -->
    <div v-else class="de de-view">
      <div class="de-vrow">
        <span class="material-symbols-outlined">calendar_today</span>
        <span>{{ viewDate }}<template v-if="viewTime"> · {{ viewTime }}</template></span>
      </div>
      <h3 class="de-vtitle" :class="{ done: entry?.done }">{{ entry?.title }}</h3>
      <p v-if="entry?.description" class="de-vdesc">{{ entry.description }}</p>

      <div v-if="entry?.linked_task_id" class="de-task">
        <span class="material-symbols-outlined">link</span>
        К записи привязана задача
        <router-link class="de-task-link" to="/tasks">Открыть задачи</router-link>
      </div>
    </div>

    <template #footer>
      <div class="de-footer">
        <template v-if="editing">
          <span class="de-foot-spacer" />
          <button class="btn-text" :disabled="saving" @click="cancelEdit">Отмена</button>
          <button class="btn-filled" :disabled="saving" @click="save">
            <span v-if="saving" class="material-symbols-outlined spin">progress_activity</span>
            Сохранить
          </button>
        </template>
        <template v-else>
          <button v-if="canEdit" class="btn-icon danger" title="Удалить" @click="confirmDelete = true">
            <span class="material-symbols-outlined">delete</span>
          </button>
          <span class="de-foot-spacer" />
          <button
            v-if="canEdit && canCreateTask && !entry?.linked_task_id"
            class="btn-text" title="Создать задачу с юнитом" @click="$emit('create-task', entry)"
          >
            <span class="material-symbols-outlined">add_task</span>
            <span class="de-btn-label">Задача</span>
          </button>
          <button v-if="canEdit" class="btn-tonal" @click="toggleDone">
            <span class="material-symbols-outlined">{{ entry?.done ? 'undo' : 'check' }}</span>
            <span class="de-btn-label">{{ entry?.done ? 'В активные' : 'Выполнено' }}</span>
          </button>
          <button v-if="canEdit" class="btn-filled" @click="startEdit">
            <span class="material-symbols-outlined">edit</span>
            <span class="de-btn-label">Изменить</span>
          </button>
          <button v-if="!canEdit" class="btn-text" @click="onClose(false)">Закрыть</button>
        </template>
      </div>
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
import { computed, nextTick, reactive, ref, watch } from 'vue'
import DatePicker from 'primevue/datepicker'
import AppDialog from '@/components/common/AppDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import TimePicker from '@/components/common/TimePicker.vue'
import { useDiariesStore, dayKey } from '@/stores/diaries.js'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  entry: { type: Object, default: null },       // null → создание
  readonly: { type: Boolean, default: false },   // чужой/публичный — только чтение
  defaultDate: { type: [Date, String, Number], default: null },
})
const emit = defineEmits(['update:modelValue', 'create-task'])

const store = useDiariesStore()
const auth = useAuthStore()
const notif = useNotificationsStore()

const editing = ref(false)
const saving = ref(false)
const isNew = ref(false)
const showDetails = ref(false)
const confirmDelete = ref(false)
const titleInput = ref(null)

const form = reactive({ title: '', description: '', date: new Date(), start: null, end: null })

const canEdit = computed(() => !props.readonly)
const canCreateTask = computed(() => !auth.isSuperAdmin && auth.roleLevel > 0)

const pad = (n) => String(n).padStart(2, '0')
const minToStr = (m) => (m == null ? null : `${pad(Math.floor(m / 60))}:${pad(m % 60)}`)
const strToMin = (s) => {
  if (!/^\d{2}:\d{2}$/.test(s || '')) return null
  const [h, m] = s.split(':').map(Number)
  return h * 60 + m
}
function parseEntryDate(s) {
  if (!s) return new Date()
  const [y, m, d] = String(s).split('-').map(Number)
  return new Date(y, (m || 1) - 1, d || 1)
}

const title = computed(() => (isNew.value ? 'Новая запись' : editing.value ? 'Редактирование' : 'Запись'))
const detailsHint = computed(() => {
  const d = form.date ? new Date(form.date).toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' }) : ''
  const t = form.start ? ` · ${form.start}${form.end ? '–' + form.end : ''}` : ''
  return d + t
})

const viewDate = computed(() => {
  if (!props.entry) return ''
  return parseEntryDate(props.entry.entry_date).toLocaleDateString('ru-RU',
    { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
})
const viewTime = computed(() => {
  const e = props.entry
  if (!e || e.start_min == null) return ''
  return e.end_min != null ? `${minToStr(e.start_min)}–${minToStr(e.end_min)}` : minToStr(e.start_min)
})

watch(() => props.modelValue, (open) => {
  if (!open) return
  isNew.value = !props.entry && !props.readonly
  editing.value = isNew.value
  confirmDelete.value = false
  showDetails.value = false
  if (props.entry) {
    form.title = props.entry.title || ''
    form.description = props.entry.description || ''
    form.date = parseEntryDate(props.entry.entry_date)
    form.start = minToStr(props.entry.start_min)
    form.end = minToStr(props.entry.end_min)
  } else {
    form.title = ''
    form.description = ''
    form.date = props.defaultDate ? new Date(props.defaultDate) : new Date()
    form.start = null
    form.end = null
  }
  if (isNew.value) nextTick(() => titleInput.value?.focus())
})

function startEdit() {
  showDetails.value = true
  editing.value = true
}

function cancelEdit() {
  if (isNew.value) { onClose(false); return }
  editing.value = false
}

async function save() {
  const t = form.title.trim()
  if (!t) { notif.error('Укажите название записи'); return }
  if (!form.date) { notif.error('Укажите дату записи'); return }
  saving.value = true
  try {
    const body = {
      entry_date: dayKey(form.date),
      start_min: strToMin(form.start),
      end_min: strToMin(form.end),
      title: t,
      description: form.description.trim(),
    }
    if (isNew.value) {
      await store.createEntry(body)
      notif.success('Запись добавлена')
    } else {
      await store.updateEntry(props.entry.id, body)
      notif.success('Запись сохранена')
    }
    onClose(false)
  } catch (e) {
    notif.error(e?.message || 'Не удалось сохранить запись')
  } finally {
    saving.value = false
  }
}

async function toggleDone() {
  try {
    await store.toggleDone(props.entry.id, !props.entry.done)
    onClose(false)
  } catch (e) {
    notif.error(e?.message || 'Не удалось изменить статус')
  }
}

async function doDelete() {
  confirmDelete.value = false
  try {
    await store.deleteEntry(props.entry.id)
    notif.success('Запись удалена')
    onClose(false)
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить запись')
  }
}

function onClose(v) { emit('update:modelValue', v) }
</script>

<style scoped>
.de { display: flex; flex-direction: column; gap: 14px; }
.de-field { display: flex; flex-direction: column; gap: 6px; }
.de-label { font-size: 13px; font-weight: 600; color: var(--color-text-dim); }
.de-req { color: var(--color-error); margin-left: 2px; }
.de-input {
  width: 100%; padding: 10px 12px; font: inherit; color: var(--color-text);
  background: var(--color-surface-high); border: 1px solid var(--color-outline-variant);
  border-radius: var(--radius-md, 14px); outline: none; transition: border-color 0.15s;
}
.de-input:focus { border-color: var(--color-primary); }
.de-textarea { resize: vertical; min-height: 72px; }
.de-date { width: 100%; }

.de-more {
  display: inline-flex; align-items: center; gap: 6px; align-self: flex-start;
  border: none; background: none; cursor: pointer; padding: 4px 0;
  color: var(--color-primary); font-weight: 600; font-size: 14px;
}
.de-more-hint { color: var(--color-text-dim); font-weight: 500; }
.de-details { display: flex; flex-direction: column; gap: 14px; }
.de-times { display: flex; gap: 12px; }
.de-times > .de-field { flex: 1; }

.de-reveal-enter-active, .de-reveal-leave-active { transition: opacity 0.2s, transform 0.2s; }
.de-reveal-enter-from, .de-reveal-leave-to { opacity: 0; transform: translateY(-6px); }

/* Просмотр */
.de-view { gap: 10px; }
.de-vrow { display: inline-flex; align-items: center; gap: 8px; color: var(--color-text-dim); font-size: 14px; text-transform: capitalize; }
.de-vrow .material-symbols-outlined { font-size: 20px; }
.de-vtitle { margin: 0; font-size: 19px; font-weight: 700; color: var(--color-text); }
.de-vtitle.done { text-decoration: line-through; color: var(--color-text-dim); }
.de-vdesc { margin: 0; color: var(--color-text); white-space: pre-wrap; line-height: 1.5; }
.de-task {
  display: inline-flex; align-items: center; gap: 8px; margin-top: 4px; padding: 8px 12px;
  background: var(--color-primary-container); color: var(--color-on-primary-container);
  border-radius: var(--radius-md); font-size: 13px; font-weight: 600;
}
.de-task-link { color: inherit; text-decoration: underline; }

/* Кнопки футера. Контейнер переносит кнопки целиком, а не ломает текст внутри. */
.de-footer { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; width: 100%; }
.de-foot-spacer { flex: 1 1 auto; }
.btn-text, .btn-tonal, .btn-filled {
  display: inline-flex; align-items: center; gap: 6px; border: none; cursor: pointer;
  border-radius: var(--radius-full); font-weight: 600; font-size: 14px; white-space: nowrap;
}
.btn-text { padding: 9px 14px; background: none; color: var(--color-text-dim); }
.btn-text:hover { background: var(--color-surface-high); color: var(--color-text); }
.btn-text.danger { color: var(--color-error); }
.btn-tonal { padding: 9px 16px; background: var(--color-primary-container); color: var(--color-on-primary-container); }
.btn-filled { padding: 9px 18px; background: var(--color-primary); color: var(--color-on-primary); }
.btn-icon {
  display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0;
  width: 40px; height: 40px; border: none; border-radius: var(--radius-full);
  background: none; cursor: pointer; color: var(--color-text-dim);
}
.btn-icon:hover { background: var(--color-surface-high); }
.btn-icon.danger { color: var(--color-error); }
.spin { animation: despin 1s linear infinite; }
@keyframes despin { to { transform: rotate(360deg); } }

/* Мобайл: на узком экране подписи действий прячем — кнопки становятся
   компактными иконками и помещаются в один ряд. */
@media (max-width: 480px) {
  .de-btn-label { display: none; }
  .btn-tonal, .btn-filled { padding: 9px 14px; }
}
</style>
