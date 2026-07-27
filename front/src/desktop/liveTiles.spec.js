import { describe, it, expect } from 'vitest'
import { tileFaces, plural } from './liveTiles.js'

const ctx = (over = {}) => ({
  data: {}, messenger: null, portal: null, pets: null, units: null, auth: null, ...over,
})

describe('грани живых плиток', () => {
  it('задачи: счётчик, ближайший дедлайн и активный юнит', () => {
    const faces = tileFaces('tasks', ctx({
      data: { tasks: { total: 3, items: [{ name: 'Отчёт', deadline: '2000-01-01T00:00:00Z' }] } },
      units: { activeUnit: { task_name: 'Вёрстка' } },
    }))
    expect(faces[0]).toMatchObject({ value: '3', label: 'задачи в работе' })
    // Дедлайн в прошлом — тревожная грань.
    expect(faces[1]).toMatchObject({ value: 'Просрочено', label: 'Отчёт', tone: 'alert' })
    expect(faces[2]).toMatchObject({ value: 'Идёт работа', label: 'Вёрстка' })
  })

  it('ежедневник: сколько дел осталось и какое ближайшее', () => {
    const faces = tileFaces('diaries', ctx({
      data: { diaries: { total: 2, items: [{ title: 'Позвонить в банк', start_min: 570 }] } },
    }))
    expect(faces[0]).toMatchObject({ value: '2', label: 'дела на сегодня' })
    expect(faces[1]).toMatchObject({ value: 'в 09:30', label: 'Позвонить в банк' })
  })

  it('пустой день показывает, что дел не осталось', () => {
    expect(tileFaces('diaries', ctx({ data: { diaries: { total: 0, items: [] } } })))
      .toEqual([{ key: 'empty', value: 'Всё сделано', label: 'На сегодня дел не осталось', tone: null }])
  })

  it('питомец: болезнь идёт первой гранью и помечена тревожной', () => {
    const faces = tileFaces('pets', ctx({
      pets: { pet: { name: 'Пушок', sick: true, runaway_in_days: 3, kudos: 42 } },
    }))
    expect(faces[0].tone).toBe('alert')
    expect(faces[0].label).toContain('3 дн')
    expect(faces.at(-1)).toMatchObject({ value: '42', label: 'кудоса' })
  })

  it('без данных раздела граней нет — плитка остаётся обычной', () => {
    expect(tileFaces('notes', ctx())).toEqual([])
    expect(tileFaces('settings', ctx())).toEqual([])
  })

  it('склонение числительных', () => {
    expect(plural(1, 'задача', 'задачи', 'задач')).toBe('задача')
    expect(plural(3, 'задача', 'задачи', 'задач')).toBe('задачи')
    expect(plural(11, 'задача', 'задачи', 'задач')).toBe('задач')
    expect(plural(21, 'задача', 'задачи', 'задач')).toBe('задача')
  })
})
