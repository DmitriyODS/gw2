import { describe, it, expect } from 'vitest'
import { renderMarkdown } from './markdown.js'

describe('renderMarkdown', () => {
  it('пустой вход → пустая строка', () => {
    expect(renderMarkdown('')).toBe('')
    expect(renderMarkdown(null)).toBe('')
  })

  it('экранирует HTML (XSS невозможен)', () => {
    const html = renderMarkdown('<script>alert(1)</script>')
    expect(html).not.toContain('<script>')
    expect(html).toContain('&lt;script&gt;')
  })

  it('заголовки, жирный, курсив, зачёркнутый, инлайн-код', () => {
    expect(renderMarkdown('# Заголовок')).toContain('<h1 class="md-h md-h1">Заголовок</h1>')
    expect(renderMarkdown('### Третий')).toContain('<h3')
    const inline = renderMarkdown('**ж** *к* ~~з~~ `код`')
    expect(inline).toContain('<strong>ж</strong>')
    expect(inline).toContain('<em>к</em>')
    expect(inline).toContain('<s>з</s>')
    expect(inline).toContain('<code class="md-code">код</code>')
  })

  it('число с пробелами в тексте не портится (плейсхолдеры инлайн-кода)', () => {
    expect(renderMarkdown('жду 5 минут')).toContain('жду 5 минут')
    expect(renderMarkdown('в `x` было 7 раз')).toContain('было 7 раз')
  })

  it('ссылки [текст](url) и автолинки; javascript: не проходит', () => {
    expect(renderMarkdown('[сайт](https://a.ru)')).toContain('href="https://a.ru"')
    expect(renderMarkdown('см. https://b.ru')).toContain('href="https://b.ru"')
    expect(renderMarkdown('[x](javascript:alert(1))')).not.toContain('href="javascript:')
  })

  it('внутри инлайн-кода разметка не парсится', () => {
    const html = renderMarkdown('`**не жирный**`')
    expect(html).toContain('**не жирный**')
    expect(html).not.toContain('<strong>')
  })

  it('картинки ![alt](url); относительный /uploads проходит', () => {
    const html = renderMarkdown('![обложка](/uploads/notes/1.png)')
    expect(html).toContain('<img class="md-img" src="/uploads/notes/1.png" alt="обложка"')
  })

  it('маркированный и нумерованный списки', () => {
    const ul = renderMarkdown('- один\n- два')
    expect(ul).toContain('<ul class="md-list">')
    expect(ul).toContain('<li>один</li>')
    const ol = renderMarkdown('1. раз\n2. два')
    expect(ol).toContain('<ol class="md-list">')
    expect(ol).toContain('<li>два</li>')
  })

  it('чек-лист - [ ] / - [x]', () => {
    const html = renderMarkdown('- [ ] сделать\n- [x] готово')
    expect(html).toContain('<li class="md-task"><input type="checkbox" disabled><span>сделать</span></li>')
    expect(html).toContain('<input type="checkbox" disabled checked><span>готово</span>')
  })

  it('цитата из нескольких строк', () => {
    const html = renderMarkdown('> первая\n> вторая')
    expect(html).toContain('<blockquote class="md-quote">первая<br>вторая</blockquote>')
  })

  it('горизонтальная линейка ---', () => {
    expect(renderMarkdown('до\n\n---\n\nпосле')).toContain('<hr class="md-hr">')
  })

  it('код-фенс сохраняет содержимое как есть', () => {
    const html = renderMarkdown('```\nconst a = 1\n**не жирный**\n```')
    expect(html).toContain('<pre class="md-pre">')
    expect(html).toContain('const a = 1')
    expect(html).not.toContain('<strong>')
  })

  it('таблица | a | b | с разделителем', () => {
    const html = renderMarkdown('| Имя | Роль |\n|---|---|\n| Аня | админ |')
    expect(html).toContain('<table class="md-table">')
    expect(html).toContain('<th>Имя</th>')
    expect(html).toContain('<td>админ</td>')
  })

  it('переносы строк внутри абзаца → <br>', () => {
    expect(renderMarkdown('раз\nдва')).toContain('раз<br>два')
  })
})
