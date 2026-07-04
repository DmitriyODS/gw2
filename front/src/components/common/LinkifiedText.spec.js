import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import LinkifiedText from './LinkifiedText.vue'

const anchors = (w) => w.findAll('a.linkified-a')

describe('LinkifiedText', () => {
  it('оборачивает http/https/www в <a target=_blank rel=noopener>', () => {
    const w = mount(LinkifiedText, { props: { text: 'см www.a.ru и https://b.ru' } })
    const a = anchors(w)
    expect(a.map((x) => x.attributes('href'))).toEqual(['https://www.a.ru', 'https://b.ru'])
    expect(a[0].attributes('target')).toBe('_blank')
    expect(a[0].attributes('rel')).toBe('noopener noreferrer')
  })

  it('текст без ссылок остаётся текстом (нет <a>)', () => {
    const w = mount(LinkifiedText, { props: { text: 'обычный текст' } })
    expect(anchors(w)).toHaveLength(0)
    expect(w.text()).toBe('обычный текст')
  })

  it('хвостовая точка не входит в ссылку', () => {
    const w = mount(LinkifiedText, { props: { text: 'зайди на https://site.ru.' } })
    expect(anchors(w)[0].attributes('href')).toBe('https://site.ru')
    expect(w.text()).toContain('.')
  })

  // БАГ (исправлено): парная скобка сохраняется, непарная — отрезается.
  it('парную скобку в URL сохраняет (Wikipedia)', () => {
    const url = 'https://ru.wikipedia.org/wiki/Vue_(framework)'
    const w = mount(LinkifiedText, { props: { text: url } })
    expect(anchors(w)[0].attributes('href')).toBe(url)
  })

  it('непарную закрывающую скобку отрезает', () => {
    const w = mount(LinkifiedText, { props: { text: '(ссылка http://a.ru)' } })
    expect(anchors(w)[0].attributes('href')).toBe('http://a.ru')
  })

  it('XSS-безопасность: спецсимволы экранируются, тег не оживает', () => {
    const w = mount(LinkifiedText, { props: { text: '<script>alert(1)</script>' } })
    // Нет реального script-элемента, только текст.
    expect(w.find('script').exists()).toBe(false)
    expect(w.element.querySelector('script')).toBeNull()
    expect(w.text()).toContain('<script>alert(1)</script>')
  })

  it('кавычки/угловые скобки обрывают ссылку (не попадают в href)', () => {
    const w = mount(LinkifiedText, { props: { text: 'http://a.ru"onmouseover=x' } })
    expect(anchors(w)[0].attributes('href')).toBe('http://a.ru')
  })
})
