import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// Мокаем API-слой — стор тестируем без сети.
vi.mock('@/api/notes.js', () => ({
  getNotes: vi.fn(() => Promise.resolve({ notes: [] })),
  getGroups: vi.fn(() => Promise.resolve({ groups: [] })),
  createNote: vi.fn(() => Promise.resolve({})),
  importNote: vi.fn(() => Promise.resolve({})),
  deleteNote: vi.fn(() => Promise.resolve({})),
  createGroup: vi.fn(() => Promise.resolve({})),
  renameGroup: vi.fn(() => Promise.resolve({})),
  deleteGroup: vi.fn(() => Promise.resolve({})),
}))

import * as api from '@/api/notes.js'
import { useNotesStore } from './notes.js'

const tile = (id, over = {}) => ({
  id, owner_id: 1, title: `Заметка ${id}`, excerpt: '', group_ids: [],
  created_at: '2026-07-01T10:00:00Z', updated_at: `2026-07-0${id}T10:00:00Z`, ...over,
})

describe('notes store', () => {
  let store
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    store = useNotesStore()
  })

  it('selectGroup передаёт group_id серверной выборке', async () => {
    store.selectGroup(5)
    expect(api.getNotes).toHaveBeenCalledWith(
      expect.objectContaining({ group_id: 5 }),
      expect.anything(),
    )
    // Повторный выбор той же группы не дёргает сеть.
    api.getNotes.mockClear()
    store.selectGroup(5)
    expect(api.getNotes).not.toHaveBeenCalled()
  })

  describe('applyNoteSocket', () => {
    it('created/updated идемпотентны — плитка не дублируется', () => {
      store.applyNoteSocket('created', tile(1))
      store.applyNoteSocket('created', tile(1))
      store.applyNoteSocket('updated', tile(1, { title: 'Новое имя' }))
      expect(store.notes).toHaveLength(1)
      expect(store.notes[0].title).toBe('Новое имя')
    })

    it('deleted убирает плитку и переживает повтор', () => {
      store.applyNoteSocket('created', tile(1))
      store.applyNoteSocket('deleted', { id: 1 })
      store.applyNoteSocket('deleted', { id: 1 })
      expect(store.notes).toHaveLength(0)
    })

    it('сортировка по updated_at DESC после обновления', () => {
      store.applyNoteSocket('created', tile(1, { updated_at: '2026-07-01T10:00:00Z' }))
      store.applyNoteSocket('created', tile(2, { updated_at: '2026-07-02T10:00:00Z' }))
      store.applyNoteSocket('updated', tile(1, { updated_at: '2026-07-03T10:00:00Z' }))
      expect(store.notes.map((n) => n.id)).toEqual([1, 2])
    })

    it('плитка не из активной группы выпадает из выборки', () => {
      store.activeGroupId = 7
      store.applyNoteSocket('created', tile(1, { group_ids: [7] }))
      expect(store.notes).toHaveLength(1)
      // Заметку убрали из группы 7 — событие несёт новые group_ids.
      store.applyNoteSocket('updated', tile(1, { group_ids: [3] }))
      expect(store.notes).toHaveLength(0)
    })
  })

  describe('applyGroupSocket', () => {
    it('идемпотентный upsert групп', () => {
      const g = { id: 2, name: 'Работа', notes_count: 0 }
      store.applyGroupSocket('created', g)
      store.applyGroupSocket('created', g)
      store.applyGroupSocket('updated', { ...g, name: 'Дом' })
      expect(store.groups).toHaveLength(1)
      expect(store.groups[0].name).toBe('Дом')
    })

    it('удаление активной группы возвращает на «Все»', () => {
      store.applyGroupSocket('created', { id: 2, name: 'Работа' })
      store.activeGroupId = 2
      store.applyGroupSocket('deleted', { id: 2 })
      expect(store.groups).toHaveLength(0)
      expect(store.activeGroupId).toBe(0)
    })
  })
})
