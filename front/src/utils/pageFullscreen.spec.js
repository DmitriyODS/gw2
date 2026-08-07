import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import {
  isPageFullscreen, pageFullscreenSupported, togglePageFullscreen,
} from './pageFullscreen.js'

/* Fullscreen API в jsdom нет — подставляем свои методы: проверяем, что модуль
   зовёт правильный из них и различает состояние. */
const el = document.documentElement

beforeEach(() => {
  el.requestFullscreen = vi.fn(() => Promise.resolve())
  document.exitFullscreen = vi.fn(() => Promise.resolve())
  document.fullscreenElement = null
})

afterEach(() => {
  delete el.requestFullscreen
  delete document.exitFullscreen
  delete document.fullscreenElement
})

describe('полноэкранный режим браузера', () => {
  it('разворачивает страницу, когда режим выключен', async () => {
    expect(isPageFullscreen()).toBe(false)
    await togglePageFullscreen()
    expect(el.requestFullscreen).toHaveBeenCalled()
    expect(document.exitFullscreen).not.toHaveBeenCalled()
  })

  it('сворачивает, когда режим уже включён', async () => {
    document.fullscreenElement = el
    expect(isPageFullscreen()).toBe(true)
    await togglePageFullscreen()
    expect(document.exitFullscreen).toHaveBeenCalled()
    expect(el.requestFullscreen).not.toHaveBeenCalled()
  })

  it('отказ браузера не роняет интерфейс', async () => {
    el.requestFullscreen = vi.fn(() => { throw new Error('gesture required') })
    await expect(togglePageFullscreen()).resolves.toBeUndefined()
  })

  it('без поддержки API кнопку не показываем', () => {
    expect(pageFullscreenSupported()).toBe(true)
    delete el.requestFullscreen
    expect(pageFullscreenSupported()).toBe(false)
  })
})
