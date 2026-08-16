<template>
  <AppStack :gap="14">
    <AppCard title="Приём ответов" :gap="4">
      <AppRow title="Состояние" hint="Черновик не открывается даже по ссылке" inline>
        <Select
          :model-value="form.status"
          class="fst-status"
          :options="STATUSES"
          option-label="label"
          option-value="value"
          :disabled="!canEdit"
          @update:model-value="patch({ status: $event })"
        />
      </AppRow>
      <AppSwitchRow
        :model-value="form.allow_anonymous"
        title="Отвечать без входа"
        hint="Выключено — по ссылке пустят только вошедших в аккаунт"
        :disabled="!canEdit"
        @update:model-value="patch({ allow_anonymous: $event })"
      />
      <AppSwitchRow
        :model-value="form.one_response"
        title="Один ответ от человека"
        hint="Работает для вошедших: гость по ссылке не опознаётся"
        :disabled="!canEdit"
        @update:model-value="patch({ one_response: $event })"
      />
      <AppSwitchRow
        :model-value="form.allow_edit"
        title="Разрешить менять свой ответ"
        hint="Пока приём открыт, отвечающий может исправить отправленное"
        :disabled="!canEdit"
        @update:model-value="patch({ allow_edit: $event })"
      />
      <AppSwitchRow
        :model-value="form.collect_email"
        title="Спрашивать почту"
        hint="Отдельный обязательный вопрос в начале формы"
        :disabled="!canEdit"
        @update:model-value="patch({ collect_email: $event })"
      />
      <AppSwitchRow
        :model-value="form.collect_name"
        title="Спрашивать имя"
        hint="Вопрос «Как вас зовут» видят только гости по ссылке: вошедший подписан аккаунтом"
        :disabled="!canEdit"
        @update:model-value="patch({ collect_name: $event })"
      />
      <AppSwitchRow
        :model-value="form.show_progress"
        title="Показывать прогресс"
        hint="Полоса «раздел N из M» при заполнении"
        :disabled="!canEdit"
        @update:model-value="patch({ show_progress: $event })"
      />
    </AppCard>

    <AppCard title="Сроки и потолок" :gap="4">
      <AppRow title="Открыть с" hint="До этого времени форма не принимает ответы" inline>
        <DatePicker
          :model-value="toDate(form.opens_at)"
          class="fst-date"
          show-time hour-format="24" date-format="dd.mm.yy"
          show-icon icon-display="input" show-button-bar
          placeholder="Без ограничения"
          :disabled="!canEdit"
          @update:model-value="patch({ opens_at: toISO($event) })"
        />
      </AppRow>
      <AppRow title="Закрыть в" hint="После этого времени приём завершится сам" inline>
        <DatePicker
          :model-value="toDate(form.closes_at)"
          class="fst-date"
          show-time hour-format="24" date-format="dd.mm.yy"
          show-icon icon-display="input" show-button-bar
          placeholder="Без ограничения"
          :disabled="!canEdit"
          @update:model-value="patch({ closes_at: toISO($event) })"
        />
      </AppRow>
      <AppRow title="Максимум ответов" hint="0 — без ограничения" inline>
        <InputNumber
          :model-value="form.max_responses"
          class="fst-num"
          :min="0" :max="1000000" show-buttons
          :disabled="!canEdit"
          @update:model-value="patch({ max_responses: $event || 0 })"
        />
      </AppRow>
    </AppCard>

    <AppCard title="Режим теста" :gap="4">
      <AppSwitchRow
        :model-value="form.quiz"
        title="Это тест"
        hint="У вопросов появятся баллы и правильные ответы"
        :disabled="!canEdit"
        @update:model-value="patch({ quiz: $event })"
      />
      <template v-if="form.quiz">
        <AppRow title="Показывать оценку" inline>
          <Select
            :model-value="form.quiz_release"
            class="fst-status"
            :options="RELEASES"
            option-label="label"
            option-value="value"
            :disabled="!canEdit"
            @update:model-value="patch({ quiz_release: $event })"
          />
        </AppRow>
        <AppSwitchRow
          :model-value="form.quiz_show_answers"
          title="Показывать правильные ответы"
          hint="Вместе с оценкой отвечающий увидит разбор"
          :disabled="!canEdit"
          @update:model-value="patch({ quiz_show_answers: $event })"
        />
      </template>
    </AppCard>

    <AppCard title="После отправки" :gap="8">
      <Textarea
        :model-value="form.confirmation"
        class="fst-wide"
        rows="2"
        auto-resize
        maxlength="2000"
        placeholder="Спасибо! Ответ записан"
        :disabled="!canEdit"
        @update:model-value="patchDebounced({ confirmation: $event })"
      />
      <AppSwitchRow
        :model-value="form.show_summary"
        title="Показывать сводку отвечающему"
        hint="После отправки человек увидит общие результаты опроса"
        :disabled="!canEdit"
        @update:model-value="patch({ show_summary: $event })"
      />
    </AppCard>

    <AppCard title="Описание формы" :gap="8">
      <Textarea
        :model-value="form.description"
        class="fst-wide"
        rows="3"
        auto-resize
        maxlength="4000"
        placeholder="Расскажите, о чём эта форма"
        :disabled="!canEdit"
        @update:model-value="patchDebounced({ description: $event })"
      />
    </AppCard>
  </AppStack>
</template>

<script setup>
/* Настройки формы. Каждая правка уходит на сервер сразу отдельным ключом:
   сервер отличает «не трогать» от значения, поэтому сохранять форму целиком
   ради одного тумблера не нужно. Текстовые поля ждут паузы в наборе — иначе
   запрос уходил бы на каждую букву. */
import { onBeforeUnmount } from 'vue'
import DatePicker from 'primevue/datepicker'
import InputNumber from 'primevue/inputnumber'
import Select from 'primevue/select'
import Textarea from 'primevue/textarea'
import AppCard from '@/components/ui/AppCard.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppSwitchRow from '@/components/ui/AppSwitchRow.vue'

const props = defineProps({
  form: { type: Object, required: true },
  canEdit: { type: Boolean, default: false },
  /** save(patch) → Promise. */
  save: { type: Function, required: true },
})
const emit = defineEmits(['error'])

const STATUSES = [
  { value: 'draft', label: 'Черновик' },
  { value: 'open', label: 'Принимает ответы' },
  { value: 'closed', label: 'Приём закрыт' },
]

const RELEASES = [
  { value: 'immediately', label: 'Сразу после отправки' },
  { value: 'manual', label: 'После проверки автором' },
]

async function patch(fields) {
  if (!props.canEdit) return
  try {
    await props.save(fields)
  } catch (e) {
    emit('error', e?.message || 'Не удалось сохранить настройку')
  }
}

let timer = null
function patchDebounced(fields) {
  clearTimeout(timer)
  timer = setTimeout(() => patch(fields), 600)
}

onBeforeUnmount(() => clearTimeout(timer))

function toDate(value) {
  if (!value) return null
  const d = new Date(value)
  return Number.isNaN(d.getTime()) ? null : d
}

// Пустая строка означает «убрать срок» — сервер её так и понимает.
function toISO(date) {
  return date ? new Date(date).toISOString() : ''
}
</script>

<style scoped>
.fst-status { width: 220px; }
.fst-date { width: 240px; }
.fst-num { width: 150px; }
.fst-wide { width: 100%; }

@container (max-width: 620px) {
  .fst-status, .fst-date, .fst-num { width: 100%; }
}

@media (max-width: 620px) {
  .fst-status, .fst-date, .fst-num { width: 100%; }
}
</style>
