import { describe, it, expect, beforeEach, vi } from 'vitest'

describe('стекло: размытие и прозрачность', () => {
  beforeEach(() => {
    vi.resetModules()
    localStorage.clear()
    document.documentElement.removeAttribute('data-opaque')
    document.documentElement.removeAttribute('data-no-blur')
  })

  function mockSystem(reduce) {
    window.matchMedia = vi.fn().mockReturnValue({ matches: reduce, addEventListener: vi.fn() })
  }
  const has = (attr) => document.documentElement.hasAttribute(attr)

  it('по умолчанию стекло целиком', async () => {
    mockSystem(false)
    const t = await import('./transparency.js')
    t.initTransparency()
    expect(has('data-opaque')).toBe(false)
    expect(has('data-no-blur')).toBe(false)
  })

  it('системная настройка выключает и прозрачность, и размытие', async () => {
    mockSystem(true)
    const t = await import('./transparency.js')
    t.initTransparency()
    expect(has('data-opaque')).toBe(true)
    expect(has('data-no-blur')).toBe(true)
  })

  it('без размытия панели остаются прозрачными', async () => {
    mockSystem(false)
    const t = await import('./transparency.js')
    t.setBlur(false)
    expect(has('data-no-blur')).toBe(true)
    expect(has('data-opaque')).toBe(false)
  })

  it('без прозрачности размытие тоже снято', async () => {
    mockSystem(false)
    const t = await import('./transparency.js')
    t.setTransparency(false)
    expect(has('data-opaque')).toBe(true)
    expect(has('data-no-blur')).toBe(true)
    expect(t.blurEnabled.value).toBe(false)
  })

  it('выбор переживает перезагрузку, сброс возвращает системный', async () => {
    mockSystem(false)
    let t = await import('./transparency.js')
    t.setTransparency(false)
    vi.resetModules()
    t = await import('./transparency.js')
    t.initTransparency()
    expect(t.transparencyChoice.value).toBe('off')
    t.setTransparency(null)
    expect(has('data-opaque')).toBe(false)
  })
})
