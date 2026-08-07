import { useBoardsStore } from '@/stores/boards.js'

// События досок приходят в комнаты владельца и аудитории шаринга (адресаты +
// участники компаний, в т.ч. через расшаренные папки-предки). board_collab:*
// (присутствие/курсоры/живые правки холста) стор не трогают — их слушает
// открытый редактор доски (useBoardCollab).
export function registerBoardsSocketHandlers(socket) {
  socket.on('board:created', (p) => useBoardsStore().applyBoardSocket('created', p))
  socket.on('board:updated', (p) => useBoardsStore().applyBoardSocket('updated', p))
  socket.on('board:deleted', (p) => useBoardsStore().applyBoardSocket('deleted', p))

  socket.on('board_folder:created', (p) => useBoardsStore().applyFolderSocket('created', p))
  socket.on('board_folder:updated', (p) => useBoardsStore().applyFolderSocket('updated', p))
  socket.on('board_folder:deleted', (p) => useBoardsStore().applyFolderSocket('deleted', p))


  // Доска/папка появилась или пропала в «Поделились со мной».
  socket.on('board_member:added', () => useBoardsStore().applyShareSocket())
  socket.on('board_member:removed', () => useBoardsStore().applyShareSocket())
  socket.on('board_folder:shared', () => useBoardsStore().applyShareSocket())
  socket.on('board_folder:unshared', () => useBoardsStore().applyShareSocket())
}
