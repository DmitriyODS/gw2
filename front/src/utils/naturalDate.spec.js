import { describe, it, expect } from 'vitest'
import { extractWhen, humanWhen } from './naturalDate.js'

// Понедельник, 27 июля 2026, 10:00 — «сейчас» во всех проверках.
const now = new Date(2026, 6, 27, 10, 0, 0)

const at = (text) => extractWhen(text, now).at
const iso = (d) => (d ? `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}` : null)
const pad = (n) => String(n).padStart(2, '0')

describe('разбор человеческих сроков', () => {
  it('понимает относительные дни со временем', () => {
    expect(iso(at('позвонить в банк завтра в 9'))).toBe('2026-07-28 09:00')
    expect(iso(at('созвон сегодня в 15:30'))).toBe('2026-07-27 15:30')
    expect(iso(at('послезавтра в 8 утра'))).toBe('2026-07-29 08:00')
  })

  it('считает «через N»', () => {
    expect(iso(at('выйти через 20 минут'))).toBe('2026-07-27 10:20')
    expect(iso(at('перезвонить через час'))).toBe('2026-07-27 11:00')
    expect(iso(at('через полчаса'))).toBe('2026-07-27 10:30')
    expect(iso(at('через 2 дня'))).toBe('2026-07-29 10:00')
  })

  it('находит ближайший день недели и дату с месяцем', () => {
    expect(iso(at('в пятницу в 12:00'))).toBe('2026-07-31 12:00')
    expect(iso(at('15 августа в 10:30'))).toBe('2026-08-15 10:30')
    // Прошедшая в этом году дата означает следующий год.
    expect(iso(at('1 марта в 9:00'))).toBe('2027-03-01 09:00')
  })

  it('время суток без часов и вечерние часы', () => {
    expect(iso(at('завтра вечером'))).toBe('2026-07-28 18:00')
    expect(iso(at('сегодня в 7 вечера'))).toBe('2026-07-27 19:00')
  })

  it('время без дня, которое уже прошло, — про завтра', () => {
    expect(iso(at('в 8:00'))).toBe('2026-07-28 08:00')
    expect(iso(at('в 14:00'))).toBe('2026-07-27 14:00')
  })

  it('день без времени назначает утро, а сегодняшнее утро уже позади', () => {
    expect(iso(at('завтра'))).toBe('2026-07-28 09:00')
    expect(iso(at('сегодня'))).toBe('2026-07-27 11:00')
  })

  it('вынимает срок из фразы, оставляя название', () => {
    const r = extractWhen('позвонить в банк завтра в 9:00', now)
    expect(r.rest).toBe('позвонить в банк')
    expect(r.repeat).toBeNull()
  })

  it('распознаёт повторы', () => {
    const daily = extractWhen('зарядка каждый день в 7:00', now)
    expect(daily.repeat).toEqual({ kind: 'daily', interval: 1, days: [] })
    expect(daily.rest).toBe('зарядка')

    const weekly = extractWhen('планёрка каждую пятницу в 10:00', now)
    expect(weekly.repeat).toEqual({ kind: 'weekly', interval: 1, days: [5] })
    expect(iso(weekly.at)).toBe('2026-07-31 10:00')

    expect(extractWhen('отчёт по будням в 18:00', now).repeat.kind).toBe('weekdays')
    expect(extractWhen('оплата каждые 2 недели в 12:00', now).repeat)
      .toEqual({ kind: 'weekly', interval: 2, days: [1] })
  })

  it('без срока фразу не трогает', () => {
    const r = extractWhen('позвонить в банк', now)
    expect(r).toEqual({ at: null, repeat: null, rest: 'позвонить в банк' })
    // Числа в тексте сроком не считаются.
    expect(extractWhen('обновить до версии 1.2', now).at).toBeNull()
    expect(extractWhen('подождать через 20', now).at).toBeNull()
  })

  it('человеческая подпись срока', () => {
    expect(humanWhen(new Date(2026, 6, 27, 14, 30), now)).toBe('сегодня в 14:30')
    expect(humanWhen(new Date(2026, 6, 28, 9, 0), now)).toBe('завтра в 09:00')
    expect(humanWhen(new Date(2026, 7, 12, 10, 0), now)).toContain('авг')
  })
})
