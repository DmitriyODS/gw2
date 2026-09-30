import { useAuthStore } from '@/stores/auth'
import { notifyPlanLimit } from '@/utils/planLimit.js'
import { useNetworkStore } from '@/stores/network.js'

let refreshPromise = null

function anySignal(signals) {
  const list = signals.filter(Boolean)
  if (!list.length) return undefined
  if (list.length === 1) return list[0]
  if (AbortSignal.any) return AbortSignal.any(list)

  const ctrl = new AbortController()
  const abort = () => ctrl.abort()
  for (const signal of list) {
    if (signal.aborted) {
      abort()
      break
    }
    signal.addEventListener('abort', abort, { once: true })
  }
  return ctrl.signal
}

function fetchWithTimeout(url, options = {}, ms = 8000) {
  const ctrl = new AbortController()
  const id = setTimeout(() => ctrl.abort(), ms)
  const signal = anySignal([options.signal, ctrl.signal])
  return fetch(url, { ...options, signal }).finally(() => clearTimeout(id))
}

async function refreshToken() {
  let resp
  try {
    resp = await fetchWithTimeout('/api/auth/refresh', { method: 'POST', credentials: 'include' }, 5000)
  } catch {
    // Сеть/таймаут: сервер НЕ ответил — это не отказ в сессии.
    throw { status: 0, error: 'NETWORK_ERROR' }
  }
  if (!resp.ok) throw { status: resp.status, error: 'refresh_failed' }
  // Тело несёт и токен, и клеймы сессии (PASETO на клиенте не декодируется).
  return resp.json()
}

/**
 * Обновляет access-токен; параллельные вызовы (запросы с 401, сокет с
 * протухшим токеном) ждут ОДИН запрос /auth/refresh — два одновременных
 * обновления одной сессии не нужны серверу и гоняются за cookie.
 * Ошибка сети — `{ status: 0 }`, отказ сервера — его статус.
 */
export function refreshSession() {
  if (!refreshPromise) {
    refreshPromise = refreshToken()
      .then((data) => {
        useAuthStore().applySession(data)
        return data
      })
      .finally(() => { refreshPromise = null })
  }
  return refreshPromise
}

// Тела, которые нельзя сериализовать в JSON: форма, кусок файла, буфер.
function isRawBody(body) {
  return body instanceof FormData || body instanceof Blob || body instanceof ArrayBuffer
    || ArrayBuffer.isView(body)
}

/** Запрос не дошёл до сервера: текст говорит, что проверить. */
function networkError() {
  useNetworkStore().reportRequest(false)
  const offline = typeof navigator !== 'undefined' && navigator.onLine === false
  return {
    status: 0,
    error: 'NETWORK_ERROR',
    offline,
    message: offline
      ? 'Нет подключения к интернету'
      : 'Нет связи с сервером. Проверьте подключение к интернету',
  }
}

export async function apiRequest(path, options = {}) {
  const auth = useAuthStore()

  const headers = { ...options.headers }
  // Сырое тело (кусок файла) уходит как есть: тип задаёт вызывающий, а
  // JSON-сериализация превратила бы Blob в «{}».
  if (!isRawBody(options.body)) {
    headers['Content-Type'] = 'application/json'
  }
  if (auth.token) {
    headers['Authorization'] = `Bearer ${auth.token}`
  }

  // Активная компания берётся из access-токена (claims.company_id) — на клиенте
  // её больше не подмешиваем через ?company_id=. Переключение между компаниями —
  // через switch-company (перевыпуск токена).

  let resp
  try {
    resp = await fetchWithTimeout(`/api${path}`, {
      ...options,
      credentials: 'include',
      headers,
      body: isRawBody(options.body) ? options.body :
            options.body ? JSON.stringify(options.body) : undefined,
    }, options.timeout ?? 8000)
  } catch {
    // Отменил сам вызывающий (пришёл новый запрос, ушли с экрана) — не ошибка.
    if (options.signal?.aborted) {
      throw { status: 0, error: 'ABORTED', message: 'Запрос отменён', silent: true }
    }
    // Всё остальное — сеть: и обрыв, и собственный таймаут (он тоже приходит
    // AbortError'ом, и прежде показывался как «Запрос отменён»).
    throw networkError()
  }
  useNetworkStore().reportRequest(true)

  if (resp.status === 401 && !options._retry && path !== '/auth/refresh') {
    // Намеренный выход или уже нет активной сессии — не дёргаем refresh и не
    // шумим «Сессия истекла»: 401 от запросов, стартовавших до/во время
    // logout, ожидаем. Сам запрос logout (_isLogout) — исключение: ему нужно
    // дойти до сервера (через refresh, если access протух), чтобы погасить
    // refresh-cookie.
    if (!options._isLogout && (auth.loggingOut || !auth.token)) {
      throw { status: 401, error: 'unauthorized', message: '', silent: true }
    }
    try {
      await refreshSession()
    } catch (e) {
      // Refresh не ДОШЁЛ до сервера (обрыв сети/таймаут) — сессия не истекла:
      // не разлогиниваем, отдаём сетевую ошибку; access обновится следующим
      // запросом, когда сеть вернётся.
      if ((e?.status ?? 0) === 0 || e?.status >= 500) {
        throw networkError()
      }
      auth.clearAuth()
      throw { status: 401, error: 'unauthorized', message: 'Сессия истекла' }
    }
    return apiRequest(path, { ...options, _retry: true })
  }

  // Blob-загрузки (экспорт файлов): при не-OK ответе НЕ отдаём тело как файл
  // (иначе сохранится битый/ошибочный файл) — бросаем ошибку с сообщением.
  if (options.blob) {
    if (!resp.ok) {
      let err = { status: resp.status, error: 'unknown', message: 'Ошибка сервера' }
      try { err = { ...err, ...await resp.json() } } catch { /* не JSON */ }
      throw err
    }
    return resp
  }

  if (!resp.ok) {
    let err = { status: resp.status, error: 'unknown', message: 'Ошибка сервера' }
    try { err = { ...err, ...await resp.json() } } catch {}
    // COMPANY_DISABLED — глобальная блокировка: компания пользователя
    // отключена. Поднимаем флаг в auth-store, App.vue показывает экран.
    if (err.error === 'COMPANY_DISABLED') {
      auth.companyDisabled = err.company_name || true
    }
    // LEGAL_CONSENT_REQUIRED — действующая редакция правовых документов не
    // принята: работать нельзя до согласия (152-ФЗ). Сессия жива, поэтому
    // разлогинивать не надо — App.vue показывает плашку поверх приложения.
    // Флаг приходит и в теле сессии; здесь добиваем случай, когда документы
    // обновились в середине сеанса.
    if (err.error === 'LEGAL_CONSENT_REQUIRED') {
      auth.legalRequired = true
    }
    // Лимиты тарифа приходят из любого раздела и означают одно и то же:
    // «нужно в магазин». Показываем это в одном месте, чтобы каждый экран не
    // расписывал апсейл сам (сам запрос всё равно упал — раздел покажет свою
    // ошибку, если ему нужно).
    if (err.status === 402 &&
        (err.error === 'LIMIT_REACHED' || err.error === 'PLAN_FEATURE_REQUIRED' ||
         err.error === 'AI_NO_TOKENS')) {
      notifyPlanLimit(err)
    }
    throw err
  }

  if (resp.status === 204) return null
  return resp.json()
}
