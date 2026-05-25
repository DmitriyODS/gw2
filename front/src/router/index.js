import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes = [
  { path: '/login', component: () => import('@/views/LoginView.vue'), meta: { public: true } },
  { path: '/tasks', component: () => import('@/views/TasksView.vue'), meta: { requiresAuth: true } },
  { path: '/stats', component: () => import('@/views/StatsView.vue'), meta: { requiresAuth: true } },
  { path: '/settings', component: () => import('@/views/SettingsView.vue'), meta: { requiresAuth: true } },
  { path: '/profile', component: () => import('@/views/ProfileView.vue'), meta: { requiresAuth: true } },
  { path: '/employees', component: () => import('@/views/EmployeesView.vue'), meta: { requiresAuth: true } },
  {
    path: '/messenger/:conversationId(\\d+)?',
    component: () => import('@/views/MessengerView.vue'),
    meta: { requiresAuth: true },
    props: true,
  },
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
  // Ждём завершения восстановления сессии перед любым решением о доступе —
  // иначе первый переход случается с ещё пустым token и кидает на /login.
  await auth.ensureReady()
  if (!to.meta.public && !auth.token) {
    return '/login'
  }
  if (to.path === '/login' && auth.token) {
    return '/tasks'
  }
})

export default router
