import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { devToolsOn, hideDevTools, tapBuildNumber } from './devTools.js'

describe('скрытый раздел DevTools', () => {
  beforeEach(() => {
    localStorage.clear()
    hideDevTools()
    vi.useFakeTimers()
  })

  afterEach(() => vi.useRealTimers())

  it('открывается пятью быстрыми нажатиями по номеру сборки', () => {
    for (let i = 0; i < 4; i++) expect(tapBuildNumber()).toBe(false)
    expect(devToolsOn.value).toBe(false)

    expect(tapBuildNumber()).toBe(true)
    expect(devToolsOn.value).toBe(true)
  })

  it('редкие нажатия не складываются', () => {
    tapBuildNumber()
    tapBuildNumber()
    vi.advanceTimersByTime(5000) // спустя время счёт начинается заново
    for (let i = 0; i < 4; i++) expect(tapBuildNumber()).toBe(false)
    expect(tapBuildNumber()).toBe(true)
  })

  it('скрытый раздел возвращается тем же способом', () => {
    for (let i = 0; i < 5; i++) tapBuildNumber()
    hideDevTools()
    expect(devToolsOn.value).toBe(false)

    for (let i = 0; i < 5; i++) tapBuildNumber()
    expect(devToolsOn.value).toBe(true)
  })
})
