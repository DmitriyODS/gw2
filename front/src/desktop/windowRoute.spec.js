import { describe, it, expect, beforeEach } from 'vitest'
import { defineComponent, h, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { useRoute, useRouter } from 'vue-router'
import { setActivePinia, createPinia } from 'pinia'
import { useDesktopStore } from '@/stores/desktop.js'
import { provideWindowRoute } from './windowRoute.js'

// Раздел внутри окна пользуется обычными useRoute()/useRouter() — и должен
// получать маршрут СВОЕГО окна, а не адрес страницы.
const mounted = []

const Section = defineComponent({
  setup() {
    const route = useRoute()
    const router = useRouter()
    mounted.push({ route, router })
    return () => h('div', route.path)
  },
})

const Host = defineComponent({
  props: { win: { type: Object, required: true } },
  setup(props) {
    provideWindowRoute(props.win, useDesktopStore())
    return () => h(Section)
  },
})

function setup(path = '/notes') {
  setActivePinia(createPinia())
  mounted.length = 0
  const desktop = useDesktopStore()
  desktop.setArea({ x: 12, y: 12, w: 1400, h: 800 })
  const win = desktop.open(path)
  const wrapper = mount(Host, { props: { win } })
  return { desktop, win, wrapper, section: mounted[mounted.length - 1] }
}

describe('маршрут окна', () => {
  beforeEach(() => { localStorage.clear() })

  it('раздел видит путь и параметры своего окна', async () => {
    const { desktop, win, section } = setup('/notes')
    expect(section.route.path).toBe('/notes')

    desktop.navigate(win.id, '/notes/42')
    await nextTick()
    expect(section.route.path).toBe('/notes/42')
    expect(section.route.params.id).toBe('42')
  })

  it('два окна одного раздела не мешают друг другу', () => {
    const first = setup('/notes')
    const win2 = first.desktop.open('/notes/7', { newWindow: true })
    mount(Host, { props: { win: win2 } })
    const second = mounted[mounted.length - 1]
    expect(first.section.route.path).toBe('/notes')
    expect(second.route.path).toBe('/notes/7')
  })

  it('router.push внутри раздела ведёт по своему окну, а не по странице', () => {
    const { win, section } = setup('/notes')
    section.router.push('/notes/9')
    expect(win.path).toBe('/notes/9')
    expect(win.history).toEqual(['/notes', '/notes/9'])
  })

  it('переход в чужой раздел открывает отдельное окно', () => {
    const { desktop, win, section } = setup('/notes')
    section.router.push('/tasks')
    expect(win.path).toBe('/notes')
    expect(desktop.windows.map((w) => w.appId)).toEqual(['notes', 'tasks'])
  })

  it('router.back откатывает историю окна', () => {
    const { win, section } = setup('/notes')
    section.router.push('/notes/9')
    section.router.back()
    expect(win.path).toBe('/notes')
  })
})
