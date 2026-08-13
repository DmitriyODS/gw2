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
  { type: 'stock',    label: 'Наличие',         icon: 'inventory_2' },
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
    // min/max — необязательные границы числа ('' = предела нет).
    case 'number': return { pattern: '', qr: false, min: '', max: '' }
    case 'select': return { options: [], multiple: false }
    case 'text': return { multiline: false, qr: false }
    case 'datetime': return { year: true, month_day: true, time: true }
    // «Наличие» настроек не требует: по умолчанию позиция на месте.
    case 'stock': return {}
    default: return {}
  }
}

/* Теги реестра: администратор назначает источником ОДНО списковое поле, и его
   варианты становятся чипами-фильтрами над таблицей. Здесь — единственное место,
   где это правило читается (раздел и публичная страница ссылки берут отсюда). */
export function tagField(registry) {
  const id = registry?.tag_field_id
  if (!id) return null
  const field = (registry.fields || []).find((f) => f.id === id)
  return field && field.type === 'select' ? field : null
}

export function tagOptions(registry) {
  const field = tagField(registry)
  return field ? (field.config?.options || []).filter(Boolean) : []
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

// stockText — надпись поля «Наличие»: пусто и снятая галочка означают «на
// месте», поэтому у новой записи поле молчит и ничего не занимает.
// Зеркало pkg/records.StockText — держать в паре.
export function stockText(value) {
  if (!value?.taken) return 'В наличии'
  if (!value.until) return 'Забрали'
  const [y, m, d] = String(value.until).split('-')
  return d ? `Забрали до ${d}.${m}.${y}` : `Забрали до ${value.until}`
}

// textValue — компактное текстовое представление значения (таблица/поиск).
export function textValue(field, value) {
  if (field.type === 'stock') return stockText(value)
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

// ── Файлы и картинки ──
// Значение поля image/file — объект {path, name, mime, size, thumb?}; наружу
// хранилище раздаётся по /uploads/<ключ>.
export function fileUrl(value) {
  return value?.path ? `/uploads/${value.path}` : ''
}

// thumbUrl — превью для таблицы: уменьшенную копию делает сервер при загрузке.
// Её нет у мелких картинок и у записей, созданных до появления превью, — тогда
// показываем оригинал (браузер сожмёт его сам, а грузит только видимые строки).
export function thumbUrl(value) {
  if (!value?.path) return ''
  return value.thumb ? `/uploads/${value.thumb}` : fileUrl(value)
}

// Участвует ли тип в сквозном поиске / сортировке таблицы.
export function isSearchable(type) {
  return ['text', 'number', 'link', 'datetime', 'select'].includes(type)
}
export function isSortable(type) {
  return ['text', 'number', 'datetime', 'link'].includes(type)
}
// «Наличие» — объект {taken, until}: как строку его сортировать бессмысленно.
// Экспортируется в xlsx всё, кроме картинок и файлов (их нельзя свести к ячейке).
export function isExportable(type) {
  return type !== 'image' && type !== 'file'
}

// ── QR-коды ──
// QR несёт значение поля как есть, поэтому поддерживаются только «плоские»
// типы — текст и число (config.qr включает показ, печать и поиск по коду).
export function isQrCapable(type) {
  return type === 'text' || type === 'number'
}
export function hasQr(field) {
  return isQrCapable(field?.type) && !!field?.config?.qr
}
// Значение поля в виде строки для QR ('' — кодировать нечего).
export function qrValue(value) {
  return value == null ? '' : String(value).trim()
}
