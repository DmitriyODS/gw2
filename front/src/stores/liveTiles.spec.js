import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('@/api/notes.js', () => ({ getNotesSummary: vi.fn().mockResolvedValue({ total: 1 }) }))
vi.mock('@/api/stats.js', () => ({ getStatsSummary: vi.fn().mockResolvedValue({ week_hours: 2 }) }))

import { getNotesSummary } from '@/api/notes.js'
import { getStatsSummary } from '@/api/stats.js'
import { useLiveTilesStore } from './liveTiles.js'

describe('liveTiles: свежесть по источнику', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
    vi.clearAllMocks()
  })
  afterEach(() => vi.useRealTimers())

  it('свежую сводку не перезапрашивает, устаревшую по времени — да', async () => {
    const live = useLiveTilesStore()
    await live.refresh(['notes', 'stats'])
    expect(getNotesSummary).toHaveBeenCalledTimes(1)
    expect(getStatsSummary).toHaveBeenCalledTimes(1)

    // Через две минуты «часы» устарели, а заметки — нет: их освежает событие.
    vi.advanceTimersByTime(2 * 60_000)
    await live.refresh(['notes', 'stats'])
    expect(getNotesSummary).toHaveBeenCalledTimes(1)
    expect(getStatsSummary).toHaveBeenCalledTimes(2)
  })

  it('событие раздела делает его сводку устаревшей', async () => {
    const live = useLiveTilesStore()
    await live.refresh(['notes'])
    const tick = live.staleTick
    live.invalidate('notes')
    expect(live.staleTick).toBe(tick + 1)
    await live.refresh(['notes'])
    expect(getNotesSummary).toHaveBeenCalledTimes(2)
  })

  it('событие незагруженного раздела ничего не будит', () => {
    const live = useLiveTilesStore()
    live.invalidate('boards')
    expect(live.staleTick).toBe(0)
  })
})
