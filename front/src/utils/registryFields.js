// Типы полей реестра. Набор синхронизирован с Go-доменом
// (back-go/registry/internal/domain/models.go) — держать в паре.

export const FIELD_TYPES = [
  { type: 'text',     label: 'Текст',           icon: 'notes' },
  { type: 'number',   label: 'Число',           icon: 'tag' },
  { type: 'select',   label: 'Список выбора',   icon: 'checklist' },
  { type: 'checkbox', label: 'Галочка',         icon: 'check_box' },
  { type: 'date',     label: 'Дата и время',    icon: 'event', value: 'datetime' },
  { type: 'link',     label: 'Ссылка',          icon: 'link' },
  { type: 'image',    label: 'Картинка',        icon: 'image' },
  { type: 'file',     label: 'Файл',            icon: 'attach_file' },
]

// Внутренний идентификатор типа даты в домене — 'datetime' (отображаем как «Дата и время»).
export const FIELD_DATETIME = 'datetime'

const META = Object.fromEntries(
  FIELD_TYPES.map((f) => [f.value || f.type, f]),
)

export function fieldMeta(type) {
  return META[type] || { type, label: type, icon: 'help' }
}

export function fieldLabel(type) {
  return fieldMeta(type).label
}

export function fieldIcon(type) {
  return fieldMeta(type).icon
}

// Конфиг по умолчанию для нового поля выбранного типа.
export function defaultConfig(type) {
  switch (type) {
    case 'number': return { pattern: '' }
    case 'select': return { options: [], multiple: false }
    case 'text': return { multiline: false }
    case 'datetime': return { year: true, month_day: true, time: true }
    default: return {}
  }
}

// formatDateTime — строка по включённым частям (config.year/month_day/time).
// Значение хранится как ISO-строка.
export function formatDateTime(value, config = {}) {
  if (!value) return ''
  const d = new Date(value)
  if (isNaN(d.getTime())) return String(value)
  const pad = (n) => String(n).padStart(2, '0')
  const parts = []
  if (config.month_day !== false && config.year !== false) {
    parts.push(`${pad(d.getDate())}.${pad(d.getMonth() + 1)}.${d.getFullYear()}`)
  } else if (config.month_day) {
    parts.push(`${pad(d.getDate())}.${pad(d.getMonth() + 1)}`)
  } else if (config.year) {
    parts.push(String(d.getFullYear()))
  }
  if (config.time) parts.push(`${pad(d.getHours())}:${pad(d.getMinutes())}`)
  return parts.join(' ')
}

// textValue — компактное текстовое представление значения (таблица/поиск).
export function textValue(field, value) {
  if (value == null || value === '') return ''
  switch (field.type) {
    case 'checkbox': return value ? 'Да' : 'Нет'
    case 'select': return Array.isArray(value) ? value.join(', ') : String(value)
    case 'datetime': return formatDateTime(value, field.config || {})
    case 'image': return value?.name || 'Картинка'
    case 'file': return value?.name || 'Файл'
    default: return String(value)
  }
}

// Участвует ли тип в сквозном поиске / сортировке таблицы.
export function isSearchable(type) {
  return ['text', 'number', 'link', 'datetime', 'select'].includes(type)
}
export function isSortable(type) {
  return ['text', 'number', 'datetime', 'link'].includes(type)
}
// Экспортируется в xlsx всё, кроме картинок и файлов (их нельзя свести к ячейке).
export function isExportable(type) {
  return type !== 'image' && type !== 'file'
}
