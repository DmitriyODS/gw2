<template>
  <!-- Системное сообщение о звонке: плашка по центру с иконкой, статусом и длительностью. -->
  <div v-if="message.kind === 'call'" class="call-row">
    <div class="call-pill" :class="callClass">
      <div class="call-icon">
        <span class="material-symbols-outlined">{{ callIcon }}</span>
      </div>
      <div class="call-body">
        <div class="call-title">{{ callTitle }}</div>
        <div class="call-sub">
          <span>{{ formatTime(message.created_at) }}</span>
          <template v-if="callDurationText">
            <span class="dot">·</span>
            <span>{{ callDurationText }}</span>
          </template>
        </div>
      </div>
      <button
        v-if="canJoin"
        class="call-join"
        :title="isMine ? 'Звонок ещё идёт' : 'Присоединиться'"
        @click="$emit('join-call', message.call)"
      >
        <span class="material-symbols-outlined">{{ joinIcon }}</span>
        <span class="call-join-label">{{ joinLabel }}</span>
      </button>
    </div>
  </div>

  <div v-else class="msg-row" :class="{ outgoing: isMine }">
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
      <div v-if="message.text" class="msg-text"><template v-for="(part, i) in textParts" :key="i"><a
          v-if="part.type === 'link'"
          :href="part.href"
          class="msg-link"
          target="_blank"
          rel="noopener noreferrer"
          @click.stop
        >{{ part.value }}</a><template v-else>{{ part.value }}</template></template></div>
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
import { computed } from 'vue'
import AttachmentView from './AttachmentView.vue'
import { linkifyParts } from '@/utils/linkify.js'

const props = defineProps({
  message: { type: Object, required: true },
  isMine: { type: Boolean, default: false },
  showReply: { type: Boolean, default: true },
  showForward: { type: Boolean, default: true },
  showDelete: { type: Boolean, default: true },
})

defineEmits(['delete', 'reply', 'forward', 'join-call'])

function attachmentTag() { return AttachmentView }

/* Текст сообщения с распознанными ссылками: обычный текст + кликабельные <a>. */
const textParts = computed(() => linkifyParts(props.message.text))

function formatTime(iso) {
  if (!iso) return ''
  return new Date(iso).toLocaleTimeString('ru', { hour: '2-digit', minute: '2-digit' })
}

function quotePreview(reply) {
  if (reply.text) return reply.text
  if (reply.has_attachments) return 'Вложение'
  return 'Сообщение'
}

/* Системная плашка звонка. status:
   - 'ringing' — идёт звон, никто не принял
   - 'active'  — разговор идёт прямо сейчас
   - 'ended'   — нормально завершён (есть длительность)
   - 'missed'  — никто не ответил (длительности нет)
*/
const callInfo = computed(() => props.message.call || {})
const callStatus = computed(() => callInfo.value.status)
const isVideo = computed(() => callInfo.value.media === 'video')
const isLive = computed(() => callStatus.value === 'ringing' || callStatus.value === 'active')
const isMissed = computed(() => callStatus.value === 'missed'
  || (callStatus.value === 'ended' && !callInfo.value.duration_sec))
const canJoin = computed(() => isLive.value && !props.isMine)

const callIcon = computed(() => {
  if (isMissed.value) return isVideo.value ? 'videocam_off' : 'phone_missed'
  if (isLive.value)  return isVideo.value ? 'videocam' : 'call'
  return isVideo.value ? 'videocam' : 'call'
})

const callClass = computed(() => ({
  live: isLive.value,
  missed: isMissed.value,
  ended: callStatus.value === 'ended' && !isMissed.value,
  mine: props.isMine,
}))

const callTitle = computed(() => {
  const v = isVideo.value ? 'Видеозвонок' : 'Аудиозвонок'
  if (isMissed.value) {
    return props.isMine ? `${v} · Без ответа` : `Пропущенный ${v.toLowerCase()}`
  }
  if (isLive.value) {
    return `${v} · Идёт ${callStatus.value === 'ringing' ? 'вызов' : 'сейчас'}`
  }
  return v
})

function fmtDuration(sec) {
  const s = Math.max(0, sec | 0)
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const ss = (s % 60).toString().padStart(2, '0')
  if (h > 0) return `${h}:${m.toString().padStart(2, '0')}:${ss}`
  return `${m}:${ss}`
}

const callDurationText = computed(() => {
  if (isLive.value || isMissed.value) return null
  return callInfo.value.duration_sec ? fmtDuration(callInfo.value.duration_sec) : null
})

const joinIcon = computed(() => isVideo.value ? 'videocam' : 'call')
const joinLabel = computed(() => 'Присоединиться')
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

/* Кликабельные ссылки внутри текста. Цвет — через токены, чтобы корректно
   читаться на светлой/тёмной/любой кастомной теме. На исходящем пузыре фон
   уже акцентный (primary-container), поэтому ссылка наследует контрастный
   on-цвет, оставаясь подчёркнутой. */
.msg-link {
  color: var(--color-primary);
  text-decoration: underline;
  text-underline-offset: 2px;
  word-break: break-word;
  overflow-wrap: anywhere;
  cursor: pointer;
}

.msg-link:hover {
  text-decoration-thickness: 2px;
}

.msg-row.outgoing .msg-link {
  color: var(--color-on-primary-container);
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

/* ─── Системная плашка звонка ─────────────────────────────────────
   Центрируется на всю ширину как «системное» сообщение, чтобы её
   нельзя было перепутать с обычной репликой. Все цвета — только
   через семантические токены, чтобы корректно работать со светлой/
   тёмной/любой кастомной темой. */
.call-row {
  display: flex;
  justify-content: center;
  margin: 6px 0;
}

.call-pill {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
  background: var(--color-surface-low);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg);
  max-width: min(440px, 92%);
  width: 100%;
}

.call-pill.live {
  background: var(--color-primary-container);
  border-color: var(--color-primary);
  color: var(--color-on-primary-container);
}

.call-pill.missed {
  background: var(--color-error-container);
  border-color: color-mix(in oklch, var(--color-error) 50%, transparent);
  color: var(--color-on-error-container);
}

.call-icon {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--color-surface);
  color: var(--color-primary);
  flex-shrink: 0;
}

.call-pill.live .call-icon {
  background: var(--color-primary);
  color: var(--color-on-primary);
  animation: callPulse 1.6s ease-in-out infinite;
}

.call-pill.missed .call-icon {
  background: var(--color-error);
  color: var(--color-on-error);
}

.call-icon .material-symbols-outlined { font-size: 20px; }

@keyframes callPulse {
  0%, 100% { box-shadow: 0 0 0 0 color-mix(in oklch, var(--color-primary) 50%, transparent); }
  50%      { box-shadow: 0 0 0 8px color-mix(in oklch, var(--color-primary) 0%, transparent); }
}

.call-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.call-title {
  font-size: 13.5px;
  font-weight: 600;
  color: inherit;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.call-sub {
  font-size: 12px;
  color: color-mix(in oklch, currentColor 70%, transparent);
  display: flex;
  align-items: center;
  gap: 6px;
}

.call-sub .dot { opacity: 0.6; }

.call-join {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: var(--radius-full);
  background: var(--color-primary);
  color: var(--color-on-primary);
  border: none;
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  flex-shrink: 0;
  transition: background 0.15s, transform 0.12s;
}

.call-join:hover { transform: translateY(-1px); }
.call-join:active { transform: translateY(0); }

.call-join .material-symbols-outlined { font-size: 18px; }

@media (max-width: 480px) {
  .call-join-label { display: none; }
  .call-join { padding: 8px 10px; }
}
</style>
