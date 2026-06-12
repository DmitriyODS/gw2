import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as api from '@/api/messenger.js'
import { useAuthStore } from './auth.js'
// Циклический импорт (call.js ↔ messenger.js) безопасен: оба стора зовут
// друг друга только внутри функций, к этому моменту биндинги уже живые.
import { useCallStore } from './call.js'

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
  // Закреплённые сообщения по диалогам { [convId]: Message[] } (свежее — первым).
  const pinnedByConv = ref({})
  const loadingList = ref(false)
  const loadingMessages = ref(false)
  const sending = ref(false)
  // Присутствие: множество id онлайн-пользователей и живые last_seen
  // (приходят в presence:update при выходе из сети — точнее, чем в профиле).
  const onlineIds = ref(new Set())
  const lastSeenById = ref({})
  // Support-inbox (для Администратора системы): отдельный список чатов
  // техподдержки всех пользователей. Не сливается с conversations, чтобы
  // вкладка «Чаты» не видела сообщения из «Техподдержки».
  const supportInbox = ref([])
  const loadingSupportInbox = ref(false)
  let listSeq = 0
  let supportSeq = 0
  let listCtrl = null
  let supportCtrl = null
  const messagesCtrlByConv = new Map()
  const messagesSeqByConv = new Map()

  // Индекс по id и в обычных диалогах, и в support-inbox: support-чаты у
  // Администратора системы живут в отдельном списке (своя вкладка), но
  // activeConversation/applyIncomingMessage должны находить их одинаково.
  // conversations имеет приоритет — для обычного пользователя dev-чат лежит
  // именно там, и у него актуальный unread_count/last_message.
  const conversationById = computed(() => {
    const map = new Map()
    for (const c of supportInbox.value) map.set(c.id, c)
    for (const c of conversations.value) map.set(c.id, c)
    return map
  })

  const activeConversation = computed(() =>
    conversationById.value.get(activeConversationId.value) || null
  )

  const activeMessages = computed(() =>
    activeConversationId.value ? (messagesByConv.value[activeConversationId.value] || []) : []
  )

  const activePinned = computed(() =>
    activeConversationId.value ? (pinnedByConv.value[activeConversationId.value] || []) : []
  )

  async function fetchConversations() {
    const seq = ++listSeq
    listCtrl?.abort()
    listCtrl = new AbortController()
    loadingList.value = true
    try {
      const items = await api.listConversations({ signal: listCtrl.signal })
      if (seq !== listSeq) return
      conversations.value = sortConversations(items)
      recomputeUnread()
    } catch (e) {
      if (e?.error !== 'ABORTED') throw e
    } finally {
      if (seq === listSeq) {
        listCtrl = null
        loadingList.value = false
      }
    }
  }

  async function fetchUnreadCount() {
    try {
      const r = await api.getUnreadCount()
      totalUnread.value = r?.total ?? 0
    } catch {}
  }

  function recomputeUnread() {
    const convSum = conversations.value.reduce((s, c) => s + (c.unread_count || 0), 0)
    const supportSum = supportInbox.value.reduce((s, c) => s + (c.unread_count || 0), 0)
    totalUnread.value = convSum + supportSum
  }

  async function openDevChat() {
    // Личный чат техподдержки текущего пользователя. Бэк гарантирует, что чат
    // существует — get-or-create.
    const data = await api.openDevChat()
    const existing = conversationById.value.get(data.id)
    if (!existing) {
      conversations.value = sortConversations([
        {
          id: data.id,
          other_user: null,
          last_message: null,
          unread_count: 0,
          last_message_at: data.last_message_at,
          is_pinned: false,
          pinned_at: null,
          is_dev_chat: true,
          company_id: data.company_id,
          company_name: null,
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

  async function openPetChat() {
    // Чат с Грувиком текущего пользователя (get-or-create на бэке).
    const data = await api.openPetChat()
    const existing = conversationById.value.get(data.id)
    if (!existing) {
      conversations.value = sortConversations([
        {
          id: data.id,
          other_user: null,
          last_message: null,
          unread_count: 0,
          last_message_at: data.last_message_at,
          is_pinned: false,
          pinned_at: null,
          is_dev_chat: false,
          is_pet_chat: true,
          pet_name: data.pet_name || null,
          company_id: data.company_id,
          company_name: null,
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

  async function fetchSupportInbox() {
    const seq = ++supportSeq
    supportCtrl?.abort()
    supportCtrl = new AbortController()
    loadingSupportInbox.value = true
    try {
      const items = await api.listSupportInbox({ signal: supportCtrl.signal })
      if (seq !== supportSeq) return
      supportInbox.value = items
    } catch (e) {
      if (e?.error !== 'ABORTED') throw e
    } finally {
      if (seq === supportSeq) {
        supportCtrl = null
        loadingSupportInbox.value = false
      }
    }
  }

  const supportUnread = computed(() =>
    supportInbox.value.reduce((s, c) => s + (c.unread_count || 0), 0)
  )

  async function openWith(userId) {
    const data = await api.openConversation(userId)
    // upsert в список
    const existing = conversationById.value.get(data.id)
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
    fetchPinned(conversationId)
    await markRead(conversationId)
  }

  async function fetchPinned(conversationId) {
    try {
      pinnedByConv.value = {
        ...pinnedByConv.value,
        [conversationId]: await api.listPinnedMessages(conversationId),
      }
    } catch {}
  }

  async function fetchMessages(conversationId, beforeId = null) {
    const seq = beforeId ? null : (messagesSeqByConv.get(conversationId) || 0) + 1
    if (!beforeId) messagesSeqByConv.set(conversationId, seq)
    if (!beforeId) {
      messagesCtrlByConv.get(conversationId)?.abort()
      messagesCtrlByConv.set(conversationId, new AbortController())
    }
    loadingMessages.value = true
    try {
      const signal = beforeId ? null : messagesCtrlByConv.get(conversationId)?.signal
      const msgs = await api.listMessages(conversationId, { beforeId, signal })
      if (!beforeId && messagesSeqByConv.get(conversationId) !== seq) return msgs
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
    } catch (e) {
      if (e?.error === 'ABORTED') return []
      throw e
    } finally {
      if (beforeId || messagesSeqByConv.get(conversationId) === seq) {
        if (!beforeId) messagesCtrlByConv.delete(conversationId)
        loadingMessages.value = false
      }
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

  async function send(conversationId, { text, attachment_ids, reply_to_id, task_id }) {
    sending.value = true
    try {
      const msg = await api.sendMessage(conversationId, {
        text: text || null,
        attachment_ids: attachment_ids || [],
        reply_to_id: reply_to_id || null,
        task_id: task_id || null,
      })
      // Локально добавим сразу (сокет-эхо проигнорируется по id)
      applyIncomingMessage(conversationId, msg, /* fromMe */ true)
      // Раз пользователь отвечает в этот диалог — все входящие здесь точно
      // прочитаны. Гасим непрочитанные на сервере (и шлём read-receipt).
      markRead(conversationId)
      return msg
    } finally {
      sending.value = false
    }
  }

  async function forwardMessage(messageId, { conversationIds = [], userIds = [] } = {}) {
    // Сервер разошлёт message:new и нашим вкладкам тоже, но на эхо нельзя
    // полагаться: при пересылке новому собеседнику чат и сообщение появляются
    // в одном HTTP-ответе, а emit к нашей же комнате `user_{me}` иногда
    // обгоняет рендер списка (нужно ещё успеть подтянуть `conversations`).
    // Поэтому применяем ответ сразу — applyIncomingMessage дедуп'ит по id,
    // эхо просто не сработает повторно.
    const r = await api.forwardMessage(messageId, { conversationIds, userIds })
    for (const item of r?.forwarded || []) {
      if (item?.conversation_id && item?.message) {
        applyIncomingMessage(item.conversation_id, item.message, /* fromMe */ true)
      }
    }
    return r
  }

  /* «Активно смотрю на чат» — открыт И вкладка в фокусе. Только в этом случае
     входящее считается сразу прочитанным; иначе растёт счётчик непрочитанных. */
  function isViewingActively(conversationId) {
    // Полноэкранный звонок перекрывает мессенджер целиком — даже «открытый»
    // диалог в этот момент не виден, читать его нельзя.
    try {
      const call = useCallStore()
      if ((call.phase === 'active' || call.phase === 'outgoing') && !call.isMinimized) {
        return false
      }
    } catch { /* стор звонка ещё не инициализирован */ }
    return conversationId === activeConversationId.value
      && typeof document !== 'undefined'
      && document.visibilityState === 'visible'
      && (typeof document.hasFocus !== 'function' || document.hasFocus())
  }

  /* Помечает входящие прочитанными на сервере (всегда, без локального
     guard'а — иначе сообщения, пришедшие в открытый чат, оставались
     непрочитанными на сервере, и бейдж скакал после refetch). */
  async function markRead(conversationId) {
    try {
      await api.markRead(conversationId)
    } catch {}
    const conv = conversationById.value.get(conversationId)
    if (conv) conv.unread_count = 0
    recomputeUnread()
  }

  /* Обработка входящего сообщения (своего эхо или собеседника). */
  function applyIncomingMessage(conversationId, msg, fromMe = false) {
    const arr = messagesByConv.value[conversationId] || []
    if (arr.some(m => m.id === msg.id)) return
    messagesByConv.value[conversationId] = [...arr, msg]

    const auth = useAuthStore()
    const inConversations = conversations.value.some(c => c.id === conversationId)
    const inSupportInbox = supportInbox.value.some(c => c.id === conversationId)
    const viewingActively = isViewingActively(conversationId)
    const conv = conversationById.value.get(conversationId)

    if (conv) {
      conv.last_message = msg
      conv.last_message_at = msg.created_at
      if (!fromMe) {
        if (viewingActively) {
          // Сразу гасим на сервере — собеседник увидит «прочитано»,
          // а локальный счётчик не растёт.
          markRead(conversationId)
        } else {
          conv.unread_count = (conv.unread_count || 0) + 1
        }
      }
      // Пересортируем тот список, где живёт запись (закреплённые остаются
      // сверху). Support-inbox админ видит на своей вкладке, его сортировка
      // та же — по last_message_at.
      if (inConversations) {
        conversations.value = sortConversations(conversations.value)
      }
      if (inSupportInbox) {
        supportInbox.value = sortConversations(supportInbox.value)
      }
    } else {
      // Диалог появился впервые (или вернулся из «скрытых») — перезапрос
      // правильного списка. Для рут-админа dev-чаты живут в support-inbox,
      // для всех остальных — в обычном списке.
      if (auth.isRootAdmin) fetchSupportInbox()
      else fetchConversations()
    }
    recomputeUnread()
  }

  /* Обновление существующего сообщения. Используется для системной плашки
     звонка (kind='call'), которая при start статус 'ringing', а потом
     обновляется до 'active' (приняли) / 'ended' (положили) / 'missed'
     (никто не ответил). */
  function applyMessageUpdated(conversationId, msg) {
    const arr = messagesByConv.value[conversationId]
    if (arr) {
      const idx = arr.findIndex(m => m.id === msg.id)
      if (idx !== -1) {
        const next = [...arr]
        next[idx] = msg
        messagesByConv.value[conversationId] = next
      }
    }
    const conv = conversationById.value.get(conversationId)
    if (conv && conv.last_message?.id === msg.id) {
      conv.last_message = msg
    }
  }

  function applyMessageDeleted(conversationId, messageId) {
    const arr = messagesByConv.value[conversationId]
    if (arr) {
      messagesByConv.value[conversationId] = arr.filter(m => m.id !== messageId)
    }
    // Удалённое сообщение не должно оставаться в закреплённых.
    const pinned = pinnedByConv.value[conversationId]
    if (pinned?.some(m => m.id === messageId)) {
      pinnedByConv.value = {
        ...pinnedByConv.value,
        [conversationId]: pinned.filter(m => m.id !== messageId),
      }
    }
    const conv = conversationById.value.get(conversationId)
    if (conv && conv.last_message?.id === messageId) {
      const left = messagesByConv.value[conversationId] || []
      conv.last_message = left.length ? left[left.length - 1] : null
      conv.last_message_at = conv.last_message?.created_at || null
      conversations.value = sortConversations(conversations.value)
    }
  }

  function applyConversationDeleted(conversationId) {
    conversations.value = conversations.value.filter(c => c.id !== conversationId)
    supportInbox.value = supportInbox.value.filter(c => c.id !== conversationId)
    delete messagesByConv.value[conversationId]
    delete hasMoreHistoryByConv.value[conversationId]
    messagesSeqByConv.delete(conversationId)
    messagesCtrlByConv.get(conversationId)?.abort()
    messagesCtrlByConv.delete(conversationId)
    if (activeConversationId.value === conversationId) {
      activeConversationId.value = null
    }
    recomputeUnread()
  }

  function applyPinChange(conversationId, isPinned) {
    const conv = conversationById.value.get(conversationId)
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

  /* Закрепление сообщения изменилось (своё действие или эхо собеседника).
     Обновляем флаг в кеше сообщений и список закреплённых. */
  function applyMessagePin(conversationId, messageId, pinned, message) {
    const arr = messagesByConv.value[conversationId]
    if (arr) {
      const idx = arr.findIndex(m => m.id === messageId)
      if (idx !== -1) {
        const next = [...arr]
        next[idx] = { ...next[idx], pinned_at: message?.pinned_at ?? (pinned ? new Date().toISOString() : null), pinned_by_id: message?.pinned_by_id ?? null }
        messagesByConv.value[conversationId] = next
      }
    }
    const current = pinnedByConv.value[conversationId] || []
    let nextPinned
    if (pinned) {
      const item = message || arr?.find(m => m.id === messageId)
      nextPinned = item ? [item, ...current.filter(m => m.id !== messageId)] : current
    } else {
      nextPinned = current.filter(m => m.id !== messageId)
    }
    pinnedByConv.value = { ...pinnedByConv.value, [conversationId]: nextPinned }
  }

  async function togglePinMessageAction(messageId) {
    const r = await api.togglePinMessage(messageId)
    const convId = activeConversationId.value
    if (convId) applyMessagePin(convId, messageId, r.pinned, r.message)
    return r
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

  async function fetchPresence() {
    try {
      const r = await api.getPresence()
      onlineIds.value = new Set(r?.online || [])
    } catch {}
  }

  function applyPresence({ user_id, online, last_seen_at }) {
    const s = new Set(onlineIds.value)
    if (online) s.add(user_id)
    else s.delete(user_id)
    onlineIds.value = s
    if (last_seen_at) {
      lastSeenById.value = { ...lastSeenById.value, [user_id]: last_seen_at }
    }
  }

  function isOnline(userId) {
    return userId != null && onlineIds.value.has(userId)
  }

  function lastSeenOf(userId, fallback = null) {
    return lastSeenById.value[userId] || fallback
  }

  function reset() {
    conversations.value = []
    activeConversationId.value = null
    messagesByConv.value = {}
    hasMoreHistoryByConv.value = {}
    pinnedByConv.value = {}
    totalUnread.value = 0
    supportInbox.value = []
    onlineIds.value = new Set()
    lastSeenById.value = {}
    listSeq = 0
    supportSeq = 0
    listCtrl?.abort()
    supportCtrl?.abort()
    listCtrl = null
    supportCtrl = null
    messagesCtrlByConv.forEach(ctrl => ctrl.abort())
    messagesCtrlByConv.clear()
    messagesSeqByConv.clear()
  }

  return {
    conversations, conversationById, activeConversationId, messagesByConv, totalUnread,
    pinnedByConv,
    supportInbox, loadingSupportInbox, supportUnread,
    loadingList, loadingMessages, sending,
    onlineIds, lastSeenById,
    activeConversation, activeMessages, activePinned,
    fetchConversations, fetchUnreadCount, openWith, openDevChat, openPetChat,
    fetchSupportInbox, setActive, fetchMessages,
    fetchPinned, pollNewMessages, hasMoreHistory,
    send, forwardMessage, markRead,
    applyIncomingMessage, applyReadReceipt, applyMessageUpdated,
    applyMessageDeleted, applyConversationDeleted, applyPinChange,
    applyMessagePin, togglePinMessageAction,
    deleteMessage, deleteConversationAction, togglePinAction,
    fetchPresence, applyPresence, isOnline, lastSeenOf,
    reset,
  }
})
