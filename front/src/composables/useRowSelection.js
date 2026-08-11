import { computed, ref, watch } from 'vue'

/**
 * Выбор строк списка под массовые действия (удаление, выгрузка).
 *
 * Множество пересобирается целиком, а не мутируется: реактивность Set в Vue
 * не отслеживает add/delete у вложенного значения.
 *
 * @param {() => Array} items — записи текущей страницы.
 */
export function useRowSelection(items) {
  const selected = ref(new Set())

  const allSelected = computed(() => {
    const list = items() || []
    return list.length > 0 && list.every((r) => selected.value.has(r.id))
  })

  function toggle(id) {
    const next = new Set(selected.value)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    selected.value = next
  }

  function toggleAll() {
    selected.value = allSelected.value ? new Set() : new Set((items() || []).map((r) => r.id))
  }

  function clear() {
    if (selected.value.size) selected.value = new Set()
  }

  // Страница сменилась или список перечитан — выбор не должен пережить записи,
  // которых на экране больше нет.
  watch(items, (list) => {
    if (!selected.value.size) return
    const ids = new Set((list || []).map((r) => r.id))
    const next = new Set([...selected.value].filter((id) => ids.has(id)))
    if (next.size !== selected.value.size) selected.value = next
  })

  return { selected, allSelected, toggle, toggleAll, clear }
}
