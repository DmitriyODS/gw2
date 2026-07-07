// Контент aside-карточки справа: зависит от asideKind текущего слайда.
// Чистая функция — TvView зовёт её из computed и передаёт снятые значения.
import { num } from './tvFormat.js'

export function buildAsideContent(slide, ctx) {
  const kind = slide?.asideKind
  if (!kind) return null
  const { common, extended, grooveData, commonByPeriod, responsibles, totalHours } = ctx

  if (kind === 'hours-today') {
    return {
      headline: 'Всего отработано',
      value: totalHours, format: 'hours',
      sub: 'все сотрудники, все юниты',
    }
  }
  if (kind === 'hours-period') {
    return {
      headline: 'Команда за период',
      value: totalHours, format: 'hours',
      sub: 'суммарно по всем сотрудникам',
    }
  }
  if (kind === 'closed-today') {
    return {
      headline: 'Закрыто',
      value: num(common?.tasks?.closed), format: 'int',
      prefix: '−',
      sub: 'задач за день',
    }
  }
  if (kind === 'closed-period') {
    return {
      headline: 'Уже закрыто',
      value: num(common?.tasks?.closed), format: 'int',
      prefix: '−',
      sub: 'задач за период — так держать',
    }
  }
  if (kind === 'top-dept') {
    const top = (extended?.by_departments || [])
      .slice().sort((a, b) => num(b.tasks_count) - num(a.tasks_count))[0]
    if (!top) return { headline: 'нет данных' }
    return {
      headline: top.name,
      value: num(top.tasks_count), format: 'int',
      sub: 'задач у лидера',
    }
  }
  if (kind === 'top-worktype') {
    const top = (extended?.by_unit_types || [])
      .slice().sort((a, b) => num(b.total_hours) - num(a.total_hours))[0]
    if (!top) return { headline: 'нет данных' }
    return {
      headline: top.name,
      value: num(top.total_hours), format: 'hours',
      sub: 'на главном типе работ',
    }
  }
  if (kind === 'flow-balance') {
    const closed = num(common?.tasks?.closed)
    const received = num(common?.tasks?.received)
    const diff = closed - received
    return {
      headline: 'Баланс потока',
      value: diff, format: 'int',
      prefix: diff > 0 ? '+' : '',
      sub: 'закрыто минус поступило за период',
    }
  }
  if (kind === 'open-responsibles') {
    const list = responsibles || []
    if (!list.length) return { headline: 'нет данных' }
    return {
      headline: 'В работе',
      value: list.reduce((acc, r) => acc + num(r.open_count), 0), format: 'int',
      sub: 'задач у ответственных суммарно',
    }
  }
  if (kind === 'sparkline-closed') {
    const arr = (extended?.calendar || []).map(d => num(d.closed))
    return {
      headline: 'Закрытий по дням',
      value: arr.reduce((a, b) => a + b, 0), format: 'int',
      sub: 'за период',
      sparkline: arr,
    }
  }
  if (kind === 'sparkline-hours') {
    const arr = (extended?.calendar || []).map(d => num(d.total_hours))
    return {
      headline: 'Часы по дням',
      value: arr.reduce((a, b) => a + b, 0), format: 'hours',
      sub: 'за период',
      sparkline: arr,
    }
  }
  if (kind === 'top-pet') {
    const leader = (grooveData?.pets || [])[0]
    if (!leader) return { headline: 'нет данных' }
    return {
      headline: leader.name,
      value: num(leader.xp), format: 'int',
      sub: `XP у лидера зала славы · ${leader.user?.fio || ''}`,
    }
  }
  if (kind === 'today-snapshot') {
    const today = commonByPeriod?.['day']
    return {
      headline: 'Сегодня',
      value: num(today?.tasks?.closed), format: 'int',
      prefix: '−',
      sub: 'задач закрыто',
    }
  }
  return null
}
