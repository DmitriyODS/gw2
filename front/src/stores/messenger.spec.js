import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { watchEffect, nextTick } from 'vue'

// Стор зовёт API лениво (api.*), поэтому хватает того, что трогают тесты.
vi.mock('@/api/messenger.js', () => ({ syncConversations: vi.fn() }))

import { syncConversations } from '@/api/messenger.js'
import { useMessengerStore } from './messenger.js'

describe('presence в сторе мессенджера', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('событие о человеке будит только тех, кто его показывает', async () => {
    const m = useMessengerStore()
    let runs = 0
    watchEffect(() => { m.isOnline(5); runs++ })
    expect(runs).toBe(1)

    m.applyPresence({ user_id: 7, online: true })
    await nextTick()
    expect(runs).toBe(1)

    m.applyPresence({ user_id: 5, online: true })
    await nextTick()
    expect(runs).toBe(2)
    expect(m.isOnline(5)).toBe(true)

    // Повтор того же состояния — не событие.
    m.applyPresence({ user_id: 5, online: true })
    await nextTick()
    expect(runs).toBe(2)

    m.applyPresence({ user_id: 5, online: false, last_seen_at: '2026-09-29T10:00:00Z' })
    await nextTick()
    expect(m.isOnline(5)).toBe(false)
    expect(m.lastSeenById[5]).toBe('2026-09-29T10:00:00Z')
  })

  it('счётчик онлайна плитки видит изменения', () => {
    const m = useMessengerStore()
    m.applyPresence({ user_id: 1, online: true })
    m.applyPresence({ user_id: 2, online: true })
    m.applyPresence({ user_id: 1, online: false })
    expect(m.onlineIds.size).toBe(1)
  })
})

describe('память ленты сообщений', () => {
  beforeEach(() => setActivePinia(createPinia()))

  const page = (conv, n) => Array.from({ length: n }, (_, i) => ({ id: conv * 1000 + i }))

  it('у покинутого чата остаётся последняя страница, история — по прокрутке', async () => {
    const m = useMessengerStore()
    m.messagesByConv[1] = page(1, 180)
    m.activeConversationId = 1
    await nextTick()
    m.activeConversationId = 2
    await nextTick()
    expect(m.messagesByConv[1]).toHaveLength(50)
    expect(m.messagesByConv[1].at(-1).id).toBe(1179)
    expect(m.hasMoreHistory(1)).toBe(true)
  })

  it('держит ленты только последних чатов', async () => {
    const m = useMessengerStore()
    for (let id = 1; id <= 22; id++) {
      m.messagesByConv[id] = page(id, 3)
      m.activeConversationId = id
      await nextTick()
    }
    expect(m.messagesByConv[1]).toBeUndefined()
    expect(m.messagesByConv[2]).toBeUndefined()
    expect(m.messagesByConv[3]).toHaveLength(3)
    expect(m.messagesByConv[22]).toHaveLength(3)
  })
})

describe('синхронизация списка диалогов', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('после сна применяется дельта: изменённые обновлены, ушедшие убраны', async () => {
    const m = useMessengerStore()
    syncConversations.mockResolvedValueOnce({
      full: true, cursor: 100,
      conversations: [
        { id: 1, unread_count: 0, last_message_at: '2026-09-29T10:00:00Z' },
        { id: 2, unread_count: 0, last_message_at: '2026-09-29T09:00:00Z' },
      ],
    })
    await m.fetchConversations()
    expect(syncConversations).toHaveBeenLastCalledWith(0, expect.anything())

    syncConversations.mockResolvedValueOnce({
      full: false, cursor: 200,
      conversations: [{ id: 2, unread_count: 3, last_message_at: '2026-09-29T11:00:00Z' }],
      removed: [1],
    })
    await m.resyncConversations()
    expect(syncConversations).toHaveBeenLastCalledWith(100)
    expect(m.conversations.map((c) => c.id)).toEqual([2])
    expect(m.conversations[0].unread_count).toBe(3)

    // Следующая пересинхронизация — от нового курсора.
    syncConversations.mockResolvedValueOnce({ full: false, cursor: 300, conversations: [], removed: [] })
    await m.resyncConversations()
    expect(syncConversations).toHaveBeenLastCalledWith(200)
  })

  it('сервер не ручается за дельту — список заменяется целиком', async () => {
    const m = useMessengerStore()
    syncConversations.mockResolvedValueOnce({ full: true, cursor: 10, conversations: [{ id: 1 }, { id: 2 }] })
    await m.fetchConversations()
    syncConversations.mockResolvedValueOnce({ full: true, cursor: 20, conversations: [{ id: 5 }] })
    await m.resyncConversations()
    expect(m.conversations.map((c) => c.id)).toEqual([5])
  })
})
