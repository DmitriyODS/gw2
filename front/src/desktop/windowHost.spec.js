import { describe, it, expect } from 'vitest'
import { defineComponent, h, nextTick, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { provideWindowHost, useModalHost } from './windowHost.js'

// Модалка раздела спрашивает цель телепорта у своего окна; вне окна цель
// прежняя — body, поэтому один и тот же компонент годится и разделу в окне, и
// глобальному плавающему виджету.
const seen = []

const Modal = defineComponent({
  setup() {
    const { host, inWindow } = useModalHost()
    seen.push({ host, inWindow })
    return () => h('div')
  },
})

const Window = defineComponent({
  setup() {
    const body = ref(null)
    provideWindowHost(body)
    return () => h('section', { ref: body }, [h(Modal)])
  },
})

describe('хост модалок окна', () => {
  it('вне окна модалка уходит в body', () => {
    seen.length = 0
    mount(Modal)
    expect(seen[0].host.value).toBe('body')
    expect(seen[0].inWindow.value).toBe(false)
  })

  it('в окне модалка уходит в тело своего окна', async () => {
    seen.length = 0
    const wrapper = mount(Window)
    await nextTick()
    expect(seen[0].host.value).toBe(wrapper.find('section').element)
    expect(seen[0].inWindow.value).toBe(true)
  })

  it('без элемента цель — body: телепорту всегда есть куда целиться', () => {
    seen.length = 0
    const empty = defineComponent({
      setup() {
        provideWindowHost(ref(null))
        return () => h(Modal)
      },
    })
    mount(empty)
    expect(seen[0].host.value).toBe('body')
    expect(seen[0].inWindow.value).toBe(false)
  })
})
