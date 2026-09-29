import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import MarkdownView from './MarkdownView.vue'

describe('MarkdownView', () => {
  it('рисует разметку парсера элементами, а не строкой', () => {
    const w = mount(MarkdownView, { props: { source: '**жирный** и *курсив*\n\n- [x] готово' } })
    expect(w.find('strong').text()).toBe('жирный')
    expect(w.find('em').text()).toBe('курсив')
    const box = w.find('input[type="checkbox"]').element
    expect(box.checked).toBe(true)
    expect(box.disabled).toBe(true)
  })

  it('HTML в тексте остаётся текстом', () => {
    const w = mount(MarkdownView, { props: { source: '<img src=x onerror=alert(1)>' } })
    expect(w.find('img').exists()).toBe(false)
    expect(w.text()).toContain('<img src=x onerror=alert(1)>')
  })

  it('опасная схема ссылки до DOM не доходит', () => {
    const w = mount(MarkdownView, { props: { source: '[жми](javascript:alert(1))' } })
    const href = w.find('a').attributes('href') || ''
    expect(href.toLowerCase().startsWith('javascript:')).toBe(false)
  })

  it('клик по хештегу отдаёт тег наружу', async () => {
    const w = mount(MarkdownView, { props: { source: 'про #релиз' } })
    await w.find('.md-tag').trigger('click')
    expect(w.emitted('tag')).toEqual([['релиз']])
  })

  it('один и тот же текст в нескольких местах рисуется целиком (кэш разбора)', async () => {
    const source = '**общий** текст с [ссылкой](https://a.ru)'
    const a = mount(MarkdownView, { props: { source } })
    const b = mount(MarkdownView, { props: { source } })
    expect(a.html()).toBe(b.html())
    expect(b.find('strong').text()).toBe('общий')
    await a.setProps({ source: 'другой' })
    expect(a.text()).toBe('другой')
    expect(b.find('a').attributes('href')).toBe('https://a.ru')
  })
})
