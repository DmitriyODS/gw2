/* Деньги и объёмы магазина.
   Сервер держит суммы в КОПЕЙКАХ (целые), поэтому округления с плавающей
   точкой в цене невозможны — здесь только форматирование. */

/** «299 руб.» / «2 388 руб.» / «Бесплатно» (0). */
export function formatPrice(kopecks, { free = 'Бесплатно' } = {}) {
  if (!kopecks) return free
  const rub = Math.round(kopecks / 100)
  const rest = kopecks % 100
  const head = rub.toLocaleString('ru-RU')
  return rest ? `${(kopecks / 100).toLocaleString('ru-RU', { minimumFractionDigits: 2 })} руб.` : `${head} руб.`
}

/** Цена за месяц при годовой оплате: «199 руб./мес». */
export function formatPerMonth(kopecks) {
  return `${Math.round(kopecks / 100 / 12).toLocaleString('ru-RU')} руб./мес`
}

/** «4.5 Гб» — как в карточке хранилища на макете. */
export function formatBytes(bytes) {
  if (bytes == null) return '—'
  if (bytes < 0) return '∞'
  const units = ['Б', 'Кб', 'Мб', 'Гб', 'Тб']
  let value = bytes
  let i = 0
  while (value >= 1024 && i < units.length - 1) {
    value /= 1024
    i += 1
  }
  // Дробную часть показываем, только когда она есть: «10 Гб», но «4,5 Гб».
  const digits = i > 1 && Math.abs(value - Math.round(value)) >= 0.05 ? 1 : 0
  return `${value.toFixed(digits).replace('.', ',')} ${units[i]}`
}

/** Число с разделителями разрядов: 10 000 токенов. */
export function formatCount(value) {
  if (value == null) return '—'
  if (value < 0) return '∞'
  return Number(value).toLocaleString('ru-RU')
}

/** Доля заполнения шкалы 0..1 (безлимит — всегда пусто). */
export function usageRatio(used, limit) {
  if (limit == null || limit < 0 || limit === 0) return 0
  return Math.min(1, Math.max(0, used / limit))
}

/** «до 3 сентября» — срок действия подписки. */
export function formatUntil(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric' })
}
