<template>
  <div v-if="!hidden" class="mini-mess">
    <!-- Панель -->
    <transition name="mini-pop">
      <div v-if="open" class="mini-panel" :style="panelStyle">
        <!-- Режим: вкладки хаба (Ассистент / список диалогов) -->
        <template v-if="!(activeTab === 'messages' && threadId)">
          <header class="mini-head">
            <SegmentedTabs v-model="activeTab" :tabs="hubTabs" dense />
            <button class="mini-icon" title="Свернуть" aria-label="Свернуть" @click="open = false">
              <span class="material-symbols-outlined">close</span>
            </button>
          </header>

          <!-- Вкладка «Ассистент» -->
          <template v-if="activeTab === 'assistant'">
            <div v-if="assistantStore.unavailable" class="mini-empty">
              <span class="material-symbols-outlined">smart_toy</span>
              <p>Ассистент доступен только внутри компании</p>
            </div>
            <template v-else>
              <div ref="assistantThreadEl" class="mini-thread assistant-thread">
                <div v-if="assistantStore.loading && !assistantStore.messages.length" class="mini-loading">
                  <ProgressSpinner style="width:28px;height:28px" />
                </div>
                <div v-else-if="!assistantStore.messages.length" class="mini-empty">
                  <span class="material-symbols-outlined">smart_toy</span>
                  <p>Задайте вопрос о задачах или статистике компании</p>
                  <!-- Подсказки в пустом чате — учат диапазону возможностей
                       ассистента одним взглядом (паттерн Notion AI/Copilot). -->
                  <div v-if="!assistantStore.disabled" class="assistant-chips">
                    <button
                      v-for="s in ASSISTANT_SUGGESTIONS"
                      :key="s"
                      class="assistant-chip"
                      type="button"
                      @click="onAssistantSend(s)"
                    >{{ s }}</button>
                  </div>
                </div>
                <div v-if="assistantStore.disabled" class="assistant-note">
                  ИИ выключен для компании — обратитесь к администратору.
                </div>
                <div
                  v-for="m in assistantStore.messages"
                  :key="m.id"
                  class="assistant-row"
                  :class="{ outgoing: m.role === 'user' }"
                >
                  <div class="assistant-msg">
                    <div class="assistant-bubble">
                      <!-- Ответы ассистента приходят в Markdown (LLM), реплики
                           пользователя — простой текст с линкификацией. -->
                      <MarkdownView v-if="m.role === 'assistant'" :source="m.text" />
                      <LinkifiedText v-else :text="m.text" />
                    </div>
                    <div v-if="m.role === 'assistant' && m.sources" class="assistant-sources">{{ m.sources }}</div>
                    <div v-if="canRate(m)" class="assistant-feedback">
                      <button
                        class="assistant-fb-btn"
                        :class="{ active: assistantStore.myFeedback[m.id] === 'up' }"
                        type="button"
                        aria-label="Полезный ответ"
                        title="Полезный ответ"
                        @click="onFeedbackUp(m)"
                      >
                        <span class="material-symbols-outlined">thumb_up</span>
                      </button>
                      <button
                        class="assistant-fb-btn"
                        :class="{ active: assistantStore.myFeedback[m.id] === 'down' }"
                        type="button"
                        aria-label="Неудачный ответ"
                        title="Неудачный ответ"
                        @click="onFeedbackDown(m)"
                      >
                        <span class="material-symbols-outlined">thumb_down</span>
                      </button>
                      <template v-if="feedbackReasonFor === m.id">
                        <button
                          v-for="r in FEEDBACK_REASONS"
                          :key="r.value"
                          class="assistant-fb-chip"
                          type="button"
                          @click="onFeedbackReason(m, r.value)"
                        >{{ r.label }}</button>
                      </template>
                    </div>
                  </div>
                </div>
                <div v-if="assistantStore.sending" class="assistant-row">
                  <div class="assistant-bubble assistant-typing">
                    <span class="ai-dot"></span><span class="ai-dot"></span><span class="ai-dot"></span>
                  </div>
                </div>
                <div v-if="assistantStore.error" class="assistant-error">{{ assistantStore.error }}</div>
              </div>
              <AssistantInput
                ref="assistantInputRef"
                :sending="assistantStore.sending"
                :disabled="assistantStore.disabled"
                @send="onAssistantSend"
              />
            </template>
          </template>

          <!-- Вкладка «Сообщения»: список диалогов -->
          <template v-else>
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
                <div v-if="c.is_dev_chat" class="mini-avatar-wrap mini-avatar-wrap--dev">
                  <span class="material-symbols-outlined">support_agent</span>
                </div>
                <div v-else class="mini-avatar-wrap">
                  <img class="mini-avatar" :src="avatarOf(c.other_user)" :alt="c.other_user?.fio" />
                  <span v-if="messenger.isOnline(c.other_user?.id)" class="online-dot mini-list-dot" title="В сети"></span>
                </div>
                <div class="mini-conv-body">
                  <div class="mini-conv-name">
                    <template v-if="c.is_dev_chat">Техподдержка</template>
                    <template v-else>{{ c.other_user?.fio }}</template>
                  </div>
                  <div class="mini-conv-preview">{{ preview(c.last_message) }}</div>
                </div>
                <span v-if="c.unread_count" class="mini-badge">{{ c.unread_count }}</span>
              </li>
            </ul>
          </template>
        </template>

        <!-- Режим: переписка -->
        <template v-else>
          <header class="mini-head">
            <button class="mini-icon" title="Назад" aria-label="Назад" @click="closeThread">
              <span class="material-symbols-outlined">arrow_back</span>
            </button>
            <div v-if="threadConv?.is_dev_chat" class="mini-head-avatar-wrap mini-head-avatar-wrap--dev">
              <span class="material-symbols-outlined">support_agent</span>
            </div>
            <div v-else class="mini-head-avatar-wrap">
              <img class="mini-head-avatar" :src="avatarOf(threadConv?.other_user)" :alt="threadConv?.other_user?.fio" />
              <span v-if="threadOnline" class="online-dot mini-head-dot" title="В сети"></span>
            </div>
            <div class="mini-head-title">
              <span class="mini-title--name">
                <template v-if="threadConv?.is_dev_chat">Техподдержка</template>
                <template v-else>{{ threadConv?.other_user?.fio }}</template>
              </span>
              <span class="mini-head-status" :class="{ online: threadOnline }">
                <template v-if="threadConv?.is_dev_chat">Поддержка Groove Work</template>
                <template v-else>{{ threadOnline ? 'в сети' : threadLastSeen }}</template>
              </span>
            </div>
            <button class="mini-icon" title="Свернуть" aria-label="Свернуть" @click="open = false">
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
              :show-pin="false"
              @reply="startReply"
              @forward="startForward"
              @delete="askDeleteMessage"
              @context-menu="openContextMenu"
              @join-call="onJoinCall"
              @open-task="openTask"
              @open-post="openPost"
              @quote-click="onQuoteClick"
            />
          </div>
          <MessageInput
            ref="miniInputRef"
            :sending="messenger.sending"
            :reply-to="replyTo"
            v-model:attached-task="attachedTask"
            placeholder="Сообщение…"
            @send="onSend"
            @cancel-reply="replyTo = null"
            @attach-task="attachTaskOpen = true"
          />
        </template>
      </div>
    </transition>

    <ForwardDialog
      ref="forwardDialogRef"
      v-model="forwardOpen"
      :message="forwardSource"
      mask-class="above-mini-mess"
      dialog-class="above-mini-mess"
      @confirm="onForwardConfirm"
    />

    <DeleteScopeDialog
      v-model="deleteDialogOpen"
      :title="deleteDialog.title"
      :text="deleteDialog.text"
      :can-for-all="deleteDialog.canForAll"
      :other-name="deleteDialog.otherName"
      mask-class="above-mini-mess"
      dialog-class="above-mini-mess"
      @confirm="onDeleteConfirm"
    />

    <AttachTaskDialog
      v-model="attachTaskOpen"
      :company-id="threadConv?.company_id ?? null"
      mask-class="above-mini-mess"
      dialog-class="above-mini-mess"
      @pick="onPickTask"
    />

    <MessageContextMenu
      :visible="ctxMenu.visible"
      :x="ctxMenu.x"
      :y="ctxMenu.y"
      :is-pinned="!!ctxMenu.message?.pinned_at"
      :show-pin="false"
      :show-forward="!threadConv?.is_dev_chat"
      :show-copy="!!ctxMenu.message?.text"
      :show-delete="true"
      @close="ctxMenu.visible = false"
      @action="onCtxAction"
    />

    <!-- Кнопка-FAB (прячется, пока открыт любой AppDialog — на мобильном
         bottom sheet живёт в той же нижней зоне экрана) -->
    <button
      v-show="!anyModalOpen || open"
      class="mini-fab"
      :class="{ active: open }"
      @click="toggle"
      :title="fabTitle"
      :aria-label="fabTitle"
    >
      <span class="material-symbols-outlined">{{ fabIcon }}</span>
      <span v-if="!open && messenger.totalUnread" class="mini-fab-badge">
        {{ messenger.totalUnread > 99 ? '99+' : messenger.totalUnread }}
      </span>
    </button>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { anyModalOpen } from '@/composables/useOpenModals.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { useAuthStore } from '@/stores/auth.js'
import { useCallStore } from '@/stores/call.js'
import { useAssistantStore } from '@/stores/assistant.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useJumpToMessage } from '@/composables/useJumpToMessage.js'
import { formatLastSeen } from '@/utils/presence.js'
import MessageBubble from './MessageBubble.vue'
import MessageInput from './MessageInput.vue'
import AssistantInput from './AssistantInput.vue'
import ForwardDialog from './ForwardDialog.vue'
import DeleteScopeDialog from './DeleteScopeDialog.vue'
import AttachTaskDialog from './AttachTaskDialog.vue'
import MessageContextMenu from './MessageContextMenu.vue'
import SegmentedTabs from '@/components/common/SegmentedTabs.vue'
import LinkifiedText from '@/components/common/LinkifiedText.vue'
import MarkdownView from '@/components/common/MarkdownView.vue'
import ProgressSpinner from 'primevue/progressspinner'

const route = useRoute()
const router = useRouter()
const messenger = useMessengerStore()
const authStore = useAuthStore()
const callStore = useCallStore()
const assistantStore = useAssistantStore()
const notif = useNotificationsStore()

async function onJoinCall(callInfo) {
  await callStore.joinExistingCall(callInfo)
  open.value = false
}

// Клик по плашке прикреплённой задачи: открываем её карточку на /tasks и
// сворачиваем мини-чат, чтобы он не перекрывал модалку.
function openTask(taskId) {
  router.push({ path: '/tasks', query: { open: taskId } })
  open.value = false
}

// Клик по плашке пересланного поста портала — переходим на его страницу и
// сворачиваем мини-чат, как и при переходе к задаче.
function openPost(postId) {
  router.push(`/portal/${postId}`)
  open.value = false
}

const open = ref(false)
const threadId = ref(null)
const replyTo = ref(null)
const attachedTask = ref(null)
const attachTaskOpen = ref(false)
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

// Ассистенту нужен постоянный доступ (везде, включая мобильные и
// /messenger) — хаб скрываем только на fullscreen-роутах (ТВ, звонок по
// ссылке-приглашению) и во время активного полноэкранного звонка (сам
// CallView тоже перекрывает экран, z-index 11500 > 10050 — но явно прячем
// FAB и на случай его мини-режима смены).
const hidden = computed(() => {
  if (route.meta?.fullscreen) return true
  return callStore.isInCall && !callStore.isMinimized
})

/* ── Вкладки хаба: «Ассистент» (дефолт) / «Сообщения» ──────── */
const TAB_STORAGE_KEY = 'gw_assistant_hub_tab'
function loadStoredTab() {
  try {
    return localStorage.getItem(TAB_STORAGE_KEY) === 'messages' ? 'messages' : 'assistant'
  } catch {
    return 'assistant'
  }
}
const activeTab = ref(loadStoredTab())

const hubTabs = computed(() => [
  { value: 'assistant', label: 'Ассистент', icon: 'smart_toy' },
  {
    value: 'messages',
    label: 'Сообщения',
    icon: 'forum',
    badge: messenger.totalUnread ? (messenger.totalUnread > 99 ? '99+' : messenger.totalUnread) : null,
  },
])

// Иконка/подсказка FAB отражают вкладку, на которую попадёт клик открытия.
const fabIcon = computed(() => {
  if (open.value) return 'close'
  return activeTab.value === 'messages' ? 'chat' : 'smart_toy'
})
const fabTitle = computed(() => {
  if (open.value) return 'Свернуть'
  return activeTab.value === 'messages' ? 'Открыть чаты' : 'Открыть ассистента'
})

watch(activeTab, (tab) => {
  try { localStorage.setItem(TAB_STORAGE_KEY, tab) } catch { /* приватный режим */ }
  if (!open.value) return
  if (tab === 'assistant') ensureAssistantHistory()
  else ensureMessagesFresh()
})

const threadConv = computed(() =>
  messenger.conversationById.get(threadId.value) || null
)

const threadOnline = computed(() => messenger.isOnline(threadConv.value?.other_user?.id))
const threadLastSeen = computed(() => {
  const u = threadConv.value?.other_user
  if (!u) return ''
  return formatLastSeen(messenger.lastSeenOf(u.id, u.last_seen_at))
})

function ensureMessagesFresh() {
  if (!messenger.conversations.length) messenger.fetchConversations()
  // Свежий снимок онлайн-статусов при открытии.
  messenger.fetchPresence()
  // Вернулись к открытому ранее треду — он снова «активен».
  if (threadId.value) messenger.setActive(threadId.value)
}

function ensureAssistantHistory() {
  if (!assistantStore.loaded && !assistantStore.loading) assistantStore.fetchHistory()
}

// Подсказки в пустом чате ассистента — по одной на каждый класс инструментов
// (статистика, лидеры, отделы, поиск задач).
const ASSISTANT_SUGGESTIONS = [
  'Сводка по компании за эту неделю',
  'Кто закрыл больше всего задач за месяц?',
  'Сколько часов по отделам за неделю?',
  'Найди задачу про отчёт',
]

// Мобильная панель — во весь экран; при открытой клавиатуре iOS поджимаем
// высоту под visualViewport (fixed-элементы клавиатура не двигает).
const panelStyle = ref({})
function updatePanelViewport() {
  const vv = typeof window !== 'undefined' ? window.visualViewport : null
  if (!open.value || !vv || window.innerWidth > 768) {
    panelStyle.value = {}
    return
  }
  panelStyle.value = vv.height < window.innerHeight - 60
    ? { height: `${Math.round(vv.height)}px` }
    : {}
}
watch(open, () => nextTick(updatePanelViewport))
onMounted(() => {
  window.visualViewport?.addEventListener('resize', updatePanelViewport)
})
onBeforeUnmount(() => {
  window.visualViewport?.removeEventListener('resize', updatePanelViewport)
})

function toggle() {
  open.value = !open.value
  if (open.value) {
    if (activeTab.value === 'messages') ensureMessagesFresh()
    else ensureAssistantHistory()
  } else if (threadId.value) {
    // Панель закрыта — тред больше не виден, входящие в него не должны
    // тихо помечаться прочитанными.
    messenger.activeConversationId = null
  }
}

async function openThread(id) {
  threadId.value = id
  attachedTask.value = null
  await messenger.setActive(id)
  await nextTick()
  scrollBottom()
}

function closeThread() {
  threadId.value = null
  replyTo.value = null
  attachedTask.value = null
  // Снимаем «активность», чтобы входящие в этот чат снова считались
  // непрочитанными, пока мы на него не смотрим.
  messenger.activeConversationId = null
}

// Переход к процитированному сообщению — та же логика, что в MessengerView.
const { jumpToMessage } = useJumpToMessage({
  container: threadEl,
  getMessages: () => messenger.activeMessages,
  hasMore: () => messenger.hasMoreHistory(threadId.value),
  loadOlder: (beforeId) => messenger.fetchMessages(threadId.value, beforeId),
})

async function onQuoteClick(id) {
  if (!await jumpToMessage(id)) {
    notif.warn('Сообщение не найдено')
  }
}

function startReply(message) {
  replyTo.value = {
    id: message.id,
    sender_fio: message.sender_id === authStore.user?.id
      ? 'Вы'
      : (threadConv.value?.other_user?.fio || ''),
    text: message.text,
    kind: message.kind,
    has_attachments: !!message.attachments?.length,
  }
  // Сразу в поле ввода — можно писать ответ без лишнего клика.
  nextTick(() => miniInputRef.value?.focus())
}

/* ── Пересылка ─────────────────────────────────────────────── */
const forwardOpen = ref(false)
const forwardSource = ref(null)
const forwardDialogRef = ref(null)

function startForward(message) {
  forwardSource.value = message
  forwardOpen.value = true
}

async function onForwardConfirm({ userIds }) {
  try {
    await messenger.forwardMessage(forwardSource.value.id, { userIds })
    notif.success(userIds.length > 1 ? 'Сообщение переслано' : 'Сообщение переслано')
  } catch (e) {
    console.error('forward failed', e)
    notif.error(e?.message || 'Не удалось переслать сообщение')
  } finally {
    forwardDialogRef.value?.stopSending()
    forwardOpen.value = false
    forwardSource.value = null
  }
}

/* ── Удаление сообщения ────────────────────────────────────── */
const deleteDialogOpen = ref(false)
const deleteDialog = ref({
  title: '',
  text: '',
  canForAll: true,
  otherName: '',
  payload: null,
})

function askDeleteMessage(message) {
  const isMine = message.sender_id === authStore.user?.id
  const other = threadConv.value?.other_user?.fio || ''
  deleteDialog.value = {
    title: 'Удалить сообщение?',
    text: isMine
      ? 'Сообщение исчезнет у вас. Можно также удалить его у собеседника.'
      : 'Сообщение скроется только у вас — у собеседника останется.',
    canForAll: isMine,
    otherName: other,
    payload: { id: message.id },
  }
  deleteDialogOpen.value = true
}

async function onDeleteConfirm({ scope }) {
  const p = deleteDialog.value.payload
  if (!p) return
  try {
    await messenger.deleteMessage(p.id, scope)
  } catch (e) {
    console.error('delete failed', e)
  }
}

/* ── Контекстное меню (ПКМ / long-press) ───────────────────── */
const ctxMenu = ref({ visible: false, x: 0, y: 0, message: null })

function openContextMenu({ x, y, message }) {
  ctxMenu.value = { visible: true, x, y, message }
}

function onCtxAction(action) {
  const m = ctxMenu.value.message
  if (!m) return
  if (action === 'reply') startReply(m)
  else if (action === 'forward') startForward(m)
  else if (action === 'delete') askDeleteMessage(m)
  else if (action === 'copy') copyMessageText(m)
}

function copyMessageText(m) {
  if (!m?.text) return
  if (navigator.clipboard?.writeText) {
    navigator.clipboard.writeText(m.text).catch(() => {})
  }
}

function onPickTask(task) { attachedTask.value = task }

async function onSend(payload) {
  try {
    await messenger.send(threadId.value, payload)
  } catch (e) {
    const msg = e?.error === 'TASK_WRONG_COMPANY'
      ? 'Задача относится к другой компании'
      : (e?.message || 'Не удалось отправить сообщение')
    notif.error(msg)
    return
  }
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
  if (msg.kind === 'call') {
    return msg.call?.media === 'audio' ? '📞 Аудиозвонок' : '📹 Видеозвонок'
  }
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

// Открытие чата из системного уведомления — разворачиваем мини-чат на
// вкладке «Сообщения», даже если сейчас открыт «Ассистент».
function handleExternalOpen(e) {
  if (hidden.value) return
  const id = e.detail?.conversation_id
  if (id) {
    activeTab.value = 'messages'
    open.value = true
    openThread(id)
  }
}

if (typeof window !== 'undefined') {
  window.addEventListener('messenger:open-conversation', handleExternalOpen)
}

/* ── Вкладка «Ассистент» ────────────────────────────────────── */
const assistantThreadEl = ref(null)
const assistantInputRef = ref(null)

function scrollAssistantBottom() {
  const el = assistantThreadEl.value
  if (el) el.scrollTop = el.scrollHeight
}

function onAssistantSend(text) {
  assistantStore.send(text)
}

/* Обратная связь 👍/👎 — только для ответов с реальным id из БД (оптимистично
   добавленные local-* оценивать нечего — их нет на сервере). */
const FEEDBACK_REASONS = [
  { value: 'inaccurate', label: 'Неточно' },
  { value: 'irrelevant', label: 'Не по делу' },
  { value: 'incomplete', label: 'Неполно' },
]
const feedbackReasonFor = ref(null)

const canRate = (m) => m.role === 'assistant' && typeof m.id === 'number'

function onFeedbackUp(m) {
  feedbackReasonFor.value = null
  assistantStore.sendFeedback(m.id, 'up')
}

// 👎 голосует сразу; чипы причин уточняют его повторным upsert-голосом.
function onFeedbackDown(m) {
  feedbackReasonFor.value = m.id
  assistantStore.sendFeedback(m.id, 'down')
}

function onFeedbackReason(m, reason) {
  feedbackReasonFor.value = null
  assistantStore.sendFeedback(m.id, 'down', reason)
}

// Автоскролл на новое сообщение/начало «печатает» в открытой вкладке.
watch(() => assistantStore.messages.length, async () => {
  if (activeTab.value !== 'assistant') return
  await nextTick()
  scrollAssistantBottom()
})
watch(() => assistantStore.sending, async () => {
  if (activeTab.value !== 'assistant') return
  await nextTick()
  scrollAssistantBottom()
})
// Открыли панель/переключились на вкладку — прокрутка к последнему сообщению.
watch([open, activeTab], async ([isOpen, tab]) => {
  if (!isOpen || tab !== 'assistant') return
  await nextTick()
  scrollAssistantBottom()
})
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
  height: 70dvh;
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

.mini-title--name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text);
}

/* Хаб-шапка (вкладки хаба вместо статичного заголовка): кнопка «Свернуть» —
   у правого края, вкладки — компактно слева. */
.mini-head > .mini-icon:last-child {
  margin-left: auto;
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

.mini-head-avatar-wrap--dev {
  background: var(--color-tertiary-container);
  display: grid;
  place-items: center;
}

.mini-avatar-wrap--dev {
  display: grid;
  place-items: center;
  border-radius: 50%;
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  flex-shrink: 0;
}
.mini-head-avatar-wrap--dev { width: 32px; height: 32px; }
.mini-head-avatar-wrap--dev .material-symbols-outlined { font-size: 18px; font-variation-settings: 'FILL' 1; }
.mini-avatar-wrap--dev { width: 40px; height: 40px; }
.mini-avatar-wrap--dev .material-symbols-outlined { font-size: 22px; font-variation-settings: 'FILL' 1; }

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
    /* Хаб теперь виден и на мобильном (в т.ч. на /tasks и /messenger, где
       есть свой AppFab «Создать» на right:16px/bottom:~136px+safe) —
       поднимаем выше, чтобы 56px-кружки не накладывались друг на друга. */
    bottom: calc(150px + env(safe-area-inset-bottom, 0px));
  }
  /* Открытая панель — во весь экран: 70dvh-поповер тесен, а fixed-элемент
     над клавиатурой iOS перекрывал поле ввода. Высота дополнительно
     поджимается JS'ом под visualViewport при открытой клавиатуре. */
  .mini-panel {
    position: fixed;
    inset: 0;
    width: 100vw;
    height: 100dvh;
    max-height: none;
    border-radius: 0;
    border: none;
    padding-top: env(safe-area-inset-top, 0px);
  }
}

/* ── Вкладка «Ассистент» ────────────────────────────────────── */
.assistant-thread {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.assistant-chips {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 14px;
  width: 100%;
  max-width: 280px;
}
.assistant-chip {
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: var(--color-surface);
  color: var(--color-text);
  font: inherit;
  font-size: 13px;
  padding: 10px 14px;
  cursor: pointer;
  text-align: left;
  transition: background 0.15s, border-color 0.15s;
}
.assistant-chip:hover { background: var(--color-surface-high); border-color: var(--color-primary); }

.assistant-row {
  display: flex;
  justify-content: flex-start;
}

.assistant-row.outgoing { justify-content: flex-end; }

/* Обёртка «bubble + провенанс + фидбек» — колонкой, чтобы служебные строки
   висели под пузырём и не ломали его ширину. */
.assistant-msg {
  max-width: 78%;
  min-width: 0;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
}

.assistant-row.outgoing .assistant-msg { align-items: flex-end; }

.assistant-msg .assistant-bubble { max-width: 100%; }

.assistant-sources {
  font-size: 11px;
  color: var(--color-text-dim);
  margin-top: 3px;
  padding: 0 4px;
}

.assistant-feedback {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 2px;
  flex-wrap: wrap;
}

.assistant-fb-btn {
  width: 28px;
  height: 28px;
  border: none;
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.12s, color 0.12s;
}

.assistant-fb-btn:hover { background: var(--color-surface-high); color: var(--color-text); }

.assistant-fb-btn.active {
  color: var(--color-on-primary-container);
  background: var(--color-primary-container);
}

.assistant-fb-btn .material-symbols-outlined { font-size: 16px; }

.assistant-fb-chip {
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: var(--color-surface);
  color: var(--color-text-dim);
  font: inherit;
  font-size: 11.5px;
  min-height: 28px;
  padding: 4px 10px;
  cursor: pointer;
  transition: border-color 0.12s, color 0.12s;
}

.assistant-fb-chip:hover { border-color: var(--color-primary); color: var(--color-text); }

.assistant-bubble {
  max-width: 78%;
  background: var(--color-surface-high);
  color: var(--color-text);
  padding: 8px 12px;
  border-radius: var(--radius-lg);
  border-top-left-radius: var(--radius-xs);
  box-shadow: var(--shadow-sm);
  font-size: 13.5px;
  line-height: 1.4;
  word-wrap: break-word;
  white-space: pre-wrap;
}

.assistant-row.outgoing .assistant-bubble {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  border-top-left-radius: var(--radius-lg);
  border-top-right-radius: var(--radius-xs);
}

.assistant-note {
  align-self: center;
  text-align: center;
  font-size: 12.5px;
  color: var(--color-text-dim);
  background: var(--color-surface-low);
  border-radius: var(--radius-md);
  padding: 8px 12px;
}

.assistant-error {
  align-self: center;
  text-align: center;
  font-size: 12.5px;
  color: var(--color-error);
  padding: 4px 8px;
}

.assistant-typing {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 10px 14px;
}

.ai-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-text-dim);
  animation: aiTypingBounce 1.1s ease-in-out infinite;
}
.ai-dot:nth-child(2) { animation-delay: 0.15s; }
.ai-dot:nth-child(3) { animation-delay: 0.3s; }

@keyframes aiTypingBounce {
  0%, 60%, 100% { transform: translateY(0); opacity: 0.5; }
  30% { transform: translateY(-3px); opacity: 1; }
}
</style>

<style>
/* Мини-чат живёт на z-index 10050 (поверх ActiveUnitModal) — диалоги,
   открытые из него, надо поднять выше, иначе панель перекрывает маску. */
.app-dialog-mask.above-mini-mess {
  z-index: 10060 !important;
}
.app-dialog-root.above-mini-mess {
  z-index: 10061 !important;
}
</style>
