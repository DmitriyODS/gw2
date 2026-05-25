import { apiRequest } from './client.js'

export const listConversations = () =>
  apiRequest('/messenger/conversations')

export const openConversation = (userId) =>
  apiRequest('/messenger/conversations', { method: 'POST', body: { user_id: userId } })

export const listMessages = (conversationId, beforeId = null, limit = 50) => {
  const params = new URLSearchParams()
  if (beforeId) params.set('before_id', String(beforeId))
  if (limit) params.set('limit', String(limit))
  const qs = params.toString()
  return apiRequest(`/messenger/conversations/${conversationId}/messages${qs ? '?' + qs : ''}`)
}

export const sendMessage = (conversationId, payload) =>
  apiRequest(`/messenger/conversations/${conversationId}/messages`, { method: 'POST', body: payload })

export const markRead = (conversationId) =>
  apiRequest(`/messenger/conversations/${conversationId}/read`, { method: 'POST' })

export const uploadAttachment = (file) => {
  const form = new FormData()
  form.append('file', file)
  return apiRequest('/messenger/uploads', { method: 'POST', body: form })
}

export const getUnreadCount = () =>
  apiRequest('/messenger/unread')
