<script setup>
/* Создание и правка напоминания: о чём, когда и как часто повторять.
   Время задаётся в зоне пользователя, а на сервер уходит момент в UTC —
   повторы сервер считает уже по IANA-зоне, которую мы здесь и передаём. */
import { computed, ref, watch } from 'vue'
import DatePicker from 'primevue/datepicker'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import AppDialog from '@/components/ui/AppDialog.vue'
import TimePicker from '@/components/common/TimePicker.vue'
import { useRemindersStore } from '@/stores/reminders.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  reminder: { type: Object, default: null }, // null — создание
  // Пресет привязки к записи ежедневника/календаря.
  link: { type: Object, default: null },
  // Готовое название (команда «создай напоминание …» из строки поиска).
  presetTitle: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue', 'saved'])

const store = useRemindersStore()
const notify = useNotificationsStore()

const REPEATS = [
  { key: 'none', label: 'Не повторять' },
  { key: 'daily', label: 'Каждый день' },
  { key: 'weekdays', label: 'По рабочим дням' },
  { key: 'weekly', label: 'По дням недели' },
  { key: 'monthly', label: 'Каждый месяц' },
  { key: 'yearly', label: 'Каждый год' },
]

const WEEKDAYS = [
  { value: 1, label: 'Пн' }, { value: 2, label: 'Вт' }, { value: 3, label: 'Ср' },
  { value: 4, label: 'Чт' }, { value: 5, label: 'Пт' }, { value: 6, label: 'Сб' },
  { value: 7, label: 'Вс' },
]

// Быстрые сроки — самый частый сценарий («напомни через час»).
const QUICK = [
  { label: 'Через 15 минут', minutes: 15 },
  { label: 'Через час', minutes: 60 },
  { label: 'Завтра утром', tomorrowAt: '09:00' },
]

const title = ref('')
const note = ref('')
const date = ref(null)     // Date — общий выбор даты приложения
const time = ref('09:00')  // HH:mm
const repeatKind = ref('none')
const repeatInterval = ref(1)
const repeatDays = ref([])
const saving = ref(false)

const isEdit = computed(() => !!props.reminder)
const showDays = computed(() => repeatKind.value === 'weekly')
const showInterval = computed(() => ['daily', 'weekly', 'monthly', 'yearly'].includes(repeatKind.value))

const intervalSuffix = computed(() => ({
  daily: 'дн.', weekly: 'нед.', monthly: 'мес.', yearly: 'г.',
}[repeatKind.value] || ''))

watch(() => props.modelValue, (open) => {
  if (!open) return
  const r = props.reminder
  if (r) {
    title.value = r.title || ''
    note.value = r.note || ''
    const at = new Date(r.remind_at)
    date.value = at
    time.value = toTimeInput(at)
    repeatKind.value = r.repeat?.kind || 'none'
    repeatInterval.value = r.repeat?.interval || 1
    repeatDays.value = [...(r.repeat?.days || [])]
  } else {
    const soon = new Date(Date.now() + 60 * 60 * 1000)
    soon.setMinutes(Math.ceil(soon.getMinutes() / 5) * 5, 0, 0)
    title.value = props.presetTitle || props.link?.title || ''
    note.value = ''
    date.value = soon
    time.value = toTimeInput(soon)
    repeatKind.value = 'none'
    repeatInterval.value = 1
    repeatDays.value = []
  }
})

function toTimeInput(d) {
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`
}

function applyQuick(q) {
  const d = new Date()
  if (q.minutes) {
    d.setMinutes(d.getMinutes() + q.minutes)
  } else if (q.tomorrowAt) {
    d.setDate(d.getDate() + 1)
    const [h, m] = q.tomorrowAt.split(':').map(Number)
    d.setHours(h, m, 0, 0)
  }
  date.value = d
  time.value = toTimeInput(d)
}

/** Момент срабатывания: выбранный день + время из TimePicker. */
function combine(day, hhmm) {
  if (!day) return null
  const [h, m] = String(hhmm || '00:00').split(':').map(Number)
  const at = new Date(day)
  at.setHours(Number.isFinite(h) ? h : 0, Number.isFinite(m) ? m : 0, 0, 0)
  return Number.isNaN(at.getTime()) ? null : at
}

function toggleDay(value) {
  const idx = repeatDays.value.indexOf(value)
  if (idx >= 0) repeatDays.value.splice(idx, 1)
  else repeatDays.value.push(value)
}

async function save() {
  if (!title.value.trim()) {
    notify.warn('Напишите, о чём напомнить')
    return
  }
  const at = combine(date.value, time.value)
  if (!at) {
    notify.warn('Проверьте дату и время')
    return
  }
  saving.value = true
  try {
    const body = {
      title: title.value.trim(),
      note: note.value.trim(),
      remind_at: at.toISOString(),
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC',
      repeat: {
        kind: repeatKind.value,
        interval: Number(repeatInterval.value) || 1,
        days: repeatKind.value === 'weekly' ? [...repeatDays.value].sort((a, b) => a - b) : [],
      },
    }
    if (props.link) body.link = props.link
    const saved = isEdit.value
      ? await store.update(props.reminder.id, body)
      : await store.create(body)
    emit('saved', saved)
    close()
  } catch (e) {
    notify.error(e?.message || 'Не удалось сохранить напоминание')
  } finally {
    saving.value = false
  }
}

function close() {
  emit('update:modelValue', false)
}
</script>

<template>
  <AppDialog
    :model-value="modelValue"
    :title="isEdit ? 'Напоминание' : 'Новое напоминание'"
    size="sm"
    :busy="saving"
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: isEdit ? 'Сохранить' : 'Создать' },
    ]"
    @confirm="save"
    @cancel="close"
    @update:model-value="(v) => !v && close()"
  >
    <div class="rd">
      <InputText v-model="title" placeholder="О чём напомнить" maxlength="300" @keydown.enter="save" />
      <Textarea v-model="note" placeholder="Заметка (необязательно)" rows="2" auto-resize maxlength="2000" />

      <div v-if="link" class="rd-link">
        <span class="material-symbols-outlined">{{ link.kind === 'calendar' ? 'event' : 'checklist' }}</span>
        Привязано к записи: {{ link.title || 'без названия' }}
      </div>

      <div class="rd-quick">
        <button v-for="q in QUICK" :key="q.label" type="button" class="rd-chip" @click="applyQuick(q)">
          {{ q.label }}
        </button>
      </div>

      <div class="rd-when">
        <DatePicker
          v-model="date"
          date-format="dd.mm.yy"
          placeholder="Выберите день"
          show-button-bar
          class="rd-date"
        />
        <TimePicker v-model="time" :minute-step="5" />
      </div>

      <div class="rd-field">
        <span class="rd-label">Повтор</span>
        <div class="rd-repeat">
          <button
            v-for="r in REPEATS"
            :key="r.key"
            type="button"
            class="rd-chip"
            :class="{ 'is-active': repeatKind === r.key }"
            @click="repeatKind = r.key"
          >{{ r.label }}</button>
        </div>
      </div>

      <div v-if="showInterval" class="rd-field rd-interval">
        <span class="rd-label">Каждые</span>
        <InputText v-model="repeatInterval" type="number" min="1" max="365" class="rd-num" />
        <span class="rd-label">{{ intervalSuffix }}</span>
      </div>

      <div v-if="showDays" class="rd-field">
        <span class="rd-label">Дни недели</span>
        <div class="rd-days">
          <button
            v-for="d in WEEKDAYS"
            :key="d.value"
            type="button"
            class="rd-day"
            :class="{ 'is-active': repeatDays.includes(d.value) }"
            @click="toggleDay(d.value)"
          >{{ d.label }}</button>
        </div>
      </div>
    </div>
  </AppDialog>
</template>

<style scoped>
.rd { display: flex; flex-direction: column; gap: 10px; }

.rd-field { display: flex; flex-direction: column; gap: 6px; }
.rd-interval { flex-direction: row; align-items: center; gap: 8px; }

.rd-label {
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
}

.rd-quick,
.rd-repeat,
.rd-days { display: flex; flex-wrap: wrap; gap: 6px; }

.rd-chip {
  padding: 5px 12px;
  border: 1px solid var(--color-outline-variant);
  border-radius: 999px;
  background: var(--glass-bg);
  color: var(--color-text);
  font-size: 12px;
  cursor: pointer;
}

.rd-chip.is-active {
  border-color: transparent;
  background: var(--color-primary);
  color: var(--color-on-primary);
}

.rd-day {
  min-width: 38px;
  max-width: 38px;
  min-height: 34px;
  max-height: 34px;
  border: 1px solid var(--color-outline-variant);
  border-radius: var(--radius-sm);
  background: var(--glass-bg);
  color: var(--color-text);
  font-size: 12px;
  cursor: pointer;
}

.rd-day.is-active {
  border-color: transparent;
  background: var(--color-primary);
  color: var(--color-on-primary);
}

.rd-when { display: flex; gap: 8px; align-items: center; }
.rd-date { flex: 1; min-width: 0; }
.rd-num { max-width: 90px; }

.rd-link {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  background: var(--color-surface-variant);
  font-size: 12px;
  color: var(--color-text-muted);
}
</style>
