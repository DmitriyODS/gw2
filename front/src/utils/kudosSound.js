// Звук отправки кудосов: тёплый восходящий «звон монетки» на Web Audio,
// без аудиофайлов (ничего не грузим). Контекст создаётся лениво по первому
// вызову — вызовы идут из обработчиков кликов, жест уже есть.

let ctx = null

function audioContext() {
  if (!ctx) {
    const AC = window.AudioContext || window.webkitAudioContext
    if (!AC) return null
    ctx = new AC()
  }
  if (ctx.state === 'suspended') ctx.resume().catch(() => {})
  return ctx
}

// Один колокольчик: основная синусоида + октавная гармоника, мягкая атака
// и экспоненциальное затухание.
function chime(ac, freq, at, dur, gainPeak) {
  const gain = ac.createGain()
  gain.gain.setValueAtTime(0.0001, at)
  gain.gain.exponentialRampToValueAtTime(gainPeak, at + 0.012)
  gain.gain.exponentialRampToValueAtTime(0.0001, at + dur)
  gain.connect(ac.destination)

  const osc = ac.createOscillator()
  osc.type = 'sine'
  osc.frequency.setValueAtTime(freq, at)
  osc.connect(gain)
  osc.start(at)
  osc.stop(at + dur + 0.05)

  const shimmer = ac.createGain()
  shimmer.gain.setValueAtTime(0.0001, at)
  shimmer.gain.exponentialRampToValueAtTime(gainPeak * 0.28, at + 0.012)
  shimmer.gain.exponentialRampToValueAtTime(0.0001, at + dur * 0.7)
  shimmer.connect(ac.destination)

  const overtone = ac.createOscillator()
  overtone.type = 'triangle'
  overtone.frequency.setValueAtTime(freq * 2, at)
  overtone.connect(shimmer)
  overtone.start(at)
  overtone.stop(at + dur + 0.05)
}

// Отправка перевода: три быстрые восходящие ноты (мажорное арпеджио) +
// финальный «блеск» октавой выше — короткий, тёплый, не назойливый.
export function playKudosSent() {
  const ac = audioContext()
  if (!ac) return
  const t = ac.currentTime + 0.02
  chime(ac, 1046.5, t, 0.28, 0.09)          // C6
  chime(ac, 1318.5, t + 0.085, 0.3, 0.09)   // E6
  chime(ac, 1568.0, t + 0.17, 0.42, 0.1)    // G6
  chime(ac, 2093.0, t + 0.26, 0.5, 0.05)    // C7 — блеск
}

// Входящие кудосы: два ласковых колокольчика повыше.
export function playKudosReceived() {
  const ac = audioContext()
  if (!ac) return
  const t = ac.currentTime + 0.02
  chime(ac, 1568.0, t, 0.3, 0.07)          // G6
  chime(ac, 2093.0, t + 0.12, 0.5, 0.08)   // C7
}
