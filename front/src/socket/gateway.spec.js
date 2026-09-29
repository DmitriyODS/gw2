import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { GatewaySocket } from './gateway.js'

class FakeWS {
  static OPEN = 1
  static all = []
  constructor(url) {
    this.url = url
    this.readyState = 0
    this.sent = []
    FakeWS.all.push(this)
  }
  send(frame) { this.sent.push(JSON.parse(frame)) }
  close() { this.onclose?.() }
  // Сервер: принять или отвергнуть авторизацию.
  accept() { this.readyState = FakeWS.OPEN; this.onmessage({ data: '{"event":"_connected"}' }) }
  reject(code = 'AUTH_FAILED') {
    this.onmessage({ data: JSON.stringify({ event: '_error', data: { code } }) })
    this.onclose()
  }
}

const last = () => FakeWS.all[FakeWS.all.length - 1]
const setHidden = (hidden) => Object.defineProperty(document, 'hidden', { value: hidden, configurable: true })
const setOnline = (online) => Object.defineProperty(navigator, 'onLine', { value: online, configurable: true })

describe('GatewaySocket: переподключение', () => {
  let sock

  beforeEach(() => {
    vi.useFakeTimers()
    FakeWS.all = []
    globalThis.WebSocket = FakeWS
    setHidden(false)
    setOnline(true)
  })

  afterEach(() => {
    sock?.disconnect()
    vi.useRealTimers()
  })

  it('отказ авторизации: сначала новый токен, потом попытка с ним', async () => {
    const refreshAuth = vi.fn().mockResolvedValue('fresh')
    sock = new GatewaySocket('ws://x', { auth: { token: 'stale' }, refreshAuth })
    last().onopen()
    expect(last().sent[0].data.token).toBe('stale')
    last().reject()

    await vi.advanceTimersByTimeAsync(1000)
    expect(refreshAuth).toHaveBeenCalledTimes(1)
    expect(FakeWS.all).toHaveLength(2)
    last().onopen()
    expect(last().sent[0].data.token).toBe('fresh')
  })

  it('придержанные события доходят после готовности обработчиков, по порядку', async () => {
    sock = new GatewaySocket('ws://x', { auth: { token: 'a' } })
    let ready
    sock.holdEvents(new Promise((r) => { ready = r }))
    const got = []
    const onConnect = vi.fn()
    sock.on('connect', onConnect)
    last().onopen()
    last().accept()
    expect(onConnect).toHaveBeenCalledTimes(1)
    last().onmessage({ data: JSON.stringify({ event: 'x:1', data: 1 }) })
    last().onmessage({ data: JSON.stringify({ event: 'x:2', data: 2 }) })
    sock.on('x:1', (d) => got.push(d))
    sock.on('x:2', (d) => got.push(d))
    expect(got).toEqual([])
    ready()
    await Promise.resolve()
    expect(got).toEqual([1, 2])
    last().onmessage({ data: JSON.stringify({ event: 'x:1', data: 3 }) })
    expect(got).toEqual([1, 2, 3])
  })

  it('новый токен уходит живому соединению кадром auth, без переподключения', () => {
    sock = new GatewaySocket('ws://x', { auth: { token: 'a' } })
    last().onopen()
    last().accept()
    sock.setToken('b')
    expect(FakeWS.all).toHaveLength(1)
    expect(last().sent.at(-1)).toEqual({ event: 'auth', data: { token: 'b' } })
  })

  it('токен до подключения кадром не шлётся, но идёт в первую авторизацию', () => {
    sock = new GatewaySocket('ws://x', { auth: { token: 'a' } })
    sock.setToken('b')
    expect(last().sent).toHaveLength(0)
    last().onopen()
    expect(last().sent[0].data.token).toBe('b')
  })

  it('сон: соединение закрыто и не восстанавливается до пробуждения', async () => {
    sock = new GatewaySocket('ws://x', { auth: { token: 'a' } })
    last().onopen()
    last().accept()
    sock.sleep()
    await vi.advanceTimersByTimeAsync(120_000)
    expect(FakeWS.all).toHaveLength(1)
    expect(sock.connected).toBe(false)

    sock.wake()
    expect(FakeWS.all).toHaveLength(2)
    last().onopen()
    expect(last().sent[0]).toEqual({ event: 'auth', data: { token: 'a' } })
  })

  it('обычный обрыв токен не трогает', async () => {
    const refreshAuth = vi.fn()
    sock = new GatewaySocket('ws://x', { auth: { token: 't' }, refreshAuth })
    last().onopen()
    last().accept()
    last().close()
    await vi.advanceTimersByTimeAsync(1000)
    expect(refreshAuth).not.toHaveBeenCalled()
    expect(FakeWS.all).toHaveLength(2)
  })

  it('без сети не пробует, пока не вернётся online', async () => {
    sock = new GatewaySocket('ws://x', { auth: { token: 't' } })
    last().accept()
    setOnline(false)
    last().close()
    await vi.advanceTimersByTimeAsync(120_000)
    expect(FakeWS.all).toHaveLength(1)

    setOnline(true)
    window.dispatchEvent(new Event('online'))
    expect(FakeWS.all).toHaveLength(2)
  })

  it('на скрытой вкладке ждёт дольше, а возврат на вкладку пробует сразу', async () => {
    sock = new GatewaySocket('ws://x', { auth: { token: 't' } })
    setHidden(true)
    // Серия неудач раскручивает задержку за пределы видимого потолка (5 с).
    for (let i = 0; i < 5; i++) {
      last().close()
      await vi.advanceTimersByTimeAsync(2 ** i * 1000)
    }
    const before = FakeWS.all.length
    last().close()
    await vi.advanceTimersByTimeAsync(10_000)
    expect(FakeWS.all).toHaveLength(before)

    setHidden(false)
    document.dispatchEvent(new Event('visibilitychange'))
    expect(FakeWS.all).toHaveLength(before + 1)
  })

  it('очередь офлайн-кадров ограничена', () => {
    sock = new GatewaySocket('ws://x', { auth: { token: 't' } })
    for (let i = 0; i < 500; i++) sock.emit('typing', { i })
    last().onopen()
    last().accept()
    const typing = last().sent.filter((f) => f.event === 'typing')
    expect(typing).toHaveLength(200)
    expect(typing[typing.length - 1].data.i).toBe(499)
  })
})
