<template>
  <AppDialog
    :model-value="modelValue"
    :title="who"
    :subtitle="subtitle"
    size="lg"
    :actions="[{ kind: 'cancel', label: 'Закрыть' }]"
    @update:model-value="$emit('update:modelValue', $event)"
    @cancel="$emit('update:modelValue', false)"
  >
    <AppStack :gap="12">
      <div v-if="quiz" class="rd-score">
        <span class="rd-score-value">{{ response?.score }} / {{ response?.max_score }}</span>
        <AppChip
          size="sm"
          :tone="response?.graded ? 'success' : 'warning'"
          :label="response?.graded ? 'Оценка открыта' : 'Оценка скрыта'"
        />
        <span class="rd-spacer" />
        <AppButton
          v-if="!response?.graded && canEdit"
          variant="glass" size="sm" icon="visibility" label="Открыть оценку"
          :loading="busy"
          @click="publish"
        />
      </div>

      <div v-for="q in questions" :key="q.id" class="rd-item">
        <span class="rd-q">{{ q.title || 'Вопрос без названия' }}</span>

        <!-- Файлы скачиваются общим механизмом платформы: своя ссылка
             download в обёртке молча не срабатывает. -->
        <div v-if="q.type === 'file' && fileList(q).length" class="rd-files">
          <AppButton
            v-for="file in fileList(q)"
            :key="file.path"
            variant="glass"
            size="sm"
            icon="download"
            :label="file.name || 'Файл'"
            @click="download(file)"
          />
        </div>
        <span v-else-if="answerText(q, response?.answers?.[String(q.id)])" class="rd-a">
          {{ answerText(q, response?.answers?.[String(q.id)]) }}
        </span>
        <span v-else class="rd-empty">Без ответа</span>
      </div>
    </AppStack>
  </AppDialog>
</template>

<script setup>
/* Разбор одного ответа: вопрос — ответ, файлы и оценка теста. */
import { computed, ref } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppStack from '@/components/ui/AppStack.vue'
import { answerText, fileUrl, isAnswerable } from '@/utils/formFields.js'
import { allQuestions } from '@/utils/formFlow.js'
import { saveUrl } from '@/utils/download.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  form: { type: Object, default: null },
  response: { type: Object, default: null },
  canEdit: { type: Boolean, default: false },
  /** publish(responseId) → Promise (открыть оценку теста). */
  publishGrades: { type: Function, default: null },
})
const emit = defineEmits(['update:modelValue', 'error'])

const busy = ref(false)

const quiz = computed(() => !!props.form?.quiz)
const questions = computed(() =>
  allQuestions(props.form?.sections || []).filter((q) => isAnswerable(q.type)))

const who = computed(() =>
  props.response?.user_name?.trim() || props.response?.name?.trim() || 'Аноним')

const subtitle = computed(() => {
  if (!props.response) return ''
  const d = new Date(props.response.created_at)
  const time = Number.isNaN(d.getTime()) ? '' : d.toLocaleString('ru-RU')
  return [time, props.response.email].filter(Boolean).join(' · ')
})

function fileList(q) {
  const v = props.response?.answers?.[String(q.id)]
  return Array.isArray(v) ? v : []
}

function download(file) {
  saveUrl(fileUrl(file), file.name || 'file')
}

async function publish() {
  if (!props.publishGrades || busy.value) return
  busy.value = true
  try {
    await props.publishGrades(props.response.id)
  } catch (e) {
    emit('error', e?.message || 'Не удалось открыть оценку')
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.rd-score { display: flex; align-items: center; gap: 10px; }
.rd-score-value { font-size: 20px; font-weight: 700; color: var(--color-primary); }
.rd-spacer { flex: 1; }

.rd-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: var(--acrylic-card-bg);
}

.rd-q { font-size: 13px; color: var(--color-text-dim); overflow-wrap: anywhere; }
.rd-a { font-size: 14px; overflow-wrap: anywhere; white-space: pre-wrap; }
.rd-empty { font-size: 14px; color: var(--color-text-dim); font-style: italic; }
.rd-files { display: flex; gap: 8px; flex-wrap: wrap; }
</style>
