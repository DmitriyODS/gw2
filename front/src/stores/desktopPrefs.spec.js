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
    expect(prefs.isPinned('desktop', 'tasks')).toBe(false)
    prefs.pin('desktop', 'tasks')
    prefs.pin('desktop', 'tasks') // повторное закрепление не дублирует
    expect(prefs.pinnedList('desktop')).toEqual(['tasks'])
    prefs.togglePin('desktop', 'tasks')
    expect(prefs.pinnedList('desktop')).toEqual([])
  })

  it('раскладка стола и мобилы независимы', () => {
    const prefs = freshStore()
    prefs.pin('desktop', 'tasks')
    prefs.setTileSize('desktop', 'notes', 'wide')
    prefs.pin('mobile', 'portal')
    prefs.setTileSize('mobile', 'notes', 'square')

    expect(prefs.pinnedList('desktop')).toEqual(['tasks'])
    expect(prefs.pinnedList('mobile')).toEqual(['portal'])
    expect(prefs.tileSize('desktop', 'notes')).toBe('wide')
    expect(prefs.tileSize('mobile', 'notes')).toBe('square')
  })

  it('размер плитки берётся из настроек, иначе — размер по умолчанию', () => {
    const prefs = freshStore()
    expect(prefs.tileSize('desktop', 'notes', 'square')).toBe('square')
    prefs.setTileSize('desktop', 'notes', 'wide')
    expect(prefs.tileSize('desktop', 'notes', 'square')).toBe('wide')
  })

  it('изменения уходят на сервер одной отложенной записью', () => {
    const prefs = freshStore()
    prefs.pin('desktop', 'tasks')
    prefs.setTileSize('desktop', 'notes', 'wide')
    prefs.setWallpaper({ gradient: { preset: 'aurora' } })
    expect(api.saveDesktopPrefs).not.toHaveBeenCalled()

    vi.runAllTimers()
    expect(api.saveDesktopPrefs).toHaveBeenCalledTimes(1)
    expect(api.saveDesktopPrefs.mock.calls[0][0]).toMatchObject({
      layouts: { desktop: { pinned: ['tasks'], tiles: { notes: 'wide' } } },
    })
  })

  it('настройки переживают перезагрузку и обновляются с сервера', async () => {
    const first = freshStore()
    first.pin('desktop', 'portal')
    vi.runAllTimers()

    // Новая сессия видит кэш мгновенно, затем подтягивает серверную версию.
    const second = freshStore()
    expect(second.pinnedList('desktop')).toEqual(['portal'])
    await second.load()
    // Старый плоский формат с сервера трактуется как раскладка стола —
    // мобильная остаётся пустой, человек расставляет её сам.
    expect(second.pinnedList('desktop')).toEqual(['tasks'])
    expect(second.pinnedList('mobile')).toEqual([])
    expect(second.tileSize('desktop', 'notes')).toBe('wide')
  })

  it('выход из системы очищает настройки', () => {
    const prefs = freshStore()
    prefs.pin('desktop', 'tasks')
    prefs.setWallpaper({ gradient: { preset: 'aurora' } })
    prefs.reset()
    expect(prefs.pinnedList('desktop')).toEqual([])
    expect(prefs.wallpaper).toBeNull()
  })
})
