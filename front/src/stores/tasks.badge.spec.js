import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from './auth.js'
import { useTasksStore } from './tasks.js'

const ME = 5

describe('бейдж «моих задач»: какие события его касаются', () => {
  let tasks

  beforeEach(() => {
    setActivePinia(createPinia())
    useAuthStore().applySession({ access_token: 't', user_id: ME, company_id: 1 })
    tasks = useTasksStore()
    tasks.tasks = [
      { id: 10, company_id: 1, responsible_user_id: ME },
      { id: 11, company_id: 1, responsible_user_id: 7 },
    ]
  })

  it('другая компания — никогда', () => {
    expect(tasks.affectsMyCount({ id: 99, company_id: 2, responsible_user_id: ME })).toBe(false)
  })

  it('новая задача: только если назначена на меня', () => {
    expect(tasks.affectsMyCount({ id: 20, company_id: 1, responsible_user_id: ME })).toBe(true)
    expect(tasks.affectsMyCount({ id: 21, company_id: 1, responsible_user_id: 7 })).toBe(false)
    expect(tasks.affectsMyCount({ id: 22, company_id: 1, responsible_user_id: null })).toBe(false)
  })

  it('смена ответственного: важен и прежний', () => {
    // Была моей — ушла коллеге.
    expect(tasks.affectsMyCount({ id: 10, company_id: 1, responsible_user_id: 7 })).toBe(true)
    // Была чужой — осталась чужой.
    expect(tasks.affectsMyCount({ id: 11, company_id: 1, responsible_user_id: 8 })).toBe(false)
  })

  it('архив/удаление без ответственного в событии — по задаче из стора', () => {
    expect(tasks.affectsMyCount({ task_id: 10 })).toBe(true)
    expect(tasks.affectsMyCount({ task_id: 11 })).toBe(false)
    // Незагруженная задача: судить не по чему — пересчитываем.
    expect(tasks.affectsMyCount({ task_id: 404 })).toBe(true)
  })
})
