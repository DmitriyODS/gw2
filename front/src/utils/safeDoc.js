/**
 * Документ TipTap без ключей `__proto__` на любой глубине.
 *
 * TipTap v2 (mergeAttributes) превращает такой ключ в унаследованные атрибуты
 * DOM — чужой документ с `attrs.__proto__.onerror` исполнил бы обработчик у
 * открывшего заметку. Сервер вырезает ключ при сохранении (notesvc
 * `SanitizeDoc`); здесь — второй рубеж для кадров соавторов и старых записей.
 */
export function safeDoc(doc) {
  if (!doc || typeof doc !== 'object') return doc
  if (Array.isArray(doc)) return doc.map(safeDoc)
  const out = {}
  for (const key of Object.keys(doc)) {
    if (key === '__proto__') continue
    out[key] = safeDoc(doc[key])
  }
  return out
}
