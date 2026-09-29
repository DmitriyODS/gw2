/* Обработчики событий разделов, которым нечего делать в первом кадре: их
   сторы и api нужны только открытому разделу, поэтому модуль грузится
   отдельным чанком при подключении сокета (см. connectSocket), а кадры,
   пришедшие до его загрузки, придерживает GatewaySocket.holdEvents. */
import { registerTaskSocketHandlers } from '@/socket/tasks.js'
import { registerRegistrySocketHandlers } from '@/socket/registry.js'
import { registerFormsSocketHandlers } from '@/socket/forms.js'
import { registerCalendarSocketHandlers } from '@/socket/calendar.js'
import { registerDiarySocketHandlers } from '@/socket/diary.js'
import { registerScheduleSocketHandlers } from '@/socket/schedule.js'
import { registerNotesSocketHandlers } from '@/socket/notes.js'
import { registerBoardsSocketHandlers } from '@/socket/boards.js'
import { registerDriveSocketHandlers } from '@/socket/drive.js'
import { registerRemindersSocketHandlers } from '@/socket/reminders.js'
import { registerBillingSocketHandlers } from '@/socket/billing.js'

export function registerSectionSocketHandlers(socket) {
  registerTaskSocketHandlers(socket)
  registerRegistrySocketHandlers(socket)
  registerFormsSocketHandlers(socket)
  registerCalendarSocketHandlers(socket)
  registerDiarySocketHandlers(socket)
  registerScheduleSocketHandlers(socket)
  registerNotesSocketHandlers(socket)
  registerBoardsSocketHandlers(socket)
  registerDriveSocketHandlers(socket)
  registerRemindersSocketHandlers(socket)
  registerBillingSocketHandlers(socket)
}
