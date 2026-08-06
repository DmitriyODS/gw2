<template>
  <!-- Узкую раскладку — «список ⇄ чат с возвратом» — ведёт AppListDetail по
       ширине ПАНЕЛИ, а не экрана: раздел живёт окном рабочего стола. -->
  <AppListDetail
    class="messenger"
    :list-width="listWidth"
    :narrow-at="768"
    :open="!!activeId"
    @update:open="onDetailOpen"
    @narrow-change="isNarrow = $event"
  >
    <template #list="{ narrow, toggle }">
      <ConversationList
        :narrow="narrow"
        @toggle-list="toggle"
        :conversations="visibleConversations"
        :active-id="activeId"
        :loading="listLoading"
        :show-support-tab="authStore.isSuperAdmin"
        :tab="listTab"
        :support-unread="supportTabUnread"
        @select="selectConversation"
        @new-chat="newChatOpen = true"
        @new-group="newGroupOpen = true"
        @new-call="startEmptyCall"
        @toggle-pin="onTogglePin"
        @delete="askDeleteConversation"
        @change-tab="onChangeTab"
      />
    </template>

    <template #detail="{ narrow, collapsed, toggle }">
    <!-- Переписка — панель ядра (headless: шапку чата рисуем сами, общая
         слишком высока для переписки), обои — слотом `background`. -->
    <AppPage
      embedded
      flush
      :scroll="false"
      headless
      @dragenter.prevent="onDragEnter"
      @dragover.prevent="onDragOver"
      @dragleave.prevent="onDragLeave"
      @drop.prevent="onDrop"
    >
      <template #background>
        <ChatBackgroundLayer v-if="active" :recipe="activeChatBg" />
      </template>

      <header v-if="active" class="chat-head">
        <AppButton
          v-if="isNarrow"
          variant="icon"
          size="sm"
          icon="arrow_back"
          aria-label="К списку чатов"
          title="К списку чатов"
          @click="goBack"
        />
        <AppButton
          v-else-if="collapsed"
          variant="icon"
          size="sm"
          icon="left_panel_open"
          aria-label="Показать список чатов"
          title="Показать список чатов"
          @click="toggle"
        />
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
          v-else-if="active.is_group"
          class="chat-avatar-wrap group as-btn"
          aria-label="О группе"
          @click="groupInfoOpen = true"
        >
          <img v-if="active.avatar_path" class="chat-avatar" :src="`/uploads/${active.avatar_path}`" :alt="active.title" />
          <span v-else class="material-symbols-outlined">groups</span>
        </button>
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
          :class="{ 'as-btn': !!profileUser || active.is_group }"
          :role="(profileUser || active.is_group) ? 'button' : null"
          :tabindex="(profileUser || active.is_group) ? 0 : null"
          @click="onTitleClick"
          @keydown.enter="onTitleClick"
        >
          <div class="chat-fio">
            <template v-if="active.is_dev_chat && devChatOwner">{{ devChatOwner.fio }}</template>
            <template v-else-if="active.is_dev_chat">Техподдержка</template>
            <template v-else-if="active.is_group">{{ active.title }}</template>
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
            <template v-else-if="active.is_group">
              {{ active.member_count || messenger.groupMembers(active.id).length }} участник{{ groupPlural }}
            </template>
            <template v-else-if="peerTyping">печатает…</template>
            <template v-else>
              <span v-if="peerStatusText" class="chat-status-note">{{ peerStatusText }} · </span>
              {{ otherOnline ? 'в сети' : lastSeenText }}
            </template>
          </div>
        </div>
        <AppCommandBar
          class="chat-head-commands"
          size="sm"
          :commands="chatCommands"
          @command="onChatCommand"
        />
      </header>
      <div v-if="dragOver && active" class="chat-drop-overlay">
        <span class="material-symbols-outlined">upload_file</span>
        <span>Отпустите файл — он прикрепится к сообщению</span>
      </div>

      <EmptyState
        v-if="!active"
        class="chat-empty"
        icon="chat"
        title="Выберите чат"
        subtitle="Откройте беседу слева — или начните новую из списка."
      />

      <!-- Клик по полосе прокручивает к сообщению и листает закреплённые. -->
      <AppInfoBar
        v-if="active && pinnedMessages.length"
        class="pinned-bar"
        icon="keep"
        :title="pinnedMessages.length > 1 ? `Закреплённое ${pinnedIndex + 1}/${pinnedMessages.length}` : 'Закреплённое'"
        :message="pinnedPreview"
        @click="cyclePinned"
      >
        <template #actions>
          <AppButton
            variant="icon"
            size="sm"
            icon="close"
            aria-label="Открепить"
            title="Открепить"
            @click.stop="unpinMessage(currentPinned)"
          />
        </template>
      </AppInfoBar>

      <div
        v-if="active"
        ref="messagesEl"
        class="messages-area"
        @scroll="onScroll"
      >
        <div v-if="messenger.loadingMessages && !messenger.activeMessages.length" class="msg-loading">
          <BrandLoader :size="64" />
        </div>
        <template v-else>
          <div v-if="loadingOlder" class="msg-loading-older">
            <ProgressSpinner style="width:22px;height:22px" />
            <span>Загружаем историю…</span>
          </div>
          <div v-for="g in messageGroups" :key="g.key" class="msg-day-group">
            <MessageDateDivider :label="g.label" @jump="jumpToDay" />
            <MessageBubble
              v-for="m in g.items"
              :key="m.id"
              v-memo="[m.id, m.text, m.edited_at, m.read_at, m.pinned_at, m.reactions, m.call?.status, authStore.user?.id, active?.is_dev_chat, active?.is_group, groupReadCount(m)]"
              :message="m"
              :is-mine="m.sender_id === authStore.user?.id"
              :sender-name="senderNameFor(m)"
              :me-id="authStore.user?.id"
              :is-group="!!active?.is_group"
              :read-count="groupReadCount(m)"
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
              @read-by="openReadBy"
            />
          </div>
        </template>
      </div>

      <Transition name="jump-down">
        <AppButton
          v-if="active && showJumpDown"
          class="jump-down-btn"
          variant="icon"
          icon="keyboard_arrow_down"
          :style="{ bottom: jumpDownBottom }"
          aria-label="К последним сообщениям"
          @click="scrollToBottomSmooth"
        />
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
    </AppPage>
    </template>

    <AppFab
      :visible="isNarrow && !activeId && listTab === 'chats'"
      icon="edit_square"
      tone="tertiary"
      aria-label="Новый чат"
      @click="newChatOpen = true"
    />

    <NewChatDialog v-model="newChatOpen" @pick="startWith" />
    <NewGroupDialog v-model="newGroupOpen" @created="selectConversation" />
    <GroupInfoDialog
      v-if="active?.is_group"
      v-model="groupInfoOpen"
      :conversation-id="active.id"
      @left="onLeftGroup"
    />
    <MessageReadByDialog v-model="readByOpen" :message-id="readByMessageId" />
    <ChatBackgroundDialog v-model="chatBgOpen" :conversation="active" />

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
  </AppListDetail>
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
import { groupMessagesByDay } from '@/utils/chatDates.js'
import ConversationList from '@/components/messenger/ConversationList.vue'
import MessageBubble from '@/components/messenger/MessageBubble.vue'
import MessageDateDivider from '@/components/messenger/MessageDateDivider.vue'
import MessageInput from '@/components/messenger/MessageInput.vue'
import NewChatDialog from '@/components/messenger/NewChatDialog.vue'
import NewGroupDialog from '@/components/messenger/NewGroupDialog.vue'
import GroupInfoDialog from '@/components/messenger/GroupInfoDialog.vue'
import MessageReadByDialog from '@/components/messenger/MessageReadByDialog.vue'
import ChatBackgroundLayer from '@/components/common/ChatBackgroundLayer.vue'
import ChatBackgroundDialog from '@/components/messenger/ChatBackgroundDialog.vue'
import DeleteScopeDialog from '@/components/messenger/DeleteScopeDialog.vue'
import ForwardDialog from '@/components/messenger/ForwardDialog.vue'
import AttachTaskDialog from '@/components/messenger/AttachTaskDialog.vue'
import MessageContextMenu from '@/components/messenger/MessageContextMenu.vue'
import EmployeeProfileDialog from '@/components/common/EmployeeProfileDialog.vue'
import AppFab from '@/components/ui/AppFab.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppCommandBar from '@/components/ui/AppCommandBar.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AppListDetail from '@/components/ui/AppListDetail.vue'
import AppPage from '@/components/ui/AppPage.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ProgressSpinner from 'primevue/progressspinner'
import BrandLoader from '@/components/common/BrandLoader.vue'

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

/* Узкая раскладка = колонки показываются по одной. Считает её AppListDetail по
   ширине ПАНЕЛИ: раздел живёт окном рабочего стола, и в узком окне список с
   чатом рядом не помещаются так же, как на телефоне. isMobile тут не годится —
   он про ширину экрана (её проверяют только вещи про мобильную клавиатуру). */
const isNarrow = ref(false)


const newChatOpen = ref(false)
const newGroupOpen = ref(false)
const groupInfoOpen = ref(false)
const chatBgOpen = ref(false)
const readByOpen = ref(false)
const readByMessageId = ref(null)
const attachTaskOpen = ref(false)
const attachedTask = ref(null)
const profileOpen = ref(false)
const ctxMenu = ref({ visible: false, x: 0, y: 0, message: null })
const messagesEl = ref(null)

/* Mobile keyboard: прокрутка ленты остаётся заякоренной У НИЗА — как в
   Telegram: при появлении клавиатуры верх сообщений обрезается, а расстояние до
   низа сохраняется, так что сообщение, на которое смотрел пользователь, не
   уезжает.

   Саму высоту панели подгонять больше не нужно: раздел живёт в области, которую
   отвёл мобильный каркас, и при adjustResize она ужимается вместе с окном.
   Значения вьюпорта храним только чтобы отличать реальное изменение от шума. */
const viewportHeight = ref(0) // px; 0 — не мобилка / ещё не измеряли
const viewportTop = ref(0)
function onViewportChange() {
  if (!window.visualViewport || !isMobile.value) return
  const vv = window.visualViewport
  if (vv.height === viewportHeight.value && vv.offsetTop === viewportTop.value) return

  const el = messagesEl.value
  const fromBottom = el ? el.scrollHeight - el.scrollTop - el.clientHeight : 0
  viewportHeight.value = vv.height
  viewportTop.value = vv.offsetTop
  if (el) {
    nextTick(() => {
      // Чтение clientHeight форсирует reflow — новая высота уже учтена.
      el.scrollTop = el.scrollHeight - el.clientHeight - fromBottom
    })
  }
}

const messageInputRef = ref(null)
const messageGroups = computed(() => groupMessagesByDay(messenger.activeMessages))
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
  // В группе входящие подписываем именем автора.
  if (active.value?.is_group && m.sender_id && m.sender_id !== authStore.user?.id) {
    const mem = messenger.groupMembers(active.value.id).find((x) => x.user?.id === m.sender_id)
    return mem?.user?.fio || ''
  }
  return ''
}

// Сколько ДРУГИХ участников прочитали моё групповое сообщение.
function groupReadCount(m) {
  if (!active.value?.is_group || m.sender_id !== authStore.user?.id) return 0
  return messenger.readCountForMessage(active.value.id, m)
}

function openReadBy(messageId) {
  readByMessageId.value = messageId
  readByOpen.value = true
}

/* Команды переписки. Звонки — кнопками (в мессенджере это главное действие),
   остальное AppCommandBar убирает в меню «ещё»; своё выпадающее меню чата
   вместе с обработчиком клика «вне» больше не нужно. */
const chatCommands = computed(() => {
  const c = active.value
  if (!c) return []
  const dialog = !c.is_dev_chat && !c.is_group
  return [
    { key: 'audio', label: 'Аудиозвонок', icon: 'call', variant: 'icon', primary: true, hidden: !dialog },
    { key: 'video', label: 'Видеозвонок', icon: 'videocam', variant: 'icon', primary: true, hidden: !dialog },
    { key: 'group-info', label: 'О группе', icon: 'info', hidden: !c.is_group },
    {
      key: 'mute',
      label: c.muted ? 'Включить уведомления' : 'Выключить уведомления',
      icon: c.muted ? 'notifications' : 'notifications_off',
      hidden: !c.is_group,
    },
    { key: 'pin', label: c.is_pinned ? 'Открепить чат' : 'Закрепить чат', icon: c.is_pinned ? 'keep_off' : 'keep' },
    { key: 'background', label: 'Оформление чата', icon: 'palette' },
    { key: 'leave', label: 'Выйти из группы', icon: 'logout', danger: true, hidden: !c.is_group },
    { key: 'delete', label: 'Удалить чат', icon: 'delete', danger: true, hidden: c.is_dev_chat || c.is_group },
  ]
})

function onChatCommand(key) {
  const c = active.value
  if (!c) return
  if (key === 'audio') startCall('audio')
  else if (key === 'video') startCall('video')
  else if (key === 'group-info' || key === 'leave') groupInfoOpen.value = true
  else if (key === 'mute') messenger.setGroupMuteAction(c.id, !c.muted)
  else if (key === 'pin') onTogglePin(c.id)
  else if (key === 'background') chatBgOpen.value = true
  else if (key === 'delete') askDeleteConversation(c)
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
// Контент из системного «Поделиться» (Android): открыли выбранный чат — сеем
// текст в поле и грузим файлы во вложения один раз (останется отправить).
// Следим И за pendingDraft: openWith выставляет activeConversationId РАНЬШЕ,
// чем App.vue кладёт draft, поэтому по одному activeConversationId черновик мог
// не застать (файлы не прикреплялись). Реагируем на любое из двух изменений.
watch([() => messenger.activeConversationId, () => messenger.pendingDraft], async ([id]) => {
  const d = messenger.pendingDraft
  if (!id || !d || d.convId !== id) return
  messenger.pendingDraft = null
  await nextTick()
  if (d.text) messageInputRef.value?.setText(d.text)
  if (d.files?.length) messageInputRef.value?.addFiles(d.files)
}, { immediate: true })

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
  // В группе удаление всегда «для всех» (личного скрытия нет).
  if (active.value?.is_group) {
    deleteDialog.value = {
      title: 'Удалить сообщение?',
      text: 'Сообщение будет удалено у всех участников группы.',
      canForAll: false,
      otherName: '',
      payload: { kind: 'message', id: message.id },
    }
    deleteDialogOpen.value = true
    return
  }
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

// Переход к сообщению с подсветкой и догрузкой истории.
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
const activeChatBg = computed(() => messenger.resolveChatBg(activeId.value))

// Вышли из группы или распустили её — закрываем карточку и уходим из чата.
function onLeftGroup() {
  groupInfoOpen.value = false
  router.replace('/messenger')
}

// Клик по шапке: у группы — карточка группы, у 1:1 — профиль собеседника.
function onTitleClick() {
  if (active.value?.is_group) groupInfoOpen.value = true
  else if (profileUser.value) profileOpen.value = true
}

const groupPlural = computed(() => {
  const n = active.value?.member_count || messenger.groupMembers(active.value?.id).length || 0
  const d = n % 10, dd = n % 100
  if (d === 1 && dd !== 11) return ''
  if (d >= 2 && d <= 4 && (dd < 10 || dd >= 20)) return 'а'
  return 'ов'
})

// Активная вкладка списка слева: 'chats' | 'support'. Вторая доступна
// только Администратору системы (он отвечает в чужие чаты техподдержки).
const listTab = ref('chats')
/* Ширина колонки списка. С рейлом папок панель шире ровно на него — условие то
   же, что внутри ConversationList (вкладка «Чаты» и папки заведены): ширину
   колонки задаёт каркас, поэтому знать о рейле приходится здесь. */
const listWidth = computed(() =>
  listTab.value === 'chats' && messenger.folders.length ? 416 : 340,
)

/* Для рут-админа техподдержка — отдельный inbox (входящие из чужих компаний),
   отображается на собственной вкладке. У обычных пользователей dev-чат с
   техподдержкой — это обычный диалог в общем списке, без отдельной вкладки. */
const visibleConversations = computed(() => {
  if (authStore.isSuperAdmin && listTab.value === 'support') {
    return messenger.supportInbox
  }
  // На вкладке «Чаты» дополнительно фильтруем по активной папке (null — все).
  return messenger.conversationsInFolder(messenger.activeFolderId)
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
  /* replace, а не push: список и переписка — два состояния ОДНОГО раздела, и
     возврат к списку уже нарисован стрелкой в шапке чата. С push окно копило
     свою историю и рисовало ВТОРУЮ стрелку «назад» в рамке — две кнопки об
     одном и том же. Историю браузера (системный жест «назад» на телефоне)
     ведёт каркас по адресу активного раздела — она от этого не страдает. */
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
    // Очищаем поле ТОЛЬКО после успешной отправки — при сбое текст остаётся.
    messageInputRef.value?.clearAfterSend()
    replyTo.value = null
    await nextTick()
    scrollToBottom()
  } catch (e) {
    // Поле не очищаем: пользователь может повторить отправку тем же текстом.
    const code = e?.error
    const msg = code === 'TASK_WRONG_COMPANY'
      ? 'Задача относится к другой компании'
      : (e?.message || 'Не удалось отправить сообщение')
    useNotificationsStore().error(msg)
  }
}

/* Узкая раскладка закрыла чат («назад» каркаса или переход к списку) —
   снимаем активный диалог, чтобы список снова стал главным экраном. */
function onDetailOpen(open) {
  if (!open) goBack()
}

/* Возврат к списку — обычный переход на адрес раздела; активную переписку
   снимает watch по адресу (единственный источник правды), поэтому «назад»
   работает одинаково откуда бы ни пришло: стрелка чата, рамка окна, системный
   жест. */
function goBack() {
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

// Клик по прилипшей плашке даты — прокрутка к началу этого дня (первому его
// сообщению). Считаем по rect первого пузыря группы (сам разделитель sticky,
// его позиция «прилипла» и для расчёта непригодна); оставляем ~44px сверху,
// чтобы пилюля даты осталась видимой над первым сообщением.
function jumpToDay(dividerEl) {
  const el = messagesEl.value
  const first = dividerEl?.nextElementSibling
  if (!el || !first) return
  const top = el.scrollTop + (first.getBoundingClientRect().top - el.getBoundingClientRect().top) - 44
  el.scrollTo({ top: Math.max(0, top), behavior: 'smooth' })
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
  const tasks = [
    messenger.fetchConversations().catch(() => {}),
    messenger.fetchFolders().catch(() => {}),
  ]
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
  window.visualViewport?.addEventListener('resize', onViewportChange)
  window.visualViewport?.addEventListener('scroll', onViewportChange)
})

onBeforeUnmount(() => {
  window.removeEventListener('messenger:open-conversation', handleExternalOpen)
  inputResizeObserver?.disconnect()
  window.visualViewport?.removeEventListener('resize', onViewportChange)
  window.visualViewport?.removeEventListener('scroll', onViewportChange)
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

/* Адрес — источник правды об открытой переписке: ушёл id (стрелка чата, «назад»
   в рамке окна, системный жест) — закрываем чат и показываем список. Раньше
   пустой id молча игнорировался, и стрелка окна выглядела нерабочей. */
watch(() => route.params.conversationId, async (id) => {
  if (!id) {
    messenger.activeConversationId = null
    return
  }
  await activateRouteConversation()
})
</script>

<style scoped>
/* Каркас — AppListDetail (как у Заметок и Реестров): две колонки с зазором,
   узкая раскладка «список ⇄ чат». Панели-стёкла рисуют сами колонки:
   ConversationList — свою, .chat-panel — свою. */
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

/* Шапка переписки — своя (режим headless): плотная строка, а не заголовок
   раздела. Матовость даёт полупрозрачность поверх обоев чата; собственного
   backdrop-filter НЕТ — иначе шапка станет backdrop-root и погасит акрил
   выпадающих меню под ней. */
.chat-head {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
  min-width: 0;
  padding: 7px 10px;
  border-bottom: 1px solid var(--color-outline-dim);
  background: var(--acrylic-card-bg);
}
.chat-head-commands { margin-left: auto; }

/* Полоса закреплённого кликается целиком (листает закреплённые). */
.pinned-bar { cursor: pointer; margin: 0 12px 8px; }

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
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}
.chat-avatar-wrap.dev .material-symbols-outlined { font-size: 22px; }

.chat-avatar-wrap.group {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  overflow: hidden;
  border: none;
  cursor: pointer;
}
.chat-avatar-wrap.group .material-symbols-outlined { font-size: 24px; font-variation-settings: 'FILL' 1; }
.chat-avatar-wrap.group .chat-avatar { width: 100%; height: 100%; }

.chat-avatar {
  width: 36px;
  height: 36px;
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


.chat-status-note {
  color: var(--color-text-dim);
  font-weight: 400;
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
  /* Фон даёт слой .chat-bg (градиент/узор) под лентой; сама лента прозрачна. */
  background: transparent;
  min-height: 0;
}

/* Плавающая кнопка «к последним сообщениям». */
/* Кнопка «к последним сообщениям» — обычная иконочная кнопка ядра, здесь
   только её место над полем ввода (фактический отступ считает jumpDownBottom). */
.jump-down-btn {
  position: absolute;
  right: 16px;
  bottom: 96px;
  z-index: 5;
  box-shadow: var(--shadow-md);
}

.jump-down-enter-active,
.jump-down-leave-active { transition: opacity 0.15s, transform 0.15s; }

.jump-down-enter-from,
.jump-down-leave-to { opacity: 0; transform: translateY(8px); }

/* Баннер закреплённых сообщений — между шапкой и лентой. */
.messages-area {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  /* Фон даёт слой .chat-bg (градиент/узор) под лентой; сама лента прозрачна. */
  background: transparent;
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

@media (max-width: 768px) {
  /* Раздел НЕ вырывается из потока: место между панелью статусов и панелью
     задач ему уже отвёл мобильный каркас (AppScreen). Прежний
     `position: fixed; inset: 0` — наследие времён до каркаса-«ОС»: он лез под
     панель статусов (шапка чата обрезалась) и резервировал место под нижнюю
     навигацию, которой давно нет. Остаётся только полноэкранный вид панели —
     без рамки и скруглений, как у остальных разделов на телефоне. */
  .chat-panel {
    background: var(--color-bg);
    border: none;
    border-radius: 0;
  }
  /* ===== Шапка активного чата ===== */
  /* Системный вырез отбирает сам каркас (AppScreen) — компенсировать его тут
     значило бы отступать дважды. */
  .chat-header {
    padding: 8px 12px !important;
    gap: 10px !important;
    min-height: 56px;
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
