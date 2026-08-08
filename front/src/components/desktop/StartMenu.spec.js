import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '@/stores/auth.js'
import { useDesktopStore } from '@/stores/desktop.js'
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import { useLiveTilesStore } from '@/stores/liveTiles.js'
import { useActivityStore } from '@/stores/activity.js'
import StartMenu from './StartMenu.vue'
import LiveTile from './LiveTile.vue'

vi.mock('@/api/users.js', () => ({
  getDesktopPrefs: vi.fn(() => Promise.resolve({ prefs: {} })),
  saveDesktopPrefs: vi.fn(() => Promise.resolve({})),
}))

function setup() {
  setActivePinia(createPinia())
  useAuthStore().applySession({ access_token: 't', role_level: 1, company_id: 1 })
  const desktop = useDesktopStore()
  desktop.setArea({ x: 12, y: 12, w: 1400, h: 800 })
  const prefs = useDesktopPrefsStore()
  const wrapper = mount(StartMenu, {
    global: { stubs: { CompanySelect: true, ContextMenu: true } },
  })
  return { desktop, prefs, wrapper }
}

// Перетаскивание: dataTransfer в jsdom не эмулируется, подсовываем свой.
function dataTransfer() {
  return { effectAllowed: '', setData: vi.fn() }
}

// Название раздела на плитке рисует LiveTile (.lt-title).
const labels = (wrapper) => wrapper.findAll('.lt-title').map((n) => n.text())

describe('меню «Пуск»', () => {
  beforeEach(() => { localStorage.clear() })

  it('плитки идут группами в порядке реестра', () => {
    const { wrapper } = setup()
    expect(labels(wrapper).slice(0, 3)).toEqual(['Задачи', 'Реестры', 'Календари'])
  })

  it('перетаскивание меняет порядок внутри группы и запоминает его', async () => {
    const { prefs, wrapper } = setup()
    const tiles = wrapper.findAll('.sm-tile')

    await tiles[0].trigger('dragstart', { dataTransfer: dataTransfer() }) // «Задачи»
    await tiles[2].trigger('dragover') // отпускаем над «Календарями»
    await tiles[0].trigger('dragend')

    expect(prefs.prefs.layouts.desktop.order.work.slice(0, 3)).toEqual(['registries', 'calendars', 'tasks'])
    expect(labels(wrapper).slice(0, 3)).toEqual(['Реестры', 'Календари', 'Задачи'])
  })

  it('перетаскивание в другой раздел переносит плитку туда', async () => {
    const { prefs, wrapper } = setup()
    const tiles = wrapper.findAll('.sm-tile')
    const messengerTile = tiles.find((t) => t.find('.lt-title').text() === 'Мессенджер')

    await tiles[0].trigger('dragstart', { dataTransfer: dataTransfer() }) // «Задачи»
    await messengerTile.trigger('dragover') // цель — раздел «Коммуникация»
    await tiles[0].trigger('dragend')

    expect(prefs.prefs.layouts.desktop.appGroup.tasks).toBe('team')
    expect(prefs.prefs.layouts.desktop.order.team[0]).toBe('tasks')
    expect(labels(wrapper).slice(0, 2)).toEqual(['Реестры', 'Календари'])
  })

  it('свой раздел создаётся, переименовывается и удаляется без потери плиток', async () => {
    const { prefs, wrapper } = setup()
    await wrapper.find('.sm-add-group').trigger('click')

    const key = prefs.prefs.layouts.desktop.groups[0].key
    expect(key).toBeTruthy()

    // Новый раздел сразу открыт на переименование — вводим имя и жмём Enter.
    const input = wrapper.find('.sm-group-input')
    await input.setValue('Мои дела')
    await input.trigger('keyup.enter')
    expect(prefs.prefs.layouts.desktop.groups[0].label).toBe('Мои дела')
    await wrapper.vm.$nextTick()
    expect(wrapper.findAll('.sm-group-label').map((n) => n.text())).toContain('Мои дела')

    // Плитка, перенесённая в свой раздел, возвращается в родной при удалении.
    prefs.moveTileToGroup('desktop', 'notes', key, ['notes'])
    expect(prefs.prefs.layouts.desktop.appGroup.notes).toBe(key)
    prefs.removeGroup('desktop', key)
    expect(prefs.prefs.layouts.desktop.groups).toHaveLength(0)
    expect(prefs.prefs.layouts.desktop.appGroup.notes).toBeUndefined()
  })

  it('раздел сворачивается кликом по заголовку', async () => {
    const { prefs, wrapper } = setup()
    await wrapper.find('.sm-group-toggle').trigger('click')
    expect(prefs.isCollapsed('desktop', 'work')).toBe(true)
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.sm-group-body').classes()).toContain('collapsed')
  })

  it('живые плитки выключаются в настройках — сводок у плитки не остаётся', async () => {
    const { prefs, wrapper } = setup()
    useLiveTilesStore().data = { notes: { total: 4, latest: null } }
    await wrapper.vm.$nextTick()

    // Грани первой гранью не показываются (плитка начинает со значка) —
    // проверяем сам набор, который получил LiveTile.
    const notesTile = () => wrapper.findAllComponents(LiveTile).find((c) => c.props('title') === 'Заметки')
    expect(notesTile().props('faces')[0]).toMatchObject({ value: '4', label: 'заметки' })

    prefs.setLiveTiles(false)
    await wrapper.vm.$nextTick()
    expect(notesTile().props('faces')).toEqual([])
  })

  it('лента «Моя активность» ведёт к элементу и закрывает меню', async () => {
    const { desktop, wrapper } = setup()
    useActivityStore().record({ section: 'notes', id: 5, title: 'Идеи', path: '/notes/5' })
    desktop.startOpen = true
    await wrapper.vm.$nextTick()

    const item = wrapper.find('.ap-item')
    expect(item.text()).toContain('Идеи')
    expect(item.text()).toContain('Заметки')

    await item.trigger('click')
    expect(desktop.windows[0].path).toBe('/notes/5')
    expect(desktop.startOpen).toBe(false)
  })

  it('открытые разделы идут в ленте наравне с элементами и со временем', async () => {
    const { desktop, wrapper } = setup()
    useActivityStore().record({ section: 'notes', id: 5, title: 'Идеи', path: '/notes/5' })
    desktop.open('/tasks')
    desktop.startOpen = true
    await wrapper.vm.$nextTick()

    const rows = wrapper.findAll('.ap-item')
    expect(rows.map((n) => n.find('.ap-item-title').text())).toEqual(['Задачи', 'Идеи'])
    // Когда это было — в подстроке строки («Открыт раздел · только что»).
    expect(rows[0].find('.ap-item-sub').text()).toContain('Открыт раздел')
  })

  it('размер плитки берётся из настроек', async () => {
    const { prefs, wrapper } = setup()
    expect(wrapper.find('.sm-tile').classes()).toContain('is-wide')
    prefs.setTileSize('desktop', 'tasks', 'square')
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.sm-tile').classes()).toContain('is-square')
  })
})
