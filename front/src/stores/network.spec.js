import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useNetworkStore } from './network.js'

describe('состояние сети', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
  })
  afterEach(() => vi.useRealTimers())

  it('нет интернета — сразу «офлайн»', () => {
    const net = useNetworkStore()
    net._setOnline(false)
    expect(net.status).toBe('offline')
  })

  it('короткий обрыв сокета не показывается', () => {
    const net = useNetworkStore()
    net.setSocketActive(true)
    net.setSocketConnected(true)
    net.setSocketConnected(false)
    vi.advanceTimersByTime(2000)
    net.setSocketConnected(true)
    vi.advanceTimersByTime(5000)
    expect(net.status).toBe('online')
    expect(net.restored).toBe(false)
  })

  it('долгий обрыв — «подключение», затем «восстановлено»', () => {
    const net = useNetworkStore()
    net.setSocketActive(true)
    net.setSocketConnected(false)
    vi.advanceTimersByTime(3100)
    expect(net.status).toBe('connecting')
    net.setSocketConnected(true)
    expect(net.status).toBe('online')
    expect(net.restored).toBe(true)
    vi.advanceTimersByTime(3000)
    expect(net.restored).toBe(false)
  })

  it('сеть вернулась, а сервер ещё нет — без ложного «восстановлено»', () => {
    const net = useNetworkStore()
    net.setSocketActive(true)
    net._setOnline(false)
    net._setOnline(true)
    expect(net.status).toBe('connecting')
    expect(net.restored).toBe(false)
  })

  it('упавший запрос без сокета забывается сам', () => {
    const net = useNetworkStore()
    net.reportRequest(false)
    vi.advanceTimersByTime(3100)
    expect(net.status).toBe('connecting')
    vi.advanceTimersByTime(15000)
    expect(net.status).toBe('online')
  })

  it('намеренный сон сокета — не обрыв', () => {
    const net = useNetworkStore()
    net.setSocketActive(true)
    net.setSocketConnected(true)
    net.setSocketActive(false)
    vi.advanceTimersByTime(10000)
    expect(net.status).toBe('online')
  })
})
