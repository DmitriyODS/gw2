/* Service worker Groove Work — ТОЛЬКО OS-уведомления.

   Зачем он вообще нужен: на Android Chrome конструктор new Notification()
   запрещён, и показать уведомление из живой вкладки можно единственным способом —
   registration.showNotification. Мобильный веб у нас рабочий сценарий, поэтому
   регистрация остаётся. Нативным клиентам он не нужен: Android-обёртка получает
   FCM-пуши, Electron показывает тосты конструктором.

   Push-канала здесь нет: уведомления показываются, пока жива вкладка.

   Кэша здесь тоже НЕТ, и это намеренно. Прежняя офлайн-оболочка кэшировала всё,
   что не живой трафик, — на dev-сервере под это попадали модули Vite, которые
   потом отдавались cache-first и разъезжались со свежим графом: разделы падали
   на ровном месте. Офлайн-старт клиентов от неё не зависел и не зависит — у
   Electron свой сплэш с автоповтором (desktop/main.js), у Capacitor-обёртки свой
   error.html. Ценой отказа от fetch-обработчика Chrome перестаёт предлагать
   «Установить» — установка PWA и так нигде не предлагалась. */

self.addEventListener('install', () => self.skipWaiting())

self.addEventListener('activate', (event) => {
  event.waitUntil((async () => {
    // Разовая уборка за прежними версиями: кэшей оболочки мы больше не ведём,
    // а у клиентов они остались с предыдущих сборок.
    const keys = await caches.keys()
    await Promise.all(keys.filter((k) => k.startsWith('gw-shell')).map((k) => caches.delete(k)))
    await self.clients.claim()
  })())
})

self.addEventListener('notificationclick', (event) => {
  const data = event.notification.data || {}
  event.notification.close()
  event.waitUntil((async () => {
    const all = await self.clients.matchAll({ type: 'window', includeUncontrolled: true })
    // Предпочитаем уже видимую/сфокусированную вкладку: клик по уведомлению
    // не должен уводить в случайное фоновое окно.
    const target = all.find((c) => c.focused)
      || all.find((c) => c.visibilityState === 'visible')
      || all.find((c) => 'focus' in c)
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
