import { describe, it, expect } from 'vitest'
import { docToMarkdown } from './tiptapMarkdown.js'

const t = (text, ...marks) => ({ type: 'text', text, marks: marks.map((m) => (typeof m === 'string' ? { type: m } : m)) })
const p = (...content) => ({ type: 'paragraph', content })
const doc = (...content) => ({ type: 'doc', content })

describe('docToMarkdown', () => {
  it('пустой документ → пустая строка', () => {
    expect(docToMarkdown(null)).toBe('')
    expect(docToMarkdown(doc())).toBe('')
    expect(docToMarkdown(doc(p()))).toBe('')
  })

  it('абзацы и марки: bold/italic/strike/code/link', () => {
    const md = docToMarkdown(doc(p(t('обычный '), t('жирный', 'bold'), t(' и '), t('код', 'code'))))
    expect(md).toBe('обычный **жирный** и `код`')
    expect(docToMarkdown(doc(p(t('к', 'italic'))))).toBe('*к*')
    expect(docToMarkdown(doc(p(t('з', 'strike'))))).toBe('~~з~~')
    expect(docToMarkdown(doc(p(t('сайт', { type: 'link', attrs: { href: 'https://a.ru' } })))))
      .toBe('[сайт](https://a.ru)')
  })

  it('underline/highlight не имеют md-аналога — текст сохраняется без маркеров', () => {
    expect(docToMarkdown(doc(p(t('текст', 'underline', 'highlight'))))).toBe('текст')
  })

  it('заголовки с клампом уровня к 1..3', () => {
    expect(docToMarkdown(doc({ type: 'heading', attrs: { level: 2 }, content: [t('Раздел')] })))
      .toBe('## Раздел')
    expect(docToMarkdown(doc({ type: 'heading', attrs: { level: 5 }, content: [t('Глубокий')] })))
      .toBe('### Глубокий')
  })

  it('списки: маркированный, нумерованный, чек-лист', () => {
    const ul = { type: 'bulletList', content: [
      { type: 'listItem', content: [p(t('один'))] },
      { type: 'listItem', content: [p(t('два'))] },
    ] }
    expect(docToMarkdown(doc(ul))).toBe('- один\n- два')

    const ol = { type: 'orderedList', content: [
      { type: 'listItem', content: [p(t('раз'))] },
      { type: 'listItem', content: [p(t('два'))] },
    ] }
    expect(docToMarkdown(doc(ol))).toBe('1. раз\n2. два')

    const tasks = { type: 'taskList', content: [
      { type: 'taskItem', attrs: { checked: true }, content: [p(t('готово'))] },
      { type: 'taskItem', attrs: { checked: false }, content: [p(t('нет'))] },
    ] }
    expect(docToMarkdown(doc(tasks))).toBe('- [x] готово\n- [ ] нет')
  })

  it('цитата, код-блок, линейка, hardBreak', () => {
    expect(docToMarkdown(doc({ type: 'blockquote', content: [p(t('раз')), p(t('два'))] })))
      .toBe('> раз\n> два')
    expect(docToMarkdown(doc({ type: 'codeBlock', content: [t('a = 1')] })))
      .toBe('```\na = 1\n```')
    expect(docToMarkdown(doc({ type: 'horizontalRule' }))).toBe('---')
    expect(docToMarkdown(doc(p(t('раз'), { type: 'hardBreak' }, t('два'))))).toBe('раз\nдва')
  })

  it('картинка → ![alt](src)', () => {
    expect(docToMarkdown(doc({ type: 'image', attrs: { src: '/uploads/notes/1.png', alt: 'схема' } })))
      .toBe('![схема](/uploads/notes/1.png)')
  })

  it('таблица → md-таблица с разделителем', () => {
    const cell = (tag, txt) => ({ type: tag, content: [p(t(txt))] })
    const table = { type: 'table', content: [
      { type: 'tableRow', content: [cell('tableHeader', 'Имя'), cell('tableHeader', 'Роль')] },
      { type: 'tableRow', content: [cell('tableCell', 'Аня'), cell('tableCell', 'админ')] },
    ] }
    expect(docToMarkdown(doc(table))).toBe('| Имя | Роль |\n| --- | --- |\n| Аня | админ |')
  })

  it('фрагмент выделения (массив блоков) тоже конвертируется', () => {
    expect(docToMarkdown([p(t('кусочек', 'bold'))])).toBe('**кусочек**')
  })

  it('блоки разделяются пустой строкой', () => {
    expect(docToMarkdown(doc(p(t('раз')), p(t('два'))))).toBe('раз\n\nдва')
  })
})
