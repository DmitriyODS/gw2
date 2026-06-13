<template>
  <div class="cpanel">
    <header class="cpanel-head">
      <span class="cpanel-title">Чат звонка</span>
      <button class="cpanel-close" title="Закрыть" @click="callStore.sidePanel = null">
        <span class="material-symbols-outlined">close</span>
      </button>
    </header>

    <div ref="listEl" class="cpanel-body">
      <div v-if="!callStore.chatMessages.length" class="chat-empty">
        <span class="material-symbols-outlined">forum</span>
        <p>Сообщения видят все участники звонка. Чат живёт, пока идёт звонок.</p>
      </div>
      <div
        v-for="m in callStore.chatMessages"
        :key="m.id"
        class="chat-msg"
        :class="{ own: m.own }"
      >
        <div class="chat-meta">
          <span class="chat-author">{{ m.own ? 'Вы' : m.name }}</span>
          <span class="chat-time">{{ fmtTime(m.ts) }}</span>
        </div>
        <div class="chat-bubble">{{ m.text }}</div>
      </div>
    </div>

    <footer class="cpanel-foot">
      <textarea
        ref="inputEl"
        v-model="draft"
        class="chat-input"
        rows="1"
        placeholder="Сообщение…"
        maxlength="2000"
        @input="autoGrow"
        @keydown.enter.exact="onEnter"
      />
      <button class="chat-send" :disabled="!draft.trim()" title="Отправить" @click="send">
        <span class="material-symbols-outlined">send</span>
      </button>
    </footer>
  </div>
</template>

<script setup>
import { ref, watch, nextTick, onMounted } from 'vue'
import { useCallStore } from '@/stores/call.js'

const callStore = useCallStore()
const draft = ref('')
const listEl = ref(null)
const inputEl = ref(null)

const MAX_INPUT_PX = 120

// На тач-устройствах Enter — перенос строки (отправка только кнопкой):
// случайные отправки с экранной клавиатуры раздражают сильнее лишнего тапа.
// На десктопе Enter отправляет, Shift+Enter — новая строка (см. MessageInput).
const isTouchDevice = window.matchMedia?.('(hover: none) and (pointer: coarse)').matches ?? false

function autoGrow() {
  const el = inputEl.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = `${Math.min(el.scrollHeight, MAX_INPUT_PX)}px`
}

function onEnter(e) {
  if (isTouchDevice) return // default-поведение textarea — перенос строки
  e.preventDefault()
  send()
}

function send() {
  const text = draft.value.trim()
  if (!text) return
  callStore.sendChat(text)
  draft.value = ''
  nextTick(autoGrow)
}

function fmtTime(ts) {
  return new Date(ts).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}

function scrollDown() {
  nextTick(() => {
    if (listEl.value) listEl.value.scrollTop = listEl.value.scrollHeight
  })
}

watch(() => callStore.chatMessages.length, scrollDown)
onMounted(scrollDown)
</script>

<style scoped>
.cpanel {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}

.cpanel-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--color-outline-dim);
  flex-shrink: 0;
}

.cpanel-title { font-weight: 700; font-size: 15px; }

.cpanel-close {
  margin-left: auto;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: 0;
  background: transparent;
  color: var(--color-text);
  display: grid;
  place-items: center;
  cursor: pointer;
}

.cpanel-close:hover { background: var(--color-surface-high); }
.cpanel-close .material-symbols-outlined { font-size: 18px; }

.cpanel-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 12px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.chat-empty {
  margin: auto;
  text-align: center;
  color: var(--color-text-dim);
  font-size: 13px;
  padding: 0 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.chat-empty .material-symbols-outlined {
  font-size: 36px;
  color: var(--color-primary);
  opacity: 0.6;
}

.chat-msg { display: flex; flex-direction: column; gap: 2px; max-width: 92%; }
.chat-msg.own { align-self: flex-end; align-items: flex-end; }

.chat-meta {
  display: flex;
  gap: 8px;
  font-size: 11px;
  color: var(--color-text-dim);
  padding: 0 4px;
}

.chat-author { font-weight: 600; }

.chat-bubble {
  padding: 8px 12px;
  border-radius: 14px;
  background: var(--color-surface-high);
  color: var(--color-text);
  font-size: 14px;
  line-height: 1.35;
  white-space: pre-wrap;
  word-break: break-word;
}

.chat-msg.own .chat-bubble {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.cpanel-foot {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  padding: 12px 16px calc(12px + env(safe-area-inset-bottom, 0px));
  border-top: 1px solid var(--color-outline-dim);
  flex-shrink: 0;
}

.chat-input {
  flex: 1;
  min-width: 0;
  padding: 9px 14px;
  border: 1px solid var(--color-outline-dim);
  border-radius: 18px;
  background: var(--color-surface-high);
  color: var(--color-text);
  font: inherit;
  font-size: 14px;
  line-height: 1.35;
  outline: none;
  resize: none;
  max-height: 120px;
  overflow-y: auto;
}

.chat-input:focus { border-color: var(--color-primary); }

.chat-send {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  border: 0;
  background: var(--color-primary);
  color: var(--color-on-primary);
  display: grid;
  place-items: center;
  cursor: pointer;
  flex-shrink: 0;
}

.chat-send:disabled { opacity: 0.5; cursor: default; }
.chat-send .material-symbols-outlined { font-size: 18px; }
</style>
