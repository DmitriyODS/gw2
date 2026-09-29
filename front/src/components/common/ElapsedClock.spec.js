import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import ElapsedClock from './ElapsedClock.vue'

describe('ElapsedClock', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-09-29T10:00:00Z'))
  })
  afterEach(() => vi.useRealTimers())

  it('без идущего юнита таймер не заводится', () => {
    const spy = vi.spyOn(globalThis, 'setInterval')
    const w = mount(ElapsedClock, { props: { start: null } })
    expect(spy).not.toHaveBeenCalled()
    expect(w.text()).toBe('00:00:00')
    spy.mockRestore()
  })

  it('тикает раз в секунду и останавливается, когда юнит закончен', async () => {
    const w = mount(ElapsedClock, { props: { start: '2026-09-29T09:59:30Z' } })
    await nextTick()
    expect(w.text()).toBe('00:00:30')
    vi.advanceTimersByTime(2000)
    await nextTick()
    expect(w.text()).toBe('00:00:32')

    const clear = vi.spyOn(globalThis, 'clearInterval')
    await w.setProps({ start: null })
    expect(clear).toHaveBeenCalled()
    clear.mockRestore()
  })
})
