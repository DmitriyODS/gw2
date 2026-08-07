import { describe, it, expect } from 'vitest'
import { cascadeRect, clampPosition, clampSize, rectForZone, refitRect, snapZoneAt } from './geometry.js'

const area = { x: 12, y: 12, w: 1200, h: 700 }

describe('snapZoneAt', () => {
  it('края экрана дают половины и полный экран', () => {
    expect(snapZoneAt(area.x, 400, area)).toBe('left')
    expect(snapZoneAt(area.x + area.w, 400, area)).toBe('right')
    expect(snapZoneAt(600, area.y, area)).toBe('max')
  })

  it('углы дают четверти', () => {
    expect(snapZoneAt(area.x, area.y + 20, area)).toBe('tl')
    expect(snapZoneAt(area.x + area.w, area.y + 20, area)).toBe('tr')
    expect(snapZoneAt(area.x, area.y + area.h - 20, area)).toBe('bl')
    expect(snapZoneAt(area.x + area.w, area.y + area.h - 20, area)).toBe('br')
  })

  it('середина экрана зоны не даёт', () => {
    expect(snapZoneAt(600, 400, area)).toBeNull()
  })
})

describe('rectForZone', () => {
  it('половины и четверти делят рабочую область без зазоров', () => {
    const left = rectForZone('left', area)
    const right = rectForZone('right', area)
    expect(left.w + right.w).toBe(area.w)
    expect(right.x).toBe(area.x + left.w)

    const tl = rectForZone('tl', area)
    const br = rectForZone('br', area)
    expect(tl.h + br.h).toBe(area.h)
    expect(br.y).toBe(area.y + tl.h)
  })

  it('неизвестная зона — null', () => {
    expect(rectForZone('nope', area)).toBeNull()
  })
})

describe('clampSize / clampPosition', () => {
  it('размер не больше рабочей области и не меньше минимального', () => {
    const big = clampSize({ x: 0, y: 0, w: 5000, h: 5000 }, area)
    expect(big.w).toBe(area.w)
    expect(big.h).toBe(area.h)

    const tiny = clampSize({ x: 0, y: 0, w: 10, h: 10 }, area, { w: 400, h: 300 })
    expect(tiny.w).toBe(400)
    expect(tiny.h).toBe(300)
  })

  it('заголовок окна всегда остаётся на экране', () => {
    const above = clampPosition({ x: 100, y: -500, w: 600, h: 400 }, area)
    expect(above.y).toBe(area.y)

    const below = clampPosition({ x: 100, y: 5000, w: 600, h: 400 }, area)
    expect(below.y).toBe(area.y + area.h - 48)

    const offRight = clampPosition({ x: 5000, y: 100, w: 600, h: 400 }, area)
    expect(offRight.x).toBe(area.x + area.w - 120)
  })
})

describe('cascadeRect / refitRect', () => {
  it('новые окна сдвигаются каскадом и остаются в области', () => {
    const first = cascadeRect(0, { w: 600, h: 400 }, area)
    const second = cascadeRect(1, { w: 600, h: 400 }, area)
    expect(second.x).toBeGreaterThan(first.x)
    expect(second.y).toBeGreaterThan(first.y)
    expect(first.x).toBeGreaterThanOrEqual(area.x)
  })

  it('после сжатия экрана окно вписывается в новую область', () => {
    const small = { x: 12, y: 12, w: 400, h: 300 }
    const fitted = refitRect({ x: 900, y: 900, w: 1200, h: 900 }, small)
    expect(fitted.w).toBeLessThanOrEqual(small.w)
    expect(fitted.h).toBeLessThanOrEqual(small.h)
    expect(fitted.y).toBeLessThanOrEqual(small.y + small.h)
  })
})
