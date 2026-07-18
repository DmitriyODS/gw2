import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { getMe } from '@/api/users.js'
import { login as apiLogin, logout as apiLogout, changeDefault as apiChangeDefault, refreshToken,
  register as apiRegister, selectCompany as apiSelectCompany, switchCompany as apiSwitchCompany,
  verifyEmail as apiVerifyEmail, resendVerification as apiResendVerification,
  forgotPassword as apiForgotPassword, resetPassword as apiResetPassword } from '@/api/auth.js'
import { joinCompanyByCode as apiJoinCompany, acceptCompanyInvite as apiAcceptInvite } from '@/api/companies.js'
import router from '@/router/index.js'
import { disconnectSocket, updateSocketAuth } from '@/socket/index.js'
// call.js статически импортирует auth.js — цикл безопасен: useCallStore
// вызывается только внутри функций (в рантайме, после инициализации сторов),
// а call.js и так эагерно грузится в App.vue, поэтому вес чанка не растёт.
import { useCallStore } from './call.js'

export const useAuthStore = defineStore('auth', () => {
  const user = ref(null)
  const token = ref(null)
  // Клеймы сессии (company_id/company_name/company_settings/role_level/
  // is_super_admin/force_change). PASETO-токен на клиенте не декодируется —
  // authsvc дублирует клеймы в теле ответов login/refresh/change-default.
  // role_level может быть 0, а company_id — null: пользователь развязан с
  // компаниями и может не иметь активной (это нормальное состояние).
  const claims = ref({})
  // Компании, в которых состоит пользователь, с ролью в каждой (из тела
  // login/select/switch/refresh). Для многокомпанийных — переключатель и пикер.
  const companies = ref([])
  const forceChange = ref(false)
  const ready = ref(false)
  // Старт без сети: сервер недоступен и статус сессии ещё неизвестен —
  // App.vue показывает «подключаемся», а не экран входа.
  const connecting = ref(false)
  // Идёт намеренный выход: client.js на это время глушит «Сессия истекла»
  // от хвостовых запросов, чтобы logout был тихим.
  const loggingOut = ref(false)
  // Сообщение от backend о блокировке компании. Если не null — глобальный
  // обработчик показывает экран блокировки вместо обычного приложения.
  const companyDisabled = ref(null)
  let _restorePromise = null

  watch(token, (t) => {
    updateSocketAuth(t)
  })

  const isAuth = computed(() => !!token.value)

  // Id текущего пользователя (из тела сессии); до /me доступен из claims.
  const userId = computed(() => claims.value.user_id ?? user.value?.id ?? null)
  const companyId = computed(() => claims.value.company_id ?? null)
  const companyName = computed(() => claims.value.company_name ?? null)
  const companySettings = computed(() => claims.value.company_settings ?? null)
  // Платформенный супер-админ — отдельный флаг (НЕ роль компании).
  const isSuperAdmin = computed(() => !!claims.value.is_super_admin)
  // Роль в активной компании; 0 — нет активной компании (норма).
  const roleLevel = computed(() => claims.value.role_level ?? 0)
  // Многокомпанийный обычный пользователь — может переключать активную компанию.
  const isMultiCompany = computed(() => !isSuperAdmin.value && companies.value.length > 1)

  // Применить ответ login/select/switch/refresh/change-default: токен + клеймы
  // сессии + список компаний пользователя.
  function applySession(data) {
    loggingOut.value = false
    token.value = data.access_token
    claims.value = {
      user_id: data.user_id ?? null,
      company_id: data.company_id ?? null,
      company_name: data.company_name ?? null,
      company_settings: data.company_settings ?? null,
      role_level: data.role_level ?? 0,
      is_super_admin: !!data.is_super_admin,
    }
    companies.value = data.companies ?? []
    forceChange.value = !!data.force_change
  }

  // Локально подмешать изменённые настройки активной компании в клеймы сессии,
  // не дожидаясь рефреша токена (15 мин). Нужно, когда Руководитель сам меняет
  // настройку своей компании (например выключает «Мой Groove») — иначе меню и
  // гард раздела отстают от факта до следующего refresh.
  function patchCompanySettings(patch) {
    claims.value = {
      ...claims.value,
      company_settings: { ...(claims.value.company_settings || {}), ...patch },
    }
  }

  // Активная компания запоминается в браузере — на следующем логине пикер
  // пред-выбирает её (внутри сессии активную компанию несёт refresh-cookie).
  function rememberCompany(companyId) {
    try { localStorage.setItem('gw_active_company_id', String(companyId)) } catch { /* ignore */ }
  }

  async function login(loginVal, password) {
    try {
      const data = await apiLogin({ login: loginVal, password })
      // Несколько компаний — сначала выбор: сессию не применяем, отдаём список
      // и select-токен наверх (LoginView показывает пикер → selectCompany).
      if (data.needs_company_selection) {
        return { needsSelection: true, companies: data.companies ?? [], selectToken: data.select_token }
      }
      applySession(data)
      companyDisabled.value = null
      // Профиль грузим в фоне — вход и редирект не ждут /users/me: иначе на
      // медленном канале форма логина висит секунды поверх готового приложения
      // (шелл рендерится по token, а user подтягивается следом).
      if (!forceChange.value) {
        loadMe().catch(() => {})
      }
      return { forceChange: forceChange.value }
    } catch (e) {
      // 403 COMPANY_DISABLED — бэк сообщил, что компания отключена.
      // client.js уже выставил флаг — добиваем здесь на случай прямого fetch.
      if (e?.error === 'COMPANY_DISABLED') {
        companyDisabled.value = e?.company_name || true
      }
      throw e
    }
  }

  // Регистрация нового пользователя (без компании). Сессию НЕ выдаёт — сначала
  // подтверждение email: возвращает {verificationRequired, email}, фронт ведёт
  // на экран ввода кода.
  async function register(payload) {
    const data = await apiRegister(payload)
    return { verificationRequired: true, email: data.email }
  }

  // Подтверждение email (по коду {email, code} или ссылке {token}). При успехе
  // выдаётся сессия (как login) — пользователь входит в систему. У новичка нет
  // активной компании (company_id=null): он создаёт компанию или вступает по
  // приглашению.
  async function verifyEmail(payload) {
    const data = await apiVerifyEmail(payload)
    applySession(data)
    companyDisabled.value = null
    if (!forceChange.value) {
      loadMe().catch(() => {})
    }
    return { forceChange: forceChange.value }
  }

  async function resendVerification(email) {
    await apiResendVerification(email)
  }

  // Запрос письма со сбросом пароля (ответ всегда ok — не раскрываем аккаунт).
  async function forgotPassword(email) {
    await apiForgotPassword(email)
  }

  // Установка нового пароля по токену. Сессию НЕ выдаёт — фронт ведёт на вход
  // (с префиллом логина). Возвращает {login}.
  async function resetPassword(token, newPassword) {
    return await apiResetPassword(token, newPassword)
  }

  // Принять email-приглашение в компанию: бэкенд добавляет членство с ролью и
  // возвращает сессию, переключённую на компанию.
  async function acceptInvite(token) {
    const data = await apiAcceptInvite(token)
    applySession(data)
    companyDisabled.value = null
    if (data.company_id != null) rememberCompany(data.company_id)
    await loadMe().catch(() => {})
    return data
  }

  // Завершить логин выбором компании (после login с needs_company_selection).
  async function selectCompany(selectToken, companyId) {
    const data = await apiSelectCompany({ select_token: selectToken, company_id: companyId })
    applySession(data)
    companyDisabled.value = null
    rememberCompany(companyId)
    if (!forceChange.value) {
      loadMe().catch(() => {})
    }
    return { forceChange: forceChange.value }
  }

  // Сменить активную компанию в текущей сессии (перевыпуск токенов + перечит.
  // профиля). Сторы данных перезагрузятся по watch на claims.company_id.
  async function switchCompany(targetCompanyId) {
    if (targetCompanyId === claims.value.company_id) return
    const data = await apiSwitchCompany(targetCompanyId)
    applySession(data)
    rememberCompany(targetCompanyId)
    await loadMe().catch(() => {})
  }

  // Вступить в компанию по ссылке-приглашению (авторизованный пользователь):
  // бэкенд добавляет членство и возвращает сессию, переключённую на компанию.
  async function joinCompany(code) {
    const data = await apiJoinCompany(code)
    applySession(data)
    companyDisabled.value = null
    if (data.company_id != null) rememberCompany(data.company_id)
    await loadMe().catch(() => {})
    return data
  }

  // Применить сессию, полученную по QR/коду (LinkClaim). Ведёт себя как login:
  // при нескольких компаниях у пользователя — возвращает {needsSelection} для
  // пикера, иначе применяет сессию. Для ТВ-киоска сессия уже привязана к
  // компании — сразу входим.
  function applyLinkSession(session) {
    if (session?.needs_company_selection) {
      return { needsSelection: true, companies: session.companies ?? [], selectToken: session.select_token }
    }
    applySession(session)
    companyDisabled.value = null
    if (session?.company_id != null) rememberCompany(session.company_id)
    if (!forceChange.value) {
      loadMe().catch(() => {})
    }
    return { forceChange: forceChange.value }
  }

  async function loadMe() {
    const me = await getMe()
    user.value = me
  }

  async function logout() {
    // Глушим «Сессия истекла» от запросов, стартовавших до выхода: до сброса
    // флага хвостовые 401 уходят тихо (см. client.js).
    loggingOut.value = true
    try {
      // Сначала выходим из звонка (если он идёт): иначе после разлогина медиа
      // LiveKit продолжает жить — собеседника видно и слышно.
      try { useCallStore().hangup() } catch {}
      // Мобильная обёртка: снимаем FCM-токен устройства, пока сессия жива —
      // иначе после выхода устройство продолжит получать пуши. Импорт ленивый
      // (nativeApp → api/push → client → auth), в браузере — no-op.
      try {
        const { unregisterNativePush } = await import('@/utils/nativeApp.js')
        await unregisterNativePush()
      } catch {}
      try { await apiLogout() } catch {}
      clearAuth()
      router.push('/login')
    } finally {
      // Хвостовые запросы, стартовавшие с токеном до clearAuth, и так глушатся
      // веткой !auth.token в client.js — флаг можно снимать сразу.
      loggingOut.value = false
    }
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
    try { useCallStore().reset() } catch {}
    disconnectSocket()
    user.value = null
    token.value = null
    claims.value = {}
    companies.value = []
    forceChange.value = false
    companyDisabled.value = null
  }

  // Пауза перед повторной попыткой восстановления: либо браузер сообщил
  // о появлении сети (online), либо просто прошло время (навигатор в
  // Electron/WebView не всегда честен про offline).
  function waitReconnect(ms = 3000) {
    return new Promise((resolve) => {
      const done = () => {
        window.removeEventListener('online', done)
        clearTimeout(timer)
        resolve()
      }
      const timer = setTimeout(done, ms)
      window.addEventListener('online', done, { once: true })
    })
  }

  async function _restore() {
    if (token.value) { ready.value = true; return }
    // «Сервер недоступен» ≠ «не залогинен»: refresh-cookie может быть жива, и
    // показать экран входа было бы враньём (десктоп/мобильная обёртка часто
    // стартуют раньше сети). Ретраим до внятного ответа сервера; на login
    // отправляет только настоящий отказ (4xx от /auth/refresh).
    for (;;) {
      try {
        const data = await refreshToken()
        if (!data.force_change) {
          applySession(data)
          // Профиль — не повод крутить восстановление заново: сессия уже есть,
          // /users/me догрузится штатными ретраями экранов.
          await loadMe().catch(() => {})
        }
        break
      } catch (e) {
        const status = e?.status ?? 0
        if (status === 0 || status >= 500) {
          connecting.value = true
          await waitReconnect()
          continue
        }
        break // валидной refresh-cookie нет — остаёмся неавторизованными
      }
    }
    connecting.value = false
    ready.value = true
  }

  function ensureReady() {
    if (!_restorePromise) _restorePromise = _restore()
    return _restorePromise
  }

  return {
    user, token, forceChange, isAuth, ready, connecting, loggingOut,
    userId, companyId, companyName, companySettings, isSuperAdmin, companyDisabled,
    companies, isMultiCompany, roleLevel,
    ensureReady, login, register, verifyEmail, resendVerification,
    forgotPassword, resetPassword, acceptInvite,
    logout, loadMe, clearAuth, applySession, applyLinkSession, patchCompanySettings,
    selectCompany, switchCompany, joinCompany,
    changeDefaultCredentials,
  }
})
