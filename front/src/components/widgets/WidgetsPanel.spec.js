import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '@/stores/auth.js'
import { useDesktopStore } from '@/stores/desktop.js'
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import WidgetsPanel from './WidgetsPanel.vue'
import LiveTile from '@/components/desktop/LiveTile.vue'

vi.mock('@/api/users.js', () => ({
  getDesktopPrefs: vi.fn(() => Promise.resolve({ prefs: {} })),
  saveDesktopPrefs: vi.fn(() => Promise.resolve({})),
}))

function setup(props = {}) {
  setActivePinia(createPinia())
  useAuthStore().applySession({ access_token: 't', role_level: 1, company_id: 1 })
  const desktop = useDesktopStore()
  desktop.setArea({ x: 300, y: 0, w: 1100, h: 800 })
  const prefs = useDesktopPrefsStore()
  const wrapper = mount(WidgetsPanel, {
    props: { width: 300, ...props },
    global: { stubs: { CompanySelect: true, ContextMenu: true, AppDialog: true } },
  })
  return { desktop, prefs, wrapper }
}

// Перетаскивание: dataTransfer в jsdom не эмулируется, подсовываем свой.
function dataTransfer() {
  return { effectAllowed: '', setData: vi.fn() }
}

const labels = (wrapper) => wrapper.findAll('.lt-title').map((n) => n.text())

describe('панель каркаса «Виджеты»', () => {
  beforeEach(() => { localStorage.clear() })

  it('плитки идут группами в порядке реестра и всегда широкие', () => {
    const { wrapper } = setup()
    expect(labels(wrapper).slice(0, 3)).toEqual(['Задачи', 'Реестры', 'Календари'])
    // Размера у плиток нет: панель — одна колонка.
    expect(wrapper.findAllComponents(LiveTile).every((c) => c.props('wide'))).toBe(true)
  })

  it('клик по плитке открывает раздел, повторный — возвращает туда, где были', async () => {
    const { desktop, wrapper } = setup()
    await wrapper.findAll('.wp-tile')[0].trigger('click')
    expect(desktop.windows).toHaveLength(1)
    expect(desktop.focused.path).toBe('/tasks')

    // Ушли внутрь раздела и переключились на другой.
    desktop.navigate(desktop.focused.id, '/tasks/7')
    await wrapper.findAll('.wp-tile')[1].trigger('click')
    expect(desktop.focused.appId).toBe('registries')

    // Возврат по плитке не перезапускает раздел с корневого пути.
    await wrapper.findAll('.wp-tile')[0].trigger('click')
    expect(desktop.focused.path).toBe('/tasks/7')
  })

  it('повторный клик по активному разделу возвращает пустой холст', async () => {
    const { desktop, wrapper } = setup()
    await wrapper.findAll('.wp-tile')[0].trigger('click')
    expect(desktop.startOpen).toBe(false)

    await wrapper.findAll('.wp-tile')[0].trigger('click')
    expect(desktop.startOpen).toBe(true)
    // Раздел при этом остаётся открытым — холст его не закрывает.
    expect(desktop.windows).toHaveLength(1)
  })

  it('открытый раздел помечен, активный — подсвечен', async () => {
    const { desktop, wrapper } = setup()
    desktop.open('/tasks')
    desktop.open('/registries')
    await wrapper.vm.$nextTick()

    const tiles = wrapper.findAll('.wp-tile')
    expect(tiles[0].classes()).toContain('opened')
    expect(tiles[0].classes()).not.toContain('active')
    expect(tiles[1].classes()).toContain('active')
  })

  it('раскладка панели своя — «Пуск» рабочего стола она не трогает', async () => {
    const { prefs, wrapper } = setup()
    const tiles = wrapper.findAll('.wp-tile')

    await tiles[0].trigger('dragstart', { dataTransfer: dataTransfer() }) // «Задачи»
    await tiles[2].trigger('dragover') // отпускаем над «Календарями»
    await tiles[0].trigger('dragend')

    expect(prefs.prefs.layouts.widgets.order.work.slice(0, 3)).toEqual(['registries', 'calendars', 'tasks'])
    expect(prefs.prefs.layouts.desktop.order.work).toBeUndefined()
    expect(labels(wrapper).slice(0, 3)).toEqual(['Реестры', 'Календари', 'Задачи'])
  })

  it('раздел сворачивается кликом по заголовку', async () => {
    const { prefs, wrapper } = setup()
    await wrapper.find('.wp-group-toggle').trigger('click')
    expect(prefs.isCollapsed('widgets', 'work')).toBe(true)
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.wp-group-body').classes()).toContain('collapsed')
  })

  it('свёрнутая панель показывает одни значки, а кнопка просит развернуть', async () => {
    const { wrapper } = setup({ collapsed: true, width: 64 })
    expect(wrapper.findAll('.wp-tile')).toHaveLength(0)
    expect(wrapper.findAll('.wp-rail-btn').length).toBeGreaterThan(3)
    expect(wrapper.find('.wp').classes()).toContain('rail')

    await wrapper.findAll('.wp-icon-btn')[2].trigger('click')
    expect(wrapper.emitted('toggle')).toHaveLength(1)
  })

  it('пока ничего не закреплено — панель показывает все разделы без переключателя', () => {
    const { wrapper } = setup()
    expect(wrapper.find('.wp-view').exists()).toBe(false)
    expect(wrapper.findAll('.wp-group').length).toBeGreaterThan(1)
  })

  it('закреплённые собираются в «Избранном» и переставляются перетаскиванием', async () => {
    const { prefs, wrapper } = setup()
    prefs.pin('widgets', 'notes')
    prefs.pin('widgets', 'tasks')
    await wrapper.vm.$nextTick()

    // Появился переключатель, и панель встала на избранное (настройка по умолчанию).
    expect(wrapper.find('.wp-view').exists()).toBe(true)
    expect(labels(wrapper)).toEqual(['Заметки', 'Задачи'])
    // Категорий в этом виде нет — только сами плитки.
    expect(wrapper.findAll('.wp-group')).toHaveLength(0)

    const tiles = wrapper.findAll('.wp-tile')
    await tiles[0].trigger('dragstart', { dataTransfer: dataTransfer() }) // «Заметки»
    await tiles[1].trigger('dragover') // над «Задачами»
    await tiles[0].trigger('dragend')

    expect(prefs.pinnedList('widgets')).toEqual(['tasks', 'notes'])
    expect(labels(wrapper)).toEqual(['Задачи', 'Заметки'])
  })

  it('переключение на «Все» показывает категории целиком', async () => {
    const { prefs, wrapper } = setup()
    prefs.pin('widgets', 'notes')
    prefs.setWidgetsView('all')
    await wrapper.vm.$nextTick()

    expect(wrapper.findAll('.wp-group').length).toBeGreaterThan(1)
    expect(labels(wrapper).length).toBeGreaterThan(5)
  })

  it('подвал ведёт в аккаунт, а марка — в «О приложении»', async () => {
    const { desktop, wrapper } = setup()
    await wrapper.find('.wp-user').trigger('click')
    expect(desktop.focused.path).toBe('/settings?section=account')

    await wrapper.find('.wp-brand').trigger('click')
    expect(desktop.focused.path).toBe('/settings?section=about')
  })
})
