import { describe, it, expect } from 'vitest'
import { num, sumHours, barPercent, toneColor, formatHoursShort, plural } from './tvFormat.js'

describe('tvFormat.num', () => {
  it('парсит числа, нечисла → 0', () => {
    expect(num(5)).toBe(5)
    expect(num('7.5')).toBe(7.5)
    expect(num('abc')).toBe(0)
    expect(num(null)).toBe(0)
    expect(num(undefined)).toBe(0)
    expect(num(Infinity)).toBe(0)
  })
})

describe('tvFormat.sumHours', () => {
  it('суммирует total_hours, пустой список → 0', () => {
    expect(sumHours(null)).toBe(0)
    expect(sumHours([])).toBe(0)
    expect(sumHours([{ total_hours: 2 }, { total_hours: '3.5' }, { total_hours: 'x' }])).toBe(5.5)
  })
})

describe('tvFormat.barPercent', () => {
  it('минимум 6% при валидном max, 0 при max=0', () => {
    expect(barPercent(0, 100)).toBe(6)
    expect(barPercent(50, 100)).toBe(50)
    expect(barPercent(100, 100)).toBe(100)
    expect(barPercent(1, 0)).toBe(0)
  })
})

describe('tvFormat.toneColor', () => {
  it('маппит тон в CSS-токен, неизвестный → primary', () => {
    expect(toneColor('success')).toBe('var(--color-success)')
    expect(toneColor('error')).toBe('var(--color-error)')
    expect(toneColor('bogus')).toBe('var(--color-primary)')
    expect(toneColor(undefined)).toBe('var(--color-primary)')
  })
})

describe('tvFormat.formatHoursShort', () => {
  it('0 и меньше → "0 ч"', () => {
    expect(formatHoursShort(0)).toBe('0 ч')
    expect(formatHoursShort(-5)).toBe('0 ч')
  })

  it('малые значения — часы/минуты', () => {
    expect(formatHoursShort(2)).toBe('2 ч')
    expect(formatHoursShort(0.5)).toBe('30 мин')
    expect(formatHoursShort(2.5)).toBe('2 ч 30 мин')
  })

  it('от 5 рабочих дней переходит в дни (порог = hoursPerDay*5)', () => {
    expect(formatHoursShort(40, 8)).toBe('5 дн')
    expect(formatHoursShort(44, 8)).toBe('5 дн 4 ч')
  })

  it('порог учитывает настраиваемую длину дня', () => {
    // при 4-часовом дне порог = 20 ч
    expect(formatHoursShort(20, 4)).toBe('5 дн')
    // но при 8-часовом те же 20 ч ещё «часами»
    expect(formatHoursShort(20, 8)).toBe('20 ч')
  })

  it('нулевой/битый hoursPerDay откатывается на 8', () => {
    expect(formatHoursShort(40, 0)).toBe('5 дн')
    expect(formatHoursShort(40, 'x')).toBe('5 дн')
  })
})

describe('tvFormat.plural', () => {
  it('русская форма по числу', () => {
    expect(plural(1, 'час', 'часа', 'часов')).toBe('час')
    expect(plural(2, 'час', 'часа', 'часов')).toBe('часа')
    expect(plural(4, 'час', 'часа', 'часов')).toBe('часа')
    expect(plural(5, 'час', 'часа', 'часов')).toBe('часов')
    expect(plural(11, 'час', 'часа', 'часов')).toBe('часов')
    expect(plural(21, 'час', 'часа', 'часов')).toBe('час')
    expect(plural(0, 'час', 'часа', 'часов')).toBe('часов')
  })
})
