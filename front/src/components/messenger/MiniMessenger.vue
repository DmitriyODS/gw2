<template>
  <div v-if="!hidden" class="mini-mess">
    <!-- Панель -->
    <transition name="mini-pop">
      <div v-if="open" class="mini-panel">
        <!-- Режим: список диалогов -->
        <template v-if="!threadId">
          <header class="mini-head">
            <span class="mini-title">Сообщения</span>
            <button class="mini-icon" title="Свернуть" @click="open = false">
              <span class="material-symbols-outlined">close</span>
            </button>
          </header>
          <div v-if="!messenger.conversations.length" class="mini-empty">
            <span class="material-symbols-outlined">forum</span>
            <p>Пока нет диалогов</p>
          </div>
          <ul v-else class="mini-list">
            <li
              v-for="c in messenger.conversations"
              :key="c.id"
              class="mini-conv"
              :class="{ unread: c.unread_count > 0 }"
              @click="openThread(c.id)"
            >
              <div class="mini-avatar-wrap">
                <img class="mini-avatar" :src="avatarOf(c.other_user)" :alt="c.other_user?.fio" />
                <span v-if="messenger.isOnline(c.other_user?.id)" class="online-dot mini-list-dot" title="В сети"></span>
              </div>
              <div class="mini-conv-body">
                <div class="mini-conv-name">{{ c.other_user?.fio }}</div>
                <div class="mini-conv-preview">{{ preview(c.last_message) }}</div>
              </div>
              <span v-if="c.unread_count" class="mini-badge">{{ c.unread_count }}</span>
            </li>
          </ul>
        </template>

        <!-- Режим: переписка -->
        <template v-else>
          <header class="mini-head">
            <button class="mini-icon" title="Назад" @click="closeThread">
              <span class="material-symbols-outlined">arrow_back</span>
            </button>
            <div class="mini-head-avatar-wrap">
              <img class="mini-head-avatar" :src="avatarOf(threadConv?.other_user)" :alt="threadConv?.other_user?.fio" />
              <span v-if="threadOnline" class="online-dot mini-head-dot" title="В сети"></span>
            </div>
            <div class="mini-head-title">
              <span class="mini-title--name">{{ threadConv?.other_user?.fio }}</span>
              <span class="mini-head-status" :class="{ online: threadOnline }">
                {{ threadOnline ? 'в сети' : threadLastSeen }}
              </span>
            </div>
            <button class="mini-icon" title="Свернуть" @click="open = false">
              <span class="material-symbols-outlined">close</span>
            </button>
          </header>
          <div
            ref="threadEl"
            class="mini-thread"
            @dragenter.prevent="onDragEnter"
            @dragover.prevent="onDragOver"
            @dragleave.prevent="onDragLeave"
            @drop.prevent="onDrop"
          >
            <div v-if="dragOver" class="mini-drop-overlay">
              <span class="material-symbols-outlined">upload_file</span>
              <span>Отпустите файл</span>
            </div>
            <div v-if="messenger.loadingMessages && !messenger.activeMessages.length" class="mini-loading">
              <ProgressSpinner style="width:28px;height:28px" />
            </div>
            <MessageBubble
              v-for="m in messenger.activeMessages"
              :key="m.id"
              :message="m"
              :is-mine="m.sender_id === authStore.user?.id"
              :show-forward="false"
              :show-delete="false"
              @reply="startReply"
            />
          </div>
          <MessageInput
            ref="miniInputRef"
            :sending="messenger.sending"
            :reply-to="replyTo"
            placeholder="Сообщение…"
            @send="onSend"
            @cancel-reply="replyTo = null"
          />
        </template>
      </div>
    </transition>

    <!-- Кнопка-FAB -->
    <button class="mini-fab" :class="{ active: open }" @click="toggle" :title="open ? 'Свернуть чат' : 'Открыть чаты'">
      <span class="material-symbols-outlined">{{ open ? 'close' : 'chat' }}</span>
      <span v-if="!open && messenger.totalUnread" class="mini-fab-badge">
        {{ messenger.totalUnread > 99 ? '99+' : messenger.totalUnread }}
      </span>
    </button>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { useRoute } from 'vue-router'
import { useMessengerStore } from '@/stores/messenger.js'
import { useAuthStore } from '@/stores/auth.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { formatLastSeen } from '@/utils/presence.js'
import MessageBubble from './MessageBubble.vue'
import MessageInput from './MessageInput.vue'
import ProgressSpinner from 'primevue/progressspinner'

const route = useRoute()
const messenger = useMessengerStore()
const authStore = useAuthStore()
const { isMobile } = useBreakpoint()

const open = ref(false)
const threadId = ref(null)
const replyTo = ref(null)
const threadEl = ref(null)
const miniInputRef = ref(null)
const dragOver = ref(false)
let dragDepth = 0

function dragHasFiles(e) {
  const types = e.dataTransfer?.types
  return types && Array.from(types).includes('Files')
}

function onDragEnter(e) {
  if (!dragHasFiles(e)) return
  dragDepth++
  dragOver.value = true
}

function onDragOver(e) {
  if (dragHasFiles(e)) e.dataTransfer.dropEffect = 'copy'
}

function onDragLeave() {
  dragDepth = Math.max(0, dragDepth - 1)
  if (dragDepth === 0) dragOver.value = false
}

async function onDrop(e) {
  dragDepth = 0
  dragOver.value = false
  const files = Array.from(e.dataTransfer?.files || [])
  if (files.length) miniInputRef.value?.addFiles(files)
}

// На полном экране мессенджера FAB не нужен — там есть всё то же самое.
// На мобильном тоже скрываем: есть вкладка «Чаты» в нижней навигации, а FAB
// налезал бы на кнопку создания задачи и прочие действия.
const hidden = computed(() => isMobile.value || route.path.startsWith('/messenger'))

const threadConv = computed(() =>
  messenger.conversations.find(c => c.id === threadId.value) || null
)

const threadOnline = computed(() => messenger.isOnline(threadConv.value?.other_user?.id))
const threadLastSeen = computed(() => {
  const u = threadConv.value?.other_user
  if (!u) return ''
  return formatLastSeen(messenger.lastSeenOf(u.id, u.last_seen_at))
})

function toggle() {
  open.value = !open.value
  if (open.value) {
    if (!messenger.conversations.length) messenger.fetchConversations()
    // Свежий снимок онлайн-статусов при открытии.
    messenger.fetchPresence()
  }
}

async function openThread(id) {
  threadId.value = id
  await messenger.setActive(id)
  await nextTick()
  scrollBottom()
}

function closeThread() {
  threadId.value = null
  replyTo.value = null
  // Снимаем «активность», чтобы входящие в этот чат снова считались
  // непрочитанными, пока мы на него не смотрим.
  messenger.activeConversationId = null
}

function startReply(message) {
  replyTo.value = {
    id: message.id,
    sender_fio: message.sender_id === authStore.user?.id
      ? 'Вы'
      : (threadConv.value?.other_user?.fio || ''),
    text: message.text,
    has_attachments: !!message.attachments?.length,
  }
}

async function onSend(payload) {
  await messenger.send(threadId.value, payload)
  replyTo.value = null
  await nextTick()
  scrollBottom()
}

function scrollBottom() {
  const el = threadEl.value
  if (el) el.scrollTop = el.scrollHeight
}

function avatarOf(u) {
  if (!u) return ''
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

function preview(msg) {
  if (!msg) return 'Нет сообщений'
  if (msg.text) return msg.text
  if (msg.attachments?.length) return 'Вложение'
  return ''
}

// Автоскролл при новом сообщении в открытом треде.
const lastId = computed(() => {
  const arr = messenger.activeMessages
  return arr.length ? arr[arr.length - 1].id : 0
})

watch(lastId, async (id, prev) => {
  if (!threadId.value || !id || id <= prev) return
  await nextTick()
  scrollBottom()
})

// Открытие чата из системного уведомления — разворачиваем мини-чат, если
// пользователь не на странице мессенджера.
function handleExternalOpen(e) {
  if (hidden.value) return
  const id = e.detail?.conversation_id
  if (id) {
    open.value = true
    openThread(id)
  }
}

if (typeof window !== 'undefined') {
  window.addEventListener('messenger:open-conversation', handleExternalOpen)
}
</script>

<style scoped>
/* Поверх ActiveUnitModal (z-index 9999) — чтобы можно было ответить, не
   закрывая активный юнит. */
.mini-mess {
  position: fixed;
  right: 20px;
  bottom: 20px;
  z-index: 10050;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 12px;
}

.mini-fab {
  width: 56px;
  height: 56px;
  border-radius: var(--radius-full);
  border: none;
  background: var(--color-primary);
  color: var(--color-on-primary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: var(--shadow-lg);
  position: relative;
  transition: background 0.15s, transform 0.12s;
}

.mini-fab:hover { background: var(--color-primary-hover); transform: translateY(-1px); }
.mini-fab:active { transform: scale(0.96); }
.mini-fab.active { background: var(--color-surface-high); color: var(--color-text); }
.mini-fab .material-symbols-outlined { font-size: 26px; }

.mini-fab-badge {
  position: absolute;
  top: -2px;
  right: -2px;
  min-width: 20px;
  height: 20px;
  padding: 0 5px;
  border-radius: var(--radius-full);
  background: var(--color-error);
  color: var(--color-on-error);
  font-size: 11px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px solid var(--color-bg);
}

.mini-panel {
  width: 360px;
  max-width: calc(100vw - 32px);
  height: 70vh;
  max-height: 560px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-xl);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.mini-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
  flex-shrink: 0;
}

.mini-title {
  font-size: 15px;
  font-weight: 700;
  color: var(--color-text);
  flex: 1;
}

.mini-title--name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text);
}

.mini-head-avatar-wrap {
  position: relative;
  flex-shrink: 0;
}

.mini-head-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  object-fit: cover;
  display: block;
}

.online-dot {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--color-success);
  border: 2px solid var(--color-surface);
}

.mini-head-title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.mini-head-status {
  font-size: 11px;
  color: var(--color-text-dim);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mini-head-status.online {
  color: var(--color-success);
  font-weight: 600;
}

.mini-icon {
  width: 34px;
  height: 34px;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.mini-icon:hover { background: var(--color-surface-low); color: var(--color-text); }
.mini-icon .material-symbols-outlined { font-size: 20px; }

.mini-list {
  list-style: none;
  margin: 0;
  padding: 0;
  overflow-y: auto;
  flex: 1;
}

.mini-conv {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  cursor: pointer;
  transition: background 0.12s;
}

.mini-conv:hover { background: var(--color-surface-low); }

.mini-avatar-wrap {
  position: relative;
  flex-shrink: 0;
}

.mini-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
  display: block;
}

.mini-list-dot {
  width: 11px;
  height: 11px;
}

.mini-conv-body { flex: 1; min-width: 0; }

.mini-conv-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mini-conv-preview {
  font-size: 12.5px;
  color: var(--color-text-dim);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.mini-conv.unread .mini-conv-preview { color: var(--color-text); font-weight: 500; }

.mini-badge {
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 11px;
  font-weight: 700;
  padding: 2px 7px;
  border-radius: var(--radius-full);
  min-width: 20px;
  text-align: center;
  flex-shrink: 0;
}

.mini-thread {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
  background: var(--color-bg);
  min-height: 0;
  position: relative;
}

.mini-drop-overlay {
  position: absolute;
  inset: 6px;
  z-index: 5;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  background: color-mix(in oklch, var(--color-primary-container) 90%, transparent);
  border: 2px dashed var(--color-primary);
  border-radius: var(--radius-md);
  color: var(--color-on-primary-container);
  font-size: 13px;
  font-weight: 600;
  pointer-events: none;
}

.mini-drop-overlay .material-symbols-outlined { font-size: 32px; }

.mini-loading { display: flex; justify-content: center; padding: 12px; }

.mini-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--color-text-dim);
}

.mini-empty .material-symbols-outlined { font-size: 40px; }

.mini-pop-enter-active, .mini-pop-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
  transform-origin: bottom right;
}
.mini-pop-enter-from, .mini-pop-leave-to {
  opacity: 0;
  transform: scale(0.92) translateY(8px);
}

@media (max-width: 768px) {
  .mini-mess {
    right: 12px;
    bottom: calc(70px + env(safe-area-inset-bottom, 0px));
  }
  .mini-panel {
    width: calc(100vw - 24px);
    height: 70vh;
  }
}
</style>
