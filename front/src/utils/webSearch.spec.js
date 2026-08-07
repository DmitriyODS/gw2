import { describe, it, expect, beforeEach } from 'vitest'
import {
  SEARCH_ENGINES, DEFAULT_ENGINE, enginesInOrder, getSearchEngine, setSearchEngine, searchUrl, parseUrl,
} from './webSearch.js'
import { loadHistory, pushHistory, removeHistory, clearHistory } from './holaHistory.js'

describe('поиск в интернете из Hola', () => {
  beforeEach(() => localStorage.clear())

  it('по умолчанию ищет Яндексом', () => {
    expect(DEFAULT_ENGINE).toBe('yandex')
    expect(getSearchEngine()).toBe('yandex')
    expect(enginesInOrder()[0].key).toBe('yandex')
  })

  it('выбранный поисковик запоминается и встаёт первым', () => {
    setSearchEngine('google')
    expect(getSearchEngine()).toBe('google')
    expect(enginesInOrder().map((e) => e.key)).toEqual(['google', 'yandex', 'duckduckgo'])
  })

  it('незнакомый поисковик не сохраняется — остаётся прежний', () => {
    setSearchEngine('bing')
    expect(getSearchEngine()).toBe('yandex')
  })

  it('запрос уезжает в адрес закодированным', () => {
    expect(searchUrl('yandex', 'отпуск по ТК РФ'))
      .toBe('https://yandex.ru/search/?text=' + encodeURIComponent('отпуск по ТК РФ'))
    // Все поисковики выдачи знают, как принять запрос.
    for (const e of SEARCH_ENGINES) expect(searchUrl(e.key, 'x')).toBe(`${e.url}x`)
  })
})

describe('адрес сайта в строке Hola', () => {
  it('узнаёт домен без протокола', () => {
    expect(parseUrl('vk.com')).toEqual({ href: 'https://vk.com/', label: 'vk.com' })
    expect(parseUrl('дом.рф').href).toContain('xn--')       // IDN нормализуется браузером
    expect(parseUrl('github.com/user/repo?tab=1').href).toBe('https://github.com/user/repo?tab=1')
  })

  it('явный протокол принимает как есть', () => {
    expect(parseUrl('https://ya.ru/maps').href).toBe('https://ya.ru/maps')
    expect(parseUrl('http://localhost:5173').href).toBe('http://localhost:5173/')
  })

  it('обычный запрос адресом не считает', () => {
    expect(parseUrl('отчёт по задачам')).toBeNull()   // пробелы
    expect(parseUrl('2+2*2')).toBeNull()
    expect(parseUrl('1.5')).toBeNull()                // домен верхнего уровня — буквы
    expect(parseUrl('ivan@example.com')).toBeNull()   // почта, а не сайт
    expect(parseUrl('')).toBeNull()
  })

  it('имя файла не путает с доменом', () => {
    expect(parseUrl('смета.pdf')).toBeNull()
    expect(parseUrl('HolaPanel.vue')).toBeNull()
    // …но сайт с таким же хвостом в пути остаётся адресом.
    expect(parseUrl('example.com/смета.pdf').href).toBe('https://example.com/%D1%81%D0%BC%D0%B5%D1%82%D0%B0.pdf')
  })
})

describe('история запросов Hola', () => {
  beforeEach(() => localStorage.clear())

  it('новый запрос встаёт наверх', () => {
    pushHistory('первый')
    const rows = pushHistory('второй')
    expect(rows.map((r) => r.text)).toEqual(['второй', 'первый'])
  })

  it('повтор поднимает прежнюю строку, а не плодит дубль', () => {
    pushHistory('отчёт')
    pushHistory('задачи')
    const rows = pushHistory('Отчёт')
    expect(rows.map((r) => r.text)).toEqual(['Отчёт', 'задачи'])
  })

  it('хранит ограниченное число последних запросов', () => {
    for (let i = 0; i < 30; i += 1) pushHistory(`запрос ${i}`)
    expect(loadHistory()).toHaveLength(12)
    expect(loadHistory()[0].text).toBe('запрос 29')
  })

  it('строку можно убрать, а историю очистить', () => {
    pushHistory('первый')
    pushHistory('второй')
    expect(removeHistory('второй').map((r) => r.text)).toEqual(['первый'])
    expect(clearHistory()).toEqual([])
    expect(loadHistory()).toEqual([])
  })

  it('пустой запрос в историю не попадает', () => {
    expect(pushHistory('   ')).toEqual([])
  })
})
