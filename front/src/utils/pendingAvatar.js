import { storageGet, storageSet, storageRemove } from '@/utils/storage.js'
import { uploadAvatar } from '@/api/users.js'

/**
 * Аватарка, выбранная НА РЕГИСТРАЦИИ.
 *
 * Регистрация сессию не выдаёт (сначала подтверждение почты), а загрузка
 * аватарки требует токена. Поэтому обрезанный снимок ждёт своего часа в
 * localStorage и уходит на сервер первым же делом после появления сессии —
 * на экране подтверждения почты.
 */
const KEY = 'gw_pending_avatar'

/** Сохранить обрезанный blob как data-URL (переживает перезагрузку страницы). */
export function savePendingAvatar(dataUrl) {
  if (!dataUrl) return
  storageSet(KEY, dataUrl)
}

export function getPendingAvatar() {
  return storageGet(KEY, '')
}

export function clearPendingAvatar() {
  storageRemove(KEY)
}

/** data-URL → File без fetch(): заводской WebView старых Android его на data: не умеет. */
function dataUrlToFile(dataUrl, name) {
  const comma = dataUrl.indexOf(',')
  if (comma < 0) return null
  const meta = dataUrl.slice(0, comma)
  const type = /data:([^;]+)/.exec(meta)?.[1] || 'image/jpeg'
  const binary = atob(dataUrl.slice(comma + 1))
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i)
  return new File([bytes], name, { type })
}

/**
 * Отправить отложенную аватарку, если она есть. Ошибка не критична: аккаунт
 * уже создан, фото пользователь всегда поставит в настройках.
 */
export async function flushPendingAvatar() {
  const dataUrl = getPendingAvatar()
  if (!dataUrl) return false
  clearPendingAvatar()
  try {
    const file = dataUrlToFile(dataUrl, 'avatar.jpg')
    if (!file) return false
    await uploadAvatar(file)
    return true
  } catch {
    return false
  }
}
