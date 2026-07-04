import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// Мокаем API-слой — стор тестируем без сети.
vi.mock('@/api/diaries.js', () => ({
  getDiaries: vi.fn(() => Promise.resolve({ diaries: [] })),
  getEntries: vi.fn(() => Promise.resolve({ items: [] })),
  createEntry: vi.fn(() => Promise.resolve({})),
  updateEntry: vi.fn(() => Promise.resolve({})),
  setEntryDone: vi.fn(() => Promise.resolve({})),
  reorderEntries: vi.fn(() => Promise.resolve({})),
  moveEntry: vi.fn(() => Promise.resolve({})),
  linkEntryTask: vi.fn(() => Promise.resolve({})),
  deleteEntry: vi.fn(() => Promise.resolve({})),
  bulkDeleteEntries: vi.fn(() => Promise.resolve({})),
  createDiary: vi.fn(() => Promise.resolve({})),
  updateDiary: vi.fn(() => Promise.resolve({})),
  deleteDiary: vi.fn(() => Promise.resolve({})),
}))

import * as api from '@/api/diaries.js'
import { useDiariesStore, dayKey } from './diaries.js'
import { useAuthStore } from './auth.js'

describe('diaries.dayKey', () => {
  it('локальный YYYY-MM-DD с ведущими нулями и на границе года', () => {
    expect(dayKey(new Date(2026, 0, 1))).toBe('2026-01-01')
    expect(dayKey(new Date(2026, 11, 31, 23, 59))).toBe('2026-12-31')
    expect(dayKey(new Date(2025, 8, 7))).toBe('2025-09-07')
  })
})

describe('diaries store', () => {
  let store
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    store = useDiariesStore()
  })

  describe('range (неделя с понедельника)', () => {
    it('day — сутки от начала дня', () => {
      store.view = 'day'
      store.cursor = new Date(2026, 6, 8, 15, 0) // среда
      const { from, to } = store.range
      expect(dayKey(from)).toBe('2026-07-08')
      expect(dayKey(to)).toBe('2026-07-09')
    })

    it('week — с понедельника на 7 дней (воскресенье попадает в свою неделю)', () => {
      store.view = 'week'
      store.cursor = new Date(2026, 6, 12) // воскресенье 12.07.2026
      const { from, to } = store.range
      expect(dayKey(from)).toBe('2026-07-06') // понедельник
      expect(dayKey(to)).toBe('2026-07-13')
    })

    it('month — сетка 6 недель от понедельника недели 1-го числа', () => {
      store.view = 'month'
      store.cursor = new Date(2026, 6, 20) // июль 2026, 1-е — среда
      const { from, to } = store.range
      expect(dayKey(from)).toBe('2026-06-29') // понедельник недели, где 1 июля
      expect(dayKey(to)).toBe('2026-08-10')   // +42 дня
    })
  })

  it('entriesByDay группирует записи по entry_date', () => {
    store.entries = [
      { id: 1, entry_date: '2026-07-08' },
      { id: 2, entry_date: '2026-07-08' },
      { id: 3, entry_date: '2026-07-09' },
    ]
    expect(store.entriesByDay['2026-07-08'].map((e) => e.id)).toEqual([1, 2])
    expect(store.entriesByDay['2026-07-09'].map((e) => e.id)).toEqual([3])
  })

  describe('canToggle / readonly', () => {
    it('владелец (вкладка «Мои») может отмечать', () => {
      store.tab = 'mine'
      store.diaries = [{ id: 1, shared: false }]
      store.selectedId = 1
      expect(store.readonly).toBe(false)
      expect(store.canToggle).toBe(true)
    })

    it('адресат без can_check — read-only, отмечать нельзя', () => {
      store.tab = 'shared'
      store.diaries = [{ id: 1, shared: true, can_check: false }]
      store.selectedId = 1
      expect(store.readonly).toBe(true)
      expect(store.canToggle).toBe(false)
    })

    it('адресат с can_check — read-only, но отмечать можно', () => {
      store.tab = 'shared'
      store.diaries = [{ id: 1, shared: true, can_check: true }]
      store.selectedId = 1
      expect(store.canToggle).toBe(true)
    })
  })

  describe('bumpCounts (через toggleDone, кламп ≥ 0)', () => {
    it('done=true: +1 к выполненным, −1 к активным', async () => {
      store.selectedId = 1
      store.diaries = [{ id: 1, done_count: 2, active_count: 3 }]
      await store.toggleDone(10, true)
      expect(store.diaries[0].done_count).toBe(3)
      expect(store.diaries[0].active_count).toBe(2)
    })

    it('не уводит счётчики в минус', async () => {
      store.selectedId = 1
      store.diaries = [{ id: 1, done_count: 0, active_count: 0 }]
      await store.toggleDone(10, false) // done=false → active+1, done-1(кламп 0)
      expect(store.diaries[0].done_count).toBe(0)
      expect(store.diaries[0].active_count).toBe(1)
    })
  })

  describe('reorderDay (оптимистично + откат)', () => {
    beforeEach(() => {
      store.selectedId = 1
      store.entries = [
        { id: 'a', entry_date: '2026-07-08' },
        { id: 'x', entry_date: '2026-07-09' }, // другой день — не трогается
        { id: 'b', entry_date: '2026-07-08' },
        { id: 'c', entry_date: '2026-07-08' },
      ]
    })

    it('переставляет id в пределах дня, чужой день на месте', async () => {
      await store.reorderDay('2026-07-08', ['c', 'a', 'b'])
      expect(store.entries.map((e) => e.id)).toEqual(['c', 'x', 'a', 'b'])
      expect(api.reorderEntries).toHaveBeenCalledWith(1, '2026-07-08', ['c', 'a', 'b'])
    })

    it('при ошибке api — refetch (возврат серверного порядка) и проброс', async () => {
      api.reorderEntries.mockRejectedValueOnce(new Error('fail'))
      api.getEntries.mockResolvedValueOnce({ items: [{ id: 'server', entry_date: '2026-07-08' }] })
      await expect(store.reorderDay('2026-07-08', ['c', 'a', 'b'])).rejects.toThrow('fail')
      expect(api.getEntries).toHaveBeenCalled()
      expect(store.entries.map((e) => e.id)).toEqual(['server'])
    })
  })

  describe('applyDiarySocket (фильтр по владельцу/вкладке)', () => {
    function setMe(id) {
      useAuthStore().applySession({ access_token: 't', user_id: id })
    }

    it('created своего ежедневника попадает во вкладку «Мои»', () => {
      setMe(7)
      store.tab = 'mine'
      store.applyDiarySocket('created', { id: 1, owner_id: 7, name: 'A' })
      expect(store.diaries.map((d) => d.id)).toEqual([1])
    })

    it('created чужого ежедневника во вкладку «Мои» игнорируется', () => {
      setMe(7)
      store.tab = 'mine'
      store.applyDiarySocket('created', { id: 2, owner_id: 99, name: 'B' })
      expect(store.diaries).toHaveLength(0)
    })

    it('deleted убирает ежедневник и сбрасывает выбор', () => {
      setMe(7)
      store.tab = 'mine'
      store.diaries = [{ id: 1, owner_id: 7 }]
      store.selectedId = 1
      store.applyDiarySocket('deleted', { id: 1 })
      expect(store.diaries).toHaveLength(0)
      expect(store.selectedId).toBeNull()
    })

    it('идемпотентность: повторный created обновляет, а не дублирует', () => {
      setMe(7)
      store.tab = 'mine'
      store.applyDiarySocket('created', { id: 1, owner_id: 7, name: 'A' })
      store.applyDiarySocket('created', { id: 1, owner_id: 7, name: 'A2' })
      expect(store.diaries).toHaveLength(1)
      expect(store.diaries[0].name).toBe('A2')
    })
  })
})
