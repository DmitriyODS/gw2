/* Системные уведомления браузера и звук для мессенджера.

   Требует разрешения пользователя (Notification.requestPermission).
   Звук генерируется через Web Audio API (короткий двухтональный «бип»),
   чтобы не таскать mp3 в репозитории. Браузеры запрещают звук до первого
   user gesture — первые попытки до клика тихо проглатываются. */

let warned = false
let audioCtx = null

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

export function showSystemNotification(title, body, onClick) {
  if (!notificationsAllowed()) return null
  try {
    const n = new Notification(title, {
      body,
      icon: '/favicon.svg',
      silent: true, // звук проигрываем сами, чтобы не задвоился
      tag: 'gw2-message',
      renotify: true,
    })
    if (onClick) {
      n.onclick = () => {
        try { onClick() } finally { n.close() }
      }
    }
    return n
  } catch (e) {
    if (!warned) { console.warn('Notification failed', e); warned = true }
    return null
  }
}

export function playNotifySound() {
  try {
    playBeep()
  } catch (e) {
    if (!warned) { console.warn('notify sound failed', e); warned = true }
  }
}
