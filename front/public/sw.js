/* Минимальный service worker Groove Work.
   Нужен ради ServiceWorkerRegistration.showNotification — на Android Chrome
   конструктор new Notification() запрещён, OS-уведомления показываются только
   через SW. Push здесь нет (нет сервера рассылки): уведомления показываются,
   пока вкладка жива, из основного потока через registration.showNotification.
   Клик по уведомлению — фокус на существующей вкладке или открытие новой. */

self.addEventListener('install', (event) => {
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim())
})

self.addEventListener('notificationclick', (event) => {
  const data = event.notification.data || {}
  event.notification.close()
  event.waitUntil((async () => {
    const all = await self.clients.matchAll({ type: 'window', includeUncontrolled: true })
    const target = all.find((c) => 'focus' in c)
    // Уведомление о звонке: focus + сообщение основному потоку, чтобы тот
    // показал/развернул overlay с принятым/входящим вызовом.
    if (data.kind === 'call') {
      if (target) {
        await target.focus()
        target.postMessage({ type: 'focus-call', call_id: data.call_id })
        return
      }
      if (self.clients.openWindow) await self.clients.openWindow('/')
      return
    }
    // Сообщение мессенджера: открыть конкретный чат
    if (target) {
      await target.focus()
      target.postMessage({ type: 'open-conversation', conversation_id: data.conversation_id })
      return
    }
    const url = data.conversation_id ? `/messenger/${data.conversation_id}` : '/messenger'
    if (self.clients.openWindow) await self.clients.openWindow(url)
  })())
})
