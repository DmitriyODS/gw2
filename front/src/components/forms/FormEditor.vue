<template>
  <div class="fe">
    <AppInfoBar
      v-if="dirty"
      class="fe-banner"
      tone="warning"
      icon="edit"
      message="Есть несохранённые изменения структуры"
      inline
    >
      <template #actions>
        <AppButton
          variant="filled" size="sm" icon="save" label="Сохранить"
          :loading="busy" @click="save"
        />
        <AppButton variant="text" size="sm" label="Отменить" :disabled="busy" @click="reset" />
      </template>
    </AppInfoBar>

    <div
      v-for="(section, si) in draft"
      :key="section.key"
      class="fe-section"
      :class="{ over: dropSection === si }"
      @dragover.prevent="onDragOver(si)"
      @drop.prevent="onDrop(si)"
    >
      <AppCard class="fe-section-head" :gap="8">
        <div class="fe-inline">
          <InputText
            v-model="section.title"
            class="fe-title"
            :placeholder="`Раздел ${si + 1}`"
            maxlength="200"
          />
          <AppButton
            variant="icon" size="sm" icon="content_copy"
            title="Дублировать раздел" aria-label="Дублировать раздел"
            @click="duplicateSection(si)"
          />
          <AppButton
            v-if="draft.length > 1"
            variant="icon" size="sm" tone="danger" icon="delete"
            title="Удалить раздел" aria-label="Удалить раздел"
            @click="removeSection(si)"
          />
        </div>
        <InputText
          v-model="section.description"
          class="fe-desc"
          placeholder="Описание раздела (необязательно)"
          maxlength="2000"
        />
        <!-- Переход после раздела. Ветвление по вариантам задаётся у вопроса и
             сильнее этого правила — так же, как в исходном образце. -->
        <div class="fe-inline">
          <span class="fe-label">После раздела</span>
          <Select
            v-model="section.next"
            class="fe-next"
            :options="nextChoices(si)"
            option-label="label"
            option-value="value"
          />
        </div>

        <!-- Условное отображение раздела: он выводится, только если на
             вопрос-источник дан один из ожидаемых ответов. Источником бывает
             лишь УЖЕ сохранённый вопрос — у нового id ещё нет. -->
        <div class="fe-inline">
          <span class="fe-label">Показывать, если</span>
          <Select
            :model-value="section.visible_question_id || null"
            class="fe-next"
            :options="sourceChoices(si)"
            option-label="label"
            option-value="value"
            :placeholder="sourceChoices(si).length ? 'всегда' : 'нет вопросов выше'"
            :disabled="!sourceChoices(si).length"
            show-clear
            @update:model-value="setSectionSource(section, $event)"
          />
          <MultiSelect
            v-if="section.visible_question_id && sourceOptions(section.visible_question_id).length"
            v-model="section.visible_values"
            class="fe-next"
            :options="sourceOptions(section.visible_question_id)"
            placeholder="любой ответ"
            display="chip"
          />
          <span v-else-if="section.visible_question_id" class="fe-label">
            на него вообще ответили
          </span>
        </div>
      </AppCard>

      <AppStack :gap="10">
        <FormQuestionEditor
          v-for="(q, qi) in section.questions"
          :key="q.key"
          :question="q"
          :sections="draft"
          :section-index="si"
          :quiz="quiz"
          @update="updateQuestion(si, qi, $event)"
          @remove="removeQuestion(si, qi)"
          @duplicate="duplicateQuestion(si, qi)"
          @dragstart="onDragStart(si, qi, $event)"
          @dragend="onDragEnd"
        />
      </AppStack>

      <div class="fe-add">
        <!-- Одна кнопка: пояснение — такой же вопрос с типом «Пояснение»,
             отдельный вход для него был лишним. -->
        <AppButton variant="glass" icon="add" label="Вопрос" @click="addQuestion(si)" />
      </div>
    </div>

    <div class="fe-foot">
      <AppButton variant="glass" icon="playlist_add" label="Добавить раздел" @click="addSection" />
      <span class="fe-spacer" />
      <AppButton
        variant="filled" icon="save" label="Сохранить структуру"
        :disabled="!dirty" :loading="busy" @click="save"
      />
    </div>
  </div>
</template>

<script setup>
/* Конструктор формы: разделы-страницы и вопросы внутри них.

   Черновик живёт ЗДЕСЬ и уезжает на сервер целиком по кнопке: структура —
   связная вещь (ветвление ссылается на разделы), и сохранять её по одному полю
   значило бы держать на сервере промежуточные состояния, в которых переход
   ведёт в никуда.

   Ключ `key` у раздела и вопроса — устойчивый идентификатор строки редактора: у
   новых элементов id ещё нет, а без ключа Vue переиспользовал бы DOM соседа при
   перетаскивании. */
import { computed, ref, watch } from 'vue'
import InputText from 'primevue/inputtext'
import MultiSelect from 'primevue/multiselect'
import Select from 'primevue/select'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AppStack from '@/components/ui/AppStack.vue'
import FormQuestionEditor from './FormQuestionEditor.vue'
import { defaultConfig, isAnswerable, isBooking, isChoice } from '@/utils/formFields.js'

const props = defineProps({
  form: { type: Object, required: true },
  /** save(sections) → Promise. */
  save: { type: Function, required: true },
})
const emit = defineEmits(['error', 'saved'])

const quiz = computed(() => !!props.form.quiz)
const draft = ref([])
const busy = ref(false)
let keySeq = 0

// snapshot — слепок черновика для сравнения: правка структуры не должна
// пропасть при переключении вкладок, но и «сохранить» без изменений не нужно.
const saved = ref('')
const dirty = computed(() => JSON.stringify(payload()) !== saved.value)

watch(() => props.form, (form) => {
  if (!form) return
  reset()
}, { immediate: true })

function reset() {
  const sections = props.form.sections?.length ? props.form.sections : [{ questions: [] }]
  draft.value = sections.map((s) => toDraftSection(s, sections))
  saved.value = JSON.stringify(payload())
}

function toDraftSection(section, all) {
  let next = ''
  if (section.next_action === 'submit') next = 'submit'
  else if (section.next_action === 'section' && section.next_section_id) {
    const i = all.findIndex((s) => s.id === section.next_section_id)
    if (i !== -1) next = `#${i}`
  }
  return {
    key: `s${++keySeq}`,
    id: section.id || 0,
    title: section.title || '',
    description: section.description || '',
    next,
    visible_question_id: section.visible_question_id || null,
    visible_values: section.visible_values || [],
    questions: (section.questions || []).map(toDraftQuestion),
  }
}

function toDraftQuestion(q) {
  return {
    key: `q${++keySeq}`,
    id: q.id || 0,
    type: q.type || 'short_text',
    title: q.title || '',
    description: q.description || '',
    required: !!q.required,
    config: { ...defaultConfig(q.type || 'short_text'), ...(q.config || {}) },
    points: q.points || 0,
    answer_key: { ...(q.answer_key || {}) },
  }
}

// nextChoices — куда ведёт раздел: дальше по порядку, к другому разделу или
// сразу к отправке (свой раздел целью не предлагаем — это петля).
function nextChoices(index) {
  return [
    { value: '', label: 'Следующий раздел' },
    ...draft.value
      .map((s, i) => ({ i, label: s.title || `Раздел ${i + 1}` }))
      .filter((s) => s.i !== index)
      .map((s) => ({ value: `#${s.i}`, label: `Перейти к «${s.label}»` })),
    { value: 'submit', label: 'Отправить форму' },
  ]
}

function addSection() {
  draft.value = [...draft.value, {
    key: `s${++keySeq}`, id: 0, title: '', description: '', next: '',
    visible_question_id: null, visible_values: [], questions: [],
  }]
}

/* duplicateSection — копия раздела со всеми вопросами. Идентификаторы
   обнуляются: копия — новые раздел и вопросы, иначе сервер обновил бы
   исходные. Ветвление копии не переносим — его цели у оригинала свои. */
function duplicateSection(i) {
  const src = draft.value[i]
  const copy = {
    ...JSON.parse(JSON.stringify(src)),
    key: `s${++keySeq}`,
    id: 0,
    title: src.title ? `${src.title} (копия)` : '',
    next: '',
    visible_question_id: null,
    visible_values: [],
    questions: src.questions.map((q) => ({
      ...JSON.parse(JSON.stringify(q)),
      key: `q${++keySeq}`,
      id: 0,
      config: { ...q.config, targets: {}, visible_question_id: 0, visible_values: [] },
    })),
  }
  const next = [...draft.value]
  next.splice(i + 1, 0, copy)
  draft.value = next
}

/* Источники условия показа — сохранённые вопросы разделов ВЫШЕ этого: условие
   на вопрос, которого человек ещё не видел, никогда не выполнится. У только
   что добавленного вопроса id ещё нет, и сослаться на него неоткуда. */
function sourceChoices(sectionIndex) {
  return draft.value
    .slice(0, sectionIndex)
    .flatMap((s) => s.questions)
    .filter((q) => q.id && isAnswerable(q.type))
    .map((q) => ({ value: q.id, label: q.title || `Вопрос ${q.id}` }))
}

function sourceOptions(questionId) {
  const q = draft.value.flatMap((s) => s.questions).find((x) => x.id === questionId)
  if (!q || (!isChoice(q.type) && !isBooking(q.type))) return []
  return q.config?.options || []
}

function setSectionSource(section, id) {
  section.visible_question_id = id || null
  section.visible_values = []
}

function removeSection(i) {
  draft.value = draft.value.filter((_, idx) => idx !== i)
  // Переходы на удалённый раздел сбрасываем: ссылка в никуда увела бы
  // отвечающего мимо половины формы.
  const gone = `#${i}`
  for (const s of draft.value) {
    if (s.next === gone) s.next = ''
    for (const q of s.questions) {
      const targets = q.config?.targets
      if (!targets) continue
      for (const key of Object.keys(targets)) {
        if (targets[key] === gone) delete targets[key]
      }
    }
  }
}

function addQuestion(si, type = 'short_text') {
  const section = draft.value[si]
  section.questions = [...section.questions, {
    key: `q${++keySeq}`,
    id: 0,
    type,
    title: '',
    description: '',
    required: false,
    config: defaultConfig(type),
    points: 0,
    answer_key: {},
  }]
}

function updateQuestion(si, qi, question) {
  draft.value[si].questions[qi] = { ...question, key: draft.value[si].questions[qi].key }
}

function removeQuestion(si, qi) {
  draft.value[si].questions = draft.value[si].questions.filter((_, i) => i !== qi)
}

function duplicateQuestion(si, qi) {
  const src = draft.value[si].questions[qi]
  const copy = {
    ...JSON.parse(JSON.stringify(src)),
    key: `q${++keySeq}`,
    id: 0, // копия — новый вопрос, иначе сервер обновил бы исходный
  }
  const list = [...draft.value[si].questions]
  list.splice(qi + 1, 0, copy)
  draft.value[si].questions = list
}

// ── Перетаскивание вопросов (в том числе между разделами) ──
const dragFrom = ref(null)
const dropSection = ref(null)

function onDragStart(si, qi, e) {
  dragFrom.value = { si, qi }
  e.dataTransfer.effectAllowed = 'move'
  // Safari не начинает перетаскивание без данных в буфере.
  e.dataTransfer.setData('text/plain', `${si}:${qi}`)
}

function onDragOver(si) {
  if (dragFrom.value) dropSection.value = si
}

function onDrop(si) {
  const from = dragFrom.value
  onDragEnd()
  if (!from) return
  const source = draft.value[from.si]
  const [moved] = source.questions.splice(from.qi, 1)
  if (!moved) return
  draft.value[si].questions = [...draft.value[si].questions, moved]
}

function onDragEnd() {
  dragFrom.value = null
  dropSection.value = null
}

// payload — структура в том виде, в каком её ждёт сервер: ветвление позициями
// разделов (id новых он выдаст сам).
function payload() {
  return draft.value.map((s) => ({
    id: s.id,
    title: s.title.trim(),
    description: s.description.trim(),
    next_action: s.next === 'submit' ? 'submit' : (s.next ? 'section' : 'next'),
    next_index: s.next && s.next !== 'submit' ? Number(s.next.slice(1)) : -1,
    visible_question_id: s.visible_question_id || null,
    visible_values: s.visible_values || [],
    questions: s.questions.map((q) => ({
      id: q.id,
      type: q.type,
      title: q.title.trim(),
      description: q.description.trim(),
      required: q.required,
      config: q.config,
      points: q.points,
      answer_key: q.answer_key,
    })),
  }))
}

async function save() {
  if (busy.value) return
  busy.value = true
  try {
    await props.save(payload())
    saved.value = JSON.stringify(payload())
    emit('saved')
  } catch (e) {
    emit('error', e?.message || 'Не удалось сохранить структуру')
  } finally {
    busy.value = false
  }
}

// reset — откат черновика к сохранённому состоянию (уход «не сохранять»).
defineExpose({ dirty, save, reset })
</script>

<style scoped>
.fe { display: flex; flex-direction: column; gap: 16px; min-width: 0; }

/* Баннер прилипает к верху прокрутки: конструктор длинный, и подсказка
   «есть несохранённое» в самом начале списка не видна ровно тогда, когда
   нужна. Фон непрозрачный — под ним проезжают карточки вопросов. */
.fe-banner {
  position: sticky;
  top: 0;
  z-index: 3;
  background: var(--color-surface);
  box-shadow: 0 6px 12px -10px var(--color-shadow, rgb(0 0 0 / 35%));
}

.fe-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
  border-radius: var(--radius-lg);
  /* Своих полей у раздела нет: их дают карточки внутри, а третий слой отступов
     на телефоне съедал половину ширины. Цель перетаскивания подсвечиваем
     контуром — он рисуется поверх и содержимое не сдвигает. */
  outline: 1px solid transparent;
  outline-offset: 3px;
}

.fe-section.over { outline-color: var(--color-primary); }
.fe-section-head { border-left: 3px solid var(--color-primary); }

.fe-inline { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.fe-title { flex: 1; min-width: 0; font-weight: 600; }
.fe-desc { width: 100%; }
.fe-next { width: 240px; }
.fe-label { font-size: 12px; color: var(--color-text-dim); }

.fe-add { display: flex; gap: 8px; flex-wrap: wrap; }
.fe-foot { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.fe-spacer { flex: 1; }

/* На узкой панели ряд кнопок не помещается в строку: распорка съедает место, и
   «Сохранить» уезжает за край (горизонтальной прокрутки у раздела быть не
   должно). Там кнопки встают колонкой во всю ширину. Дубль `@media` — для
   заводского WebView старых Android без `@container`. */
@container (max-width: 520px) {
  .fe-foot { flex-direction: column; align-items: stretch; }
  .fe-spacer { display: none; }
  .fe-add > * { flex: 1; }
}

@media (max-width: 520px) {
  .fe-foot { flex-direction: column; align-items: stretch; }
  .fe-spacer { display: none; }
  .fe-add > * { flex: 1; }
}

@container (max-width: 620px) {
  .fe-next { width: 100%; }
}

@media (max-width: 620px) {
  .fe-next { width: 100%; }
}
</style>
