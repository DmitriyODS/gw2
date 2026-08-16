/* Типы вопросов формы. Набор синхронизирован с доменом Go
   (back-go/forms/internal/domain/questions.go) — держать в паре.

   Значение ответа хранится по строковому id вопроса, и его форма зависит от
   типа: текст — строка, выбор — строка, флажки — массив строк, шкала и
   оценка — число, сетка — {строка: столбец(ы)}, дата — «ГГГГ-ММ-ДД», время —
   «ЧЧ:ММ», файлы — массив загруженных файлов, пояснение — ответа не имеет. */

export const QUESTION_TYPES = [
  { type: 'short_text', label: 'Короткий ответ', icon: 'short_text' },
  { type: 'paragraph', label: 'Абзац', icon: 'notes' },
  { type: 'radio', label: 'Один из списка', icon: 'radio_button_checked' },
  { type: 'checkbox', label: 'Несколько из списка', icon: 'check_box' },
  { type: 'dropdown', label: 'Выпадающий список', icon: 'arrow_drop_down_circle' },
  { type: 'scale', label: 'Шкала', icon: 'linear_scale' },
  { type: 'rating', label: 'Оценка', icon: 'star' },
  { type: 'grid_radio', label: 'Сетка выбора', icon: 'grid_on' },
  { type: 'grid_checkbox', label: 'Сетка флажков', icon: 'apps' },
  { type: 'date', label: 'Дата', icon: 'event' },
  { type: 'time', label: 'Время', icon: 'schedule' },
  { type: 'file', label: 'Загрузка файла', icon: 'attach_file' },
  { type: 'booking', label: 'Запись', icon: 'event_available' },
  { type: 'note', label: 'Пояснение', icon: 'info' },
]

const META = Object.fromEntries(QUESTION_TYPES.map((q) => [q.type, q]))

export function questionMeta(type) {
  return META[type] || { type, label: type, icon: 'help' }
}

export const questionLabel = (type) => questionMeta(type).label
export const questionIcon = (type) => questionMeta(type).icon

// Требует ли тип ответа (у пояснительного блока ответа нет).
export const isAnswerable = (type) => type !== 'note'
// Типы с заранее известным набором вариантов.
export const isChoice = (type) => ['radio', 'checkbox', 'dropdown'].includes(type)
/* «Запись» — варианты с ограниченным числом мест: смена, время приёма, место в
   зале. Рисуется отдельно от обычного выбора, потому что показывает остаток и
   гасит занятые варианты. */
export const isBooking = (type) => type === 'booking'
export const isGrid = (type) => ['grid_radio', 'grid_checkbox'].includes(type)
// Умеет ли тип уводить на другой раздел по выбранному варианту.
export const isBranching = (type) => ['radio', 'dropdown'].includes(type)
// Можно ли назначить правильный ответ (режим теста).
export const isGradable = (type) =>
  ['radio', 'dropdown', 'checkbox', 'short_text', 'scale', 'rating', 'grid_radio', 'grid_checkbox']
    .includes(type)

// Потолки — зеркало domain: варианты, файлы, размеры.
export const MAX_OPTIONS = 200
export const MAX_FILES = 10
export const MAX_FILE_BYTES = 1024 * 1024 * 1024

// Конфиг по умолчанию для нового вопроса выбранного типа.
export function defaultConfig(type) {
  switch (type) {
    case 'radio':
    case 'dropdown':
      return { options: ['Вариант 1'], other: false, shuffle: false, targets: {} }
    case 'checkbox':
      return { options: ['Вариант 1'], other: false, shuffle: false, min_choices: 0, max_choices: 0 }
    case 'scale':
      return { min: 1, max: 5, min_label: '', max_label: '' }
    case 'rating':
      return { max: 5, icon: 'star' }
    case 'grid_radio':
    case 'grid_checkbox':
      return { rows: ['Строка 1'], cols: ['Столбец 1'], require_each_row: false }
    case 'date':
      return { with_time: false }
    case 'file':
      return { max_files: 1, max_size_mb: 10 }
    case 'booking':
      // capacity — мест на каждый вариант; ключ — название варианта.
      return { options: ['Вариант 1'], capacity: { 'Вариант 1': 10 } }
    case 'short_text':
    case 'paragraph':
      return { validation: { kind: 'none', pattern: '', hint: '', min: '', max: '' } }
    case 'note':
      return { text: '' }
    default:
      return {}
  }
}

// Пустой ответ выбранного типа — с чего начинается заполнение.
export function emptyAnswer(type) {
  switch (type) {
    case 'checkbox':
      return []
    case 'grid_radio':
    case 'grid_checkbox':
      return {}
    case 'file':
      return []
    case 'scale':
    case 'rating':
      return null
    default:
      return ''
  }
}

// Заполнен ли ответ (проверка обязательности; зеркало domain.Filled).
export function isFilled(value) {
  if (value == null || value === '') return false
  if (Array.isArray(value)) return value.length > 0
  if (typeof value === 'object') return Object.keys(value).length > 0
  return true
}

/* Места «Записи»: сколько всего, сколько занято и сколько осталось. Занятость
   считает сервер и присылает вместе с формой (FillView.booking) — клиенту её
   неоткуда взять, а показывать остаток нужно до отправки. */
export function bookingSlots(question, taken = {}) {
  const capacity = question?.config?.capacity || {}
  return (question?.config?.options || []).map((option) => {
    const total = Number(capacity[option]) || 0
    const used = Number(taken[option]) || 0
    return { option, total, used, left: Math.max(0, total - used) }
  })
}

// Границы шкалы (зеркало Question.Scale).
export function scaleBounds(question) {
  const cfg = question?.config || {}
  const min = Number(cfg.min) === 0 ? 0 : 1
  let max = Number(cfg.max) || 5
  if (max < min + 1) max = min + 1
  if (max > 10) max = 10
  return { min, max }
}

// Сколько делений у оценки (3..10).
export function ratingMax(question) {
  const max = Number(question?.config?.max) || 5
  return Math.min(10, Math.max(3, max))
}

// Потолок одного файла в байтах для файлового вопроса.
export function fileLimit(question) {
  const mb = Number(question?.config?.max_size_mb) || 10
  return Math.min(MAX_FILE_BYTES, Math.max(1, mb) * 1024 * 1024)
}

export function fileCount(question) {
  const n = Number(question?.config?.max_files) || 1
  return Math.min(MAX_FILES, Math.max(1, n))
}

// Значение поля file — массив объектов {path, name, mime, size, thumb?};
// наружу хранилище раздаётся по /uploads/<ключ>.
export function fileUrl(file) {
  return file?.path ? `/uploads/${file.path}` : ''
}

export function thumbUrl(file) {
  if (!file?.path) return ''
  return file.thumb ? `/uploads/${file.thumb}` : fileUrl(file)
}

/* answerText — текстовое представление ответа: таблица ответов, карточка и
   поиск (зеркало domain.AnswerText). */
export function answerText(question, value) {
  if (value == null || value === '') return ''
  switch (question.type) {
    case 'checkbox':
      return Array.isArray(value) ? value.join(', ') : String(value)
    case 'grid_radio':
    case 'grid_checkbox': {
      const rows = question.config?.rows || []
      return rows
        .filter((row) => value[row] != null)
        .map((row) => {
          const cell = value[row]
          return `${row}: ${Array.isArray(cell) ? cell.join(', ') : cell}`
        })
        .join('; ')
    }
    case 'file':
      return Array.isArray(value) ? value.map((f) => f?.name).filter(Boolean).join(', ') : ''
    case 'date':
      return String(value).replace('T', ' ')
    default:
      return String(value)
  }
}

/* validationError — проверка ответа на клиенте (зеркало CoerceAnswer сервера).
   Возвращает текст ошибки или '' — сервер проверяет всё то же самое, здесь это
   нужно, чтобы человек узнал об ошибке до отправки длинной анкеты. */
export function validationError(question, value) {
  if (!isAnswerable(question.type) || !isFilled(value)) return ''
  switch (question.type) {
    case 'short_text':
    case 'paragraph':
      return textError(question, String(value).trim())
    case 'checkbox': {
      const cfg = question.config || {}
      const n = Array.isArray(value) ? value.length : 0
      if (cfg.min_choices > 0 && n < cfg.min_choices) {
        return `Выберите не меньше ${cfg.min_choices} вариантов`
      }
      if (cfg.max_choices > 0 && n > cfg.max_choices) {
        return `Выберите не больше ${cfg.max_choices} вариантов`
      }
      return ''
    }
    case 'grid_radio':
    case 'grid_checkbox': {
      const rows = question.config?.rows || []
      if (question.config?.require_each_row && rows.some((row) => !isFilled(value[row]))) {
        return 'Ответьте в каждой строке'
      }
      return ''
    }
    case 'file': {
      const limit = fileCount(question)
      if (Array.isArray(value) && value.length > limit) {
        return `Можно приложить не больше ${limit} файлов`
      }
      return ''
    }
    default:
      return ''
  }
}

const EMAIL_RE = /^[^@\s]+@[^@\s.]+(\.[^@\s.]+)+$/
const URL_RE = /^(https?:\/\/)?[^\s/$.?#][^\s]*$/
const NUM_RE = /^[+-]?([0-9]+([.][0-9]*)?|[.][0-9]+)$/

function textError(question, text) {
  const v = question.config?.validation || {}
  const min = v.min === '' || v.min == null ? null : Number(v.min)
  const max = v.max === '' || v.max == null ? null : Number(v.max)
  switch (v.kind) {
    case 'number': {
      if (!NUM_RE.test(text)) return 'Принимается только число'
      const n = Number(text)
      if (min != null && n < min) return `Не меньше ${min}`
      if (max != null && n > max) return `Не больше ${max}`
      return ''
    }
    case 'email':
      return EMAIL_RE.test(text) ? '' : 'Непохоже на адрес почты'
    case 'url':
      return URL_RE.test(text) ? '' : 'Непохоже на ссылку'
    case 'regex':
      if (!v.pattern) return ''
      try {
        return new RegExp(v.pattern).test(text) ? '' : (v.hint || 'Ответ не соответствует шаблону')
      } catch {
        return '' // кривой шаблон — вина составителя формы, а не отвечающего
      }
    case 'length': {
      const n = [...text].length
      if (min != null && n < min) return `Не короче ${min} символов`
      if (max != null && n > max) return `Не длиннее ${max} символов`
      return ''
    }
    default:
      return ''
  }
}

/* isCorrect — совпал ли ответ с ключом теста (зеркало domain.Grade). Нужен
   показу разбора после отправки: сервер присылает ключи вместе с оценкой. */
/* Условие показа вопроса: {question_id, values}. Пустой список значений —
   «любой непустой ответ». Зеркало domain.QuestionVisible. */
export function visibleCondition(question) {
  const id = Number(question?.config?.visible_question_id) || 0
  if (!id || id === question?.id) return null
  return { questionId: id, values: question.config.visible_values || [] }
}

export function isCorrect(question, value, key) {
  if (!key) return false
  switch (question.type) {
    case 'radio':
    case 'dropdown':
      return String(value ?? '').toLowerCase() === String(key.value ?? '').toLowerCase()
    case 'short_text': {
      const got = String(value ?? '').trim().toLowerCase()
      return (key.values || []).some((v) => String(v).trim().toLowerCase() === got)
    }
    case 'checkbox': {
      const want = key.values || []
      const got = Array.isArray(value) ? value : []
      return want.length > 0 && want.length === got.length && want.every((v) => got.includes(v))
    }
    case 'scale':
    case 'rating':
      return Number(value) === Number(key.number)
    case 'grid_radio':
    case 'grid_checkbox': {
      const want = key.grid || {}
      const got = value || {}
      const rows = Object.keys(want)
      if (!rows.length) return false
      return rows.every((row) => {
        if (question.type === 'grid_radio') return got[row] === want[row]
        const wantRow = want[row] || []
        const gotRow = got[row] || []
        return wantRow.length === gotRow.length && wantRow.every((c) => gotRow.includes(c))
      })
    }
    default:
      return false
  }
}

// Состояние приёма ответов — подпись и тон чипа.
export const FORM_STATUSES = {
  draft: { label: 'Черновик', tone: 'neutral', icon: 'edit_note' },
  open: { label: 'Принимает ответы', tone: 'success', icon: 'play_circle' },
  closed: { label: 'Приём закрыт', tone: 'warning', icon: 'stop_circle' },
}

export function statusMeta(status) {
  return FORM_STATUSES[status] || FORM_STATUSES.draft
}

// Уровни доступа к форме (зеркало domain/access.go).
export const ACCESS_LEVELS = [
  { value: 'respond', label: 'Заполнить', hint: 'Форма появится у человека в «Мне назначены»' },
  { value: 'view', label: 'Видеть ответы', hint: 'Плюс сводка и выгрузка' },
  { value: 'edit', label: 'Редактировать', hint: 'Плюс правка самой формы и доступов' },
]

export function accessLabel(access) {
  return ACCESS_LEVELS.find((a) => a.value === access)?.label || 'Заполнить'
}
