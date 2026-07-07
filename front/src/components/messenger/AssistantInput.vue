<template>
  <div class="ai-input">
    <textarea
      ref="textarea"
      v-model="text"
      :placeholder="sending ? 'Ассистент печатает…' : 'Спросите про задачи или статистику…'"
      rows="1"
      class="ai-textarea"
      enterkeyhint="enter"
      :disabled="disabled"
      @keydown.enter.exact="onEnterKey"
      @input="autoresize"
    />
    <button
      class="ai-send-btn"
      :disabled="!canSend"
      @click="submit"
      :title="isTouchDevice ? 'Отправить' : 'Отправить (Enter)'"
    >
      <span class="material-symbols-outlined">send</span>
    </button>
  </div>
</template>

<script setup>
// Минимальный ввод для чата ассистента — обычный текст без вложений/ответов
// (в отличие от MessageInput.vue, заточенного под мессенджер). Повторяет
// автоувеличение textarea и правило Enter/Shift+Enter из MessageInput.
import { ref, computed, nextTick } from 'vue'

const props = defineProps({
  sending: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
})
const emit = defineEmits(['send'])

const text = ref('')
const textarea = ref(null)

const isTouchDevice = window.matchMedia?.('(hover: none) and (pointer: coarse)').matches ?? false

const canSend = computed(() => !props.sending && !props.disabled && Boolean(text.value.trim()))

function onEnterKey(e) {
  if (isTouchDevice) return
  e.preventDefault()
  submit()
}

function autoresize() {
  const el = textarea.value
  if (!el) return
  const max = 140
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, max) + 'px'
  el.style.overflowY = el.scrollHeight > max ? 'auto' : 'hidden'
}

function submit() {
  if (!canSend.value) return
  emit('send', text.value.trim())
  text.value = ''
  nextTick(() => autoresize())
}

defineExpose({ focus: () => textarea.value?.focus() })
</script>

<style scoped>
.ai-input {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  padding: 10px 14px 12px;
  border-top: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
}

.ai-textarea {
  flex: 1;
  resize: none;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  padding: 10px 12px;
  font: inherit;
  font-size: 14px;
  background: var(--color-surface-low);
  color: var(--color-text);
  outline: none;
  max-height: 140px;
  min-height: 40px;
  line-height: 1.4;
  overflow-y: hidden;
}

.ai-textarea:focus { border-color: var(--color-primary); }
.ai-textarea:disabled { opacity: 0.6; cursor: not-allowed; }

.ai-send-btn {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  border: none;
  background: var(--color-primary);
  color: var(--color-on-primary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: background 0.15s, opacity 0.15s;
}

.ai-send-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.ai-send-btn:not(:disabled):hover { background: var(--color-primary-hover); }
</style>
