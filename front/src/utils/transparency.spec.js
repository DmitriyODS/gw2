import { describe, it, expect, beforeEach, vi } from 'vitest'

describe('transparency', () => {
  beforeEach(() => {
    vi.resetModules()
    localStorage.clear()
    document.documentElement.removeAttribute('data-reduce-transparency')
  })

  function mockSystem(reduce) {
    window.matchMedia = vi.fn().mockReturnValue({ matches: reduce, addEventListener: vi.fn() })
  }

  it('по умолчанию следует системе', async () => {
    mockSystem(true)
    const t = await import('./transparency.js')
    t.initTransparency()
    expect(t.reduceTransparency.value).toBe(true)
    expect(document.documentElement.hasAttribute('data-reduce-transparency')).toBe(true)
  })

  it('явный выбор главнее системы и переживает перезагрузку', async () => {
    mockSystem(true)
    let t = await import('./transparency.js')
    t.initTransparency()
    t.setReduceTransparency(false)
    expect(document.documentElement.hasAttribute('data-reduce-transparency')).toBe(false)

    vi.resetModules()
    t = await import('./transparency.js')
    t.initTransparency()
    expect(t.transparencyChoice.value).toBe('off')
    expect(t.reduceTransparency.value).toBe(false)
  })

  it('сброс возвращает «как в системе»', async () => {
    mockSystem(false)
    const t = await import('./transparency.js')
    t.setReduceTransparency(true)
    expect(t.reduceTransparency.value).toBe(true)
    t.setReduceTransparency(null)
    expect(t.transparencyChoice.value).toBe(null)
    expect(t.reduceTransparency.value).toBe(false)
  })
})
