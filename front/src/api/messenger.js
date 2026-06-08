import { apiRequest } from './client.js'

export const listConversations = (options = {}) =>
  apiRequest('/messenger/conversations', options)

export const openConversation = (userId) =>
  apiRequest('/messenger/conversations', { method: 'POST', body: { user_id: userId } })

export const listMessages = (conversationId, { beforeId = null, afterId = null, limit = 50, signal = null } = {}) => {
  const params = new URLSearchParams()
  if (beforeId) params.set('before_id', String(beforeId))
  if (afterId) params.set('after_id', String(afterId))
  if (limit) params.set('limit', String(limit))
  const qs = params.toString()
  return apiRequest(`/messenger/conversations/${conversationId}/messages${qs ? '?' + qs : ''}`, { signal })
}

export const sendMessage = (conversationId, payload) =>
  apiRequest(`/messenger/conversations/${conversationId}/messages`, { method: 'POST', body: payload })

export const forwardMessage = (messageId, { conversationIds = [], userIds = [] } = {}) =>
  apiRequest('/messenger/forward', {
    method: 'POST',
    body: { message_id: messageId, conversation_ids: conversationIds, user_ids: userIds },
  })

export const markRead = (conversationId) =>
  apiRequest(`/messenger/conversations/${conversationId}/read`, { method: 'POST' })

export const uploadAttachment = (file) => {
  const form = new FormData()
  form.append('file', file)
  return apiRequest('/messenger/uploads', { method: 'POST', body: form })
}

export const getUnreadCount = () =>
  apiRequest('/messenger/unread')

export const getPresence = () =>
  apiRequest('/messenger/presence')

export const deleteMessage = (messageId, scope = 'me') =>
  apiRequest(`/messenger/messages/${messageId}?scope=${scope}`, { method: 'DELETE' })

export const deleteConversation = (conversationId, scope = 'me') =>
  apiRequest(`/messenger/conversations/${conversationId}?scope=${scope}`, { method: 'DELETE' })

export const togglePin = (conversationId) =>
  apiRequest(`/messenger/conversations/${conversationId}/pin`, { method: 'POST' })

export const togglePinMessage = (messageId) =>
  apiRequest(`/messenger/messages/${messageId}/pin`, { method: 'POST' })

export const listPinnedMessages = (conversationId) =>
  apiRequest(`/messenger/conversations/${conversationId}/pinned`)

// Личный чат с техподдержкой текущего пользователя.
export const openDevChat = () => apiRequest('/messenger/dev-chat')

// Для Администратора системы: список чатов техподдержки всех пользователей.
export const listSupportInbox = (options = {}) => apiRequest('/messenger/support-inbox', options)
