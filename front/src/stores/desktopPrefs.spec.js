import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

vi.mock('@/api/users.js', () => ({
  getDesktopPrefs: vi.fn(() => Promise.resolve({ prefs: { pinned: ['tasks'], tiles: { notes: 'wide' } } })),
  saveDesktopPrefs: vi.fn(() => Promise.resolve({})),
}))

import * as api from '@/api/users.js'
import { useDesktopPrefsStore } from './desktopPrefs.js'

function freshStore() {
  setActivePinia(createPinia())
  return useDesktopPrefsStore()
}

describe('настройки рабочего стола', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()
    vi.useFakeTimers()
  })

  it('закрепление и открепление раздела', () => {
    const prefs = freshStore()
    expect(prefs.isPinned('tasks')).toBe(false)
    prefs.pin('tasks')
    prefs.pin('tasks') // повторное закрепление не дублирует
    expect(prefs.pinned).toEqual(['tasks'])
    prefs.togglePin('tasks')
    expect(prefs.pinned).toEqual([])
  })

  it('размер плитки берётся из настроек, иначе — размер по умолчанию', () => {
    const prefs = freshStore()
    expect(prefs.tileSize('notes', 'square')).toBe('square')
    prefs.setTileSize('notes', 'wide')
    expect(prefs.tileSize('notes', 'square')).toBe('wide')
  })

  it('изменения уходят на сервер одной отложенной записью', () => {
    const prefs = freshStore()
    prefs.pin('tasks')
    prefs.setTileSize('notes', 'wide')
    prefs.setWallpaper({ gradient: { preset: 'aurora' } })
    expect(api.saveDesktopPrefs).not.toHaveBeenCalled()

    vi.runAllTimers()
    expect(api.saveDesktopPrefs).toHaveBeenCalledTimes(1)
    expect(api.saveDesktopPrefs.mock.calls[0][0]).toMatchObject({
      pinned: ['tasks'],
      tiles: { notes: 'wide' },
    })
  })

  it('настройки переживают перезагрузку и обновляются с сервера', async () => {
    const first = freshStore()
    first.pin('portal')
    vi.runAllTimers()

    // Новая сессия видит кэш мгновенно, затем подтягивает серверную версию.
    const second = freshStore()
    expect(second.pinned).toEqual(['portal'])
    await second.load()
    expect(second.pinned).toEqual(['tasks'])
    expect(second.tileSize('notes')).toBe('wide')
  })

  it('выход из системы очищает настройки', () => {
    const prefs = freshStore()
    prefs.pin('tasks')
    prefs.setWallpaper({ gradient: { preset: 'aurora' } })
    prefs.reset()
    expect(prefs.pinned).toEqual([])
    expect(prefs.wallpaper).toBeNull()
  })
})
