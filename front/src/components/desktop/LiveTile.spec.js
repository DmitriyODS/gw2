import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import LiveTile from './LiveTile.vue'
import { TILE_PERIOD } from '@/composables/useTileClock.js'

const faces = [
  { key: 'a', value: '1', label: 'первая' },
  { key: 'b', value: '2', label: 'вторая' },
]

// Уходящая грань ещё в DOM, пока идёт анимация, — видна последняя.
const value = (w) => w.findAll('.lt-value').at(-1).text()

describe('LiveTile', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('сразу показывает данные и листает грани, пока видна', async () => {
    const w = mount(LiveTile, { props: { title: 'Задачи', icon: 'task', faces }, attachTo: document.body })
    expect(value(w)).toBe('1')
    globalThis.__intersect(w.element, true)
    vi.advanceTimersByTime(TILE_PERIOD)
    await nextTick()
    expect(value(w)).toBe('2')
    w.unmount()
  })

  it('невидимая плитка стоит — часы не идут вовсе', async () => {
    const spy = vi.spyOn(globalThis, 'setInterval')
    const w = mount(LiveTile, { props: { title: 'Задачи', icon: 'task', faces }, attachTo: document.body })
    globalThis.__intersect(w.element, false)
    vi.advanceTimersByTime(TILE_PERIOD * 3)
    await nextTick()
    expect(value(w)).toBe('1')
    expect(spy).not.toHaveBeenCalled()
    spy.mockRestore()
    w.unmount()
  })

  it('на паузе (курсор, перетаскивание) грань не меняется', async () => {
    const w = mount(LiveTile, { props: { title: 'Задачи', icon: 'task', faces, paused: true }, attachTo: document.body })
    globalThis.__intersect(w.element, true)
    vi.advanceTimersByTime(TILE_PERIOD * 2)
    await nextTick()
    expect(value(w)).toBe('1')
    w.unmount()
  })
})
