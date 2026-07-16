import { usePetsStore } from '@/stores/pets.js'

// Канал petsvc (gw2:pets:events): pet:update (после кормления/прогулки/
// лечения/поглаживания/покупки/эволюции, в user-комнату владельца —
// синхронизация вкладок плюс отражение в зоопарке) и pet:deleted
// (администратор удалил питомца сотрудника, комната all).
export function registerPetsSocketHandlers(socket) {
  socket.on('pet:update', (data) => {
    try { usePetsStore().applyPetUpdate(data) } catch { /* noop */ }
  })

  // Администратор удалил питомца сотрудника (комната all, фильтр по компании
  // в сторе): зоопарк без него, владельцу питомец пересоздаётся заново.
  socket.on('pet:deleted', (data) => {
    try { usePetsStore().applyPetDeleted(data) } catch { /* noop */ }
  })

  // Входящий перевод кудо-банка (адресно в комнату получателя): тост
  // «+N кудосов от …»; баланс приедет соседним pet:update.
  socket.on('kudos:received', (data) => {
    try { usePetsStore().applyKudosReceived(data) } catch { /* noop */ }
  })

  // Грувик заболел / сбежал (адресно владельцу; офлайн-хозяину те же события
  // приезжают пушем через pushsvc).
  socket.on('pet:sick', (data) => {
    try { usePetsStore().applyPetSick(data) } catch { /* noop */ }
  })
  socket.on('pet:runaway', (data) => {
    try { usePetsStore().applyPetRunaway(data) } catch { /* noop */ }
  })

  // Благотворительный сбор компании: создан/пополнен/собран/закрыт
  // (комната all, фильтр по компании в сторе).
  socket.on('bank:fund', (data) => {
    try { usePetsStore().applyBankFund(data) } catch { /* noop */ }
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
