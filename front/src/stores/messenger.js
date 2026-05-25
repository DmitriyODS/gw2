import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as api from '@/api/messenger.js'
import { useAuthStore } from './auth.js'

/* Сортировка: закреплённые сверху (по pinned_at desc), затем по
   last_message_at desc. Чистая функция, чтобы переиспользовать после каждого
   изменения списка. */
function sortConversations(list) {
  return [...list].sort((a, b) => {
    const ap = a.is_pinned ? new Date(a.pinned_at || 0).getTime() : 0
    const bp = b.is_pinned ? new Date(b.pinned_at || 0).getTime() : 0
    if (ap && !bp) return -1
    if (!ap && bp) return 1
    if (ap && bp) return bp - ap
    const at = a.last_message_at ? new Date(a.last_message_at).getTime() : 0
    const bt = b.last_message_at ? new Date(b.last_message_at).getTime() : 0
    return bt - at
  })
}

/* Состояние мессенджера.
   conversations — список диалогов (для левой панели).
   activeConversationId — открытый диалог (для правой панели).
   messagesByConv — кеш сообщений { [convId]: Message[] }.
   pendingUploads — временное состояние загружаемых файлов на форме ввода. */
export const useMessengerStore = defineStore('messenger', () => {
  const conversations = ref([])
  const activeConversationId = ref(null)
  const messagesByConv = ref({})
  // hasMoreHistoryByConv[convId] = false означает «вся история уже подгружена»,
  // больше не нужно дёргать /messages?before_id — иначе скролл «примагничивается»
  // к верху, бесконечно повторяя пустой запрос.
  const hasMoreHistoryByConv = ref({})
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
      conversations.value = sortConversations(await api.listConversations())
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
      conversations.value = sortConversations([
        {
          id: data.id,
          other_user: data.other_user,
          last_message: null,
          unread_count: 0,
          last_message_at: data.last_message_at,
          is_pinned: false,
          pinned_at: null,
        },
        ...conversations.value,
      ])
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
      const msgs = await api.listMessages(conversationId, { beforeId })
      const existing = messagesByConv.value[conversationId] || []
      if (beforeId) {
        messagesByConv.value[conversationId] = [...msgs, ...existing]
        // Если страница вернулась короче лимита (или пустая) — история закончилась.
        if (msgs.length < 50) {
          hasMoreHistoryByConv.value[conversationId] = false
        }
      } else {
        messagesByConv.value[conversationId] = msgs
        // Первая загрузка: если меньше лимита — старых сообщений больше нет.
        hasMoreHistoryByConv.value[conversationId] = msgs.length >= 50
      }
      return msgs
    } finally {
      loadingMessages.value = false
    }
  }

  function hasMoreHistory(conversationId) {
    return hasMoreHistoryByConv.value[conversationId] !== false
  }

  /* Тихий polling-fallback: подтягивает только сообщения новее последнего
     известного id. Не трогает loadingMessages и не сбрасывает скролл/историю. */
  async function pollNewMessages(conversationId) {
    const existing = messagesByConv.value[conversationId] || []
    const lastId = existing.length ? existing[existing.length - 1].id : 0
    try {
      const fresh = await api.listMessages(conversationId, { afterId: lastId, limit: 100 })
      if (!fresh.length) return
      for (const m of fresh) {
        applyIncomingMessage(conversationId, m, m.sender_id === useAuthStore().user?.id)
      }
    } catch {}
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
      // Пересортируем с учётом нового времени (закреплённые остаются вверху).
      conversations.value = sortConversations(conversations.value)
    } else {
      // Диалог появился впервые (или вернулся из «скрытых») — перезапрос.
      fetchConversations()
    }
    recomputeUnread()
  }

  function applyMessageDeleted(conversationId, messageId) {
    const arr = messagesByConv.value[conversationId]
    if (arr) {
      messagesByConv.value[conversationId] = arr.filter(m => m.id !== messageId)
    }
    const conv = conversations.value.find(c => c.id === conversationId)
    if (conv && conv.last_message?.id === messageId) {
      const left = messagesByConv.value[conversationId] || []
      conv.last_message = left.length ? left[left.length - 1] : null
      conv.last_message_at = conv.last_message?.created_at || null
      conversations.value = sortConversations(conversations.value)
    }
  }

  function applyConversationDeleted(conversationId) {
    conversations.value = conversations.value.filter(c => c.id !== conversationId)
    delete messagesByConv.value[conversationId]
    delete hasMoreHistoryByConv.value[conversationId]
    if (activeConversationId.value === conversationId) {
      activeConversationId.value = null
    }
    recomputeUnread()
  }

  function applyPinChange(conversationId, isPinned) {
    const conv = conversations.value.find(c => c.id === conversationId)
    if (!conv) return
    conv.is_pinned = isPinned
    conv.pinned_at = isPinned ? new Date().toISOString() : null
    conversations.value = sortConversations(conversations.value)
  }

  async function deleteMessage(messageId, scope = 'me') {
    // Локально убираем сразу — UI плавнее.
    const convId = activeConversationId.value
    if (convId) applyMessageDeleted(convId, messageId)
    try {
      await api.deleteMessage(messageId, scope)
    } catch (e) {
      // Если не получилось — откатываемся перезагрузкой.
      if (convId) await fetchMessages(convId)
      throw e
    }
  }

  async function deleteConversationAction(conversationId, scope = 'me') {
    try {
      await api.deleteConversation(conversationId, scope)
    } finally {
      applyConversationDeleted(conversationId)
    }
  }

  async function togglePinAction(conversationId) {
    const conv = conversations.value.find(c => c.id === conversationId)
    if (!conv) return
    const optimisticPinned = !conv.is_pinned
    applyPinChange(conversationId, optimisticPinned)
    try {
      const r = await api.togglePin(conversationId)
      if (r.is_pinned !== optimisticPinned) {
        applyPinChange(conversationId, r.is_pinned)
      }
    } catch (e) {
      applyPinChange(conversationId, !optimisticPinned)
      throw e
    }
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
    hasMoreHistoryByConv.value = {}
    totalUnread.value = 0
  }

  return {
    conversations, activeConversationId, messagesByConv, totalUnread,
    loadingList, loadingMessages, sending,
    activeConversation, activeMessages,
    fetchConversations, fetchUnreadCount, openWith, setActive, fetchMessages,
    pollNewMessages, hasMoreHistory,
    send, markRead,
    applyIncomingMessage, applyReadReceipt,
    applyMessageDeleted, applyConversationDeleted, applyPinChange,
    deleteMessage, deleteConversationAction, togglePinAction,
    reset,
  }
})
