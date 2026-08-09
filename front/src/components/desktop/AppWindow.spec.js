import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { useDesktopStore } from '@/stores/desktop.js'
import AppWindow from './AppWindow.vue'

function setup(path = '/notes') {
  setActivePinia(createPinia())
  const desktop = useDesktopStore()
  desktop.setArea({ x: 12, y: 12, w: 1400, h: 800 })
  const win = desktop.open(path)
  const wrapper = mount(AppWindow, {
    props: { win },
    // Содержимое раздела — отдельная ответственность (грузит вью маршрута).
    global: { stubs: { WindowContent: true } },
  })
  return { desktop, win, wrapper }
}

describe('AppWindow', () => {
  beforeEach(() => { localStorage.clear() })

  it('заголовок берётся из реестра приложений и уточняется маршрутом', () => {
    const { wrapper } = setup('/notes')
    expect(wrapper.find('.win-title').text()).toBe('Заметки')

    const { wrapper: editor } = setup('/notes/5')
    expect(editor.find('.win-title').text()).toBe('Заметка')
  })

  it('геометрия окна отражается в стиле', () => {
    const { win, wrapper } = setup()
    const style = wrapper.find('.win').attributes('style')
    expect(style).toContain(`${win.w}px`)
    expect(style).toContain(`translate3d(${win.x}px, ${win.y}px, 0)`)
  })

  it('кнопки сворачивают, разворачивают и закрывают окно', async () => {
    const { desktop, win, wrapper } = setup()
    const [minimizeBtn, maxBtn, closeBtn] = wrapper.findAll('.win-btn')

    await minimizeBtn.trigger('click')
    expect(win.minimized).toBe(true)

    desktop.restore(win.id)
    await maxBtn.trigger('click')
    expect(win.mode).toBe('max')

    await closeBtn.trigger('click')
    expect(desktop.windows).toHaveLength(0)
  })

  it('двойной клик по заголовку разворачивает и возвращает окно', async () => {
    const { win, wrapper } = setup()
    await wrapper.find('.win-bar').trigger('dblclick')
    expect(win.mode).toBe('max')
    await wrapper.find('.win-bar').trigger('dblclick')
    expect(win.mode).toBe('normal')
  })

  it('раздел лежит в своём контейнере, а не прямо в теле окна', () => {
    // Тело окна — цель телепорта модалок, а Teleport оставляет там пустые
    // текстовые узлы даже у закрытой модалки. Соседствуй с ними корень раздела
    // — при смене раздела Vue брал бы якорь вставки из них и падал на
    // insertBefore, унося всё окно. Контейнер эту соседство исключает.
    const { wrapper } = setup()
    const view = wrapper.find('.win-body > .win-view')
    expect(view.exists()).toBe(true)
    expect(view.find('window-content-stub').exists()).toBe(true)
  })

  it('кнопка «назад» появляется только при истории окна', async () => {
    const { desktop, win, wrapper } = setup('/notes')
    expect(wrapper.find('.win-nav').exists()).toBe(false)

    desktop.navigate(win.id, '/notes/5')
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.win-nav').exists()).toBe(true)

    await wrapper.find('.win-nav').trigger('click')
    expect(win.path).toBe('/notes')
  })
})
