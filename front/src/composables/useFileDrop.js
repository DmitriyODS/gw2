import { ref } from 'vue'

export function useFileDrop({ canDrop, onFiles }) {
  const dragOver = ref(false)
  let dragDepth = 0

  function dragHasFiles(e) {
    const types = e.dataTransfer?.types
    return types && Array.from(types).includes('Files')
  }

  function onDragEnter(e) {
    if (!canDrop() || !dragHasFiles(e)) return
    dragDepth++
    dragOver.value = true
  }

  function onDragOver(e) {
    if (canDrop() && dragHasFiles(e)) e.dataTransfer.dropEffect = 'copy'
  }

  function onDragLeave() {
    dragDepth = Math.max(0, dragDepth - 1)
    if (dragDepth === 0) dragOver.value = false
  }

  function onDrop(e) {
    dragDepth = 0
    dragOver.value = false
    if (!canDrop()) return
    const files = Array.from(e.dataTransfer?.files || [])
    if (files.length) onFiles(files)
  }

  return {
    dragOver,
    onDragEnter,
    onDragOver,
    onDragLeave,
    onDrop,
  }
}
