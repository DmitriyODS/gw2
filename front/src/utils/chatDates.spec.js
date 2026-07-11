import { describe, it, expect } from 'vitest'
import { dayLabel, groupMessagesByDay } from './chatDates.js'

const NOW = new Date(2026, 6, 11, 15, 0)

describe('dayLabel', () => {
  it('сегодня и вчера', () => {
    expect(dayLabel(new Date(2026, 6, 11, 0, 5).toISOString(), NOW)).toBe('Сегодня')
    expect(dayLabel(new Date(2026, 6, 10, 23, 59).toISOString(), NOW)).toBe('Вчера')
  })

  it('текущий год — без года, прошлый — с годом', () => {
    expect(dayLabel(new Date(2026, 6, 5).toISOString(), NOW)).toBe('5 июля')
    expect(dayLabel(new Date(2025, 11, 31).toISOString(), NOW)).toBe('31 декабря 2025')
  })
})

describe('groupMessagesByDay', () => {
  it('группирует подряд идущие сообщения по дням', () => {
    const msgs = [
      { id: 1, created_at: new Date(2026, 6, 10, 9, 0).toISOString() },
      { id: 2, created_at: new Date(2026, 6, 10, 18, 0).toISOString() },
      { id: 3, created_at: new Date(2026, 6, 11, 8, 0).toISOString() },
    ]
    const groups = groupMessagesByDay(msgs, NOW)
    expect(groups).toHaveLength(2)
    expect(groups[0].label).toBe('Вчера')
    expect(groups[0].items.map(m => m.id)).toEqual([1, 2])
    expect(groups[1].label).toBe('Сегодня')
    expect(groups[1].items.map(m => m.id)).toEqual([3])
  })

  it('пустой список — пустые группы', () => {
    expect(groupMessagesByDay([], NOW)).toEqual([])
  })
})
