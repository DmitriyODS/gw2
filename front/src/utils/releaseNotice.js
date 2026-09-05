/* Уведомление о новом выпуске.

   Приложение обновляется молча: человек заходит в изменившийся интерфейс и не
   знает, что стало иначе. Поэтому при первом запуске после выката показываем
   карточку и ведём в «Настройки → О приложении» — там перечислено, что именно
   изменилось.

   Виденный выпуск помнит УСТРОЙСТВО (localStorage): обновление — событие
   клиента, а не аккаунта, и на каждом устройстве человек встречает его свой
   первый раз. */

import { pushNotification } from '@/composables/useDesktopNotifications.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useAppVersion } from '@/composables/useAppVersion.js'

const SEEN_KEY = 'gw_seen_release'

// Карточка выпуска живёт в настройках — туда и ведёт нажатие.
export const ABOUT_PATH = '/settings?section=about'

// Выпуск опознаём парой «версия + сборка»: сборка меняется и без версии.
function releaseStamp(release) {
  const version = release?.version
  return version ? `${version}·${release?.build ?? ''}` : ''
}

/**
 * Рассказать об обновлении, если оно случилось с прошлого запуска.
 * Зовёт каркас при старте — один раз на загрузку приложения.
 */
export async function announceRelease() {
  const { load } = useAppVersion()
  /* Спрашиваем сервер, а не кэш: кэш живёт шесть часов, и зашедший сразу
     после выката узнал бы об обновлении только к вечеру. Запрос один на
     запуск приложения — той экономии, ради которой кэш заводился, он не
     отменяет. */
  const release = await load({ force: true })
  const stamp = releaseStamp(release)
  if (!stamp) return

  let seen = null
  try {
    seen = localStorage.getItem(SEEN_KEY)
    localStorage.setItem(SEEN_KEY, stamp)
  } catch { /* приватное окно — обойдёмся без отметки */ }

  // Первый запуск на устройстве обновлением не считается: человек только
  // пришёл, и «что нового» ему не с чем сравнивать.
  if (!seen || seen === stamp) return

  const title = `Обновление ${release.version}`
  const text = release.title || 'Посмотрите, что изменилось'

  pushNotification({
    key: `release-${stamp}`,
    icon: 'auto_awesome',
    tone: 'primary',
    title,
    text,
    path: ABOUT_PATH,
  })
  useNotificationsStore().notify({
    severity: 'info',
    summary: title,
    detail: text,
    life: 12_000,
    path: ABOUT_PATH,
  })
}
