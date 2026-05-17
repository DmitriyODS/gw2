import { useAuthStore } from '@/stores/auth'

let isRefreshing = false
let refreshQueue = []

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
    throw err
  }

  if (resp.status === 204) return null
  return resp.json()
}
