<template>
  <AppDialog
    :model-value="modelValue"
    title="Управление позицией"
    :subtitle="title"
    size="md"
    :actions="[{ kind: 'cancel', label: 'Закрыть' }]"
    @update:model-value="$emit('update:modelValue', $event)"
    @cancel="$emit('update:modelValue', false)"
  >
    <AppTabs v-model="tab" :tabs="TABS" variant="tint" full-width />

    <!-- ── Выдать / вернуть ── -->
    <AppStack v-if="tab === 'action'" :gap="14" class="ri-pane">
      <AppInfoBar v-if="open" :tone="stateTone" :message="stateText" />

      <template v-if="!open">
        <!-- Получатель и ответственный — РАЗНЫЕ сведения: вещь уходит в отдел
             или на объект, а отвечает за неё конкретный человек. -->
        <div class="ri-row">
          <div class="ri-field">
            <span class="ri-label">Кому выдаём</span>
            <InputText
              v-model="form.issued_to"
              placeholder="Отдел, объект, бригада"
              maxlength="200"
              :invalid="touched && !hasRecipient"
            />
          </div>
        </div>
        <div class="ri-row">
          <div class="ri-field">
            <span class="ri-label">ФИО ответственного</span>
            <InputText
              v-model="form.holder_name"
              placeholder="Иванов Иван Иванович"
              maxlength="200"
              :invalid="touched && !hasHolder"
            />
          </div>
          <div class="ri-field">
            <span class="ri-label">Телефон</span>
            <InputText
              v-model="form.holder_phone"
              placeholder="+7 (900) 000-00-00"
              :invalid="touched && !hasPhone"
            />
          </div>
        </div>
      </template>

      <!-- Срок: дни и дата возврата — два вида одного значения, поэтому правка
           любого пересчитывает второе. -->
      <template v-if="!open || action === 'extend'">
        <div class="ri-row">
          <div class="ri-field ri-field-sm">
            <span class="ri-label">На сколько дней</span>
            <InputNumber v-model="days" :min="1" :max="3650" @update:model-value="daysToDate" />
          </div>
          <div class="ri-field">
            <span class="ri-label">Дата возврата</span>
            <DatePicker
              v-model="dueDate"
              date-format="dd.mm.yy"
              placeholder="Выберите день"
              show-button-bar
              @update:model-value="dateToDays"
            />
          </div>
        </div>
        <AppSwitch v-model="skipWeekends" label="Считать только рабочие дни" @update:model-value="daysToDate" />
        <span class="ri-hint">
          Выходными считаются суббота и воскресенье. Праздники платформе неизвестны — их учтите сами.
        </span>
      </template>

      <div class="ri-field">
        <span class="ri-label">Комментарий</span>
        <Textarea v-model="form.comment" rows="2" auto-resize placeholder="Необязательно" maxlength="1000" />
      </div>

      <!-- Позиция на месте — её выдают; позиция на руках — продлевают или
           принимают обратно. Ветки разведены ОДНИМ условием: предупреждение о
           незаполненных полях живёт ВНУТРИ своей ветки, иначе оно разрывало
           цепочку v-if/v-else, и у невыданной позиции появлялось «Продлить». -->
      <AppStack row :gap="8">
        <template v-if="!open">
          <AppButton
            variant="filled" icon="output" label="Выдать"
            :loading="busy" :disabled="!canIssue"
            :title="canIssue ? '' : 'Укажите получателя, ФИО ответственного и телефон'"
            @click="submitIssue"
          />
          <span v-if="touched && !canIssue" class="ri-warn">
            Заполните получателя, ФИО ответственного и телефон.
          </span>
        </template>
        <template v-else>
          <AppButton
            v-if="action !== 'extend'"
            variant="filled" icon="schedule" label="Продлить"
            @click="startExtend"
          />
          <AppButton
            v-else
            variant="filled" icon="check" label="Сохранить срок"
            :loading="busy" @click="submitExtend"
          />
          <AppButton
            variant="glass" tone="success" icon="input" label="Вернуть"
            :loading="busy" @click="submitReturn"
          />
        </template>
      </AppStack>
    </AppStack>

    <!-- ── История ── -->
    <div v-else class="ri-pane">
      <div v-if="loading" class="ri-empty">Загрузка…</div>
      <EmptyState
        v-else-if="!history.length"
        size="sm"
        icon="history"
        title="Движений не было"
        subtitle="Позиция ни разу не выдавалась."
      />
      <ul v-else class="ri-history">
        <li v-for="issue in history" :key="issue.id" class="ri-issue">
          <div class="ri-issue-head">
            <span class="ri-issue-who">{{ issue.issued_to || issue.holder_name || 'Не указан' }}</span>
            <span v-if="issue.holder_name" class="ri-issue-phone">
              отв. {{ issue.holder_name }}<template v-if="issue.holder_phone">, {{ issue.holder_phone }}</template>
            </span>
            <span class="ri-spacer" />
            <AppChip
              :label="issue.returned_at ? 'возвращено' : 'на руках'"
              :tone="issue.returned_at ? 'success' : 'warning'"
            />
          </div>
          <ul class="ri-events">
            <li v-for="e in issue.events || []" :key="e.id" class="ri-event">
              <span class="ri-event-kind">{{ EVENT_LABEL[e.kind] || e.kind }}</span>
              <span class="ri-event-when">{{ when(e.created_at) }}</span>
              <span v-if="e.due_at" class="ri-event-due">до {{ day(e.due_at) }}</span>
              <span v-if="e.actor_name" class="ri-event-actor">· {{ e.actor_name }}</span>
              <span v-if="e.comment" class="ri-event-comment">{{ e.comment }}</span>
            </li>
          </ul>
        </li>
      </ul>
    </div>
  </AppDialog>
</template>

<script setup>
/* Учётный реестр: выдача позиции под ответственного и вся история движений.

   Срок задаётся ЛИБО количеством дней, ЛИБО датой возврата: правка одного
   пересчитывает второе (флажок «только рабочие дни» — тоже). Считаем здесь, на
   клиенте: сервер хранит готовую дату, потому что «через 5 дней» без точки
   отсчёта смысла не имеет, а «выдано до» нужно показывать и через месяц. */
import { computed, ref, watch } from 'vue'
import DatePicker from 'primevue/datepicker'
import InputNumber from 'primevue/inputnumber'
import InputText from 'primevue/inputtext'
import Textarea from 'primevue/textarea'
import AppButton from '@/components/ui/AppButton.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppSwitch from '@/components/ui/AppSwitch.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import { normalizePhone, validPhone } from '@/utils/registryFields.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  registryId: { type: [Number, null], default: null },
  record: { type: Object, default: null },
  title: { type: String, default: '' },
  /* issue(recordId, body) / extend(recordId, body) / back(recordId, comment) /
     history(recordId) — раздел ходит своими ручками, публичная страница по коду
     ссылки; сам диалог про источник ничего не знает. */
  issue: { type: Function, required: true },
  extend: { type: Function, required: true },
  back: { type: Function, required: true },
  history: { type: Function, required: true },
})
const emit = defineEmits(['update:modelValue', 'error'])

const TABS = [
  { value: 'action', label: 'Выдать / вернуть' },
  { value: 'history', label: 'История' },
]
const EVENT_LABEL = { issue: 'Выдано', extend: 'Продлено', return: 'Возвращено' }

const tab = ref('action')
const busy = ref(false)
const loading = ref(false)
const history = ref([])
const action = ref('')

const form = ref({ issued_to: '', holder_name: '', holder_phone: '', comment: '' })
const days = ref(7)
const dueDate = ref(null)
const skipWeekends = ref(false)

const open = computed(() => props.record?.issue || null)

const stateText = computed(() => {
  const i = open.value
  if (!i) return ''
  const who = i.issued_to || i.holder_name
  const resp = i.holder_name && i.holder_name !== who ? ` (отв. ${i.holder_name})` : ''
  if (!i.due_at) return `На руках у «${who}»${resp}, срок не назначен.`
  const overdue = overdueDays(i.due_at)
  return overdue > 0
    ? `Просрочено на ${overdue} дн. — у «${who}»${resp} с ${day(i.issued_at)}.`
    : `Выдано «${who}»${resp} до ${day(i.due_at)}.`
})

const stateTone = computed(() => {
  const i = open.value
  if (!i?.due_at) return 'info'
  return overdueDays(i.due_at) > 0 ? 'error' : 'warning'
})

/* Выдавать без ответственного и его телефона нельзя: запись «неизвестно кому»
   не отвечает на главный вопрос учёта — у кого вещь и как его найти. То же
   правило стоит на сервере: форма не единственный вход. */
const hasRecipient = computed(() => form.value.issued_to.trim().length > 0)
const hasHolder = computed(() => form.value.holder_name.trim().length > 0)
const hasPhone = computed(() => validPhone(form.value.holder_phone))
const canIssue = computed(() => hasRecipient.value && hasHolder.value && hasPhone.value)

// Подсвечиваем незаполненное не сразу, а после первой попытки: краснеть на
// пустой форме, которую человек только открыл, незачем.
const touched = ref(false)

function overdueDays(due) {
  const diff = Date.now() - new Date(due).getTime()
  return diff > 0 ? Math.ceil(diff / 86400000) : 0
}

function day(iso) {
  const d = new Date(iso)
  return isNaN(d) ? '' : d.toLocaleDateString('ru-RU')
}

function when(iso) {
  const d = new Date(iso)
  return isNaN(d) ? '' : d.toLocaleString('ru-RU', { dateStyle: 'short', timeStyle: 'short' })
}

// daysToDate — прибавить срок к сегодняшнему дню, при надобности перешагивая
// выходные.
function daysToDate() {
  const n = Number(days.value) || 0
  const d = new Date()
  d.setHours(12, 0, 0, 0)
  let left = n
  while (left > 0) {
    d.setDate(d.getDate() + 1)
    if (!skipWeekends.value || (d.getDay() !== 0 && d.getDay() !== 6)) left--
  }
  dueDate.value = d
}

// dateToDays — обратный пересчёт: сколько дней до выбранной даты.
function dateToDays() {
  if (!dueDate.value) return
  const target = new Date(dueDate.value)
  target.setHours(12, 0, 0, 0)
  const d = new Date()
  d.setHours(12, 0, 0, 0)
  let n = 0
  while (d < target) {
    d.setDate(d.getDate() + 1)
    if (!skipWeekends.value || (d.getDay() !== 0 && d.getDay() !== 6)) n++
  }
  days.value = n
}

watch(() => props.modelValue, (isOpen) => {
  if (!isOpen) return
  tab.value = 'action'
  action.value = ''
  form.value = { issued_to: '', holder_name: '', holder_phone: '', comment: '' }
  touched.value = false
  days.value = 7
  skipWeekends.value = false
  daysToDate()
  loadHistory()
})

watch(tab, (t) => {
  if (t === 'history') loadHistory()
})

async function loadHistory() {
  if (!props.record) return
  loading.value = true
  try {
    const d = await props.history(props.record.id)
    history.value = d.issues ?? []
  } catch (e) {
    emit('error', e?.message || 'Не удалось загрузить историю')
  } finally {
    loading.value = false
  }
}

function startExtend() {
  action.value = 'extend'
  dueDate.value = open.value?.due_at ? new Date(open.value.due_at) : null
  dateToDays()
}

async function run(fn) {
  busy.value = true
  try {
    await fn()
    await loadHistory()
    action.value = ''
    form.value.comment = ''
  } catch (e) {
    emit('error', e?.message || 'Не удалось выполнить действие')
  } finally {
    busy.value = false
  }
}

const submitIssue = () => {
  touched.value = true
  if (!canIssue.value) return
  return run(() => props.issue(props.record.id, {
    issued_to: form.value.issued_to.trim(),
    holder_name: form.value.holder_name.trim(),
    holder_phone: normalizePhone(form.value.holder_phone),
    due_at: dueDate.value ? new Date(dueDate.value).toISOString() : null,
    comment: form.value.comment.trim(),
  }))
}

const submitExtend = () => run(() => props.extend(props.record.id, {
  due_at: dueDate.value ? new Date(dueDate.value).toISOString() : null,
  comment: form.value.comment.trim(),
}))

const submitReturn = () => run(() => props.back(props.record.id, form.value.comment.trim()))
</script>

<style scoped>
.ri-pane { padding-top: 14px; padding-bottom: 2px; }

.ri-row { display: flex; gap: 10px; flex-wrap: wrap; }

.ri-field {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 6px;
}

/* Ширины раздаются ТОЛЬКО полям в строке: у поля, лежащего прямо в колоночной
   стопке (комментарий), flex-basis стал бы ВЫСОТОЙ — под текстовым полем
   зияла пустота в двести пикселей.
   min-width: 0 обязателен: без него поле не сжимается уже своего содержимого и
   наезжает на соседнее. */
.ri-row > .ri-field { flex: 1 1 200px; }
.ri-row > .ri-field-sm { flex: 0 1 160px; }

/* Контролы PrimeVue по умолчанию меряются своим содержимым — растягиваем их на
   всю колонку, иначе они торчат за её край. */
.ri-field :deep(.p-inputnumber),
.ri-field :deep(.p-datepicker),
.ri-field :deep(.p-inputtext),
.ri-field :deep(input),
.ri-field :deep(textarea) { width: 100%; min-width: 0; }
.ri-label { font-size: 13px; color: var(--color-text-dim); }
.ri-hint { font-size: 12px; line-height: 1.4; color: var(--color-text-dim); }
.ri-warn { align-self: center; font-size: 12px; color: var(--color-error); }
.ri-empty { padding: 16px; text-align: center; font-size: 14px; color: var(--color-text-dim); }

.ri-history { display: flex; flex-direction: column; gap: 10px; margin: 0; padding: 0; list-style: none; }

.ri-issue {
  padding: 10px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: var(--acrylic-card-bg);
}

.ri-issue-head { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.ri-issue-who { font-size: 14px; font-weight: 600; overflow-wrap: anywhere; }
.ri-issue-phone { font-size: 12px; color: var(--color-text-dim); }
.ri-spacer { flex: 1; }

.ri-events { display: flex; flex-direction: column; gap: 4px; margin: 8px 0 0; padding: 0; list-style: none; }

.ri-event {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  font-size: 12px;
  color: var(--color-text-dim);
}

.ri-event-kind { font-weight: 600; color: var(--color-text); }
.ri-event-comment { flex-basis: 100%; overflow-wrap: anywhere; }
</style>
