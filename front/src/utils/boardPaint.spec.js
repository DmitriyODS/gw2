import { describe, expect, it } from 'vitest'
import {
  cssFilter, defaultGradient, hasEffects, normalizeEffects, normalizeGradient, withAlpha,
} from './boardPaint.js'

describe('градиент', () => {
  it('нужны хотя бы две точки', () => {
    expect(normalizeGradient(null)).toBeNull()
    expect(normalizeGradient({ stops: [{ color: 'red', at: 0 }] })).toBeNull()
  })

  it('точки сортируются по позиции, а позиции держатся в 0..1', () => {
    const g = normalizeGradient({ type: 'linear', angle: 45, stops: [{ color: 'red', at: 3 }, { color: 'blue', at: -1 }] })
    expect(g.stops.map((s) => s.color)).toEqual(['blue', 'red'])
    expect(g.stops.map((s) => s.at)).toEqual([0, 1])
  })

  it('незнакомый тип считается линейным', () => {
    expect(normalizeGradient({ type: 'конус', stops: [{ color: 'red', at: 0 }, { color: 'blue', at: 1 }] }).type)
      .toBe('linear')
  })

  it('градиент по умолчанию строится от заданного цвета', () => {
    expect(defaultGradient('teal').stops[0].color).toBe('teal')
  })
})

describe('эффекты', () => {
  it('пустые эффекты не хранятся', () => {
    expect(normalizeEffects(null)).toBeNull()
    expect(normalizeEffects({ blur: 0, brightness: 1 })).toBeNull()
    expect(hasEffects(null)).toBe(false)
  })

  it('значения обрезаются по границам', () => {
    const e = normalizeEffects({ blur: 999, hue: -400, grayscale: 5 })
    expect(e.blur).toBe(200)
    expect(e.hue).toBe(-180)
    expect(e.grayscale).toBe(1)
  })

  it('тень получает недостающие поля', () => {
    const e = normalizeEffects({ shadow: {} })
    expect(e.shadow).toMatchObject({ x: 0, y: 4, blur: 8, color: 'ink', opacity: 0.35 })
  })

  it('строка фильтра идёт в том же порядке, что и на сервере', () => {
    const e = normalizeEffects({ blur: 4, brightness: 1.2, saturate: 0, sepia: 0.5 })
    expect(cssFilter(e)).toBe('blur(4px) brightness(1.2) saturate(0) sepia(0.5)')
    expect(cssFilter(null)).toBe('')
  })
})

describe('цвет тени', () => {
  it('hex превращается в rgba с нужной прозрачностью', () => {
    expect(withAlpha('#3b74d6', 0.5)).toBe('rgba(59, 116, 214, 0.5)')
  })

  it('полностью непрозрачный цвет не трогаем', () => {
    expect(withAlpha('#3b74d6', 1)).toBe('#3b74d6')
  })

  it('цвет темы смешивается средствами CSS', () => {
    expect(withAlpha('oklch(0.5 0.1 20)', 0.4)).toContain('color-mix')
  })
})
