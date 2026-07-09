import { useNotesStore } from '@/stores/notes.js'

// События заметок приходят в комнаты владельца и адресатов шаринга. Имена
// групп — note_group:*, адресный шаринг — note_member:* (не пересекаться с
// другими сервисами). note_collab:* (присутствие/курсоры/живые правки) стор
// не трогают — их слушает открытый редактор заметки (useNoteCollab).
export function registerNotesSocketHandlers(socket) {
  socket.on('note:created', (p) => useNotesStore().applyNoteSocket('created', p))
  socket.on('note:updated', (p) => useNotesStore().applyNoteSocket('updated', p))
  socket.on('note:deleted', (p) => useNotesStore().applyNoteSocket('deleted', p))

  socket.on('note_group:created', (p) => useNotesStore().applyGroupSocket('created', p))
  socket.on('note_group:updated', (p) => useNotesStore().applyGroupSocket('updated', p))
  socket.on('note_group:deleted', (p) => useNotesStore().applyGroupSocket('deleted', p))

  socket.on('note_member:added', (p) => useNotesStore().applyMemberSocket('added', p))
  socket.on('note_member:removed', (p) => useNotesStore().applyMemberSocket('removed', p))
}
