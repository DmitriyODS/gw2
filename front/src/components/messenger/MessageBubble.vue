<template>
  <div class="msg-row" :class="{ outgoing: isMine }">
    <div class="msg-bubble">
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
})

function attachmentTag() { return AttachmentView }

function formatTime(iso) {
  if (!iso) return ''
  return new Date(iso).toLocaleTimeString('ru', { hour: '2-digit', minute: '2-digit' })
}
</script>

<style scoped>
.msg-row {
  display: flex;
  margin-bottom: 8px;
}

.msg-row.outgoing { justify-content: flex-end; }

.msg-bubble {
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
