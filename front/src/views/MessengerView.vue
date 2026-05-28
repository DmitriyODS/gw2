<template>
  <div class="messenger" :class="{ 'mobile-chat-open': isMobile && activeId }">
    <ConversationList
      :conversations="messenger.conversations"
      :active-id="activeId"
      :loading="messenger.loadingList"
      :hide-on-mobile="isMobile && !!activeId"
      @select="selectConversation"
      @new-chat="newChatOpen = true"
      @toggle-pin="onTogglePin"
      @delete="askDeleteConversation"
    />

    <section
      class="chat-panel"
      :class="{ 'is-mobile-hidden': isMobile && !activeId }"
      @dragenter.prevent="onDragEnter"
      @dragover.prevent="onDragOver"
      @dragleave.prevent="onDragLeave"
      @drop.prevent="onDrop"
    >
      <div v-if="dragOver && active" class="chat-drop-overlay">
        <span class="material-symbols-outlined">upload_file</span>
        <span>Отпустите файл — он прикрепится к сообщению</span>
      </div>
      <header v-if="active" class="chat-header">
        <button v-if="isMobile" class="back-btn" @click="goBack" title="Назад">
          <span class="material-symbols-outlined">arrow_back</span>
        </button>
        <div class="chat-avatar-wrap">
          <img class="chat-avatar" :src="avatarOf(active.other_user)" :alt="active.other_user?.fio" />
          <span v-if="otherOnline" class="online-dot" title="В сети"></span>
        </div>
        <div class="chat-title">
          <div class="chat-fio">{{ active.other_user?.fio }}</div>
          <div class="chat-status" :class="{ online: otherOnline }">
            {{ otherOnline ? 'в сети' : lastSeenText }}
          </div>
        </div>
        <div class="chat-tools" data-tutorial="chat-tools" ref="toolsRef">
          <button
            class="chat-tool"
            :class="{ active: chatMenuOpen }"
            title="Действия"
            aria-haspopup="menu"
            :aria-expanded="chatMenuOpen"
            @click="chatMenuOpen = !chatMenuOpen"
          >
            <span class="material-symbols-outlined">more_vert</span>
          </button>
          <Transition name="chat-menu">
            <div v-if="chatMenuOpen" class="chat-menu" role="menu">
              <button
                class="chat-menu-item"
                data-tutorial="chat-call-audio"
                @click="onMenuAction(() => startCall('audio'))"
              >
                <span class="material-symbols-outlined chat-menu-ico tone-success">call</span>
                <span>Аудиозвонок</span>
              </button>
              <button
                class="chat-menu-item"
                data-tutorial="chat-call-video"
                @click="onMenuAction(() => startCall('video'))"
              >
                <span class="material-symbols-outlined chat-menu-ico tone-success">videocam</span>
                <span>Видеозвонок</span>
              </button>
              <button
                class="chat-menu-item"
                @click="onMenuAction(() => onTogglePin(active.id))"
              >
                <span class="material-symbols-outlined chat-menu-ico" :class="{ 'tone-tertiary': active.is_pinned }">
                  {{ active.is_pinned ? 'keep_off' : 'keep' }}
                </span>
                <span>{{ active.is_pinned ? 'Открепить чат' : 'Закрепить чат' }}</span>
              </button>
              <div class="chat-menu-divider" />
              <button
                class="chat-menu-item danger"
                @click="onMenuAction(() => askDeleteConversation(active))"
              >
                <span class="material-symbols-outlined chat-menu-ico tone-error">delete</span>
                <span>Удалить чат</span>
              </button>
            </div>
          </Transition>
        </div>
      </header>
      <div v-else class="chat-empty">
        <div class="chat-empty-icon">
          <span class="material-symbols-outlined">chat</span>
        </div>
        <h3 class="chat-empty-title">Выберите чат</h3>
        <p class="chat-empty-text">Откройте беседу слева — или начните новую из списка.</p>
      </div>

      <div
        v-if="active"
        ref="messagesEl"
        class="messages-area"
        @scroll="onScroll"
      >
        <div v-if="messenger.loadingMessages && !messenger.activeMessages.length" class="msg-loading">
          <ProgressSpinner style="width:32px;height:32px" />
        </div>
        <template v-else>
          <MessageBubble
            v-for="m in messenger.activeMessages"
            :key="m.id"
            :message="m"
            :is-mine="m.sender_id === authStore.user?.id"
            @delete="askDeleteMessage"
            @reply="startReply"
            @forward="startForward"
            @join-call="onJoinCall"
          />
        </template>
      </div>

      <MessageInput
        v-if="active"
        ref="messageInputRef"
        :sending="messenger.sending"
        :reply-to="replyTo"
        @send="onSend"
        @cancel-reply="replyTo = null"
      />
    </section>

    <!-- FAB «новый чат» — только на мобильном и только в режиме списка -->
    <Teleport to="body">
      <button
        v-if="isMobile && !activeId"
        class="fab"
        @click="newChatOpen = true"
        aria-label="Новый чат"
      >
        <span class="material-symbols-outlined">edit_square</span>
      </button>
    </Teleport>

    <NewChatDialog v-model="newChatOpen" @pick="startWith" />

    <ForwardDialog
      ref="forwardDialogRef"
      v-model="forwardOpen"
      :message="forwardSource"
      @confirm="onForwardConfirm"
    />

    <DeleteScopeDialog
      v-model="deleteDialogOpen"
      :title="deleteDialog.title"
      :text="deleteDialog.text"
      :can-for-all="deleteDialog.canForAll"
      :other-name="deleteDialog.otherName"
      @confirm="onDeleteConfirm"
    />
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessengerStore } from '@/stores/messenger.js'
import { useAuthStore } from '@/stores/auth.js'
import { useCallStore } from '@/stores/call.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import {
  requestNotificationPermission, notificationsAllowed,
} from '@/utils/systemNotify.js'
import { formatLastSeen } from '@/utils/presence.js'
import ConversationList from '@/components/messenger/ConversationList.vue'
import MessageBubble from '@/components/messenger/MessageBubble.vue'
import MessageInput from '@/components/messenger/MessageInput.vue'
import NewChatDialog from '@/components/messenger/NewChatDialog.vue'
import DeleteScopeDialog from '@/components/messenger/DeleteScopeDialog.vue'
import ForwardDialog from '@/components/messenger/ForwardDialog.vue'
import ProgressSpinner from 'primevue/progressspinner'

const route = useRoute()
const router = useRouter()
const messenger = useMessengerStore()
const callStore = useCallStore()

async function startCall(media) {
  const other = active.value?.other_user
  if (!other) return
  try {
    await callStore.startCall({ userIds: [other.id], media, conversationId: active.value.id })
  } catch {/* ошибка отображена в store.error */}
}

async function onJoinCall(callInfo) {
  await callStore.joinExistingCall(callInfo)
}
const authStore = useAuthStore()
const { isMobile } = useBreakpoint()

const newChatOpen = ref(false)
const messagesEl = ref(null)
const messageInputRef = ref(null)
const dragOver = ref(false)
let dragDepth = 0
const replyTo = ref(null)

const chatMenuOpen = ref(false)
const toolsRef = ref(null)

function onMenuAction(fn) {
  chatMenuOpen.value = false
  fn()
}

function handleOutsideMenu(e) {
  if (!chatMenuOpen.value) return
  const root = toolsRef.value
  if (root && !root.contains(e.target)) chatMenuOpen.value = false
}

// При переключении диалога закрываем меню действий — иначе оно остаётся открытым
// поверх шапки нового чата.
watch(() => messenger.activeConversationId, () => { chatMenuOpen.value = false })

function dragHasFiles(e) {
  const types = e.dataTransfer?.types
  return types && Array.from(types).includes('Files')
}

function onDragEnter(e) {
  if (!active.value || !dragHasFiles(e)) return
  dragDepth++
  dragOver.value = true
}

function onDragOver(e) {
  if (active.value && dragHasFiles(e)) e.dataTransfer.dropEffect = 'copy'
}

function onDragLeave() {
  dragDepth = Math.max(0, dragDepth - 1)
  if (dragDepth === 0) dragOver.value = false
}

async function onDrop(e) {
  dragDepth = 0
  dragOver.value = false
  if (!active.value) return
  const files = Array.from(e.dataTransfer?.files || [])
  if (files.length) messageInputRef.value?.addFiles(files)
}
const forwardOpen = ref(false)
const forwardSource = ref(null)
const forwardDialogRef = ref(null)

function startReply(message) {
  replyTo.value = {
    id: message.id,
    sender_fio: message.sender_id === authStore.user?.id
      ? 'Вы'
      : (active.value?.other_user?.fio || ''),
    text: message.text,
    has_attachments: !!message.attachments?.length,
  }
}

function startForward(message) {
  forwardSource.value = message
  forwardOpen.value = true
}

async function onForwardConfirm({ userIds }) {
  try {
    await messenger.forwardMessage(forwardSource.value.id, { userIds })
  } catch (e) {
    console.error('forward failed', e)
  } finally {
    forwardDialogRef.value?.stopSending()
    forwardOpen.value = false
    forwardSource.value = null
  }
}

const deleteDialogOpen = ref(false)
const deleteDialog = ref({
  title: '',
  text: '',
  canForAll: true,
  otherName: '',
  payload: null,        // { kind: 'message' | 'conversation', id }
})

function askDeleteMessage(message) {
  const isMine = message.sender_id === authStore.user?.id
  const other = active.value?.other_user?.fio || ''
  deleteDialog.value = {
    title: 'Удалить сообщение?',
    text: isMine
      ? 'Сообщение исчезнет у вас. Можно также удалить его у собеседника.'
      : 'Сообщение скроется только у вас — у собеседника останется.',
    canForAll: isMine,
    otherName: other,
    payload: { kind: 'message', id: message.id },
  }
  deleteDialogOpen.value = true
}

function askDeleteConversation(conv) {
  const other = conv?.other_user?.fio || ''
  deleteDialog.value = {
    title: 'Удалить чат?',
    text: 'Чат пропадёт у вас. Можно также удалить его у собеседника — переписка исчезнет у обоих.',
    canForAll: true,
    otherName: other,
    payload: { kind: 'conversation', id: conv.id },
  }
  deleteDialogOpen.value = true
}

async function onDeleteConfirm({ scope }) {
  const p = deleteDialog.value.payload
  if (!p) return
  try {
    if (p.kind === 'message') {
      await messenger.deleteMessage(p.id, scope)
    } else if (p.kind === 'conversation') {
      await messenger.deleteConversationAction(p.id, scope)
      if (activeId.value === p.id) {
        router.replace('/messenger')
      }
    }
  } catch (e) {
    console.error('delete failed', e)
  }
}

async function onTogglePin(conversationId) {
  try {
    await messenger.togglePinAction(conversationId)
  } catch (e) {
    console.error('pin failed', e)
  }
}

const activeId = computed(() => messenger.activeConversationId)
const active = computed(() => messenger.activeConversation)

const otherOnline = computed(() => messenger.isOnline(active.value?.other_user?.id))
const lastSeenText = computed(() => {
  const u = active.value?.other_user
  if (!u) return ''
  return formatLastSeen(messenger.lastSeenOf(u.id, u.last_seen_at))
})

function avatarOf(u) {
  if (!u) return ''
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

async function selectConversation(id) {
  replyTo.value = null
  await messenger.setActive(id)
  router.replace(`/messenger/${id}`)
  await nextTick()
  scrollToBottom()
}

async function startWith(user) {
  const id = await messenger.openWith(user.id)
  router.replace(`/messenger/${id}`)
  await nextTick()
  scrollToBottom()
}

async function onSend(payload) {
  await messenger.send(activeId.value, payload)
  replyTo.value = null
  await nextTick()
  scrollToBottom()
}

function goBack() {
  messenger.activeConversationId = null
  router.replace('/messenger')
}

function scrollToBottom() {
  const el = messagesEl.value
  if (!el) return
  el.scrollTop = el.scrollHeight
}

// Локальный гард, чтобы scroll-событие не запускало вторую подгрузку,
// пока первая ещё в полёте, и не падало в бесконечный «магнит» к верху,
// если страница вернулась пустой.
let loadingOlder = false

async function onScroll() {
  const el = messagesEl.value
  if (!el || loadingOlder) return
  if (el.scrollTop > 80) return
  if (!messenger.hasMoreHistory(activeId.value)) return
  const arr = messenger.activeMessages
  if (!arr.length) return

  loadingOlder = true
  try {
    const firstId = arr[0].id
    const prevHeight = el.scrollHeight
    const prevTop = el.scrollTop
    const added = await messenger.fetchMessages(activeId.value, firstId)
    if (!added || !added.length) return
    await nextTick()
    // Сохраняем визуальную позицию: пиксель, на который смотрел пользователь,
    // должен остаться на том же месте после вставки старых сообщений сверху.
    const delta = el.scrollHeight - prevHeight
    if (delta > 0) {
      el.scrollTop = prevTop + delta
    }
  } finally {
    loadingOlder = false
  }
}

function handleExternalOpen(e) {
  const id = e.detail?.conversation_id
  if (id) {
    selectConversation(id)
  }
}

onMounted(async () => {
  await messenger.fetchConversations()
  if (notificationsAllowed() === false) {
    requestNotificationPermission()
  }
  const urlId = Number(route.params.conversationId)
  if (urlId && messenger.conversations.some(c => c.id === urlId)) {
    await messenger.setActive(urlId)
  }
  await nextTick()
  scrollToBottom()
  window.addEventListener('messenger:open-conversation', handleExternalOpen)
  document.addEventListener('mousedown', handleOutsideMenu)
  document.addEventListener('touchstart', handleOutsideMenu, { passive: true })
})

onBeforeUnmount(() => {
  window.removeEventListener('messenger:open-conversation', handleExternalOpen)
  document.removeEventListener('mousedown', handleOutsideMenu)
  document.removeEventListener('touchstart', handleOutsideMenu)
})

/* Скроллим вниз только когда появилось НОВОЕ сообщение снизу (lastId вырос),
   а не любая мутация длины: подгрузка старых сверху тоже увеличивает length,
   но скроллить в конец тогда нельзя — пользователь читает историю. */
const lastMessageId = computed(() => {
  const arr = messenger.activeMessages
  return arr.length ? arr[arr.length - 1].id : 0
})

watch(lastMessageId, async (id, prevId) => {
  if (!id || id <= prevId) return
  await nextTick()
  const el = messagesEl.value
  if (!el) return
  const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 200
  if (nearBottom) scrollToBottom()
})

watch(() => route.params.conversationId, async (id) => {
  const n = Number(id)
  if (n && n !== messenger.activeConversationId) {
    await messenger.setActive(n)
    await nextTick()
    scrollToBottom()
  }
})
</script>

<style scoped>
.messenger {
  display: flex;
  height: 100%;
  min-height: 0;
  background: var(--color-bg);
}

.chat-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  position: relative;
}

/* Зона сброса файлов — на всю область чата, а не только на поле ввода. */
.chat-drop-overlay {
  position: absolute;
  inset: 8px;
  z-index: 50;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  background: color-mix(in oklch, var(--color-primary-container) 90%, transparent);
  border: 2px dashed var(--color-primary);
  border-radius: var(--radius-lg);
  color: var(--color-on-primary-container);
  font-size: 15px;
  font-weight: 600;
  text-align: center;
  padding: 16px;
  pointer-events: none;
}

.chat-drop-overlay .material-symbols-outlined { font-size: 44px; }

.chat-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
  flex-shrink: 0;
}

.back-btn {
  background: none;
  border: none;
  cursor: pointer;
  color: var(--color-text);
  display: flex;
  align-items: center;
}

.chat-avatar-wrap {
  position: relative;
  flex-shrink: 0;
}

.chat-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
  display: block;
}

.chat-avatar-wrap .online-dot {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 11px;
  height: 11px;
  border-radius: 50%;
  background: var(--color-success);
  border: 2px solid var(--color-surface);
}

.chat-title { min-width: 0; flex: 1; }

.chat-status {
  font-size: 12px;
  color: var(--color-text-dim);
}

.chat-status.online {
  color: var(--color-success);
  font-weight: 600;
}

.chat-tools {
  position: relative;
  display: flex;
  gap: 2px;
  margin-left: auto;
  flex-shrink: 0;
}

.chat-tool {
  width: 40px;
  height: 40px;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s, color 0.15s;
}

.chat-tool:hover {
  background: var(--color-surface-low);
  color: var(--color-text);
}

.chat-tool.active {
  background: var(--color-surface-low);
  color: var(--color-text);
}

.chat-tool .material-symbols-outlined { font-size: 22px; }

/* Выпадающее меню действий по чату */
.chat-menu {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  min-width: 220px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  padding: 6px;
  box-shadow: var(--shadow-lg);
  z-index: 60;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.chat-menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: none;
  background: transparent;
  color: var(--color-text);
  font-size: 14px;
  font-weight: 500;
  text-align: left;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.chat-menu-item:hover {
  background: var(--color-surface-low);
}

.chat-menu-item.danger { color: var(--color-error); }

.chat-menu-item.danger:hover {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}

.chat-menu-ico {
  font-size: 20px;
  color: var(--color-text-dim);
}

.chat-menu-ico.tone-success { color: var(--color-success); }
.chat-menu-ico.tone-tertiary { color: var(--color-tertiary); }
.chat-menu-ico.tone-error { color: var(--color-error); }

.chat-menu-divider {
  height: 1px;
  background: var(--color-outline-dim);
  margin: 4px 4px;
}

.chat-menu-enter-active,
.chat-menu-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
  transform-origin: top right;
}

.chat-menu-enter-from,
.chat-menu-leave-to {
  opacity: 0;
  transform: scale(0.96) translateY(-4px);
}

.chat-fio {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text);
}

.chat-meta {
  font-size: 12px;
  color: var(--color-text-dim);
}

.chat-empty {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: var(--color-text-dim);
  text-align: center;
  padding: 24px;
}

.chat-empty-icon {
  width: 96px;
  height: 96px;
  border-radius: 50%;
  background: var(--color-surface-high);
  color: var(--color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 6px;
}

.chat-empty-icon .material-symbols-outlined {
  font-size: 48px;
  font-variation-settings: 'FILL' 1, 'wght' 400, 'GRAD' 0, 'opsz' 48;
}

.chat-empty-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text);
}

.chat-empty-text {
  margin: 0;
  font-size: 13.5px;
  line-height: 1.45;
  color: var(--color-text-dim);
  max-width: 320px;
}

.messages-area {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  background: var(--color-bg);
  min-height: 0;
}

.msg-loading {
  display: flex;
  justify-content: center;
  padding: 16px;
}

.is-mobile-hidden { display: none; }

@media (max-width: 768px) {
  /* Статичный полноэкранный макет: фиксируем к вьюпорту, чтобы экран не «ёрзал»
     при показе/скрытии адресной строки браузера. Нижняя навигация (z-index 200)
     остаётся поверх; снизу резервируем под неё высоту. */
  .messenger {
    position: fixed;
    inset: 0;
    height: auto;
    z-index: 100;
  }

  .chat-panel {
    position: fixed;
    inset: 0;
    z-index: 150;
    background: var(--color-bg);
    /* Резерв ровно под нижнюю навигацию (её высота = 64px + safe-area внутри),
       без лишнего «воздуха» под полем ввода. */
    padding-bottom: calc(64px + env(safe-area-inset-bottom, 0px));
  }
  .messenger.mobile-chat-open .chat-panel {
    display: flex;
  }
}

/* Мобильный FAB «новый чат» — поведение и вид как на экране задач. */
.fab {
  display: none;
}

@media (max-width: 768px) {
  .fab {
    position: fixed;
    right: 16px;
    bottom: calc(64px + 16px + env(safe-area-inset-bottom, 0px));
    width: 56px;
    height: 56px;
    border-radius: 50%;
    border: none;
    background: var(--gw-primary);
    color: var(--color-on-primary);
    box-shadow: 0 4px 14px color-mix(in oklch, var(--gw-primary) 50%, transparent);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 210;
    transition: transform 0.28s cubic-bezier(0.4, 0, 0.2, 1),
                background 0.15s;
  }

  .fab:active {
    background: var(--gw-primary-hover);
    transform: scale(0.96);
  }

  .fab .material-symbols-outlined {
    font-size: 24px;
  }
}
</style>
