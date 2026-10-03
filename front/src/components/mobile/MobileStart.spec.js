import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '@/stores/auth.js'
import { useDesktopPrefsStore } from '@/stores/desktopPrefs.js'
import MobileStart from './MobileStart.vue'

vi.mock('@/api/users.js', () => ({
  getDesktopPrefs: vi.fn(() => Promise.resolve({ prefs: {} })),
  saveDesktopPrefs: vi.fn(() => Promise.resolve({})),
}))

function setup(platform = 'mobile') {
  setActivePinia(createPinia())
  useAuthStore().applySession({ access_token: 't', role_level: 1, company_id: 1 })
  const prefs = useDesktopPrefsStore()
  const wrapper = mount(MobileStart, {
    props: { platform },
    // Смена экранов — <Transition mode="out-in">: заглушка меняет их сразу.
    global: { stubs: { CompanySelect: true, ContextMenu: true, transition: true } },
  })
  return { prefs, wrapper }
}

const tileLabels = (wrapper) => wrapper.findAll('.lt-title').map((n) => n.text())
const rowLabels = (wrapper) => wrapper.findAll('.mst-row-title').map((n) => n.text())
const rowOf = (wrapper, title) => wrapper.findAll('.mst-row').find((r) => r.find('.mst-row-title').text() === title)
const switchScreen = (wrapper) => wrapper.find('.mst-switch button').trigger('click')

describe('стартовый экран телефона и планшета', () => {
  beforeEach(() => { localStorage.clear() })

  it('открывается на избранном и переключается на все разделы и обратно', async () => {
    const { wrapper } = setup()
    expect(tileLabels(wrapper).slice(0, 2)).toEqual(['Мессенджер', 'Задачи'])
    expect(wrapper.find('.mst-row').exists()).toBe(false)

    await switchScreen(wrapper)
    expect(wrapper.find('.mst-tile').exists()).toBe(false)
    expect(rowLabels(wrapper).slice(0, 2)).toEqual(['Задачи', 'Реестры'])

    await switchScreen(wrapper)
    expect(tileLabels(wrapper)[0]).toBe('Мессенджер')
  })

  it('избранное у телефона своё — стол и планшет его не трогают', async () => {
    const { prefs, wrapper } = setup('mobile')
    await switchScreen(wrapper)
    await rowOf(wrapper, 'Реестры').find('.mst-row-star').trigger('click')

    expect(prefs.favoritesList('mobile')).toContain('registries')
    expect(prefs.prefs.layouts.desktop.favorites).toBeNull()
    expect(prefs.prefs.layouts.tablet.favorites).toBeNull()
  })

  it('порядок избранного меняется пунктом меню «переместить ниже»', async () => {
    const { prefs, wrapper } = setup()
    const first = wrapper.findAll('.mst-tile')[0]
    await first.trigger('contextmenu', { clientX: 10, clientY: 10 })
    wrapper.findComponent({ name: 'ContextMenu' }).vm.$emit('select', 'moveDown')
    await wrapper.vm.$nextTick()

    expect(prefs.favoritesList('mobile').slice(0, 2)).toEqual(['tasks', 'messenger'])
    expect(tileLabels(wrapper).slice(0, 2)).toEqual(['Задачи', 'Мессенджер'])
  })

  it('без избранного экран — сразу каталог, переключателя нет', async () => {
    const { prefs, wrapper } = setup()
    prefs.setStartFavorites(false)
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.mst-tile').exists()).toBe(false)
    expect(rowLabels(wrapper)[0]).toBe('Задачи')
    expect(wrapper.find('.mst-switch').exists()).toBe(false)
  })
})
