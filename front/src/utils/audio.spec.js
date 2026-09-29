import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

class FakeContext {
  constructor() {
    this.state = 'suspended'
    this.currentTime = 0
    this.resume = vi.fn(() => { this.state = 'running'; return Promise.resolve() })
    this.suspend = vi.fn(() => { this.state = 'suspended'; return Promise.resolve() })
    FakeContext.last = this
  }
}

describe('общий AudioContext', () => {
  let audio

  beforeEach(async () => {
    vi.useFakeTimers()
    vi.resetModules()
    window.AudioContext = FakeContext
    audio = await import('./audio.js')
  })

  afterEach(() => {
    vi.useRealTimers()
    delete window.AudioContext
  })

  it('просыпается на звук и засыпает после него', () => {
    const render = vi.fn()
    audio.playSound(0.5, render)
    const ctx = FakeContext.last
    expect(ctx.resume).toHaveBeenCalled()
    expect(render).toHaveBeenCalledWith(ctx, expect.any(Number))

    vi.advanceTimersByTime(500)
    expect(ctx.suspend).not.toHaveBeenCalled()
    vi.advanceTimersByTime(1000)
    expect(ctx.suspend).toHaveBeenCalledTimes(1)
    expect(ctx.state).toBe('suspended')
  })

  it('новый звук отодвигает сон, а не обрывает себя', () => {
    audio.playSound(0.5, () => {})
    vi.advanceTimersByTime(600)
    audio.playSound(1, () => {})
    const ctx = FakeContext.last
    vi.advanceTimersByTime(800)
    expect(ctx.suspend).not.toHaveBeenCalled()
    vi.advanceTimersByTime(1000)
    expect(ctx.suspend).toHaveBeenCalledTimes(1)
  })

  it('разблокировка жестом не оставляет контекст работать', () => {
    audio.unlockAudio()
    audio.unlockAudio()
    const ctx = FakeContext.last
    expect(ctx.resume).toHaveBeenCalledTimes(1)
    vi.advanceTimersByTime(1000)
    expect(ctx.state).toBe('suspended')
  })
})
