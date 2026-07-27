import { describe, it, expect } from 'vitest'
import { calculate, formatResult } from './calc.js'

describe('калькулятор строки поиска', () => {
  it('считает с приоритетом операций и скобками', () => {
    expect(calculate('2+2*2')).toBe(6)
    expect(calculate('(1200-350)/2')).toBe(425)
    expect(calculate('2^10')).toBe(1024)
    expect(calculate('-5 + 8')).toBe(3)
  })

  it('понимает десятичную запятую и знаки ×/÷', () => {
    expect(calculate('1,5*4')).toBe(6)
    expect(calculate('9 ÷ 3 × 2')).toBe(6)
  })

  it('считает проценты: долю, прибавку и остаток', () => {
    expect(calculate('15% от 2000')).toBe(300)
    expect(calculate('2000 + 15%')).toBe(2300)
    expect(calculate('7 % 3')).toBe(1)
  })

  it('обычный запрос выражением не считает', () => {
    expect(calculate('отчёт по задачам')).toBeNull()
    expect(calculate('2026')).toBeNull()
    expect(calculate('')).toBeNull()
    expect(calculate('2 + отчёт')).toBeNull()
  })

  it('незавершённое выражение не ломает разбор', () => {
    expect(calculate('2+')).toBeNull()
    expect(calculate('(2+2')).toBeNull()
    expect(calculate('5/0')).toBeNull() // Infinity — не показываем
  })

  it('результат читается без хвоста плавающей точки', () => {
    expect(formatResult(calculate('0.1+0.2'))).toBe('0,3')
    // Разряды разделяет неразрывный пробел — сравниваем, нормализовав его.
    expect(formatResult(calculate('1000000/2')).replace(/\u00a0/g, ' ')).toBe('500 000')
  })
})
