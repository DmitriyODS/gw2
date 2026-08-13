import { computed, ref, watch } from 'vue'

/**
 * Выбор строк списка под массовые действия (удаление, выгрузка).
 *
 * Выбор ПЕРЕЖИВАЕТ смену страницы: набор описывается либо перечнем id, либо
 * режимом «выбрано всё по текущему фильтру» плюс снятые галочки. Второй режим
 * не тянет на клиент ни одного лишнего id — серверу уезжает тот же фильтр,
 * которым нарисован экран, и список исключений.
 *
 * Множества пересобираются целиком, а не мутируются: реактивность Set в Vue
 * не отслеживает add/delete у вложенного значения.
 *
 * @param {() => Array} items — записи текущей страницы.
 * @param {object} [opts]
 * @param {() => number} [opts.total] — всего записей по фильтру (для режима «все»).
 * @param {() => any} [opts.scope] — ключ выборки (реестр + поиск + тег): сменился —
 *        выбор начинается заново, иначе «всё» означало бы уже другое множество.
 */
export function useRowSelection(items, { total, scope } = {}) {
  // 'include' — picked содержит ВЫБРАННЫЕ; 'all' — picked содержит СНЯТЫЕ.
  const mode = ref('include')
  const picked = ref(new Set())

  const count = computed(() => (mode.value === 'all'
    ? Math.max(0, (total?.() ?? 0) - picked.value.size)
    : picked.value.size))

  function isSelected(id) {
    return mode.value === 'all' ? !picked.value.has(id) : picked.value.has(id)
  }

  const allSelected = computed(() => {
    const list = items() || []
    return list.length > 0 && list.every((r) => isSelected(r.id))
  })

  function toggle(id) {
    const next = new Set(picked.value)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    picked.value = next
  }

  // Галочка в шапке — про ТЕКУЩУЮ страницу: «выбрать всё в базе» отдельной
  // кнопкой в плашке, иначе одним кликом сносится больше, чем человек видел.
  function toggleAll() {
    const list = items() || []
    const on = !allSelected.value
    const next = new Set(picked.value)
    for (const r of list) {
      // В режиме «все» отметить строку значит убрать её из исключений.
      if (on === (mode.value === 'all')) next.delete(r.id)
      else next.add(r.id)
    }
    picked.value = next
  }

  function selectAllMatching() {
    mode.value = 'all'
    picked.value = new Set()
  }

  function clear() {
    mode.value = 'include'
    if (picked.value.size) picked.value = new Set()
  }

  if (scope) watch(scope, clear)

  /* Что уходит серверу: перечень id либо «всё по фильтру минус снятые».
     Фильтр (поиск, тег) добавляет вызывающий — он его и держит. */
  const payload = computed(() => (mode.value === 'all'
    ? { all: true, exclude: [...picked.value] }
    : { ids: [...picked.value] }))

  return {
    mode, picked, count, isSelected, allSelected,
    toggle, toggleAll, selectAllMatching, clear, payload,
  }
}
