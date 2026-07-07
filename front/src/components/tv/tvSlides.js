// Каталог слайдов табло и реестр kind → компонент.
// title/settingsNote — для чекбоксов в настройках табло (TvSettingsDialog).
import SlideHero from './slides/SlideHero.vue'
import SlidePodium from './slides/SlidePodium.vue'
import SlideRanking from './slides/SlideRanking.vue'
import SlideDepartments from './slides/SlideDepartments.vue'
import SlideQuad from './slides/SlideQuad.vue'
import SlideBrand from './slides/SlideBrand.vue'
import SlideGroove from './slides/SlideGroove.vue'
import SlidePulse from './slides/SlidePulse.vue'
import SlideWorkTypes from './slides/SlideWorkTypes.vue'
import SlideDebt from './slides/SlideDebt.vue'
import SlideResponsibles from './slides/SlideResponsibles.vue'

export const SLIDE_COMPONENTS = {
  'hero-number': SlideHero,
  podium: SlidePodium,
  ranking: SlideRanking,
  departments: SlideDepartments,
  quad: SlideQuad,
  brand: SlideBrand,
  groove: SlideGroove,
  pulse: SlidePulse,
  'work-types': SlideWorkTypes,
  debt: SlideDebt,
  responsibles: SlideResponsibles,
}

// Отбор слайдов, которые сейчас показываются на табло: без выключенных в
// настройках (disabledIds) и без слайда «долг», когда долга нет. Если всё
// выключили — табло не гаснет, показываем хотя бы брендовый слайд.
export function visibleSlides(disabledIds = [], { debtValue = 0 } = {}) {
  const list = SLIDES.filter((s) => {
    if (disabledIds.includes(s.id)) return false
    if (s.kind === 'debt' && debtValue <= 0) return false
    return true
  })
  return list.length ? list : SLIDES.filter((s) => s.kind === 'brand')
}

export const SLIDES = [
  // 1. Сегодня • закрытия
  {
    id: 'today-closed', title: 'Закрыто сегодня', period: 'day', kind: 'hero-number',
    icon: 'today', periodLabel: 'Сегодня',
    heroIcon: 'task_alt', heroEyebrow: 'Закрыто задач за день',
    heroKey: 'closed', heroFormat: 'int', tone: 'success',
    heroCaption: 'столько работ команда довела до финала сегодня',
    secondaries: [
      { label: 'Поступило', key: 'received', tone: 'primary', prefix: '+' },
      { label: 'В работе',  key: 'remaining', tone: 'tertiary' },
    ],
    asideTone: 'primary', asideIcon: 'schedule', asideTitle: 'Время команды',
    asideKind: 'hours-today',
  },
  // 2. Сегодня • подиум
  {
    id: 'today-podium', title: 'Лидеры дня', period: 'day', kind: 'podium',
    icon: 'today', periodLabel: 'Сегодня',
    heroEyebrow: 'Лидеры дня',
    asideTone: 'tertiary', asideIcon: 'apartment', asideTitle: 'Активный отдел',
    asideKind: 'top-dept',
  },
  // 3. Сегодня • отделы
  {
    id: 'today-departments', title: 'Отделы дня', period: 'day', kind: 'departments',
    icon: 'today', periodLabel: 'Сегодня',
    heroEyebrow: 'Задачи по отделам',
    asideTone: 'success', asideIcon: 'task_alt', asideTitle: 'Сегодня закрыто',
    asideKind: 'closed-today',
  },
  // 4. Сейчас • ответственные
  {
    id: 'responsibles', title: 'Ответственные', period: 'day', kind: 'responsibles',
    icon: 'assignment_ind', periodLabel: 'Прямо сейчас',
    heroEyebrow: 'Задачи на ответственных',
    asideTone: 'tertiary', asideIcon: 'hourglass_top', asideTitle: 'Суммарно',
    asideKind: 'open-responsibles',
  },
  // 5. Неделя • часы команды
  {
    id: 'week-hours', title: 'Часы недели', period: 'week', kind: 'hero-number',
    icon: 'date_range', periodLabel: 'Последние 7 дней',
    heroIcon: 'schedule', heroEyebrow: 'Часы команды за неделю',
    heroKey: 'total_hours', heroFormat: 'hours', tone: 'secondary',
    heroCaption: 'это суммарное время работы всей команды',
    secondaries: [
      { label: 'Закрыто', key: 'closed', tone: 'success', prefix: '−' },
      { label: 'Поступило', key: 'received', tone: 'primary', prefix: '+' },
    ],
    asideTone: 'primary', asideIcon: 'show_chart', asideTitle: 'Динамика',
    asideKind: 'sparkline-closed',
  },
  // 6. Неделя • пульс потока (поступило vs закрыто по дням)
  {
    id: 'week-pulse', title: 'Пульс недели', period: 'week', kind: 'pulse',
    icon: 'date_range', periodLabel: 'Последние 7 дней',
    heroEyebrow: 'Пульс недели',
    asideTone: 'success', asideIcon: 'balance', asideTitle: 'Баланс',
    asideKind: 'flow-balance',
  },
  // 7. Неделя • топ-5 сотрудников
  {
    id: 'week-ranking', title: 'Топ недели', period: 'week', kind: 'ranking',
    icon: 'date_range', periodLabel: 'Последние 7 дней',
    heroEyebrow: 'Топ сотрудников недели',
    asideTone: 'secondary', asideIcon: 'schedule', asideTitle: 'Всего часов',
    asideKind: 'hours-period',
  },
  // 8. Неделя • структура работ
  {
    id: 'week-worktypes', title: 'Структура работ', period: 'week', kind: 'work-types',
    icon: 'category', periodLabel: 'Последние 7 дней',
    heroEyebrow: 'Структура работ недели',
    asideTone: 'secondary', asideIcon: 'category', asideTitle: 'Главный тип',
    asideKind: 'top-worktype',
  },
  // 9. Фокус • долг (показывается только когда debt > 0)
  {
    id: 'debt', title: 'Фокус недели (долг)', period: 'week', kind: 'debt',
    settingsNote: 'показывается, только когда есть задачи дольше срока',
    icon: 'assignment_late', periodLabel: 'Фокус недели',
    heroEyebrow: 'Фокус недели',
    asideTone: 'success', asideIcon: 'task_alt', asideTitle: 'Противовес',
    asideKind: 'closed-period',
  },
  // 10. Месяц • четверть KPI
  {
    id: 'month-quad', title: 'Месяц одной картой', period: 'month', kind: 'quad',
    icon: 'calendar_month', periodLabel: 'Последние 30 дней',
    heroEyebrow: 'Месяц одной картой',
    asideTone: 'tertiary', asideIcon: 'apartment', asideTitle: 'Топ-отдел месяца',
    asideKind: 'top-dept',
  },
  // 11. Месяц • MVP
  {
    id: 'month-podium', title: 'MVP месяца', period: 'month', kind: 'podium',
    icon: 'calendar_month', periodLabel: 'Последние 30 дней',
    heroEyebrow: 'MVP месяца',
    asideTone: 'secondary', asideIcon: 'show_chart', asideTitle: 'Динамика',
    asideKind: 'sparkline-hours',
  },
  // 12. Питомцы • зал славы Грувиков
  {
    id: 'groove-pets', title: 'Зал славы Грувиков', period: 'day', kind: 'groove',
    icon: 'pets', periodLabel: 'Питомцы',
    heroEyebrow: 'Зал славы Грувиков',
    asideTone: 'tertiary', asideIcon: 'military_tech', asideTitle: 'Лидер зала славы',
    asideKind: 'top-pet',
  },
  // 13. Брендовый слайд
  {
    id: 'brand', title: 'Брендовый слайд', period: 'day', kind: 'brand',
    icon: 'auto_awesome', periodLabel: 'Хорошего дня',
    heroEyebrow: 'Groove Work',
    asideTone: 'primary', asideIcon: 'today', asideTitle: 'Сегодня',
    asideKind: 'today-snapshot',
  },
]
