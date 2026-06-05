import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getMe } from '@/api/users.js'
import { login as apiLogin, logout as apiLogout, changeDefault as apiChangeDefault, refreshToken } from '@/api/auth.js'
import router from '@/router/index.js'

export const useAuthStore = defineStore('auth', () => {
  const user = ref(null)
  const token = ref(null)
  const forceChange = ref(false)
  const ready = ref(false)
  // Сообщение от backend о блокировке компании. Если не null — глобальный
  // обработчик показывает экран блокировки вместо обычного приложения.
  const companyDisabled = ref(null)
  let _restorePromise = null

  const isAuth = computed(() => !!token.value)

  function decodeToken(t) {
    try {
      const payload = t.split('.')[1]
      return JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')))
    } catch { return {} }
  }

  // Доп. клеймы из JWT — companyId/companyName/isRootAdmin/roleLevel.
  const tokenClaims = computed(() => token.value ? decodeToken(token.value) : {})
  const companyId = computed(() => tokenClaims.value.company_id ?? null)
  const companyName = computed(() => tokenClaims.value.company_name ?? null)
  const companySettings = computed(() => tokenClaims.value.company_settings ?? null)
  const isRootAdmin = computed(() => !!tokenClaims.value.is_root_admin)

  async function login(loginVal, password) {
    try {
      const data = await apiLogin({ login: loginVal, password })
      token.value = data.access_token
      const payload = decodeToken(data.access_token)
      forceChange.value = !!payload.force_change
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
    token.value = result.access_token
    forceChange.value = false
    await loadMe()
  }

  function clearAuth() {
    user.value = null
    token.value = null
    forceChange.value = false
    companyDisabled.value = null
  }

  function setToken(t) {
    token.value = t
    const payload = decodeToken(t)
    forceChange.value = !!payload.force_change
  }

  async function _restore() {
    if (token.value) { ready.value = true; return }
    try {
      const data = await refreshToken()
      const payload = decodeToken(data.access_token)
      if (!payload.force_change) {
        setToken(data.access_token)
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
    ensureReady, login, logout, loadMe, clearAuth, setToken,
    changeDefaultCredentials,
  }
})
