import { defineStore } from 'pinia'
import { useAuthStore } from './auth.js'

export const useNotificationsStore = defineStore('notifications', () => {
  let _toast = null

  function setToast(toastInstance) {
    _toast = toastInstance
  }

  function notify({ severity = 'info', summary = '', detail = '', life = 4000 }) {
    // При выходе/без активной сессии хвостовые запросы авторизованных экранов
    // отваливаются по 401 — это ожидаемо, поэтому не сыпем тостами ошибок
    // («Ошибка загрузки статистики» и т.п.). Сигнал надёжен: после clearAuth
    // токен null до следующего входа, так что под глушилку попадают и запросы,
    // отвалившиеся уже после редиректа на /login. Неавторизованные флоу (логин,
    // гостевой вход в звонок) тосты не используют — их это не затрагивает.
    if (severity === 'error') {
      const auth = useAuthStore()
      if (auth.loggingOut || !auth.token) return
    }
    _toast?.add({ severity, summary, detail, life })
  }

  function success(detail, summary = 'Успешно') {
    notify({ severity: 'success', summary, detail })
  }

  function error(detail, summary = 'Ошибка') {
    notify({ severity: 'error', summary, detail, life: 6000 })
  }

  function warn(detail, summary = 'Внимание') {
    notify({ severity: 'warn', summary, detail })
  }

  return { setToast, notify, success, error, warn }
})
