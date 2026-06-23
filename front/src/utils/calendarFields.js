// Поля карточки записи календаря используют тот же набор типов и хелперы, что и
// реестры (см. utils/registryFields.js) — переиспользуем их, не дублируя.
// Здесь — только специфика календаря: условная видимость полей и заголовок
// записи для плитки.
export {
  FIELD_TYPES,
  FIELD_DATETIME,
  fieldMeta,
  fieldLabel,
  fieldIcon,
  defaultConfig,
  formatDateTime,
  textValue,
  isSearchable,
  isSortable,
  isExportable,
} from './registryFields.js'

import { textValue } from './registryFields.js'

// isFieldVisible — показывать ли поле в карточке при текущих значениях.
// Правило условной видимости: поле visible_field_id должно иметь значение,
// равное visible_value. Для checkbox-источника visible_value == 'true'
// («когда галочка отмечена»), для select — выбранный вариант. Без условия —
// поле видно всегда.
export function isFieldVisible(field, data) {
  const src = field?.visible_field_id
  if (src == null) return true
  const v = data?.[String(src)]
  const target = field.visible_value ?? ''
  if (Array.isArray(v)) return v.some((x) => String(x) === String(target))
  if (typeof v === 'boolean') return String(v) === String(target)
  return String(v ?? '') === String(target)
}

// canBeCondition — может ли поле быть источником условия (checkbox/select).
export function canBeCondition(type) {
  return type === 'checkbox' || type === 'select'
}

// entryTitle — заголовок записи для плитки/списка: первое поле, помеченное
// «показывать в таблице», иначе первое поле вообще; пусто → запасной текст.
export function entryTitle(calendar, entry, fallback = 'Запись') {
  const fields = calendar?.fields || []
  const pick = fields.find((f) => f.show_in_table) || fields[0]
  if (pick) {
    const v = textValue(pick, entry?.data?.[String(pick.id)])
    if (v) return v
  }
  return fallback
}

// hhmm — время записи без секунд (для плиток и режима «День»).
export function hhmm(value) {
  if (!value) return ''
  const d = new Date(value)
  if (isNaN(d.getTime())) return ''
  const pad = (n) => String(n).padStart(2, '0')
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`
}
