<template>
  <nav class="bottom-nav">
    <button
      v-for="item in items"
      :key="item.path"
      :data-tutorial="item.tutorial"
      class="bottom-nav-item"
      :class="{ active: item.active() }"
      @click="router.push(item.path)"
    >
      <template v-if="item.avatar">
        <img class="bottom-nav-avatar" :src="avatarSrc" :alt="authStore.user?.fio" />
      </template>
      <template v-else>
        <span class="material-symbols-outlined">{{ item.icon }}</span>
        <span v-if="item.badge && item.badge()" class="bottom-badge">
          {{ item.badge() > 99 ? '99+' : item.badge() }}
        </span>
        <span v-if="item.dot && item.dot()" class="unit-dot" />
      </template>
      <span class="bottom-nav-label">{{ item.label }}</span>
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

// На мобильной нижней навигации помещается ~5 пунктов. Списки/Компании
// прячем под кнопкой "Ещё" в Этапе 2; пока — выводим базовые 5.
const items = computed(() => {
  const arr = [
    { path: '/tasks', icon: 'grid_view', label: 'Задачи', tutorial: 'nav-tasks',
      active: () => route.path === '/tasks',
      dot: () => !!unitsStore.activeUnit },
    { path: '/stats', icon: 'query_stats', label: 'Статистика', tutorial: 'nav-stats',
      active: () => route.path === '/stats' },
    { path: '/employees', icon: 'groups', label: 'Люди', tutorial: 'nav-employees',
      active: () => route.path === '/employees' },
    { path: '/messenger', icon: 'chat', label: 'Чаты', tutorial: 'nav-messenger',
      active: () => route.path.startsWith('/messenger'),
      badge: () => messenger.totalUnread },
  ]
  if (isAtLeast(ROLES.ADMIN)) {
    arr.push({ path: '/companies', icon: 'domain', label: 'Компании', tutorial: 'nav-companies',
      active: () => route.path.startsWith('/companies') })
  } else {
    arr.push({ path: '/settings', icon: 'settings', label: 'Настройки', tutorial: 'nav-settings',
      active: () => route.path === '/settings' })
  }
  arr.push({ path: '/profile', avatar: true, label: 'Профиль', tutorial: 'profile-avatar',
    active: () => route.path === '/profile' })
  return arr
})

const avatarSrc = computed(() => {
  const user = authStore.user
  if (!user) return ''
  if (user.avatar_path) return `/uploads/${user.avatar_path}`
  return `/api/users/${user.id}/identicon`
})
</script>

<style scoped>
.bottom-nav { display: none; }

@media (max-width: 768px) {
  .bottom-nav {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    min-height: calc(64px + env(safe-area-inset-bottom, 0px));
    background: var(--gw-surface);
    border-top: 1px solid var(--gw-border);
    display: flex;
    align-items: stretch;
    z-index: 200;
    padding-top: 6px;
    padding-bottom: max(8px, env(safe-area-inset-bottom, 0px));
    box-sizing: border-box;
  }
}

.bottom-nav-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 3px;
  border: none;
  background: transparent;
  cursor: pointer;
  color: var(--gw-text-secondary);
  position: relative;
  transition: color 0.15s;
  padding: 4px 2px;
  min-width: 0;
}

.bottom-nav-item:active { background: var(--gw-primary-light); }
.bottom-nav-item.active { color: var(--gw-primary); }

.bottom-nav-item .material-symbols-outlined { font-size: 22px; }

.bottom-nav-label {
  font-size: 11px;
  font-weight: 500;
  line-height: 1.1;
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
