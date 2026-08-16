/**
 * Реестр «приложений» рабочего стола.
 *
 * Пути разделов и их компоненты остаются в router/index.js — единственном
 * источнике истины (окно рендерит компонент, полученный из router.resolve).
 * Здесь описаны только оконные свойства раздела: заголовок, иконка, группа в
 * меню «Пуск», стартовый размер окна и правило доступности.
 *
 * available(ctx) — ctx собирает вызывающий: { hasCompany, isSuperAdmin, settings }.
 */

import { resolveSectionKey } from '@/utils/settingsSections.js'
import { SUBSCRIPTIONS_VISIBLE } from '@/utils/release.js'

export const APP_GROUPS = [
  { key: 'work', label: 'Рабочие процессы' },
  { key: 'team', label: 'Коммуникация' },
  { key: 'manage', label: 'Управление и анализ' },
]

/* Подписи разделов настроек для заголовка окна. Дублировать каталог целиком
   ради заголовка не нужно — здесь только имена, права проверяет сам экран. */
const SETTINGS_TITLES = {
  general: 'Общие',
  theme: 'Темы и оформление',
  desktop: 'Рабочий стол',
  chats: 'Чаты и портал',
  account: 'Аккаунт',
  ai: 'ИИ возможности',
  help: 'Справка и поддержка',
  about: 'О приложении',
  backup: 'Резервная копия',
}

const company = (ctx) => ctx.hasCompany
const always = () => true

export const APPS = [
  {
    id: 'tasks',
    title: 'Задачи',
    icon: 'dashboard_customize',
    group: 'work',
    tile: 'wide',
    path: '/tasks',
    match: (p) => p === '/tasks' || /^\/tasks\/\d+$/.test(p),
    size: [1240, 820],
    min: [560, 420],
    available: company,
  },
  {
    id: 'registries',
    title: 'Реестры',
    icon: 'list_alt_add',
    group: 'work',
    tile: 'square',
    path: '/registries',
    match: (p) => p.startsWith('/registries'),
    size: [1320, 840],
    min: [620, 440],
    available: company,
    /* Своя версия раздела: он переписывается отдельно от платформы. Её
       показывает «О разделе» — по клику на значок в заголовке окна. */
    about: { version: '2.0.0', date: '2026-08-13' },
  },
  {
    id: 'calendars',
    title: 'Календари',
    icon: 'calendar_today',
    group: 'work',
    tile: 'wide',
    path: '/calendars',
    match: (p) => p.startsWith('/calendars'),
    size: [1320, 860],
    min: [620, 460],
    available: company,
  },
  {
    id: 'forms',
    title: 'Формы и опросы',
    icon: 'assignment',
    group: 'work',
    tile: 'wide',
    path: '/forms',
    match: (p) => p.startsWith('/forms'),
    size: [1280, 860],
    min: [600, 460],
    /* Компания не нужна: форма принадлежит человеку, а компании и коллеги
       получают её назначением. */
    available: always,
    about: { version: '1.0.0', date: '2026-08-16' },
  },
  {
    id: 'diaries',
    title: 'Ежедневники',
    icon: 'event_list',
    group: 'work',
    tile: 'wide',
    path: '/diaries',
    match: (p) => p.startsWith('/diaries'),
    size: [1200, 820],
    min: [560, 440],
    available: always,
  },
  {
    id: 'notes',
    title: 'Заметки',
    icon: 'filter_none',
    group: 'work',
    tile: 'square',
    path: '/notes',
    match: (p) => p.startsWith('/notes'),
    size: [1180, 800],
    min: [520, 420],
    available: always,
    titleFor: (route) => (/^\/notes\/\d+$/.test(route.path) ? 'Заметка' : null),
  },
  {
    id: 'drive',
    title: 'Диск',
    icon: 'cloud',
    group: 'work',
    tile: 'square',
    path: '/drive',
    // Публичная ссылка /drive/s/<code> — самостоятельная страница, а не окно.
    match: (p) => p === '/drive',
    size: [1180, 800],
    min: [520, 420],
    available: always,
  },
  {
    id: 'boards',
    title: 'Доски',
    icon: 'gesture',
    group: 'work',
    tile: 'wide',
    path: '/boards',
    match: (p) => p.startsWith('/boards'),
    size: [1280, 860],
    min: [560, 460],
    available: always,
    titleFor: (route) => (/^\/boards\/\d+$/.test(route.path) ? 'Доска' : null),
  },
  {
    id: 'reminders',
    title: 'Напоминания',
    icon: 'alarm',
    group: 'work',
    tile: 'square',
    path: '/reminders',
    match: (p) => p.startsWith('/reminders'),
    size: [900, 760],
    min: [460, 440],
    available: always,
  },
  {
    id: 'messenger',
    title: 'Мессенджер',
    icon: 'forum',
    group: 'team',
    tile: 'wide',
    path: '/messenger',
    match: (p) => p.startsWith('/messenger'),
    size: [1120, 780],
    min: [520, 440],
    available: always,
  },
  {
    id: 'portal',
    title: 'Портал',
    icon: 'web_stories',
    group: 'team',
    tile: 'square',
    path: '/portal',
    match: (p) => p.startsWith('/portal'),
    size: [1080, 840],
    min: [520, 440],
    available: company,
  },
  {
    // Сотрудники — самостоятельный раздел, а не вкладка портала: ходят сюда
    // редко и по своему поводу, а вкладки отнимали строку у поиска.
    id: 'employees',
    title: 'Сотрудники',
    icon: 'groups',
    group: 'team',
    tile: 'square',
    path: '/employees',
    match: (p) => p.startsWith('/employees'),
    size: [1080, 820],
    min: [520, 440],
    available: company,
    titleFor: (route) => (/^\/employees\/\d+\/activity$/.test(route.path) ? 'Активность' : null),
  },
  {
    id: 'pets',
    title: 'Питомцы',
    icon: 'pets',
    group: 'team',
    tile: 'square',
    path: '/pets',
    match: (p) => p.startsWith('/pets'),
    size: [1120, 820],
    min: [520, 440],
    available: (ctx) => ctx.hasCompany && ctx.settings.uses_groove !== false,
    titleFor: (route) => {
      if (route.path === '/pets/bank') return 'Кудо-банк'
      if (route.path === '/pets/shop') return 'Магазин'
      return null
    },
  },
  {
    id: 'stats',
    title: 'Статистика',
    icon: 'bar_chart',
    group: 'manage',
    tile: 'wide',
    path: '/stats',
    match: (p) => p === '/stats',
    size: [1280, 840],
    min: [620, 460],
    available: company,
  },
  {
    id: 'users',
    title: 'Пользователи',
    icon: 'group',
    group: 'manage',
    tile: 'square',
    path: '/users',
    match: (p) => p === '/users',
    size: [1160, 780],
    min: [560, 440],
    available: (ctx) => ctx.isSuperAdmin,
  },
  {
    id: 'settings',
    title: 'Настройки',
    icon: 'settings',
    group: 'manage',
    tile: 'square',
    path: '/settings',
    match: (p) => p.startsWith('/settings'),
    size: [980, 760],
    min: [480, 440],
    available: always,
    // Заголовок окна уточняет открытый раздел — «Настройки · Темы и оформление».
    titleFor: (route) => {
      const s = SETTINGS_TITLES[resolveSectionKey(route.query?.section)]
      return s ? `Настройки · ${s}` : null
    },
  },
  {
    /* Ярлык на панель настроек: своего окна у справки нет, поэтому путь она
       НЕ «присваивает» (match всегда false) — иначе окно настроек считалось бы
       то одним разделом, то другим. Плитка нужна, чтобы справку находили и на
       телефоне, где подвала «Пуска» с кнопками нет. */
    id: 'help',
    title: 'Справка и поддержка',
    icon: 'help',
    group: 'manage',
    tile: 'square',
    path: '/settings?section=help',
    match: () => false,
    size: [980, 760],
    min: [480, 440],
    available: always,
  },
  {
    id: 'calculator',
    title: 'Калькулятор',
    icon: 'calculate',
    group: 'work',
    tile: 'square',
    path: '/calculator',
    match: (p) => p === '/calculator',
    size: [380, 580],
    min: [280, 420],
    available: always,
  },
  {
    id: 'store',
    title: 'Магазин',
    icon: 'shopping_bag',
    group: 'team',
    tile: 'square',
    path: '/store',
    match: (p) => p.startsWith('/store'),
    size: [1100, 800],
    min: [520, 460],
    // Витрина ждёт оплату — раздел скрыт целиком (см. utils/release.js).
    // Правило `available` заодно закрывает и прямую ссылку: каркас не откроет
    // окно раздела, которого пользователю не положено.
    available: () => SUBSCRIPTIONS_VISIBLE,
  },
]

const BY_ID = new Map(APPS.map((a) => [a.id, a]))

export function appById(id) {
  return BY_ID.get(id) || null
}

export function appForPath(path) {
  return APPS.find((a) => a.match(path)) || null
}

/**
 * Разделы меню «Пуск» с плитками, отфильтрованными по правам.
 *
 * layout — личная раскладка пользователя (переезжает между устройствами):
 *   groups   — свои разделы [{key, label}] (идут после встроенных);
 *   labels   — переименования разделов ({ [key]: label });
 *   appGroup — куда перенесена плитка ({ [appId]: groupKey });
 *   order    — порядок плиток внутри раздела ({ [key]: [appId] }); плитки не из
 *              списка идут следом в порядке реестра.
 * Встроенные разделы возвращаются всегда (в т.ч. пустые) — иначе некуда было бы
 * вернуть плитку; прятать пустые решает вызывающий.
 */
export function menuGroups(ctx, layout = {}) {
  const { groups = [], labels = {}, appGroup = {}, order = {} } = layout
  const all = [...APP_GROUPS, ...groups.map((g) => ({ key: g.key, label: g.label, custom: true }))]
  const known = new Set(all.map((g) => g.key))
  const available = APPS.filter((a) => a.group && a.available(ctx))

  return all.map((g) => {
    const items = available.filter((a) => {
      const moved = appGroup[a.id]
      return known.has(moved) ? moved === g.key : a.group === g.key
    })
    return {
      key: g.key,
      label: labels[g.key] || g.label,
      custom: !!g.custom,
      items: sortByOrder(items, order[g.key]),
    }
  })
}

export function sortByOrder(items, ids) {
  if (!Array.isArray(ids) || !ids.length) return items
  const rank = new Map(ids.map((id, i) => [id, i]))
  return [...items].sort((a, b) => (rank.get(a.id) ?? ids.length) - (rank.get(b.id) ?? ids.length))
}

/** Заголовок окна: базовый заголовок приложения либо уточнение по маршруту. */
export function windowTitle(app, route) {
  if (!app) return 'Groove Work'
  return (route && app.titleFor?.(route)) || app.title
}
