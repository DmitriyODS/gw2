// Градиентные темы комнаты грувика. Ключи синхронизированы с
// domain.HouseThemes (petsvc); визуал целиком здесь — только токены
// (--tag-* пастели адаптированы к светлой/тёмной теме).
// Каждая тема — два цветных радиальных пятна поверх диагонали: сочно
// и в превью-свотче, и в полной сцене комнаты.

export const HOUSE_THEMES = [
  {
    key: 'cozy',
    title: 'Уют',
    background: `radial-gradient(80% 90% at 50% 110%,
        color-mix(in oklch, var(--color-tertiary-container) 70%, transparent) 0%, transparent 70%),
      radial-gradient(60% 70% at 12% -8%,
        color-mix(in oklch, var(--color-primary) 22%, transparent) 0%, transparent 65%),
      linear-gradient(160deg,
        color-mix(in oklch, var(--color-primary-container) 45%, var(--color-surface)),
        color-mix(in oklch, var(--color-secondary-container) 60%, var(--color-surface)))`,
  },
  {
    key: 'sunset',
    title: 'Закат',
    background: `radial-gradient(90% 80% at 50% 115%,
        color-mix(in oklch, var(--tag-orange-accent) 45%, transparent) 0%, transparent 70%),
      radial-gradient(60% 70% at 85% -10%,
        color-mix(in oklch, var(--tag-pink-accent) 26%, transparent) 0%, transparent 65%),
      linear-gradient(160deg, var(--tag-amber-surface), var(--tag-pink-surface))`,
  },
  {
    key: 'night',
    title: 'Ночь',
    background: `radial-gradient(70% 70% at 78% 12%,
        color-mix(in oklch, var(--tag-violet-accent) 48%, transparent) 0%, transparent 65%),
      radial-gradient(70% 80% at 10% 110%,
        color-mix(in oklch, var(--tag-blue-accent) 28%, transparent) 0%, transparent 65%),
      linear-gradient(160deg, var(--tag-violet-surface), var(--tag-blue-surface))`,
  },
  {
    key: 'forest',
    title: 'Лес',
    background: `radial-gradient(85% 85% at 50% 112%,
        color-mix(in oklch, var(--tag-green-accent) 42%, transparent) 0%, transparent 70%),
      radial-gradient(55% 65% at 88% -8%,
        color-mix(in oklch, var(--tag-teal-accent) 26%, transparent) 0%, transparent 65%),
      linear-gradient(160deg, var(--tag-green-surface), var(--tag-teal-surface))`,
  },
  {
    key: 'ocean',
    title: 'Океан',
    background: `radial-gradient(85% 85% at 50% -10%,
        color-mix(in oklch, var(--tag-blue-accent) 42%, transparent) 0%, transparent 65%),
      radial-gradient(70% 80% at 15% 112%,
        color-mix(in oklch, var(--tag-teal-accent) 30%, transparent) 0%, transparent 70%),
      linear-gradient(160deg, var(--tag-blue-surface), var(--tag-teal-surface))`,
  },
  {
    key: 'candy',
    title: 'Карамель',
    background: `radial-gradient(80% 90% at 22% 108%,
        color-mix(in oklch, var(--tag-pink-accent) 45%, transparent) 0%, transparent 70%),
      radial-gradient(60% 70% at 90% -8%,
        color-mix(in oklch, var(--tag-violet-accent) 28%, transparent) 0%, transparent 65%),
      linear-gradient(160deg, var(--tag-pink-surface), var(--tag-violet-surface))`,
  },
  {
    key: 'aurora',
    title: 'Сияние',
    background: `radial-gradient(90% 70% at 30% -10%,
        color-mix(in oklch, var(--tag-green-accent) 40%, transparent) 0%, transparent 60%),
      radial-gradient(80% 80% at 80% 115%,
        color-mix(in oklch, var(--tag-violet-accent) 42%, transparent) 0%, transparent 65%),
      radial-gradient(50% 60% at 70% 10%,
        color-mix(in oklch, var(--tag-teal-accent) 30%, transparent) 0%, transparent 60%),
      linear-gradient(160deg, var(--tag-teal-surface), var(--tag-violet-surface))`,
  },
  {
    key: 'lavender',
    title: 'Лаванда',
    background: `radial-gradient(85% 85% at 50% 115%,
        color-mix(in oklch, var(--tag-violet-accent) 40%, transparent) 0%, transparent 70%),
      radial-gradient(55% 65% at 12% -8%,
        color-mix(in oklch, var(--tag-blue-accent) 24%, transparent) 0%, transparent 65%),
      linear-gradient(160deg, var(--tag-violet-surface), var(--tag-pink-surface))`,
  },
  {
    key: 'peach',
    title: 'Персик',
    background: `radial-gradient(85% 90% at 70% 112%,
        color-mix(in oklch, var(--tag-orange-accent) 42%, transparent) 0%, transparent 70%),
      radial-gradient(55% 65% at 10% -8%,
        color-mix(in oklch, var(--tag-amber-accent) 30%, transparent) 0%, transparent 65%),
      linear-gradient(160deg, var(--tag-orange-surface), var(--tag-amber-surface))`,
  },
  {
    key: 'space',
    title: 'Космос',
    background: `radial-gradient(70% 70% at 20% 10%,
        color-mix(in oklch, var(--tag-blue-accent) 45%, transparent) 0%, transparent 60%),
      radial-gradient(70% 80% at 85% 110%,
        color-mix(in oklch, var(--tag-pink-accent) 34%, transparent) 0%, transparent 65%),
      radial-gradient(40% 50% at 60% 40%,
        color-mix(in oklch, var(--tag-violet-accent) 26%, transparent) 0%, transparent 60%),
      linear-gradient(160deg, var(--tag-blue-surface), var(--tag-violet-surface))`,
  },
]

export function houseThemeBackground(key) {
  const theme = HOUSE_THEMES.find((t) => t.key === key) || HOUSE_THEMES[0]
  return theme.background
}
