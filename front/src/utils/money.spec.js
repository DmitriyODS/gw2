import { describe, expect, it } from 'vitest'
import { formatBytes, formatCount, formatPerMonth, formatPrice, usageRatio } from './money.js'

// toLocaleString разделяет разряды НЕРАЗРЫВНЫМ пробелом — сравниваем по
// обычному, иначе тест ловит невидимую разницу.
const plain = (s) => s.replace(/\u00a0/g, ' ')

// Суммы приходят с сервера в КОПЕЙКАХ — форматтер не должен ошибаться на
// рубль, иначе цена в карточке товара разойдётся с суммой заказа.
describe('formatPrice', () => {
  it('рубли без копеек', () => {
    expect(formatPrice(29900)).toBe('299 руб.')
    expect(plain(formatPrice(238800))).toBe('2 388 руб.')
  })

  it('ноль — бесплатно', () => {
    expect(formatPrice(0)).toBe('Бесплатно')
    expect(formatPrice(0, { free: '—' })).toBe('—')
  })

  it('копейки показываются', () => {
    expect(formatPrice(19950)).toContain('199,50')
  })
})

describe('formatPerMonth', () => {
  it('годовая цена делится на 12', () => {
    expect(formatPerMonth(238800)).toBe('199 руб./мес')
    expect(formatPerMonth(478800)).toBe('399 руб./мес')
  })
})

describe('formatBytes', () => {
  it('переводит байты в человеческие единицы', () => {
    expect(formatBytes(5 * 1024 ** 3)).toBe('5 Гб')
    expect(formatBytes(4.5 * 1024 ** 3)).toBe('4,5 Гб')
  })

  it('безлимит и пустое значение', () => {
    expect(formatBytes(-1)).toBe('∞')
    expect(formatBytes(null)).toBe('—')
  })
})

describe('formatCount', () => {
  it('разряды и безлимит', () => {
    expect(plain(formatCount(10000))).toBe('10 000')
    expect(formatCount(-1)).toBe('∞')
  })
})

describe('usageRatio', () => {
  it('доля заполнения шкалы', () => {
    expect(usageRatio(50, 100)).toBe(0.5)
    // Перерасход не должен рисовать шкалу длиннее самой шкалы.
    expect(usageRatio(150, 100)).toBe(1)
    // Безлимит — шкала всегда пустая.
    expect(usageRatio(50, -1)).toBe(0)
  })
})
