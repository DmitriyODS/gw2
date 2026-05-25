<template>
  <div class="messenger" :class="{ 'mobile-chat-open': isMobile && activeId }">
    <ConversationList
      :conversations="messenger.conversations"
      :active-id="activeId"
      :loading="messenger.loadingList"
      :hide-on-mobile="isMobile && !!activeId"
      @select="selectConversation"
      @new-chat="newChatOpen = true"
    />

    <section class="chat-panel" :class="{ 'is-mobile-hidden': isMobile && !activeId }">
      <header v-if="active" class="chat-header">
        <button v-if="isMobile" class="back-btn" @click="goBack" title="Назад">
          <span class="material-symbols-outlined">arrow_back</span>
        </button>
        <img class="chat-avatar" :src="avatarOf(active.other_user)" :alt="active.other_user?.fio" />
        <div class="chat-title">
          <div class="chat-fio">{{ active.other_user?.fio }}</div>
          <div class="chat-meta">@{{ active.other_user?.login }} · {{ active.other_user?.post || active.other_user?.role?.name }}</div>
        </div>
      </header>
      <div v-else class="chat-empty">
        <span class="material-symbols-outlined">chat</span>
        <p>Выберите чат слева или начните новый</p>
        <button class="btn-primary" @click="newChatOpen = true">
          <span class="material-symbols-outlined">edit_square</span>
          Новый чат
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
          <MessageBubble
            v-for="m in messenger.activeMessages"
            :key="m.id"
            :message="m"
            :is-mine="m.sender_id === authStore.user?.id"
          />
        </template>
      </div>

      <MessageInput
        v-if="active"
        :sending="messenger.sending"
        @send="onSend"
      />
    </section>

    <NewChatDialog v-model="newChatOpen" @pick="startWith" />
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMessengerStore } from '@/stores/messenger.js'
import { useAuthStore } from '@/stores/auth.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import {
  requestNotificationPermission, notificationsAllowed,
} from '@/utils/systemNotify.js'
import ConversationList from '@/components/messenger/ConversationList.vue'
import MessageBubble from '@/components/messenger/MessageBubble.vue'
import MessageInput from '@/components/messenger/MessageInput.vue'
import NewChatDialog from '@/components/messenger/NewChatDialog.vue'
import ProgressSpinner from 'primevue/progressspinner'

const route = useRoute()
const router = useRouter()
const messenger = useMessengerStore()
const authStore = useAuthStore()
const { isMobile } = useBreakpoint()

const newChatOpen = ref(false)
const messagesEl = ref(null)

const activeId = computed(() => messenger.activeConversationId)
const active = computed(() => messenger.activeConversation)

function avatarOf(u) {
  if (!u) return ''
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

async function selectConversation(id) {
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

async function onScroll() {
  const el = messagesEl.value
  if (!el || messenger.loadingMessages) return
  if (el.scrollTop > 80) return
  const arr = messenger.activeMessages
  if (!arr.length) return
  const firstId = arr[0].id
  const prevHeight = el.scrollHeight
  await messenger.fetchMessages(activeId.value, firstId)
  await nextTick()
  el.scrollTop = el.scrollHeight - prevHeight
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
})

onBeforeUnmount(() => {
  window.removeEventListener('messenger:open-conversation', handleExternalOpen)
})

watch(() => messenger.activeMessages.length, async (n, prev) => {
  if (n > prev) {
    await nextTick()
    const el = messagesEl.value
    if (!el) return
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 200
    if (nearBottom) scrollToBottom()
  }
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
  height: calc(100vh - 0px);
  min-height: 0;
  background: var(--color-bg);
}

.chat-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
}

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

.chat-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
}

.chat-title { min-width: 0; }

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
  gap: 12px;
  color: var(--color-text-dim);
  text-align: center;
  padding: 24px;
}

.chat-empty .material-symbols-outlined { font-size: 64px; }

.btn-primary {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  background: var(--color-primary);
  color: var(--color-on-primary);
  border: none;
  padding: 10px 18px;
  border-radius: var(--radius-md);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}

.btn-primary:hover { background: var(--color-primary-hover); }

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
  .chat-panel {
    position: fixed;
    inset: 0;
    z-index: 150;
    background: var(--color-bg);
    padding-bottom: 60px;
  }
  .messenger.mobile-chat-open .chat-panel {
    display: flex;
  }
}
</style>
