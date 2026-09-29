import { describe, expect, it } from 'vitest'
import { safeHref } from './safeHref.js'

describe('safeHref', () => {
  it('пропускает сайты, почту и телефон', () => {
    expect(safeHref('https://a.ru/?q=1')).toBe('https://a.ru/?q=1')
    expect(safeHref('mailto:a@b.ru')).toBe('mailto:a@b.ru')
    expect(safeHref('tel:+7900')).toBe('tel:+7900')
    expect(safeHref('/tasks/1')).toBe('/tasks/1')
  })

  it('дописывает https адресу без схемы', () => {
    expect(safeHref('site.ru')).toBe('https://site.ru')
    expect(safeHref('site.ru:8080/x')).toBe('https://site.ru:8080/x')
  })

  it('отбрасывает исполняемые схемы', () => {
    expect(safeHref('javascript:alert(1)')).toBe('')
    expect(safeHref(' JavaScript:alert(1)')).toBe('')
    expect(safeHref('java\tscript:alert(1)')).toBe('')
    expect(safeHref('data:text/html,x')).toBe('')
    expect(safeHref('vbscript:x')).toBe('')
  })

  it('пустое значение — пустая строка', () => {
    expect(safeHref(null)).toBe('')
    expect(safeHref('  ')).toBe('')
  })
})
