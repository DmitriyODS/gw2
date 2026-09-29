/**
 * «Меньше прозрачности»: плотные подложки вместо размытого стекла.
 *
 * Каждое стекло — это backdrop-filter, который GPU пересчитывает при любом
 * изменении под ним; на слабой видеокарте и от батареи это заметно. Режим
 * ставит на <html> атрибут `data-reduce-transparency` — по нему tokens.css
 * делает акрил непрозрачным и снимает размытие со всех элементов.
 *
 * По умолчанию следует системной настройке (`prefers-reduced-transparency`),
 * явный выбор человека хранится на УСТРОЙСТВЕ (localStorage): это про железо,
 * а не про вкус, как и раскладка каркаса.
 */
import { ref } from 'vue'

const KEY = 'gw_reduce_transparency'

function readChoice() {
  try {
    const v = localStorage.getItem(KEY)
    return v === 'on' || v === 'off' ? v : null
  } catch {
    return null
  }
}

const mq = typeof window !== 'undefined' && typeof window.matchMedia === 'function'
  ? window.matchMedia('(prefers-reduced-transparency: reduce)')
  : null

/** Явный выбор: 'on' | 'off' | null (как в системе). */
export const transparencyChoice = ref(readChoice())
/** Действует ли режим сейчас. */
export const reduceTransparency = ref(false)

function apply() {
  const on = transparencyChoice.value ? transparencyChoice.value === 'on' : !!mq?.matches
  reduceTransparency.value = on
  if (typeof document !== 'undefined') {
    document.documentElement.toggleAttribute('data-reduce-transparency', on)
  }
}

let installed = false

/** Применить при старте и следить за системной настройкой. */
export function initTransparency() {
  apply()
  if (installed || !mq) return
  installed = true
  mq.addEventListener?.('change', apply)
}

/** Явно включить/выключить; null — вернуть «как в системе». */
export function setReduceTransparency(value) {
  transparencyChoice.value = value === null ? null : (value ? 'on' : 'off')
  try {
    if (transparencyChoice.value) localStorage.setItem(KEY, transparencyChoice.value)
    else localStorage.removeItem(KEY)
  } catch { /* приватный режим — выбор живёт до перезагрузки */ }
  apply()
}
