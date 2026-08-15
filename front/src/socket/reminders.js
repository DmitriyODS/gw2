import { useRemindersStore } from '@/stores/reminders.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { showSystemNotification } from '@/utils/systemNotify.js'
import { pushNotification } from '@/composables/useDesktopNotifications.js'
import { isSourceEnabled } from '@/utils/notifySettings.js'

/* Напоминания приходят только в комнату владельца. reminder:fire — момент
   срабатывания: тост во вкладке, системное (в десктоп-обёртке — нативное ОС)
   уведомление и звук. Тем, кто офлайн, то же событие ушло FCM-пушем из
   pushsvc, поэтому здесь дубля не будет. */
export function registerRemindersSocketHandlers(socket) {
  socket.on('reminder:fire', (p) => {
    const store = useRemindersStore()
    store.applyFire(p)
    // Раздел выключен в «Настройки → Уведомления»: срабатывание видно в самих
    // напоминаниях, но ни тоста, ни системного уведомления, ни звука.
    if (!isSourceEnabled('reminders')) return

    const title = p?.title || 'Напоминание'
    const body = p?.note || 'Пора!'
    // Тост живёт секунды, а сработавшее напоминание больше нигде не хранится —
    // кладём его в центр уведомлений, чтобы пережило перезагрузку страницы.
    pushNotification({
      key: `reminder-${p?.id}`,
      icon: 'alarm',
      tone: 'alert',
      title,
      text: body,
      path: '/reminders',
    })
    useNotificationsStore().notify({
      severity: 'info', summary: `⏰ ${title}`, detail: body, life: 12_000, sound: 'alarm', source: 'reminders',
    })
    showSystemNotification(`⏰ ${title}`, body, {
      data: { type: 'reminder', reminder_id: p?.id },
      onClick: () => { window.location.href = '/reminders' },
    })
  })

  socket.on('reminder:created', (p) => useRemindersStore().applySocket('created', p))
  socket.on('reminder:updated', (p) => useRemindersStore().applySocket('updated', p))
  socket.on('reminder:deleted', (p) => useRemindersStore().applySocket('deleted', p))
}
