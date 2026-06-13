// Транслитерация кириллицы в латиницу и генерация логина по ФИО.

const MAP = {
  а: 'a', б: 'b', в: 'v', г: 'g', д: 'd', е: 'e', ё: 'e', ж: 'zh', з: 'z',
  и: 'i', й: 'y', к: 'k', л: 'l', м: 'm', н: 'n', о: 'o', п: 'p', р: 'r',
  с: 's', т: 't', у: 'u', ф: 'f', х: 'h', ц: 'ts', ч: 'ch', ш: 'sh',
  щ: 'shch', ъ: '', ы: 'y', ь: '', э: 'e', ю: 'yu', я: 'ya',
}

export function translit(text) {
  return String(text || '')
    .toLowerCase()
    .split('')
    .map(ch => (ch in MAP ? MAP[ch] : ch))
    .join('')
}

// Логин по ФИО: до 6 букв фамилии, точка и по одной (транслитерированной)
// букве остальных слов, всё в нижнем регистре.
// «Иванов Дмитрий Сергеевич» → «ivanov.ds».
export function loginFromFio(fio) {
  const words = String(fio || '').trim().split(/\s+/).filter(Boolean)
  if (!words.length) return ''
  const clean = s => translit(s).replace(/[^a-z0-9]/g, '')
  const surname = clean(words[0]).slice(0, 6)
  const initials = words.slice(1).map(w => clean(w[0])).join('')
  return initials ? `${surname}.${initials}` : surname
}
