/* Системные уведомления браузера и звук для мессенджера.

   Требует разрешения пользователя (Notification.requestPermission).
   Звук генерируется через Web Audio API (короткий двухтональный «бип»),
   чтобы не таскать mp3 в репозитории. Браузеры запрещают звук до первого
   user gesture — первые попытки до клика тихо проглатываются. */

let warned = false
let audioCtx = null
let unlockInstalled = false

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
    // Клик по уведомлению из SW → открываем нужный чат во вкладке.
    navigator.serviceWorker.addEventListener('message', (e) => {
      if (e.data?.type === 'open-conversation') {
        window.focus?.()
        window.dispatchEvent(new CustomEvent('messenger:open-conversation', {
          detail: { conversation_id: e.data.conversation_id },
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

export function playNotifySound() {
  try {
    playBeep()
  } catch (e) {
    if (!warned) { console.warn('notify sound failed', e); warned = true }
  }
}
