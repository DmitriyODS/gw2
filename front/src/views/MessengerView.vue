<template>
  <div class="messenger" :class="{ 'mobile-chat-open': isMobile && activeId }">
    <ConversationList
      :conversations="visibleConversations"
      :active-id="activeId"
      :loading="listLoading"
      :hide-on-mobile="isMobile && !!activeId"
      :show-support-tab="authStore.isSuperAdmin"
      :tab="listTab"
      :support-unread="supportTabUnread"
      @select="selectConversation"
      @new-chat="newChatOpen = true"
      @new-call="startEmptyCall"
      @toggle-pin="onTogglePin"
      @delete="askDeleteConversation"
      @change-tab="onChangeTab"
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
        <!-- 3 варианта шапки: обычный диалог (фото собеседника), dev-чат
             у владельца (иконка support_agent), dev-чат у админа в support-inbox
             (фото пользователя, который написал в поддержку). -->
        <button
          v-if="active.is_dev_chat && devChatOwner"
          class="chat-avatar-wrap as-btn"
          aria-label="Открыть профиль пользователя"
          @click="profileOpen = true"
        >
          <img class="chat-avatar" :src="avatarOf(devChatOwner)" :alt="devChatOwner.fio" />
          <span v-if="messenger.isOnline(devChatOwner.id)" class="online-dot" title="В сети"></span>
        </button>
        <div v-else-if="active.is_dev_chat" class="chat-avatar-wrap dev">
          <span class="material-symbols-outlined">support_agent</span>
        </div>
        <button
          v-else
          class="chat-avatar-wrap as-btn"
          aria-label="Открыть профиль"
          @click="profileOpen = true"
        >
          <img class="chat-avatar" :src="avatarOf(active.other_user)" :alt="active.other_user?.fio" />
          <span v-if="otherOnline" class="online-dot" title="В сети"></span>
        </button>
        <div
          class="chat-title"
          :class="{ 'as-btn': !!profileUser }"
          :role="profileUser ? 'button' : null"
          :tabindex="profileUser ? 0 : null"
          @click="profileUser && (profileOpen = true)"
          @keydown.enter="profileUser && (profileOpen = true)"
        >
          <div class="chat-fio">
            <template v-if="active.is_dev_chat && devChatOwner">{{ devChatOwner.fio }}</template>
            <template v-else-if="active.is_dev_chat">Техподдержка</template>
            <template v-else>{{ active.other_user?.fio }}</template>
            <span v-if="peerStatusEmoji" class="chat-fio-status" :title="peerStatusText || 'Статус'">{{ peerStatusEmoji }}</span>
          </div>
          <div class="chat-status" :class="{ online: chatOnline || peerTyping }">
            <template v-if="active.is_dev_chat && devChatOwner">
              <span v-if="active.company_name">{{ active.company_name }} · </span>
              <template v-if="chatOnline">в сети</template>
              <template v-else>{{ ownerLastSeenText }}</template>
            </template>
            <template v-else-if="active.is_dev_chat">
              Личный чат с командой разработчиков
            </template>
            <template v-else-if="peerTyping">печатает…</template>
            <template v-else>
              <span v-if="peerStatusText" class="chat-status-note">{{ peerStatusText }} · </span>
              {{ otherOnline ? 'в сети' : lastSeenText }}
            </template>
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
                v-if="!active.is_dev_chat"
                class="chat-menu-item"
                data-tutorial="chat-call-audio"
                @click="onMenuAction(() => startCall('audio'))"
              >
                <span class="material-symbols-outlined chat-menu-ico tone-success">call</span>
                <span>Аудиозвонок</span>
              </button>
              <button
                v-if="!active.is_dev_chat"
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
              <template v-if="!active.is_dev_chat">
                <div class="chat-menu-divider" />
                <button
                  class="chat-menu-item danger"
                  @click="onMenuAction(() => askDeleteConversation(active))"
                >
                  <span class="material-symbols-outlined chat-menu-ico tone-error">delete</span>
                  <span>Удалить чат</span>
                </button>
              </template>
            </div>
          </Transition>
        </div>
      </header>
      <EmptyState
        v-else
        class="chat-empty"
        icon="chat"
        title="Выберите чат"
        subtitle="Откройте беседу слева — или начните новую из списка."
      />

      <!-- Баннер закреплённых сообщений. Клик по тексту прокручивает к
           сообщению и листает закреплённые; кнопка справа — открепить. -->
      <div v-if="active && pinnedMessages.length" class="pinned-bar" @click="cyclePinned">
        <span class="material-symbols-outlined pinned-bar-icon">keep</span>
        <div class="pinned-bar-body">
          <div class="pinned-bar-title">
            Закреплённое
            <span v-if="pinnedMessages.length > 1" class="pinned-bar-count">
              {{ pinnedIndex + 1 }}/{{ pinnedMessages.length }}
            </span>
          </div>
          <div class="pinned-bar-text">{{ pinnedPreview }}</div>
        </div>
        <button class="pinned-bar-unpin" title="Открепить" @click.stop="unpinMessage(currentPinned)">
          <span class="material-symbols-outlined">close</span>
        </button>
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
          <div v-if="loadingOlder" class="msg-loading-older">
            <ProgressSpinner style="width:22px;height:22px" />
            <span>Загружаем историю…</span>
          </div>
          <MessageBubble
            v-for="m in messenger.activeMessages"
            :key="m.id"
            v-memo="[m.id, m.text, m.edited_at, m.read_at, m.pinned_at, m.reactions, m.call?.status, authStore.user?.id, active?.is_dev_chat]"
            :message="m"
            :is-mine="m.sender_id === authStore.user?.id"
            :sender-name="senderNameFor(m)"
            :me-id="authStore.user?.id"
            @delete="askDeleteMessage"
            @reply="startReply"
            @forward="startForward"
            @pin="onTogglePinMessage"
            @join-call="onJoinCall"
            @open-task="openTask"
            @open-post="openPost"
            @context-menu="openContextMenu"
            @quote-click="onQuoteClick"
            @react="emoji => onReact(m, emoji)"
          />
        </template>
      </div>

      <Transition name="jump-down">
        <button
          v-if="active && showJumpDown"
          class="jump-down-btn"
          :style="{ bottom: jumpDownBottom }"
          aria-label="К последним сообщениям"
          @click="scrollToBottomSmooth"
        >
          <span class="material-symbols-outlined">keyboard_arrow_down</span>
        </button>
      </Transition>

      <MessageInput
        v-if="active"
        ref="messageInputRef"
        :sending="messenger.sending"
        :reply-to="replyTo"
        :editing-message="editing"
        v-model:attached-task="attachedTask"
        @send="onSend"
        @save-edit="onSaveEdit"
        @cancel-reply="replyTo = null"
        @cancel-edit="editing = null"
        @attach-task="attachTaskOpen = true"
        @typing="onTyping"
      />
    </section>

    <AppFab
      :visible="isMobile && !activeId && listTab === 'chats'"
      icon="edit_square"
      tone="tertiary"
      aria-label="Новый чат"
      @click="newChatOpen = true"
    />

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

    <AttachTaskDialog
      v-model="attachTaskOpen"
      :company-id="active?.company_id ?? null"
      @pick="onPickTask"
    />

    <EmployeeProfileDialog
      v-if="profileUser"
      v-model="profileOpen"
      :user="profileUser"
    />

    <MessageContextMenu
      :visible="ctxMenu.visible"
      :x="ctxMenu.x"
      :y="ctxMenu.y"
      :is-pinned="!!ctxMenu.message?.pinned_at"
      :show-edit="canEditCtxMessage"
      :show-pin="!active?.is_dev_chat"
      :show-forward="!active?.is_dev_chat"
      :show-copy="!!ctxMenu.message?.text"
      :show-delete="true"
      :my-reactions="ctxMyReactions"
      @close="ctxMenu.visible = false"
      @action="onCtxAction"
      @react="onCtxReact"
    />
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessengerStore } from '@/stores/messenger.js'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useCallStore } from '@/stores/call.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { useFileDrop } from '@/composables/useFileDrop.js'
import { useJumpToMessage } from '@/composables/useJumpToMessage.js'
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
import AttachTaskDialog from '@/components/messenger/AttachTaskDialog.vue'
import MessageContextMenu from '@/components/messenger/MessageContextMenu.vue'
import EmployeeProfileDialog from '@/components/common/EmployeeProfileDialog.vue'
import AppFab from '@/components/common/AppFab.vue'
import EmptyState from '@/components/common/EmptyState.vue'
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

/* «Пустой звонок»: комната с одним собой — коллег зовут уже из звонка
   (кнопка person_add или ссылка-приглашение). Стартуем без камеры:
   видео каждый включает сам по желанию. */
async function startEmptyCall() {
  try {
    await callStore.startCall({ userIds: [], media: 'video', videoOff: true })
  } catch {/* ошибка отображена в store.error */}
}
const authStore = useAuthStore()
const { isMobile } = useBreakpoint()

const newChatOpen = ref(false)
const attachTaskOpen = ref(false)
const attachedTask = ref(null)
const profileOpen = ref(false)
const ctxMenu = ref({ visible: false, x: 0, y: 0, message: null })
const messagesEl = ref(null)
const messageInputRef = ref(null)
const replyTo = ref(null)
const editing = ref(null)

// Редактировать можно только своё текстовое сообщение (не бота, не плашку).
const canEditCtxMessage = computed(() => {
  const m = ctxMenu.value.message
  if (!m) return false
  const isText = !m.kind || m.kind === 'text'
  return isText && !m.is_bot && m.sender_id === authStore.user?.id && !!m.text
})
const {
  dragOver,
  onDragEnter,
  onDragOver,
  onDragLeave,
  onDrop,
} = useFileDrop({
  canDrop: () => !!active.value,
  onFiles: files => messageInputRef.value?.addFiles(files),
})

function openContextMenu({ x, y, message }) {
  ctxMenu.value = { visible: true, x, y, message }
}

function onCtxAction(action) {
  const m = ctxMenu.value.message
  if (!m) return
  if (action === 'reply') startReply(m)
  else if (action === 'edit') startEdit(m)
  else if (action === 'forward') startForward(m)
  else if (action === 'pin') onTogglePinMessage(m)
  else if (action === 'delete') askDeleteMessage(m)
  else if (action === 'copy') copyMessageText(m)
}

// Мои реакции на сообщении контекстного меню (подсветка быстрого ряда).
const ctxMyReactions = computed(() => {
  const m = ctxMenu.value.message
  if (!m) return []
  const me = authStore.user?.id
  return (m.reactions || []).filter(r => r.user_id === me).map(r => r.emoji)
})

async function onReact(message, emoji) {
  try {
    await messenger.toggleReactionAction(message.id, emoji)
  } catch (e) {
    useNotificationsStore().error(e?.message || 'Не удалось поставить реакцию')
  }
}

function onCtxReact(emoji) {
  const m = ctxMenu.value.message
  if (m) onReact(m, emoji)
}

function startEdit(message) {
  replyTo.value = null
  editing.value = message
  nextTick(() => messageInputRef.value?.focus())
}

async function onSaveEdit(text) {
  const m = editing.value
  if (!m) return
  editing.value = null
  if (!text || text === m.text) return
  try {
    await messenger.editMessage(activeId.value, m.id, text)
  } catch (e) {
    useNotificationsStore().error(e?.message || 'Не удалось изменить сообщение')
  }
}

function copyMessageText(m) {
  if (!m.text) return
  if (navigator.clipboard?.writeText) {
    navigator.clipboard.writeText(m.text).catch(() => {/* no-op */})
  }
}

function onPickTask(task) { attachedTask.value = task }

function openTask(taskId) {
  router.push({ path: '/tasks', query: { open: taskId } })
}

function openPost(postId) {
  router.push(`/portal/${postId}`)
}

function senderNameFor(m) {
  // В dev-чате сообщения от админа техподдержки всегда подписываются
  // «Техподдержка» — ФИО админа намеренно скрыто (как у Telegram support-ботов).
  // В обычных p2p-диалогах подпись не нужна — там и так только один собеседник.
  if (m.is_from_support) return 'Техподдержка'
  return ''
}

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

async function activateRouteConversation() {
  const rawId = route.params.conversationId
  const n = Number(rawId)
  if (!n) return
  // Проверяем по объединённому индексу (обычные диалоги + support-inbox у
  // рут-админа) — иначе глубокая ссылка на support-чат не активировалась.
  if (!messenger.conversationById.get(n)) return
  if (messenger.activeConversationId !== n) {
    await messenger.setActive(n)
    await nextTick()
    scrollToBottom()
  }
}

// При переключении диалога закрываем меню действий — иначе оно остаётся открытым
// поверх шапки нового чата.
watch(() => messenger.activeConversationId, () => { chatMenuOpen.value = false })

const forwardOpen = ref(false)
const forwardSource = ref(null)
const forwardDialogRef = ref(null)

function startReply(message) {
  editing.value = null
  replyTo.value = {
    id: message.id,
    sender_fio: message.sender_id === authStore.user?.id
      ? 'Вы'
      : (active.value?.other_user?.fio || ''),
    text: message.text,
    kind: message.kind,
    has_attachments: !!message.attachments?.length,
  }
  // Сразу в поле ввода — можно писать ответ без лишнего клика.
  nextTick(() => messageInputRef.value?.focus())
}

function startForward(message) {
  forwardSource.value = message
  forwardOpen.value = true
}

async function onForwardConfirm({ userIds }) {
  try {
    await messenger.forwardMessage(forwardSource.value.id, { userIds })
    useNotificationsStore().success('Сообщение переслано')
  } catch (e) {
    console.error('forward failed', e)
    useNotificationsStore().error(e?.message || 'Не удалось переслать сообщение')
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

/* ── Закреплённые сообщения ─────────────────────────────────────── */
const pinnedMessages = computed(() => messenger.activePinned)
const pinnedIndex = ref(0)
const currentPinned = computed(() => pinnedMessages.value[pinnedIndex.value] || null)
const pinnedPreview = computed(() => {
  const m = currentPinned.value
  if (!m) return ''
  if (m.kind === 'call') {
    return m.call?.media === 'audio' ? '📞 Аудиозвонок' : '📹 Видеозвонок'
  }
  if (m.text) return m.text
  if (m.attachments?.length) return 'Вложение'
  return 'Сообщение'
})

// При смене чата/списка закреплённых держим индекс в границах.
watch(pinnedMessages, (list) => {
  if (pinnedIndex.value >= list.length) pinnedIndex.value = 0
})

async function onTogglePinMessage(message) {
  try {
    await messenger.togglePinMessageAction(message.id)
  } catch (e) {
    console.error('pin message failed', e)
  }
}

// Переход к сообщению с подсветкой и догрузкой истории — общий с MiniMessenger.
const { jumping, jumpToMessage } = useJumpToMessage({
  container: messagesEl,
  getMessages: () => messenger.activeMessages,
  hasMore: () => messenger.hasMoreHistory(activeId.value),
  loadOlder: (beforeId) => messenger.fetchMessages(activeId.value, beforeId),
})

async function onQuoteClick(id) {
  if (!await jumpToMessage(id)) {
    useNotificationsStore().warn('Сообщение не найдено')
  }
}

function cyclePinned() {
  const list = pinnedMessages.value
  if (!list.length) return
  const m = list[pinnedIndex.value]
  if (m) jumpToMessage(m.id)
  // Следующий клик — к следующему закреплённому.
  pinnedIndex.value = (pinnedIndex.value + 1) % list.length
}

async function unpinMessage(message) {
  if (!message) return
  await onTogglePinMessage(message)
}

const activeId = computed(() => messenger.activeConversationId)
const active = computed(() => messenger.activeConversation)

// Активная вкладка списка слева: 'chats' | 'support'. Вторая доступна
// только Администратору системы (он отвечает в чужие чаты техподдержки).
const listTab = ref('chats')

/* Для рут-админа техподдержка — отдельный inbox (входящие из чужих компаний),
   отображается на собственной вкладке. У обычных пользователей dev-чат с
   техподдержкой — это обычный диалог в общем списке, без отдельной вкладки. */
const visibleConversations = computed(() => {
  if (authStore.isSuperAdmin && listTab.value === 'support') {
    return messenger.supportInbox
  }
  return messenger.conversations
})

const listLoading = computed(() =>
  listTab.value === 'support' && authStore.isSuperAdmin
    ? messenger.loadingSupportInbox
    : messenger.loadingList
)

const supportTabUnread = computed(() =>
  authStore.isSuperAdmin ? messenger.supportUnread : 0
)

async function onChangeTab(t) {
  listTab.value = t
  if (t === 'support' && authStore.isSuperAdmin && !messenger.supportInbox.length) {
    await messenger.fetchSupportInbox()
  }
}

const otherOnline = computed(() => messenger.isOnline(active.value?.other_user?.id))

// «Печатает…» собеседника; свои сигналы шлём из поля ввода (типизацию
// дев-чата не транслируем — у него нет единственного адресата).
const peerTyping = computed(() =>
  !!active.value && !active.value.is_dev_chat && messenger.isTyping(active.value.id)
)

function onTyping(isTypingNow) {
  const id = active.value?.id
  if (!id) return
  if (isTypingNow) messenger.notifyTyping(id)
  else messenger.notifyTypingStop(id)
}
const lastSeenText = computed(() => {
  const u = active.value?.other_user
  if (!u) return ''
  return formatLastSeen(messenger.lastSeenOf(u.id, u.last_seen_at))
})

// Владелец dev-чата (для админа в support-inbox): данные кладутся бэком
// в поле owner_user. У собственного dev-чата сотрудника поля нет.
const devChatOwner = computed(() => active.value?.owner_user || null)

const chatOnline = computed(() => {
  if (active.value?.is_dev_chat) {
    return devChatOwner.value ? messenger.isOnline(devChatOwner.value.id) : false
  }
  return otherOnline.value
})

const ownerLastSeenText = computed(() => {
  const u = devChatOwner.value
  if (!u) return ''
  return formatLastSeen(messenger.lastSeenOf(u.id, u.last_seen_at))
})

// Чей профиль открывать по клику на шапку. У обычного диалога — собеседник.
// У dev-чата в support-inbox — владелец чата. У своего dev-чата — никого.
const profileUser = computed(() => {
  const a = active.value
  if (!a) return null
  if (a.is_dev_chat) return devChatOwner.value
  return a.other_user || null
})

// Пользовательский статус собеседника (эмодзи у имени + текст в подзаголовке).
const peerStatusEmoji = computed(() => profileUser.value?.status_emoji || '')
const peerStatusText = computed(() => profileUser.value?.status_text || '')

function avatarOf(u) {
  if (!u) return ''
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

async function selectConversation(id) {
  replyTo.value = null
  editing.value = null
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
  try {
    await messenger.send(activeId.value, payload)
    replyTo.value = null
    await nextTick()
    scrollToBottom()
  } catch (e) {
    const code = e?.error
    const msg = code === 'TASK_WRONG_COMPANY'
      ? 'Задача относится к другой компании'
      : (e?.message || 'Не удалось отправить сообщение')
    useNotificationsStore().error(msg)
  }
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

function scrollToBottomSmooth() {
  messagesEl.value?.scrollTo({ top: messagesEl.value.scrollHeight, behavior: 'smooth' })
}

// Гард, чтобы scroll-событие не запускало вторую подгрузку, пока первая ещё
// в полёте, и не падало в бесконечный «магнит» к верху, если страница
// вернулась пустой. Реактивный — на нём же висит индикатор подгрузки истории.
const loadingOlder = ref(false)

// Плавающая кнопка «к последним сообщениям» — видна, когда пользователь
// ушёл вверх по истории (паттерн Telegram/WhatsApp).
const showJumpDown = ref(false)

// Кнопка позиционируется НАД полем ввода: отступ = расстояние от низа
// chat-panel до верха инпута (учитывает reply-баннер, вложения, многострочный
// textarea и паддинг панели под мобильную навигацию) — иначе она ложится на
// кнопку отправки.
const inputClearance = ref(84)
const jumpDownBottom = computed(() => `${inputClearance.value + 12}px`)
let inputResizeObserver = null

function measureInputClearance() {
  const el = messageInputRef.value?.$el
  const panel = el?.parentElement
  if (!el || !panel) return
  inputClearance.value = Math.max(0,
    Math.round(panel.getBoundingClientRect().bottom - el.getBoundingClientRect().top))
}

watch([() => messageInputRef.value, () => active.value?.id], async () => {
  inputResizeObserver?.disconnect()
  await nextTick()
  const el = messageInputRef.value?.$el
  if (!el || !(el instanceof HTMLElement)) return
  inputResizeObserver = new ResizeObserver(measureInputClearance)
  inputResizeObserver.observe(el)
  measureInputClearance()
}, { immediate: true })

async function onScroll() {
  const el = messagesEl.value
  if (!el) return
  showJumpDown.value = el.scrollHeight - el.scrollTop - el.clientHeight > 320
  if (loadingOlder.value || jumping.value) return
  if (el.scrollTop > 80) return
  if (!messenger.hasMoreHistory(activeId.value)) return
  const arr = messenger.activeMessages
  if (!arr.length) return

  loadingOlder.value = true
  try {
    const firstId = arr[0].id
    const prevHeight = el.scrollHeight
    const prevTop = el.scrollTop
    const added = await messenger.fetchMessages(activeId.value, firstId)
    // Индикатор убираем до замера высоты, чтобы он не искажал расчёт позиции.
    loadingOlder.value = false
    if (!added || !added.length) return
    await nextTick()
    // Сохраняем визуальную позицию: пиксель, на который смотрел пользователь,
    // должен остаться на том же месте после вставки старых сообщений сверху.
    const delta = el.scrollHeight - prevHeight
    if (delta > 0) {
      el.scrollTop = prevTop + delta
    }
  } finally {
    loadingOlder.value = false
  }
}

function handleExternalOpen(e) {
  const id = e.detail?.conversation_id
  if (id) {
    selectConversation(id)
  }
}

onMounted(async () => {
  // Грузим оба списка параллельно: для рут-админа support-inbox нужен сразу
  // (бейдж непрочитанных, активация глубокой ссылки на support-чат), но
  // не должен задерживать первичный рендер обычных диалогов.
  const tasks = [messenger.fetchConversations().catch(() => {})]
  if (authStore.isSuperAdmin) {
    tasks.push(messenger.fetchSupportInbox().catch(() => {}))
  }
  await Promise.all(tasks)
  if (notificationsAllowed() === false) {
    requestNotificationPermission()
  }
  await activateRouteConversation()
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
  inputResizeObserver?.disconnect()
  // Уходим со страницы — диалог больше не «открыт», иначе входящие в него
  // продолжали бы тихо помечаться прочитанными.
  messenger.activeConversationId = null
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
  if (!id) return
  await activateRouteConversation()
})
</script>

<style scoped>
/* Каркас как у мастер-детейл разделов (Заметки/Реестры): страница на «сиянии»
   .app-layout, две стеклянные панели с зазором. */
.messenger {
  display: flex;
  gap: 16px;
  padding: 16px;
  height: 100%;
  min-height: 0;
}

.chat-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  position: relative;
  background: var(--acrylic-card-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  overflow: hidden;
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
  /* Панель уже акриловая — шапка прозрачная, без второго плотного слоя. */
  background: transparent;
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

.chat-avatar-wrap.as-btn {
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
}

.chat-avatar-wrap.dev {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}
.chat-avatar-wrap.dev .material-symbols-outlined { font-size: 22px; }

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

.chat-title.as-btn { cursor: pointer; }
.chat-title.as-btn:hover .chat-fio { color: var(--color-primary); }

.chat-fio-status {
  margin-left: 6px;
  font-size: 14px;
}

.chat-status-note {
  color: var(--color-text-dim);
  font-weight: 400;
}

.chat-status {
  font-size: 12px;
  color: var(--color-text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
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
}

.messages-area {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  background: var(--color-bg);
  min-height: 0;
}

/* Плавающая кнопка «к последним сообщениям». */
.jump-down-btn {
  position: absolute;
  right: 16px;
  bottom: 96px; /* фолбэк; фактический отступ считается от высоты поля ввода */
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--color-outline-dim);
  border-radius: 50%;
  background: var(--color-surface-high);
  color: var(--color-text);
  box-shadow: var(--shadow-md);
  cursor: pointer;
  z-index: 5;
  transition: background 0.15s;
}

.jump-down-btn:hover {
  background: color-mix(in oklch, var(--color-primary) 10%, var(--color-surface-high));
}

.jump-down-enter-active,
.jump-down-leave-active { transition: opacity 0.15s, transform 0.15s; }

.jump-down-enter-from,
.jump-down-leave-to { opacity: 0; transform: translateY(8px); }

/* Баннер закреплённых сообщений — между шапкой и лентой. */
.pinned-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 16px;
  background: var(--acrylic-card-bg);
  border-bottom: 1px solid var(--color-outline-dim);
  border-left: 3px solid var(--color-tertiary);
  cursor: pointer;
  flex-shrink: 0;
  transition: background 0.15s;
}

.pinned-bar:hover { background: var(--color-surface-low); }

.pinned-bar-icon {
  font-size: 20px;
  color: var(--color-tertiary);
  flex-shrink: 0;
  font-variation-settings: 'FILL' 1, 'wght' 500, 'GRAD' 0, 'opsz' 24;
}

.pinned-bar-body { flex: 1; min-width: 0; }

.pinned-bar-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 700;
  color: var(--color-tertiary);
}

.pinned-bar-count {
  font-weight: 600;
  color: var(--color-text-dim);
}

.pinned-bar-text {
  font-size: 13px;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.pinned-bar-unpin {
  width: 32px;
  height: 32px; min-height: 0;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: background 0.15s, color 0.15s;
}

.pinned-bar-unpin:hover {
  background: var(--color-surface-high);
  color: var(--color-text);
}

.pinned-bar-unpin .material-symbols-outlined { font-size: 18px; }

.msg-loading {
  display: flex;
  justify-content: center;
  padding: 16px;
}

/* Индикатор подгрузки старых сообщений при скролле вверх. */
.msg-loading-older {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 10px;
  font-size: 12px;
  color: var(--color-text-dim);
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
    padding: 0;
    gap: 0;
  }

  .chat-panel {
    position: fixed;
    inset: 0;
    z-index: 150;
    background: var(--color-bg);
    border: none;
    border-radius: 0;
    padding-bottom: calc(64px + env(safe-area-inset-bottom, 0px));
  }
  .messenger.mobile-chat-open .chat-panel {
    display: flex;
  }

  /* ===== Шапка активного чата ===== */
  .chat-header {
    padding: 8px 12px !important;
    gap: 10px !important;
    min-height: 56px;
    padding-top: calc(8px + env(safe-area-inset-top, 0px)) !important;
  }
  .back-btn {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    background: transparent;
    border: none;
    color: var(--color-text);
    display: grid;
    place-items: center;
    flex-shrink: 0;
    cursor: pointer;
  }
  .back-btn:active { background: var(--color-surface-high); }
  .back-btn .material-symbols-outlined { font-size: 22px; }

  .chat-avatar-wrap, .chat-avatar { width: 40px; height: 40px; }
  .chat-fio { font-size: 15px; font-weight: 700; }
  .chat-status { font-size: 11.5px; }

  .chat-tool {
    width: 40px;
    height: 40px;
  }
  .chat-tool .material-symbols-outlined { font-size: 22px; }

  /* Закреплённое сообщение — компактнее. */
  .pinned-bar {
    padding: 8px 12px;
    gap: 8px;
  }
  .pinned-bar-icon { font-size: 18px; }
  .pinned-bar-title { font-size: 11px; }
  .pinned-bar-text { font-size: 12px; }

  /* Лента сообщений — крупнее, удобнее. */
  .messages-area {
    padding: 12px 10px !important;
    gap: 4px !important;
  }

}

</style>

<!-- Вне scoped: :global() внутри scoped-блока LightningCSS компилирует в
     битый селектор. Классы .messenger/.chat-panel на элементах сохраняются
     и без атрибута скоупа — правило работает как есть. -->
<style>
@media (max-width: 768px) {
  /* Идёт работа: плашка юнита занимает верх экрана — fixed-мессенджер
     начинается под ней, иначе список рисуется поверх плашки. */
  .app-layout.has-unit-banner .messenger,
  .app-layout.has-unit-banner .messenger .chat-panel {
    top: var(--unit-banner-height, 54px);
  }
}
</style>
