import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useAuthStore } from './auth.js'
import { playNotifySound } from '@/utils/systemNotify.js'
import { isSourceEnabled, notifyPrefs } from '@/utils/notifySettings.js'

/* Всплывающие уведомления приложения. Показывает их `components/common/
   AppToasts.vue` — стопкой в углу, с паузой отсчёта под курсором; стор держит
   только список. Больше MAX карточек на экране бессмысленно: нижние всё равно
   не прочитать, поэтому самые давние вытесняются новыми. */
const MAX = 5

export const useNotificationsStore = defineStore('notifications', () => {
  const toasts = ref([])
  let seq = 0

  /* sound — голос уведомления (по умолчанию по severity); `false` для событий
     со своей мелодией (перевод кудосов), иначе прозвучали бы оба.
     source — раздел-источник СОБЫТИЯ (мессенджер, напоминания…): от него
     зависит, ждёт ли человек таких уведомлений вообще («Настройки →
     Уведомления»). Ответы на собственные действия источника не имеют и
     приходят всегда. life — по умолчанию из настроек. */
  function notify({ severity = 'info', summary = '', detail = '', life, sound, source }) {
    if (!isSourceEnabled(source)) return
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
    // Новое — сверху стопки; лишнее уходит с её хвоста.
    toasts.value.unshift({ id: ++seq, severity, summary, detail, life: life ?? notifyPrefs.life })
    if (toasts.value.length > MAX) toasts.value.splice(MAX)
    if (sound !== false) playNotifySound(sound || severity)
  }

  function success(detail, summary = 'Успешно') {
    notify({ severity: 'success', summary, detail })
  }

  // Ошибке даём чуть больше времени: её текст длиннее и важнее остальных.
  function error(detail, summary = 'Ошибка') {
    notify({ severity: 'error', summary, detail, life: notifyPrefs.life + 2000 })
  }

  function warn(detail, summary = 'Внимание') {
    notify({ severity: 'warn', summary, detail })
  }

  function dismiss(id) {
    toasts.value = toasts.value.filter((t) => t.id !== id)
  }

  return { toasts, notify, dismiss, success, error, warn }
})
