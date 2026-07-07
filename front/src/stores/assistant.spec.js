import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('@/api/assistant.js', () => ({
  sendAssistantMessage: vi.fn(),
  getAssistantHistory: vi.fn(),
  sendAssistantFeedback: vi.fn(),
}))

import * as api from '@/api/assistant.js'
import { useAssistantStore } from './assistant.js'
import { useAuthStore } from './auth.js'

function setActiveCompany(companyId = 10) {
  useAuthStore().applySession({ access_token: 't', user_id: 1, company_id: companyId })
}

describe('assistant store', () => {
  let assistant
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    assistant = useAssistantStore()
  })

  it('без активной компании fetchHistory не дёргает API и ставит unavailable', async () => {
    await assistant.fetchHistory()
    expect(api.getAssistantHistory).not.toHaveBeenCalled()
    expect(assistant.unavailable).toBe(true)
    expect(assistant.messages).toEqual([])
  })

  it('fetchHistory разворачивает историю (новые→старые) в старые→новые', async () => {
    setActiveCompany()
    api.getAssistantHistory.mockResolvedValue([
      { id: 3, role: 'assistant', text: 'Ответ 2', created_at: '2026-07-07T10:02:00Z' },
      { id: 2, role: 'user', text: 'Вопрос 2', created_at: '2026-07-07T10:01:00Z' },
      { id: 1, role: 'assistant', text: 'Ответ 1', created_at: '2026-07-07T10:00:00Z' },
    ])
    await assistant.fetchHistory()
    expect(assistant.messages.map(m => m.id)).toEqual([1, 2, 3])
    expect(assistant.loaded).toBe(true)
    expect(assistant.unavailable).toBe(false)
  })

  it('fetchHistory нормализует sources и наполняет myFeedback из my_feedback', async () => {
    setActiveCompany()
    api.getAssistantHistory.mockResolvedValue([
      {
        id: 2, role: 'assistant', text: 'Ответ', created_at: '2026-07-07T10:01:00Z',
        sources: 'Данные: статистика за эту неделю', my_feedback: 'up',
      },
      { id: 1, role: 'user', text: 'Вопрос', created_at: '2026-07-07T10:00:00Z' },
    ])
    await assistant.fetchHistory()
    expect(assistant.messages[1].sources).toBe('Данные: статистика за эту неделю')
    expect(assistant.messages[0].sources).toBeNull()
    expect(assistant.myFeedback).toEqual({ 2: 'up' })
  })

  it('AI_DISABLED при загрузке истории ставит disabled, не error', async () => {
    setActiveCompany()
    api.getAssistantHistory.mockRejectedValue({ error: 'AI_DISABLED', message: 'AI выключен' })
    await assistant.fetchHistory()
    expect(assistant.disabled).toBe(true)
    expect(assistant.error).toBeNull()
  })

  it('send оптимистично добавляет реплику пользователя, затем ответ ассистента', async () => {
    setActiveCompany()
    api.sendAssistantMessage.mockResolvedValue({ text: 'Вот статистика недели' })
    const promise = assistant.send('Сколько задач закрыто?')
    expect(assistant.messages).toHaveLength(1)
    expect(assistant.messages[0]).toMatchObject({ role: 'user', text: 'Сколько задач закрыто?' })
    expect(assistant.sending).toBe(true)
    await promise
    expect(assistant.sending).toBe(false)
    expect(assistant.messages).toHaveLength(2)
    expect(assistant.messages[1]).toMatchObject({ role: 'assistant', text: 'Вот статистика недели' })
  })

  it('send использует реальный id и sources из ответа сервера', async () => {
    setActiveCompany()
    api.sendAssistantMessage.mockResolvedValue({
      id: 42, text: 'Ответ', sources: 'Данные: лидеры за этот месяц', created_at: '2026-07-07T10:05:00Z',
    })
    await assistant.send('Кто лидер месяца?')
    expect(assistant.messages[1]).toMatchObject({
      id: 42, role: 'assistant', sources: 'Данные: лидеры за этот месяц', createdAt: '2026-07-07T10:05:00Z',
    })
  })

  it('sendFeedback обновляет myFeedback и шлёт голос на сервер', async () => {
    setActiveCompany()
    api.sendAssistantFeedback.mockResolvedValue({ status: 'ok' })
    await assistant.sendFeedback(42, 'down', 'inaccurate')
    expect(api.sendAssistantFeedback).toHaveBeenCalledWith({ messageId: 42, verdict: 'down', reason: 'inaccurate' })
    expect(assistant.myFeedback[42]).toBe('down')
    // Повторный голос заменяет прежний.
    await assistant.sendFeedback(42, 'up')
    expect(assistant.myFeedback[42]).toBe('up')
  })

  it('sendFeedback откатывает голос при ошибке сервера', async () => {
    setActiveCompany()
    api.sendAssistantFeedback.mockRejectedValue({ error: 'NOT_FOUND', message: 'Сообщение не найдено' })
    await assistant.sendFeedback(99, 'up')
    expect(assistant.myFeedback[99]).toBeUndefined()
    expect(assistant.error).toBe('Сообщение не найдено')
  })

  it('send откатывает оптимистичное сообщение при ошибке и не трогает error для AI_DISABLED', async () => {
    setActiveCompany()
    api.sendAssistantMessage.mockRejectedValue({ error: 'AI_DISABLED', message: 'AI выключен' })
    await assistant.send('Привет')
    expect(assistant.messages).toEqual([])
    expect(assistant.disabled).toBe(true)
    expect(assistant.error).toBeNull()
  })

  it('send откатывает сообщение и показывает текст ошибки для прочих сбоев', async () => {
    setActiveCompany()
    api.sendAssistantMessage.mockRejectedValue({ error: 'INTERNAL_ERROR', message: 'Сервер недоступен' })
    await assistant.send('Привет')
    expect(assistant.messages).toEqual([])
    expect(assistant.error).toBe('Сервер недоступен')
    expect(assistant.disabled).toBe(false)
    expect(assistant.unavailable).toBe(false)
  })

  it('send игнорирует пустой текст и параллельные отправки', async () => {
    setActiveCompany()
    api.sendAssistantMessage.mockResolvedValue({ text: 'ok' })
    await assistant.send('   ')
    expect(api.sendAssistantMessage).not.toHaveBeenCalled()
  })

  it('reset очищает состояние', async () => {
    setActiveCompany()
    api.getAssistantHistory.mockResolvedValue([{ id: 1, role: 'user', text: 'привет', created_at: '2026-07-07T10:00:00Z' }])
    await assistant.fetchHistory()
    assistant.reset()
    expect(assistant.messages).toEqual([])
    expect(assistant.loaded).toBe(false)
  })
})
