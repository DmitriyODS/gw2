import { useSchedulesStore } from '@/stores/schedules.js'

// События расписаний приходят адресно (комнаты владельца и адресатов): общей
// комнаты у раздела нет — расписание не принадлежит компании.
export function registerScheduleSocketHandlers(socket) {
  socket.on('schedule:created', (p) => useSchedulesStore().applyScheduleSocket('created', p))
  socket.on('schedule:updated', (p) => useSchedulesStore().applyScheduleSocket('updated', p))
  socket.on('schedule:deleted', (p) => useSchedulesStore().applyScheduleSocket('deleted', p))
  socket.on('schedule:shared', (p) => useSchedulesStore().applyScheduleSocket('shared', p))
  socket.on('schedule:unshared', (p) => useSchedulesStore().applyScheduleSocket('unshared', p))

  socket.on('schedule_item:created', (p) => useSchedulesStore().applyItemSocket('created', p))
  socket.on('schedule_item:updated', (p) => useSchedulesStore().applyItemSocket('updated', p))
  socket.on('schedule_item:deleted', (p) => useSchedulesStore().applyItemSocket('deleted', p))
}
