/* «Часто выбираемое» в выпадающих списках.

   Справочники сортируются по алфавиту — это удобно, пока их пять, и мучительно,
   когда полсотни: человек каждый раз ищет в списке те же два-три значения.
   Здесь ведётся ЛИЧНЫЙ счётчик выбора (localStorage), по нему список
   поднимает наверх то, чем пользуются чаще.

   Хранилище личное и на устройстве: это подсказка интерфейса, а не данные
   компании — синхронизировать её между устройствами незачем. */

const STORAGE_KEY = 'gw_recent_picks'

// Сколько значений помним на один справочник: длинный хвост редких выборов
// только мешает и раздувает localStorage.
const MAX_ENTRIES = 30

// Вес последнего выбора: свежее решение важнее давней привычки, поэтому к
// счётчику добавляется небольшая надбавка за недавность (в днях).
const RECENCY_DAYS = 30

function readAll() {
  try {
    const raw = JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}')
    return raw && typeof raw === 'object' ? raw : {}
  } catch {
    return {}
  }
}

function writeAll(data) {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data))
  } catch { /* приватный режим — подсказка просто не запомнится */ }
}

/** Ключ ветки хранилища: вид справочника + скоуп (обычно компания). */
function bucketKey(kind, scope) {
  return `${kind}:${scope ?? 0}`
}

/** Запомнить выбор: +1 к счётчику значения. */
export function rememberPick(kind, scope, id) {
  if (id == null) return
  const all = readAll()
  const key = bucketKey(kind, scope)
  const bucket = all[key] || {}
  const prev = bucket[id] || { n: 0, at: 0 }
  bucket[id] = { n: prev.n + 1, at: Date.now() }

  // Обрезаем хвост: оставляем самые частые (при равенстве — свежие).
  const entries = Object.entries(bucket)
  if (entries.length > MAX_ENTRIES) {
    entries.sort((a, b) => (b[1].n - a[1].n) || (b[1].at - a[1].at))
    all[key] = Object.fromEntries(entries.slice(0, MAX_ENTRIES))
  } else {
    all[key] = bucket
  }
  writeAll(all)
}

/** Вес значения: чем чаще и свежее выбирали, тем выше. 0 — не выбирали. */
export function pickScore(kind, scope, id) {
  const rec = readAll()[bucketKey(kind, scope)]?.[id]
  if (!rec) return 0
  const days = (Date.now() - (rec.at || 0)) / 86_400_000
  const freshness = Math.max(0, (RECENCY_DAYS - days) / RECENCY_DAYS)
  return rec.n + freshness
}

/**
 * Список, разложенный на «часто выбираемое» и остальное.
 * items — массив объектов, idKey — поле идентификатора, nameKey — подпись.
 * Возвращает { frequent, rest }: frequent отсортирован по весу, rest — по
 * алфавиту (привычный порядок справочника сохраняется).
 */
export function splitByFrequency(items, { kind, scope, idKey = 'id', nameKey = 'name', limit = 5 } = {}) {
  const scored = items
    .map((item) => ({ item, score: pickScore(kind, scope, item[idKey]) }))
    .filter((row) => row.score > 0)
    .sort((a, b) => b.score - a.score)
    .slice(0, limit)

  const top = new Set(scored.map((row) => row.item[idKey]))
  const byName = (a, b) => String(a[nameKey] || '').localeCompare(String(b[nameKey] || ''), 'ru')
  return {
    frequent: scored.map((row) => row.item),
    rest: items.filter((item) => !top.has(item[idKey])).sort(byName),
  }
}
