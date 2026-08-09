<template>
  <div ref="threadEl" class="hc" :class="{ compact }">
    <div v-if="assistant.loading && !assistant.messages.length" class="hc-center">
      <BrandLoader :size="48" />
    </div>

    <div v-else-if="!assistant.messages.length" class="hc-center hc-empty">
      <HolaIcon v-if="!compact" :size="40" />
      <p class="hc-empty-title">Спросите Hola о работе компании</p>
      <p v-if="!compact" class="hc-empty-sub">Отвечаю по реальным данным: часы, отделы, загрузка, поиск задач со ссылками.</p>
      <div class="hc-chips">
        <button
          v-for="s in SUGGESTIONS"
          :key="s"
          class="hc-chip"
          type="button"
          @click="$emit('suggest', s)"
        >{{ s }}</button>
      </div>
    </div>

    <template v-for="group in groups" :key="group.key">
      <div class="hc-day"><span>{{ group.label }}</span></div>
      <div
        v-for="m in group.items"
        :key="m.id"
        class="hc-row"
        :class="{ mine: m.role === 'user' }"
      >
        <div class="hc-msg">
          <div class="hc-bubble">
            <!-- Ответы ассистента приходят в Markdown (LLM), реплики
                 пользователя — простой текст с линкификацией. -->
            <MarkdownView v-if="m.role === 'assistant'" :source="m.text" />
            <LinkifiedText v-else :text="m.text" />
            <span class="hc-time">{{ timeOf(m.createdAt) }}</span>
          </div>
          <div v-if="m.role === 'assistant' && m.sources" class="hc-sources">{{ m.sources }}</div>
          <div v-if="canRate(m)" class="hc-fb">
            <button
              class="hc-fb-btn"
              :class="{ active: assistant.myFeedback[m.id] === 'up' }"
              type="button"
              title="Полезный ответ"
              aria-label="Полезный ответ"
              @click="rate(m, 'up')"
            >
              <span class="material-symbols-outlined">thumb_up</span>
            </button>
            <button
              class="hc-fb-btn"
              :class="{ active: assistant.myFeedback[m.id] === 'down' }"
              type="button"
              title="Неудачный ответ"
              aria-label="Неудачный ответ"
              @click="askReason(m)"
            >
              <span class="material-symbols-outlined">thumb_down</span>
            </button>
            <template v-if="reasonFor === m.id">
              <button
                v-for="r in FEEDBACK_REASONS"
                :key="r.value"
                class="hc-fb-chip"
                type="button"
                @click="rate(m, 'down', r.value)"
              >{{ r.label }}</button>
            </template>
          </div>
        </div>
      </div>
    </template>

    <div v-if="assistant.sending" class="hc-row">
      <div class="hc-bubble hc-typing">
        <span class="hc-dot" /><span class="hc-dot" /><span class="hc-dot" />
      </div>
    </div>

    <p v-if="assistant.error" class="hc-error">{{ assistant.error }}</p>
  </div>
</template>

<script setup>
/* Лента диалога с Hola. Ввод живёт в шапке окна (одно поле на все вкладки),
   поэтому здесь только история, подсказки пустого чата и оценка ответов. */
import { computed, nextTick, ref, watch } from 'vue'
import { useAssistantStore } from '@/stores/assistant.js'
import { dayLabel } from '@/utils/chatDates.js'
import HolaIcon from '@/components/common/HolaIcon.vue'
import MarkdownView from '@/components/common/MarkdownView.vue'
import LinkifiedText from '@/components/common/LinkifiedText.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'

defineProps({
  /** Панель зажата клавиатурой: лента и подсказки пустого чата ужимаются. */
  compact: { type: Boolean, default: false },
})

defineEmits(['suggest'])

const SUGGESTIONS = [
  'Сколько часов отработали на этой неделе?',
  'Топ сотрудников за месяц',
  'Найди задачу про отчёт',
]

const FEEDBACK_REASONS = [
  { value: 'inaccurate', label: 'Неточно' },
  { value: 'irrelevant', label: 'Не по теме' },
  { value: 'incomplete', label: 'Неполно' },
]

const assistant = useAssistantStore()
const threadEl = ref(null)
const reasonFor = ref(null)

/* Группировка по дням — как в мессенджере, но по своему полю createdAt
   (у ассистента своя нормализация сообщений). */
const groups = computed(() => {
  const out = []
  let cur = null
  for (const m of assistant.messages) {
    const d = new Date(m.createdAt)
    const key = `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`
    if (!cur || cur.key !== key) {
      cur = { key, label: dayLabel(m.createdAt), items: [] }
      out.push(cur)
    }
    cur.items.push(m)
  }
  return out
})

function timeOf(iso) {
  const d = new Date(iso)
  return Number.isNaN(d.getTime())
    ? ''
    : d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}

// Оценивать можно только сохранённый ответ: у локальных заглушек нет id в БД.
function canRate(m) {
  return m.role === 'assistant' && typeof m.id === 'number'
}

function askReason(m) {
  reasonFor.value = reasonFor.value === m.id ? null : m.id
}

function rate(m, verdict, reason = null) {
  assistant.sendFeedback(m.id, verdict, reason)
  if (verdict === 'up' || reason) reasonFor.value = null
}

function scrollDown() {
  nextTick(() => {
    const el = threadEl.value
    if (el) el.scrollTop = el.scrollHeight
  })
}

watch(() => [assistant.messages.length, assistant.sending], scrollDown, { immediate: true })

defineExpose({ scrollDown })
</script>

<style scoped>
.hc {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 4px 2px 8px;
  scrollbar-width: thin;
}

.hc-center {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 24px 16px;
  text-align: center;
  color: var(--color-text-dim);
}

.hc-empty { color: var(--color-text-dim); }
.hc-empty :deep(.hola-icon) { color: color-mix(in oklch, var(--color-primary) 70%, transparent); }
.hc-empty-title { margin: 0; font-size: 15px; font-weight: 600; color: var(--color-text); }
.hc-empty-sub { margin: 0; max-width: 380px; font-size: 13px; line-height: 1.45; }

.hc-chips { display: flex; flex-wrap: wrap; gap: 8px; justify-content: center; margin-top: 6px; }

.hc-chip {
  padding: 8px 14px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font-size: 12.5px;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s;
}

.hc-chip:hover { border-color: var(--color-primary); color: var(--color-primary); }

/* Дата-разделитель — пилюля по центру ленты, как в переписке. */
.hc-day { display: flex; justify-content: center; margin: 6px 0 2px; }

.hc-day span {
  padding: 4px 12px;
  border-radius: var(--radius-full);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  font-size: 11.5px;
  font-weight: 600;
  color: var(--color-text-dim);
}

.hc-row { display: flex; }
.hc-row.mine { justify-content: flex-end; }

.hc-msg { max-width: min(78%, 560px); display: flex; flex-direction: column; gap: 4px; }

.hc-bubble {
  position: relative;
  padding: 12px 14px 20px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  font-size: 14px;
  line-height: 1.5;
  color: var(--color-text);
  overflow-wrap: anywhere;
}

.hc-row.mine .hc-bubble {
  background: color-mix(in oklch, var(--color-primary) 10%, var(--glass-bg));
  border-color: color-mix(in oklch, var(--color-primary) 26%, var(--acrylic-border));
}

.hc-time {
  position: absolute;
  right: 12px;
  bottom: 6px;
  font-size: 10.5px;
  color: var(--color-text-dim);
}

.hc-sources { padding: 0 4px; font-size: 11.5px; color: var(--color-text-dim); }

.hc-fb { display: flex; align-items: center; gap: 6px; padding: 0 2px; flex-wrap: wrap; }

.hc-fb-btn {
  width: 28px;
  min-width: 28px;
  max-width: 28px;
  height: 28px;
  min-height: 28px;
  max-height: 28px;
  display: grid;
  place-items: center;
  padding: 0;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  transition: color 0.15s, background 0.15s;
}

.hc-fb-btn:hover { color: var(--color-primary); background: color-mix(in oklch, var(--color-primary) 10%, transparent); }
.hc-fb-btn.active { color: var(--color-primary); }
.hc-fb-btn .material-symbols-outlined { font-size: 16px; }

.hc-fb-chip {
  padding: 4px 10px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: var(--glass-bg);
  color: var(--color-text-dim);
  font-size: 11.5px;
  cursor: pointer;
}

.hc-fb-chip:hover { color: var(--color-primary); border-color: var(--color-primary); }

.hc-typing { display: flex; gap: 5px; padding: 14px; }

.hc-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-text-dim);
  animation: hcBlink 1.2s infinite ease-in-out;
}

.hc-dot:nth-child(2) { animation-delay: 0.15s; }
.hc-dot:nth-child(3) { animation-delay: 0.3s; }

@keyframes hcBlink {
  0%, 80%, 100% { opacity: 0.3; }
  40% { opacity: 1; }
}

.hc-error {
  margin: 0;
  padding: 10px 14px;
  border-radius: var(--radius-md);
  background: color-mix(in oklch, var(--color-error) 12%, transparent);
  color: var(--color-error);
  font-size: 13px;
}

/* Компактная лента: панель зажата клавиатурой — оставляем сами реплики и
   подсказки пустого чата, всё остальное урезано. */
.hc.compact { gap: 6px; padding: 2px 2px 4px; }
.hc.compact .hc-center { gap: 6px; padding: 8px 10px; }
.hc.compact .hc-empty-title { font-size: 13.5px; }
.hc.compact .hc-chips { gap: 6px; margin-top: 2px; }
.hc.compact .hc-chip { padding: 6px 11px; font-size: 12px; }
.hc.compact .hc-msg { max-width: min(88%, 560px); }
.hc.compact .hc-bubble { padding: 8px 11px 18px; font-size: 13.5px; }
.hc.compact .hc-day { margin: 2px 0 0; }
</style>
