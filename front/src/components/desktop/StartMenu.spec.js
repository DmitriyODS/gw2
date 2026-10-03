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
    // Смена экранов — <Transition mode="out-in">: заглушка меняет их сразу.
    global: { stubs: { CompanySelect: true, ContextMenu: true, transition: true } },
  })
  return { desktop, prefs, wrapper }
}

// Перетаскивание: dataTransfer в jsdom не эмулируется, подсовываем свой.
function dataTransfer() {
  return { effectAllowed: '', setData: vi.fn() }
}

// Название раздела на плитке избранного рисует LiveTile (.lt-title), в
// каталоге — строка (.sm-row-title).
const labels = (wrapper) => wrapper.findAll('.lt-title').map((n) => n.text())
const rowLabels = (wrapper) => wrapper.findAll('.sm-row-title').map((n) => n.text())
const rowOf = (wrapper, title) => wrapper.findAll('.sm-row').find((r) => r.find('.sm-row-title').text() === title)

async function switchScreen(wrapper) {
  await wrapper.find('.sm-switch button').trigger('click')
}

describe('меню «Пуск»', () => {
  beforeEach(() => { localStorage.clear() })

  it('открывается на избранном: повседневные разделы по умолчанию', () => {
    const { wrapper } = setup()
    expect(labels(wrapper).slice(0, 3)).toEqual(['Мессенджер', 'Задачи', 'Заметки'])
    expect(wrapper.find('.sm-row').exists()).toBe(false)
  })

  it('«Все разделы» показывают каталог по категориям и ведут обратно в избранное', async () => {
    const { wrapper } = setup()
    await switchScreen(wrapper)
    expect(wrapper.find('.sm-tile').exists()).toBe(false)
    expect(rowLabels(wrapper).slice(0, 3)).toEqual(['Задачи', 'Реестры', 'Календари'])

    await switchScreen(wrapper)
    expect(wrapper.find('.sm-row').exists()).toBe(false)
    expect(labels(wrapper)[0]).toBe('Мессенджер')
  })

  it('звёздочка в каталоге добавляет раздел в избранное и убирает его', async () => {
    const { prefs, wrapper } = setup()
    await switchScreen(wrapper)

    await rowOf(wrapper, 'Реестры').find('.sm-row-star').trigger('click')
    expect(prefs.favoritesList('desktop')).toContain('registries')
    // Набор по умолчанию при первой правке становится своим списком целиком.
    expect(prefs.favoritesList('desktop')).toContain('messenger')

    await rowOf(wrapper, 'Мессенджер').find('.sm-row-star').trigger('click')
    expect(prefs.favoritesList('desktop')).not.toContain('messenger')

    await switchScreen(wrapper)
    expect(labels(wrapper)).toContain('Реестры')
    expect(labels(wrapper)).not.toContain('Мессенджер')
  })

  it('без избранного меню сразу показывает все разделы и не предлагает переключение', async () => {
    const { prefs, wrapper } = setup()
    prefs.setStartFavorites(false)
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.sm-tile').exists()).toBe(false)
    expect(rowLabels(wrapper)[0]).toBe('Задачи')
    expect(wrapper.find('.sm-switch').exists()).toBe(false)
  })

  it('кнопка «во весь экран» переводит меню в планшетный вид и обратно', async () => {
    const { desktop, wrapper } = setup()
    await wrapper.find('.sm-full').trigger('click')
    expect(desktop.startFull).toBe(true)
    expect(wrapper.find('.start-menu').exists()).toBe(false)
    expect(wrapper.find('.sm-tablet').exists()).toBe(true)

    desktop.startFull = false
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.start-menu').exists()).toBe(true)
  })

  it('ленту активности можно спрятать — меню становится одной колонкой', async () => {
    const { prefs, wrapper } = setup()
    expect(wrapper.find('.ap').exists()).toBe(true)
    prefs.setStartActivity(false)
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.ap').exists()).toBe(false)
    expect(wrapper.find('.start-menu').classes()).toContain('no-activity')
  })

  it('перетаскивание в избранном меняет порядок и запоминает его', async () => {
    const { prefs, wrapper } = setup()
    const tiles = wrapper.findAll('.sm-tile')

    await tiles[0].trigger('dragstart', { dataTransfer: dataTransfer() }) // «Мессенджер»
    await tiles[2].trigger('dragover') // над «Заметками»
    await tiles[0].trigger('dragend')

    expect(prefs.favoritesList('desktop').slice(0, 3)).toEqual(['tasks', 'notes', 'messenger'])
    expect(labels(wrapper).slice(0, 3)).toEqual(['Задачи', 'Заметки', 'Мессенджер'])
  })

  it('перетаскивание в каталоге меняет порядок внутри категории', async () => {
    const { prefs, wrapper } = setup()
    await switchScreen(wrapper)
    const rows = wrapper.findAll('.sm-row')

    await rows[0].trigger('dragstart', { dataTransfer: dataTransfer() }) // «Задачи»
    await rows[2].trigger('dragover') // над «Календарями»
    await rows[0].trigger('dragend')

    expect(prefs.prefs.layouts.desktop.order.work.slice(0, 3)).toEqual(['registries', 'calendars', 'tasks'])
    expect(rowLabels(wrapper).slice(0, 3)).toEqual(['Реестры', 'Календари', 'Задачи'])
  })

  it('перетаскивание в другую категорию переносит раздел туда', async () => {
    const { prefs, wrapper } = setup()
    await switchScreen(wrapper)
    const rows = wrapper.findAll('.sm-row')

    await rows[0].trigger('dragstart', { dataTransfer: dataTransfer() }) // «Задачи»
    await rowOf(wrapper, 'Мессенджер').trigger('dragover') // категория «Коммуникация»
    await rows[0].trigger('dragend')

    expect(prefs.prefs.layouts.desktop.appGroup.tasks).toBe('team')
    expect(prefs.prefs.layouts.desktop.order.team[0]).toBe('tasks')
    expect(rowLabels(wrapper).slice(0, 2)).toEqual(['Реестры', 'Календари'])
  })

  it('своя категория создаётся, переименовывается и удаляется без потери разделов', async () => {
    const { prefs, wrapper } = setup()
    await switchScreen(wrapper)
    await wrapper.find('.sm-add-group').trigger('click')

    const key = prefs.prefs.layouts.desktop.groups[0].key
    expect(key).toBeTruthy()

    // Новая категория сразу открыта на переименование — вводим имя и жмём Enter.
    const input = wrapper.find('.sm-group-input')
    await input.setValue('Мои дела')
    await input.trigger('keyup.enter')
    expect(prefs.prefs.layouts.desktop.groups[0].label).toBe('Мои дела')
    await wrapper.vm.$nextTick()
    expect(wrapper.findAll('.sm-group-label').map((n) => n.text())).toContain('Мои дела')

    // Раздел, перенесённый в свою категорию, возвращается в родную при удалении.
    prefs.moveTileToGroup('desktop', 'notes', key, ['notes'])
    expect(prefs.prefs.layouts.desktop.appGroup.notes).toBe(key)
    prefs.removeGroup('desktop', key)
    expect(prefs.prefs.layouts.desktop.groups).toHaveLength(0)
    expect(prefs.prefs.layouts.desktop.appGroup.notes).toBeUndefined()
  })

  it('категория сворачивается кликом по заголовку', async () => {
    const { prefs, wrapper } = setup()
    await switchScreen(wrapper)
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
    const tasksTile = () => wrapper.findAll('.sm-tile').find((t) => t.find('.lt-title').text() === 'Задачи')
    expect(tasksTile().classes()).toContain('is-wide')
    prefs.setTileSize('desktop', 'tasks', 'square')
    await wrapper.vm.$nextTick()
    expect(tasksTile().classes()).toContain('is-square')
  })
})
