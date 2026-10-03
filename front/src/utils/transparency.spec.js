import { describe, it, expect, beforeEach, vi } from 'vitest'

describe('материал: размытие и прозрачность', () => {
  beforeEach(() => {
    vi.resetModules()
    localStorage.clear()
    document.documentElement.removeAttribute('data-opaque')
    document.documentElement.removeAttribute('data-no-blur')
  })

  const has = (attr) => document.documentElement.hasAttribute(attr)

  it('по умолчанию материал матовый: без прозрачности и размытия', async () => {
    const t = await import('./transparency.js')
    t.initTransparency()
    expect(has('data-opaque')).toBe(true)
    expect(has('data-no-blur')).toBe(true)
  })

  it('стекло включается явным выбором', async () => {
    const t = await import('./transparency.js')
    t.setTransparency(true)
    t.setBlur(true)
    expect(has('data-opaque')).toBe(false)
    expect(has('data-no-blur')).toBe(false)
  })

  it('прозрачность без размытия — панели прозрачные, но не размытые', async () => {
    const t = await import('./transparency.js')
    t.setTransparency(true)
    expect(has('data-opaque')).toBe(false)
    expect(has('data-no-blur')).toBe(true)
  })

  it('без прозрачности размытие снято, даже если его включали', async () => {
    const t = await import('./transparency.js')
    t.setBlur(true)
    expect(t.blurEnabled.value).toBe(false)
    expect(has('data-no-blur')).toBe(true)
  })

  it('выбор переживает перезагрузку, сброс возвращает матовый материал', async () => {
    let t = await import('./transparency.js')
    t.setTransparency(true)
    vi.resetModules()
    t = await import('./transparency.js')
    t.initTransparency()
    expect(t.transparencyChoice.value).toBe('on')
    expect(has('data-opaque')).toBe(false)
    t.setTransparency(null)
    expect(has('data-opaque')).toBe(true)
  })
})
