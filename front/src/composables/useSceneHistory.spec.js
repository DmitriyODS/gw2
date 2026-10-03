import { describe, expect, it } from 'vitest'
import { useSceneHistory } from './useSceneHistory.js'

describe('useSceneHistory', () => {
  it('отмена возвращает прежнюю сцену, повтор — отменённую', () => {
    const h = useSceneHistory()
    const a = { objects: [] }
    const b = { objects: [{ id: '1' }] }
    h.push(a)
    expect(h.canUndo.value).toBe(true)
    expect(h.undo(b)).toEqual(a)
    expect(h.canRedo.value).toBe(true)
    expect(h.redo(a)).toEqual(b)
  })

  it('новая правка сбрасывает повтор', () => {
    const h = useSceneHistory()
    h.push({ v: 1 })
    h.undo({ v: 2 })
    h.push({ v: 1 })
    expect(h.canRedo.value).toBe(false)
  })

  it('пустая история ничего не возвращает', () => {
    expect(useSceneHistory().undo({})).toBeNull()
  })
})
