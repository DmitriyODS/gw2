// Интеграция с нативной мобильной обёрткой (mobile/ — Capacitor поверх
// прод-URL). Фронт приезжает с сервера и НЕ бандлит @capacitor/*: обёртка
// сама инжектирует мост в window.Capacitor, поэтому все обращения — через
// него и с guard'ами (в браузере/Electron модуль превращается в no-op).
import { registerPushToken, unregisterPushToken } from '@/api/push.js'

export const isNativeApp = () => window.Capacitor?.isNativePlatform?.() === true

const pushPlugin = () => window.Capacitor?.Plugins?.PushNotifications

let currentToken = null
let listenersInstalled = false

// Регистрация пуш-уведомлений: запрос разрешения (Android 13+) → register()
// → событие 'registration' с FCM-токеном → POST /api/push/register.
// Идемпотентна: слушатели вешаются один раз, повторный register безвреден
// (тот же токен просто обновит updated_at в device_tokens).
export async function initNativePush(onOpen) {
  const push = pushPlugin()
  if (!isNativeApp() || !push) return

  if (!listenersInstalled) {
    listenersInstalled = true
    // Срабатывает и при ротации FCM-токена — перерегистрируем прозрачно.
    push.addListener('registration', async ({ value }) => {
      currentToken = value
      try {
        await registerPushToken(value, window.Capacitor.getPlatform?.() || 'android')
      } catch {}
    })
    // Тап по системному уведомлению — открываем адресный экран.
    push.addListener('pushNotificationActionPerformed', ({ notification }) => {
      onOpen?.(notification?.data || {})
    })
  }

  try {
    let { receive } = await push.checkPermissions()
    if (receive !== 'granted') {
      ;({ receive } = await push.requestPermissions())
    }
    if (receive === 'granted') await push.register()
  } catch {}
}

// Снятие токена при логауте (до apiLogout — запросу нужна живая сессия),
// иначе устройство продолжит получать чужие пуши после выхода.
export async function unregisterNativePush() {
  if (!currentToken) return
  const token = currentToken
  currentToken = null
  try {
    await unregisterPushToken(token)
  } catch {}
}

/* ── NativeShell: собственный плагин обёртки (mobile/android) ── */

const nativeShell = () => window.Capacitor?.Plugins?.NativeShell

// Системные панели (статус-бар и навигация) следуют теме приложения: бар
// красится в ФАКТИЧЕСКИЙ базовый фон приложения — resolved background-color
// самого .app-layout (последний слой его background — var(--color-bg), поверх
// которого лежат градиентные пятна), поэтому полоса выглядит продолжением
// страницы. ВАЖНО: вызывать после обновления DOM (watch с flush:'post') —
// токены переопределяются селектором [data-dark] на .app-layout, и до
// перерисовки резолвится цвет предыдущей темы.
export function syncNativeSystemBars() {
  const shell = nativeShell()
  if (!isNativeApp() || !shell) return
  const host = document.querySelector('.app-layout') || document.body
  let rgba = cssColorToRgba(getComputedStyle(host).backgroundColor)
  if (!rgba || rgba[3] < 250) {
    // Фон не задан/полупрозрачен (экран логина и т.п.) — цвет из токенов,
    // резолвим probe-элементом внутри host (там действуют [data-dark]).
    const probe = document.createElement('div')
    probe.style.cssText = 'position:fixed;visibility:hidden;pointer-events:none;background:var(--color-bg, var(--color-surface))'
    host.appendChild(probe)
    rgba = cssColorToRgba(getComputedStyle(probe).backgroundColor)
    probe.remove()
  }
  if (!rgba) return
  const [r, g, b] = rgba
  const hex = '#' + [r, g, b].map((v) => v.toString(16).padStart(2, '0')).join('')
  // Читаемость иконок бара — по фактической яркости фона, а не по флагу темы:
  // «светлая» тема с тёмным кастомным фоном тоже получит светлые иконки.
  const dark = (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255 < 0.5
  shell.setSystemBars({ color: hex, dark }).catch(() => {})
}

/* getComputedStyle отдаёт цвет В ФОРМАТЕ ЕГО ОПИСАНИЯ: токены темы — это
   oklch(), т.е. строка вида "oklch(0.99 0.003 260)", а не rgb(). Прежний
   парсер выдирал из неё числа как каналы 0..255 и собирал мусорный hex —
   нативный плагин отвергал его, и панели вовсе не следовали теме. Канва
   конвертирует ЛЮБОЙ понятный браузеру цвет в честный sRGB-пиксель. */
let colorCtx = null
function cssColorToRgba(color) {
  if (!color) return null
  if (!colorCtx) {
    const canvas = document.createElement('canvas')
    canvas.width = canvas.height = 1
    colorCtx = canvas.getContext('2d', { willReadFrequently: true })
    if (!colorCtx) return null
  }
  colorCtx.clearRect(0, 0, 1, 1)
  colorCtx.fillStyle = '#000' // сброс: невалидный color оставил бы прошлый fillStyle
  colorCtx.fillStyle = color
  colorCtx.fillRect(0, 0, 1, 1)
  return Array.from(colorCtx.getImageData(0, 0, 1, 1).data)
}

// Номер установленной сборки обёртки (ГГММДДН) — для «О приложении».
export async function getNativeBuild() {
  const shell = nativeShell()
  if (!isNativeApp() || !shell) return null
  try {
    const { build } = await shell.getInfo()
    return build
  } catch {
    return null
  }
}

// Принудительная проверка обновления обёртки (без 6-часового троттла
// автопроверки): {current, latest, updateAvailable}.
export async function checkNativeUpdate() {
  const shell = nativeShell()
  if (!isNativeApp() || !shell) throw new Error('Недоступно вне приложения')
  return shell.checkUpdate()
}

// Скачивание и установка обновления. onProgress(0..1 | -1) — ход загрузки.
// Ответ {status: 'installing' | 'needs_permission'} — во втором случае
// система открыла настройки «установка из неизвестных источников»,
// пользователю нужно разрешить и повторить.
export async function installNativeUpdate(onProgress) {
  const shell = nativeShell()
  if (!isNativeApp() || !shell) throw new Error('Недоступно вне приложения')
  let sub = null
  if (onProgress) {
    sub = await shell.addListener('updateProgress', ({ progress }) => onProgress(progress))
  }
  try {
    return await shell.installUpdate()
  } finally {
    sub?.remove?.()
  }
}
