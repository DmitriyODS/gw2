import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { getMe } from '@/api/users.js'
import { login as apiLogin, logout as apiLogout, changeDefault as apiChangeDefault, refreshToken } from '@/api/auth.js'
import router from '@/router/index.js'
import { disconnectSocket, updateSocketAuth } from '@/socket/index.js'

export const useAuthStore = defineStore('auth', () => {
  const user = ref(null)
  const token = ref(null)
  // Клеймы сессии (company_id/company_name/company_settings/role_level/
  // is_root_admin/force_change). PASETO-токен на клиенте не декодируется —
  // authsvc дублирует клеймы в теле ответов login/refresh/change-default.
  const claims = ref({})
  const forceChange = ref(false)
  const ready = ref(false)
  // Сообщение от backend о блокировке компании. Если не null — глобальный
  // обработчик показывает экран блокировки вместо обычного приложения.
  const companyDisabled = ref(null)
  let _restorePromise = null

  watch(token, (t) => {
    updateSocketAuth(t)
  })

  const isAuth = computed(() => !!token.value)

  const companyId = computed(() => claims.value.company_id ?? null)
  const companyName = computed(() => claims.value.company_name ?? null)
  const companySettings = computed(() => claims.value.company_settings ?? null)
  const isRootAdmin = computed(() => !!claims.value.is_root_admin)

  // Применить ответ login/refresh/change-default: токен + клеймы сессии.
  function applySession(data) {
    token.value = data.access_token
    claims.value = {
      company_id: data.company_id ?? null,
      company_name: data.company_name ?? null,
      company_settings: data.company_settings ?? null,
      role_level: data.role_level ?? 0,
      is_root_admin: !!data.is_root_admin,
    }
    forceChange.value = !!data.force_change
  }

  async function login(loginVal, password) {
    try {
      const data = await apiLogin({ login: loginVal, password })
      applySession(data)
      if (!forceChange.value) {
        await loadMe()
      }
      companyDisabled.value = null
      return forceChange.value
    } catch (e) {
      // 403 COMPANY_DISABLED — бэк сообщил, что компания отключена.
      // client.js уже выставил флаг — добиваем здесь на случай прямого fetch.
      if (e?.error === 'COMPANY_DISABLED') {
        companyDisabled.value = e?.company_name || true
      }
      throw e
    }
  }

  async function loadMe() {
    const me = await getMe()
    user.value = me
  }

  async function logout() {
    // Сначала выходим из звонка (если он идёт): иначе после разлогина медиа
    // LiveKit продолжает жить — собеседника видно и слышно. Импорт ленивый,
    // чтобы не закольцевать стора (call.js импортирует auth.js).
    try {
      const { useCallStore } = await import('./call.js')
      useCallStore().hangup()
    } catch {}
    try { await apiLogout() } catch {}
    clearAuth()
    router.push('/login')
  }

  async function changeDefaultCredentials({ login, password, confirmPassword }) {
    const result = await apiChangeDefault({
      new_login: login,
      new_password: password,
      confirm_password: confirmPassword,
    })
    applySession(result)
    await loadMe()
  }

  function clearAuth() {
    // Сессия закончилась (logout или умерший refresh) — медиа звонка тоже.
    import('./call.js')
      .then(({ useCallStore }) => { try { useCallStore().reset() } catch {} })
      .catch(() => {})
    disconnectSocket()
    user.value = null
    token.value = null
    claims.value = {}
    forceChange.value = false
    companyDisabled.value = null
  }

  async function _restore() {
    if (token.value) { ready.value = true; return }
    try {
      const data = await refreshToken()
      if (!data.force_change) {
        applySession(data)
        await loadMe()
      }
    } catch {
      // Валидной refresh-cookie нет — остаёмся неавторизованными.
    } finally {
      ready.value = true
    }
  }

  function ensureReady() {
    if (!_restorePromise) _restorePromise = _restore()
    return _restorePromise
  }

  return {
    user, token, forceChange, isAuth, ready,
    companyId, companyName, companySettings, isRootAdmin, companyDisabled,
    ensureReady, login, logout, loadMe, clearAuth, applySession,
    changeDefaultCredentials,
  }
})
