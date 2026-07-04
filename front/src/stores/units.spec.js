import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('@/api/units.js', () => ({
  getActiveUnit: vi.fn(() => Promise.resolve(null)),
  stopUnit: vi.fn(() => Promise.resolve({})),
}))

import * as api from '@/api/units.js'
import { useUnitsStore } from './units.js'
import { useTasksStore } from './tasks.js'

describe('units store', () => {
  let units, tasks
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    units = useUnitsStore()
    tasks = useTasksStore()
  })

  it('один активный юнит: setActiveUnit заменяет предыдущий', () => {
    units.setActiveUnit({ id: 1 })
    units.setActiveUnit({ id: 2 })
    expect(units.activeUnit.id).toBe(2)
  })

  it('смена юнита разворачивает модалку (minimized=false), обновление того же — нет', () => {
    units.setActiveUnit({ id: 1 })
    units.minimize()
    expect(units.minimized).toBe(true)
    // тот же юнит (unit:updated) — не разворачиваем
    units.setActiveUnit({ id: 1, note: 'upd' })
    expect(units.minimized).toBe(true)
    // новый юнит — разворачиваем
    units.setActiveUnit({ id: 2 })
    expect(units.minimized).toBe(false)
  })

  it('startUnit оптимистично отражает юнит на карточке задачи', () => {
    tasks.tasks = [{ id: 5, has_units: false, active_users: [] }]
    units.startUnit({ id: 9, task_id: 5, user: { id: 3, fio: 'Иван', avatar_path: null } })
    expect(units.activeUnit.id).toBe(9)
    expect(units.minimized).toBe(false)
    const t = tasks.tasks[0]
    expect(t.has_units).toBe(true)
    expect(t.active_users.map((u) => u.id)).toEqual([3])
  })

  it('stop() дергает API и убирает активного пользователя с задачи', async () => {
    tasks.tasks = [{ id: 5, active_users: [{ id: 3, fio: 'И' }] }]
    units.activeUnit = { id: 9, task_id: 5, user_id: 3 }
    await units.stop()
    expect(api.stopUnit).toHaveBeenCalledWith(9)
    expect(units.activeUnit).toBeNull()
    expect(tasks.tasks[0].active_users).toEqual([])
  })

  it('clearActiveUnit сбрасывает юнит и сворачивание', () => {
    units.setActiveUnit({ id: 1 })
    units.minimize()
    units.clearActiveUnit()
    expect(units.activeUnit).toBeNull()
    expect(units.minimized).toBe(false)
  })
})
