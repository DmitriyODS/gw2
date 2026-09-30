import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('@/api/messenger.js', () => ({ sendMessage: vi.fn() }))
vi.mock('./messenger.js', () => ({
  useMessengerStore: () => ({ applyIncomingMessage: applied, markRead: vi.fn() }),
}))
vi.mock('./auth.js', () => ({ useAuthStore: () => ({ user: { id: 1 } }) }))
vi.mock('./notifications.js', () => ({ useNotificationsStore: () => ({ error: vi.fn() }) }))

const { applied } = vi.hoisted(() => ({ applied: vi.fn() }))

import { sendMessage } from '@/api/messenger.js'
import { useMessageOutboxStore } from './messageOutbox.js'

const offline = { status: 0, error: 'NETWORK_ERROR', message: 'Нет подключения к интернету' }
const flushPromises = () => new Promise((r) => setTimeout(r, 0))

describe('очередь отправки сообщений', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })
  afterEach(() => vi.useRealTimers())

  it('сообщение сразу в ленте «с часиками», после ответа — уходит из очереди', async () => {
    let resolve
    sendMessage.mockReturnValueOnce(new Promise((r) => { resolve = r }))
    const out = useMessageOutboxStore()
    out.enqueue(5, { text: 'привет' })
    const [draft] = out.messagesFor(5)
    expect(draft.text).toBe('привет')
    expect(draft.outbox).toBe('sending')
    expect(sendMessage.mock.calls[0][1].client_id).toBe(draft.client_id)

    resolve({ id: 100, client_id: draft.client_id })
    await flushPromises()
    expect(out.messagesFor(5)).toHaveLength(0)
    expect(applied).toHaveBeenCalledWith(5, { id: 100, client_id: draft.client_id }, true)
  })

  it('без сети ждёт и дошлёт тем же ключом, не нарушая порядок', async () => {
    vi.useFakeTimers()
    sendMessage.mockRejectedValueOnce(offline)
    const out = useMessageOutboxStore()
    out.enqueue(5, { text: 'первое' })
    out.enqueue(5, { text: 'второе' })
    await vi.advanceTimersByTimeAsync(0)
    expect(sendMessage).toHaveBeenCalledTimes(1)
    expect(out.messagesFor(5).map((m) => m.outbox)).toEqual(['pending', 'pending'])
    const firstKey = sendMessage.mock.calls[0][1].client_id

    sendMessage.mockImplementation((_c, body) => Promise.resolve({ id: Math.random(), client_id: body.client_id }))
    await vi.advanceTimersByTimeAsync(2000)
    expect(sendMessage.mock.calls[1][1]).toMatchObject({ text: 'первое', client_id: firstKey })
    expect(sendMessage.mock.calls[2][1].text).toBe('второе')
    expect(out.messagesFor(5)).toHaveLength(0)
  })

  it('отказ сервера помечает ошибкой, «Повторить» отправляет снова', async () => {
    sendMessage.mockRejectedValueOnce({ status: 403, error: 'TASK_WRONG_COMPANY', message: 'Задача чужой компании' })
    const out = useMessageOutboxStore()
    out.enqueue(5, { text: 'x', task_id: 9 })
    await flushPromises()
    const [failed] = out.messagesFor(5)
    expect(failed.outbox).toBe('failed')
    expect(failed.outbox_error).toBe('Задача чужой компании')

    sendMessage.mockResolvedValueOnce({ id: 7, client_id: failed.client_id })
    out.retry(failed.client_id)
    await flushPromises()
    expect(out.messagesFor(5)).toHaveLength(0)
  })

  it('сокет подтвердил раньше HTTP — черновик исчезает сразу', () => {
    sendMessage.mockReturnValueOnce(new Promise(() => {}))
    const out = useMessageOutboxStore()
    out.enqueue(5, { text: 'x' })
    out.confirm(out.messagesFor(5)[0].client_id)
    expect(out.messagesFor(5)).toHaveLength(0)
  })

  it('очередь переживает перезагрузку, недосланное повторяется', () => {
    sendMessage.mockReturnValueOnce(new Promise(() => {}))
    useMessageOutboxStore().enqueue(5, { text: 'x' })
    setActivePinia(createPinia())
    const [restored] = useMessageOutboxStore().messagesFor(5)
    expect(restored.text).toBe('x')
    expect(restored.outbox).toBe('pending')
  })

  it('выход из аккаунта чистит очередь', () => {
    sendMessage.mockReturnValueOnce(new Promise(() => {}))
    const out = useMessageOutboxStore()
    out.enqueue(5, { text: 'x' })
    out.reset()
    expect(out.messagesFor(5)).toHaveLength(0)
    expect(JSON.parse(localStorage.getItem('gw_msg_outbox'))).toEqual([])
  })
})
