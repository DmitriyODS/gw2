import { ref } from 'vue'

// Общее состояние перетаскивания заметок/папок между деревом, сеткой проводника
// и хлебными крошками. Одно на приложение — DnD в один момент один.
const dragItem = ref(null) // { kind: 'note' | 'folder', id, name }

export function useDragItem() {
  const start = (kind, id, name) => { dragItem.value = { kind, id, name } }
  const end = () => { dragItem.value = null }
  return { dragItem, start, end }
}
