import { apiRequest } from './client.js'

export const listConversations = () =>
  apiRequest('/messenger/conversations')

export const openConversation = (userId) =>
  apiRequest('/messenger/conversations', { method: 'POST', body: { user_id: userId } })

export const listMessages = (conversationId, { beforeId = null, afterId = null, limit = 50 } = {}) => {
  const params = new URLSearchParams()
  if (beforeId) params.set('before_id', String(beforeId))
  if (afterId) params.set('after_id', String(afterId))
  if (limit) params.set('limit', String(limit))
  const qs = params.toString()
  return apiRequest(`/messenger/conversations/${conversationId}/messages${qs ? '?' + qs : ''}`)
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

export const openDevChat = (companyId = null) => {
  const qs = companyId ? `?company_id=${companyId}` : ''
  return apiRequest(`/messenger/dev-chat${qs}`)
}

export const listDevChats = () =>
  apiRequest('/messenger/dev-chats')
