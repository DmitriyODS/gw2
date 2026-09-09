/* Раскладка списка раздела: закреплённые, папки, остальные.

   Организация списка — ЛИЧНАЯ (см. stores/listPrefs.js): в одной панели
   соседствуют свои и расшаренные записи, и раскладывает их каждый под себя.
   Поэтому здесь чистые функции над «настройки + элементы», без запросов и
   без знания о конкретном разделе.

   Порядок хранится ОДНИМ плоским списком id на раздел, а не по группам:
   перетаскивание всегда даёт готовую последовательность видимых пунктов, и
   один массив избавляет от рассинхрона «порядок в папке ≠ порядок группы».
   Элементы, которых в нём ещё нет (только что созданные, чужие), идут после
   упорядоченных в том порядке, в каком их отдал сервер. */

export const PINNED_KEY = 'pinned'
export const LOOSE_KEY = 'loose'

/** Ключ элемента в настройках: id всегда строкой — ключи JSON строковые. */
export const keyOf = (item, idKey = 'id') => String(item?.[idKey] ?? '')

/**
 * Группы списка в порядке показа: закреплённые, папки, остальное.
 * Пустая папка остаётся в списке (в неё перетаскивают), пустые «закреплённые»
 * и «остальное» — нет.
 */
export function organizeList(items = [], prefs = {}, idKey = 'id') {
  const folders = Array.isArray(prefs.folders) ? prefs.folders : []
  const pinned = new Set((prefs.pinned || []).map(String))
  const folderOf = prefs.folderOf || {}
  const rank = new Map((prefs.order || []).map((id, i) => [String(id), i]))
  const known = new Set(folders.map((f) => String(f.id)))

  const sorted = items
    .map((item, i) => ({ item, i, key: keyOf(item, idKey) }))
    .sort((a, b) => {
      const ra = rank.has(a.key) ? rank.get(a.key) : Infinity
      const rb = rank.has(b.key) ? rank.get(b.key) : Infinity
      return ra === rb ? a.i - b.i : ra - rb
    })
    .map((x) => x.item)

  const pinnedItems = []
  const loose = []
  const byFolder = new Map(folders.map((f) => [String(f.id), []]))

  for (const item of sorted) {
    const key = keyOf(item, idKey)
    if (pinned.has(key)) { pinnedItems.push(item); continue }
    // Папку могли удалить в другой вкладке — элемент возвращается в общий список.
    const fid = folderOf[key] != null ? String(folderOf[key]) : ''
    if (fid && known.has(fid)) byFolder.get(fid).push(item)
    else loose.push(item)
  }

  /* Сворачивается ЛЮБАЯ группа, а не только папка: `collapseKey` — под каким
     ключом её состояние живёт в настройках (у папки это её id, у закреплённых
     и остальных — свои ключи; id папок начинаются с «f», поэтому не пересекутся). */
  const collapsedOf = (key) => !!prefs.collapsed?.[key]

  const groups = []
  if (pinnedItems.length) {
    groups.push({
      key: PINNED_KEY,
      kind: 'pinned',
      id: null,
      collapseKey: PINNED_KEY,
      name: 'Закреплённые',
      collapsed: collapsedOf(PINNED_KEY),
      items: pinnedItems,
    })
  }
  for (const f of folders) {
    const id = String(f.id)
    groups.push({
      key: `f:${id}`,
      kind: 'folder',
      id,
      collapseKey: id,
      name: f.name || 'Папка',
      collapsed: collapsedOf(id),
      items: byFolder.get(id) || [],
    })
  }
  if (loose.length) {
    groups.push({
      key: LOOSE_KEY,
      kind: 'loose',
      id: null,
      collapseKey: LOOSE_KEY,
      name: 'Остальные',
      collapsed: collapsedOf(LOOSE_KEY),
      items: loose,
    })
  }
  return groups
}

/** Плоская последовательность ключей в порядке показа — она и есть `order`.
    Свёрнутая группа своих пунктов из порядка не теряет: он про весь список. */
export function flattenGroups(groups, idKey = 'id') {
  return groups.flatMap((g) => g.items.map((item) => keyOf(item, idKey)))
}

/**
 * Перестановка в плоском порядке: `moving` встаёт перед `target` (или после
 * него при before=false). Отсутствующий в списке `moving` просто добавляется —
 * так в порядок попадают новые элементы, которых там ещё не было.
 */
export function moveKey(ids, moving, target, before = true) {
  const next = ids.filter((id) => id !== moving)
  if (moving === target || !target) {
    next.push(moving)
    return next
  }
  const at = next.indexOf(target)
  if (at === -1) { next.push(moving); return next }
  next.splice(before ? at : at + 1, 0, moving)
  return next
}

/**
 * Сдвиг элемента на шаг вверх/вниз ВНУТРИ своей группы (пункт меню и тач, где
 * перетаскивания нет): меняется местами с соседом по группе, а не по всему
 * списку — иначе пункт «уходил» бы в чужую папку.
 */
export function shiftKey(ids, moving, dir, groupKeys) {
  const at = groupKeys.indexOf(moving)
  const to = at + dir
  if (at === -1 || to < 0 || to >= groupKeys.length) return ids
  return moveKey(ids, moving, groupKeys[to], dir < 0)
}
