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
  // Стандартный таймаут apiRequest (8с) обрывает загрузку больших файлов
  // на середине — даём вложениям до 3 минут.
  return apiRequest('/messenger/uploads', { method: 'POST', body: form, timeout: 180000 })
}

export const getUnreadCount = () =>
  apiRequest('/messenger/unread')

export const getPresence = () =>
  apiRequest('/messenger/presence')

export const deleteMessage = (messageId, scope = 'me') =>
  apiRequest(`/messenger/messages/${messageId}?scope=${scope}`, { method: 'DELETE' })

export const updateMessage = (messageId, text) =>
  apiRequest(`/messenger/messages/${messageId}`, { method: 'PATCH', body: { text } })

export const deleteConversation = (conversationId, scope = 'me') =>
  apiRequest(`/messenger/conversations/${conversationId}?scope=${scope}`, { method: 'DELETE' })

export const togglePin = (conversationId) =>
  apiRequest(`/messenger/conversations/${conversationId}/pin`, { method: 'POST' })

export const togglePinMessage = (messageId) =>
  apiRequest(`/messenger/messages/${messageId}/pin`, { method: 'POST' })

export const toggleReaction = (messageId, emoji) =>
  apiRequest(`/messenger/messages/${messageId}/reactions`, { method: 'POST', body: { emoji } })

export const listPinnedMessages = (conversationId) =>
  apiRequest(`/messenger/conversations/${conversationId}/pinned`)

// ── Группы ──────────────────────────────────────────────────────

export const createGroup = ({ title, memberIds = [], avatarAttachmentId = null } = {}) =>
  apiRequest('/messenger/groups', {
    method: 'POST',
    body: { title, member_ids: memberIds, avatar_attachment_id: avatarAttachmentId },
  })

export const getGroup = (conversationId) =>
  apiRequest(`/messenger/groups/${conversationId}`)

export const renameGroup = (conversationId, title) =>
  apiRequest(`/messenger/groups/${conversationId}`, { method: 'PATCH', body: { title } })

export const setGroupAvatar = (conversationId, avatarAttachmentId) =>
  apiRequest(`/messenger/groups/${conversationId}/avatar`, {
    method: 'POST', body: { avatar_attachment_id: avatarAttachmentId },
  })

export const addGroupMembers = (conversationId, userIds) =>
  apiRequest(`/messenger/groups/${conversationId}/members`, { method: 'POST', body: { user_ids: userIds } })

export const removeGroupMember = (conversationId, userId) =>
  apiRequest(`/messenger/groups/${conversationId}/members/${userId}`, { method: 'DELETE' })

export const setMemberRole = (conversationId, userId, role) =>
  apiRequest(`/messenger/groups/${conversationId}/members/${userId}`, { method: 'PATCH', body: { role } })

export const setMemberRights = (conversationId, userId, rights) =>
  apiRequest(`/messenger/groups/${conversationId}/members/${userId}`, { method: 'PATCH', body: { rights } })

export const transferOwnership = (conversationId, userId) =>
  apiRequest(`/messenger/groups/${conversationId}/members/${userId}/owner`, { method: 'POST' })

export const leaveGroup = (conversationId) =>
  apiRequest(`/messenger/groups/${conversationId}/leave`, { method: 'POST' })

export const muteGroup = (conversationId, muted) =>
  apiRequest(`/messenger/groups/${conversationId}/mute`, { method: 'POST', body: { muted } })

export const groupInviteLink = (conversationId) =>
  apiRequest(`/messenger/groups/${conversationId}/invite-link`, { method: 'POST' })

export const revokeGroupInviteLink = (conversationId) =>
  apiRequest(`/messenger/groups/${conversationId}/invite-link`, { method: 'DELETE' })

export const groupInvitePreview = (code) =>
  apiRequest(`/messenger/groups/invite/${code}`)

export const joinGroup = (code) =>
  apiRequest(`/messenger/groups/join/${code}`, { method: 'POST' })

export const messageReadBy = (messageId) =>
  apiRequest(`/messenger/messages/${messageId}/read-by`)

// Личный чат с техподдержкой текущего пользователя.
export const openDevChat = () => apiRequest('/messenger/dev-chat')

// Для Администратора системы: список чатов техподдержки всех пользователей.
export const listSupportInbox = (options = {}) => apiRequest('/messenger/support-inbox', options)

// ── Оформление чатов (личное, синк между устройствами) ────────────
export const getChatBackgrounds = () => apiRequest('/messenger/chat-bg')

// conversationId === null — общий дефолт пользователя.
export const setChatBackground = (conversationId, recipe) =>
  apiRequest('/messenger/chat-bg', {
    method: 'PUT',
    body: { conversation_id: conversationId ?? null, recipe },
  })

export const deleteChatBackground = (conversationId = null) => {
  const qs = conversationId ? `?conversation_id=${conversationId}` : ''
  return apiRequest(`/messenger/chat-bg${qs}`, { method: 'DELETE' })
}

// ── Папки чатов (личная навигация, синк между устройствами) ───────
export const listFolders = () => apiRequest('/messenger/folders')

export const createFolder = (payload) =>
  apiRequest('/messenger/folders', { method: 'POST', body: payload })

export const updateFolder = (folderId, payload) =>
  apiRequest(`/messenger/folders/${folderId}`, { method: 'PATCH', body: payload })

export const deleteFolder = (folderId) =>
  apiRequest(`/messenger/folders/${folderId}`, { method: 'DELETE' })

export const reorderFolders = (order) =>
  apiRequest('/messenger/folders/reorder', { method: 'POST', body: { order } })

export const addFolderItem = (folderId, conversationId) =>
  apiRequest(`/messenger/folders/${folderId}/items`, { method: 'POST', body: { conversation_id: conversationId } })

export const removeFolderItem = (folderId, conversationId) =>
  apiRequest(`/messenger/folders/${folderId}/items/${conversationId}`, { method: 'DELETE' })
