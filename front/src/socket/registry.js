import { useRegistriesStore } from '@/stores/registries.js'

export function registerRegistrySocketHandlers(socket) {
  socket.on('registry:created', (p) => useRegistriesStore().applyRegistrySocket('created', p))
  socket.on('registry:updated', (p) => useRegistriesStore().applyRegistrySocket('updated', p))
  socket.on('registry:deleted', (p) => useRegistriesStore().applyRegistrySocket('deleted', p))

  socket.on('record:created', (p) => useRegistriesStore().applyRecordSocket('created', p))
  socket.on('record:updated', (p) => useRegistriesStore().applyRecordSocket('updated', p))
  socket.on('record:deleted', (p) => useRegistriesStore().applyRecordSocket('deleted', p))
  socket.on('record:bulk-deleted', (p) => useRegistriesStore().applyRecordSocket('deleted', p))
}
