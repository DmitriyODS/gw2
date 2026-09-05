import { describe, expect, it } from 'vitest'
import {
  itemsOfDay, mondayOf, occursOn, repeatLabel, toMinutes, weekIndex, weekLabel,
} from '@/utils/scheduleCycle.js'

/* Кейсы держатся в паре с back-go/schedule/internal/domain/cycle_test.go:
   правило «идёт ли занятие на этой неделе» считают двое, и расхождение между
   ними означало бы, что плитка и экран показывают разное. */

const schedule = { cycle_weeks: 2, cycle_anchor: '2026-08-31' }

describe('mondayOf', () => {
  it('приводит любую дату недели к её понедельнику', () => {
    expect(mondayOf('2026-08-31').toISOString().slice(0, 10)).toBe('2026-08-31')
    expect(mondayOf('2026-09-05').toISOString().slice(0, 10)).toBe('2026-08-31')
    expect(mondayOf('2026-09-06').toISOString().slice(0, 10)).toBe('2026-08-31')
    expect(mondayOf('2026-09-07').toISOString().slice(0, 10)).toBe('2026-09-07')
  })
})

describe('weekIndex', () => {
  it('нумерует недели цикла в обе стороны от якоря', () => {
    expect(weekIndex('2026-08-31', '2026-08-31', 2)).toBe(1)
    expect(weekIndex('2026-08-31', '2026-09-05', 2)).toBe(1)
    expect(weekIndex('2026-08-31', '2026-09-07', 2)).toBe(2)
    expect(weekIndex('2026-08-31', '2026-09-14', 2)).toBe(1)
    expect(weekIndex('2026-08-31', '2026-08-24', 2)).toBe(2)
  })

  it('работает для цикла из четырёх недель', () => {
    expect(weekIndex('2026-08-31', '2026-09-21', 4)).toBe(4)
    expect(weekIndex('2026-08-31', '2026-09-28', 4)).toBe(1)
    expect(weekIndex('2026-08-31', '2026-08-24', 4)).toBe(4)
  })
})

describe('occursOn', () => {
  it('пустой набор недель означает «каждую неделю»', () => {
    const item = { weeks: [] }
    expect(occursOn(schedule, item, '2026-08-31')).toBe(true)
    expect(occursOn(schedule, item, '2026-09-07')).toBe(true)
  })

  it('числитель и знаменатель — это недели 1 и 2 цикла', () => {
    expect(occursOn(schedule, { weeks: [1] }, '2026-08-31')).toBe(true)
    expect(occursOn(schedule, { weeks: [1] }, '2026-09-07')).toBe(false)
    expect(occursOn(schedule, { weeks: [2] }, '2026-09-07')).toBe(true)
  })

  it('своё правило считается от своей даты и не смотрит на цикл', () => {
    const item = { repeat_every: 3, repeat_from: '2026-08-31', repeat_until: '2026-10-05' }
    expect(occursOn(schedule, item, '2026-08-24')).toBe(false) // раньше начала
    expect(occursOn(schedule, item, '2026-08-31')).toBe(true)
    expect(occursOn(schedule, item, '2026-09-14')).toBe(false)
    expect(occursOn(schedule, item, '2026-09-21')).toBe(true)
    expect(occursOn(schedule, item, '2026-10-12')).toBe(false) // позже конца
  })

  it('без даты конца повтор продолжается', () => {
    const item = { repeat_every: 3, repeat_from: '2026-08-31' }
    expect(occursOn(schedule, item, '2026-10-12')).toBe(true)
  })
})

describe('itemsOfDay', () => {
  const items = [
    { id: 1, weekday: 0, start_min: 620, end_min: 715, weeks: [] },
    { id: 2, weekday: 0, start_min: 500, end_min: 560, weeks: [2] },
    { id: 3, weekday: 1, start_min: 600, end_min: 700, weeks: [] },
  ]

  it('берёт занятия своего дня и своей недели, упорядочивая их по времени', () => {
    const monday = itemsOfDay(schedule, items, '2026-08-31')
    expect(monday.map((i) => i.id)).toEqual([1])

    const nextMonday = itemsOfDay(schedule, items, '2026-09-07')
    expect(nextMonday.map((i) => i.id)).toEqual([2, 1])
  })
})

describe('weekLabel и repeatLabel', () => {
  it('называет неделю своим именем, а без него — номером', () => {
    const named = { ...schedule, week_labels: ['Числитель', ''] }
    expect(weekLabel(named, 1)).toBe('Числитель')
    expect(weekLabel(named, 2)).toBe('Неделя 2')
  })

  it('полный набор недель равен «каждую неделю»', () => {
    expect(repeatLabel(schedule, { weeks: [] })).toBe('Каждую неделю')
    expect(repeatLabel(schedule, { weeks: [1, 2] })).toBe('Каждую неделю')
    expect(repeatLabel(schedule, { weeks: [1] })).toBe('Неделя 1')
    expect(repeatLabel(schedule, { repeat_every: 3 })).toBe('Каждые 3 нед.')
  })
})

describe('toMinutes', () => {
  it('разбирает ЧЧ:ММ и не пропускает мусор', () => {
    expect(toMinutes('10:20')).toBe(620)
    expect(toMinutes('00:00')).toBe(0)
    expect(toMinutes('ерунда')).toBe(0)
  })
})
