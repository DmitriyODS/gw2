/* Общий AudioContext коротких звуков (уведомления, кудосы).

   Работающий контекст непрерывно гоняет аудиопоток ОС, даже когда молчит:
   на телефоне это держит аудиотракт, на macOS не пускает приложение в App Nap.
   Поэтому контекст живёт в suspended и просыпается только на время звука.

   Браузеры разрешают звук после первого жеста: `unlockAudio()` зовут из
   обработчика жеста, дальше `playSound()` будит контекст и без него. */

const IDLE_MS = 400 // запас после последней ноты — хвост затухания не обрезается

let ctx = null
let sleepTimer = null
let busyUntil = 0
let unlocked = false

function getContext() {
  if (ctx) return ctx
  try {
    const Ctx = window.AudioContext || window.webkitAudioContext
    if (Ctx) ctx = new Ctx()
  } catch {
    ctx = null
  }
  return ctx
}

function scheduleSleep(seconds) {
  busyUntil = Math.max(busyUntil, Date.now() + seconds * 1000)
  clearTimeout(sleepTimer)
  sleepTimer = setTimeout(() => {
    // Без проверки state: незавершённый resume() ещё числится suspended, а
    // suspend/resume браузер выполняет по очереди — итог всегда «спит».
    if (ctx && ctx.state !== 'closed') ctx.suspend().catch(() => {})
  }, busyUntil - Date.now() + IDLE_MS)
}

/** Разблокировка из обработчика жеста: контекст просыпается и сразу засыпает. */
export function unlockAudio() {
  if (unlocked) return
  const ac = getContext()
  if (!ac) return
  unlocked = true
  if (ac.state === 'suspended') ac.resume().catch(() => {})
  scheduleSleep(0)
}

/**
 * Проигрывает звук: `render(ctx, startAt)` расставляет узлы, `duration` — его
 * длина в секундах (по ней контекст засыпает обратно).
 */
export function playSound(duration, render) {
  const ac = getContext()
  if (!ac) return
  try {
    if (ac.state === 'suspended') ac.resume().catch(() => {})
    render(ac, ac.currentTime + 0.02)
    scheduleSleep(duration + 0.02)
  } catch {}
}
