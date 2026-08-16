<template>
  <div class="ff">
    <!-- Шапка формы: название, описание и состояние приёма. -->
    <AppCard class="ff-head" :gap="8">
      <h2 class="ff-title">{{ form.title }}</h2>
      <p v-if="form.description" class="ff-desc">{{ form.description }}</p>
      <div v-if="form.quiz || totalPoints" class="ff-badges">
        <AppChip v-if="form.quiz" size="sm" icon="quiz" label="Тест" tone="primary" />
        <AppChip v-if="totalPoints" size="sm" icon="star" :label="`${totalPoints} баллов`" />
      </div>
    </AppCard>

    <!-- Результат: показывается после отправки (и при повторном заходе, если
         ответ уже есть и правка запрещена). -->
    <AppCard v-if="done" class="ff-done" :gap="10">
      <EmptyState
        icon="task_alt"
        tone="soft"
        :title="form.confirmation || 'Ответ записан'"
        :subtitle="doneSubtitle"
      />
      <div v-if="result?.graded" class="ff-score">
        <span class="ff-score-value">{{ result.score }} / {{ result.max_score }}</span>
        <span class="ff-score-label">баллов</span>
      </div>
      <AppStack :gap="8" row>
        <AppButton
          v-if="canEditMine"
          variant="glass" icon="edit" label="Изменить ответ"
          @click="answerAgain(true)"
        />
        <AppButton
          v-if="canAnswerMore"
          variant="glass" icon="add" label="Отправить ещё ответ"
          @click="answerAgain(false)"
        />
      </AppStack>
    </AppCard>

    <template v-else>
      <AppInfoBar
        v-if="!canRespond"
        tone="warning"
        icon="lock"
        :message="reason || 'Форма сейчас не принимает ответы'"
      />

      <!-- Прогресс по разделам: сколько страниц маршрута пройдено. -->
      <div v-if="form.show_progress && route.length > 1" class="ff-progress">
        <div class="ff-progress-bar">
          <span class="ff-progress-fill" :style="{ width: `${progressPercent}%` }" />
        </div>
        <span class="ff-progress-text">Раздел {{ stepNumber }} из {{ route.length }}</span>
      </div>

      <AppCard v-if="section" :gap="14">
        <div v-if="section.title || section.description" class="ff-section">
          <h3 v-if="section.title" class="ff-section-title">{{ section.title }}</h3>
          <p v-if="section.description" class="ff-section-desc">{{ section.description }}</p>
        </div>

        <!-- Почта отвечающего: спрашивается один раз, на первой странице. -->
        <div v-if="form.collect_email && isFirstStep" class="ff-question">
          <div class="ff-q-head">
            <span class="ff-q-title">Ваш адрес почты<span class="ff-req">*</span></span>
          </div>
          <InputText v-model="email" class="ff-wide" placeholder="name@example.com" maxlength="200" />
        </div>

        <!-- Имя гостя: у вошедшего оно берётся из аккаунта. -->
        <div v-if="askName && form.collect_name !== false && isFirstStep" class="ff-question">
          <div class="ff-q-head">
            <span class="ff-q-title">Как вас зовут</span>
          </div>
          <InputText v-model="name" class="ff-wide" placeholder="Имя" maxlength="200" />
        </div>

        <div
          v-for="q in shownQuestions"
          :key="q.id"
          class="ff-question"
          :class="{ 'is-wrong': reviewOf(q) === 'wrong', 'is-right': reviewOf(q) === 'right' }"
        >
          <div class="ff-q-head">
            <span class="ff-q-title">
              {{ q.title || 'Вопрос без названия' }}
              <span v-if="q.required" class="ff-req">*</span>
            </span>
            <AppChip v-if="form.quiz && q.points" size="sm" :label="`${q.points} б.`" />
          </div>
          <p v-if="q.description" class="ff-q-desc">{{ q.description }}</p>

          <FormAnswerInput
            :question="q"
            :model-value="answers[String(q.id)]"
            :disabled="!canRespond || busy"
            :error="errors[String(q.id)] || ''"
            :upload="uploadFor(q)"
            :taken="booking[String(q.id)] || {}"
            @update:model-value="setAnswer(q, $event)"
            @error="$emit('error', $event)"
          />
        </div>
      </AppCard>

      <div class="ff-actions">
        <AppButton
          v-if="!isFirstStep"
          variant="glass"
          icon="arrow_back"
          label="Назад"
          :disabled="busy"
          @click="back"
        />
        <span class="ff-spacer" />
        <AppButton
          v-if="hasNext"
          variant="filled"
          icon="arrow_forward"
          label="Далее"
          :disabled="!canRespond || busy"
          @click="next"
        />
        <AppButton
          v-else
          variant="filled"
          icon="send"
          label="Отправить"
          :disabled="!canRespond || busy"
          :loading="busy"
          @click="send"
        />
      </div>
    </template>
  </div>
</template>

<script setup>
/* Заполнение формы — один компонент на раздел и на публичную страницу.

   Маршрут по разделам считает utils/formFlow (зеркало domain/flow.go): выбор
   варианта уводит на другой раздел или сразу к отправке, поэтому «следующая
   страница» — не всегда соседняя, а обязательными считаются только вопросы
   пройденного маршрута. Проверку значений сервер всё равно повторит: здесь она
   ради того, чтобы человек узнал об ошибке до отправки длинной анкеты. */
import { computed, ref, watch } from 'vue'
import InputText from 'primevue/inputtext'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AppStack from '@/components/ui/AppStack.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import FormAnswerInput from './FormAnswerInput.vue'
import { emptyAnswer, isAnswerable, isCorrect, isFilled, validationError } from '@/utils/formFields.js'
import { nextSectionIndex, visibleQuestions, visitedSections } from '@/utils/formFlow.js'

const props = defineProps({
  form: { type: Object, required: true },
  canRespond: { type: Boolean, default: true },
  reason: { type: String, default: '' },
  /** Уже отправленный ответ (повторный показ и правка). */
  mine: { type: Object, default: null },
  /** Правильные ответы: приходят вместе с открытой оценкой теста. */
  answerKeys: { type: Object, default: null },
  /** Спрашивать имя (гость по ссылке). */
  askName: { type: Boolean, default: false },
  /** Занятые места вопросов «Запись»: {вопрос: {вариант: занято}}. */
  booking: { type: Object, default: () => ({}) },
  /** submit({ answers, email, name }) → Promise<{score, max_score, graded}>. */
  submit: { type: Function, required: true },
  /** upload(file, onProgress) → Promise<файл>. */
  upload: { type: Function, default: null },
})
const emit = defineEmits(['error', 'submitted'])

const answers = ref({})
const errors = ref({})
const email = ref('')
const name = ref('')
const busy = ref(false)
// editing — правим отправленный ответ (а не отправляем новый).
const editing = ref(false)
const step = ref(0)
const done = ref(false)
const result = ref(null)

const sections = computed(() => props.form.sections || [])
const section = computed(() => sections.value[step.value] || null)
// Показываем только вопросы, прошедшие своё условие: скрытые сервер и не
// спросит, и не сохранит.
const shownQuestions = computed(() => visibleQuestions(section.value, answers.value))

// Маршрут при текущих ответах: по нему считается прогресс и «последняя страница».
const route = computed(() => visitedSections(sections.value, answers.value))
const stepNumber = computed(() => {
  const i = route.value.findIndex((s) => s.id === section.value?.id)
  return i === -1 ? 1 : i + 1
})
const progressPercent = computed(() => Math.round((stepNumber.value / route.value.length) * 100))
const isFirstStep = computed(() => step.value === 0)
const hasNext = computed(() => nextSectionIndex(sections.value, step.value, answers.value) !== -1)

const totalPoints = computed(() => sections.value
  .flatMap((s) => s.questions || [])
  .reduce((sum, q) => sum + (q.points || 0), 0))

// Ответ уже есть: показываем его результат сразу, не заставляя заполнять заново.
watch(() => props.mine, (mine) => {
  reset()
  if (!mine) return
  answers.value = { ...(mine.answers || {}) }
  email.value = mine.email || ''
  editing.value = !!props.form.allow_edit
  if (!props.canRespond) {
    done.value = true
    result.value = { score: mine.score, max_score: mine.max_score, graded: mine.graded }
  }
}, { immediate: true })

/* Что можно после отправки: править свой ответ (если автор это разрешил) и
   отправить ещё один (если форма не требует единственного). Это разные вещи:
   «менять ответ» — про уже отправленное, «один ответ от человека» — про право
   ответить снова. */
const canEditMine = computed(() => props.canRespond && props.form.allow_edit && !!props.mine)
const canAnswerMore = computed(() => props.canRespond && !props.form.one_response)

const doneSubtitle = computed(() => {
  if (result.value?.graded) return 'Проверка теста завершена'
  if (props.form.quiz) return 'Оценку покажут после проверки'
  return props.form.allow_edit ? 'Вы можете изменить ответ, пока приём открыт' : 'Спасибо за ответ!'
})

function reset() {
  answers.value = {}
  errors.value = {}
  step.value = 0
  done.value = false
  editing.value = false
  result.value = null
}

function setAnswer(q, value) {
  answers.value = { ...answers.value, [String(q.id)]: value }
  if (errors.value[String(q.id)]) {
    const next = { ...errors.value }
    delete next[String(q.id)]
    errors.value = next
  }
}

// uploadFor — загрузка файла именно этого вопроса (потолок размера у каждого свой).
function uploadFor(q) {
  if (!props.upload) return null
  return (file, onProgress) => props.upload(file, q, onProgress)
}

/* validateStep — обязательные и неверные ответы ТЕКУЩЕГО раздела. Вопросы
   других разделов проверять рано: до них человек может и не дойти. */
function validateStep() {
  const found = {}
  for (const q of shownQuestions.value) {
    if (!isAnswerable(q.type)) continue
    const value = answers.value[String(q.id)]
    if (q.required && !isFilled(value)) {
      found[String(q.id)] = 'Обязательный вопрос'
      continue
    }
    const err = validationError(q, value)
    if (err) found[String(q.id)] = err
  }
  if (props.form.collect_email && isFirstStep.value && !email.value.trim()) {
    emit('error', 'Укажите адрес почты')
    return false
  }
  errors.value = found
  return Object.keys(found).length === 0
}

function next() {
  if (!validateStep()) return
  const i = nextSectionIndex(sections.value, step.value, answers.value)
  if (i === -1) return
  step.value = i
  scrollTop()
}

function back() {
  // Возвращаемся по фактически пройденному маршруту, а не по номеру раздела:
  // при ветвлении соседний раздел человек мог и не видеть.
  const i = route.value.findIndex((s) => s.id === section.value?.id)
  const prev = route.value[i - 1]
  step.value = prev ? sections.value.findIndex((s) => s.id === prev.id) : 0
  scrollTop()
}

async function send() {
  if (!validateStep() || busy.value) return
  busy.value = true
  try {
    // Отправляем только ответы пройденного маршрута: значения разделов, которых
    // человек не видел, остались бы от прежних попыток ветвления.
    const allowed = new Set(route.value
      .flatMap((s) => visibleQuestions(s, answers.value))
      .map((q) => String(q.id)))
    const payload = Object.fromEntries(
      Object.entries(answers.value).filter(([key, v]) => allowed.has(key) && isFilled(v)),
    )
    const res = await props.submit({
      answers: payload,
      email: email.value.trim(),
      name: name.value.trim(),
      // Правка отправленного и новая отправка идут разными ручками — какая
      // именно, знает только заполняющий.
      edit: editing.value && !!props.mine,
    })
    result.value = res
    done.value = true
    emit('submitted', res)
  } catch (e) {
    emit('error', e?.message || 'Не удалось отправить ответ')
  } finally {
    busy.value = false
  }
}

// answerAgain — вернуться к заполнению: правкой прежнего ответа либо новым
// (тогда форма очищается — прежние значения к новому ответу отношения не имеют).
function answerAgain(edit) {
  editing.value = edit
  if (!edit) blankAnswers()
  done.value = false
  step.value = 0
}

// blankAnswers — пустые значения по типам вопросов.
function blankAnswers() {
  const next = {}
  for (const q of sections.value.flatMap((s) => s.questions || [])) {
    if (isAnswerable(q.type)) next[String(q.id)] = emptyAnswer(q.type)
  }
  answers.value = next
  errors.value = {}
}

// reviewOf — разбор теста: подсветка верных и неверных ответов, когда оценка
// уже открыта («» — разбора нет).
function reviewOf(q) {
  if (!props.answerKeys || !props.form.quiz) return ''
  const key = props.answerKeys[String(q.id)]
  if (!key) return ''
  return isCorrect(q, answers.value[String(q.id)], key) ? 'right' : 'wrong'
}

function scrollTop() {
  document.querySelector('.ff')?.scrollIntoView?.({ behavior: 'smooth', block: 'start' })
}

// Новая форма — новое заполнение: значения прежней здесь ни к чему.
watch(() => props.form?.id, () => {
  reset()
  for (const q of sections.value.flatMap((s) => s.questions || [])) {
    if (isAnswerable(q.type)) answers.value[String(q.id)] = emptyAnswer(q.type)
  }
})
</script>

<style scoped>
.ff { display: flex; flex-direction: column; gap: 14px; min-width: 0; }

.ff-head { border-left: 3px solid var(--color-primary); }
.ff-title { margin: 0; font-size: 20px; font-weight: 600; overflow-wrap: anywhere; }
.ff-desc { margin: 0; font-size: 14px; color: var(--color-text-dim); overflow-wrap: anywhere; }
.ff-badges { display: flex; gap: 8px; flex-wrap: wrap; }

.ff-section { display: flex; flex-direction: column; gap: 4px; }
.ff-section-title { margin: 0; font-size: 16px; font-weight: 600; }
.ff-section-desc { margin: 0; font-size: 13px; color: var(--color-text-dim); }

.ff-question {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: var(--acrylic-card-bg);
}

.ff-question.is-right { border-color: var(--color-success); }
.ff-question.is-wrong { border-color: var(--color-error); }

.ff-q-head { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.ff-q-title { font-size: 15px; font-weight: 500; overflow-wrap: anywhere; }
.ff-q-desc { margin: 0; font-size: 13px; color: var(--color-text-dim); overflow-wrap: anywhere; }
.ff-req { color: var(--color-error); }
.ff-wide { width: 100%; }

.ff-progress { display: flex; align-items: center; gap: 10px; }
.ff-progress-bar {
  flex: 1;
  min-width: 0;
  height: 6px;
  border-radius: 3px;
  background: var(--color-surface-low);
  overflow: hidden;
}
.ff-progress-fill { display: block; height: 100%; background: var(--color-primary); transition: width 0.2s; }
.ff-progress-text { font-size: 12px; color: var(--color-text-dim); white-space: nowrap; }

.ff-actions { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.ff-spacer { flex: 1; }

/* На узкой панели «Назад» и «Далее» не умещаются в строку — тогда они идут
   колонкой во всю ширину, а распорка между ними не нужна. */
@container (max-width: 420px) {
  .ff-actions { flex-direction: column-reverse; align-items: stretch; }
  .ff-spacer { display: none; }
}

@media (max-width: 420px) {
  .ff-actions { flex-direction: column-reverse; align-items: stretch; }
  .ff-spacer { display: none; }
}

.ff-done { align-items: center; }
.ff-score { display: flex; align-items: baseline; gap: 8px; }
.ff-score-value { font-size: 28px; font-weight: 700; color: var(--color-primary); }
.ff-score-label { font-size: 14px; color: var(--color-text-dim); }
</style>
