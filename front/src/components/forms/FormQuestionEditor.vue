<template>
  <AppCard class="qe" :gap="10">
    <!-- Шапка вопроса: рукоятка переноса, текст, тип и действия. -->
    <div class="qe-head">
      <button
        class="qe-grip"
        type="button"
        draggable="true"
        title="Перетащить вопрос"
        aria-label="Перетащить вопрос"
        @dragstart="$emit('dragstart', $event)"
        @dragend="$emit('dragend')"
      >
        <span class="material-symbols-outlined">drag_indicator</span>
      </button>

      <div class="qe-main">
        <div class="qe-row">
          <InputText
            :model-value="question.title"
            class="qe-title"
            placeholder="Текст вопроса"
            maxlength="500"
            @update:model-value="patch({ title: $event })"
          />
          <Select
            :model-value="question.type"
            class="qe-type"
            :options="typeOptions"
            option-label="label"
            option-value="value"
            @update:model-value="changeType($event)"
          />
        </div>

        <InputText
          :model-value="question.description"
          class="qe-desc"
          placeholder="Пояснение (необязательно)"
          maxlength="2000"
          @update:model-value="patch({ description: $event })"
        />
      </div>

      <div class="qe-tools">
        <AppButton
          variant="icon" size="sm" icon="content_copy"
          title="Дублировать вопрос" aria-label="Дублировать вопрос"
          @click="$emit('duplicate')"
        />
        <AppButton
          variant="icon" size="sm" tone="danger" icon="delete"
          title="Удалить вопрос" aria-label="Удалить вопрос"
          @click="$emit('remove')"
        />
      </div>
    </div>

    <!-- Настройки конкретного типа. -->
    <div v-if="isChoice(question.type)" class="qe-block">
      <AppStack :gap="6">
        <div v-for="(opt, i) in options" :key="i" class="qe-option">
          <span class="material-symbols-outlined qe-option-icon">{{ optionIcon }}</span>
          <InputText
            :model-value="opt"
            class="qe-option-text"
            placeholder="Вариант"
            maxlength="300"
            @update:model-value="setOption(i, $event)"
          />
          <!-- Ветвление: вариант уводит на другой раздел или сразу к отправке. -->
          <Select
            v-if="isBranching(question.type) && sectionChoices.length"
            :model-value="targetOf(opt)"
            class="qe-target"
            :options="sectionChoices"
            option-label="label"
            option-value="value"
            @update:model-value="setTarget(opt, $event)"
          />
          <AppButton
            variant="icon" size="sm" tone="danger" icon="close"
            title="Убрать вариант" aria-label="Убрать вариант"
            @click="removeOption(i)"
          />
        </div>
        <div class="qe-inline">
          <AppButton variant="text" size="sm" icon="add" label="Вариант" @click="addOption" />
          <AppSwitch
            :model-value="!!question.config.other"
            label="Свой вариант"
            @update:model-value="patchConfig({ other: $event })"
          />
          <AppSwitch
            :model-value="!!question.config.shuffle"
            label="Перемешивать"
            @update:model-value="patchConfig({ shuffle: $event })"
          />
        </div>
        <div v-if="question.type === 'checkbox'" class="qe-inline">
          <span class="qe-label">Выбрать</span>
          <InputNumber
            :model-value="question.config.min_choices || 0"
            class="qe-num" :min="0" :max="options.length" show-buttons
            @update:model-value="patchConfig({ min_choices: $event || 0 })"
          />
          <span class="qe-label">…</span>
          <InputNumber
            :model-value="question.config.max_choices || 0"
            class="qe-num" :min="0" :max="options.length" show-buttons
            @update:model-value="patchConfig({ max_choices: $event || 0 })"
          />
          <span class="qe-hint">0 — без ограничения</span>
        </div>
      </AppStack>
    </div>

    <div v-else-if="isBooking(question.type)" class="qe-block">
      <AppStack :gap="6">
        <div v-for="(opt, i) in options" :key="i" class="qe-option">
          <span class="material-symbols-outlined qe-option-icon">event_seat</span>
          <InputText
            :model-value="opt"
            class="qe-option-text"
            placeholder="Вариант записи (смена, время, место)"
            maxlength="300"
            @update:model-value="renameSlot(i, $event)"
          />
          <div class="qe-slot">
            <span class="qe-label">Мест</span>
            <InputNumber
              :model-value="capacityOf(opt)"
              class="qe-slot-num" :min="0" :max="10000" show-buttons
              @update:model-value="setCapacity(opt, $event || 0)"
            />
          </div>
          <AppButton
            variant="icon" size="sm" tone="danger" icon="close"
            title="Убрать вариант" aria-label="Убрать вариант"
            @click="removeOption(i)"
          />
        </div>
        <AppButton variant="text" size="sm" icon="add" label="Вариант" @click="addOption" />
        <span class="qe-hint">
          Отвечающий видит остаток мест; когда мест не осталось, вариант выбрать нельзя.
        </span>
      </AppStack>
    </div>

    <div v-else-if="isGrid(question.type)" class="qe-block">
      <AppStack :gap="8">
        <div class="qe-lists">
          <div class="qe-list">
            <span class="qe-label">Строки</span>
            <Textarea
              :model-value="(question.config.rows || []).join('\n')"
              class="qe-area" rows="4" auto-resize
              placeholder="По строке на пункт"
              @update:model-value="patchConfig({ rows: toList($event) })"
            />
          </div>
          <div class="qe-list">
            <span class="qe-label">Столбцы</span>
            <Textarea
              :model-value="(question.config.cols || []).join('\n')"
              class="qe-area" rows="4" auto-resize
              placeholder="По строке на пункт"
              @update:model-value="patchConfig({ cols: toList($event) })"
            />
          </div>
        </div>
        <AppSwitch
          :model-value="!!question.config.require_each_row"
          label="Ответ в каждой строке"
          @update:model-value="patchConfig({ require_each_row: $event })"
        />
      </AppStack>
    </div>

    <div v-else-if="question.type === 'scale'" class="qe-block qe-inline">
      <span class="qe-label">От</span>
      <Select
        :model-value="question.config.min ?? 1"
        class="qe-small" :options="[0, 1]"
        @update:model-value="patchConfig({ min: $event })"
      />
      <span class="qe-label">до</span>
      <Select
        :model-value="question.config.max ?? 5"
        class="qe-small" :options="[2, 3, 4, 5, 6, 7, 8, 9, 10]"
        @update:model-value="patchConfig({ max: $event })"
      />
      <InputText
        :model-value="question.config.min_label || ''"
        class="qe-half" placeholder="Подпись слева" maxlength="60"
        @update:model-value="patchConfig({ min_label: $event })"
      />
      <InputText
        :model-value="question.config.max_label || ''"
        class="qe-half" placeholder="Подпись справа" maxlength="60"
        @update:model-value="patchConfig({ max_label: $event })"
      />
    </div>

    <div v-else-if="question.type === 'rating'" class="qe-block qe-inline">
      <span class="qe-label">Делений</span>
      <Select
        :model-value="question.config.max ?? 5"
        class="qe-small" :options="[3, 4, 5, 6, 7, 8, 9, 10]"
        @update:model-value="patchConfig({ max: $event })"
      />
    </div>

    <div v-else-if="question.type === 'date'" class="qe-block qe-inline">
      <AppSwitch
        :model-value="!!question.config.with_time"
        label="Спрашивать время"
        @update:model-value="patchConfig({ with_time: $event })"
      />
    </div>

    <div v-else-if="question.type === 'file'" class="qe-block qe-inline">
      <span class="qe-label">Файлов</span>
      <InputNumber
        :model-value="question.config.max_files || 1"
        class="qe-num" :min="1" :max="MAX_FILES" show-buttons
        @update:model-value="patchConfig({ max_files: $event || 1 })"
      />
      <span class="qe-label">по</span>
      <InputNumber
        :model-value="question.config.max_size_mb || 10"
        class="qe-num" :min="1" :max="1024" show-buttons
        @update:model-value="patchConfig({ max_size_mb: $event || 10 })"
      />
      <span class="qe-label">МБ</span>
    </div>

    <div v-else-if="question.type === 'note'" class="qe-block">
      <Textarea
        :model-value="question.config.text || ''"
        class="qe-area" rows="3" auto-resize
        placeholder="Текст пояснения — его увидит отвечающий"
        maxlength="2000"
        @update:model-value="patchConfig({ text: $event })"
      />
    </div>

    <div
      v-else-if="['short_text', 'paragraph'].includes(question.type)"
      class="qe-block qe-inline"
    >
      <span class="qe-label">Проверка</span>
      <Select
        :model-value="validation.kind || 'none'"
        class="qe-type" :options="VALIDATIONS" option-label="label" option-value="value"
        @update:model-value="setValidation({ kind: $event })"
      />
      <InputText
        v-if="validation.kind === 'regex'"
        :model-value="validation.pattern || ''"
        class="qe-half" placeholder="Регулярное выражение"
        @update:model-value="setValidation({ pattern: $event })"
      />
      <InputText
        v-if="validation.kind === 'regex'"
        :model-value="validation.hint || ''"
        class="qe-half" placeholder="Подсказка человеку" maxlength="120"
        @update:model-value="setValidation({ hint: $event })"
      />
      <template v-if="['number', 'length'].includes(validation.kind)">
        <InputText
          :model-value="validation.min ?? ''"
          class="qe-small" placeholder="от"
          @update:model-value="setValidation({ min: $event })"
        />
        <InputText
          :model-value="validation.max ?? ''"
          class="qe-small" placeholder="до"
          @update:model-value="setValidation({ max: $event })"
        />
      </template>
    </div>

    <!-- Режим теста: баллы и правильный ответ. -->
    <div v-if="quiz && isGradable(question.type)" class="qe-block qe-quiz">
      <div class="qe-inline">
        <span class="qe-label">Баллы</span>
        <InputNumber
          :model-value="question.points || 0"
          class="qe-num" :min="0" :max="1000" show-buttons
          @update:model-value="patch({ points: $event || 0 })"
        />
        <span class="qe-label">Правильный ответ</span>
      </div>

      <Select
        v-if="['radio', 'dropdown'].includes(question.type)"
        :model-value="answerKey.value || null"
        class="qe-half" :options="options" placeholder="Выберите вариант" show-clear
        @update:model-value="setKey({ value: $event || '' })"
      />
      <MultiSelect
        v-else-if="question.type === 'checkbox'"
        :model-value="answerKey.values || []"
        class="qe-half" :options="options" placeholder="Верные варианты" display="chip"
        @update:model-value="setKey({ values: $event || [] })"
      />
      <InputText
        v-else-if="question.type === 'short_text'"
        :model-value="(answerKey.values || []).join(', ')"
        class="qe-half" placeholder="Принимаемые ответы через запятую"
        @update:model-value="setKey({ values: toCsv($event) })"
      />
      <InputNumber
        v-else-if="['scale', 'rating'].includes(question.type)"
        :model-value="answerKey.number ?? null"
        class="qe-num" :min="0" :max="10"
        @update:model-value="setKey({ number: $event ?? 0 })"
      />
      <div v-else-if="isGrid(question.type)" class="qe-grid-key">
        <div v-for="row in (question.config.rows || [])" :key="row" class="qe-inline">
          <span class="qe-label qe-grid-row">{{ row }}</span>
          <Select
            v-if="question.type === 'grid_radio'"
            :model-value="(answerKey.grid || {})[row] || null"
            class="qe-half" :options="question.config.cols || []" placeholder="Верный столбец" show-clear
            @update:model-value="setGridKey(row, $event)"
          />
          <MultiSelect
            v-else
            :model-value="(answerKey.grid || {})[row] || []"
            class="qe-half" :options="question.config.cols || []" placeholder="Верные столбцы" display="chip"
            @update:model-value="setGridKey(row, $event)"
          />
        </div>
      </div>

      <InputText
        :model-value="answerKey.feedback || ''"
        class="qe-desc" placeholder="Пояснение к ответу (покажем вместе с оценкой)" maxlength="500"
        @update:model-value="setKey({ feedback: $event })"
      />
    </div>

    <!-- Условное отображение: вопрос выводится, только если на ПРЕДЫДУЩИЙ
         вопрос дан ожидаемый ответ. Источником бывает лишь уже сохранённый
         вопрос — у нового id ещё нет, и ссылаться не на что. -->
    <div v-if="isAnswerable(question.type)" class="qe-block qe-inline">
      <span class="qe-label">Показывать, если</span>
      <Select
        :model-value="question.config.visible_question_id || null"
        class="qe-half"
        :options="sourceChoices"
        option-label="label"
        option-value="value"
        :placeholder="sourceChoices.length ? 'всегда' : 'нет предыдущих вопросов'"
        :disabled="!sourceChoices.length"
        show-clear
        @update:model-value="setVisibleSource($event)"
      />
      <MultiSelect
        v-if="visibleSource && sourceOptions.length"
        :model-value="question.config.visible_values || []"
        class="qe-half"
        :options="sourceOptions"
        placeholder="любой ответ"
        display="chip"
        @update:model-value="patchConfig({ visible_values: $event || [] })"
      />
      <span v-else-if="visibleSource" class="qe-hint">на него вообще ответили</span>
      <span v-else-if="!sourceChoices.length" class="qe-hint">
        сохраните структуру — и вопросы выше станут условием
      </span>
    </div>

    <div class="qe-foot">
      <AppSwitch
        v-if="isAnswerable(question.type)"
        :model-value="question.required"
        label="Обязательный"
        @update:model-value="patch({ required: $event })"
      />
    </div>
  </AppCard>
</template>

<script setup>
/* Редактор одного вопроса конструктора.

   Своего состояния не держит: вопрос приходит пропсом, любая правка уходит
   событием `update` целым объектом — так родитель остаётся единственным
   владельцем черновика структуры и не рискует потерять правку при
   перетаскивании. */
import { computed } from 'vue'
import InputNumber from 'primevue/inputnumber'
import InputText from 'primevue/inputtext'
import MultiSelect from 'primevue/multiselect'
import Select from 'primevue/select'
import Textarea from 'primevue/textarea'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppSwitch from '@/components/ui/AppSwitch.vue'
import {
  MAX_FILES, QUESTION_TYPES, defaultConfig, isAnswerable, isBooking, isBranching,
  isChoice, isGradable, isGrid,
} from '@/utils/formFields.js'

const props = defineProps({
  question: { type: Object, required: true },
  /** Разделы формы — цели ветвления (кроме своего). */
  sections: { type: Array, default: () => [] },
  /** Индекс раздела, которому принадлежит вопрос. */
  sectionIndex: { type: Number, default: 0 },
  quiz: { type: Boolean, default: false },
})
const emit = defineEmits(['update', 'remove', 'duplicate', 'dragstart', 'dragend'])

const typeOptions = QUESTION_TYPES.map((q) => ({ label: q.label, value: q.type }))

const VALIDATIONS = [
  { value: 'none', label: 'Без проверки' },
  { value: 'number', label: 'Число' },
  { value: 'email', label: 'Почта' },
  { value: 'url', label: 'Ссылка' },
  { value: 'regex', label: 'По шаблону' },
  { value: 'length', label: 'Длина текста' },
]

const options = computed(() => props.question.config?.options || [])
const validation = computed(() => props.question.config?.validation || { kind: 'none' })
const answerKey = computed(() => props.question.answer_key || {})

const optionIcon = computed(() =>
  props.question.type === 'checkbox' ? 'check_box_outline_blank' : 'radio_button_unchecked')

/* Цели ветвления: любой раздел, кроме своего, плюс «отправить». Разделы
   называются позициями — у нового раздела id ещё нет, и сервер переводит
   позицию в идентификатор при сохранении. */
const sectionChoices = computed(() => [
  { value: '', label: 'Дальше по порядку' },
  ...props.sections
    .map((s, i) => ({ index: i, label: s.title || `Раздел ${i + 1}` }))
    .filter((s) => s.index !== props.sectionIndex)
    .map((s) => ({ value: `#${s.index}`, label: `→ ${s.label}` })),
  { value: 'submit', label: '→ Отправить форму' },
])

function patch(fields) {
  emit('update', { ...props.question, ...fields })
}

function patchConfig(fields) {
  patch({ config: { ...props.question.config, ...fields } })
}

// Смена типа сбрасывает настройки прежнего: они относились к другому типу и в
// новом означали бы случайное.
function changeType(type) {
  patch({ type, config: defaultConfig(type), answer_key: {} })
}

/* Источники условия показа — сохранённые вопросы, идущие ВЫШЕ этого: условие
   на вопрос, которого человек ещё не видел, никогда не выполнится. У нового
   вопроса id ещё нет, поэтому он в источники не попадает (то же правило, что и
   у подразделов реестра). */
const allQuestions = computed(() => props.sections.flatMap((s) => s.questions || []))

const sourceChoices = computed(() => {
  const mine = allQuestions.value.findIndex((q) => q.key === props.question.key)
  const before = mine === -1 ? allQuestions.value : allQuestions.value.slice(0, mine)
  return before
    .filter((q) => q.id && isAnswerable(q.type))
    .map((q) => ({ value: q.id, label: q.title || `Вопрос ${q.id}` }))
})

const visibleSource = computed(() =>
  allQuestions.value.find((q) => q.id === props.question.config?.visible_question_id) || null)

// Значения условия предлагаем только у вопросов с вариантами; у остальных
// условие означает «на источник вообще ответили».
const sourceOptions = computed(() => {
  const q = visibleSource.value
  if (!q || (!isChoice(q.type) && !isBooking(q.type))) return []
  return q.config?.options || []
})

function setVisibleSource(id) {
  patchConfig({ visible_question_id: id || 0, visible_values: [] })
}

// ── Места «Записи» ──
function capacityOf(option) {
  return Number(props.question.config?.capacity?.[option]) || 0
}

function setCapacity(option, value) {
  patchConfig({ capacity: { ...(props.question.config?.capacity || {}), [option]: value } })
}

// renameSlot — переименование варианта переносит и его места: иначе после
// правки названия вариант остался бы без мест.
function renameSlot(i, text) {
  const prev = options.value[i]
  const next = [...options.value]
  next[i] = text
  const capacity = { ...(props.question.config?.capacity || {}) }
  capacity[text] = capacity[prev] ?? 0
  if (prev !== text) delete capacity[prev]
  patchConfig({ options: next, capacity })
}

function setOption(i, text) {
  const next = [...options.value]
  next[i] = text
  patchConfig({ options: next })
}

function addOption() {
  patchConfig({ options: [...options.value, `Вариант ${options.value.length + 1}`] })
}

function removeOption(i) {
  const removed = options.value[i]
  const next = options.value.filter((_, idx) => idx !== i)
  const targets = { ...(props.question.config?.targets || {}) }
  delete targets[removed]
  patchConfig({ options: next, targets })
}

function targetOf(option) {
  return props.question.config?.targets?.[option] || ''
}

function setTarget(option, target) {
  const targets = { ...(props.question.config?.targets || {}) }
  if (target) targets[option] = target
  else delete targets[option]
  patchConfig({ targets })
}

function setValidation(fields) {
  patchConfig({ validation: { ...validation.value, ...fields } })
}

function setKey(fields) {
  patch({ answer_key: { ...answerKey.value, ...fields } })
}

function setGridKey(row, value) {
  const grid = { ...(answerKey.value.grid || {}) }
  if (value == null || (Array.isArray(value) && !value.length)) delete grid[row]
  else grid[row] = value
  setKey({ grid })
}

// toList — построчный ввод в набор пунктов (сетки задаются списком строк).
function toList(text) {
  return String(text || '').split('\n').map((s) => s.trim()).filter(Boolean)
}

function toCsv(text) {
  return String(text || '').split(',').map((s) => s.trim()).filter(Boolean)
}
</script>

<style scoped>
.qe { border: 1px solid var(--acrylic-border); }

.qe-head { display: flex; gap: 8px; align-items: flex-start; }

.qe-grip {
  display: flex;
  padding: 4px 0 0;
  border: none;
  background: none;
  color: var(--color-text-dim);
  cursor: grab;
}
.qe-grip:active { cursor: grabbing; }

.qe-main { display: flex; flex: 1; min-width: 0; flex-direction: column; gap: 8px; }
.qe-tools { display: flex; gap: 4px; }

.qe-row { display: flex; gap: 8px; align-items: center; }
/* min-width: 0 обязателен — иначе поле не сожмётся уже содержимого и карточка
   поедет горизонтальной прокруткой. */
.qe-title { flex: 1; min-width: 0; }
.qe-type { width: 190px; flex: none; }
.qe-desc { width: 100%; }

.qe-block { display: flex; flex-direction: column; gap: 8px; align-self: stretch; }

/* `flex-direction: row` здесь обязателен: строка настроек часто несёт ОБА
   класса (`qe-block qe-inline`), и без него оставалась колонка из .qe-block —
   тогда `align-items: center` центрировал содержимое по горизонтали, и
   «Проверка» вставала посреди карточки вместо левого края. */
.qe-inline {
  display: flex;
  flex-direction: row;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
  align-self: stretch;
}
.qe-label { font-size: 12px; color: var(--color-text-dim); }
.qe-hint { font-size: 12px; color: var(--color-text-dim); }

/* Строка варианта переносится: у «Записи» в ней три контрола, и в узкой панели
   они не умещаются в ряд — без переноса поле названия сжималось до нечитаемого,
   а счётчик мест уезжал за край. */
.qe-option { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.qe-slot { display: flex; align-items: center; gap: 6px; }
.qe-slot-num { width: 132px; }
.qe-option-icon { font-size: 18px; color: var(--color-text-dim); }
.qe-option-text { flex: 1; min-width: 0; }
.qe-target { width: 200px; flex: none; }

.qe-lists { display: flex; gap: 10px; flex-wrap: wrap; }
.qe-list { display: flex; flex: 1; min-width: 200px; flex-direction: column; gap: 4px; }
.qe-area { width: 100%; }

.qe-half { flex: 1; min-width: 160px; }
.qe-small { width: 100px; flex: none; }
.qe-num { width: 130px; flex: none; }

.qe-quiz {
  padding: 10px;
  border: 1px dashed var(--color-outline-dim);
  border-radius: var(--radius-md);
}
.qe-grid-key { display: flex; flex-direction: column; gap: 6px; }
.qe-grid-row { min-width: 120px; }

.qe-foot { display: flex; justify-content: flex-end; }

@container (max-width: 620px) {
  .qe-row { flex-wrap: wrap; }
  .qe-type { width: 100%; }
  .qe-target { width: 100%; }
  .qe-option-text { flex: 1 1 100%; }
  .qe-slot { flex: 1 1 auto; }
}

@media (max-width: 620px) {
  .qe-row { flex-wrap: wrap; }
  .qe-type { width: 100%; }
  .qe-target { width: 100%; }
  .qe-option-text { flex: 1 1 100%; }
  .qe-slot { flex: 1 1 auto; }
}
</style>
