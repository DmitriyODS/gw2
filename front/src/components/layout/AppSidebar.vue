<template>
  <!-- Внешний .sidebar резервирует 72px в потоке; внутренний разворачивается
       поверх контента при наведении (см. v2.7.0). Пункты — данными по ролям. -->
  <nav class="sidebar">
    <div class="sidebar-inner" :class="{ expanded }" @mouseenter="hovered = true" @mouseleave="hovered = false">
      <div class="sidebar-logo" data-tutorial="logo" @click="openChangelog" title="Что нового">
        <Logo class="sidebar-logo-img" :size="40" />
        <span class="sidebar-logo-text">Groove Work</span>
      </div>

      <!-- Рендерится ВСЕГДА (а не только в развёрнутом виде): иначе при
           разворачивании плашка компании возникает над кнопками и сдвигает
           их вниз — пользователь промахивается мимо раздела. В свёрнутом виде
           показывается одним бейджем по центру колонки, место под неё всегда
           зарезервировано, поэтому кнопки не «прыгают». -->
      <div class="sidebar-company">
        <CompanySelect
          variant="row"
          @show="companyDropdownOpen = true"
          @hide="companyDropdownOpen = false"
        />
      </div>

      <div class="sidebar-nav">
        <button
          v-for="item in navItems"
          :key="item.path"
          :data-tutorial="item.tutorial"
          class="nav-btn"
          :class="{ active: item.active() }"
          @click="router.push(item.path)"
          :title="item.label"
        >
          <span class="nav-btn-icon">
            <span class="material-symbols-outlined">{{ item.icon }}</span>
            <span v-if="item.badge && item.badge()" class="nav-badge">
              {{ item.badge() > 99 ? '99+' : item.badge() }}
            </span>
          </span>
          <span class="nav-label">{{ item.label }}</span>
        </button>
      </div>

      <div class="sidebar-bottom">
        <button class="user-row" @click="router.push('/profile')" title="Профиль">
          <img
            data-tutorial="profile-avatar"
            class="user-avatar"
            :src="avatarSrc"
            :alt="authStore.user?.fio"
          />
          <span class="nav-label user-name">{{ authStore.user?.fio || 'Профиль' }}</span>
        </button>
      </div>
    </div>
  </nav>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import { useChangelog } from '@/composables/useChangelog.js'
import CompanySelect from '@/components/common/CompanySelect.vue'
import Logo from '@/components/common/Logo.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const messenger = useMessengerStore()
const { isAtLeast } = usePermission()
const { open: openChangelog } = useChangelog()

const hovered = ref(false)
const companyDropdownOpen = ref(false)

// Сайдбар развёрнут, пока курсор на нём — ИЛИ пока открыта overlay-выпадашка
// CompanySelect: пользователь уже навёл курсор на пункт в overlay-панели,
// которая расположена ПРАВЕЕ сайдбара (за пределами .sidebar-inner). На mouseleave
// сайдбар бы свернулся, выпадашка пропала бы. Держим его развёрнутым по флагу.
const expanded = computed(() => hovered.value || companyDropdownOpen.value)

// Состав боковой панели по ролям (см. PLAN_V3.md):
//   Сотрудник/Менеджер/Руководитель: Задачи, Статистика, Мессенджер,
//     Сотрудники [+ Списки от Руководителя], Настройки, Аккаунт.
//   Администратор системы: всё то же + Компании + Списки.
const navItems = computed(() => {
  const items = [
    { path: '/tasks', icon: 'grid_view', label: 'Задачи', tutorial: 'nav-tasks',
      active: () => route.path === '/tasks' },
    { path: '/stats', icon: 'query_stats', label: 'Статистика', tutorial: 'nav-stats',
      active: () => route.path === '/stats' },
    { path: '/messenger', icon: 'chat', label: 'Мессенджер', tutorial: 'nav-messenger',
      active: () => route.path.startsWith('/messenger'),
      badge: () => messenger.totalUnread },
    { path: '/groove', icon: 'celebration', label: 'Мой Groove', tutorial: 'nav-groove',
      active: () => route.path === '/groove' },
    { path: '/employees', icon: 'groups', label: 'Сотрудники', tutorial: 'nav-employees',
      active: () => route.path === '/employees' },
  ]

  // Раздел "Компании" — только Администратор системы.
  if (isAtLeast(ROLES.ADMIN)) {
    items.push({ path: '/companies', icon: 'domain', label: 'Компании', tutorial: 'nav-companies',
      active: () => route.path.startsWith('/companies') })
  }

  // Раздел "Списки" — от Руководителя (новый, ex-настройки). Сотрудникам
  // и Менеджерам — только просмотр через будущий API, ссылка скрыта.
  if (isAtLeast(ROLES.DIRECTOR)) {
    items.push({ path: '/lists', icon: 'list_alt', label: 'Списки', tutorial: 'nav-lists',
      active: () => route.path.startsWith('/lists') })
  }

  items.push({ path: '/settings', icon: 'settings', label: 'Настройки', tutorial: 'nav-settings',
    active: () => route.path === '/settings' })

  return items
})

const avatarSrc = computed(() => {
  const user = authStore.user
  if (!user) return ''
  if (user.avatar_path) return `/uploads/${user.avatar_path}`
  return `/api/users/${user.id}/identicon`
})
</script>

<style scoped>
.sidebar { width: 72px; flex-shrink: 0; position: relative; z-index: 100; }

@media (max-width: 768px) { .sidebar { display: none; } }

.sidebar-inner {
  position: sticky;
  top: 0;
  height: 100vh;
  height: 100dvh;
  width: 72px;
  background: var(--gw-sidebar-bg);
  border-right: 1px solid var(--gw-border);
  display: flex;
  flex-direction: column;
  align-items: stretch;
  padding: 16px 12px;
  overflow: hidden;
  transition: width 0.24s cubic-bezier(0.4, 0, 0.2, 1), box-shadow 0.24s ease;
}

.sidebar-inner.expanded { width: 256px; box-shadow: var(--shadow-lg); }

.sidebar-logo {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 48px;
  margin-bottom: 16px;
  padding: 0 4px;
  cursor: pointer;
  border-radius: 12px;
  transition: background 0.15s;
  overflow: hidden;
}

.sidebar-logo:hover { background: var(--gw-primary-light); }

.sidebar-logo-img { width: 40px; height: 40px; border-radius: 12px; display: block; flex-shrink: 0; }

.sidebar-logo-text {
  font-size: 17px;
  font-weight: 700;
  color: var(--color-text);
  white-space: nowrap;
}

.sidebar-company {
  padding: 0 4px 12px;
  display: flex;
}

.sidebar-company > * { flex: 1; min-width: 0; }

/* Свёрнутая панель: компанию показываем одним бейджем/иконкой по центру
   колонки (как иконки разделов), текст скрыт. Высота секции постоянна в обоих
   состояниях, поэтому кнопки разделов не сдвигаются при разворачивании. */
.sidebar-inner:not(.expanded) :deep(.company-row),
.sidebar-inner:not(.expanded) :deep(.company-chip) {
  background: transparent;
  justify-content: center;
  padding-left: 4px;
  padding-right: 4px;
  gap: 0;
}

.sidebar-inner:not(.expanded) :deep(.company-row-text),
.sidebar-inner:not(.expanded) :deep(.company-chip-label) {
  width: 0;
  min-width: 0;
  opacity: 0;
  overflow: hidden;
}

.sidebar-inner:not(.expanded) :deep(.company-row-chev) {
  display: none;
}

.sidebar-inner :deep(.company-row-text),
.sidebar-inner :deep(.company-chip-label) {
  transition: opacity 0.18s ease;
}

.sidebar-nav { display: flex; flex-direction: column; gap: 6px; flex: 1; }

.nav-btn {
  display: flex;
  align-items: center;
  gap: 14px;
  height: 48px;
  width: 100%;
  padding: 0 12px;
  border-radius: 12px;
  border: none;
  background: transparent;
  color: var(--gw-text-secondary);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
  overflow: hidden;
}

.nav-btn:hover { background: var(--gw-primary-light); color: var(--gw-primary); }
.nav-btn.active { background: var(--gw-primary); color: var(--color-on-primary); }

.nav-btn-icon { position: relative; width: 24px; height: 24px; display: grid; place-items: center; flex-shrink: 0; }

.nav-btn-icon .material-symbols-outlined { font-size: 24px; }

.nav-label {
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
  opacity: 0;
  transition: opacity 0.18s ease;
}

.sidebar-inner.expanded .nav-label { opacity: 1; }

.nav-badge {
  position: absolute;
  top: -6px;
  right: -8px;
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

.sidebar-bottom { margin-top: auto; padding-top: 16px; }

.user-row {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  padding: 6px 8px;
  border: none;
  background: transparent;
  border-radius: 12px;
  cursor: pointer;
  transition: background 0.15s;
  overflow: hidden;
}

.user-row:hover { background: var(--gw-primary-light); }

.user-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid var(--gw-border);
  transition: border-color 0.15s;
  flex-shrink: 0;
}

.user-row:hover .user-avatar { border-color: var(--gw-primary); }

.user-name { color: var(--color-text); overflow: hidden; text-overflow: ellipsis; }
</style>
