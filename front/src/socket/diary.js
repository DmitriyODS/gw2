import { useDiariesStore } from '@/stores/diaries.js'

// События ежедневников приходят адресно (комнаты владельца и адресатов), поэтому
// клиент получает только релевантные ему. Имена записей — diary_entry:* (чтобы
// не пересекаться с записями календаря entry:*).
export function registerDiarySocketHandlers(socket) {
  socket.on('diary:created', (p) => useDiariesStore().applyDiarySocket('created', p))
  socket.on('diary:updated', (p) => useDiariesStore().applyDiarySocket('updated', p))
  socket.on('diary:deleted', (p) => useDiariesStore().applyDiarySocket('deleted', p))
  socket.on('diary:shared', (p) => useDiariesStore().applyDiarySocket('shared', p))
  socket.on('diary:unshared', (p) => useDiariesStore().applyDiarySocket('unshared', p))

  socket.on('diary_entry:created', (p) => useDiariesStore().applyEntrySocket(p))
  socket.on('diary_entry:updated', (p) => useDiariesStore().applyEntrySocket(p))
  socket.on('diary_entry:deleted', (p) => useDiariesStore().applyEntrySocket(p))
  socket.on('diary_entry:bulk-deleted', (p) => useDiariesStore().applyEntrySocket(p))
}
