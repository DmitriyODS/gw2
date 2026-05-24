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
      <ChangelogModal v-if="isChangelogOpen" @close="closeChangelog" />
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
import { useToast } from 'primevue/usetoast'
import { useAuthStore } from '@/stores/auth.js'
import { useThemeStore } from '@/stores/theme.js'
import { useUnitsStore } from '@/stores/units.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useTutorial } from '@/composables/useTutorial.js'
import { useChangelog } from '@/composables/useChangelog.js'
import { connectSocket } from '@/socket/index.js'
import AppSidebar from '@/components/layout/AppSidebar.vue'
import AppBottomNav from '@/components/layout/AppBottomNav.vue'
import ActiveUnitModal from '@/components/layout/ActiveUnitModal.vue'
import AppTutorial from '@/components/layout/AppTutorial.vue'
import ChangelogModal from '@/components/layout/ChangelogModal.vue'
import Toast from 'primevue/toast'
import ProgressSpinner from 'primevue/progressspinner'

const authStore = useAuthStore()
const themeStore = useThemeStore()
const unitsStore = useUnitsStore()
const notif = useNotificationsStore()
// isOpen деструктурирован как топ-левел ref — Vue auto-unwraps в шаблоне
const { isOpen: isTutorialOpen, open: openTutorial, shouldAutoShow } = useTutorial()
const { isOpen: isChangelogOpen, close: closeChangelog, checkForNewVersion } = useChangelog()

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
  // Восстановление сессии централизовано в auth-store и уже инициируется
  // router guard'ом — здесь лишь дожидаемся его и поднимаем сокет/юнит.
  await authStore.ensureReady()
  if (authStore.token) {
    connectSocket()
    await unitsStore.fetchActiveUnit()
    // Лог версий показываем существующим пользователям; новичкам сначала тур,
    // а лог всплывёт при следующем входе.
    if (!shouldAutoShow()) {
      checkForNewVersion()
    }
  }
  initializing.value = false
})
</script>

<style>
.app-loading {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 100vh;
  background: var(--gw-bg);
}
</style>
