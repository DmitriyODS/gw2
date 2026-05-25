import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as api from '@/api/messenger.js'
import { useAuthStore } from './auth.js'

/* Состояние мессенджера.
   conversations — список диалогов (для левой панели).
   activeConversationId — открытый диалог (для правой панели).
   messagesByConv — кеш сообщений { [convId]: Message[] }.
   pendingUploads — временное состояние загружаемых файлов на форме ввода. */
export const useMessengerStore = defineStore('messenger', () => {
  const conversations = ref([])
  const activeConversationId = ref(null)
  const messagesByConv = ref({})
  const totalUnread = ref(0)
  const loadingList = ref(false)
  const loadingMessages = ref(false)
  const sending = ref(false)

  const activeConversation = computed(() =>
    conversations.value.find(c => c.id === activeConversationId.value) || null
  )

  const activeMessages = computed(() =>
    activeConversationId.value ? (messagesByConv.value[activeConversationId.value] || []) : []
  )

  async function fetchConversations() {
    loadingList.value = true
    try {
      conversations.value = await api.listConversations()
      recomputeUnread()
    } finally {
      loadingList.value = false
    }
  }

  async function fetchUnreadCount() {
    try {
      const r = await api.getUnreadCount()
      totalUnread.value = r?.total ?? 0
    } catch {}
  }

  function recomputeUnread() {
    totalUnread.value = conversations.value.reduce((s, c) => s + (c.unread_count || 0), 0)
  }

  async function openWith(userId) {
    const data = await api.openConversation(userId)
    // upsert в список
    const existing = conversations.value.find(c => c.id === data.id)
    if (!existing) {
      conversations.value.unshift({
        id: data.id,
        other_user: data.other_user,
        last_message: null,
        unread_count: 0,
        last_message_at: data.last_message_at,
      })
    }
    activeConversationId.value = data.id
    if (!messagesByConv.value[data.id]) {
      await fetchMessages(data.id)
    }
    return data.id
  }

  async function setActive(conversationId) {
    activeConversationId.value = conversationId
    if (!messagesByConv.value[conversationId]) {
      await fetchMessages(conversationId)
    }
    await markRead(conversationId)
  }

  async function fetchMessages(conversationId, beforeId = null) {
    loadingMessages.value = true
    try {
      const msgs = await api.listMessages(conversationId, beforeId)
      const existing = messagesByConv.value[conversationId] || []
      if (beforeId) {
        messagesByConv.value[conversationId] = [...msgs, ...existing]
      } else {
        messagesByConv.value[conversationId] = msgs
      }
    } finally {
      loadingMessages.value = false
    }
  }

  async function send(conversationId, { text, attachment_ids }) {
    sending.value = true
    try {
      const msg = await api.sendMessage(conversationId, {
        text: text || null,
        attachment_ids: attachment_ids || [],
      })
      // Локально добавим сразу (сокет-эхо проигнорируется по id)
      applyIncomingMessage(conversationId, msg, /* fromMe */ true)
      return msg
    } finally {
      sending.value = false
    }
  }

  async function markRead(conversationId) {
    const conv = conversations.value.find(c => c.id === conversationId)
    if (!conv || conv.unread_count === 0) return
    try {
      await api.markRead(conversationId)
      conv.unread_count = 0
      recomputeUnread()
    } catch {}
  }

  /* Обработка входящего сообщения (своего эхо или собеседника). */
  function applyIncomingMessage(conversationId, msg, fromMe = false) {
    const arr = messagesByConv.value[conversationId] || []
    if (arr.some(m => m.id === msg.id)) return
    messagesByConv.value[conversationId] = [...arr, msg]

    let conv = conversations.value.find(c => c.id === conversationId)
    if (conv) {
      conv.last_message = msg
      conv.last_message_at = msg.created_at
      if (!fromMe && conversationId !== activeConversationId.value) {
        conv.unread_count = (conv.unread_count || 0) + 1
      }
      // Поднимаем диалог наверх
      conversations.value = [conv, ...conversations.value.filter(c => c.id !== conv.id)]
    } else {
      // Диалог появился впервые — перезапрашиваем список
      fetchConversations()
    }
    recomputeUnread()
  }

  function applyReadReceipt(conversationId, readerId) {
    const auth = useAuthStore()
    if (readerId === auth.user?.id) return
    const arr = messagesByConv.value[conversationId]
    if (!arr) return
    const stamp = new Date().toISOString()
    arr.forEach(m => {
      if (m.sender_id === auth.user?.id && !m.read_at) {
        m.read_at = stamp
      }
    })
  }

  function reset() {
    conversations.value = []
    activeConversationId.value = null
    messagesByConv.value = {}
    totalUnread.value = 0
  }

  return {
    conversations, activeConversationId, messagesByConv, totalUnread,
    loadingList, loadingMessages, sending,
    activeConversation, activeMessages,
    fetchConversations, fetchUnreadCount, openWith, setActive, fetchMessages,
    send, markRead, applyIncomingMessage, applyReadReceipt, reset,
  }
})
