<template>
  <nav class="sidebar">
    <div class="sidebar-logo" data-tutorial="logo" @click="openChangelog" title="Что нового">
      <img src="/logo.svg" alt="Groove Work" class="sidebar-logo-img" />
    </div>

    <div class="sidebar-nav">
      <button
        data-tutorial="nav-tasks"
        class="nav-btn"
        :class="{ active: route.path === '/tasks' }"
        @click="router.push('/tasks')"
        title="Задачи"
      >
        <span class="material-symbols-outlined">grid_view</span>
      </button>

      <button
        v-if="isAtLeast(ROLES.EMPLOYEE)"
        data-tutorial="nav-stats"
        class="nav-btn"
        :class="{ active: route.path === '/stats' }"
        @click="router.push('/stats')"
        title="Статистика"
      >
        <span class="material-symbols-outlined">query_stats</span>
      </button>

      <button
        class="nav-btn"
        :class="{ active: route.path === '/employees' }"
        @click="router.push('/employees')"
        title="Сотрудники"
      >
        <span class="material-symbols-outlined">groups</span>
      </button>

      <button
        class="nav-btn"
        :class="{ active: route.path.startsWith('/messenger') }"
        @click="router.push('/messenger')"
        title="Мессенджер"
      >
        <span class="material-symbols-outlined">chat</span>
        <span v-if="messenger.totalUnread" class="nav-badge">
          {{ messenger.totalUnread > 99 ? '99+' : messenger.totalUnread }}
        </span>
      </button>

      <button
        data-tutorial="nav-settings"
        class="nav-btn"
        :class="{ active: route.path === '/settings' }"
        @click="router.push('/settings')"
        title="Настройки"
      >
        <span class="material-symbols-outlined">settings</span>
      </button>
    </div>

    <div class="sidebar-bottom">
      <img
        data-tutorial="profile-avatar"
        class="user-avatar"
        :src="avatarSrc"
        :alt="authStore.user?.fio"
        @click="router.push('/profile')"
        title="Профиль"
      />
    </div>

  </nav>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import { useChangelog } from '@/composables/useChangelog.js'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const messenger = useMessengerStore()
const { isAtLeast } = usePermission()
const { open: openChangelog } = useChangelog()

const avatarSrc = computed(() => {
  const user = authStore.user
  if (!user) return ''
  if (user.avatar_path) return `/uploads/${user.avatar_path}`
  return `/api/users/${user.id}/identicon`
})
</script>

<style scoped>
.sidebar {
  width: 72px;
  height: 100vh;
  background: var(--gw-sidebar-bg);
  border-right: 1px solid var(--gw-border);
  position: sticky;
  top: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px 0;
  flex-shrink: 0;
  z-index: 100;
}

@media (max-width: 768px) {
  .sidebar {
    display: none;
  }
}

.sidebar-logo {
  width: 48px;
  height: 48px;
  margin-bottom: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  border-radius: 12px;
  transition: background 0.15s;
}

.sidebar-logo:hover {
  background: var(--gw-primary-light);
}

.sidebar-logo-img {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: block;
}

.sidebar-nav {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.nav-btn {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  border: none;
  background: transparent;
  color: var(--gw-text-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s, color 0.15s;
  position: relative;
}

.nav-btn:hover {
  background: var(--gw-primary-light);
  color: var(--gw-primary);
}

.nav-btn.active {
  background: var(--gw-primary);
  color: var(--color-on-primary);
}

.nav-btn .material-symbols-outlined {
  font-size: 24px;
}

.nav-badge {
  position: absolute;
  top: 4px;
  right: 4px;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: var(--radius-full);
  background: var(--color-error);
  color: var(--color-on-error);
  font-size: 10px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px solid var(--gw-sidebar-bg, var(--color-surface));
}


.sidebar-bottom {
  margin-top: auto;
  padding-top: 16px;
}

.user-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
  cursor: pointer;
  border: 2px solid var(--gw-border);
  transition: border-color 0.15s;
}

.user-avatar:hover {
  border-color: var(--gw-primary);
}
</style>
