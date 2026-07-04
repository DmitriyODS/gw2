import { describe, it, expect } from 'vitest'
import { linkifyParts } from './linkify.js'

const links = (parts) => parts.filter((p) => p.type === 'link')
const texts = (parts) => parts.filter((p) => p.type === 'text').map((p) => p.value).join('')

describe('linkifyParts', () => {
  it('пустой/отсутствующий текст → пустой массив', () => {
    expect(linkifyParts('')).toEqual([])
    expect(linkifyParts(null)).toEqual([])
    expect(linkifyParts(undefined)).toEqual([])
  })

  it('распознаёт http/https и www (www → https://)', () => {
    const p = linkifyParts('a http://x.ru b https://y.ru c www.z.ru')
    const l = links(p)
    expect(l.map((x) => x.href)).toEqual(['http://x.ru', 'https://y.ru', 'https://www.z.ru'])
    // value (видимый текст) для www сохраняется как есть.
    expect(l[2].value).toBe('www.z.ru')
  })

  it('текст без ссылок остаётся единым текстовым сегментом', () => {
    const p = linkifyParts('просто текст без ссылок')
    expect(p).toEqual([{ type: 'text', value: 'просто текст без ссылок' }])
  })

  it('отрезает хвостовую пунктуацию из ссылки обратно в текст', () => {
    const p = linkifyParts('зайди на https://site.ru.')
    const l = links(p)
    expect(l).toHaveLength(1)
    expect(l[0].href).toBe('https://site.ru')
    expect(texts(p)).toContain('.')
  })

  it('запятая после ссылки не съедается', () => {
    const p = linkifyParts('см. http://a.ru, потом')
    expect(links(p)[0].href).toBe('http://a.ru')
    expect(texts(p)).toBe('см. , потом')
  })

  // БАГ (исправлено): непарная закрывающая скобка отрезается, парная — нет.
  it('непарную закрывающую скобку отрезает: (см. http://a.ru)', () => {
    const p = linkifyParts('(см. http://a.ru)')
    expect(links(p)[0].href).toBe('http://a.ru')
    expect(texts(p)).toBe('(см. )')
  })

  it('парную закрывающую скобку СОХРАНЯЕТ: Wikipedia …/Foo_(bar)', () => {
    const url = 'https://ru.wikipedia.org/wiki/Foo_(bar)'
    const p = linkifyParts(`ссылка ${url} тут`)
    expect(links(p)[0].href).toBe(url)
  })

  it('комбо: точка после парной скобки отрезается, скобка остаётся', () => {
    const url = 'https://ru.wikipedia.org/wiki/A_(b)'
    const p = linkifyParts(`${url}.`)
    expect(links(p)[0].href).toBe(url)
    expect(texts(p)).toBe('.')
  })

  it('несколько ссылок подряд корректно разбиваются', () => {
    const p = linkifyParts('http://a.ru https://b.ru')
    expect(links(p).map((x) => x.href)).toEqual(['http://a.ru', 'https://b.ru'])
  })
})
