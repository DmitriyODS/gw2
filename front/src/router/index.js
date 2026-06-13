import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ROLES } from '@/composables/usePermission.js'
import { useCompanySettings } from '@/composables/useCompanySettings.js'
import { navProgress } from '@/composables/useNavProgress.js'

const routes = [
  { path: '/login', component: () => import('@/views/LoginView.vue'), meta: { public: true } },
  { path: '/tasks', component: () => import('@/views/TasksView.vue'), meta: { requiresAuth: true } },
  // Canonical-ссылка на конкретную задачу. Открывает тот же TasksView и сам
  // разворачивает модалку задачи (логика — в TasksView через route.params.id).
  { path: '/tasks/:id(\\d+)', component: () => import('@/views/TasksView.vue'),
    meta: { requiresAuth: true }, props: true },
  { path: '/stats', component: () => import('@/views/StatsView.vue'), meta: { requiresAuth: true } },
  { path: '/settings', component: () => import('@/views/SettingsView.vue'), meta: { requiresAuth: true } },
  { path: '/profile', component: () => import('@/views/ProfileView.vue'), meta: { requiresAuth: true } },
  { path: '/employees', component: () => import('@/views/EmployeesView.vue'), meta: { requiresAuth: true } },
  { path: '/companies', component: () => import('@/views/CompaniesView.vue'),
    meta: { requiresAuth: true, minRole: ROLES.ADMIN } },
  { path: '/lists', component: () => import('@/views/ListsView.vue'),
    meta: { requiresAuth: true, minRole: ROLES.DIRECTOR } },
  {
    path: '/messenger/:conversationId(\\d+)?',
    component: () => import('@/views/MessengerView.vue'),
    meta: { requiresAuth: true },
    props: true,
  },
  { path: '/groove', component: () => import('@/views/GrooveView.vue'),
    meta: { requiresAuth: true, feature: 'uses_groove' } },
  { path: '/tv', component: () => import('@/views/TvView.vue'), meta: { requiresAuth: true, fullscreen: true } },
  // Ссылка-приглашение в звонок: доступна и внешним гостям без аккаунта.
  { path: '/call/:code', component: () => import('@/views/CallJoinView.vue'),
    meta: { public: true, fullscreen: true } },
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
  if (to.path === '/login' && auth.token) {
    return '/tasks'
  }
  // Проверка минимальной роли — по роли в АКТИВНОЙ компании (claims), а не из /me.
  if (to.meta.minRole) {
    const level = auth.roleLevel || auth.user?.role?.level || 0
    if (level < to.meta.minRole) return '/tasks'
  }
  // Раздел выключен настройкой компании (например uses_groove === false).
  if (to.meta.feature) {
    const { settings } = useCompanySettings()
    if (settings.value[to.meta.feature] === false) return '/tasks'
  }
})

router.afterEach(() => { navProgress.value = false })
router.onError(() => { navProgress.value = false })

export default router
