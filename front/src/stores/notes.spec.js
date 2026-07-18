import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// Мокаем API-слой — стор тестируем без сети.
vi.mock('@/api/notes.js', () => ({
  getNotes: vi.fn(() => Promise.resolve({ notes: [] })),
  getFolders: vi.fn(() => Promise.resolve({ folders: [], shared: [] })),
  getTags: vi.fn(() => Promise.resolve({ tags: [] })),
  getFolderChildren: vi.fn(() => Promise.resolve({ folders: [] })),
  createNote: vi.fn(() => Promise.resolve({})),
  importNote: vi.fn(() => Promise.resolve({})),
  deleteNote: vi.fn(() => Promise.resolve({})),
}))

// Идентичность текущего пользователя — id 1.
vi.mock('@/stores/auth.js', () => ({
  useAuthStore: () => ({ userId: 1, companyId: null }),
}))

import * as api from '@/api/notes.js'
import { useNotesStore } from './notes.js'

const tile = (id, over = {}) => ({
  id, owner_id: 1, title: `Заметка ${id}`, excerpt: '', tag_ids: [], folder_id: null, archived: false,
  created_at: '2026-07-01T10:00:00Z', updated_at: `2026-07-0${id}T10:00:00Z`, ...over,
})

describe('notes store', () => {
  let store
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    store = useNotesStore()
    store.viewMode = 'hierarchy'
  })

  it('selectFolder передаёт folder_id серверной выборке', () => {
    store.selectFolder(5)
    expect(api.getNotes).toHaveBeenCalledWith(
      expect.objectContaining({ folder_id: 5 }),
      expect.anything(),
    )
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

    it('плитка не из активной папки выпадает из выборки', () => {
      store.activeFolderId = 7
      store.applyNoteSocket('created', tile(1, { folder_id: 7 }))
      expect(store.notes).toHaveLength(1)
      // Заметку перенесли в другую папку — событие несёт новый folder_id.
      store.applyNoteSocket('updated', tile(1, { folder_id: 3 }))
      expect(store.notes).toHaveLength(0)
    })

    it('расшаренная заметка подмешивается в общую кучу верхнего уровня', () => {
      store.applyNoteSocket('created', tile(1, { owner_id: 99, my_access: 'view' }))
      expect(store.notes).toHaveLength(1)
    })

    it('в выбранной папке чужая заметка не появляется', () => {
      store.activeFolderId = 7
      store.applyNoteSocket('created', tile(1, { owner_id: 99, folder_id: 7 }))
      expect(store.notes).toHaveLength(0)
    })
  })

  describe('applyFolderSocket', () => {
    it('идемпотентный upsert своих папок', () => {
      const f = { id: 2, owner_id: 1, parent_id: null, name: 'Работа', notes_count: 0 }
      store.applyFolderSocket('created', f)
      store.applyFolderSocket('created', f)
      store.applyFolderSocket('updated', { ...f, name: 'Дом' })
      expect(store.folders).toHaveLength(1)
      expect(store.folders[0].name).toBe('Дом')
    })

    it('удаление активной папки возвращает на «Все»', () => {
      store.applyFolderSocket('created', { id: 2, owner_id: 1, parent_id: null, name: 'Работа' })
      store.activeFolderId = 2
      store.applyFolderSocket('deleted', { id: 2 })
      expect(store.folders).toHaveLength(0)
      expect(store.activeFolderId).toBe(null)
    })
  })
})
