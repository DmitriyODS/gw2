import { computed, ref, watch } from 'vue'

/**
 * Колонки таблицы записей реестра — личная настройка устройства: и СОСТАВ, и
 * ПОРЯДОК. Раздел хранит их по id реестра, публичная ссылка — по своему коду (у
 * гостя нет ни аккаунта, ни доступа к самому реестру), поэтому ключ отдаёт
 * вызывающий.
 *
 * `visible` — упорядоченный список id: он и решает, в каком порядке идут
 * колонки. Умолчание берётся из реестра (`show_in_table` в порядке полей),
 * поэтому нетронутая таблица выглядит ровно так, как её задумал составитель.
 *
 * @param {() => Array} fields — поля реестра (источник значений по умолчанию:
 *   `show_in_table`; он же отсеивает колонки удалённых полей).
 * @param {() => (string|null)} storageKey — ключ localStorage; null — не хранить.
 */
export function useRegistryColumns(fields, storageKey) {
  const visible = ref([])
  // Ширины в пикселях по id колонки; пусто — таблица раскладывается сама.
  const widths = ref({})

  function load() {
    const list = fields() || []
    const known = (ids) => ids.filter((id) => list.some((f) => f.id === id))
    const key = storageKey()
    if (key) {
      try {
        const raw = localStorage.getItem(key)
        if (raw) {
          const saved = JSON.parse(raw)
          // Прежний формат — голый массив id: настройки, сохранённые до ширин,
          // должны читаться как были, а не сбрасываться в умолчание.
          if (Array.isArray(saved)) {
            visible.value = known(saved)
            widths.value = {}
            return
          }
          visible.value = known(saved.order || [])
          widths.value = saved.widths || {}
          return
        }
      } catch { /* повреждённая запись — берём умолчание реестра */ }
    }
    visible.value = list.filter((f) => f.show_in_table).map((f) => f.id)
    widths.value = {}
  }

  function save() {
    const key = storageKey()
    if (!key) return
    const payload = { order: visible.value, widths: widths.value }
    try { localStorage.setItem(key, JSON.stringify(payload)) } catch { /* приватный режим */ }
  }

  /**
   * Запомнить ширины колонок. Приходит СРАЗУ ВЕСЬ набор, а не одна колонка:
   * первое же растягивание переводит таблицу в фиксированную раскладку, и
   * остальным колонкам нужны их текущие ширины — иначе они схлопнулись бы.
   */
  function setWidths(map) {
    widths.value = { ...widths.value, ...map }
    save()
  }

  // Вернуть автоматическую раскладку: пункт «сбросить ширины» в настройке колонок.
  function resetWidths() {
    widths.value = {}
    save()
  }

  function toggle(id) {
    const i = visible.value.indexOf(id)
    if (i === -1) visible.value.push(id)
    else visible.value.splice(i, 1)
    save()
  }

  /**
   * Переставить колонку: перетащенную ставим НА МЕСТО той, над которой бросили.
   * Массив пересобираем целиком — так не бывает состояния «колонку вынули, но
   * не вставили», из-за которого при быстром броске таблица теряла столбец.
   */
  function move(dragId, dropId) {
    if (dragId === dropId) return
    const next = [...visible.value]
    const from = next.indexOf(dragId)
    const to = next.indexOf(dropId)
    if (from === -1 || to === -1) return
    next.splice(from, 1)
    next.splice(to, 0, dragId)
    visible.value = next
    save()
  }

  // Сменился реестр (или приехали его поля) — набор колонок читаем заново.
  watch([storageKey, fields], load, { immediate: true })

  /** Видимые поля В ПОРЯДКЕ КОЛОНОК (а не в порядке полей реестра). */
  const shown = computed(() => {
    const byId = new Map((fields() || []).map((f) => [f.id, f]))
    return visible.value.map((id) => byId.get(id)).filter(Boolean)
  })

  return { visible, shown, widths, toggle, move, setWidths, resetWidths }
}
