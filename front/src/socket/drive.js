import { useDriveStore } from '@/stores/drive.js'

// События диска приходят поимённо: владельцу и тем, кому он открыл доступ.
// Применяются идемпотентно (upsert по id) — своё действие уже обновило список,
// и эхо не должно плодить дубли.
export function registerDriveSocketHandlers(socket) {
  socket.on('drive_file:created', (p) => useDriveStore().upsertFile(p))
  socket.on('drive_file:updated', (p) => useDriveStore().upsertFile(p))
  socket.on('drive_file:trashed', (p) => useDriveStore().removeFile(p?.id))
  socket.on('drive_file:deleted', (p) => useDriveStore().removeFile(p?.id))

  socket.on('drive_folder:created', (p) => useDriveStore().upsertFolder(p))
  socket.on('drive_folder:updated', (p) => useDriveStore().upsertFolder(p))
  socket.on('drive_folder:trashed', (p) => useDriveStore().removeFolder(p?.id))
  socket.on('drive_folder:deleted', (p) => useDriveStore().removeFolder(p?.id))

  // Со мной поделились — вкладка «Поделились» перечитывается при открытии,
  // поэтому здесь достаточно обновить открытую именно её.
  socket.on('drive:shared', () => {
    const store = useDriveStore()
    if (store.view === 'shared') store.load()
  })
}
