import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('@/api/calls.js', () => ({
  getActiveCall: vi.fn(),
  getCallToken: vi.fn(() => Promise.reject(new Error('нет сети'))),
  joinCallByCode: vi.fn(),
}))
vi.mock('@/services/livekit.js', () => ({
  callRoom: { disconnect: vi.fn(() => Promise.resolve()), addEventListener: vi.fn(), connected: false },
  parseParticipantMetadata: () => ({}),
}))
vi.mock('@/socket/index.js', () => ({ getSocket: () => null }))

import { getActiveCall, getCallToken } from '@/api/calls.js'
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
