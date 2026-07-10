<template>
  <div
    class="msg-row"
    :class="{ outgoing: isMine, swiping: swipeDx > 0 }"
    :data-msg-id="message.id"
    :style="rowStyle"
    @contextmenu.prevent="onContextMenu"
    @pointerdown="onPointerDown"
    @pointermove="onPointerMove"
    @pointerup="onPointerUp"
    @pointercancel="onPointerUp"
  >
    <span class="swipe-reply-hint" aria-hidden="true">
      <span class="material-symbols-outlined">reply</span>
    </span>
    <div class="msg-bubble" :class="bubbleClass">
      <div v-if="isPinned" class="msg-pinned-mark" title="Закреплено">
        <span class="material-symbols-outlined">keep</span>
        Закреплено
      </div>
      <div v-if="isDevReply" class="msg-dev-badge" title="Сообщение от разработчиков">
        <span class="material-symbols-outlined">support_agent</span>
        Разработчики
      </div>
      <div v-else-if="message.is_bot" class="msg-pet-badge" title="Сообщение Грувика">
        <span class="msg-pet-emoji" aria-hidden="true">👾</span>
        Грувик
      </div>
      <div v-if="message.forwarded_from" class="msg-forwarded">
        <span class="material-symbols-outlined">forward</span>
        Переслано от {{ message.forwarded_from.fio }}
      </div>
      <div
        v-if="message.reply_to"
        class="msg-quote"
        role="button"
        tabindex="0"
        title="Перейти к сообщению"
        @click.stop="$emit('quote-click', message.reply_to.id)"
        @keydown.enter.stop="$emit('quote-click', message.reply_to.id)"
      >
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
      <button
        v-if="message.kind === 'task' && message.task"
        class="task-pill"
        :class="taskPillClass"
        :style="taskPillStyle"
        @click="$emit('open-task', message.task.id)"
      >
        <span class="material-symbols-outlined task-pill-icon">task</span>
        <div class="task-pill-body">
          <div class="task-pill-name">{{ message.task.name }}</div>
          <div v-if="message.task.responsible_fio" class="task-pill-sub">
            <span class="material-symbols-outlined">person</span>
            {{ message.task.responsible_fio }}
          </div>
        </div>
      </button>
      <div
        v-else-if="message.kind === 'task'"
        class="task-pill missing"
      >
        <span class="material-symbols-outlined">task_alt</span>
        Задача удалена
      </div>
      <button
        v-if="message.kind === 'post' && message.post"
        class="post-pill"
        @click="$emit('open-post', message.post.id)"
      >
        <img v-if="message.post.cover_url" class="post-pill-cover" :src="message.post.cover_url" alt="" />
        <span v-else class="material-symbols-outlined post-pill-icon">campaign</span>
        <div class="post-pill-body">
          <div class="post-pill-title">{{ message.post.title }}</div>
          <div v-if="message.post.excerpt" class="post-pill-sub">{{ message.post.excerpt }}</div>
        </div>
      </button>
      <div
        v-else-if="message.kind === 'post'"
        class="post-pill missing"
      >
        <span class="material-symbols-outlined">campaign</span>
        Пост удалён
      </div>
      <!-- Звонок — обычное сообщение с карточкой звонка внутри пузыря:
           переслать/удалить/ответить/закрепить работает как у текста. -->
      <div
        v-if="message.kind === 'call'"
        class="call-msg"
        :class="[callClass, { clickable: isLive }]"
        :role="isLive ? 'button' : null"
        :tabindex="isLive ? 0 : null"
        @click="isLive && $emit('join-call', message.call)"
        @keydown.enter="isLive && $emit('join-call', message.call)"
      >
        <div class="call-icon">
          <span class="material-symbols-outlined">{{ callIcon }}</span>
        </div>
        <div class="call-body">
          <div class="call-title">{{ callTitle }}</div>
          <div v-if="callDurationText" class="call-sub">{{ callDurationText }}</div>
        </div>
        <button
          v-if="isLive"
          class="call-join"
          :title="joinLabel"
          @click.stop="$emit('join-call', message.call)"
        >
          <span class="material-symbols-outlined">{{ joinIcon }}</span>
          <span class="call-join-label">{{ joinLabel }}</span>
        </button>
      </div>
      <MarkdownView v-if="message.text" class="msg-text" :source="message.text" />
      <div v-if="reactionGroups.length" class="msg-reactions">
        <button
          v-for="g in reactionGroups"
          :key="g.emoji"
          class="msg-reaction"
          :class="{ mine: g.mine }"
          @pointerdown.stop
          @pointerup.stop
          @click.stop="$emit('react', g.emoji)"
        >
          <span class="msg-reaction-emoji">{{ g.emoji }}</span>
          <span v-if="g.count > 1" class="msg-reaction-count">{{ g.count }}</span>
        </button>
      </div>
      <div class="msg-meta">
        <span v-if="senderName" class="msg-sender">{{ senderName }}</span>
        <span v-if="message.edited_at" class="msg-edited" title="Сообщение отредактировано">изменено</span>
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
import { computed, ref } from 'vue'
import AttachmentView from './AttachmentView.vue'
import MarkdownView from '@/components/common/MarkdownView.vue'

const props = defineProps({
  message: { type: Object, required: true },
  isMine: { type: Boolean, default: false },
  senderName: { type: String, default: '' },
  showReply: { type: Boolean, default: true },
  showForward: { type: Boolean, default: true },
  showDelete: { type: Boolean, default: true },
  showPin: { type: Boolean, default: true },
  meId: { type: [Number, String], default: null },
})

const emit = defineEmits(['delete', 'edit', 'reply', 'forward', 'join-call', 'pin', 'open-task', 'open-post', 'context-menu', 'quote-click', 'react'])

// Реакции сгруппированные по эмодзи; mine — подсветка своей.
const reactionGroups = computed(() => {
  const map = new Map()
  for (const r of props.message.reactions || []) {
    let g = map.get(r.emoji)
    if (!g) {
      g = { emoji: r.emoji, count: 0, mine: false }
      map.set(r.emoji, g)
    }
    g.count++
    if (props.meId != null && Number(r.user_id) === Number(props.meId)) g.mine = true
  }
  return [...map.values()]
})

const isPinned = computed(() => !!props.message.pinned_at)
// Сообщение от техподдержки: новое серверное поле is_from_support либо старый
// kind='system_dev_reply' (на случай миграции старых сообщений).
const isDevReply = computed(() =>
  !!props.message.is_from_support || props.message.kind === 'system_dev_reply'
)

const bubbleClass = computed(() => ({
  pinned: isPinned.value,
  'dev-reply': isDevReply.value,
  'pet-reply': !!props.message.is_bot,
}))

function attachmentTag() { return AttachmentView }

function formatTime(iso) {
  if (!iso) return ''
  return new Date(iso).toLocaleTimeString('ru', { hour: '2-digit', minute: '2-digit' })
}

function quotePreview(reply) {
  if (reply.kind === 'call') return '📞 Звонок'
  if (reply.text) return reply.text
  if (reply.has_attachments) return 'Вложение'
  return 'Сообщение'
}

/* ── Прикреплённая задача — pill с цветом из палитры тегов. ── */
const taskPillClass = computed(() => {
  const color = props.message.task?.color
  return color ? `tag-${color}` : ''
})
const taskPillStyle = computed(() => {
  const color = props.message.task?.color
  return color ? {
    background: `var(--tag-${color}-surface)`,
    borderColor: `var(--tag-${color}-border)`,
    color: 'var(--color-text)',
  } : {}
})

/* ── Контекстное меню (правый клик / long-press) ── */
function onContextMenu(e) {
  emit('context-menu', { x: e.clientX, y: e.clientY, message: props.message })
}

const swipeDx = ref(0)
let swipeStartX = 0
let swipeStartY = 0
let swipeActive = false
let pointerActiveId = null
let pointerType = ''
let downAt = 0

const rowStyle = computed(() =>
  swipeDx.value > 0 ? { transform: `translateX(${swipeDx.value}px)` } : {})

function onPointerDown(e) {
  if (e.button === 2) return  // ПКМ обрабатывается contextmenu
  pointerActiveId = e.pointerId
  pointerType = e.pointerType
  downAt = Date.now()
  swipeStartX = e.clientX
  swipeStartY = e.clientY
  swipeActive = false
  swipeDx.value = 0
}

function onPointerMove(e) {
  if (pointerActiveId !== e.pointerId) return
  const dx = e.clientX - swipeStartX
  const dy = e.clientY - swipeStartY

  if (!swipeActive && Math.abs(dy) > 8 && Math.abs(dy) > Math.abs(dx)) {
    // Вертикальный скролл — это не свайп.
    return
  }
  if (Math.abs(dx) > 6) {
    swipeActive = true
  }

  if (swipeActive && dx > 0) {
    // Свайп ВПРАВО → ответ. Лимит 96px (потом плавное затухание).
    const limit = 96
    swipeDx.value = Math.min(dx, limit)
  }
}

function onPointerUp(e) {
  if (pointerActiveId !== null && pointerActiveId !== e.pointerId) return
  pointerActiveId = null
  if (swipeActive) {
    const dx = swipeDx.value
    swipeDx.value = 0
    if (dx > 60) {
      emit('reply', props.message)
    }
    return
  }
  // Тап по сообщению на тач-устройстве открывает меню действий (вместо
  // long-press: удержание остаётся браузеру под выделение текста).
  if (pointerType === 'touch') maybeTapMenu(e)
}

function maybeTapMenu(e) {
  if (Date.now() - downAt > 400) return
  if (Math.abs(e.clientX - swipeStartX) > 10 || Math.abs(e.clientY - swipeStartY) > 10) return
  const sel = window.getSelection?.()
  if (sel && !sel.isCollapsed) return
  // Тапы по интерактивным элементам пузыря — не повод открывать меню.
  if (e.target.closest('a, button, .msg-quote, .msg-attachments, .call-msg.clickable')) return
  if (navigator.vibrate) {
    try { navigator.vibrate(10) } catch {/* iOS Safari */}
  }
  emit('context-menu', { x: e.clientX, y: e.clientY, message: props.message })
}

/* ── Системная плашка звонка. ── */
const callInfo = computed(() => props.message.call || {})
const callStatus = computed(() => callInfo.value.status)
const isVideo = computed(() => callInfo.value.media === 'video')
const isLive = computed(() => callStatus.value === 'ringing' || callStatus.value === 'active')
const isMissed = computed(() => callStatus.value === 'missed'
  || (callStatus.value === 'ended' && !callInfo.value.duration_sec))

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
const joinLabel = computed(() => props.isMine ? 'Вернуться' : 'Присоединиться')
</script>

<style scoped>
.msg-row {
  position: relative;
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 8px;
  transition: transform 0.18s ease;
  touch-action: pan-y;
}

.msg-row.outgoing { justify-content: flex-end; }

.msg-row.swiping {
  transition: none;
}

/* Иконка-подсказка свайпа ответом — проявляется при свайпе вправо. */
.swipe-reply-hint {
  position: absolute;
  left: -42px;
  top: 50%;
  transform: translateY(-50%);
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  opacity: 0;
  transition: opacity 0.15s;
  pointer-events: none;
}

.msg-row.swiping .swipe-reply-hint {
  opacity: 1;
}

.swipe-reply-hint .material-symbols-outlined { font-size: 18px; }

/* Метка «Закреплено» внутри пузыря */
.msg-pinned-mark {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 600;
  color: var(--color-tertiary);
  margin-bottom: 4px;
}

.msg-row.outgoing .msg-pinned-mark { color: var(--color-on-primary-container); }

.msg-pinned-mark .material-symbols-outlined {
  font-size: 14px;
  font-variation-settings: 'FILL' 1, 'wght' 500, 'GRAD' 0, 'opsz' 20;
}

.msg-bubble.pinned {
  box-shadow: var(--shadow-sm), inset 2px 0 0 0 var(--color-tertiary);
}

/* Сообщение «Разработчиков» в dev-чате: акцентный значок + фон. */
.msg-dev-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11.5px;
  font-weight: 700;
  color: var(--color-tertiary);
  margin-bottom: 4px;
  letter-spacing: 0.02em;
}
.msg-dev-badge .material-symbols-outlined {
  font-size: 14px;
  font-variation-settings: 'FILL' 1;
}
.msg-bubble.pet-reply {
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}
.msg-pet-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11.5px;
  font-weight: 700;
  margin-bottom: 4px;
  opacity: 0.85;
}
.msg-pet-emoji { font-size: 13px; }

.msg-bubble.dev-reply {
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
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

/* Цитата (ответ на сообщение) — кликабельна: переход к оригиналу. */
.msg-quote {
  display: flex;
  flex-direction: column;
  gap: 1px;
  padding: 4px 8px;
  margin-bottom: 5px;
  border-left: 3px solid var(--color-primary);
  background: color-mix(in oklch, var(--color-primary) 10%, transparent);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background 0.15s;
}

.msg-quote:hover {
  background: color-mix(in oklch, var(--color-primary) 18%, transparent);
}

.msg-row.outgoing .msg-quote {
  border-left-color: var(--color-on-primary-container);
  background: color-mix(in oklch, var(--color-on-primary-container) 10%, transparent);
}

.msg-row.outgoing .msg-quote:hover {
  background: color-mix(in oklch, var(--color-on-primary-container) 18%, transparent);
}

/* Подсветка сообщения после перехода к нему (класс вешает useJumpToMessage). */
.msg-row.msg-flash {
  border-radius: var(--radius-md);
  animation: msgRowFlash 1.5s ease-out;
}

@keyframes msgRowFlash {
  0%, 35% { background-color: color-mix(in oklch, var(--color-primary) 14%, transparent); }
  100%    { background-color: transparent; }
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
  font-size: 14px;
  line-height: 1.4;
}

/* Прикреплённая задача — плашка с цветом из палитры. */
.task-pill {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--acrylic-card-bg);
  color: var(--color-text);
  cursor: pointer;
  width: 100%;
  text-align: left;
  margin-bottom: 6px;
  font: inherit;
  transition: transform 0.12s, box-shadow 0.15s;
}

.task-pill:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

.task-pill.missing {
  cursor: default;
  color: var(--color-text-dim);
  font-style: italic;
}

.task-pill-icon {
  font-size: 22px;
  color: var(--color-primary);
  flex-shrink: 0;
  font-variation-settings: 'FILL' 1;
}

.task-pill-body { min-width: 0; flex: 1; }

.task-pill-name {
  font-size: 13.5px;
  font-weight: 700;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.task-pill-sub {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--color-text-dim);
}
.task-pill-sub .material-symbols-outlined { font-size: 14px; }

/* Пересланный пост портала — плашка-превью (снапшот на момент пересылки). */
.post-pill {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--acrylic-card-bg);
  color: var(--color-text);
  cursor: pointer;
  width: 100%;
  text-align: left;
  margin-bottom: 6px;
  font: inherit;
  transition: transform 0.12s, box-shadow 0.15s;
}

.post-pill:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-sm);
}

.post-pill.missing {
  cursor: default;
  color: var(--color-text-dim);
  font-style: italic;
}

.post-pill-cover {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-sm);
  object-fit: cover;
  flex-shrink: 0;
}

.post-pill-icon {
  font-size: 22px;
  color: var(--color-tertiary);
  flex-shrink: 0;
  font-variation-settings: 'FILL' 1;
}

.post-pill-body { min-width: 0; flex: 1; }

.post-pill-title {
  font-size: 13.5px;
  font-weight: 700;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.post-pill-sub {
  font-size: 12px;
  color: var(--color-text-dim);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.msg-attachments {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 6px;
}

/* Чипы реакций внизу пузыря. */
.msg-reactions {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  margin-top: 6px;
}

.msg-reaction {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  min-height: 24px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: var(--acrylic-card-bg);
  font: inherit;
  font-size: 13px;
  line-height: 1;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, transform 0.12s;
}

.msg-reaction:hover { transform: scale(1.08); }

.msg-reaction.mine {
  background: var(--color-primary-container);
  border-color: var(--color-primary);
}

.msg-reaction-count {
  font-size: 11.5px;
  font-weight: 700;
  color: var(--color-text-dim);
}

.msg-reaction.mine .msg-reaction-count { color: var(--color-on-primary-container); }

.msg-meta {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
  margin-top: 4px;
}

/* В dev-чате под сообщением — имя автора (для контекста, у кого спросить). */
.msg-sender {
  font-size: 11px;
  font-weight: 600;
  color: var(--color-primary);
  margin-right: auto;
}

.msg-row.outgoing .msg-sender { color: var(--color-on-primary-container); }

.msg-time {
  font-size: 11px;
  color: var(--color-text-dim);
  opacity: 0.8;
}

.msg-edited {
  font-size: 11px;
  color: var(--color-text-dim);
  opacity: 0.7;
  font-style: italic;
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

/* ─── Карточка звонка внутри обычного пузыря ───────────────────── */
.call-msg {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 2px 0;
  min-width: 190px;
}

.call-msg.clickable {
  cursor: pointer;
  border-radius: var(--radius-sm);
}

.call-msg.clickable:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}

.call-icon {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--acrylic-card-bg);
  color: var(--color-primary);
  flex-shrink: 0;
}

.call-msg.live .call-icon {
  background: var(--color-primary);
  color: var(--color-on-primary);
  animation: callPulse 1.6s ease-in-out infinite;
}

.call-msg.missed .call-icon {
  background: var(--color-error);
  color: var(--color-on-error);
}

.call-msg.missed .call-title { color: var(--color-error); }

.msg-row.outgoing .call-msg.missed .call-title {
  color: var(--color-on-primary-container);
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
}

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
