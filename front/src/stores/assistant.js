import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useAuthStore } from './auth.js'
import { sendAssistantMessage, getAssistantHistory, sendAssistantFeedback } from '@/api/assistant.js'

let localSeq = 0
const nextLocalId = () => `local-${Date.now()}-${++localSeq}`

function normalize(m) {
  return { id: m.id, role: m.role, text: m.text, sources: m.sources || null, createdAt: m.created_at }
}

// Диалог с деловым ИИ-ассистентом (aisvc). В отличие от мессенджера — один
// плоский тред на пользователя+компанию, без вложений/ответов/пересылки.
export const useAssistantStore = defineStore('assistant', () => {
  // Отсортировано старые → новые (сервер отдаёт новые → старые постранично).
  const messages = ref([])
  const loading = ref(false)
  const sending = ref(false)
  const loaded = ref(false)
  const error = ref(null)
  // Нет активной компании (супер-админ либо пользователь без компаний) —
  // ассистент недоступен в принципе, не дёргаем API вовсе.
  const unavailable = ref(false)
  // ИИ выключен для компании (сервер ответил AI_DISABLED) — временная,
  // а не структурная недоступность: показываем мягкую системную заметку.
  const disabled = ref(false)
  // Мои голоса 👍/👎: message_id → 'up'|'down' (сервер отдаёт my_feedback в
  // history — состояние переживает перезагрузку).
  const myFeedback = ref({})

  const hasActiveCompany = computed(() => useAuthStore().companyId != null)

  function applyErrorCode(e) {
    if (e?.error === 'BAD_REQUEST') {
      unavailable.value = true
      return true
    }
    if (e?.error === 'AI_DISABLED') {
      disabled.value = true
      return true
    }
    return false
  }

  async function fetchHistory() {
    if (!hasActiveCompany.value) {
      unavailable.value = true
      return
    }
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
      unavailable.value = false
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
      unavailable.value = false
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
    unavailable.value = false
    disabled.value = false
    myFeedback.value = {}
  }

  return {
    messages, loading, sending, loaded, error, unavailable, disabled, myFeedback,
    hasActiveCompany, fetchHistory, send, sendFeedback, reset,
  }
})
