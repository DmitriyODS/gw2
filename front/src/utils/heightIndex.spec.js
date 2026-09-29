import { describe, it, expect } from 'vitest'
import { HeightIndex } from './heightIndex.js'

describe('HeightIndex', () => {
  it('смещения по оценке и по замерам', () => {
    const idx = new HeightIndex(() => 10)
    idx.setKeys(['a', 'b', 'c'])
    expect(idx.total).toBe(30)
    expect(idx.set('b', 25, 1)).toBe(15)
    expect(idx.offsetOf(2)).toBe(35)
    expect(idx.total).toBe(45)
    expect(idx.set('b', 25, 1)).toBe(0)
  })

  it('строка по смещению — двоичным поиском', () => {
    const idx = new HeightIndex(() => 10)
    idx.setKeys(['a', 'b', 'c', 'd'])
    expect(idx.indexAt(0)).toBe(0)
    expect(idx.indexAt(9)).toBe(0)
    expect(idx.indexAt(10)).toBe(1)
    expect(idx.indexAt(35)).toBe(3)
    expect(idx.indexAt(999)).toBe(3)
  })

  it('замер переживает вставку строк сверху', () => {
    const idx = new HeightIndex(() => 10)
    idx.setKeys(['b', 'c'])
    idx.set('c', 40, 1)
    idx.setKeys(['x', 'y', 'b', 'c'])
    expect(idx.offsetOf(3)).toBe(30)
    expect(idx.total).toBe(70)
  })

  it('prune забывает исчезнувшие строки', () => {
    const idx = new HeightIndex(() => 10)
    idx.setKeys(['a', 'b'])
    idx.set('a', 20, 0)
    idx.set('b', 20, 1)
    idx.setKeys(['b'])
    idx.prune()
    expect(idx.measured.has('a')).toBe(false)
    expect(idx.measured.get('b')).toBe(20)
  })
})
