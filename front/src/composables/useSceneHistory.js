/**
 * История правок холста доски: снимки сцены ДО правки.
 *
 * Хранится строками, а не объектами: сцена неизменяема, но объекты внутри
 * переиспользуются следующими снимками, и сериализация — самый дешёвый
 * способ гарантировать, что отмена вернёт ровно то, что было.
 * Общая для редактора и публичной ссылки с правом правки.
 */
import { computed, ref } from 'vue'

const HISTORY_LIMIT = 60

export function useSceneHistory() {
  const undoStack = ref([])
  const redoStack = ref([])

  /** Запомнить состояние перед правкой. */
  function push(snapshot) {
    undoStack.value.push(JSON.stringify(snapshot))
    if (undoStack.value.length > HISTORY_LIMIT) undoStack.value.shift()
    redoStack.value = []
  }

  /** Шаг назад: вернуть прежнюю сцену (или null), current уходит в повтор. */
  function undo(current) {
    const prev = undoStack.value.pop()
    if (!prev) return null
    redoStack.value.push(JSON.stringify(current))
    return JSON.parse(prev)
  }

  function redo(current) {
    const next = redoStack.value.pop()
    if (!next) return null
    undoStack.value.push(JSON.stringify(current))
    return JSON.parse(next)
  }

  function reset() {
    undoStack.value = []
    redoStack.value = []
  }

  return {
    canUndo: computed(() => undoStack.value.length > 0),
    canRedo: computed(() => redoStack.value.length > 0),
    push, undo, redo, reset,
  }
}
