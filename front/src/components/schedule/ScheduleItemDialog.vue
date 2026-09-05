<template>
  <AppDialog
    :model-value="modelValue"
    :title="isNew ? 'Новое занятие' : 'Занятие'"
    size="lg"
    :busy="saving"
    @update:model-value="dismiss"
  >
    <AppStack :gap="14">
      <!-- Название и короткое имя: второе показывается в узких колонках недели
           и на телефоне, где полное название не умещается. -->
      <AppField v-slot="{ id }" label="Название" required>
        <InputText :id="id" v-model="form.title" maxlength="200" placeholder="Например, Матанализ" />
      </AppField>

      <div class="si-row">
        <AppField v-slot="{ id }" label="Коротко" hint="Показывается в узких колонках">
          <InputText :id="id" v-model="form.short" maxlength="60" placeholder="Матан" />
        </AppField>
        <AppField v-slot="{ id }" label="Категория">
          <Select
            v-model="form.category_id"
            :input-id="id"
            :options="categoryOptions"
            option-label="label"
            option-value="value"
            placeholder="Без категории"
            show-clear
          />
        </AppField>
      </div>

      <div class="si-row">
        <AppField v-slot="{ id }" label="День недели" required>
          <Select
            v-model="form.weekday"
            :input-id="id"
            :options="weekdayOptions"
            option-label="label"
            option-value="value"
          />
        </AppField>
        <AppField label="Начало" required>
          <TimePicker v-model="startTime" />
        </AppField>
        <AppField label="Конец" required>
          <TimePicker v-model="endTime" />
        </AppField>
      </div>

      <!-- Повтор: либо недели цикла, либо своё правило. Два описания одного
           занятия противоречили бы друг другу, поэтому переключатель один. -->
      <AppField label="Повторяется">
        <AppTabs
          v-model="repeatMode"
          :tabs="[
            { value: 'cycle', label: cycleTabLabel },
            { value: 'own', label: 'Своё правило' },
          ]"
          variant="tint"
          dense
        />

        <div v-if="repeatMode === 'cycle'" class="si-weeks">
          <AppChip
            v-for="w in cycleWeeks"
            :key="w"
            :label="weekLabel(schedule, w)"
            :selected="isWeekOn(w)"
            interactive
            @click="toggleWeek(w)"
          />
          <span v-if="weeksHint" class="si-hint">{{ weeksHint }}</span>
        </div>

        <div v-else class="si-row si-own">
          <AppField v-slot="{ id }" label="Каждые (недель)">
            <InputNumber v-model="form.repeat_every" :input-id="id" :min="1" :max="52" show-buttons />
          </AppField>
          <AppField v-slot="{ id }" label="С недели" hint="Приводится к понедельнику">
            <DatePicker
              v-model="repeatFrom"
              :input-id="id"
              date-format="dd.mm.yy"
              show-icon
              icon-display="input"
              show-button-bar
              placeholder="Выберите"
            />
          </AppField>
          <AppField v-slot="{ id }" label="По неделю" hint="Можно не задавать">
            <DatePicker
              v-model="repeatUntil"
              :input-id="id"
              date-format="dd.mm.yy"
              show-icon
              icon-display="input"
              show-button-bar
              placeholder="Без конца"
            />
          </AppField>
        </div>
      </AppField>

      <!-- Дополнительные поля карточки: их набор задаёт само расписание. -->
      <div v-if="fields.length" class="si-grid">
        <AppField
          v-for="f in fields"
          :key="f.id"
          :label="f.label"
          :style="{ gridColumn: `span ${Math.min(f.col_span || 1, 2)}` }"
        >
          <FieldInput
            :field="f"
            :model-value="form.data[String(f.id)] ?? null"
            @update:model-value="form.data[String(f.id)] = $event"
          />
        </AppField>
      </div>

      <AppInfoBar v-if="error" tone="error" :message="error" compact />
    </AppStack>

    <template #footer>
      <AppButton
        v-if="!isNew"
        variant="glass"
        tone="danger"
        icon="delete"
        label="Удалить"
        :disabled="saving"
        @click="confirmDelete = true"
      />
      <span class="si-spacer" />
      <AppButton label="Отмена" :disabled="saving" @click="dismiss" />
      <AppButton
        variant="filled"
        :label="isNew ? 'Создать' : 'Сохранить'"
        :loading="saving"
        @click="save"
      />
    </template>

    <ConfirmDialog
      :visible="confirmDelete"
      header="Удалить занятие?"
      message="Занятие исчезнет из всех недель расписания."
      confirm-label="Удалить"
      danger-confirm
      @confirm="removeItem"
      @cancel="confirmDelete = false"
    />
  </AppDialog>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import DatePicker from 'primevue/datepicker'
import InputNumber from 'primevue/inputnumber'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import AppButton from '@/components/ui/AppButton.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppField from '@/components/ui/AppField.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import FieldInput from '@/components/common/FieldInput.vue'
import TimePicker from '@/components/common/TimePicker.vue'
import { WEEKDAYS, dateKey, hhmm, toMinutes, utcDay, weekLabel } from '@/utils/scheduleCycle.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  schedule: { type: Object, required: true },
  // item — правим существующее занятие; null — заводим новое.
  item: { type: Object, default: null },
  // preset — заготовка нового занятия (клик по пустому месту шкалы).
  preset: { type: Object, default: null },
  // submit/remove — операции раздела: диалог ждёт их и показывает отказ сервера.
  submit: { type: Function, required: true },
  remove: { type: Function, default: null },
})

const emit = defineEmits(['update:modelValue'])

const saving = ref(false)
const error = ref('')
const confirmDelete = ref(false)
const repeatMode = ref('cycle')

const isNew = computed(() => !props.item)
const fields = computed(() => props.schedule?.fields || [])
const cycleWeeks = computed(() =>
  Array.from({ length: props.schedule?.cycle_weeks || 1 }, (_, i) => i + 1))

const weekdayOptions = WEEKDAYS.map((d, i) => ({ value: i, label: d.full }))
const categoryOptions = computed(() =>
  (props.schedule?.categories || []).map((c) => ({ value: c.id, label: c.name })))

const cycleTabLabel = computed(() =>
  (props.schedule?.cycle_weeks || 1) > 1 ? 'Недели цикла' : 'Каждую неделю')

const form = reactive({
  title: '', short: '', weekday: 0, category_id: null,
  weeks: [], repeat_every: 2, repeat_from: '', repeat_until: '',
  data: {},
})
const startTime = ref('09:00')
const endTime = ref('10:30')

/* DatePicker работает с Date, а сервер обменивается днями «YYYY-MM-DD»:
   границу типов держим здесь, чтобы дата не уехала через полночь в чужом
   часовом поясе. */
function dateModel(key) {
  return computed({
    get: () => (form[key] ? utcDay(form[key]) : null),
    set: (value) => { form[key] = value ? dateKey(value) : '' },
  })
}
const repeatFrom = dateModel('repeat_from')
const repeatUntil = dateModel('repeat_until')

const weeksHint = computed(() => {
  if ((props.schedule?.cycle_weeks || 1) === 1) return 'В расписании одна неделя цикла.'
  return form.weeks.length ? '' : 'Ни одна не отмечена — занятие идёт каждую неделю.'
})

/* Форма наполняется на ОТКРЫТИИ диалога и при смене занятия — но не на каждое
   обновление самого объекта: занятие приходит из стора, и чужая правка по
   сокету посреди набора стёрла бы то, что человек уже напечатал. Следить за
   массивом-парой для этого нельзя: он каждый раз новый, и watch срабатывал на
   любое обновление. */
watch(() => props.modelValue, (open) => { if (open) fillForm() }, { immediate: true })
watch(() => props.item?.id ?? 0, () => { if (props.modelValue) fillForm() })

function fillForm() {
  error.value = ''
  confirmDelete.value = false
  const it = props.item
  form.title = it?.title || ''
  form.short = it?.short || ''
  form.weekday = it?.weekday ?? props.preset?.weekday ?? 0
  form.category_id = it?.category_id ?? null
  form.weeks = [...(it?.weeks || [])]
  form.repeat_every = it?.repeat_every || 2
  form.repeat_from = it?.repeat_from || ''
  form.repeat_until = it?.repeat_until || ''
  form.data = { ...(it?.data || {}) }
  startTime.value = hhmm(it?.start_min ?? props.preset?.start ?? 9 * 60)
  endTime.value = hhmm(it?.end_min ?? props.preset?.end ?? 10 * 60 + 30)
  repeatMode.value = it?.repeat_every > 0 ? 'own' : 'cycle'
}

function isWeekOn(week) { return form.weeks.includes(week) }

function toggleWeek(week) {
  form.weeks = isWeekOn(week) ? form.weeks.filter((w) => w !== week) : [...form.weeks, week].sort()
}

/* dismiss — попытка ЗАКРЫТЬ ДИАЛОГ РУКАМИ (крестик, фон, «Отмена»): пока идёт
   сохранение, она игнорируется. Закрытие после успешной операции идёт мимо неё
   (close): там saving ещё поднят, и общий guard оставлял диалог висеть с уже
   сохранённым занятием. */
function dismiss() {
  if (!saving.value) close()
}

function close() {
  emit('update:modelValue', false)
}

async function save() {
  const title = form.title.trim()
  if (!title) { error.value = 'Укажите название занятия'; return }
  const start = toMinutes(startTime.value)
  const end = toMinutes(endTime.value)
  if (end <= start) { error.value = 'Конец занятия должен быть позже начала'; return }

  const own = repeatMode.value === 'own'
  if (own && !form.repeat_from) { error.value = 'Укажите неделю, с которой начинается повтор'; return }

  const body = {
    weekday: form.weekday, start_min: start, end_min: end,
    title, short: form.short.trim(), category_id: form.category_id,
    weeks: own ? [] : form.weeks,
    repeat_every: own ? Math.min(52, Math.max(1, form.repeat_every || 1)) : null,
    repeat_from: own ? form.repeat_from : null,
    repeat_until: own && form.repeat_until ? form.repeat_until : null,
    data: form.data,
  }
  saving.value = true
  error.value = ''
  try {
    await props.submit(body)
    close()
  } catch (e) {
    error.value = e?.message || 'Не удалось сохранить занятие'
  } finally {
    saving.value = false
  }
}

async function removeItem() {
  confirmDelete.value = false
  saving.value = true
  try {
    await props.remove?.(props.item)
    close()
  } catch (e) {
    error.value = e?.message || 'Не удалось удалить занятие'
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.si-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(170px, 100%), 1fr));
  gap: 10px;
}
.si-own { padding-top: 10px; }

.si-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}
@container (max-width: 520px) {
  .si-grid { grid-template-columns: minmax(0, 1fr); }
}
@media (max-width: 520px) {
  .si-grid { grid-template-columns: minmax(0, 1fr); }
}

.si-weeks {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding-top: 10px;
}
.si-hint { font-size: 0.8rem; color: var(--color-text-dim); }
.si-spacer { flex: 1; }
</style>
