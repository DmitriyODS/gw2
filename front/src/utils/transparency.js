/**
 * Стекло интерфейса — две независимые настройки устройства:
 *
 * - «Размытие» (`data-no-blur` на <html>): снимает backdrop-filter со всех
 *   элементов, панели остаются полупрозрачными. Размытие — самая дорогая
 *   часть стекла: GPU пересчитывает его при любом изменении под панелью.
 * - «Прозрачность» (`data-opaque`): окна, панели и прочие поверхности
 *   становятся плотными, обои под ними не видны. Размытие под плотной панелью
 *   бессмысленно, поэтому снимается вместе с ней.
 *
 * Материал платформы — матовый soft touch, поэтому по умолчанию обе ВЫКЛЮЧЕНЫ:
 * поверхности плотные и шероховатые (зерно в токенах), стекло — выбор
 * любителя. Явный выбор хранится на УСТРОЙСТВЕ (localStorage) — стекло
 * стоит видеокарте и батарее, это про железо, а не про вкус.
 * Значения токенов — в конце tokens.css.
 */
import { ref } from 'vue'

const BLUR_KEY = 'gw_glass_blur'
const OPACITY_KEY = 'gw_glass_transparency'

function readChoice(key) {
  try {
    const v = localStorage.getItem(key)
    return v === 'on' || v === 'off' ? v : null
  } catch {
    return null
  }
}

function writeChoice(key, value) {
  try {
    if (value) localStorage.setItem(key, value)
    else localStorage.removeItem(key)
  } catch { /* приватный режим — выбор живёт до перезагрузки */ }
}

/** Явный выбор: 'on' | 'off' | null (по умолчанию — матовый материал). */
export const blurChoice = ref(readChoice(BLUR_KEY))
export const transparencyChoice = ref(readChoice(OPACITY_KEY))
/** Действующие значения. */
export const blurEnabled = ref(false)
export const transparencyEnabled = ref(false)

const effective = (choice) => choice === 'on'

function apply() {
  transparencyEnabled.value = effective(transparencyChoice.value)
  blurEnabled.value = transparencyEnabled.value && effective(blurChoice.value)
  if (typeof document === 'undefined') return
  const root = document.documentElement
  root.toggleAttribute('data-opaque', !transparencyEnabled.value)
  root.toggleAttribute('data-no-blur', !blurEnabled.value)
}

/** Применить при старте. */
export function initTransparency() {
  apply()
}

/** Размытие: true/false — явно, null — по умолчанию. */
export function setBlur(value) {
  blurChoice.value = value === null ? null : (value ? 'on' : 'off')
  writeChoice(BLUR_KEY, blurChoice.value)
  apply()
}

/** Прозрачность: true/false — явно, null — по умолчанию. */
export function setTransparency(value) {
  transparencyChoice.value = value === null ? null : (value ? 'on' : 'off')
  writeChoice(OPACITY_KEY, transparencyChoice.value)
  apply()
}
