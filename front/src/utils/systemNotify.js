/* Системные уведомления браузера и звук для мессенджера/звонков.

   Требует разрешения пользователя (Notification.requestPermission).
   Звук генерируется через Web Audio API (короткий двухтональный «бип»),
   чтобы не таскать mp3 в репозитории. Браузеры запрещают звук до первого
   user gesture — первые попытки до клика тихо проглатываются. */

let warned = false
let audioCtx = null
let unlockInstalled = false
// Открытое сейчас уведомление о звонке (десктоп — конструктор Notification).
let activeCallNotification = null
// На SW-варианте тег используется для перезаписи и закрытия.
const CALL_NOTIF_TAG = 'gw2-call'

function getCtx() {
  if (audioCtx) return audioCtx
  try {
    const Ctx = window.AudioContext || window.webkitAudioContext
    if (!Ctx) return null
    audioCtx = new Ctx()
  } catch {
    audioCtx = null
  }
  return audioCtx
}

/* Браузеры блокируют Web Audio и (в Safari) Notification.requestPermission до
   первого пользовательского жеста. Вешаем одноразовые слушатели: при первом
   клике/нажатии «разогреваем» AudioContext, чтобы фоновый «бип» о новом
   сообщении точно проигрывался, и заодно тихо просим разрешение на уведомления,
   если оно ещё не выдано. Это закрывает «иногда приходит, иногда нет». */
export function installNotifyUnlock() {
  if (unlockInstalled || typeof window === 'undefined') return
  unlockInstalled = true
  const handler = () => {
    const ctx = getCtx()
    if (ctx && ctx.state === 'suspended') {
      ctx.resume().catch(() => {})
    }
    // Разрешение просим только пока оно «default». Слушатели снимаем лишь когда
    // вопрос решён (granted/denied) — иначе, если пользователь отмахнулся от
    // первого prompt, уведомления уже не запросятся никогда.
    if ('Notification' in window && Notification.permission === 'default') {
      requestNotificationPermission()
      return
    }
    window.removeEventListener('pointerdown', handler)
    window.removeEventListener('keydown', handler)
  }
  window.addEventListener('pointerdown', handler, { passive: true })
  window.addEventListener('keydown', handler, { passive: true })
}

function playBeep() {
  const ctx = getCtx()
  if (!ctx) return
  try {
    if (ctx.state === 'suspended') ctx.resume()
    const now = ctx.currentTime
    const tones = [
      { freq: 880, start: 0,    dur: 0.12 },
      { freq: 660, start: 0.13, dur: 0.18 },
    ]
    tones.forEach(({ freq, start, dur }) => {
      const osc = ctx.createOscillator()
      const gain = ctx.createGain()
      osc.type = 'sine'
      osc.frequency.value = freq
      gain.gain.setValueAtTime(0, now + start)
      gain.gain.linearRampToValueAtTime(0.18, now + start + 0.01)
      gain.gain.exponentialRampToValueAtTime(0.0001, now + start + dur)
      osc.connect(gain).connect(ctx.destination)
      osc.start(now + start)
      osc.stop(now + start + dur + 0.02)
    })
  } catch {}
}

let swRegistration = null

/* Регистрируем service worker — нужен для OS-уведомлений на мобильных
   (Android Chrome запрещает new Notification(), только showNotification
   через регистрацию SW). Вызывается один раз при старте приложения. */
export async function registerNotifyServiceWorker() {
  if (typeof navigator === 'undefined' || !('serviceWorker' in navigator)) return
  try {
    await navigator.serviceWorker.register('/sw.js')
    swRegistration = await navigator.serviceWorker.ready
    // Клик по уведомлению из SW → открываем нужный чат / фокусируем звонок.
    navigator.serviceWorker.addEventListener('message', (e) => {
      const t = e.data?.type
      if (t === 'open-conversation') {
        window.focus?.()
        window.dispatchEvent(new CustomEvent('messenger:open-conversation', {
          detail: { conversation_id: e.data.conversation_id },
        }))
      } else if (t === 'focus-call') {
        window.focus?.()
        window.dispatchEvent(new CustomEvent('call:focus-overlay', {
          detail: { call_id: e.data.call_id },
        }))
      }
    })
  } catch (e) {
    if (!warned) { console.warn('SW register failed', e); warned = true }
  }
}

export function notificationsAllowed() {
  return typeof window !== 'undefined'
    && 'Notification' in window
    && Notification.permission === 'granted'
}

export async function requestNotificationPermission() {
  if (typeof window === 'undefined' || !('Notification' in window)) return false
  if (Notification.permission === 'granted') return true
  if (Notification.permission === 'denied') return false
  try {
    const result = await Notification.requestPermission()
    return result === 'granted'
  } catch {
    return false
  }
}

/* Показывает OS-уведомление. data — произвольные данные (передаём
   conversation_id, чтобы клик открыл нужный чат).
   На десктопе используем конструктор Notification (самый надёжный путь, он же
   даёт onclick). Если конструктор недоступен (Android Chrome запрещает
   `new Notification` — бросает исключение), уходим в service worker. */
export function showSystemNotification(title, body, { onClick, data } = {}) {
  if (!notificationsAllowed()) return

  const options = {
    body,
    icon: '/logo.svg',
    badge: '/logo.svg',
    tag: 'gw2-message',
    renotify: true,
    data: data || {},
  }

  try {
    const n = new Notification(title, { ...options, silent: true })
    if (onClick) {
      n.onclick = () => {
        try { window.focus?.(); onClick() } finally { n.close() }
      }
    }
    return
  } catch (e) {
    // Конструктор недоступен (мобильный Chrome) — пробуем через SW.
  }

  if (swRegistration && typeof swRegistration.showNotification === 'function') {
    swRegistration.showNotification(title, options).catch((e) => {
      if (!warned) { console.warn('SW notification failed', e); warned = true }
    })
  } else if (!warned) {
    console.warn('Notification: нет ни конструктора, ни активного service worker')
    warned = true
  }
}

/* Уведомление о входящем звонке. Отличается от сообщений отдельным `tag`
   (чтобы не перезаписывало последнее сообщение и наоборот), `requireInteraction:
   true` (на десктопе ОС не скроет его автоматически через 5 секунд) и тем,
   что мы сохраняем reference и умеем явно закрыть его, когда звонок принят
   или завершён. */
export function showCallNotification(title, body, { callId, onClick } = {}) {
  if (!notificationsAllowed()) return
  closeCallNotification()

  const options = {
    body,
    icon: '/logo.svg',
    badge: '/logo.svg',
    tag: CALL_NOTIF_TAG,
    renotify: true,
    requireInteraction: true,
    silent: true,
    data: { call_id: callId, kind: 'call' },
  }

  try {
    const n = new Notification(title, options)
    activeCallNotification = n
    n.onclick = () => {
      try { window.focus?.(); onClick?.() } finally { n.close(); activeCallNotification = null }
    }
    n.onclose = () => { activeCallNotification = null }
    return
  } catch (e) {
    // Конструктор недоступен — Android Chrome и т. п.
  }

  if (swRegistration && typeof swRegistration.showNotification === 'function') {
    swRegistration.showNotification(title, options).catch((e) => {
      if (!warned) { console.warn('SW notification failed', e); warned = true }
    })
  } else if (!warned) {
    console.warn('Notification: нет ни конструктора, ни активного service worker')
    warned = true
  }
}

export function closeCallNotification() {
  if (activeCallNotification) {
    try { activeCallNotification.close() } catch {}
    activeCallNotification = null
  }
  // SW-вариант закрываем через getNotifications по тегу.
  if (swRegistration && typeof swRegistration.getNotifications === 'function') {
    swRegistration.getNotifications({ tag: CALL_NOTIF_TAG }).then((list) => {
      for (const n of list || []) {
        try { n.close() } catch {}
      }
    }).catch(() => {})
  }
}

export function playNotifySound() {
  try {
    playBeep()
  } catch (e) {
    if (!warned) { console.warn('notify sound failed', e); warned = true }
  }
}
