import { describe, it, expect } from 'vitest'
import { clampToViewport, snapX, cornerPosition } from './useDraggable.js'

describe('useDraggable.clampToViewport', () => {
  it('держит позицию внутри вьюпорта с отступом', () => {
    expect(clampToViewport(-50, -50, 64, 64, 16, 400, 800)).toEqual({ x: 16, y: 16 })
    expect(clampToViewport(9999, 9999, 64, 64, 16, 400, 800)).toEqual({ x: 320, y: 720 })
  })

  it('не трогает позицию уже внутри границ', () => {
    expect(clampToViewport(100, 200, 64, 64, 16, 400, 800)).toEqual({ x: 100, y: 200 })
  })
})

describe('useDraggable.snapX', () => {
  it('центр левее середины экрана — прилипает к левому краю', () => {
    expect(snapX(10, 64, 16, 400)).toBe(16)
  })

  it('центр правее середины экрана — прилипает к правому краю', () => {
    expect(snapX(300, 64, 16, 400)).toBe(400 - 64 - 16)
  })
})

describe('useDraggable.cornerPosition', () => {
  it('bottom-left — нижний левый угол', () => {
    expect(cornerPosition('bottom-left', 64, 64, 16, 400, 800)).toEqual({ x: 16, y: 800 - 64 - 16 })
  })

  it('top-right — верхний правый угол', () => {
    expect(cornerPosition('top-right', 64, 64, 16, 400, 800)).toEqual({ x: 400 - 64 - 16, y: 16 })
  })
})
