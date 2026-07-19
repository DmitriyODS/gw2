<template>
  <div v-if="!hidden" class="mini-mess float-fade" :class="{ 'panel-open': open, 'float-hidden': floatingHidden && !open }">
    <!-- Скрим под мобильным листом (на десктопе скрыт CSS'ом). -->
    <transition name="mini-fade">
      <div v-if="open" class="mini-backdrop" @click="closePanel" />
    </transition>

    <!-- Панель -->
    <transition name="mini-pop">
      <div v-if="open" class="mini-panel" :style="panelStyle">
        <div class="mini-handle" aria-hidden="true"></div>
        <!-- Режим: вкладки хаба (Ассистент / список диалогов) -->
        <template v-if="!(activeTab === 'messages' && threadId)">
          <header class="mini-head">
            <SegmentedTabs v-model="activeTab" :tabs="hubTabs" full-width dense />
            <button class="mini-icon" title="Свернуть" aria-label="Свернуть" @click="closePanel">
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
                  <BrandLoader :size="48" />
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
                  <div v-if="!c.is_dev_chat && messenger.isTyping(c.id)" class="mini-conv-preview mini-conv-typing">печатает…</div>
                  <div v-else class="mini-conv-preview">{{ preview(c.last_message) }}</div>
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
            <button
              v-else
              class="mini-head-avatar-wrap as-btn"
              aria-label="Открыть профиль"
              @click="profileOpen = true"
            >
              <img class="mini-head-avatar" :src="avatarOf(threadConv?.other_user)" :alt="threadConv?.other_user?.fio" />
              <span v-if="threadOnline" class="online-dot mini-head-dot" title="В сети"></span>
            </button>
            <div class="mini-head-title">
              <span class="mini-title--name">
                <template v-if="threadConv?.is_dev_chat">Техподдержка</template>
                <template v-else>{{ threadConv?.other_user?.fio }}</template>
              </span>
              <span class="mini-head-status" :class="{ online: threadOnline || threadTyping }">
                <template v-if="threadConv?.is_dev_chat">Поддержка Groove Work</template>
                <template v-else-if="threadTyping">печатает…</template>
                <template v-else>{{ threadOnline ? 'в сети' : threadLastSeen }}</template>
              </span>
            </div>
            <button class="mini-icon" title="Свернуть" aria-label="Свернуть" @click="closePanel">
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
              <BrandLoader :size="48" />
            </div>
            <div v-for="g in messageGroups" :key="g.key" class="msg-day-group">
              <MessageDateDivider :label="g.label" @jump="jumpToDay" />
              <MessageBubble
                v-for="m in g.items"
                :key="m.id"
                :message="m"
                :is-mine="m.sender_id === authStore.user?.id"
                :show-pin="false"
                :me-id="authStore.user?.id"
                @reply="startReply"
                @forward="startForward"
                @delete="askDeleteMessage"
                @context-menu="openContextMenu"
                @join-call="onJoinCall"
                @open-task="openTask"
                @open-post="openPost"
                @quote-click="onQuoteClick"
                @react="emoji => onReact(m, emoji)"
              />
            </div>
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
            @typing="onTyping"
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
      :my-reactions="ctxMyReactions"
      @close="ctxMenu.visible = false"
      @action="onCtxAction"
      @react="onCtxReact"
    />

    <EmployeeProfileDialog
      v-if="threadConv?.other_user"
      v-model="profileOpen"
      :user="threadConv.other_user"
      elevated
    />

    <!-- Кнопка-FAB (прячется, пока открыт любой AppDialog — на мобильном
         bottom sheet живёт в той же нижней зоне экрана). Перетаскивается:
         drag + snap к краю, позиция в localStorage; клик сразу после
         перетаскивания игнорируется. -->
    <div
      v-show="!anyModalOpen || open"
      class="mini-fab-anchor"
      :class="{ dragging: fabDragging }"
      :style="fabStyle"
    >
      <button
        class="mini-fab float-spring"
        :class="{ active: open }"
        data-tutorial="mini-hub"
        :title="fabTitle"
        :aria-label="fabTitle"
        @pointerdown="onFabPointerDown"
        @click="onFabClick"
      >
        <span class="material-symbols-outlined">{{ fabIcon }}</span>
        <span v-if="!open && messenger.totalUnread" class="mini-fab-badge">
          {{ messenger.totalUnread > 99 ? '99+' : messenger.totalUnread }}
        </span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { anyModalOpen, registerOpenModal, unregisterOpenModal } from '@/composables/useOpenModals.js'
import { floatingHidden, installFloatingHide } from '@/composables/useFloatingHide.js'
import { useDraggable } from '@/composables/useDraggable.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { useThemeStore } from '@/stores/theme.js'
import { storageGetJSON } from '@/utils/storage.js'
import { stripMarkdown } from '@/utils/markdown.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { useAuthStore } from '@/stores/auth.js'
import { useCallStore } from '@/stores/call.js'
import { useAssistantStore } from '@/stores/assistant.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useJumpToMessage } from '@/composables/useJumpToMessage.js'
import { formatLastSeen } from '@/utils/presence.js'
import { groupMessagesByDay } from '@/utils/chatDates.js'
import MessageBubble from './MessageBubble.vue'
import MessageDateDivider from './MessageDateDivider.vue'
import MessageInput from './MessageInput.vue'
import AssistantInput from './AssistantInput.vue'
import ForwardDialog from './ForwardDialog.vue'
import DeleteScopeDialog from './DeleteScopeDialog.vue'
import AttachTaskDialog from './AttachTaskDialog.vue'
import MessageContextMenu from './MessageContextMenu.vue'
import EmployeeProfileDialog from '@/components/common/EmployeeProfileDialog.vue'
import SegmentedTabs from '@/components/common/SegmentedTabs.vue'
import LinkifiedText from '@/components/common/LinkifiedText.vue'
import MarkdownView from '@/components/common/MarkdownView.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'

const route = useRoute()
const router = useRouter()
const messenger = useMessengerStore()
const authStore = useAuthStore()
const callStore = useCallStore()
const assistantStore = useAssistantStore()
const notif = useNotificationsStore()
const themeStore = useThemeStore()
const { isMobile } = useBreakpoint()

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
const messageGroups = computed(() => groupMessagesByDay(messenger.activeMessages))
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
// /messenger) — хаб скрываем на fullscreen-роутах (ТВ, звонок по
// ссылке-приглашению), во время активного полноэкранного звонка (сам
// CallView тоже перекрывает экран, z-index 11500 > 10050 — но явно прячем
// FAB и на случай его мини-режима смены) и когда кнопка выключена
// тумблером в настройках внешнего вида.
const hidden = computed(() => {
  if (!themeStore.hubFabEnabled) return true
  if (route.meta?.fullscreen) return true
  return callStore.isInCall && !callStore.isMinimized
})

/* ── Перетаскивание FAB (drag + snap к краю, как у FloatingPet) ── */
const FAB_SIZE = { w: 56, h: 56 }
const FAB_POS_KEY = 'gw_hub_fab_pos'
const isNarrow = () => typeof window !== 'undefined' && window.innerWidth <= 768

const { pos: fabPos, dragging: fabDragging, onPointerDown: onFabPointerDown, wasDragged } = useDraggable({
  storageKey: FAB_POS_KEY,
  size: FAB_SIZE,
  defaultCorner: 'bottom-right',
  margin: isNarrow() ? 12 : 20,
  // На мобильном не пускаем ниже зоны AppBottomNav + AppFab «Создать»
  // (прежняя фиксированная позиция — bottom:150px).
  bottomInset: () => (isNarrow() ? 138 : 0),
})

// Пока позицию не трогали, в полном мессенджере (десктоп) FAB поднимается над
// полем ввода — в стандартном углу он заслонял бы кнопку отправки.
// Пользовательскую позицию не сдвигаем: она выбрана осознанно.
const hasCustomPos = ref(!!storageGetJSON(FAB_POS_KEY, null))
watch(fabDragging, (v, prev) => {
  if (prev && !v && wasDragged()) hasCustomPos.value = true
})
const raisedOffset = computed(() =>
  (!hasCustomPos.value && !isMobile.value && route.path.startsWith('/messenger')) ? 76 : 0)

// transform вместо left/top: композит-слой, без layout на каждый кадр драга.
const fabStyle = computed(() => ({
  transform: `translate3d(${fabPos.value.x}px, ${fabPos.value.y - raisedOffset.value}px, 0)`,
}))

function onFabClick() {
  if (wasDragged()) return // клик сразу после перетаскивания — игнорируем
  toggle()
}

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
const threadTyping = computed(() =>
  !!threadConv.value && !threadConv.value.is_dev_chat && messenger.isTyping(threadConv.value.id)
)

function onTyping(isTypingNow) {
  const id = threadConv.value?.id
  if (!id) return
  if (isTypingNow) messenger.notifyTyping(id)
  else messenger.notifyTypingStop(id)
}
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

// Геометрия панели. Десктоп: поповер якорится к FAB (над ним, при нехватке
// места — под ним) и клампится во вьюпорт — FAB теперь перетаскиваемый, панель
// следует за ним. Мобильный: нижний лист из CSS; JS вмешивается только при
// открытой клавиатуре iOS — перевешиваем лист к верху и поджимаем высоту под
// visualViewport (fixed-элементы клавиатура не двигает, bottom-якорь ушёл бы
// под неё).
const viewportTick = ref(0)
const bumpViewport = () => { viewportTick.value++ }

const panelStyle = computed(() => {
  viewportTick.value // зависимость: пересчёт на resize/клавиатуру
  if (typeof window === 'undefined') return {}
  if (window.innerWidth <= 768) {
    const vv = window.visualViewport
    return open.value && vv && vv.height < window.innerHeight - 60
      ? { top: '0px', bottom: 'auto', height: `${Math.round(vv.height)}px`, borderRadius: '0px' }
      : {}
  }
  const vw = window.innerWidth
  const vh = window.innerHeight
  const margin = 16
  const gap = 12
  const w = Math.min(360, vw - 2 * margin)
  const h = Math.min(560, Math.round(vh * 0.7))
  const fx = fabPos.value.x
  const fy = fabPos.value.y - raisedOffset.value
  const left = Math.min(Math.max(fx + FAB_SIZE.w - w, margin), Math.max(margin, vw - w - margin))
  const above = fy - gap - h >= margin
  const top = above
    ? fy - gap - h
    : Math.min(Math.max(fy + FAB_SIZE.h + gap, margin), Math.max(margin, vh - h - margin))
  // Панель «выпрыгивает» из кнопки — origin анимации в её точке.
  const originX = Math.min(Math.max(fx + FAB_SIZE.w / 2 - left, 0), w)
  return {
    left: `${left}px`,
    top: `${top}px`,
    width: `${w}px`,
    height: `${h}px`,
    transformOrigin: `${originX}px ${above ? '100%' : '0%'}`,
  }
})

// На мобильном открытый лист — полноэкранная модалка: регистрируем его в
// глобальном счётчике, чтобы плавающий питомец (z-index выше) не висел поверх
// переписки. На десктопе панель — угловой поповер, там виджеты не мешают.
let hubRegistered = false
watch(open, (v) => {
  const mobile = typeof window !== 'undefined' && window.innerWidth <= 768
  if (v && mobile && !hubRegistered) {
    registerOpenModal()
    hubRegistered = true
  } else if (!v && hubRegistered) {
    unregisterOpenModal()
    hubRegistered = false
  }
})

onMounted(() => {
  window.visualViewport?.addEventListener('resize', bumpViewport)
  window.addEventListener('resize', bumpViewport, { passive: true })
  installFloatingHide()
})
onBeforeUnmount(() => {
  window.visualViewport?.removeEventListener('resize', bumpViewport)
  window.removeEventListener('resize', bumpViewport)
  if (hubRegistered) unregisterOpenModal()
})

// Единая точка закрытия (FAB, крестик в шапке, тап по скриму): тред больше
// не виден — входящие в него не должны тихо помечаться прочитанными.
function closePanel() {
  open.value = false
  if (threadId.value) messenger.activeConversationId = null
}

function toggle() {
  if (open.value) {
    closePanel()
    return
  }
  open.value = true
  if (activeTab.value === 'messages') ensureMessagesFresh()
  else ensureAssistantHistory()
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

/* ── Контекстное меню (ПКМ / тап) ──────────────────────────── */
const ctxMenu = ref({ visible: false, x: 0, y: 0, message: null })
const profileOpen = ref(false)

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
    notif.error(e?.message || 'Не удалось поставить реакцию')
  }
}

function onCtxReact(emoji) {
  const m = ctxMenu.value.message
  if (m) onReact(m, emoji)
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
    // Поле не очищаем — текст остаётся для повтора (clearAfterSend не вызван).
    const msg = e?.error === 'TASK_WRONG_COMPANY'
      ? 'Задача относится к другой компании'
      : (e?.message || 'Не удалось отправить сообщение')
    notif.error(msg)
    return
  }
  // Очищаем поле только после успешной отправки.
  miniInputRef.value?.clearAfterSend()
  replyTo.value = null
  await nextTick()
  scrollBottom()
}

function scrollBottom() {
  const el = threadEl.value
  if (el) el.scrollTop = el.scrollHeight
}

// Клик по плашке даты — прокрутка к началу дня (первому сообщению группы).
// Считаем по rect первого пузыря (сам разделитель sticky), ~44px сверху под
// прилипшую пилюлю даты.
function jumpToDay(dividerEl) {
  const el = threadEl.value
  const first = dividerEl?.nextElementSibling
  if (!el || !first) return
  const top = el.scrollTop + (first.getBoundingClientRect().top - el.getBoundingClientRect().top) - 44
  el.scrollTo({ top: Math.max(0, top), behavior: 'smooth' })
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
  // Разметка в однострочном превью вычищается (жирный/списки/ссылки → текст).
  if (msg.text) return stripMarkdown(msg.text)
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

// Клик по системному уведомлению теперь переносит В РАЗДЕЛ чата (глобальный
// роутинг в App.vue → /messenger/:id), а не разворачивает плавающий хаб —
// поэтому здесь событие messenger:open-conversation больше не перехватываем.

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
/* Корень — нулевой фиксированный якорь: FAB и панель позиционируются
   независимо (каждый сам position:fixed). Драг-transform нельзя вешать на
   общий контейнер — он стал бы containing block для fixed мобильного
   листа/скрима. Поверх ActiveUnitModal (z-index 9999) — чтобы можно было
   ответить, не закрывая активный юнит. */
.mini-mess {
  position: fixed;
  left: 0;
  top: 0;
  z-index: 10050;
}

/* Якорь FAB: позиция — transform (композит-слой, без layout при драге). */
.mini-fab-anchor {
  position: fixed;
  left: 0;
  top: 0;
  touch-action: none;
}

.mini-fab-anchor.dragging .mini-fab { cursor: grabbing; }

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

/* Плавающая панель — стекло (Expressive Glass): контент просвечивает,
   внутренние поверхности прозрачные, текст лежит на плотных пузырях. */
.mini-panel {
  /* Десктоп: left/top/размеры считает panelStyle (якорь к FAB + кламп во
     вьюпорт); CSS-размеры — фолбэк. */
  position: fixed;
  width: 360px;
  max-width: calc(100vw - 32px);
  height: 70dvh;
  max-height: 560px;
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-xl);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

/* Ручка листа и скрим — только для мобильного нижнего листа. */
.mini-handle { display: none; }
.mini-backdrop { display: none; }

.mini-fade-enter-active, .mini-fade-leave-active { transition: opacity 0.18s ease; }
.mini-fade-enter-from, .mini-fade-leave-to { opacity: 0; }

.mini-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--acrylic-border);
  background: transparent;
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

/* Хаб-шапка: вкладки растянуты на всю доступную ширину, кнопка «Свернуть» —
   у правого края. */
.mini-head :deep(.seg-tabs) { flex: 1; min-width: 0; }
.mini-head > .mini-icon:last-child {
  margin-left: auto;
}

.mini-head-avatar-wrap {
  position: relative;
  flex-shrink: 0;
}

.mini-head-avatar-wrap.as-btn {
  border: none;
  background: transparent;
  padding: 0;
  cursor: pointer;
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
  height: 34px; min-height: 0;
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

.mini-icon:hover { background: var(--glass-hover-bg); color: var(--color-text); }
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

.mini-conv:hover { background: var(--glass-hover-bg); }

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
.mini-conv-preview.mini-conv-typing { color: var(--color-primary); font-style: italic; }

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
  background: transparent;
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
  /* Пока открыт лист, FAB не нужен (закрытие — крестик в шапке или скрим),
     иначе он висит поверх листа посреди переписки. */
  .mini-mess.panel-open .mini-fab { display: none; }

  .mini-backdrop {
    display: block;
    position: fixed;
    inset: 0;
    background: var(--color-scrim, color-mix(in oklch, var(--color-text) 45%, transparent));
  }

  /* Открытая панель — нижний лист почти во весь экран, как остальные
     мобильные модалки (AppDialog mobile-sheet, шторка «Ещё»): 70dvh-поповер
     тесен, а «окошко в углу» на телефоне неуместно. При открытой клавиатуре
     iOS высота/якорь поджимаются JS'ом под visualViewport. */
  .mini-panel {
    position: fixed;
    top: auto;
    left: 0;
    right: 0;
    bottom: 0;
    width: auto;
    /* Базовый max-width (100vw - 32px) оставлял справа полосу — лист обязан
       прижиматься к обоим краям. */
    max-width: none;
    height: calc(100dvh - 28px);
    max-height: none;
    border: none;
    border-top: 1px solid var(--acrylic-border);
    border-radius: var(--radius-xl) var(--radius-xl) 0 0;
    padding-bottom: env(safe-area-inset-bottom, 0px);
  }

  .mini-handle {
    display: block;
    flex-shrink: 0;
    align-self: center;
    width: 36px;
    height: 4px;
    border-radius: var(--radius-full);
    background: var(--color-outline-dim);
    margin: 8px 0 2px;
  }

  /* Лист выезжает снизу (как шторка «Ещё»), а не «выпрыгивает» из угла. */
  .mini-pop-enter-active, .mini-pop-leave-active {
    transform-origin: center bottom;
    transition: transform 0.22s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.18s ease;
  }
  .mini-pop-enter-from, .mini-pop-leave-to {
    transform: translateY(100%);
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
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: var(--color-surface);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font: inherit;
  font-size: 13px;
  padding: 10px 14px;
  cursor: pointer;
  text-align: left;
  transition: background 0.15s, border-color 0.15s;
}
.assistant-chip:hover { background: var(--glass-hover-bg); border-color: var(--color-primary); }

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
