/* Автообновление клиента: PWA и долго открытые вкладки не должны залипать
   на старой сборке.

   Два контура:
   1. Service worker — периодический registration.update(): браузер сам
      проверяет sw.js только при навигациях, а установленная PWA живёт без
      них днями.
   2. Версия продукта — лёгкий поллинг /api/changelog (versions[0].version,
      статика nginx с Cache-Control: no-cache). Базовая версия фиксируется
      первым успешным ответом сессии; при её смене — уведомление и мягкая
      перезагрузка, когда это безопасно (вкладка ушла в фон). Перезагрузка —
      максимум одна за сессию, а после неё базовой становится уже новая
      версия, так что цикл перезагрузок невозможен. */

import { changelogApi } from '@/api/changelog.js'

const SW_UPDATE_MS = 60 * 60 * 1000 // проверка новой версии SW — раз в час
const VERSION_POLL_MS = 15 * 60 * 1000 // поллинг версии продукта
const MIN_GAP_MS = 5 * 60 * 1000 // троттлинг проверок по возврату вкладки

let installed = false

export function installAppUpdateWatcher({ onUpdateAvailable, canReload } = {}) {
  if (installed || typeof window === 'undefined') return
  installed = true

  let baseline = null // версия на момент старта сессии
  let updateAvailable = false
  let reloaded = false
  let lastCheckAt = 0

  async function updateServiceWorker() {
    if (!('serviceWorker' in navigator)) return
    try {
      const reg = await navigator.serviceWorker.getRegistration()
      await reg?.update()
    } catch {} // офлайн/приватный режим — молча, попробуем в следующий раз
  }

  function reloadIfSafe() {
    if (!updateAvailable || reloaded) return
    if (canReload && !canReload()) return
    reloaded = true
    window.location.reload()
  }

  async function checkVersion() {
    lastCheckAt = Date.now()
    let latest = null
    try {
      const data = await changelogApi.get()
      latest = data?.versions?.[0]?.version ?? null
    } catch {
      return // сеть недоступна — проверим в следующий раз
    }
    if (!latest) return
    if (!baseline) {
      baseline = latest
      return
    }
    if (latest === baseline || updateAvailable) return

    updateAvailable = true
    // Новая сборка уже на сервере — даём свежему SW установиться заранее.
    updateServiceWorker()
    onUpdateAvailable?.(latest)
    // Вкладка уже в фоне — обновляемся сразу, пользователь ничего не заметит.
    if (document.hidden) reloadIfSafe()
  }

  checkVersion()
  setInterval(checkVersion, VERSION_POLL_MS)
  setInterval(updateServiceWorker, SW_UPDATE_MS)

  document.addEventListener('visibilitychange', () => {
    if (document.hidden) {
      reloadIfSafe()
    } else if (Date.now() - lastCheckAt > MIN_GAP_MS) {
      checkVersion()
      updateServiceWorker()
    }
  })
}
