import { describe, expect, it } from 'vitest'
import {
  busyOf, clusters, gapMinutes, gapsOf, mergeSpans, scaleBounds, visibleDays,
} from '@/utils/scheduleLayout.js'

/* Раскладка дня: окна, накладки и занятость. Те же правила считает сервер для
   живой плитки (domain/layout.go) — цифры обязаны совпадать. */

const pair = (start, end, extra = {}) => ({ start_min: start, end_min: end, ...extra })

describe('mergeSpans и busyOf', () => {
  it('накладка не удваивает занятость', () => {
    const items = [pair(600, 700), pair(650, 720)]
    expect(mergeSpans(items)).toEqual([{ s: 600, e: 720 }])
    expect(busyOf(items)).toBe(120)
  })

  it('раздельные занятия складываются', () => {
    expect(busyOf([pair(600, 660), pair(700, 730)])).toBe(90)
  })
})

describe('gapsOf', () => {
  const items = [pair(620, 715), pair(730, 825)]

  it('перемена короче порога окном не считается', () => {
    expect(gapsOf(items, 20)).toEqual([])
    expect(gapMinutes(items, 20)).toBe(0)
  })

  it('со сниженным порогом та же перемена становится окном', () => {
    expect(gapsOf(items, 10)).toEqual([{ s: 715, e: 730 }])
    expect(gapMinutes(items, 10)).toBe(15)
  })

  it('до первого занятия и после последнего окон нет', () => {
    expect(gapsOf([pair(600, 660)], 5)).toEqual([])
  })
})

describe('clusters', () => {
  it('группирует пересекающиеся занятия и оставляет отдельные порознь', () => {
    const groups = clusters([pair(600, 700, { id: 1 }), pair(650, 720, { id: 2 }), pair(800, 900, { id: 3 })])
    expect(groups.map((g) => g.map((i) => i.id))).toEqual([[1, 2], [3]])
  })
})

describe('scaleBounds', () => {
  it('пустое расписание показывает рабочий день', () => {
    expect(scaleBounds([])).toEqual({ from: 480, to: 1200 })
  })

  it('границы округляются до часа с запасом на подписи', () => {
    expect(scaleBounds([pair(620, 715)])).toEqual({ from: 540, to: 780 })
  })

  it('короткий день растягивается до двух часов', () => {
    const bounds = scaleBounds([pair(600, 620)])
    expect(bounds.to - bounds.from).toBeGreaterThanOrEqual(120)
  })
})

describe('visibleDays', () => {
  it('показывает минимум рабочую неделю', () => {
    expect(visibleDays([])).toEqual([0, 1, 2, 3, 4])
  })

  it('добирает дни до последнего занятого', () => {
    expect(visibleDays([{ weekday: 5 }])).toEqual([0, 1, 2, 3, 4, 5])
  })

  it('сегодняшний день показывается, даже если он пуст', () => {
    expect(visibleDays([], 6)).toEqual([0, 1, 2, 3, 4, 5, 6])
  })
})
