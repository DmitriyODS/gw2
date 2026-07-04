import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// Стор задач в мутациях сети не трогает; мок API — на случай импорта.
vi.mock('@/api/tasks.js', () => ({
  getTasks: vi.fn(() => Promise.resolve({ tasks: [], total: 0 })),
}))

import { useTasksStore } from './tasks.js'

describe('tasks store — оптимистичные обновления', () => {
  let store
  beforeEach(() => {
    setActivePinia(createPinia())
    store = useTasksStore()
  })

  // Инвариант CLAUDE.md: чужой личный цвет не должен затираться сокет-патчем.
  // Бэкенд вырезает color из broadcast (dto.TaskBroadcast) — патч приходит БЕЗ
  // color, а merge {...old, ...patch} обязан сохранить локальный цвет.
  it('patchTask сохраняет локальный color, если патч его не содержит', () => {
    store.tasks = [{ id: 1, title: 'A', color: 'teal' }]
    store.patchTask({ id: 1, title: 'A2' })
    expect(store.tasks[0].color).toBe('teal')
    expect(store.tasks[0].title).toBe('A2')
  })

  it('patchTask не вставляет отсутствующую задачу (защита от пустых карточек)', () => {
    store.tasks = [{ id: 1 }]
    store.patchTask({ id: 999, title: 'ghost' })
    expect(store.tasks.map((t) => t.id)).toEqual([1])
  })

  it('patchTask синхронит открытую модалку', () => {
    store.tasks = [{ id: 1, title: 'A' }]
    store.activeTask = { id: 1, title: 'A' }
    store.patchTask({ id: 1, title: 'B' })
    expect(store.activeTask.title).toBe('B')
  })

  describe('addTaskFromSocket', () => {
    it('добавляет новую активную задачу на вкладке «active»', () => {
      store.filters.tab = 'active'
      store.addTaskFromSocket({ id: 2, is_archived: false })
      expect(store.tasks.map((t) => t.id)).toEqual([2])
    })

    it('дедуплицирует: повторное событие обновляет, а не дублирует', () => {
      store.filters.tab = 'active'
      store.tasks = [{ id: 2, title: 'old' }]
      store.addTaskFromSocket({ id: 2, title: 'new' })
      expect(store.tasks).toHaveLength(1)
      expect(store.tasks[0].title).toBe('new')
    })

    it('не добавляет новую задачу на не-активной вкладке', () => {
      store.filters.tab = 'archive'
      store.addTaskFromSocket({ id: 3, is_archived: false })
      expect(store.tasks).toHaveLength(0)
    })
  })

  describe('setFavorite (учитывает вкладку)', () => {
    it('снятие звезды на вкладке «Избранное» убирает карточку', () => {
      store.filters.tab = 'favorites'
      store.tasks = [{ id: 1, is_favorite: true }]
      store.setFavorite(1, false)
      expect(store.tasks).toHaveLength(0)
    })

    it('на вкладке «active» карточка остаётся, флаг меняется', () => {
      store.filters.tab = 'active'
      store.tasks = [{ id: 1, is_favorite: true }]
      store.setFavorite(1, false)
      expect(store.tasks[0].is_favorite).toBe(false)
    })
  })

  describe('archive / restore', () => {
    it('archiveTask на вкладке «active» удаляет из списка', () => {
      store.filters.tab = 'active'
      store.tasks = [{ id: 1 }]
      store.archiveTask(1, '2026-07-04')
      expect(store.tasks).toHaveLength(0)
    })

    it('archiveTask на вкладке «archive» помечает флаг', () => {
      store.filters.tab = 'archive'
      store.tasks = [{ id: 1, is_archived: false }]
      store.archiveTask(1, '2026-07-04')
      expect(store.tasks[0].is_archived).toBe(true)
    })

    it('restoreTask на вкладке «archive» удаляет из списка', () => {
      store.filters.tab = 'archive'
      store.tasks = [{ id: 1, is_archived: true }]
      store.restoreTask(1)
      expect(store.tasks).toHaveLength(0)
    })
  })

  describe('active users', () => {
    it('addActiveUser дедуплицирует по id', () => {
      store.tasks = [{ id: 1, active_users: [] }]
      store.addActiveUser(1, { id: 3, fio: 'И' })
      store.addActiveUser(1, { id: 3, fio: 'И' })
      expect(store.tasks[0].active_users).toHaveLength(1)
    })

    it('removeActiveUser убирает по id', () => {
      store.tasks = [{ id: 1, active_users: [{ id: 3 }, { id: 4 }] }]
      store.removeActiveUser(1, 3)
      expect(store.tasks[0].active_users.map((u) => u.id)).toEqual([4])
    })
  })

  describe('applyCommentSocket (идемпотентность)', () => {
    it('new не дублирует уже известный комментарий', () => {
      store.commentsByTask[1] = [{ id: 10, text: 'a' }]
      store.applyCommentSocket('new', { task_id: 1, id: 10, text: 'a' })
      store.applyCommentSocket('new', { task_id: 1, id: 11, text: 'b' })
      expect(store.commentsByTask[1].map((c) => c.id)).toEqual([10, 11])
    })

    it('игнорирует событие для задачи не в кэше', () => {
      store.applyCommentSocket('new', { task_id: 42, id: 1 })
      expect(store.commentsByTask[42]).toBeUndefined()
    })

    it('deleted убирает комментарий по comment_id', () => {
      store.commentsByTask[1] = [{ id: 10 }, { id: 11 }]
      store.applyCommentSocket('deleted', { task_id: 1, comment_id: 10 })
      expect(store.commentsByTask[1].map((c) => c.id)).toEqual([11])
    })
  })
})
