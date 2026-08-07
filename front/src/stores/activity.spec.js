import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useActivityStore } from './activity.js'

describe('журнал последних действий', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('складывает действия сверху и знает путь перехода', () => {
    const s = useActivityStore()
    s.record({ section: 'notes', id: 5, title: 'Идеи', path: '/notes/5' })
    s.record({ section: 'tasks', id: 12, title: 'Отчёт', path: '/tasks/12' })

    expect(s.items.map((e) => e.title)).toEqual(['Отчёт', 'Идеи'])
    expect(s.items[0]).toMatchObject({ section: 'tasks', action: 'created', path: '/tasks/12' })
  })

  it('повтор по тому же элементу не плодит строки, а поднимает прежнюю', () => {
    const s = useActivityStore()
    s.record({ section: 'notes', id: 5, title: 'Идеи', path: '/notes/5' })
    s.record({ section: 'tasks', id: 12, title: 'Отчёт', path: '/tasks/12' })
    s.record({ section: 'notes', id: 5, title: 'Идеи и планы', path: '/notes/5' })

    expect(s.items).toHaveLength(2)
    expect(s.items[0].title).toBe('Идеи и планы')
  })

  it('переживает перезагрузку страницы и чистится по команде', () => {
    useActivityStore().record({ section: 'portal', action: 'published', id: 3, title: 'Новости', path: '/portal/3' })

    setActivePinia(createPinia())
    const restored = useActivityStore()
    expect(restored.items[0]).toMatchObject({ section: 'portal', action: 'published', title: 'Новости' })

    restored.clear()
    setActivePinia(createPinia())
    expect(useActivityStore().items).toEqual([])
  })

  it('открытые разделы ложатся в общую ленту без повторов', () => {
    const s = useActivityStore()
    s.recordSection('tasks')
    s.recordSection('notes')
    s.recordSection('tasks')

    expect(s.items.map((e) => e.section)).toEqual(['tasks', 'notes'])
    expect(s.items[0]).toMatchObject({ action: 'opened', path: '' })
  })

  it('разделы и действия — один поток, «очистить» гасит его целиком', () => {
    const s = useActivityStore()
    s.recordSection('tasks')
    s.record({ section: 'notes', id: 5, title: 'Идеи', path: '/notes/5' })
    expect(s.items.map((e) => e.action)).toEqual(['created', 'opened'])

    s.clear()
    setActivePinia(createPinia())
    expect(useActivityStore().items).toEqual([])
  })

  it('прежний формат хранилища (разделы отдельным списком) поднимается в ленту', () => {
    localStorage.setItem('gw_activity', JSON.stringify({
      items: [{ key: 'notes:5', section: 'notes', action: 'created', title: 'Идеи', path: '/notes/5', at: '2026-07-28T10:00:00.000Z' }],
      sections: [{ id: 'tasks', at: '2026-07-28T11:00:00.000Z' }],
    }))

    setActivePinia(createPinia())
    const s = useActivityStore()
    expect(s.items.map((e) => e.section)).toEqual(['tasks', 'notes'])
    expect(s.items[0].action).toBe('opened')
  })

  it('запись без раздела или пути игнорируется', () => {
    const s = useActivityStore()
    s.record({ title: 'ничего', path: '/notes/1' })
    s.record({ section: 'notes', title: 'ничего' })
    expect(s.items).toEqual([])
  })
})
