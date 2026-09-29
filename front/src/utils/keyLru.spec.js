import { describe, it, expect, vi } from 'vitest'
import { createKeyLru } from './keyLru.js'

describe('createKeyLru', () => {
  it('вытесняет самый давний ключ, повторное обращение поднимает', () => {
    const evicted = vi.fn()
    const lru = createKeyLru(2, evicted)
    lru.touch(1)
    lru.touch(2)
    lru.touch(1)
    lru.touch(3)
    expect(evicted).toHaveBeenCalledTimes(1)
    expect(evicted).toHaveBeenCalledWith(2)
  })

  it('после clear счёт начинается заново', () => {
    const evicted = vi.fn()
    const lru = createKeyLru(1, evicted)
    lru.touch(1)
    lru.clear()
    lru.touch(2)
    expect(evicted).not.toHaveBeenCalled()
  })
})
