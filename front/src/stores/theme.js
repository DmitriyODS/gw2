import { defineStore } from 'pinia'
import { ref } from 'vue'

/* ── Built-in presets (primary / secondary / tertiary hex) ──────── */
const PRESETS = {
  classic: { primary: '#e040fb', secondary: '#00bfa5', tertiary: '#3d6ce7' },
  blue:    { primary: '#1e88e5', secondary: '#00acc1', tertiary: '#7e57c2' },
  pink:    { primary: '#f06292', secondary: '#e91e63', tertiary: '#ce93d8' },
  red:     { primary: '#e53935', secondary: '#ff7043', tertiary: '#f06292' },
  green:   { primary: '#43a047', secondary: '#00897b', tertiary: '#26a69a' },
  orange:  { primary: '#fb8c00', secondary: '#ff6d00', tertiary: '#fdd835' },
  yellow:  { primary: '#f9a825', secondary: '#fb8c00', tertiary: '#43a047' },
  violet:  { primary: '#7c4dff', secondary: '#00b0ff', tertiary: '#e040fb' },
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

  const a  =  1.9779984951 * l_ - 2.4285922050 * m_ + 0.4505937099 * s_
  const bv =  0.0259040371 * l_ + 0.7827717662 * m_ - 0.8086757660 * s_

  const C = Math.sqrt(a * a + bv * bv)
  const H = ((Math.atan2(bv, a) * 180 / Math.PI) + 360) % 360

  return { C, H }
}

/* Writes --ref-*-h and --ref-*-c CSS vars for a single palette key. */
function applyPaletteKey(root, name, hex) {
  const { C, H } = hexToOklch(hex)
  root.style.setProperty(`--ref-${name}-h`, H.toFixed(1))
  root.style.setProperty(`--ref-${name}-c`, C.toFixed(4))
}

export const useThemeStore = defineStore('theme', () => {
  /* Resolve stored preset name — map legacy 'dark' to 'classic' */
  const storedPreset = localStorage.getItem('gw_theme') || 'classic'
  const resolvedPreset = PRESETS[storedPreset] ? storedPreset : 'classic'

  const currentPreset = ref(resolvedPreset)
  const dark = ref(localStorage.getItem('gw_dark') === 'true')

  /* If the old 'dark' preset was active, enable dark mode automatically */
  if (storedPreset === 'dark' && !dark.value) {
    dark.value = true
    localStorage.setItem('gw_dark', 'true')
  }

  const customThemes = ref(JSON.parse(localStorage.getItem('gw_custom_themes') || '[]'))

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
  }

  function applyTheme(name) {
    currentPreset.value = name
    localStorage.setItem('gw_theme', name)
    applyVars(getVars(name))
  }

  function toggleDark() {
    dark.value = !dark.value
    localStorage.setItem('gw_dark', dark.value)
    document.documentElement.setAttribute('data-dark', dark.value)
  }

  function saveCustomTheme(name, vars) {
    const idx = customThemes.value.findIndex(t => t.name === name)
    if (idx >= 0) customThemes.value[idx] = { name, vars }
    else customThemes.value.push({ name, vars })
    localStorage.setItem('gw_custom_themes', JSON.stringify(customThemes.value))
  }

  function deleteCustomTheme(name) {
    customThemes.value = customThemes.value.filter(t => t.name !== name)
    localStorage.setItem('gw_custom_themes', JSON.stringify(customThemes.value))
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
    /* Accept both old format (has bg/surface/…) and new format (primary/secondary/tertiary) */
    const normalised = {
      primary:   vars.primary   || '#e040fb',
      secondary: vars.secondary || vars.accent || '#00bfa5',
      tertiary:  vars.tertiary  || '#3d6ce7',
    }
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
    exportTheme, importTheme, init, getVars,
  }
})
