import { useAuthStore } from '@/stores/auth'

let isRefreshing = false
let refreshQueue = []

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

export async function apiRequest(path, options = {}) {
  const auth = useAuthStore()

  const headers = { ...options.headers }
  if (!(options.body instanceof FormData)) {
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
      body: options.body instanceof FormData ? options.body :
            options.body ? JSON.stringify(options.body) : undefined,
    }, options.timeout ?? 8000)
  } catch (e) {
    if (e?.name === 'AbortError' || options.signal?.aborted) {
      throw { status: 0, error: 'ABORTED', message: 'Запрос отменён' }
    }
    throw { status: 0, error: 'NETWORK_ERROR', message: 'Сервер недоступен' }
  }

  if (resp.status === 401 && !options._retry && path !== '/auth/refresh') {
    // Намеренный выход или уже нет активной сессии — не дёргаем refresh и не
    // шумим «Сессия истекла»: 401 от запросов, стартовавших до/во время
    // logout, ожидаем. Сам запрос logout (_isLogout) — исключение: ему нужно
    // дойти до сервера (через refresh, если access протух), чтобы погасить
    // refresh-cookie.
    if (!options._isLogout && (auth.loggingOut || !auth.token)) {
      throw { status: 401, error: 'unauthorized', message: '', silent: true }
    }
    if (isRefreshing) {
      return new Promise((resolve, reject) => {
        refreshQueue.push({ resolve, reject, path, options })
      })
    }
    isRefreshing = true
    try {
      const data = await refreshToken()
      auth.applySession(data)
      isRefreshing = false
      refreshQueue.forEach(({ resolve, reject, path, options }) => {
        apiRequest(path, { ...options, _retry: true }).then(resolve).catch(reject)
      })
      refreshQueue = []
      return apiRequest(path, { ...options, _retry: true })
    } catch (e) {
      isRefreshing = false
      refreshQueue.forEach(({ reject }) => reject(new Error('unauthorized')))
      refreshQueue = []
      // Refresh не ДОШЁЛ до сервера (обрыв сети/таймаут) — сессия не истекла:
      // не разлогиниваем, отдаём сетевую ошибку; access обновится следующим
      // запросом, когда сеть вернётся.
      if ((e?.status ?? 0) === 0 || e?.status >= 500) {
        throw { status: 0, error: 'NETWORK_ERROR', message: 'Сервер недоступен' }
      }
      auth.clearAuth()
      throw { status: 401, error: 'unauthorized', message: 'Сессия истекла' }
    }
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
    throw err
  }

  if (resp.status === 204) return null
  return resp.json()
}
