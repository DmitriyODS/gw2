import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ROLES } from '@/composables/usePermission.js'
import { useCompanySettings } from '@/composables/useCompanySettings.js'
import { navProgress } from '@/composables/useNavProgress.js'

const routes = [
  { path: '/login', component: () => import('@/views/LoginView.vue'), meta: { public: true } },
  { path: '/register', component: () => import('@/views/RegisterView.vue'), meta: { public: true } },
  // Подтверждение email (ввод кода или переход по ссылке ?token=…) — публичный.
  { path: '/verify-email', component: () => import('@/views/VerifyEmailView.vue'), meta: { public: true } },
  // Восстановление пароля — публичные экраны.
  { path: '/forgot-password', component: () => import('@/views/ForgotPasswordView.vue'), meta: { public: true } },
  { path: '/reset-password', component: () => import('@/views/ResetPasswordView.vue'), meta: { public: true } },
  // Принятие email-приглашения в компанию (нужна авторизация; гость сперва войдёт).
  { path: '/invite/:token', component: () => import('@/views/InviteAcceptView.vue'),
    meta: { requiresAuth: true, fullscreen: true }, props: true },
  // Компанийный контент (requiresCompany) — нужна активная компания (roleLevel>0)
  // и НЕ платформенный супер-админ (у него активной компании нет).
  { path: '/tasks', component: () => import('@/views/TasksView.vue'),
    meta: { requiresAuth: true, requiresCompany: true } },
  // Canonical-ссылка на конкретную задачу. Открывает тот же TasksView и сам
  // разворачивает модалку задачи (логика — в TasksView через route.params.id).
  { path: '/tasks/:id(\\d+)', component: () => import('@/views/TasksView.vue'),
    meta: { requiresAuth: true, requiresCompany: true }, props: true },
  { path: '/stats', component: () => import('@/views/StatsView.vue'),
    meta: { requiresAuth: true, requiresCompany: true } },
  { path: '/settings', component: () => import('@/views/SettingsView.vue'), meta: { requiresAuth: true } },
  { path: '/profile', component: () => import('@/views/ProfileView.vue'), meta: { requiresAuth: true } },
  { path: '/employees', component: () => import('@/views/EmployeesView.vue'),
    meta: { requiresAuth: true, requiresCompany: true } },
  { path: '/registries', component: () => import('@/views/RegistriesView.vue'),
    meta: { requiresAuth: true, requiresCompany: true } },
  { path: '/calendars', component: () => import('@/views/CalendarView.vue'),
    meta: { requiresAuth: true, requiresCompany: true } },
  // Ежедневник — личный (кросс-компанийный): нужна только авторизация, активная
  // компания не требуется.
  { path: '/diaries', component: () => import('@/views/DiaryView.vue'),
    meta: { requiresAuth: true } },
  // Раздел «Компании»: супер-админ видит все (платформа), обычный пользователь —
  // те, что создал/администрирует (доступ к данным проверяет бэкенд).
  { path: '/companies', component: () => import('@/views/CompaniesView.vue'),
    meta: { requiresAuth: true } },
  { path: '/users', component: () => import('@/views/UsersView.vue'),
    meta: { requiresAuth: true, requiresSuperAdmin: true } },
  { path: '/companies/:id(\\d+)', component: () => import('@/views/CompanyManageView.vue'),
    meta: { requiresAuth: true }, props: true },
  {
    path: '/messenger/:conversationId(\\d+)?',
    component: () => import('@/views/MessengerView.vue'),
    meta: { requiresAuth: true },
    props: true,
  },
  { path: '/groove', component: () => import('@/views/GrooveView.vue'),
    meta: { requiresAuth: true, requiresCompany: true, feature: 'uses_groove' } },
  { path: '/tv', component: () => import('@/views/TvView.vue'), meta: { requiresAuth: true, fullscreen: true } },
  // Ссылка-приглашение в звонок: доступна и внешним гостям без аккаунта.
  { path: '/call/:code', component: () => import('@/views/CallJoinView.vue'),
    meta: { public: true, fullscreen: true } },
  // Публичный просмотр реестра по внешней ссылке (read-only, без авторизации).
  { path: '/registry/:code', component: () => import('@/views/SharedRegistryView.vue'),
    meta: { public: true } },
  // Публичный просмотр календаря по внешней ссылке (read-only, без авторизации).
  { path: '/calendar/:code', component: () => import('@/views/SharedCalendarView.vue'),
    meta: { public: true } },
  // Публичный просмотр ежедневника по внешней ссылке (read-only, без авторизации).
  { path: '/diary/:code', component: () => import('@/views/SharedDiaryView.vue'),
    meta: { public: true } },
  // Вступление в компанию по ссылке-приглашению (нужна авторизация).
  { path: '/join/:code', component: () => import('@/views/JoinView.vue'),
    meta: { requiresAuth: true, fullscreen: true } },
  { path: '/', redirect: '/tasks' },
  { path: '/:pathMatch(.*)*', redirect: '/tasks' }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// Куда отправить авторизованного пользователя без доступа к запрошенному
// компанийному разделу: супер-админа — на платформенный экран компаний;
// члена компании — на задачи; пользователя без активной компании — в мессенджер
// (доступен всегда; оттуда он создаёт/выбирает компанию).
function landingFor(auth) {
  if (auth.isSuperAdmin) return '/companies'
  if (auth.roleLevel > 0) return '/tasks'
  return '/messenger'
}

router.beforeEach(async (to) => {
  navProgress.value = true
  const auth = useAuthStore()
  await auth.ensureReady()
  if (!to.meta.public && !auth.token) {
    // Сохраняем цель (например, ссылку-приглашение /join/...), чтобы вернуться
    // на неё после входа.
    const redirect = to.fullPath !== '/' ? { redirect: to.fullPath } : {}
    return { path: '/login', query: redirect }
  }
  if ((to.path === '/login' || to.path === '/register') && auth.token) {
    return landingFor(auth)
  }
  if (!auth.token) return
  // Платформенный раздел — только супер-админ.
  if (to.meta.requiresSuperAdmin && !auth.isSuperAdmin) {
    return landingFor(auth)
  }
  // Компанийный контент — нужна активная компания и НЕ супер-админ.
  if (to.meta.requiresCompany && (auth.isSuperAdmin || auth.roleLevel <= 0)) {
    return landingFor(auth)
  }
  // Проверка минимальной роли — по роли в АКТИВНОЙ компании (claims), а не из /me.
  if (to.meta.minRole) {
    if (auth.roleLevel < to.meta.minRole) return landingFor(auth)
  }
  // Раздел выключен настройкой компании (например uses_groove === false).
  if (to.meta.feature) {
    const { settings } = useCompanySettings()
    if (settings.value[to.meta.feature] === false) return landingFor(auth)
  }
})

router.afterEach(() => { navProgress.value = false })
router.onError(() => { navProgress.value = false })

export default router
