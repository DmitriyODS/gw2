<template>
  <nav class="bottom-nav">
    <button
      v-for="item in mainItems"
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

    <!-- «Ещё» — выпадающий лист с остальными разделами. Так нижняя
         навигация не превращается в кашу из 6 крошечных кнопок. -->
    <button
      v-if="moreItems.length"
      class="bottom-nav-item"
      :class="{ active: moreOpen || moreActive }"
      @click="moreOpen = !moreOpen"
      aria-label="Ещё"
    >
      <span class="material-symbols-outlined">{{ moreOpen ? 'close' : 'more_horiz' }}</span>
      <span class="bottom-nav-label">Ещё</span>
    </button>
  </nav>

  <Teleport to="body">
    <div v-if="moreOpen" class="more-backdrop" @click="moreOpen = false" />
    <Transition name="more-sheet">
      <div v-if="moreOpen" class="more-sheet" @click.stop>
        <div class="more-handle" />
        <button
          v-for="item in moreItems"
          :key="item.path"
          class="more-item"
          :class="{ active: item.active() }"
          @click="goMore(item)"
        >
          <span class="more-item-ico material-symbols-outlined">{{ item.icon }}</span>
          <span class="more-item-label">{{ item.label }}</span>
          <span v-if="item.active()" class="material-symbols-outlined more-item-check">check</span>
        </button>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
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

const moreOpen = ref(false)
watch(() => route.path, () => { moreOpen.value = false })

// 4 главные кнопки слева — самые частые сценарии. Всё прочее уходит в «Ещё».
const mainItems = computed(() => [
  { path: '/tasks', icon: 'grid_view', label: 'Задачи', tutorial: 'nav-tasks',
    active: () => route.path === '/tasks',
    dot: () => !!unitsStore.activeUnit },
  { path: '/messenger', icon: 'chat', label: 'Чаты', tutorial: 'nav-messenger',
    active: () => route.path.startsWith('/messenger'),
    badge: () => messenger.totalUnread },
  { path: '/stats', icon: 'query_stats', label: 'Статистика', tutorial: 'nav-stats',
    active: () => route.path === '/stats' },
  { path: '/profile', avatar: true, label: 'Профиль', tutorial: 'profile-avatar',
    active: () => route.path === '/profile' },
])

const moreItems = computed(() => {
  const arr = [
    { path: '/groove', icon: 'celebration', label: 'Мой Groove',
      active: () => route.path === '/groove' },
    { path: '/employees', icon: 'groups', label: 'Сотрудники',
      active: () => route.path === '/employees' },
    { path: '/settings', icon: 'settings', label: 'Настройки',
      active: () => route.path === '/settings' },
  ]
  if (isAtLeast(ROLES.ADMIN)) {
    arr.splice(1, 0, { path: '/companies', icon: 'domain', label: 'Компании',
      active: () => route.path.startsWith('/companies') })
    arr.push({ path: '/lists', icon: 'view_list', label: 'Списки',
      active: () => route.path.startsWith('/lists') })
  }
  return arr
})

const moreActive = computed(() => moreItems.value.some((i) => i.active()))

function goMore(item) {
  moreOpen.value = false
  router.push(item.path)
}

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

<!-- Не scoped — Teleport уносит элементы в body. -->
<style>
.more-backdrop {
  position: fixed;
  inset: 0;
  z-index: 250;
  background: var(--color-scrim, color-mix(in oklch, black 45%, transparent));
}

.more-sheet {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 251;
  background: var(--color-surface);
  border-top-left-radius: 20px;
  border-top-right-radius: 20px;
  padding: 10px 12px calc(16px + env(safe-area-inset-bottom, 0px));
  box-shadow: 0 -8px 24px color-mix(in oklch, black 18%, transparent);
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.more-handle {
  align-self: center;
  width: 36px;
  height: 4px;
  border-radius: 2px;
  background: var(--color-outline-dim, color-mix(in oklch, currentColor 25%, transparent));
  margin: 2px 0 8px;
}

.more-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 14px 12px;
  border: none;
  background: transparent;
  color: var(--color-on-surface);
  cursor: pointer;
  border-radius: 14px;
  font: inherit;
  font-size: 15px;
  font-weight: 500;
  text-align: left;
  min-height: 52px;
}

.more-item:active {
  background: color-mix(in oklch, var(--color-primary) 14%, transparent);
}

.more-item.active {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.more-item-ico {
  font-size: 24px;
}

.more-item-label {
  flex: 1;
}

.more-item-check {
  font-size: 20px;
  opacity: 0.8;
}

.more-sheet-enter-active,
.more-sheet-leave-active {
  transition: transform 0.22s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.18s;
}

.more-sheet-enter-from,
.more-sheet-leave-to {
  transform: translateY(100%);
  opacity: 0;
}
</style>
