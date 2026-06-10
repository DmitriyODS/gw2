import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { ROLES } from '@/composables/usePermission.js'

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
  { path: '/groove', component: () => import('@/views/GrooveView.vue'), meta: { requiresAuth: true } },
  { path: '/tv', component: () => import('@/views/TvView.vue'), meta: { requiresAuth: true, fullscreen: true } },
  { path: '/', redirect: '/tasks' },
  { path: '/:pathMatch(.*)*', redirect: '/tasks' }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  await auth.ensureReady()
  if (!to.meta.public && !auth.token) {
    return '/login'
  }
  if (to.path === '/login' && auth.token) {
    return '/tasks'
  }
  // Проверка минимальной роли на роутах с meta.minRole.
  if (to.meta.minRole) {
    const level = auth.user?.role?.level ?? 0
    if (level < to.meta.minRole) return '/tasks'
  }
})

export default router
