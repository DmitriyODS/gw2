/**
 * Индекс высот строк виртуального списка: смещение строки и строка по
 * смещению. Высоты — измеренные (по ключу строки) либо оценочные, пока
 * строка ни разу не была на экране. Префиксные суммы пересобираются лениво:
 * строк в ленте тысячи, а меняются они пачками (замер, догрузка истории).
 */
export class HeightIndex {
  constructor(estimate) {
    this.estimate = estimate
    this.measured = new Map()
    this.keys = []
    this.prefix = [0]
    this.dirty = true
  }

  setKeys(keys) {
    this.keys = keys
    this.dirty = true
  }

  /** Запомнить высоту; возвращает разницу с прежней (0 — не изменилась). */
  set(key, height, index) {
    const prev = this.heightOf(key, index)
    this.measured.set(key, height)
    if (prev !== height) this.dirty = true
    return height - prev
  }

  heightOf(key, index) {
    const h = this.measured.get(key)
    return h === undefined ? this.estimate(index) : h
  }

  /** Забыть высоты строк, которых больше нет в списке. */
  prune() {
    if (this.measured.size <= this.keys.length) return
    const alive = new Set(this.keys)
    for (const k of this.measured.keys()) if (!alive.has(k)) this.measured.delete(k)
  }

  rebuild() {
    if (!this.dirty) return
    const n = this.keys.length
    const prefix = new Array(n + 1)
    prefix[0] = 0
    for (let i = 0; i < n; i++) prefix[i + 1] = prefix[i] + this.heightOf(this.keys[i], i)
    this.prefix = prefix
    this.dirty = false
  }

  get total() {
    this.rebuild()
    return this.prefix[this.keys.length]
  }

  /** Смещение верха строки i от начала списка. */
  offsetOf(i) {
    this.rebuild()
    return this.prefix[Math.max(0, Math.min(i, this.keys.length))]
  }

  /** Строка, в которую попадает смещение (последняя — если за концом). */
  indexAt(offset) {
    this.rebuild()
    const n = this.keys.length
    if (n === 0 || offset <= 0) return 0
    let lo = 0
    let hi = n - 1
    while (lo < hi) {
      const mid = (lo + hi + 1) >> 1
      if (this.prefix[mid] <= offset) lo = mid
      else hi = mid - 1
    }
    return lo
  }
}
