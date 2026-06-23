/* Service worker Groove Work.
   1) ServiceWorkerRegistration.showNotification — на Android Chrome конструктор
      new Notification() запрещён, OS-уведомления показываются только через SW.
      Push здесь нет (нет сервера рассылки): уведомления показываются, пока жива
      вкладка, из основного потока через registration.showNotification.
   2) Кэш оболочки приложения (app shell) — чтобы приложение было устанавливаемым
      PWA (Chrome предлагает «Установить» только при SW с fetch-обработчиком) и
      открывалось офлайн. API/WS/uploads/livekit НЕ кэшируем — это живой трафик. */

const CACHE = 'gw-shell-v1'
const APP_SHELL = ['/', '/index.html', '/logo.svg', '/manifest.webmanifest']

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE).then((c) => c.addAll(APP_SHELL)).catch(() => {})
  )
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil((async () => {
    // Удаляем кэши прежних версий оболочки.
    const keys = await caches.keys()
    await Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k)))
    await self.clients.claim()
  })())
})

// Пути живого трафика — мимо кэша, всегда в сеть.
function isLiveTraffic(url) {
  return url.pathname.startsWith('/api/')
    || url.pathname.startsWith('/ws')
    || url.pathname.startsWith('/uploads/')
    || url.pathname.startsWith('/livekit/')
    || url.pathname.startsWith('/apps/')
}

self.addEventListener('fetch', (event) => {
  const req = event.request
  if (req.method !== 'GET') return
  const url = new URL(req.url)
  if (url.origin !== self.location.origin) return // шрифты и пр. — браузеру
  if (isLiveTraffic(url)) return

  // Навигация (открытие страницы) — network-first с откатом на кэш оболочки,
  // чтобы офлайн SPA всё равно загрузилась (роутинг разрулит Vue Router).
  if (req.mode === 'navigate') {
    event.respondWith((async () => {
      try {
        const fresh = await fetch(req)
        const cache = await caches.open(CACHE)
        cache.put('/index.html', fresh.clone()).catch(() => {})
        return fresh
      } catch {
        return (await caches.match(req)) || (await caches.match('/index.html'))
      }
    })())
    return
  }

  // Статика сборки (хешированные /assets/*, иконки, лого) — stale-while-revalidate:
  // мгновенно из кэша, фоном обновляем. Имена с хешем — переписать не страшно.
  event.respondWith((async () => {
    const cache = await caches.open(CACHE)
    const cached = await cache.match(req)
    const network = fetch(req).then((res) => {
      if (res && res.ok) cache.put(req, res.clone()).catch(() => {})
      return res
    }).catch(() => null)
    return cached || (await network) || fetch(req)
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
