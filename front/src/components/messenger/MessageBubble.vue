<template>
  <div class="msg-row" :class="{ outgoing: isMine }">
    <div class="msg-actions">
      <button v-if="showReply" class="msg-action" title="Ответить" @click="$emit('reply', message)">
        <span class="material-symbols-outlined">reply</span>
      </button>
      <button v-if="showForward" class="msg-action" title="Переслать" @click="$emit('forward', message)">
        <span class="material-symbols-outlined">forward</span>
      </button>
      <button
        v-if="showDelete"
        class="msg-action danger"
        :title="isMine ? 'Удалить' : 'Удалить у меня'"
        @click="$emit('delete', message)"
      >
        <span class="material-symbols-outlined">delete</span>
      </button>
    </div>
    <div class="msg-bubble">
      <div v-if="message.forwarded_from" class="msg-forwarded">
        <span class="material-symbols-outlined">forward</span>
        Переслано от {{ message.forwarded_from.fio }}
      </div>
      <div v-if="message.reply_to" class="msg-quote">
        <span class="msg-quote-author">{{ message.reply_to.sender_fio || 'Сообщение' }}</span>
        <span class="msg-quote-text">{{ quotePreview(message.reply_to) }}</span>
      </div>
      <div v-if="message.attachments?.length" class="msg-attachments">
        <component
          :is="attachmentTag(att)"
          v-for="att in message.attachments"
          :key="att.id"
          :att="att"
        />
      </div>
      <div v-if="message.text" class="msg-text">{{ message.text }}</div>
      <div class="msg-meta">
        <span class="msg-time">{{ formatTime(message.created_at) }}</span>
        <span v-if="isMine" class="msg-read">
          <span class="material-symbols-outlined" :class="{ seen: message.read_at }">
            {{ message.read_at ? 'done_all' : 'done' }}
          </span>
        </span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { h } from 'vue'
import AttachmentView from './AttachmentView.vue'

const props = defineProps({
  message: { type: Object, required: true },
  isMine: { type: Boolean, default: false },
  showReply: { type: Boolean, default: true },
  showForward: { type: Boolean, default: true },
  showDelete: { type: Boolean, default: true },
})

defineEmits(['delete', 'reply', 'forward'])

function attachmentTag() { return AttachmentView }

function formatTime(iso) {
  if (!iso) return ''
  return new Date(iso).toLocaleTimeString('ru', { hour: '2-digit', minute: '2-digit' })
}

function quotePreview(reply) {
  if (reply.text) return reply.text
  if (reply.has_attachments) return 'Вложение'
  return 'Сообщение'
}
</script>

<style scoped>
.msg-row {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 8px;
}

.msg-row.outgoing { justify-content: flex-end; }

/* Панель действий у bubble (ответить / переслать / удалить) — показывается
   на hover. Слева для своих, справа для входящих, чтобы не мешать чтению. */
.msg-actions {
  order: 1;
  display: flex;
  align-items: center;
  gap: 2px;
  opacity: 0;
  transition: opacity 0.15s;
  flex-shrink: 0;
}

.msg-action {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: none;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s, color 0.15s;
}

.msg-action:hover {
  background: var(--color-surface-high);
  color: var(--color-text);
}

.msg-action.danger:hover {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}

.msg-action .material-symbols-outlined { font-size: 16px; }

.msg-row:hover .msg-actions,
.msg-row:focus-within .msg-actions {
  opacity: 1;
}

.msg-row.outgoing .msg-actions {
  order: 0;
}

@media (hover: none) {
  .msg-actions { opacity: 0.55; }
}

/* Метка пересланного сообщения */
.msg-forwarded {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11.5px;
  font-style: italic;
  color: var(--color-text-dim);
  margin-bottom: 4px;
}

.msg-forwarded .material-symbols-outlined { font-size: 14px; }

/* Цитата (ответ на сообщение) */
.msg-quote {
  display: flex;
  flex-direction: column;
  gap: 1px;
  padding: 4px 8px;
  margin-bottom: 5px;
  border-left: 3px solid var(--color-primary);
  background: color-mix(in oklch, var(--color-primary) 10%, transparent);
  border-radius: var(--radius-sm);
}

.msg-row.outgoing .msg-quote {
  border-left-color: var(--color-on-primary-container);
  background: color-mix(in oklch, var(--color-on-primary-container) 10%, transparent);
}

.msg-quote-author {
  font-size: 11.5px;
  font-weight: 700;
  color: var(--color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.msg-row.outgoing .msg-quote-author {
  color: var(--color-on-primary-container);
}

.msg-quote-text {
  font-size: 12px;
  color: var(--color-text-dim);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 240px;
}

.msg-row.outgoing .msg-quote-text {
  color: var(--color-on-primary-container);
  opacity: 0.85;
}

.msg-bubble {
  order: 0;
  max-width: 70%;
  background: var(--color-surface-high);
  color: var(--color-text);
  padding: 8px 12px;
  border-radius: var(--radius-lg);
  border-top-left-radius: var(--radius-xs);
  box-shadow: var(--shadow-sm);
  word-wrap: break-word;
}

.msg-row.outgoing .msg-bubble {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  border-top-left-radius: var(--radius-lg);
  border-top-right-radius: var(--radius-xs);
}

.msg-text {
  white-space: pre-wrap;
  font-size: 14px;
  line-height: 1.4;
}

.msg-attachments {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 6px;
}

.msg-meta {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
  margin-top: 4px;
}

.msg-time {
  font-size: 11px;
  color: var(--color-text-dim);
  opacity: 0.8;
}

.msg-read .material-symbols-outlined {
  font-size: 16px;
  color: var(--color-text-dim);
}

.msg-read .material-symbols-outlined.seen {
  color: var(--color-success);
}

@media (max-width: 768px) {
  .msg-bubble { max-width: 85%; }
}
</style>
