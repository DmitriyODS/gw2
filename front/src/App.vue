<template>
  <div class="app-layout" :data-dark="themeStore.dark">
    <div v-if="initializing" class="app-loading">
      <ProgressSpinner />
    </div>
    <template v-else-if="authStore.user">
      <AppSidebar />
      <main class="main-content">
        <router-view />
      </main>
      <AppBottomNav />
      <ActiveUnitModal v-if="unitsStore.activeUnit" />
      <AppTutorial v-if="isTutorialOpen" />
    </template>
    <template v-else>
      <main class="main-content">
        <router-view />
      </main>
    </template>
    <Toast position="bottom-right" />
  </div>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useToast } from 'primevue/usetoast'
import { useAuthStore } from '@/stores/auth.js'
import { useThemeStore } from '@/stores/theme.js'
import { useUnitsStore } from '@/stores/units.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useTutorial } from '@/composables/useTutorial.js'
import { connectSocket } from '@/socket/index.js'
import { refreshToken } from '@/api/auth.js'
import AppSidebar from '@/components/layout/AppSidebar.vue'
import AppBottomNav from '@/components/layout/AppBottomNav.vue'
import ActiveUnitModal from '@/components/layout/ActiveUnitModal.vue'
import AppTutorial from '@/components/layout/AppTutorial.vue'
import Toast from 'primevue/toast'
import ProgressSpinner from 'primevue/progressspinner'

const router = useRouter()
const authStore = useAuthStore()
const themeStore = useThemeStore()
const unitsStore = useUnitsStore()
const notif = useNotificationsStore()
// isOpen деструктурирован как топ-левел ref — Vue auto-unwraps в шаблоне
const { isOpen: isTutorialOpen, open: openTutorial, shouldAutoShow } = useTutorial()

watch(() => authStore.user, (user, prev) => {
  if (user && !prev && shouldAutoShow()) {
    setTimeout(() => openTutorial(), 600)
  }
})

// useToast() требует setup-контекст — вызываем здесь, не в onMounted
const toast = useToast()
notif.setToast(toast)

// Применяем тему сразу (без FOUC)
themeStore.init()

const initializing = ref(true)

onMounted(async () => {

  // Если токена нет в памяти — пробуем восстановить сессию через refresh cookie
  if (!authStore.token) {
    try {
      const data = await refreshToken()
      // Если сессия требует смены пароля — не восстанавливаем, пусть логинится заново
      const payload = JSON.parse(atob(data.access_token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')))
      if (payload.force_change) {
        router.push('/login')
      } else {
        authStore.setToken(data.access_token)
        await authStore.loadMe()
        connectSocket()
        await unitsStore.fetchActiveUnit()
        if (router.currentRoute.value.path === '/login') {
          router.push('/tasks')
        }
      }
    } catch {
      // Нет валидного refresh cookie — останемся на /login
      if (!router.currentRoute.value.meta?.public) {
        router.push('/login')
      }
    }
  } else {
    connectSocket()
    await unitsStore.fetchActiveUnit()
  }

  initializing.value = false
})
</script>

<style>
.app-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: var(--gw-bg);
}
</style>
