// Фиксированный набор из 8 цветов-тегов для задач.
// Значение `id` хранится в БД (поле task.color), визуал задаётся токенами
// --tag-<id>-surface / -border / -accent в tokens.css (адаптированы к темам).
export const TASK_COLORS = [
  { id: 'red', label: 'Коралловый' },
  { id: 'orange', label: 'Оранжевый' },
  { id: 'amber', label: 'Янтарный' },
  { id: 'green', label: 'Зелёный' },
  { id: 'teal', label: 'Бирюзовый' },
  { id: 'blue', label: 'Синий' },
  { id: 'violet', label: 'Сиреневый' },
  { id: 'pink', label: 'Розовый' },
]

export const TASK_COLOR_IDS = TASK_COLORS.map(c => c.id)

// Инлайн-стиль для окрашенной карточки: подставляет токены выбранного цвета.
export function cardColorStyle(color) {
  if (!color || !TASK_COLOR_IDS.includes(color)) return {}
  return {
    '--card-tag-surface': `var(--tag-${color}-surface)`,
    '--card-tag-border': `var(--tag-${color}-border)`,
  }
}
