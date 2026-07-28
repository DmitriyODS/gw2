import { defineStore } from 'pinia'
import { ref } from 'vue'
import { sendAssistantMessage, getAssistantHistory, sendAssistantFeedback } from '@/api/assistant.js'

let localSeq = 0
const nextLocalId = () => `local-${Date.now()}-${++localSeq}`

function normalize(m) {
  return { id: m.id, role: m.role, text: m.text, sources: m.sources || null, createdAt: m.created_at }
}

// Диалог с деловым ИИ-ассистентом (aisvc). В отличие от мессенджера — один
// плоский тред на ПОЛЬЗОВАТЕЛЯ, без вложений/ответов/пересылки. Ассистент
// подключён к человеку (личный ключ в «Профиль → Интеграции»), а не к
// компании: активная компания нужна только инструментам статистики.
export const useAssistantStore = defineStore('assistant', () => {
  // Отсортировано старые → новые (сервер отдаёт новые → старые постранично).
  const messages = ref([])
  const loading = ref(false)
  const sending = ref(false)
  const loaded = ref(false)
  const error = ref(null)
  // Личный ключ не подключён (сервер ответил AI_DISABLED) — не ошибка, а
  // приглашение подключить: показываем мягкую системную заметку со ссылкой
  // в профиль.
  const disabled = ref(false)
  // Мои голоса 👍/👎: message_id → 'up'|'down' (сервер отдаёт my_feedback в
  // history — состояние переживает перезагрузку).
  const myFeedback = ref({})

  function applyErrorCode(e) {
    if (e?.error === 'AI_DISABLED') {
      disabled.value = true
      return true
    }
    return false
  }

  async function fetchHistory() {
    loading.value = true
    error.value = null
    try {
      const items = await getAssistantHistory({ limit: 50 })
      messages.value = items.slice().reverse().map(normalize)
      const votes = {}
      for (const m of items) {
        if (m.my_feedback) votes[m.id] = m.my_feedback
      }
      myFeedback.value = votes
      disabled.value = false
      loaded.value = true
    } catch (e) {
      if (!applyErrorCode(e)) {
        error.value = e?.message || 'Не удалось загрузить историю ассистента'
      }
    } finally {
      loading.value = false
    }
  }

  async function send(text) {
    const trimmed = (text || '').trim()
    if (!trimmed || sending.value) return
    const localId = nextLocalId()
    messages.value.push({ id: localId, role: 'user', text: trimmed, createdAt: new Date().toISOString() })
    sending.value = true
    error.value = null
    try {
      const res = await sendAssistantMessage(trimmed)
      messages.value.push({
        // Реальный id из БД — по нему работает обратная связь 👍/👎.
        id: res.id ?? nextLocalId(),
        role: 'assistant',
        text: res.text,
        sources: res.sources || null,
        createdAt: res.created_at || new Date().toISOString(),
      })
      disabled.value = false
      loaded.value = true
    } catch (e) {
      // Откат: убираем оптимистично добавленную реплику пользователя.
      messages.value = messages.value.filter(m => m.id !== localId)
      if (!applyErrorCode(e)) {
        error.value = e?.message || 'Не удалось отправить сообщение'
      }
    } finally {
      sending.value = false
    }
  }

  // Голос по ответу ассистента: оптимистично, с откатом при ошибке.
  // Повторный голос заменяет прежний (сервер делает upsert).
  async function sendFeedback(messageId, verdict, reason = null) {
    const prev = myFeedback.value[messageId]
    myFeedback.value = { ...myFeedback.value, [messageId]: verdict }
    try {
      await sendAssistantFeedback({ messageId, verdict, reason })
    } catch (e) {
      const rolledBack = { ...myFeedback.value }
      if (prev) rolledBack[messageId] = prev
      else delete rolledBack[messageId]
      myFeedback.value = rolledBack
      if (!applyErrorCode(e)) {
        error.value = e?.message || 'Не удалось отправить отзыв'
      }
    }
  }

  function reset() {
    messages.value = []
    loaded.value = false
    error.value = null
    disabled.value = false
    myFeedback.value = {}
  }

  return {
    messages, loading, sending, loaded, error, disabled, myFeedback,
    fetchHistory, send, sendFeedback, reset,
  }
})
