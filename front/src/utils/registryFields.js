// Типы полей реестра. Набор синхронизирован с общим ядром Go
// (back-go/pkg/records/records.go) — держать в паре.

export const FIELD_TYPES = [
  { type: 'image',    label: 'Обложка',          icon: 'image' },
  { type: 'file',     label: 'Файл',             icon: 'attach_file' },
  { type: 'link',     label: 'Ссылка',           icon: 'link' },
  { type: 'checkbox', label: 'Флажок',           icon: 'check_box' },
  { type: 'text',     label: 'Короткий текст',   icon: 'short_text' },
  { type: 'textarea', label: 'Длинный текст',    icon: 'notes' },
  { type: 'phone',    label: 'Телефон',          icon: 'call' },
  { type: 'email',    label: 'Почта',            icon: 'alternate_email' },
  { type: 'select',   label: 'Список',           icon: 'checklist' },
  { type: 'number',   label: 'Число',            icon: 'tag' },
  { type: 'regex',    label: 'Текст по шаблону', icon: 'rule' },
  { type: 'datetime', label: 'Дата и время',     icon: 'event' },
]

/* Унаследованные типы: в каталоге при создании поля их больше НЕ предлагаем, но
   рисовать умеем — данные могли остаться у календарей и в архивных копиях.
   «Наличие» в реестрах заменил режим «Учётный реестр» (миграция 00081). */
const LEGACY_TYPES = [
  { type: 'stock', label: 'Наличие', icon: 'inventory_2' },
]

// Внутренний идентификатор типа даты в домене (календари ссылаются на него).
export const FIELD_DATETIME = 'datetime'

const META = Object.fromEntries(
  [...FIELD_TYPES, ...LEGACY_TYPES].map((f) => [f.type, f]),
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

// GRID_COLS — карточка делится на четверти: поле занимает от одной до всех
// четырёх (зеркало domain.GridCols).
export const GRID_COLS = 4

// Конфиг по умолчанию для нового поля выбранного типа.
export function defaultConfig(type) {
  switch (type) {
    // min/max — необязательные границы числа ('' = предела нет).
    case 'number': return { pattern: '', qr: false, min: '', max: '' }
    case 'regex': return { pattern: '', hint: '' }
    case 'select': return { options: [], multiple: false }
    case 'text': return { qr: false }
    case 'checkbox': return { on_label: 'Да', off_label: 'Нет' }
    case 'phone': return { country: 'RU' }
    // Части даты включаются по одной; секунды по умолчанию не нужны.
    case 'datetime': return {
      year: true, month: true, day: true, hours: true, minutes: true, seconds: false,
    }
    default: return {}
  }
}

/* Подразделы реестра: администратор назначает источником ОДНО списковое поле,
   и его варианты становятся вкладками над таблицей. Здесь — единственное место,
   где это правило читается (раздел и публичная страница ссылки берут отсюда). */
export function sectionField(registry) {
  const id = registry?.section_field_id
  if (!id) return null
  const field = (registry.fields || []).find((f) => f.id === id)
  return field && field.type === 'select' ? field : null
}

export function sectionOptions(registry) {
  const field = sectionField(registry)
  return field ? (field.config?.options || []).filter(Boolean) : []
}

// ── Дата и время ──
// dateParts — какие части включены (зеркало records.DateConfig, включая разбор
// прежней тройки year/month_day/time из архивных копий).
export function dateParts(config = {}) {
  const flag = (key, def) => (typeof config[key] === 'boolean' ? config[key] : def)
  if ('month_day' in config) {
    const md = flag('month_day', true)
    const tm = flag('time', true)
    return { year: flag('year', true), month: md, day: md, hours: tm, minutes: tm, seconds: false }
  }
  const p = {
    year: flag('year', true), month: flag('month', true), day: flag('day', true),
    hours: flag('hours', true), minutes: flag('minutes', true), seconds: flag('seconds', false),
  }
  // Поле без единой части нечего показать — трактуем как полную дату.
  if (!Object.values(p).some(Boolean)) {
    return { year: true, month: true, day: true, hours: true, minutes: true, seconds: false }
  }
  return p
}

// formatDateTime — строка по включённым частям. Значение хранится ISO-строкой.
export function formatDateTime(value, config = {}) {
  if (!value) return ''
  const d = new Date(value)
  if (isNaN(d.getTime())) return String(value)
  const pad = (n) => String(n).padStart(2, '0')
  const p = dateParts(config)

  const date = []
  if (p.day) date.push(pad(d.getDate()))
  if (p.month) date.push(pad(d.getMonth() + 1))
  if (p.year) date.push(String(d.getFullYear()))

  const clock = []
  if (p.hours) clock.push(pad(d.getHours()))
  if (p.minutes) clock.push(pad(d.getMinutes()))
  if (p.seconds) clock.push(pad(d.getSeconds()))

  return [date.join('.'), clock.join(':')].filter(Boolean).join(' ')
}

// ── Телефон ──
// normalizePhone — «+цифры» (зеркало records.NormalizePhone): один и тот же
// номер со скобками и без обязан совпадать сам с собой.
export function normalizePhone(value) {
  return String(value ?? '').replace(/(?!^\+)\D/g, '')
}

// validPhone — похоже ли на номер (зеркало records.ValidPhone): плюс и 5..15
// цифр после нормализации. Пустая строка номером не считается.
export function validPhone(value) {
  return /^\+?\d{5,15}$/.test(normalizePhone(value))
}

// formatPhone — читаемый вид российского номера, прочие показываем как есть:
// маску чужой страны мы всё равно угадаем неверно.
export function formatPhone(value) {
  const raw = normalizePhone(value)
  const ru = raw.match(/^\+?([78])(\d{3})(\d{3})(\d{2})(\d{2})$/)
  if (!ru) return raw
  return `+7 (${ru[2]}) ${ru[3]}-${ru[4]}-${ru[5]}`
}

// checkboxText — надпись флажка: составитель реестра задаёт свои слова
// (зеркало records.CheckboxText).
export function checkboxText(field, on) {
  const key = on ? 'on_label' : 'off_label'
  const custom = field?.config?.[key]
  if (typeof custom === 'string' && custom.trim()) return custom.trim()
  return on ? 'Да' : 'Нет'
}

// stockText — надпись УНАСЛЕДОВАННОГО поля «Наличие»: пусто и снятая галочка
// означают «на месте». Зеркало pkg/records.StockText.
export function stockText(value) {
  if (!value?.taken) return 'В наличии'
  if (!value.until) return 'Забрали'
  const [y, m, d] = String(value.until).split('-')
  return d ? `Забрали до ${d}.${m}.${y}` : `Забрали до ${value.until}`
}

// textValue — компактное текстовое представление значения (таблица/поиск).
export function textValue(field, value) {
  if (field.type === 'stock') return stockText(value)
  if (field.type === 'checkbox') return checkboxText(field, !!value)
  if (value == null || value === '') return ''
  switch (field.type) {
    case 'select': return Array.isArray(value) ? value.join(', ') : String(value)
    case 'datetime': return formatDateTime(value, field.config || {})
    case 'phone': return formatPhone(value)
    case 'image': return value?.name || 'Обложка'
    case 'file': return value?.name || 'Файл'
    default: return String(value)
  }
}

// ── Файлы и обложки ──
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

// MAX_IMAGE_BYTES — потолок обложки (зеркало domain.MaxImageSize). Картинку
// крупнее клиент ужимает перед отправкой, а не отвергает: человек снял фото
// телефоном, и это его нормальный сценарий.
export const MAX_IMAGE_BYTES = 2 * 1024 * 1024
// MAX_FILE_BYTES — потолок поля «Файл» (зеркало domain.MaxFileSize).
export const MAX_FILE_BYTES = 1024 * 1024 * 1024

// Участвует ли тип в сквозном поиске / сортировке таблицы.
export function isSearchable(type) {
  return ['text', 'textarea', 'number', 'regex', 'phone', 'email', 'link', 'datetime', 'select']
    .includes(type)
}
export function isSortable(type) {
  return ['text', 'textarea', 'number', 'regex', 'phone', 'email', 'link', 'datetime'].includes(type)
}
// Экспортируется в xlsx всё, кроме обложек и файлов (их нельзя свести к ячейке).
export function isExportable(type) {
  return type !== 'image' && type !== 'file'
}

/* ── Фильтры колонок ──
   Какие сравнения предлагать по типу поля. Значения op — зеркало разбора на
   сервере (repository/postgres/records.go:columnWhere). */
export function filterOps(type) {
  const base = [
    { value: 'filled', label: 'Заполнено' },
    { value: 'empty', label: 'Не заполнено' },
  ]
  switch (type) {
    case 'select':
      return [{ value: 'any', label: 'Один из' }, ...base]
    case 'number':
      return [
        { value: 'equals', label: 'Равно' },
        { value: 'gt', label: 'От' },
        { value: 'lt', label: 'До' },
        { value: 'between', label: 'В диапазоне' },
        ...base,
      ]
    case 'checkbox':
      return [{ value: 'equals', label: 'Равно' }]
    case 'datetime':
      return [{ value: 'contains', label: 'Содержит' }, ...base]
    default:
      return [
        { value: 'contains', label: 'Содержит' },
        { value: 'equals', label: 'Совпадает' },
        ...base,
      ]
  }
}

// Нужны ли операции значения (у «заполнено»/«не заполнено» их нет).
export function opNeedsValue(op) {
  return op !== 'empty' && op !== 'filled'
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
