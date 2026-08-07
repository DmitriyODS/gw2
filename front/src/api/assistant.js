import { apiRequest } from './client.js'

// Деловой ИИ-ассистент (aisvc, /api/ai/assistant/*). Диалог один на
// ПОЛЬЗОВАТЕЛЯ и работает на его личном ключе (api/ai.js — my-settings),
// поэтому активная компания не обязательна: без неё недоступны только
// инструменты компанийной статистики. Нет личного ключа — сервер отвечает
// 409 AI_DISABLED (обрабатывает stores/assistant.js).
export const sendAssistantMessage = (text) =>
  apiRequest('/ai/assistant/messages', { method: 'POST', body: { text } })

// Голос 👍/👎 по ответу ассистента: идемпотентный upsert (повторный голос
// заменяет), reason — только для 👎 ('inaccurate'|'irrelevant'|'incomplete').
export const sendAssistantFeedback = ({ messageId, verdict, reason = null }) =>
  apiRequest('/ai/assistant/feedback', {
    method: 'POST',
    body: { message_id: messageId, verdict, reason },
  })

export const getAssistantHistory = ({ limit, before } = {}) => {
  const params = new URLSearchParams()
  if (limit) params.set('limit', String(limit))
  if (before) params.set('before', String(before))
  const qs = params.toString()
  return apiRequest(`/ai/assistant/history${qs ? '?' + qs : ''}`)
}
