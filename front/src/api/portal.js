// Ведётся вручную: REST корпоративного портала живёт в portalsvc (back-go/portal).
import { apiRequest } from './client.js'

// ── Топики (разделы) ──
export const getTopics = () => apiRequest('/portal/topics')

export const createTopic = ({ name, color = null, icon = null }) =>
  apiRequest('/portal/topics', { method: 'POST', body: { name, color, icon } })

export const updateTopic = (id, { name, color = null, icon = null }) =>
  apiRequest(`/portal/topics/${id}`, { method: 'PATCH', body: { name, color, icon } })

export const deleteTopic = (id) => apiRequest(`/portal/topics/${id}`, { method: 'DELETE' })

// ── Посты ──
// Серверная keyset-пагинация: ответ {pinned, posts, next_cursor} — pinned
// (все актуально закреплённые) приходит только на первой странице (без
// cursor), дальше пустой; next_cursor null — постов больше нет.
export const getPosts = ({ topicId, search, tag, limit, cursor } = {}) => {
  const qs = new URLSearchParams()
  if (topicId != null) qs.set('topic_id', topicId)
  if (search) qs.set('search', search)
  if (tag) qs.set('tag', tag)
  if (limit != null) qs.set('limit', limit)
  if (cursor) qs.set('cursor', cursor)
  const s = qs.toString()
  return apiRequest(`/portal/posts${s ? '?' + s : ''}`)
}

// Топ популярных хештегов компании (панель «Популярные теги» ленты).
export const getPopularTags = (limit) => {
  const s = limit != null ? `?limit=${limit}` : ''
  return apiRequest(`/portal/tags${s}`)
}

export const getPost = (id) => apiRequest(`/portal/posts/${id}`)

// Отметить просмотр поста (карточка попала в поле зрения). Идемпотентно.
export const markView = (id) => apiRequest(`/portal/posts/${id}/view`, { method: 'POST' })

export const createPost = ({ topicId = null, title = '', body }) =>
  apiRequest('/portal/posts', { method: 'POST', body: { topic_id: topicId, title, body } })

export const updatePost = (id, { topicId = null, title = '', body }) =>
  apiRequest(`/portal/posts/${id}`, { method: 'PATCH', body: { topic_id: topicId, title, body } })

export const deletePost = (id) => apiRequest(`/portal/posts/${id}`, { method: 'DELETE' })

// days: 1/7/30 — срок закрепления в днях, null — бессрочно.
export const pinPost = (id, days = null) =>
  apiRequest(`/portal/posts/${id}/pin`, { method: 'POST', body: { days } })
export const unpinPost = (id) => apiRequest(`/portal/posts/${id}/pin`, { method: 'DELETE' })

// Загрузка вложения (multipart) → Attachment {id, name, size, mime, url, ...}.
export const uploadAttachment = (postId, file) => {
  const form = new FormData()
  form.append('file', file)
  return apiRequest(`/portal/posts/${postId}/attachments`, { method: 'POST', body: form })
}

export const deleteAttachment = (attachmentId) =>
  apiRequest(`/portal/attachments/${attachmentId}`, { method: 'DELETE' })

// ── Комментарии (плоские) ──
export const getComments = (postId) => apiRequest(`/portal/posts/${postId}/comments`)

// replyToId — ответ на комментарий (дерево обсуждения); null — корневой.
export const createComment = (postId, text, replyToId = null) =>
  apiRequest(`/portal/posts/${postId}/comments`, {
    method: 'POST', body: { text, reply_to_id: replyToId },
  })

export const deleteComment = (commentId) =>
  apiRequest(`/portal/comments/${commentId}`, { method: 'DELETE' })

// Лайк комментария — toggle одной ручкой (как реакции мессенджера).
export const likeComment = (commentId) =>
  apiRequest(`/portal/comments/${commentId}/like`, { method: 'POST' })

// ── Реакции ──
export const addReaction = (postId, emoji) =>
  apiRequest(`/portal/posts/${postId}/reactions`, { method: 'POST', body: { emoji } })

export const removeReaction = (postId, emoji) =>
  apiRequest(`/portal/posts/${postId}/reactions?emoji=${encodeURIComponent(emoji)}`, { method: 'DELETE' })

// ── Непрочитанные (бейдж в навигации) ──
export const getUnreadCount = () => apiRequest('/portal/unread')
export const markSeen = () => apiRequest('/portal/seen', { method: 'POST' })

// ── Пересылка в мессенджер ──
export const forwardPost = (postId, { conversationIds = [], userIds = [] } = {}) =>
  apiRequest(`/portal/posts/${postId}/forward`, {
    method: 'POST',
    body: { conversation_ids: conversationIds, user_ids: userIds },
  })

// ── Оформление ленты (личное, синк между устройствами) ──
export const getBackground = () => apiRequest('/portal/background')

export const setBackground = (recipe) =>
  apiRequest('/portal/background', { method: 'PUT', body: { recipe } })

export const deleteBackground = () =>
  apiRequest('/portal/background', { method: 'DELETE' })
