// Конвертер документа TipTap (ProseMirror JSON) в Markdown — для публикации
// заметок на портал и отправки в чат с сохранением форматирования (портал и
// мессенджер рендерят Markdown через utils/markdown.js). Underline и highlight
// в Markdown аналога не имеют — содержимое сохраняется без маркеров.

// Марки одного текстового узла: порядок вложения фиксирован, чтобы маркеры
// закрывались зеркально (`code` — самый внутренний).
const MARK_WRAP = {
  bold: ['**', '**'],
  italic: ['*', '*'],
  strike: ['~~', '~~'],
  code: ['`', '`'],
}

function inlineText(node) {
  if (node.type === 'text') {
    let s = node.text || ''
    const marks = node.marks || []
    for (const m of marks) {
      const wrap = MARK_WRAP[m.type]
      if (wrap) s = wrap[0] + s + wrap[1]
    }
    const link = marks.find((m) => m.type === 'link' && m.attrs?.href)
    if (link) s = `[${s}](${link.attrs.href})`
    return s
  }
  if (node.type === 'hardBreak') return '\n'
  if (node.type === 'image') return imageMd(node)
  return inlineOf(node)
}

function inlineOf(node) {
  return (node.content || []).map(inlineText).join('')
}

function imageMd(node) {
  const src = node.attrs?.src || ''
  if (!src) return ''
  return `![${node.attrs?.alt || ''}](${src})`
}

// prefixLines — префикс каждой строке блока (цитаты, вложенные списки).
function prefixLines(text, first, rest = first) {
  return text
    .split('\n')
    .map((l, i) => (i === 0 ? first : rest) + l)
    .join('\n')
}

function listToMarkdown(node, ordered) {
  const lines = []
  ;(node.content || []).forEach((item, idx) => {
    const marker = ordered ? `${idx + 1}. ` : '- '
    const checkbox = item.type === 'taskItem' ? (item.attrs?.checked ? '[x] ' : '[ ] ') : ''
    const parts = []
    for (const child of item.content || []) {
      if (child.type === 'paragraph') parts.push(inlineOf(child))
      else if (child.type === 'bulletList' || child.type === 'taskList') parts.push(listToMarkdown(child, false))
      else if (child.type === 'orderedList') parts.push(listToMarkdown(child, true))
      else parts.push(blockToMarkdown(child))
    }
    const [head, ...tail] = parts.filter((p) => p !== '')
    lines.push(marker + checkbox + (head ?? ''))
    // Вложенные блоки — с отступом под маркером (наш рендер списков плоский,
    // но текст остаётся читаемым и в исходнике).
    for (const t of tail) lines.push(prefixLines(t, '  '))
  })
  return lines.join('\n')
}

function tableToMarkdown(node) {
  const rows = (node.content || []).map((row) =>
    (row.content || []).map((cell) => inlineOf(cell.content?.[0] || cell).replace(/\|/g, ' ').trim()))
  if (!rows.length) return ''
  const cols = rows[0].length
  const line = (cells) => `| ${Array.from({ length: cols }, (_, i) => cells[i] ?? '').join(' | ')} |`
  const out = [line(rows[0]), `|${' --- |'.repeat(cols)}`]
  for (const r of rows.slice(1)) out.push(line(r))
  return out.join('\n')
}

function blockToMarkdown(node) {
  switch (node.type) {
    case 'paragraph':
      return inlineOf(node)
    case 'heading': {
      const lvl = Math.min(Math.max(node.attrs?.level || 1, 1), 3)
      return '#'.repeat(lvl) + ' ' + inlineOf(node)
    }
    case 'bulletList':
    case 'taskList':
      return listToMarkdown(node, false)
    case 'orderedList':
      return listToMarkdown(node, true)
    case 'blockquote':
      return (node.content || []).map((c) => prefixLines(blockToMarkdown(c), '> ')).join('\n')
    case 'codeBlock':
      return '```\n' + inlineOf(node) + '\n```'
    case 'horizontalRule':
      return '---'
    case 'image':
      return imageMd(node)
    case 'table':
      return tableToMarkdown(node)
    default:
      return node.content ? node.content.map(blockToMarkdown).join('\n\n') : inlineText(node)
  }
}

// docToMarkdown — целый документ {type: 'doc', content: [...]} или фрагмент
// (массив блоков — например, doc.slice(from, to).content.toJSON() выделения).
export function docToMarkdown(docOrNodes) {
  const nodes = Array.isArray(docOrNodes) ? docOrNodes : docOrNodes?.content || []
  return nodes
    .map(blockToMarkdown)
    .filter((b) => b.trim() !== '')
    .join('\n\n')
    .trim()
}
