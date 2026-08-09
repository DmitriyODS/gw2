/**
 * Подпись человека: «Фамилия И.О.».
 *
 * Полное ФИО в три слова не помещается ни в колонку, ни в строку списка и
 * ломается на две строки в самом неудобном месте. Форма с инициалами — то, как
 * подписывают людей в документах, и она предсказуемо коротка.
 *
 * Разбираем без предположений о числе слов: первое — фамилия, из остальных
 * берём по букве. Одно слово (логин, название) остаётся как есть.
 *
 * @param {string} fio
 * @returns {string}
 */
export function shortFio(fio) {
  const words = String(fio || '').trim().split(/\s+/).filter(Boolean)
  if (words.length < 2) return words[0] || ''
  const initials = words.slice(1)
    .map((w) => `${w[0].toUpperCase()}.`)
    .join('')
  return `${words[0]} ${initials}`
}
