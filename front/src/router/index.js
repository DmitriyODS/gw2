import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes = [
  { path: '/login', component: () => import('@/views/LoginView.vue'), meta: { public: true } },
  { path: '/tasks', component: () => import('@/views/TasksView.vue'), meta: { requiresAuth: true } },
  { path: '/stats', component: () => import('@/views/StatsView.vue'), meta: { requiresAuth: true } },
  { path: '/settings', component: () => import('@/views/SettingsView.vue'), meta: { requiresAuth: true } },
  { path: '/profile', component: () => import('@/views/ProfileView.vue'), meta: { requiresAuth: true } },
  { path: '/', redirect: '/tasks' },
  { path: '/:pathMatch(.*)*', redirect: '/tasks' }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const auth = useAuthStore()
  if (!to.meta.public && !auth.token) {
    return next('/login')
  }
  if (to.path === '/login' && auth.token) {
    return next('/tasks')
  }
  next()
})

export default router
