import { usePetsStore } from '@/stores/pets.js'

// Канал petsvc (gw2:pets:events). Единственное вещаемое событие — pet:update
// (после кормления/прогулки/лечения/поглаживания/покупки/эволюции), в свою
// user-комнату — синхронизация вкладок владельца, плюс отражение в зоопарке.
export function registerPetsSocketHandlers(socket) {
  socket.on('pet:update', (data) => {
    try { usePetsStore().applyPetUpdate(data) } catch { /* noop */ }
  })

  // Опорные точки блока «Сейчас в эфире» — сокет-события юнитов tasksvc.
  const refreshLive = () => {
    try {
      const pets = usePetsStore()
      if (pets.liveLoaded) pets.fetchLive().catch(() => {})
    } catch { /* noop */ }
  }
  socket.on('unit:started', refreshLive)
  socket.on('unit:stopped', refreshLive)
}
