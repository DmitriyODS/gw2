// Разделы резервной копии — ключи синхронизированы с back-go/auth
// (domain.BackupSections + SectionOther). Подписи — только на фронте.
// «Прочее» (other) собирает таблицы, не вошедшие в явные разделы (например,
// добавленные позже), чтобы они не терялись при бэкапе.
export const BACKUP_SECTIONS = [
  { key: 'auth', label: 'Пользователи и доступ', desc: 'Аккаунты, роли, членства, устройства, аватарки' },
  { key: 'companies', label: 'Компании', desc: 'Компании и приглашения' },
  { key: 'tasks', label: 'Задачи и юниты', desc: 'Отделы, этапы, задачи, юниты, комментарии, цвета' },
  { key: 'registry', label: 'Реестры', desc: 'Справочники, поля, записи, ссылки' },
  { key: 'calendar', label: 'Календари', desc: 'Календари, поля, записи, ссылки' },
  { key: 'diary', label: 'Ежедневники', desc: 'Ежедневники, записи и доступы' },
  { key: 'messenger', label: 'Мессенджер', desc: 'Чаты, сообщения, вложения' },
  { key: 'calls', label: 'Звонки', desc: 'Звонки и участники' },
  { key: 'groove', label: 'Мой Groove', desc: 'Питомцы, лента, рейды, локации' },
  { key: 'integration', label: 'Интеграции', desc: 'Подключённые YouGile-аккаунты' },
  { key: 'other', label: 'Прочее', desc: 'Таблицы, не вошедшие в другие разделы' },
]

export const ALL_SECTION_KEYS = BACKUP_SECTIONS.map((s) => s.key)
