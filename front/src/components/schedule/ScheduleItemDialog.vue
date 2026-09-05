<template>
  <AppDialog
    :model-value="modelValue"
    :title="isNew ? 'Новое занятие' : 'Занятие'"
    size="lg"
    :busy="saving"
    @update:model-value="close"
  >
    <AppStack :gap="14">
      <!-- Название и короткое имя: второе показывается в узких колонках недели
           и на телефоне, где полное название не умещается. -->
      <div class="si-field">
        <label class="si-label">Название<span class="si-req">*</span></label>
        <input v-model="form.title" class="ctl" type="text" maxlength="200" placeholder="Например, Матанализ" />
      </div>

      <div class="si-row">
        <div class="si-field">
          <label class="si-label">Коротко</label>
          <input v-model="form.short" class="ctl" type="text" maxlength="60" placeholder="Матан" />
        </div>
        <div class="si-field">
          <label class="si-label">Категория</label>
          <select v-model="form.category_id" class="ctl">
            <option :value="null">Без категории</option>
            <option v-for="c in categories" :key="c.id" :value="c.id">{{ c.name }}</option>
          </select>
        </div>
      </div>

      <div class="si-row">
        <div class="si-field">
          <label class="si-label">День недели<span class="si-req">*</span></label>
          <select v-model.number="form.weekday" class="ctl">
            <option v-for="(d, i) in WEEKDAYS" :key="i" :value="i">{{ d.full }}</option>
          </select>
        </div>
        <div class="si-field">
          <label class="si-label">Начало<span class="si-req">*</span></label>
          <TimePicker v-model="startTime" />
        </div>
        <div class="si-field">
          <label class="si-label">Конец<span class="si-req">*</span></label>
          <TimePicker v-model="endTime" />
        </div>
      </div>

      <!-- Повтор: либо недели цикла, либо своё правило. Два описания одного
           занятия противоречили бы друг другу, поэтому переключатель один. -->
      <div class="si-field">
        <label class="si-label">Повторяется</label>
        <AppTabs
          v-model="repeatMode"
          :tabs="[
            { value: 'cycle', label: cycleTabLabel },
            { value: 'own', label: 'Своё правило' },
          ]"
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
          <span class="si-hint">{{ weeksHint }}</span>
        </div>

        <div v-else class="si-own">
          <div class="si-row">
            <div class="si-field">
              <label class="si-label">Каждые</label>
              <div class="si-inline">
                <input v-model.number="form.repeat_every" class="ctl si-num" type="number" min="1" max="52" />
                <span>нед.</span>
              </div>
            </div>
            <div class="si-field">
              <label class="si-label">С недели</label>
              <input v-model="form.repeat_from" class="ctl" type="date" />
            </div>
            <div class="si-field">
              <label class="si-label">По неделю</label>
              <input v-model="form.repeat_until" class="ctl" type="date" />
            </div>
          </div>
          <span class="si-hint">Дата приводится к понедельнику своей недели.</span>
        </div>
      </div>

      <!-- Дополнительные поля карточки: их набор задаёт само расписание. -->
      <div v-if="fields.length" class="si-grid">
        <div
          v-for="f in fields"
          :key="f.id"
          class="si-field"
          :style="{ gridColumn: `span ${Math.min(f.col_span || 1, 2)}` }"
        >
          <label class="si-label">{{ f.label }}</label>
          <FieldInput
            :field="f"
            :model-value="form.data[String(f.id)] ?? null"
            @update:model-value="form.data[String(f.id)] = $event"
          />
        </div>
      </div>

      <span v-if="error" class="si-error">{{ error }}</span>
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
      <AppButton label="Отмена" :disabled="saving" @click="close(false)" />
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
import AppButton from '@/components/ui/AppButton.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import FieldInput from '@/components/common/FieldInput.vue'
import TimePicker from '@/components/common/TimePicker.vue'
import { WEEKDAYS, hhmm, toMinutes, weekLabel } from '@/utils/scheduleCycle.js'

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
const categories = computed(() => props.schedule?.categories || [])
const fields = computed(() => props.schedule?.fields || [])
const cycleWeeks = computed(() =>
  Array.from({ length: props.schedule?.cycle_weeks || 1 }, (_, i) => i + 1))

const cycleTabLabel = computed(() =>
  (props.schedule?.cycle_weeks || 1) > 1 ? 'Недели цикла' : 'Каждую неделю')

const form = reactive({
  title: '', short: '', weekday: 0, category_id: null,
  weeks: [], repeat_every: 2, repeat_from: '', repeat_until: '',
  data: {},
})
const startTime = ref('09:00')
const endTime = ref('10:30')

const weeksHint = computed(() => {
  if ((props.schedule?.cycle_weeks || 1) === 1) return 'В расписании одна неделя цикла.'
  return form.weeks.length ? '' : 'Ни одна не отмечена — занятие идёт каждую неделю.'
})

watch(() => [props.modelValue, props.item, props.preset], () => {
  if (!props.modelValue) return
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
}, { immediate: true })

function isWeekOn(week) { return form.weeks.includes(week) }

function toggleWeek(week) {
  form.weeks = isWeekOn(week) ? form.weeks.filter((w) => w !== week) : [...form.weeks, week].sort()
}

function close(value = false) {
  if (saving.value) return
  emit('update:modelValue', value)
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
    close(false)
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
    close(false)
  } catch (e) {
    error.value = e?.message || 'Не удалось удалить занятие'
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.si-field { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.si-label { font-size: 12px; color: var(--color-on-surface-variant); }
.si-req { color: var(--color-error); margin-left: 2px; }

.si-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(160px, 100%), 1fr));
  gap: 10px;
}
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
  padding-top: 8px;
}
.si-own { display: flex; flex-direction: column; gap: 8px; padding-top: 8px; }
.si-inline { display: flex; align-items: center; gap: 8px; }
.si-num { max-width: 90px; }
.si-hint { font-size: 12px; color: var(--color-on-surface-variant); }
.si-error { font-size: 13px; color: var(--color-error); }
.si-spacer { flex: 1; }
</style>
