import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getMe } from '@/api/users.js'
import { login as apiLogin, logout as apiLogout, changeDefault as apiChangeDefault } from '@/api/auth.js'
import router from '@/router/index.js'

export const useAuthStore = defineStore('auth', () => {
  const user = ref(null)
  const token = ref(null)
  const forceChange = ref(false)

  const isAuth = computed(() => !!token.value)

  function decodeToken(t) {
    try {
      const payload = t.split('.')[1]
      return JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')))
    } catch { return {} }
  }

  async function login(loginVal, password) {
    const data = await apiLogin({ login: loginVal, password })
    token.value = data.access_token
    const payload = decodeToken(data.access_token)
    forceChange.value = !!payload.force_change
    if (!forceChange.value) {
      await loadMe()
    }
    return forceChange.value
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
  }

  function setToken(t) {
    token.value = t
    const payload = decodeToken(t)
    forceChange.value = !!payload.force_change
  }

  return { user, token, forceChange, isAuth, login, logout, loadMe, clearAuth, setToken, changeDefaultCredentials }
})
