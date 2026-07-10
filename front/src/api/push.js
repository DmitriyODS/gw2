// Контракт pushsvc (back-go/push) — ведётся вручную, как и остальные.
// Регистрация FCM-токена устройства: вызывается только внутри нативной
// мобильной обёртки (mobile/, Capacitor) — см. utils/nativeApp.js.
import { apiRequest } from './client.js'

export const registerPushToken = (token, platform = 'android') =>
  apiRequest('/push/register', { method: 'POST', body: { token, platform } })

export const unregisterPushToken = (token) =>
  apiRequest('/push/unregister', { method: 'POST', body: { token } })
