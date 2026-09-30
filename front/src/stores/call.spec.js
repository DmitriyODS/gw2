import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('@/api/calls.js', () => ({
  getActiveCall: vi.fn(),
  getCallToken: vi.fn(() => Promise.reject(new Error('нет сети'))),
  joinCallByCode: vi.fn(),
}))
const { socket } = vi.hoisted(() => ({ socket: { emit: vi.fn() } }))

vi.mock('@/services/livekit.js', () => ({
  callRoom: {
    disconnect: vi.fn(() => Promise.resolve()),
    connect: vi.fn(() => Promise.resolve()),
    addEventListener: vi.fn(),
    preload: vi.fn(),
    remoteParticipants: () => [],
    mediaState: () => ({ audio: false, video: false, screen: false }),
    connected: false,
    canPlaybackAudio: true,
  },
  parseParticipantMetadata: () => ({}),
}))
vi.mock('@/socket/index.js', () => ({ getSocket: () => socket }))
vi.mock('@/utils/systemNotify.js', () => ({
  requestNotificationPermission: () => Promise.resolve(),
  closeCallNotification: vi.fn(),
}))

import { getActiveCall, getCallToken } from '@/api/calls.js'
import { callRoom } from '@/services/livekit.js'
import { useCallStore } from './call.js'

const ringingCall = { id: 7, status: 'ringing', media: 'audio', participants: [] }

describe('сверка звонка после сна сокета', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('звонящий звонок подхватывается гудками, а не принимается', async () => {
    getActiveCall.mockResolvedValue({ call: null, incoming: ringingCall })
    const store = useCallStore()
    await store.checkRejoin()
    expect(store.phase).toBe('incoming')
    expect(store.call.id).toBe(7)
    // Войти в комнату без принятия — ровно то, от чего защищаемся.
    expect(getCallToken).not.toHaveBeenCalled()
  })

  it('экран входящего не сбрасывается, пока звонок ещё звонит', async () => {
    getActiveCall.mockResolvedValue({ call: null, incoming: ringingCall })
    const store = useCallStore()
    store.handleIncoming(ringingCall)
    await store.checkRejoin()
    expect(store.phase).toBe('incoming')
  })

  it('звонок закончился, пока сокет спал, — экран входящего гаснет', async () => {
    getActiveCall.mockResolvedValue({ call: null })
    const store = useCallStore()
    store.handleIncoming(ringingCall)
    await store.checkRejoin()
    expect(store.phase).toBe('idle')
  })
})

describe('входящий звонок', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('повторная доставка того же звонка не отклоняет его', () => {
    const store = useCallStore()
    store.handleIncoming(ringingCall)
    store.handleIncoming(ringingCall)
    expect(store.phase).toBe('incoming')
    expect(socket.emit).not.toHaveBeenCalled()
  })

  it('чужой звонок во время входящего отклоняется автоматически', () => {
    const store = useCallStore()
    store.handleIncoming(ringingCall)
    store.handleIncoming({ ...ringingCall, id: 8 })
    expect(socket.emit).toHaveBeenCalledWith('call:decline', { call_id: 8 })
    expect(store.call.id).toBe(7)
  })

  it('экран входящего гаснет сам, если исход так и не пришёл', () => {
    vi.useFakeTimers()
    try {
      const store = useCallStore()
      store.handleIncoming(ringingCall)
      vi.advanceTimersByTime(89000)
      expect(store.phase).toBe('incoming')
      vi.advanceTimersByTime(2000)
      expect(store.phase).toBe('idle')
    } finally {
      vi.useRealTimers()
    }
  })

  it('принятый здесь звонок без call:accepted подключается по сверке', async () => {
    const live = { ...ringingCall, status: 'active' }
    getActiveCall.mockResolvedValue({ call: live })
    getCallToken.mockResolvedValueOnce({ call: live, livekit: { url: '/livekit', token: 't' } })
    const store = useCallStore()
    store.handleIncoming(ringingCall)
    store._incomingAt = 0
    store.accept()
    await store.checkRejoin()
    expect(getCallToken).toHaveBeenCalledWith(7)
    expect(callRoom.connect).toHaveBeenCalled()
    expect(store.phase).toBe('active')
  })

  it('принятый на другом устройстве звонок здесь только гасит экран', async () => {
    getActiveCall.mockResolvedValue({ call: { ...ringingCall, status: 'active' } })
    const store = useCallStore()
    store.handleIncoming(ringingCall)
    await store.checkRejoin()
    expect(store.phase).toBe('idle')
    expect(getCallToken).not.toHaveBeenCalled()
  })
})

describe('обрыв связи со звонком', () => {
  const activeCall = { id: 7, status: 'active', media: 'audio', participants: [] }

  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  function inCall() {
    const store = useCallStore()
    store.phase = 'active'
    store.call = activeCall
    return store
  }

  it('возвращается в звонок со свежим токеном', async () => {
    getCallToken.mockResolvedValueOnce({ call: activeCall, livekit: { url: '/livekit', token: 't2' } })
    const store = inCall()
    await store.reconnect()
    expect(callRoom.connect).toHaveBeenCalledWith(expect.objectContaining({ token: 't2' }))
    expect(store.connection).toBe('connected')
    expect(store.phase).toBe('active')
  })

  it('без сети показывает «связь потеряна», а не молчит', async () => {
    const store = inCall()
    await store.reconnect()
    expect(store.connection).toBe('lost')
    expect(store.phase).toBe('active')
  })

  it('звонок успел завершиться — выходим', async () => {
    getCallToken.mockRejectedValueOnce(Object.assign(new Error('завершён'), { code: 'NOT_IN_CALL' }))
    const store = inCall()
    await store.reconnect()
    expect(store.phase).toBe('idle')
  })
})
