import { useNotesStore } from '@/stores/notes.js'

// События заметок приходят в комнаты владельца и аудитории шаринга (адресаты +
// участники компаний, в т.ч. через расшаренные папки-предки). note_collab:*
// (присутствие/курсоры/живые правки) стор не трогают — их слушает открытый
// редактор заметки (useNoteCollab).
export function registerNotesSocketHandlers(socket) {
  socket.on('note:created', (p) => useNotesStore().applyNoteSocket('created', p))
  socket.on('note:updated', (p) => useNotesStore().applyNoteSocket('updated', p))
  socket.on('note:deleted', (p) => useNotesStore().applyNoteSocket('deleted', p))

  socket.on('note_folder:created', (p) => useNotesStore().applyFolderSocket('created', p))
  socket.on('note_folder:updated', (p) => useNotesStore().applyFolderSocket('updated', p))
  socket.on('note_folder:deleted', (p) => useNotesStore().applyFolderSocket('deleted', p))

  socket.on('note_tag:created', (p) => useNotesStore().applyTagSocket('created', p))
  socket.on('note_tag:updated', (p) => useNotesStore().applyTagSocket('updated', p))
  socket.on('note_tag:deleted', (p) => useNotesStore().applyTagSocket('deleted', p))

  // Заметка/папка появилась или пропала в «Поделились со мной».
  socket.on('note_member:added', () => useNotesStore().applyShareSocket('added'))
  socket.on('note_member:removed', () => useNotesStore().applyShareSocket('removed'))
  socket.on('note_folder:shared', () => useNotesStore().applyShareSocket('added'))
  socket.on('note_folder:unshared', () => useNotesStore().applyShareSocket('removed'))
}
