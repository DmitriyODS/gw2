import { useAuthStore } from '@/stores/auth'
import { useCompaniesStore } from '@/stores/companies'

let isRefreshing = false
let refreshQueue = []

// Эндпоинты, работающие в рамках конкретной компании. Для Администратора
// системы (без своей company_id) — автоматически добавляем
// ?company_id=<выбранный в селекторе>, чтобы бэк понял scope.
const COMPANY_SCOPED_PREFIXES = [
  '/tasks', '/units', '/departments', '/unit-types',
  '/stats', '/messenger', '/calls', '/users/directory',
]

function _injectCompanyParam(path, companyId) {
  if (companyId == null) return path
  if (!COMPANY_SCOPED_PREFIXES.some(p => path === p || path.startsWith(p + '/') || path.startsWith(p + '?'))) {
    return path
  }
  // Уже передан явно — не перезаписываем.
  if (/[?&]company_id=/.test(path)) return path
  const sep = path.includes('?') ? '&' : '?'
  return `${path}${sep}company_id=${companyId}`
}

function fetchWithTimeout(url, options = {}, ms = 8000) {
  const ctrl = new AbortController()
  const id = setTimeout(() => ctrl.abort(), ms)
  return fetch(url, { ...options, signal: ctrl.signal }).finally(() => clearTimeout(id))
}

async function refreshToken() {
  const resp = await fetchWithTimeout('/api/auth/refresh', { method: 'POST', credentials: 'include' }, 5000)
  if (!resp.ok) throw new Error('refresh_failed')
  const data = await resp.json()
  return data.access_token
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

  // Если пользователь — Администратор системы (нет своей company_id),
  // подмешиваем выбранную в селекторе компанию для всех scope-эндпоинтов.
  if (auth.token && auth.companyId == null) {
    try {
      const companies = useCompaniesStore()
      path = _injectCompanyParam(path, companies.activeCompanyId)
    } catch { /* пиния ещё не готова — пропускаем */ }
  }

  let resp
  try {
    resp = await fetchWithTimeout(`/api${path}`, {
      ...options,
      credentials: 'include',
      headers,
      body: options.body instanceof FormData ? options.body :
            options.body ? JSON.stringify(options.body) : undefined,
    })
  } catch (e) {
    throw { status: 0, error: 'NETWORK_ERROR', message: 'Сервер недоступен' }
  }

  if (resp.status === 401 && !options._retry && path !== '/auth/refresh') {
    if (isRefreshing) {
      return new Promise((resolve, reject) => {
        refreshQueue.push({ resolve, reject, path, options })
      })
    }
    isRefreshing = true
    try {
      const newToken = await refreshToken()
      auth.token = newToken
      isRefreshing = false
      refreshQueue.forEach(({ resolve, reject, path, options }) => {
        apiRequest(path, { ...options, _retry: true }).then(resolve).catch(reject)
      })
      refreshQueue = []
      return apiRequest(path, { ...options, _retry: true })
    } catch {
      isRefreshing = false
      refreshQueue.forEach(({ reject }) => reject(new Error('unauthorized')))
      refreshQueue = []
      auth.clearAuth()
      throw { status: 401, error: 'unauthorized', message: 'Сессия истекла' }
    }
  }

  if (options.blob) return resp

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
