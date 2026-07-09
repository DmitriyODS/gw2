import { useNotesStore } from '@/stores/notes.js'

// События заметок приходят только в комнату владельца (заметки строго
// приватные). Имена групп — note_group:* (не пересекаться с другими сервисами).
export function registerNotesSocketHandlers(socket) {
  socket.on('note:created', (p) => useNotesStore().applyNoteSocket('created', p))
  socket.on('note:updated', (p) => useNotesStore().applyNoteSocket('updated', p))
  socket.on('note:deleted', (p) => useNotesStore().applyNoteSocket('deleted', p))

  socket.on('note_group:created', (p) => useNotesStore().applyGroupSocket('created', p))
  socket.on('note_group:updated', (p) => useNotesStore().applyGroupSocket('updated', p))
  socket.on('note_group:deleted', (p) => useNotesStore().applyGroupSocket('deleted', p))
}
