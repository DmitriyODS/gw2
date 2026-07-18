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
      <span v-if="moreBadge" class="bottom-badge">{{ moreBadge > 99 ? '99+' : moreBadge }}</span>
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
          :data-tutorial="item.tutorial"
          class="more-item"
          :class="{ active: item.active() }"
          @click="goMore(item)"
        >
          <img v-if="item.avatar" class="more-item-avatar" :src="avatarSrc" :alt="authStore.user?.fio" />
          <span v-else class="more-item-ico material-symbols-outlined">{{ item.icon }}</span>
          <span class="more-item-label">{{ item.label }}</span>
          <span v-if="item.badge && item.badge()" class="more-item-badge">
            {{ item.badge() > 99 ? '99+' : item.badge() }}
          </span>
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
import { usePortalStore } from '@/stores/portal.js'
import { usePermission } from '@/composables/usePermission.js'
import { useCompanySettings } from '@/composables/useCompanySettings.js'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const unitsStore = useUnitsStore()
const messenger = useMessengerStore()
const portal = usePortalStore()
const { isSuperAdmin, hasActiveCompany } = usePermission()
const { usesGroove } = useCompanySettings()

const moreOpen = ref(false)
watch(() => route.path, () => { moreOpen.value = false })

// 4 главные кнопки слева — самые частые сценарии. Всё прочее (вкл. профиль)
// уходит в «Ещё». Компанийные разделы (задачи, портал, реестры, календари,
// статистика, грувики) — только при активной компании, как в сайдбаре.
const mainItems = computed(() => [
  ...(hasActiveCompany() ? [{ path: '/tasks', icon: 'grid_view', label: 'Задачи', tutorial: 'nav-tasks',
    active: () => route.path === '/tasks',
    dot: () => !!unitsStore.activeUnit }] : []),
  { path: '/messenger', icon: 'chat', label: 'Чаты', tutorial: 'nav-messenger',
    active: () => route.path.startsWith('/messenger'),
    badge: () => messenger.totalUnread },
  { path: '/notes', icon: 'note_stack', label: 'Заметки',
    active: () => route.path.startsWith('/notes') },
  // Единый раздел: лента портала + сотрудники (вкладки внутри).
  ...(hasActiveCompany() ? [{ path: '/portal', icon: 'campaign', label: 'Портал',
    active: () => route.path.startsWith('/portal') || route.path === '/employees',
    badge: () => portal.unread }] : []),
])

const moreItems = computed(() => {
  const arr = [
    { path: '/profile', avatar: true, icon: 'account_circle', label: 'Профиль', tutorial: 'profile-avatar',
      active: () => route.path === '/profile' },
    { path: '/diaries', icon: 'event_note', label: 'Ежедневник',
      active: () => route.path.startsWith('/diaries') },
  ]
  // Питомцы-грувики — только если компания не выключила режим.
  if (hasActiveCompany() && usesGroove.value) {
    arr.push({ path: '/pets', icon: 'pets', label: 'Грувики',
      active: () => route.path === '/pets' })
  }
  // «Компании» — всем: любой пользователь может создать свою компанию,
  // не будучи администратором ни одной (раздел сам покажет пустой список
  // с кнопкой «Создать»).
  arr.push({ path: '/companies', icon: 'domain', label: 'Компании',
    active: () => route.path.startsWith('/companies') })
  if (hasActiveCompany()) {
    arr.push(
      { path: '/registries', icon: 'table', label: 'Реестры',
        active: () => route.path.startsWith('/registries') },
      { path: '/calendars', icon: 'calendar_month', label: 'Календари',
        active: () => route.path.startsWith('/calendars') },
      { path: '/stats', icon: 'query_stats', label: 'Статистика', tutorial: 'nav-stats',
        active: () => route.path === '/stats' },
    )
  }
  if (isSuperAdmin()) {
    arr.push({ path: '/users', icon: 'group', label: 'Пользователи',
      active: () => route.path === '/users' })
  }
  arr.push({ path: '/settings', icon: 'settings', label: 'Настройки',
    active: () => route.path === '/settings' })
  return arr
})

const moreActive = computed(() => moreItems.value.some((i) => i.active()))

// Бейджи разделов, спрятанных за «Ещё», суммируются на самой кнопке —
// иначе непрочитанное не видно, пока лист закрыт.
const moreBadge = computed(() => moreItems.value.reduce((sum, i) => sum + (i.badge ? i.badge() : 0), 0))

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
  /* Акриловая панель: контент мягко просвечивает при прокрутке под ней. */
  .bottom-nav {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    min-height: calc(64px + env(safe-area-inset-bottom, 0px));
    background: var(--acrylic-bg);
    -webkit-backdrop-filter: var(--acrylic-blur);
    backdrop-filter: var(--acrylic-blur);
    border-top: 1px solid var(--acrylic-border);
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

.bottom-nav-item:active { background: color-mix(in oklch, var(--color-primary) 10%, transparent); }
.bottom-nav-item.active { color: var(--gw-primary); }

/* Иконка в «пилюле»: у активного пункта — фирменный градиент (M3-индикатор). */
.bottom-nav-item .material-symbols-outlined {
  font-size: 22px;
  padding: 3px 14px;
  border-radius: var(--radius-full);
  transition: background 0.15s, color 0.15s;
}

.bottom-nav-item.active .material-symbols-outlined {
  background: var(--grad-primary);
  color: var(--color-on-primary);
  font-variation-settings: 'FILL' 1, 'wght' 400, 'GRAD' 0, 'opsz' 24;
}

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
  top: 2px;
  right: calc(50% - 22px);
  min-width: 17px;
  height: 17px;
  padding: 0 4px;
  border-radius: 6px;
  background: color-mix(in oklch, var(--color-primary) 16%, var(--color-surface));
  color: var(--color-primary);
  font-size: 10px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid color-mix(in oklch, var(--color-primary) 26%, transparent);
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
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
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
  background: var(--grad-primary);
  color: var(--color-on-primary);
}

.more-item-ico {
  font-size: 24px;
}

.more-item-avatar {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  object-fit: cover;
}

.more-item-label {
  flex: 1;
}

.more-item-badge {
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  border-radius: 7px;
  background: color-mix(in oklch, var(--color-primary) 16%, var(--color-surface));
  color: var(--color-primary);
  border: 1px solid color-mix(in oklch, var(--color-primary) 26%, transparent);
  font-size: 11px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
}

.more-item.active .more-item-badge {
  background: var(--color-surface);
  border-color: transparent;
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
