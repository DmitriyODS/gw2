/**
 * Поиск в интернете из Hola.
 *
 * Выдачу мы не проксируем: строка результата ведёт на страницу поисковика,
 * которая открывается новой вкладкой браузера (встроить её в окно рабочего
 * стола нельзя — поисковики запрещают фрейминг). Поисковик по умолчанию
 * выбирает пользователь; остальные остаются в выдаче вторыми строками.
 */
import { storageGet, storageSet } from './storage.js'

const ENGINE_KEY = 'gw_hola_engine'

export const SEARCH_ENGINES = [
  { key: 'yandex', label: 'Яндекс', url: 'https://yandex.ru/search/?text=' },
  { key: 'google', label: 'Google', url: 'https://www.google.com/search?q=' },
  { key: 'duckduckgo', label: 'DuckDuckGo', url: 'https://duckduckgo.com/?q=' },
]

export const DEFAULT_ENGINE = 'yandex'

const byKey = new Map(SEARCH_ENGINES.map((e) => [e.key, e]))

export function engineByKey(key) {
  return byKey.get(key) || byKey.get(DEFAULT_ENGINE)
}

/** Поисковик по умолчанию (личная настройка устройства). */
export function getSearchEngine() {
  const key = storageGet(ENGINE_KEY, DEFAULT_ENGINE)
  return byKey.has(key) ? key : DEFAULT_ENGINE
}

export function setSearchEngine(key) {
  if (byKey.has(key)) storageSet(ENGINE_KEY, key)
}

export function searchUrl(engineKey, query) {
  return engineByKey(engineKey).url + encodeURIComponent(String(query || '').trim())
}

/**
 * Поисковики в порядке выдачи: сначала выбранный по умолчанию, за ним прочие.
 */
export function enginesInOrder(defaultKey = getSearchEngine()) {
  const first = engineByKey(defaultKey)
  return [first, ...SEARCH_ENGINES.filter((e) => e.key !== first.key)]
}

/** Открывает выдачу новой вкладкой (noopener — вкладка не получит доступ к нам). */
export function openWebSearch(engineKey, query) {
  openUrl(searchUrl(engineKey, query))
}

export function openUrl(href) {
  window.open(href, '_blank', 'noopener,noreferrer')
}

/* Домен верхнего уровня: буквы (в т.ч. кириллица — .рф) длиной 2..24 либо
   punycode-форма (xn--…). Часть имён файлов выглядит так же («смета.pdf»),
   поэтому частые расширения исключаем — их вводят как название документа,
   а не как адрес. */
const HOST = /^[\p{L}\d][\p{L}\d-]*(\.[\p{L}\d][\p{L}\d-]*)*\.(xn--[\p{L}\d-]+|\p{L}{2,24})$/u
const FILE_EXT = new Set([
  'txt', 'md', 'pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'csv', 'zip', 'rar',
  'png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'mp3', 'mp4', 'mov', 'js', 'ts', 'vue',
  'py', 'go', 'json', 'yml', 'yaml', 'html', 'css', 'sql', 'sh', 'log', 'apk', 'exe',
])

/**
 * Распознаёт в строке адрес сайта: «vk.com», «https://ya.ru/maps»,
 * «github.com/user/repo?tab=1». Возвращает { href, label } либо null, если это
 * обычный запрос.
 */
export function parseUrl(input) {
  const text = String(input || '').trim()
  if (!text || /\s/.test(text) || text.includes('@')) return null

  const explicit = /^https?:\/\//i.test(text)
  const withScheme = explicit ? text : `https://${text}`

  let url
  try {
    url = new URL(withScheme)
  } catch {
    return null
  }

  // Явный http(s):// пользователь написал сам — такой адрес принимаем как есть.
  // Домен проверяем по ИСХОДНОЙ строке: в url.hostname кириллица уже свёрнута
  // в punycode («дом.рф» → «xn--d1aqf.xn--p1ai»), и правило про буквенный TLD
  // на него не ложится.
  if (!explicit) {
    const host = text.split(/[/?#]/)[0].split(':')[0].toLowerCase()
    if (!HOST.test(host)) return null
    if (FILE_EXT.has(host.split('.').pop())) return null
  }

  return { href: url.href, label: text.replace(/^https?:\/\//i, '').replace(/\/$/, '') }
}
