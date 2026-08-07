/* Виды календарного периода — общие для календарей и ежедневников: они листают
   одно и то же и переключаются одинаково. Широкая панель показывает их
   вкладками (PeriodNav), тесная — пунктом меню «ещё»: строка вкладок на телефоне
   отнимала у содержимого больше, чем давала. */
export const PERIOD_VIEWS = [
  { value: 'month', label: 'Месяц', icon: 'calendar_view_month' },
  { value: 'week', label: 'Неделя', icon: 'calendar_view_week' },
  { value: 'day', label: 'День', icon: 'calendar_view_day' },
]

export function periodView(value) {
  return PERIOD_VIEWS.find((v) => v.value === value) || PERIOD_VIEWS[1]
}

/** Команда-подменю «Вид» для панели команд раздела (`commands` у AppPage). */
export function periodViewCommand(current) {
  const active = periodView(current)
  return {
    key: 'view',
    label: `Вид: ${active.label.toLowerCase()}`,
    icon: active.icon,
    children: PERIOD_VIEWS.map((v) => ({
      key: `view:${v.value}`,
      label: v.label,
      icon: v.value === current ? 'check' : v.icon,
    })),
  }
}

/** `view:week` → `week`; для остальных ключей — пусто. */
export function parseViewCommand(key) {
  return typeof key === 'string' && key.startsWith('view:') ? key.slice(5) : ''
}
