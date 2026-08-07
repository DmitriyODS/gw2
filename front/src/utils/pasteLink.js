/* Вставка ссылки НА ВЫДЕЛЕННЫЙ текст.

   Привычка из редакторов: выделил слово, нажал Ctrl+V с адресом в буфере —
   слово стало ссылкой. Без этого приходится писать разметку руками, а в чате
   ещё и помнить её синтаксис.

   Правило одно: подменяем вставку, только если выделение непустое И в буфере
   именно адрес. Всё остальное (пустое выделение, обычный текст, адрес поверх
   адреса) — обычная вставка, её делает сам браузер. */

import { parseUrl } from '@/utils/webSearch.js'

/** Ссылка из буфера или null, если там не адрес. */
export function clipboardLink(clipboardText) {
  const raw = String(clipboardText || '').trim()
  if (!raw) return null
  return parseUrl(raw)?.href || null
}

/**
 * Markdown-поля (мессенджер, портал): выделение превращается в `[текст](адрес)`.
 * Возвращает { value, caret } — новое значение поля и куда поставить курсор,
 * либо null, если применять нечего.
 */
export function linkifySelection(el, clipboardText) {
  const href = clipboardLink(clipboardText)
  if (!href || !el) return null

  const from = el.selectionStart ?? 0
  const to = el.selectionEnd ?? 0
  if (from === to) return null // нечего оборачивать — обычная вставка

  // Читаем DOM-значение, а не модель: у части клавиатур (IME) модель отстаёт
  // на один ввод, и подстановка ушла бы не в тот текст.
  const value = el.value ?? ''
  const selected = value.slice(from, to)
  // Выделен уже готовый адрес — человек хочет его заменить, а не вложить
  // ссылку в ссылку.
  if (clipboardLink(selected)) return null

  const inserted = `[${selected}](${href})`
  return {
    value: value.slice(0, from) + inserted + value.slice(to),
    caret: from + inserted.length,
  }
}
