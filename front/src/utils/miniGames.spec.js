import { describe, it, expect } from 'vitest'
import {
  distance, isInHitZone, isInGreenZone, pingPongPercent, createRubTracker,
} from './miniGames.js'

describe('miniGames.distance/isInHitZone', () => {
  it('считает евклидово расстояние', () => {
    expect(distance(0, 0, 3, 4)).toBe(5)
  })

  it('попадание внутри радиуса — true, снаружи — false', () => {
    expect(isInHitZone(10, 10, 10, 14, 5)).toBe(true)
    expect(isInHitZone(10, 10, 10, 20, 5)).toBe(false)
  })
})

describe('miniGames.isInGreenZone', () => {
  it('маркер внутри зоны', () => {
    expect(isInGreenZone(50, 40, 20)).toBe(true) // зона 40..60
    expect(isInGreenZone(39, 40, 20)).toBe(false)
    expect(isInGreenZone(61, 40, 20)).toBe(false)
    expect(isInGreenZone(40, 40, 20)).toBe(true)
    expect(isInGreenZone(60, 40, 20)).toBe(true)
  })
})

describe('miniGames.pingPongPercent', () => {
  it('треугольная волна 0..100..0 за период', () => {
    expect(pingPongPercent(0, 1000)).toBe(0)
    expect(pingPongPercent(250, 1000)).toBe(50)
    expect(pingPongPercent(500, 1000)).toBe(100)
    expect(pingPongPercent(750, 1000)).toBe(50)
    expect(pingPongPercent(1000, 1000)).toBe(0)
  })
})

describe('miniGames.createRubTracker', () => {
  it('копит дистанцию и завершает цикл на пороге', () => {
    const t = createRubTracker(100)
    expect(t.add(30, 0).completed).toBe(false)
    expect(t.add(0, 30).completed).toBe(false)
    expect(t.progress).toBeCloseTo(60)
    const r = t.add(50, 0) // 110 ≥ 100 — цикл завершён, остаток 10 переносится
    expect(r.completed).toBe(true)
    expect(t.progress).toBeCloseTo(10)
  })

  it('reset обнуляет прогресс цикла', () => {
    const t = createRubTracker(100)
    t.add(70, 0)
    t.reset()
    expect(t.progress).toBe(0)
  })

  it('отрицательные смещения считаются по модулю (трение туда-обратно)', () => {
    const t = createRubTracker(100)
    expect(t.add(-60, 0).completed).toBe(false)
    expect(t.add(-60, 0).completed).toBe(true)
  })
})
