/**
 * Очерёдность обращений к ключам кэша с пределом: `touch(key)` поднимает ключ,
 * а вытесненные сверх предела отдаёт `onEvict`. Сами данные живут там, где их
 * читают (реактивные карты сторов), — здесь только порядок.
 *
 * Приложение живёт днями (десктоп в трее), и карты «по сущности» без предела
 * копили всё когда-либо открытое.
 */
export function createKeyLru(limit, onEvict) {
  const order = []
  return {
    touch(key) {
      const at = order.indexOf(key)
      if (at >= 0) order.splice(at, 1)
      order.push(key)
      while (order.length > limit) onEvict(order.shift())
    },
    clear() {
      order.length = 0
    },
  }
}
