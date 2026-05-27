<template>
  <nav class="bottom-nav">
    <button
      data-tutorial="nav-tasks"
      class="bottom-nav-item"
      :class="{ active: route.path === '/tasks' }"
      @click="router.push('/tasks')"
    >
      <span class="material-symbols-outlined">grid_view</span>
      <span class="bottom-nav-label">Задачи</span>
      <span v-if="unitsStore.activeUnit" class="unit-dot" />
    </button>

    <button
      v-if="isAtLeast(ROLES.EMPLOYEE)"
      data-tutorial="nav-stats"
      class="bottom-nav-item"
      :class="{ active: route.path === '/stats' }"
      @click="router.push('/stats')"
    >
      <span class="material-symbols-outlined">query_stats</span>
      <span class="bottom-nav-label">Статистика</span>
    </button>

    <button
      data-tutorial="nav-employees"
      class="bottom-nav-item"
      :class="{ active: route.path === '/employees' }"
      @click="router.push('/employees')"
    >
      <span class="material-symbols-outlined">groups</span>
      <span class="bottom-nav-label">Люди</span>
    </button>

    <button
      data-tutorial="nav-messenger"
      class="bottom-nav-item"
      :class="{ active: route.path.startsWith('/messenger') }"
      @click="router.push('/messenger')"
    >
      <span class="material-symbols-outlined">chat</span>
      <span class="bottom-nav-label">Чаты</span>
      <span v-if="messenger.totalUnread" class="bottom-badge">
        {{ messenger.totalUnread > 99 ? '99+' : messenger.totalUnread }}
      </span>
    </button>

    <button
      data-tutorial="nav-settings"
      class="bottom-nav-item"
      :class="{ active: route.path === '/settings' }"
      @click="router.push('/settings')"
    >
      <span class="material-symbols-outlined">settings</span>
      <span class="bottom-nav-label">Настройки</span>
    </button>

    <button
      data-tutorial="profile-avatar"
      class="bottom-nav-item"
      :class="{ active: route.path === '/profile' }"
      @click="router.push('/profile')"
    >
      <img class="bottom-nav-avatar" :src="avatarSrc" :alt="authStore.user?.fio" />
      <span class="bottom-nav-label">Профиль</span>
    </button>
  </nav>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { useUnitsStore } from '@/stores/units.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { usePermission, ROLES } from '@/composables/usePermission.js'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const unitsStore = useUnitsStore()
const messenger = useMessengerStore()
const { isAtLeast } = usePermission()

const avatarSrc = computed(() => {
  const user = authStore.user
  if (!user) return ''
  if (user.avatar_path) return `/uploads/${user.avatar_path}`
  return `/api/users/${user.id}/identicon`
})
</script>

<style scoped>
/* По умолчанию скрыта — показывается только на мобильном */
.bottom-nav {
  display: none;
}

@media (max-width: 768px) {
  .bottom-nav {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    height: 60px;
    background: var(--gw-surface);
    border-top: 1px solid var(--gw-border);
    display: flex;
    align-items: stretch;
    z-index: 200;
    padding-bottom: env(safe-area-inset-bottom, 0);
  }
}

.bottom-nav-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
  border: none;
  background: transparent;
  cursor: pointer;
  color: var(--gw-text-secondary);
  position: relative;
  transition: color 0.15s;
  padding: 6px 4px;
  min-width: 0;
}

.bottom-nav-item:active {
  background: var(--gw-primary-light);
}

.bottom-nav-item.active {
  color: var(--gw-primary);
}

.bottom-nav-item .material-symbols-outlined {
  font-size: 22px;
}

.bottom-nav-label {
  font-size: 10px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.bottom-nav-avatar {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid currentColor;
}

.bottom-badge {
  position: absolute;
  top: 4px;
  right: calc(50% - 18px);
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: var(--radius-full);
  background: var(--color-error);
  color: var(--color-on-error);
  font-size: 10px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px solid var(--color-surface);
}

.unit-dot {
  position: absolute;
  top: 6px;
  right: calc(50% - 14px);
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--gw-accent);
  border: 2px solid var(--gw-surface);
}
</style>
