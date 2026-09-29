import { describe, it, expect, beforeEach } from 'vitest'
import { defineComponent, h, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { useVirtualList } from './useVirtualList.js'

let roCallback = null
class FakeRO {
  constructor(cb) { roCallback = cb }
  observe() {}
  unobserve() {}
  disconnect() {}
}

// Контейнер без раскладки jsdom: высоты и прокрутку задаём руками.
function fakeContainer({ clientHeight = 500, scrollHeight = 10_000 } = {}) {
  return {
    scrollTop: 0,
    clientHeight,
    scrollHeight,
    scrollTo({ top }) { this.scrollTop = top },
    getBoundingClientRect: () => ({ top: 0 }),
  }
}
function setup(n = 100, estimate = 50) {
  const container = ref(fakeContainer())
  // Список начинается у верха контейнера и уезжает вверх вместе с прокруткой.
  const list = ref({ getBoundingClientRect: () => ({ top: -container.value.scrollTop }) })
  const keys = ref(Array.from({ length: n }, (_, i) => i))
  let api
  mount(defineComponent({
    setup() {
      api = useVirtualList({ container, list, keys, estimate: () => estimate, overscan: 100 })
      return () => h('div')
    },
  }))
  return { api, container, keys }
}

function measure(api, key, height) {
  const el = document.createElement('div')
  api.measureRef(key)(el)
  roCallback([{ target: el, borderBoxSize: [{ blockSize: height }] }])
}

describe('useVirtualList', () => {
  beforeEach(() => {
    globalThis.ResizeObserver = FakeRO
    roCallback = null
  })

  it('рисует только строки у экрана с запасом', () => {
    const { api, container } = setup()
    expect(api.start.value).toBe(0)
    // 500px экрана + 100 запаса по 50px — 13 строк.
    expect(api.end.value).toBe(13)
    container.value.scrollTop = 2000
    api.onScroll()
    expect(api.start.value).toBe(38)
    expect(api.end.value).toBe(53)
    expect(api.total.value).toBe(5000)
  })

  it('замер строки выше экрана сдвигает прокрутку на разницу', () => {
    const { api, container } = setup()
    container.value.scrollTop = 2000
    api.onScroll()
    measure(api, 10, 80)
    expect(container.value.scrollTop).toBe(2030)
    expect(api.offsetOf(11)).toBe(580)
  })

  it('замер строки ниже экрана прокрутку не трогает', () => {
    const { api, container } = setup()
    container.value.scrollTop = 2000
    api.onScroll()
    measure(api, 60, 80)
    expect(container.value.scrollTop).toBe(2000)
  })

  it('у низа ленты рост строк держит прокрутку внизу', async () => {
    const { api, container } = setup()
    await api.scrollToBottom()
    container.value.scrollHeight = 10_200
    measure(api, 99, 250)
    expect(container.value.scrollTop).toBe(10_200)
  })

  it('reveal прокручивает к неотрисованной строке', async () => {
    const { api, container } = setup()
    await api.reveal(80)
    expect(container.value.scrollTop).toBe(80 * 50 - (500 - 50) / 2)
  })

  it('свёрнутое окно (нулевой замер) не портит высоты', () => {
    const { api } = setup()
    measure(api, 3, 0)
    expect(api.offsetOf(4)).toBe(200)
  })
})
