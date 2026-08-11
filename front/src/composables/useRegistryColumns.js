import { computed, ref, watch } from 'vue'

/**
 * Видимые колонки таблицы записей реестра — личная настройка устройства.
 * Раздел хранит их по id реестра, публичная ссылка — по своему коду (у гостя
 * нет ни аккаунта, ни доступа к самому реестру), поэтому ключ отдаёт вызывающий.
 *
 * @param {() => Array} fields — поля реестра (источник значений по умолчанию:
 *   `show_in_table`; он же отсеивает колонки удалённых полей).
 * @param {() => (string|null)} storageKey — ключ localStorage; null — не хранить.
 */
export function useRegistryColumns(fields, storageKey) {
  const visible = ref([])

  function load() {
    const list = fields() || []
    const key = storageKey()
    if (key) {
      try {
        const raw = localStorage.getItem(key)
        if (raw) {
          visible.value = JSON.parse(raw).filter((id) => list.some((f) => f.id === id))
          return
        }
      } catch { /* повреждённая запись — берём умолчание реестра */ }
    }
    visible.value = list.filter((f) => f.show_in_table).map((f) => f.id)
  }

  function save() {
    const key = storageKey()
    if (!key) return
    try { localStorage.setItem(key, JSON.stringify(visible.value)) } catch { /* приватный режим */ }
  }

  function toggle(id) {
    const i = visible.value.indexOf(id)
    if (i === -1) visible.value.push(id)
    else visible.value.splice(i, 1)
    save()
  }

  // Сменился реестр (или приехали его поля) — набор колонок читаем заново.
  watch([storageKey, fields], load, { immediate: true })

  /** Поля в порядке реестра, отфильтрованные по видимости. */
  const shown = computed(() => (fields() || []).filter((f) => visible.value.includes(f.id)))

  return { visible, shown, toggle }
}
