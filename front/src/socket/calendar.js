import { useCalendarsStore } from '@/stores/calendars.js'

export function registerCalendarSocketHandlers(socket) {
  socket.on('calendar:created', (p) => useCalendarsStore().applyCalendarSocket('created', p))
  socket.on('calendar:updated', (p) => useCalendarsStore().applyCalendarSocket('updated', p))
  socket.on('calendar:deleted', (p) => useCalendarsStore().applyCalendarSocket('deleted', p))

  socket.on('entry:created', (p) => useCalendarsStore().applyEntrySocket('created', p))
  socket.on('entry:updated', (p) => useCalendarsStore().applyEntrySocket('updated', p))
  socket.on('entry:deleted', (p) => useCalendarsStore().applyEntrySocket('deleted', p))
  socket.on('entry:bulk-deleted', (p) => useCalendarsStore().applyEntrySocket('deleted', p))
}
