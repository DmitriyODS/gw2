import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import EmptyState from './EmptyState.vue'

describe('EmptyState', () => {
  it('рендерит иконку, заголовок и подзаголовок', () => {
    const w = mount(EmptyState, {
      props: { icon: 'inbox', title: 'Пусто', subtitle: 'Здесь пока ничего нет' },
    })
    expect(w.find('.es-icon .material-symbols-outlined').text()).toBe('inbox')
    expect(w.find('.es-title').text()).toBe('Пусто')
    expect(w.find('.es-sub').text()).toBe('Здесь пока ничего нет')
  })

  it('без title/subtitle не рендерит соответствующие узлы', () => {
    const w = mount(EmptyState, { props: { icon: 'inbox' } })
    expect(w.find('.es-title').exists()).toBe(false)
    expect(w.find('.es-sub').exists()).toBe(false)
  })

  it('прокидывает size и tone в классы', () => {
    const w = mount(EmptyState, { props: { icon: 'error', size: 'sm', tone: 'error' } })
    expect(w.classes()).toContain('empty-state--sm')
    expect(w.classes()).toContain('empty-state--error')
  })

  it('рендерит слот действия', () => {
    const w = mount(EmptyState, {
      props: { icon: 'inbox' },
      slots: { default: '<button class="act">Создать</button>' },
    })
    expect(w.find('button.act').exists()).toBe(true)
  })
})
