import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  storageGet, storageGetJSON, storageSet, storageSetJSON,
} from '@/utils/storage.js'

/* ── Built-in presets ────────────────────────────────────────────
   primary / secondary / tertiary — три «ручки» интерфейса.
   neutral (необязательно) — задаёт цветную гамму фонов и поверхностей.
   Если neutral не задан, фон следует за основным цветом с едва заметным
   тоном (как было до появления нейтральной гаммы). */
/* Каждая тема теперь включает neutral — фоновый оттенок в той же гамме,
   что и акценты. Это даёт «единое лицо» интерфейса: и кнопки, и фоны
   живут в одной палитре, а не на нейтрально-сером заднике. */
const PRESETS = {
  classic: { primary: '#9b4dff', secondary: '#00bfa5', tertiary: '#3d6ce7', neutral: '#ece8f2' },
  blue:    { primary: '#1e88e5', secondary: '#00acc1', tertiary: '#7e57c2', neutral: '#e6ecf4' },
  pink:    { primary: '#ec4899', secondary: '#e91e63', tertiary: '#ce93d8', neutral: '#f5e8ee' },
  red:     { primary: '#e53935', secondary: '#ff7043', tertiary: '#f06292', neutral: '#f4e6e3' },
  green:   { primary: '#2e7d32', secondary: '#00897b', tertiary: '#26a69a', neutral: '#e6eee7' },
  orange:  { primary: '#ef6c00', secondary: '#ff6d00', tertiary: '#fdd835', neutral: '#f5ebde' },
  yellow:  { primary: '#c98300', secondary: '#fb8c00', tertiary: '#43a047', neutral: '#f4eedb' },
  violet:  { primary: '#7c4dff', secondary: '#00b0ff', tertiary: '#e040fb', neutral: '#ebe6f5' },
  lilac:   { primary: '#9b59b6', secondary: '#c77daa', tertiary: '#7da87e', neutral: '#f0e8ef' },
  sunset:  { primary: '#e8806e', secondary: '#e8a07a', tertiary: '#db8398', neutral: '#f1e9dc' },
  ocean:   { primary: '#0277bd', secondary: '#26c6da', tertiary: '#5e92f3', neutral: '#e3edf2' },
  mint:    { primary: '#16a085', secondary: '#1abc9c', tertiary: '#7fb3a4', neutral: '#e4efea' },
  coffee:  { primary: '#795548', secondary: '#a1887f', tertiary: '#d4a373', neutral: '#efe8e0' },
  midnight:{ primary: '#5e7fff', secondary: '#7c3aed', tertiary: '#2dd4bf', neutral: '#e6e9f2' },
  forest:  { primary: '#2f7d4f', secondary: '#558b2f', tertiary: '#a5a96d', neutral: '#e6ece2' },
}

const PRESET_LABELS = {
  classic: 'Классическая',
  blue:    'Синяя',
  pink:    'Розовая',
  red:     'Красная',
  green:   'Зелёная',
  orange:  'Оранжевая',
  yellow:  'Жёлтая',
  violet:  'Фиолетовая',
  lilac:   'Весенняя сирень',
  sunset:  'Тёплый закат',
  ocean:   'Океан',
  mint:    'Мята',
  coffee:  'Кофе с молоком',
  midnight:'Полночь',
  forest:  'Лесная',
}

/* ── oklch conversion ────────────────────────────────────────────
   Converts a CSS hex colour to { L, C, H } in the OKLCH colour space.
   Algorithm from Björn Ottosson — https://bottosson.github.io/posts/oklab/ */
function hexToOklch(hex) {
  const r = parseInt(hex.slice(1, 3), 16) / 255
  const g = parseInt(hex.slice(3, 5), 16) / 255
  const b = parseInt(hex.slice(5, 7), 16) / 255

  const toLinear = c => c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
  const rl = toLinear(r), gl = toLinear(g), bl = toLinear(b)

  const l_ = Math.cbrt(0.4122214708 * rl + 0.5363325363 * gl + 0.0514459929 * bl)
  const m_ = Math.cbrt(0.2119034982 * rl + 0.6806995451 * gl + 0.1073969566 * bl)
  const s_ = Math.cbrt(0.0883024619 * rl + 0.2817188376 * gl + 0.6299787005 * bl)

  const L  =  0.2104542553 * l_ + 0.7936177850 * m_ - 0.0040720468 * s_
  const a  =  1.9779984951 * l_ - 2.4285922050 * m_ + 0.4505937099 * s_
  const bv =  0.0259040371 * l_ + 0.7827717662 * m_ - 0.8086757660 * s_

  const C = Math.sqrt(a * a + bv * bv)
  const H = ((Math.atan2(bv, a) * 180 / Math.PI) + 360) % 360

  return { L, C, H }
}

/* Writes --ref-*-h, --ref-*-c, --ref-*-l CSS vars for an accent palette key.
   Насыщенность нормализуется в коридор [0.06 … 0.33]: это снимает «запрет»
   на очень тёмные и очень светлые цвета — из них извлекается почти серый
   оттенок, и без нижнего порога палитра выглядела бы выцветшей. Но если
   хрома исходного hex ниже NEUTRAL_C_THRESHOLD (фактически нейтральный
   цвет — белый, чёрный, серый), нижний порог НЕ применяем: иначе у белого
   atan2(0, 0) даёт случайный H ≈ 90° и кнопка получается «песочной».
   Светлота сохраняется (с безопасным клампом 0.30…0.92), чтобы выбор очень
   светлого hex действительно красил кнопку в светлый цвет; на контрастный
   текст (--color-on-{name}) ставим белый или тёмный по порогу 0.65.        */
const NEUTRAL_C_THRESHOLD = 0.015

function applyPaletteKey(root, name, hex) {
  const { L, C, H } = hexToOklch(hex)
  const c = C < NEUTRAL_C_THRESHOLD ? C : Math.min(Math.max(C, 0.06), 0.33)
  const l = Math.min(Math.max(L, 0.30), 0.92)
  root.style.setProperty(`--ref-${name}-h`, H.toFixed(1))
  root.style.setProperty(`--ref-${name}-c`, c.toFixed(4))
  root.style.setProperty(`--ref-${name}-l`, l.toFixed(4))
  // Контрастный текст на цветной плашке: светлая плашка → тёмный текст.
  // Для нейтральных цветов on-color делаем без хромы, чтобы текст не уезжал в оттенок.
  const onColor = l >= 0.65
    ? `oklch(0.18 ${c < NEUTRAL_C_THRESHOLD ? 0 : (c * 0.6).toFixed(4)} ${H.toFixed(1)})`
    : 'oklch(0.995 0 0)'
  root.style.setProperty(`--color-on-${name}-user`, onColor)
}

/* Нейтральный (фоновый) цвет особый: его оттенок задаёт гамму фона, а из
   насыщенности выводится множитель тинта (--ref-neutral-c). Эталон 0.012 ≈
   множитель 1 (текущий едва заметный тон); чем сочнее выбранный цвет —
   тем заметнее цветная гамма фонов. Кламп до 4, чтобы фон не «кричал». */
function applyNeutral(root, hex) {
  const { C, H } = hexToOklch(hex)
  const mult = Math.min(Math.max(C / 0.012, 0), 4)
  root.style.setProperty('--ref-neutral-h', H.toFixed(1))
  root.style.setProperty('--ref-neutral-c', mult.toFixed(3))
}

/* HSL → hex. Используется генератором случайных тем: цвета удобнее задавать
   в HSL (управляем оттенком/насыщенностью/светлотой), а движок сам извлечёт
   из hex параметры OKLCH. */
function hslToHex(h, s, l) {
  h = ((h % 360) + 360) % 360
  const c = (1 - Math.abs(2 * l - 1)) * s
  const x = c * (1 - Math.abs(((h / 60) % 2) - 1))
  const m = l - c / 2
  let r = 0, g = 0, b = 0
  if (h < 60) { r = c; g = x } else if (h < 120) { r = x; g = c }
  else if (h < 180) { g = c; b = x } else if (h < 240) { g = x; b = c }
  else if (h < 300) { r = x; b = c } else { r = c; b = x }
  const to = v => Math.round((v + m) * 255).toString(16).padStart(2, '0')
  return `#${to(r)}${to(g)}${to(b)}`
}

const rand = (min, max) => Math.random() * (max - min) + min
const pickOne = arr => arr[Math.floor(Math.random() * arr.length)]

/* Случайная, но гармоничная тема: базовый оттенок + смещения по одной из
   схем цветовой гармонии (аналоговая/триада/комплемент/сплит-комплемент),
   умеренная насыщенность. Работает и в светлой, и в тёмной теме, т.к. движок
   выводит тона автоматически. Нейтраль — мягкий тинт того же базового тона. */
function randomTheme() {
  const base = Math.random() * 360
  const schemes = [
    [0, 30, -30],   // аналоговая
    [0, 120, -120], // триада
    [0, 25, 180],   // комплемент с тёплым акцентом
    [0, 150, 210],  // сплит-комплемент
    [0, -40, 40],   // расширенная аналоговая
  ]
  const off = pickOne(schemes)
  const sat = rand(0.55, 0.72)
  const lig = rand(0.54, 0.63)
  return {
    primary:   hslToHex(base + off[0], sat, lig),
    secondary: hslToHex(base + off[1], sat * rand(0.85, 1), lig + rand(0, 0.06)),
    tertiary:  hslToHex(base + off[2], sat * rand(0.85, 1), lig + rand(-0.04, 0.06)),
    neutral:   hslToHex(base + rand(-25, 25), rand(0.10, 0.24), 0.92),
  }
}

export const useThemeStore = defineStore('theme', () => {
  /* Resolve stored preset name — map legacy 'dark' to 'classic' */
  const storedPreset = storageGet('gw_theme', 'classic')
  const storedCustomThemes = storageGetJSON('gw_custom_themes', [])
  const isKnownPreset = PRESETS[storedPreset] || storedCustomThemes.some(t => t.name === storedPreset)
  const resolvedPreset = isKnownPreset ? storedPreset : 'classic'

  const currentPreset = ref(resolvedPreset)
  const dark = ref(storageGet('gw_dark', 'false') === 'true')

  /* If the old 'dark' preset was active, enable dark mode automatically.
     localStorage.setItem может бросить в приватном режиме старого iOS Safari —
     это код инициализации store, исключение здесь рушит монтирование приложения
     (белый экран). Оборачиваем в try/catch. */
  if (storedPreset === 'dark' && !dark.value) {
    dark.value = true
    storageSet('gw_dark', 'true')
  }

  const customThemes = ref(storedCustomThemes)

  function getVars(name) {
    if (PRESETS[name]) return PRESETS[name]
    const custom = customThemes.value.find(t => t.name === name)
    return custom?.vars || PRESETS.classic
  }

  function applyVars(vars) {
    const root = document.documentElement
    applyPaletteKey(root, 'primary',   vars.primary)
    applyPaletteKey(root, 'secondary', vars.secondary)
    applyPaletteKey(root, 'tertiary',  vars.tertiary)
    // Нейтральная гамма фона. Без явного цвета — сбрасываем переопределение,
    // тогда фон следует за основным цветом с дефолтным тоном (как раньше).
    if (vars.neutral) {
      applyNeutral(root, vars.neutral)
    } else {
      root.style.removeProperty('--ref-neutral-h')
      root.style.removeProperty('--ref-neutral-c')
    }
  }

  function applyTheme(name) {
    currentPreset.value = name
    storageSet('gw_theme', name)
    applyVars(getVars(name))
  }

  function toggleDark() {
    dark.value = !dark.value
    storageSet('gw_dark', dark.value)
    document.documentElement.setAttribute('data-dark', dark.value)
  }

  function saveCustomTheme(name, vars) {
    const idx = customThemes.value.findIndex(t => t.name === name)
    if (idx >= 0) customThemes.value[idx] = { name, vars }
    else customThemes.value.push({ name, vars })
    storageSetJSON('gw_custom_themes', customThemes.value)
  }

  function deleteCustomTheme(name) {
    customThemes.value = customThemes.value.filter(t => t.name !== name)
    storageSetJSON('gw_custom_themes', customThemes.value)
    if (currentPreset.value === name) applyTheme('classic')
  }

  function exportTheme(name) {
    const vars = getVars(name)
    const blob = new Blob([JSON.stringify({ name, vars }, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url; a.download = `${name}-theme.json`; a.click()
    URL.revokeObjectURL(url)
  }

  function importTheme(json) {
    const parsed = typeof json === 'string' ? JSON.parse(json) : json
    const { name, vars } = parsed
    /* Accept both old format (has bg/surface/…) and new format (primary/secondary/tertiary[/neutral]) */
    const normalised = {
      primary:   vars.primary   || '#e040fb',
      secondary: vars.secondary || vars.accent || '#00bfa5',
      tertiary:  vars.tertiary  || '#3d6ce7',
    }
    if (vars.neutral) normalised.neutral = vars.neutral
    saveCustomTheme(name, normalised)
    applyTheme(name)
  }

  function init() {
    applyVars(getVars(currentPreset.value))
    document.documentElement.setAttribute('data-dark', dark.value)
  }

  return {
    currentPreset, dark, customThemes,
    presetNames: Object.keys(PRESETS),
    presetLabels: PRESET_LABELS,
    applyTheme, applyVars, toggleDark, saveCustomTheme, deleteCustomTheme,
    exportTheme, importTheme, init, getVars, randomTheme,
  }
})
