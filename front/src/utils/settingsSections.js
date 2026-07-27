/**
 * Каталог разделов настроек — единый источник для экрана «Настройки» и строки
 * поиска рабочего стола (Spotlight ищет настройки наравне с задачами и
 * заметками, поэтому список не может жить внутри вью).
 *
 * ctx собирает вызывающий: { isMobile, hasCompany, isAdmin, isSuperAdmin }.
 */
export function settingsGroups(ctx = {}) {
  const { isMobile = false, hasCompany = false, isAdmin = false, isSuperAdmin = false } = ctx
  return [
    {
      key: 'personal',
      label: 'Настройки',
      sections: [
        {
          key: 'general',
          title: 'Общие',
          // desc в списке разделов не показывается (там только значок и
          // название) — это подсказка для поиска Spotlight и подзаголовок.
          desc: (hasCompany && !isAdmin)
            ? 'Ассистент, интеграция с YouGile, тур по интерфейсу'
            : 'Ассистент и тур по интерфейсу',
          icon: 'home',
          tone: 'primary',
        },
        { key: 'theme', title: 'Темы и оформление', desc: 'Светлая и тёмная тема, палитры, свои цвета', icon: 'palette', tone: 'primary' },
        // Обои и плитки настраиваются только там, где сам рабочий стол есть.
        ...(isMobile ? [] : [
          { key: 'desktop', title: 'Рабочий стол', desc: 'Обои под окнами, живые плитки, приложение для компьютера', icon: 'desktop_windows', tone: 'tertiary' },
        ]),
        { key: 'chats', title: 'Чаты и портал', desc: 'Фон переписки и ленты корпоративного портала', icon: 'brush', tone: 'secondary' },
        { key: 'help', title: 'Справка и поддержка', desc: 'Как пользоваться разделами, чат с разработчиками', icon: 'help', tone: 'secondary' },
        { key: 'about', title: 'О приложении', desc: 'Версия, что нового, приложения для устройств', icon: 'info', tone: 'tertiary' },
      ],
    },
    // Настройки компании (ИИ, выходные, питомцы, ссылка-приглашение, интеграция
    // YouGile) живут в разделе «Компании» → карточка компании: один пользователь
    // может администрировать несколько компаний, и настройки привязаны к
    // конкретной компании, а не к активной сессии.
    ...(isSuperAdmin ? [{
      key: 'system',
      label: 'Система',
      sections: [
        { key: 'backup', title: 'Резервная копия', desc: 'Экспорт и восстановление базы данных', icon: 'backup', tone: 'error' },
      ],
    }] : []),
  ]
}

/**
 * Устаревшие ключи разделов: на них ведут ссылки, оставшиеся в интерфейсе и в
 * пользовательских закладках. Личный YouGile переехал внутрь «Общих».
 */
const SECTION_ALIASES = { yougile: 'general' }

export function resolveSectionKey(key) {
  return SECTION_ALIASES[key] || key
}

/** Плоский список разделов — для поиска. */
export function settingsSections(ctx = {}) {
  return settingsGroups(ctx).flatMap((g) => g.sections.map((s) => ({ ...s, group: g.label })))
}
