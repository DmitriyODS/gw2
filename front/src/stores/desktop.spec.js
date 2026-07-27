import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useDesktopStore } from './desktop.js'
import { appById } from '@/desktop/apps.js'

const AREA = { x: 12, y: 12, w: 1400, h: 800 }
const SCREEN = { x: 0, y: 0, w: 1424, h: 904 }

function freshStore() {
  setActivePinia(createPinia())
  const desktop = useDesktopStore()
  desktop.setScreen(SCREEN)
  desktop.setArea(AREA)
  return desktop
}

describe('оконный менеджер', () => {
  beforeEach(() => { localStorage.clear() })

  it('повторное открытие раздела поднимает уже открытое окно', () => {
    const desktop = freshStore()
    const first = desktop.open('/tasks')
    const again = desktop.open('/tasks')
    expect(desktop.windows).toHaveLength(1)
    expect(again.id).toBe(first.id)
  })

  it('newWindow открывает второе окно того же раздела', () => {
    const desktop = freshStore()
    desktop.open('/tasks')
    desktop.open('/tasks', { newWindow: true })
    expect(desktop.windows).toHaveLength(2)
    expect(desktop.focusedId).toBe(desktop.windows[1].id)
  })

  it('переход внутри окна ведёт историю, назад возвращает прежний путь', () => {
    const desktop = freshStore()
    const win = desktop.open('/notes')
    desktop.navigate(win.id, '/notes/5')
    expect(win.path).toBe('/notes/5')
    expect(desktop.canGoBack(win.id)).toBe(true)
    desktop.back(win.id)
    expect(win.path).toBe('/notes')
    expect(desktop.canGoBack(win.id)).toBe(false)
  })

  it('replace не наращивает историю окна', () => {
    const desktop = freshStore()
    const win = desktop.open('/notes')
    desktop.navigate(win.id, '/notes/5', { replace: true })
    expect(win.history).toEqual(['/notes/5'])
    expect(desktop.canGoBack(win.id)).toBe(false)
  })

  it('открытие пути чужого раздела переключает приложение окна', () => {
    const desktop = freshStore()
    const win = desktop.open('/pets')
    desktop.navigate(win.id, '/pets/bank')
    expect(win.appId).toBe('pets')
    expect(win.path).toBe('/pets/bank')
  })

  it('сворачивание снимает фокус, кнопка панели задач возвращает его', () => {
    const desktop = freshStore()
    const a = desktop.open('/tasks')
    const b = desktop.open('/notes')
    desktop.minimize(b.id)
    expect(desktop.focusedId).toBe(a.id)
    desktop.toggleFromTaskbar(b.id)
    expect(b.minimized).toBe(false)
    expect(desktop.focusedId).toBe(b.id)
    // Повторный клик по активному окну сворачивает его — как в панели задач ОС.
    desktop.toggleFromTaskbar(b.id)
    expect(b.minimized).toBe(true)
  })

  it('разворот занимает весь экран и прячет панель задач, возврат — восстанавливает', () => {
    const desktop = freshStore()
    const win = desktop.open('/tasks')
    const before = { x: win.x, y: win.y, w: win.w, h: win.h }
    desktop.maximize(win.id)
    expect(win.mode).toBe('max')
    // Полный экран — без полей рабочей области.
    expect({ x: win.x, y: win.y, w: win.w, h: win.h }).toEqual(SCREEN)
    expect(desktop.fullscreen).toBe(true)
    desktop.unmaximize(win.id)
    expect(win.mode).toBe('normal')
    expect({ x: win.x, y: win.y, w: win.w, h: win.h }).toEqual(before)
    expect(desktop.fullscreen).toBe(false)
  })

  it('панель задач возвращается, когда полноэкранное окно свернули или закрыли', () => {
    const desktop = freshStore()
    const win = desktop.open('/tasks')
    desktop.maximize(win.id)
    desktop.minimize(win.id)
    expect(desktop.fullscreen).toBe(false)

    desktop.restore(win.id)
    desktop.focus(win.id)
    expect(desktop.fullscreen).toBe(true)

    desktop.close(win.id)
    expect(desktop.fullscreen).toBe(false)
  })

  it('прилипание к половине экрана переживает изменение размеров экрана', () => {
    const desktop = freshStore()
    const win = desktop.open('/tasks')
    desktop.snapTo(win.id, 'right')
    expect(win.snap).toBe('right')
    desktop.setArea({ x: 12, y: 12, w: 1000, h: 600 })
    expect(win.w).toBe(500)
    expect(win.x).toBe(12 + 500)
    expect(win.h).toBe(600)
  })

  it('свободное перемещение снимает прилипание', () => {
    const desktop = freshStore()
    const win = desktop.open('/tasks')
    desktop.snapTo(win.id, 'left')
    desktop.setRect(win.id, { x: 300, y: 200, w: 700, h: 500 })
    expect(win.mode).toBe('normal')
    expect(win.snap).toBeNull()
  })

  it('сессия восстанавливается без недоступных разделов', () => {
    const desktop = freshStore()
    desktop.open('/tasks')
    desktop.open('/notes')

    const restored = freshStore()
    // Компанийные разделы недоступны (например, у супер-админа).
    const ok = restored.restoreSession((app) => app.available({ hasCompany: false, isSuperAdmin: true, settings: {} }))
    expect(ok).toBe(true)
    expect(restored.windows.map((w) => w.appId)).toEqual(['notes'])
    expect(appById('tasks').available({ hasCompany: false, isSuperAdmin: true, settings: {} })).toBe(false)
  })

  it('закрытие последнего окна снимает фокус', () => {
    const desktop = freshStore()
    const win = desktop.open('/tasks')
    desktop.close(win.id)
    expect(desktop.windows).toHaveLength(0)
    expect(desktop.focusedId).toBeNull()
  })
})
