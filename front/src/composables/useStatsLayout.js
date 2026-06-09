import { reactive, ref, watch } from 'vue'

/**
 * Персистентная раскладка виджетов статистики: размер (small|medium|large),
 * закрепление и порядок. Состояние — синглтон на уровне модуля (общая шина
 * для всех StatsWidget и для StatsView), сохраняется в localStorage.
 *
 * id виджетов уникальны в пределах всего раздела (общий + расширенный режим),
 * поэтому держим единый плоский список — относительный порядок внутри каждого
 * режима сохраняется сам собой.
 */

const STORAGE_KEY = 'gw2_stats_layout_v1'

export const SIZES = ['small', 'medium', 'large']

// Дефолтный порядок и размеры. Новые виджеты, которых ещё нет в сохранёнке,
// добавляются в конец при merge; исчезнувшие — отсеиваются.
const DEFAULTS = [
  { id: 'tasks-period', size: 'medium' },
  { id: 'by-employees', size: 'medium' },
  { id: 'by-hours', size: 'small' },
  { id: 'responsibles', size: 'medium' },
  { id: 'user-tasks', size: 'large' },
  { id: 'unit-types', size: 'medium' },
  { id: 'departments', size: 'small' },
  { id: 'unit-types-per-user', size: 'large' },
  { id: 'calendar', size: 'large' },
]

function normalize(item) {
  return {
    id: item.id,
    size: SIZES.includes(item.size) ? item.size : 'medium',
    pinned: !!item.pinned,
  }
}

function loadInitial() {
  let saved = []
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) saved = JSON.parse(raw)
  } catch {
    saved = []
  }
  if (!Array.isArray(saved)) saved = []

  const savedById = new Map(saved.map((it) => [it.id, it]))
  const defaultIds = new Set(DEFAULTS.map((d) => d.id))

  // Сначала — сохранённый порядок (только существующие виджеты),
  // затем — новые дефолтные, которых в сохранёнке ещё нет.
  const ordered = []
  for (const it of saved) {
    if (defaultIds.has(it.id)) ordered.push(normalize(it))
  }
  for (const d of DEFAULTS) {
    if (!savedById.has(d.id)) ordered.push(normalize(d))
  }
  return ordered
}

const items = reactive(loadInitial())
const draggingId = ref(null)

watch(
  items,
  () => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(items))
    } catch {
      /* приватный режим / переполнение — игнорируем */
    }
  },
  { deep: true }
)

function find(id) {
  return items.find((it) => it.id === id)
}

function indexOf(id) {
  return items.findIndex((it) => it.id === id)
}

function sizeOf(id) {
  return find(id)?.size || 'medium'
}

function pinnedOf(id) {
  return !!find(id)?.pinned
}

/**
 * CSS `order` для виджета: закреплённые «всплывают» наверх (большое
 * отрицательное смещение), сохраняя относительный порядок внутри группы.
 */
function orderOf(id) {
  const idx = indexOf(id)
  if (idx < 0) return 0
  return (find(id)?.pinned ? -1000 : 0) + idx
}

function cycleSize(id) {
  const it = find(id)
  if (!it) return
  const next = (SIZES.indexOf(it.size) + 1) % SIZES.length
  it.size = SIZES[next]
}

function setSize(id, size) {
  const it = find(id)
  if (it && SIZES.includes(size)) it.size = size
}

function togglePin(id) {
  const it = find(id)
  if (it) it.pinned = !it.pinned
}

function startDrag(id) {
  draggingId.value = id
}

function endDrag() {
  draggingId.value = null
}

/** Переставить перетаскиваемый виджет на позицию целевого (вставка перед ним). */
function dropOn(targetId) {
  const dragId = draggingId.value
  draggingId.value = null
  if (!dragId || dragId === targetId) return
  const from = indexOf(dragId)
  if (from < 0 || indexOf(targetId) < 0) return
  const [moved] = items.splice(from, 1)
  // indexOf(targetId) после удаления указывает на целевой — вставляем перед ним.
  items.splice(indexOf(targetId), 0, moved)
}

function reset() {
  items.splice(0, items.length, ...DEFAULTS.map(normalize))
}

export function useStatsLayout() {
  return {
    items,
    draggingId,
    sizeOf,
    pinnedOf,
    orderOf,
    cycleSize,
    setSize,
    togglePin,
    startDrag,
    endDrag,
    dropOn,
    reset,
  }
}
