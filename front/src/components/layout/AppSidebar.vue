<template>
  <!-- Внешний .sidebar резервирует место в потоке: 264px в закреплённом виде,
       72px в свёрнутом. В свёрнутом виде внутренняя колонка разворачивается
       ПОВЕРХ контента при наведении (как раньше); кнопка-тоггл в шапке
       фиксирует развёрнутое состояние (localStorage). -->
  <nav class="sidebar" :class="{ pinned }">
    <div
      class="sidebar-inner"
      :class="{ expanded }"
      @mouseenter="hovered = true"
      @mouseleave="hovered = false"
    >
      <div class="sb-head">
        <div class="sidebar-logo" data-tutorial="logo" @click="openChangelog" title="Что нового">
          <Logo class="sidebar-logo-img" :size="40" />
          <span class="sb-wordmark">
            <span class="sb-word-groove">Groove</span><span class="sb-word-work">Work</span>
          </span>
        </div>
        <button
          class="sb-toggle"
          type="button"
          :title="pinned ? 'Свернуть панель' : 'Закрепить панель'"
          :aria-label="pinned ? 'Свернуть панель' : 'Закрепить панель'"
          @click="togglePinned"
        >
          <span class="material-symbols-outlined">
            {{ pinned ? 'left_panel_close' : 'left_panel_open' }}
          </span>
        </button>
      </div>

      <div class="sb-sep" />

      <!-- Рендерится ВСЕГДА (а не только в развёрнутом виде): иначе при
           разворачивании плашка компании возникает над кнопками и сдвигает
           их вниз — пользователь промахивается мимо раздела. -->
      <div class="sb-company">
        <div class="sb-group-label">Текущая компания</div>
        <CompanySelect
          variant="row"
          @show="companyDropdownOpen = true"
          @hide="companyDropdownOpen = false"
        />
      </div>

      <div class="sidebar-nav">
        <template v-for="group in navGroups" :key="group.key">
          <div class="sb-sep" />
          <div class="sb-group-label sb-nav-label">{{ group.label }}</div>
          <button
            v-for="item in group.items"
            :key="item.path"
            :data-tutorial="item.tutorial"
            class="nav-btn"
            :class="{ active: item.active() }"
            @click="router.push(item.path)"
            :title="item.label"
          >
            <span class="nav-btn-icon">
              <span class="material-symbols-outlined">{{ item.icon }}</span>
            </span>
            <span class="nav-label">{{ item.label }}</span>
            <span v-if="item.alert && item.alert()" class="nav-badge alert">!</span>
            <span v-else-if="item.badge && item.badge()" class="nav-badge">
              {{ item.badge() > 99 ? '99+' : item.badge() }}
            </span>
          </button>
        </template>
      </div>

      <div class="sidebar-bottom">
        <div class="sb-sep" />
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
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { usePortalStore } from '@/stores/portal.js'
import { useTasksStore } from '@/stores/tasks.js'
import { usePetsStore } from '@/stores/pets.js'
import { usePermission } from '@/composables/usePermission.js'
import { useCompanySettings } from '@/composables/useCompanySettings.js'
import { useChangelog } from '@/composables/useChangelog.js'
import { storageGet, storageSet } from '@/utils/storage.js'
import CompanySelect from '@/components/common/CompanySelect.vue'
import Logo from '@/components/common/Logo.vue'

const PINNED_KEY = 'gw_sidebar_pinned'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const messenger = useMessengerStore()
const portal = usePortalStore()
const tasksStore = useTasksStore()
const petsStore = usePetsStore()
const { isSuperAdmin, canManageCompanies } = usePermission()
const { usesGroove } = useCompanySettings()
const { open: openChangelog } = useChangelog()

const hovered = ref(false)
const companyDropdownOpen = ref(false)
// Закреплённое развёрнутое состояние — выбор пользователя, переживает сессию.
const pinned = ref(storageGet(PINNED_KEY, '1') !== '0')

function togglePinned() {
  pinned.value = !pinned.value
  storageSet(PINNED_KEY, pinned.value ? '1' : '0')
}

// Активная компания есть только у обычного пользователя-члена (roleLevel>0).
// У супер-админа активной компании нет — компанийный контент он не видит.
const hasActiveCompany = computed(() => !isSuperAdmin() && authStore.roleLevel > 0)

// Развёрнут: закреплён кнопкой ИЛИ курсор на панели ИЛИ открыта overlay-выпадашка
// CompanySelect (она правее панели — на mouseleave панель бы свернулась).
const expanded = computed(() => pinned.value || hovered.value || companyDropdownOpen.value)

// Три смысловые группы разделов (пустые группы не рендерятся):
//   Рабочие процессы — личная продуктивность и данные компании;
//   Коммуникация с командой — общение и внутренняя жизнь;
//   Управление и анализ — администрирование и статистика.
const navGroups = computed(() => {
  const workflows = []
  if (hasActiveCompany.value) {
    workflows.push(
      { path: '/tasks', icon: 'dashboard_customize', label: 'Задачи', tutorial: 'nav-tasks',
        active: () => route.path.startsWith('/tasks'),
        badge: () => tasksStore.myActiveCount },
      { path: '/registries', icon: 'view_agenda', label: 'Реестры', tutorial: 'nav-registries',
        active: () => route.path.startsWith('/registries') },
    )
  }
  // Заметки (вкл. ежедневник) — личные, кросс-компанийные: доступны всегда.
  workflows.push({ path: '/notes', icon: 'note_stack', label: 'Заметки', tutorial: 'nav-diaries',
    active: () => route.path.startsWith('/notes') || route.path.startsWith('/diaries') })
  if (hasActiveCompany.value) {
    workflows.push({ path: '/calendars', icon: 'calendar_month', label: 'Календари', tutorial: 'nav-calendars',
      active: () => route.path.startsWith('/calendars') })
  }

  const team = [
    // Мессенджер доступен всегда (в т.ч. без активной компании).
    { path: '/messenger', icon: 'chat', label: 'Мессенджер', tutorial: 'nav-messenger',
      active: () => route.path.startsWith('/messenger'),
      badge: () => messenger.totalUnread },
  ]
  if (hasActiveCompany.value) {
    // Единый раздел: лента портала + сотрудники (вкладки внутри).
    team.push({ path: '/portal', icon: 'brand_awareness', label: 'Портал', tutorial: 'nav-portal',
      active: () => route.path.startsWith('/portal') || route.path === '/employees',
      badge: () => portal.unread })
    if (usesGroove.value) {
      team.push({ path: '/pets', icon: 'pets', label: 'Грувики', tutorial: 'nav-groove',
        active: () => route.path === '/pets',
        alert: () => !!petsStore.pet?.sick })
    }
  }

  const manage = []
  if (hasActiveCompany.value) {
    manage.push({ path: '/stats', icon: 'bar_chart', label: 'Статистика', tutorial: 'nav-stats',
      active: () => route.path === '/stats' })
  }
  if (canManageCompanies()) {
    manage.push({ path: '/companies', icon: 'business_center',
      label: isSuperAdmin() ? 'Компании' : 'Мои компании',
      tutorial: 'nav-companies', active: () => route.path.startsWith('/companies') })
  }
  if (isSuperAdmin()) {
    manage.push({ path: '/users', icon: 'group', label: 'Пользователи', tutorial: 'nav-users',
      active: () => route.path === '/users' })
  }
  manage.push({ path: '/settings', icon: 'settings', label: 'Настройки', tutorial: 'nav-settings',
    active: () => route.path === '/settings' })

  return [
    { key: 'work', label: 'Рабочие процессы', items: workflows },
    { key: 'team', label: 'Коммуникация с командой', items: team },
    { key: 'manage', label: 'Управление и анализ', items: manage },
  ].filter((g) => g.items.length)
})

const avatarSrc = computed(() => {
  const user = authStore.user
  if (!user) return ''
  if (user.avatar_path) return `/uploads/${user.avatar_path}`
  return `/api/users/${user.id}/identicon`
})

// Бейдж «моих» активных задач: при входе и на смену активной компании.
onMounted(() => { tasksStore.fetchMyActiveCount() })
watch(() => authStore.companyId, () => { tasksStore.fetchMyActiveCount() })
</script>

<style scoped>
/* Внешняя колонка резервирует ширину панели + отступы плавающей карточки. */
.sidebar {
  width: 96px; /* 72 + 12·2 */
  flex-shrink: 0;
  position: relative;
  z-index: 100;
  transition: width 0.24s cubic-bezier(0.4, 0, 0.2, 1);
}

.sidebar.pinned { width: 288px; /* 264 + 12·2 */ }

@media (max-width: 768px) { .sidebar { display: none; } }

/* Плавающая акриловая панель: отстоит от краёв экрана, скруглена по всему
   контуру; полупрозрачный фон + blur — фон приложения мягко просвечивает,
   при hover-развороте поверх контента стекло размывает и его. */
.sidebar-inner {
  position: sticky;
  top: 0;
  margin: 12px;
  height: calc(100vh - 24px);
  height: calc(100dvh - 24px);
  width: 72px;
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  display: flex;
  flex-direction: column;
  align-items: stretch;
  padding: 14px 0 12px;
  overflow: hidden;
  transition: width 0.24s cubic-bezier(0.4, 0, 0.2, 1), box-shadow 0.24s ease;
}

.sidebar-inner.expanded { width: 264px; }

/* Тень — только когда панель развёрнута ПОВЕРХ контента (не закреплена). */
.sidebar:not(.pinned) .sidebar-inner.expanded { box-shadow: var(--shadow-lg); }

/* ── Шапка: логотип + двухтоновый wordmark + тоггл ── */
.sb-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 12px 12px;
}

.sidebar-logo {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 44px;
  flex: 1;
  min-width: 0;
  padding: 0 4px;
  cursor: pointer;
  border-radius: var(--radius-md);
  transition: background 0.15s;
  overflow: hidden;
}

.sidebar-logo:hover { background: color-mix(in oklch, var(--color-primary) 10%, transparent); }

.sidebar-logo-img { width: 40px; height: 40px; border-radius: 12px; display: block; flex-shrink: 0; }

.sb-wordmark {
  font-size: 19px;
  /* ExtraBlack вариативного Roboto Flex — фирменное начертание wordmark. */
  font-weight: 1000;
  letter-spacing: 0.2px;
  white-space: nowrap;
  opacity: 0;
  transition: opacity 0.18s ease;
}

.sb-word-groove { color: var(--color-primary); }
.sb-word-work { color: color-mix(in oklch, var(--color-primary) 40%, var(--color-primary-container)); }

.sb-toggle {
  width: 34px;
  height: 34px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border: 1px solid color-mix(in oklch, var(--color-primary) 20%, transparent);
  border-radius: var(--radius-sm);
  background: var(--grad-primary-soft);
  color: var(--color-primary);
  cursor: pointer;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.18s ease, background 0.15s;
}

.sb-toggle:hover { background: var(--color-primary-container); }
.sb-toggle .material-symbols-outlined { font-size: 20px; }

.sidebar-inner.expanded .sb-wordmark,
.sidebar-inner.expanded .sb-toggle { opacity: 1; }
.sidebar-inner.expanded .sb-toggle { pointer-events: auto; }

/* Свёрнутый вид: логотип по центру колонки. Wordmark и тоггл не только
   прозрачные, но и не занимают места — иначе они выталкивают лого за
   пределы узкой колонки (overflow клипует именно картинку). */
.sidebar-inner:not(.expanded) .sb-head { justify-content: center; }
.sidebar-inner:not(.expanded) .sidebar-logo { flex: 0 0 auto; padding: 0; gap: 0; }
.sidebar-inner:not(.expanded) .sb-wordmark { width: 0; overflow: hidden; }
.sidebar-inner:not(.expanded) .sb-toggle { display: none; }

/* ── Разделители секций и подписи групп ── */
.sb-sep {
  height: 1px;
  flex-shrink: 0;
  background: var(--color-outline-dim);
  opacity: 0.55;
}

.sb-group-label {
  height: 30px;
  flex-shrink: 0;
  display: flex;
  align-items: flex-end;
  padding: 0 16px 4px;
  font-size: 11.5px;
  font-weight: 600;
  letter-spacing: 0.2px;
  color: var(--color-text-dim);
  white-space: nowrap;
  overflow: hidden;
  opacity: 0;
  transition: opacity 0.18s ease, height 0.24s cubic-bezier(0.4, 0, 0.2, 1), padding 0.24s cubic-bezier(0.4, 0, 0.2, 1);
}

.sidebar-inner.expanded .sb-group-label { opacity: 1; }

/* Свёрнутый вид: подписи групп схлопываются до тонкой прослойки —
   иначе между секциями остаются пустые полосы во всю их высоту. */
.sidebar-inner:not(.expanded) .sb-group-label {
  height: 8px;
  padding-bottom: 0;
}

/* ── Плашка текущей компании ── */
.sb-company {
  padding: 0 12px 12px;
  display: flex;
  flex-direction: column;
}

.sb-company .sb-group-label { padding-left: 4px; }

.sb-company :deep(.company-row),
.sb-company :deep(.company-chip) {
  background: var(--grad-primary-soft);
  border: 1px solid color-mix(in oklch, var(--color-primary) 16%, transparent);
  border-radius: var(--radius-lg);
}

.sb-company :deep(.company-row:hover) {
  background: var(--color-primary-container);
}

.sb-company :deep(.company-row-badge) {
  background: var(--grad-primary);
  color: var(--color-on-primary);
}

/* Свёрнутая панель: компанию показываем одним бейджем по центру колонки,
   текст скрыт. Высота секции постоянна в обоих состояниях, поэтому кнопки
   разделов не «прыгают» при разворачивании. */
.sidebar-inner:not(.expanded) :deep(.company-row),
.sidebar-inner:not(.expanded) :deep(.company-chip) {
  background: transparent;
  border-color: transparent;
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

.sidebar-inner:not(.expanded) :deep(.company-row-chev) { display: none; }

.sidebar-inner :deep(.company-row-text),
.sidebar-inner :deep(.company-chip-label) {
  transition: opacity 0.18s ease;
}

/* ── Список разделов ──
   Скроллится сам — логотип, компания и профиль остаются на месте.
   min-height:0, чтобы flex-элемент мог сжаться; скроллбар прячем. */
.sidebar-nav {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  scrollbar-width: none;
}
.sidebar-nav::-webkit-scrollbar { width: 0; height: 0; }

.sidebar-nav .sb-nav-label { margin-top: 2px; }

/* Пункты — на всю ширину панели (edge-to-edge, как в макете). */
.nav-btn {
  position: relative;
  display: flex;
  align-items: center;
  gap: 14px;
  height: 48px;
  flex-shrink: 0;
  width: 100%;
  padding: 0 16px;
  border: none;
  border-radius: 0;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
  overflow: hidden;
}

.nav-btn:hover {
  background: color-mix(in oklch, var(--color-primary) 10%, transparent);
  color: var(--color-primary);
}

/* Активный пункт — фирменный градиент во всю ширину. */
.nav-btn.active {
  background: var(--grad-primary);
  color: var(--color-on-primary);
}

.nav-btn-icon {
  width: 24px;
  height: 24px;
  display: grid;
  place-items: center;
  flex-shrink: 0;
}

.nav-btn-icon .material-symbols-outlined { font-size: 24px; }

.nav-btn.active .nav-btn-icon .material-symbols-outlined {
  font-variation-settings: 'FILL' 1, 'wght' 400, 'GRAD' 0, 'opsz' 24;
}

.nav-label {
  font-size: 14.5px;
  font-weight: 650;
  white-space: nowrap;
  opacity: 0;
  transition: opacity 0.18s ease;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sidebar-inner.expanded .nav-label { opacity: 1; }

/* Свёрнутый вид: иконка по центру. */
.sidebar-inner:not(.expanded) .nav-btn { justify-content: center; padding: 0; gap: 0; }
.sidebar-inner:not(.expanded) .nav-label { width: 0; }

/* Бейдж — тинтованная плашка справа (не «алярмный» красный кружок). */
.nav-badge {
  margin-left: auto;
  min-width: 24px;
  height: 22px;
  padding: 0 7px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  background: color-mix(in oklch, var(--color-primary) 14%, var(--color-surface));
  border: 1px solid color-mix(in oklch, var(--color-primary) 22%, transparent);
  color: var(--color-primary);
  font-size: 11.5px;
  font-weight: 700;
  transition: opacity 0.18s ease;
}

.nav-btn.active .nav-badge {
  background: var(--acrylic-bg-strong);
  border-color: transparent;
  color: var(--color-primary);
}

.nav-badge.alert {
  background: var(--color-error-container);
  border-color: color-mix(in oklch, var(--color-error) 30%, transparent);
  color: var(--color-on-error-container);
}

/* Свёрнутый вид: бейдж сжимается в точку у иконки. */
.sidebar-inner:not(.expanded) .nav-badge {
  position: absolute;
  top: 8px;
  right: 16px;
  min-width: 9px;
  width: 9px;
  height: 9px;
  padding: 0;
  margin: 0;
  border: none;
  border-radius: 50%;
  background: var(--color-primary);
  font-size: 0;
}

.sidebar-inner:not(.expanded) .nav-badge.alert { background: var(--color-error); }

/* ── Профиль ── */
.sidebar-bottom {
  margin-top: auto;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.user-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 0 12px;
  padding: 6px 8px;
  border: 1px solid color-mix(in oklch, var(--color-primary) 16%, transparent);
  background: var(--grad-primary-soft);
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
  overflow: hidden;
}

.user-row:hover { background: var(--color-primary-container); }

.user-avatar {
  width: 38px;
  height: 38px;
  border-radius: var(--radius-md);
  object-fit: cover;
  flex-shrink: 0;
}

.user-name {
  color: var(--color-text);
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Свёрнутый вид: только аватар по центру, без рамки-плашки. */
.sidebar-inner:not(.expanded) .user-row {
  justify-content: center;
  gap: 0;
  padding: 6px 0;
  margin: 0 8px;
  background: transparent;
  border-color: transparent;
}
</style>
