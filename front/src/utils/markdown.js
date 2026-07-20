// Минимальный безопасный markdown-парсер (без сторонних либ).
// Блоки: # h1 / ## h2 / ### h3, ```code fence```, > цитаты, списки - / 1. /
// чек-листы - [ ], таблицы | a | b |, линейка ---. Инлайн: **жирный**,
// *курсив*, ~~зачёркнутый~~, `код`, [текст](url), картинки ![alt](url),
// автоссылки http(s)://… и www.… (через linkify.js). HTML экранируется —
// XSS невозможен; url картинок/ссылок — только http(s)/mailto/tel или
// относительные пути (/uploads/…).

import { linkifyParts } from './linkify.js'

const ESCAPE_MAP = {
  '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
}

// Хештег: # не в середине слова/URL/html-сущности, затем 2..50 букв/цифр/_,
// начиная с буквы или цифры. Зеркалит серверный tagRe (portalsvc).
const TAG_RE = /(^|[^\p{L}\p{N}_&#/])#([\p{L}\p{N}][\p{L}\p{N}_]{1,49})/gu

// Упоминание: @логин (буквы/цифры/точка/подчёркивание, без точек по краям),
// не в середине слова/адреса. Зеркалит серверный mentionRe (tasksvc).
// Включается опцией renderMarkdown({ mentions:true }) — иначе @-текст не трогаем
// (нужно только в комментариях задач, не в портале).
const MENTION_RE = /(^|[^\p{L}\p{N}_.@])@([\p{L}\p{N}_](?:[\p{L}\p{N}_.]*[\p{L}\p{N}_])?)/gu

// Флаг активности упоминаний и карта login→ФИО на время одного renderMarkdown
// (синхронного — реентрантности нет, parseInline только внутри того же вызова).
// В тексте хранится @login, но в чипе показываем ФИО (data-mention = login).
let mentionsEnabled = false
let mentionNames = {}

function escapeHtml(str) {
  return String(str).replace(/[&<>"']/g, (c) => ESCAPE_MAP[c])
}

function escapeAttr(str) {
  return escapeHtml(str).replace(/`/g, '&#96;')
}

function safeUrl(url) {
  if (/^(https?:|mailto:|tel:)/i.test(url)) return url
  if (url.startsWith('/')) return url
  return 'https://' + url
}

// Заменяет URL'ы вне ссылок и инлайн-кода на <a>.
function autoLink(text) {
  const parts = linkifyParts(text)
  return parts.map((p) => {
    if (p.type === 'link') {
      return `<a href="${escapeAttr(p.href)}" target="_blank" rel="noopener noreferrer" class="md-link">${escapeHtml(p.value)}</a>`
    }
    return p.value
  }).join('')
}

// inline-парсер — на уже экранированной строке. Готовые HTML-фрагменты
// (код, картинки, ссылки) прячутся в слоты \x00N\x00 — их не трогают ни
// остальные замены, ни автолинки; \x00 в экранированном тексте невозможен.
function parseInline(text) {
  const slots = []
  const stash = (html) => {
    slots.push(html)
    return `\x00${slots.length - 1}\x00`
  }

  let s = text.replace(/`([^`\n]+)`/g, (_, code) => stash(`<code class="md-code">${code}</code>`))

  s = s.replace(/!\[([^\]]*)\]\(([^)\s]+)\)/g, (_, alt, url) =>
    stash(`<img class="md-img" src="${escapeAttr(safeUrl(url))}" alt="${escapeAttr(alt)}" loading="lazy">`))

  s = s.replace(/\*\*([^*\n]+)\*\*/g, '<strong>$1</strong>')
  s = s.replace(/(^|[^*])\*([^*\n]+)\*/g, '$1<em>$2</em>')
  s = s.replace(/~~([^~\n]+)~~/g, '<s>$1</s>')

  s = s.replace(/\[([^\]]+)\]\(([^)\s]+)\)/g, (_, label, url) =>
    stash(`<a href="${escapeAttr(safeUrl(url))}" target="_blank" rel="noopener noreferrer" class="md-link">${label}</a>`))

  // Хештеги #тег (как в соцсетях) → кликабельный чип. Сташим готовый HTML,
  // чтобы автолинк и остальные правила его не трогали. Зеркалит серверный
  // tagRe в portalsvc: # не в середине слова/URL, 2..50 букв/цифр/_.
  s = s.replace(TAG_RE, (_, pre, tag) =>
    pre + stash(`<a class="md-tag" data-tag="${escapeAttr(tag.toLowerCase())}">#${tag}</a>`))

  // @упоминания → кликабельный чип (открывает карточку пользователя). Только
  // при включённой опции — см. MENTION_RE.
  if (mentionsEnabled) {
    s = s.replace(MENTION_RE, (_, pre, login) => {
      const key = login.toLowerCase()
      const name = mentionNames[key] || login // нет в каталоге → показываем логин
      return pre + stash(`<a class="md-mention" data-mention="${escapeAttr(key)}">@${escapeHtml(name)}</a>`)
    })
  }

  s = autoLink(s)

  return s.replace(/\x00(\d+)\x00/g, (_, idx) => slots[+idx])
}

// Собирает <ul>/<ol> из строк-элементов; чек-листы — <li class="md-task">.
function renderList(items, ordered) {
  const tag = ordered ? 'ol' : 'ul'
  const body = items.map(({ text, task, checked }) => {
    if (task) {
      return `<li class="md-task"><input type="checkbox" disabled${checked ? ' checked' : ''}>` +
        `<span>${parseInline(text)}</span></li>`
    }
    return `<li>${parseInline(text)}</li>`
  }).join('')
  return `<${tag} class="md-list">${body}</${tag}>`
}

function splitRow(line) {
  const cells = line.split('|').map((c) => c.trim())
  if (cells[0] === '') cells.shift()
  if (cells.length && cells[cells.length - 1] === '') cells.pop()
  return cells
}

export function renderMarkdown(src, opts = {}) {
  if (!src) return ''
  mentionsEnabled = !!opts.mentions
  mentionNames = opts.mentionNames || {}
  const text = escapeHtml(String(src))
  const lines = text.split('\n')
  const out = []
  let i = 0
  let para = []

  const flushPara = () => {
    if (!para.length) return
    out.push(`<p>${parseInline(para.join('<br>'))}</p>`)
    para = []
  }

  while (i < lines.length) {
    const line = lines[i]

    // Code fence ```
    if (/^```/.test(line)) {
      flushPara()
      const buf = []
      i++
      while (i < lines.length && !/^```/.test(lines[i])) {
        buf.push(lines[i])
        i++
      }
      i++ // закрывающий fence
      out.push(`<pre class="md-pre"><code>${buf.join('\n')}</code></pre>`)
      continue
    }

    // Heading
    const m = line.match(/^(#{1,3})\s+(.+)$/)
    if (m) {
      flushPara()
      const lvl = m[1].length
      out.push(`<h${lvl} class="md-h md-h${lvl}">${parseInline(m[2])}</h${lvl}>`)
      i++
      continue
    }

    // Горизонтальная линейка
    if (/^\s*(-{3,}|\*{3,}|_{3,})\s*$/.test(line)) {
      flushPara()
      out.push('<hr class="md-hr">')
      i++
      continue
    }

    // Цитата (>: после экранирования — &gt;)
    if (/^&gt;\s?/.test(line)) {
      flushPara()
      const buf = []
      while (i < lines.length && /^&gt;\s?/.test(lines[i])) {
        buf.push(lines[i].replace(/^&gt;\s?/, ''))
        i++
      }
      out.push(`<blockquote class="md-quote">${parseInline(buf.join('<br>'))}</blockquote>`)
      continue
    }

    // Списки: -/*/+ или 1./1); чек-лист - [ ] / - [x]
    const li = line.match(/^\s*(?:([-*+])|(\d+)[.)])\s+(.*)$/)
    if (li) {
      flushPara()
      const ordered = !!li[2]
      const items = []
      while (i < lines.length) {
        const lm = lines[i].match(/^\s*(?:([-*+])|(\d+)[.)])\s+(.*)$/)
        if (!lm || !!lm[2] !== ordered) break
        const task = lm[3].match(/^\[( |x|X)\]\s+(.*)$/)
        items.push(task
          ? { text: task[2], task: true, checked: task[1] !== ' ' }
          : { text: lm[3] })
        i++
      }
      out.push(renderList(items, ordered))
      continue
    }

    // Таблица: строка |…| и следом разделитель |---|---|
    if (/^\|.*\|\s*$/.test(line) && i + 1 < lines.length && /^\|[\s\-:|]+\|\s*$/.test(lines[i + 1])) {
      flushPara()
      const head = splitRow(line)
      i += 2
      const rows = []
      while (i < lines.length && /^\|.*\|\s*$/.test(lines[i])) {
        rows.push(splitRow(lines[i]))
        i++
      }
      const thead = `<thead><tr>${head.map((c) => `<th>${parseInline(c)}</th>`).join('')}</tr></thead>`
      const tbody = rows.length
        ? `<tbody>${rows.map((r) => `<tr>${head.map((_, ci) => `<td>${parseInline(r[ci] ?? '')}</td>`).join('')}</tr>`).join('')}</tbody>`
        : ''
      out.push(`<div class="md-table-wrap"><table class="md-table">${thead}${tbody}</table></div>`)
      continue
    }

    if (line.trim() === '') {
      flushPara()
      i++
      continue
    }

    para.push(line)
    i++
  }
  flushPara()
  return out.join('')
}

/* Плоский текст из Markdown — для ОДНОСТРОЧНЫХ превью (список чатов и т.п.):
   разметку вычищаем, а не рендерим (блочные элементы в строку с многоточием
   не укладываются). Зеркало stripMarkdown в portalsvc (превью пересылки). */
export function stripMarkdown(src) {
  if (!src) return ''
  return String(src)
    .replace(/```[^\n`]*\n?/g, '')              // код-фенсы (маркеры)
    .replace(/!\[([^\]]*)\]\([^)]*\)/g, '$1')   // картинки → alt
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')    // ссылки → текст
    .replace(/^#{1,3}\s+/gm, '')                // заголовки
    .replace(/^>\s?/gm, '')                     // цитаты
    .replace(/^[-*+]\s+\[[ xX]\]\s+/gm, '')     // чек-листы
    .replace(/^[-*+]\s+/gm, '')                 // маркированные списки
    .replace(/^\d+\.\s+/gm, '')                 // нумерованные списки
    .replace(/(\*\*|__)(.+?)\1/g, '$2')         // жирный
    .replace(/(\*|_)(.+?)\1/g, '$2')            // курсив
    .replace(/~~(.+?)~~/g, '$1')                // зачёркнутый
    .replace(/`([^`]*)`/g, '$1')                // инлайн-код
    .replace(/\s+/g, ' ')                       // превью — одна строка
    .trim()
}
