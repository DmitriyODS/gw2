/* Разделы хранилища: как называется и каким цветом показывается сервис-владелец
   файлов в «Настройки → Хранилище».

   Ключи приходят с сервера как есть (billing_storage_files.service) и совпадают
   с кодами в FILE_OWNER_ADDRS — незнакомый ключ показываем как есть, а не
   прячем: место он занимает в любом случае. */

const SECTIONS = {
  drive: { title: 'Диск', tone: 'primary' },
  messenger: { title: 'Переписка', tone: 'primary' },
  notes: { title: 'Заметки', tone: 'secondary' },
  boards: { title: 'Доски', tone: 'tertiary' },
  registry: { title: 'Реестры', tone: 'success' },
  calendar: { title: 'Календари', tone: 'warning' },
  portal: { title: 'Портал', tone: 'error' },
  avatars: { title: 'Фото профиля', tone: 'neutral' },
}

// Цвет берётся токеном темы: конкретных значений здесь нет, поэтому разбивка
// следует оформлению приложения и остаётся читаемой в тёмном виде.
const TONE_TOKENS = {
  primary: 'var(--color-primary)',
  secondary: 'var(--color-secondary)',
  tertiary: 'var(--color-tertiary)',
  success: 'var(--color-success)',
  warning: 'var(--color-warning, var(--color-tertiary))',
  error: 'var(--color-error)',
  neutral: 'var(--color-outline)',
}

export function sectionTitle(service) {
  return SECTIONS[service]?.title || service
}

export function sectionColor(service) {
  return TONE_TOKENS[SECTIONS[service]?.tone || 'neutral']
}
