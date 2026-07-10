// Градиентные темы комнаты грувика. Ключи синхронизированы с
// domain.HouseThemes (petsvc); визуал целиком здесь — только токены
// (--tag-* пастели адаптированы к светлой/тёмной теме).

export const HOUSE_THEMES = [
  {
    key: 'cozy',
    title: 'Уют',
    background: `radial-gradient(80% 90% at 50% 110%,
        color-mix(in oklch, var(--color-tertiary-container) 55%, transparent) 0%, transparent 70%),
      linear-gradient(180deg,
        color-mix(in oklch, var(--color-primary-container) 30%, var(--color-surface)),
        color-mix(in oklch, var(--color-secondary-container) 45%, var(--color-surface)))`,
  },
  {
    key: 'sunset',
    title: 'Закат',
    background: `radial-gradient(90% 80% at 50% 115%,
        color-mix(in oklch, var(--tag-orange-accent) 26%, transparent) 0%, transparent 70%),
      linear-gradient(180deg, var(--tag-amber-surface), var(--tag-pink-surface))`,
  },
  {
    key: 'night',
    title: 'Ночь',
    background: `radial-gradient(70% 70% at 78% 12%,
        color-mix(in oklch, var(--tag-violet-accent) 30%, transparent) 0%, transparent 65%),
      linear-gradient(180deg, var(--tag-violet-surface), var(--tag-blue-surface))`,
  },
  {
    key: 'forest',
    title: 'Лес',
    background: `radial-gradient(85% 85% at 50% 112%,
        color-mix(in oklch, var(--tag-green-accent) 24%, transparent) 0%, transparent 70%),
      linear-gradient(180deg, var(--tag-green-surface), var(--tag-teal-surface))`,
  },
  {
    key: 'ocean',
    title: 'Океан',
    background: `radial-gradient(85% 85% at 50% -10%,
        color-mix(in oklch, var(--tag-blue-accent) 22%, transparent) 0%, transparent 65%),
      linear-gradient(180deg, var(--tag-blue-surface), var(--tag-teal-surface))`,
  },
  {
    key: 'candy',
    title: 'Карамель',
    background: `radial-gradient(80% 90% at 22% 108%,
        color-mix(in oklch, var(--tag-pink-accent) 26%, transparent) 0%, transparent 70%),
      linear-gradient(180deg, var(--tag-pink-surface), var(--tag-violet-surface))`,
  },
]

export function houseThemeBackground(key) {
  const theme = HOUSE_THEMES.find((t) => t.key === key) || HOUSE_THEMES[0]
  return theme.background
}
