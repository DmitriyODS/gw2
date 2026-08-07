import { describe, expect, it } from 'vitest'
import { clipboardLink, linkifySelection } from './pasteLink.js'

// Поле-заглушка: linkifySelection читает у элемента только value и границы
// выделения.
function field(value, from, to) {
  return { value, selectionStart: from, selectionEnd: to }
}

describe('clipboardLink', () => {
  it('узнаёт адрес с протоколом и без него', () => {
    expect(clipboardLink('https://example.com/a')).toBe('https://example.com/a')
    expect(clipboardLink(' vk.com ')).toBe('https://vk.com/')
  })

  it('обычный текст ссылкой не считает', () => {
    expect(clipboardLink('просто текст')).toBe(null)
    expect(clipboardLink('почта@example.com')).toBe(null)
    expect(clipboardLink('')).toBe(null)
  })
})

describe('linkifySelection', () => {
  it('оборачивает выделение в markdown-ссылку', () => {
    const res = linkifySelection(field('смотри тут дальше', 7, 10), 'https://example.com')
    expect(res.value).toBe('смотри [тут](https://example.com/) дальше')
    expect(res.caret).toBe(res.value.indexOf(' дальше'))
  })

  it('без выделения ничего не подменяет — вставка обычная', () => {
    expect(linkifySelection(field('текст', 3, 3), 'https://example.com')).toBe(null)
  })

  it('не трогает вставку, если в буфере не адрес', () => {
    expect(linkifySelection(field('текст', 0, 5), 'другой текст')).toBe(null)
  })

  it('не вкладывает ссылку в ссылку', () => {
    expect(linkifySelection(field('https://a.example', 0, 17), 'https://b.example')).toBe(null)
  })
})
