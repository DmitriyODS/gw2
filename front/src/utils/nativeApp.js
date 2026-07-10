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
