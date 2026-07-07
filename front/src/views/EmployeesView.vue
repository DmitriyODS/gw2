<template>
  <div class="admin-page">
    <header class="admin-sticky">
      <PortalHubTabs class="emp-hub-tabs" />
      <div class="page-head">
        <div class="page-head-text">
          <h1 class="page-head-title">{{ pageTitle }}</h1>
          <div class="page-head-meta">
            <span class="meta-stat">
              <span class="material-symbols-outlined">groups</span>
              <strong>{{ scopedUsers.length }}</strong>
              {{ pluralPeople(scopedUsers.length) }}
            </span>
            <span class="meta-dot" aria-hidden="true">·</span>
            <span class="meta-stat online">
              <span class="presence-pulse" />
              <strong>{{ onlineCount }}</strong> в сети
            </span>
          </div>
        </div>
      </div>

      <div class="admin-toolbar">
        <div class="emp-search">
          <span class="material-symbols-outlined">search</span>
          <input v-model.trim="search" placeholder="Поиск по ФИО, логину, должности" />
          <button v-if="search" class="search-clear" @click="search = ''" aria-label="Очистить">
            <span class="material-symbols-outlined">close</span>
          </button>
        </div>

        <div v-if="roleFilters.length > 1" class="emp-chips" role="tablist">
          <button
            v-for="f in roleFilters"
            :key="f.key"
            :class="['chip', { active: roleFilter === f.key }]"
            @click="roleFilter = f.key"
            role="tab"
            :aria-selected="roleFilter === f.key"
          >
            <span v-if="f.icon" class="material-symbols-outlined">{{ f.icon }}</span>
            {{ f.label }}
            <span class="chip-count">{{ f.count }}</span>
          </button>
        </div>
      </div>
    </header>

    <div class="admin-body">
      <!-- Карточки -->
      <div class="emp-grid">
        <article
          v-for="u in filtered"
          :key="u.id"
          class="emp-card"
          :class="{ online: messenger.isOnline(u.id), 'is-me': u.id === auth.user?.id }"
          tabindex="0"
          @click="openProfile(u)"
          @keydown.enter.prevent="openProfile(u)"
          @keydown.space.prevent="openProfile(u)"
        >
          <span
            v-if="u.is_super_admin"
            class="root-corner"
            title="Супер-администратор платформы"
          >
            <span class="material-symbols-outlined">verified</span>
          </span>

          <div class="emp-card-avatar-wrap">
            <span class="avatar avatar-lg" :class="presenceClass(u)">
              <img :src="avatarOf(u)" :alt="u.fio" />
            </span>
          </div>

          <h3 class="emp-card-name">
            {{ u.fio }}
            <span v-if="u.id === auth.user?.id" class="me-tag">это вы</span>
          </h3>
          <p class="emp-card-post" :title="u.post || ''">
            {{ u.post || '—' }}
          </p>

          <RolePill :level="u.role?.level" :name="u.role?.name" />

          <p class="emp-card-status" :class="{ on: messenger.isOnline(u.id) }">
            {{ statusOf(u) }}
          </p>

          <div
            v-if="u.id !== auth.user?.id"
            class="emp-card-actions"
            @click.stop
          >
            <button
              class="card-act"
              title="Написать"
              @click="writeTo(u)"
              :aria-label="`Написать ${u.fio}`"
            >
              <span class="material-symbols-outlined">chat_bubble</span>
            </button>
            <button
              v-if="callsOn"
              class="card-act"
              title="Видеозвонок"
              @click="callTo(u, 'video')"
              :aria-label="`Видеозвонок: ${u.fio}`"
            >
              <span class="material-symbols-outlined">videocam</span>
            </button>
            <button
              v-if="callsOn"
              class="card-act"
              title="Аудиозвонок"
              @click="callTo(u, 'audio')"
              :aria-label="`Аудиозвонок: ${u.fio}`"
            >
              <span class="material-symbols-outlined">call</span>
            </button>
          </div>
        </article>

        <EmptyState
          v-if="!filtered.length"
          class="emp-grid-empty"
          :icon="search ? 'search_off' : 'person_off'"
          :title="search ? 'Никого не нашли' : 'Сотрудников пока нет'"
          :subtitle="search ? 'Попробуйте уточнить запрос или сбросить фильтры.' : ''"
        />
      </div>
    </div>

    <!-- Профиль сотрудника. -->
    <Dialog
      v-model:visible="profileOpen"
      modal
      :draggable="false"
      :show-header="false"
      :dismissable-mask="true"
      :style="{ width: '460px', maxWidth: 'calc(100vw - 24px)' }"
      :pt="{
        root: { class: 'emp-dialog' },
        content: { style: 'overflow-x: hidden; padding: 0; background: transparent' },
        mask: { style: 'background: var(--color-scrim)' },
      }"
    >
      <div v-if="selected" class="emp-profile">
        <div class="profile-cover" aria-hidden="true"></div>
        <button class="profile-close" @click="profileOpen = false" aria-label="Закрыть">
          <span class="material-symbols-outlined">close</span>
        </button>

        <div class="profile-hero">
          <button class="profile-avatar-btn" @click="lightboxOpen = true" aria-label="Открыть фото">
            <span class="avatar avatar-xl" :class="presenceClass(selected)">
              <img :src="avatarOf(selected)" :alt="selected.fio" />
            </span>
          </button>
          <h2 class="profile-name">
            {{ selected.fio }}
            <span
              v-if="selected.is_super_admin"
              class="root-badge inline"
              title="Супер-администратор платформы"
            >
              <span class="material-symbols-outlined">verified</span>
            </span>
          </h2>
          <div class="profile-tags">
            <RolePill :level="selected.role?.level" :name="selected.role?.name" />
            <span :class="['profile-status', { on: messenger.isOnline(selected.id) }]">
              <span class="status-dot" />
              {{ statusOf(selected) }}
            </span>
          </div>
        </div>

        <div class="profile-list">
          <div v-if="selected.post" class="profile-row">
            <span class="row-ico" data-tone="primary">
              <span class="material-symbols-outlined">badge</span>
            </span>
            <span class="row-text">
              <span class="row-label">Должность</span>
              <span class="row-value">{{ selected.post }}</span>
            </span>
          </div>
          <div class="profile-row">
            <span class="row-ico" data-tone="secondary">
              <span class="material-symbols-outlined">alternate_email</span>
            </span>
            <span class="row-text">
              <span class="row-label">Логин</span>
              <span class="row-value">@{{ selected.login }}</span>
            </span>
          </div>
          <a
            v-if="selected.phone"
            class="profile-row link"
            :href="`tel:${selected.phone}`"
          >
            <span class="row-ico" data-tone="tertiary">
              <span class="material-symbols-outlined">phone</span>
            </span>
            <span class="row-text">
              <span class="row-label">Телефон</span>
              <span class="row-value">{{ fmtPhone(selected.phone) }}</span>
            </span>
            <span class="material-symbols-outlined row-chev">arrow_outward</span>
          </a>
          <a
            v-if="selected.email"
            class="profile-row link"
            :href="`mailto:${selected.email}`"
          >
            <span class="row-ico" data-tone="tertiary">
              <span class="material-symbols-outlined">mail</span>
            </span>
            <span class="row-text">
              <span class="row-label">Email</span>
              <span class="row-value">{{ selected.email }}</span>
            </span>
            <span class="material-symbols-outlined row-chev">arrow_outward</span>
          </a>
          <div v-if="companyOf(selected)" class="profile-row">
            <span class="row-ico" data-tone="primary">
              <span class="material-symbols-outlined">domain</span>
            </span>
            <span class="row-text">
              <span class="row-label">Компания</span>
              <span class="row-value">{{ companyOf(selected) }}</span>
            </span>
          </div>
        </div>

        <div v-if="selected.id !== auth.user?.id" class="profile-actions">
          <button class="btn-filled" @click="writeTo(selected)">
            <span class="material-symbols-outlined">chat</span>
            Написать
          </button>
          <button
            v-if="callsOn"
            class="btn-tonal"
            @click="callTo(selected, 'video')"
          >
            <span class="material-symbols-outlined">videocam</span>
            <span class="hide-narrow">Видео</span>
          </button>
          <button
            v-if="callsOn"
            class="btn-tonal tertiary"
            @click="callTo(selected, 'audio')"
          >
            <span class="material-symbols-outlined">call</span>
            <span class="hide-narrow">Аудио</span>
          </button>
        </div>
      </div>
    </Dialog>

    <AvatarLightbox
      v-if="selected"
      v-model="lightboxOpen"
      :src="avatarOf(selected)"
      :alt="selected.fio"
      :caption="selected.fio"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, h } from 'vue'
import { useRouter } from 'vue-router'
import Dialog from 'primevue/dialog'
import { getDirectory, getUsers } from '@/api/users.js'
import { useAuthStore } from '@/stores/auth.js'
import { useCompaniesStore } from '@/stores/companies.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { useCallStore } from '@/stores/call.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { ROLE_NAMES } from '@/composables/usePermission.js'
import { formatLastSeen } from '@/utils/presence.js'
import AvatarLightbox from '@/components/common/AvatarLightbox.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import PortalHubTabs from '@/components/portal/PortalHubTabs.vue'
import { useBreakpoint } from '@/composables/useBreakpoint.js'

const { isMobile } = useBreakpoint()
const pageTitle = computed(() => (isMobile.value ? 'Люди' : 'Сотрудники'))

const router = useRouter()
const auth = useAuthStore()
const companies = useCompaniesStore()
const messenger = useMessengerStore()
const callStore = useCallStore()
const notif = useNotificationsStore()

const users = ref([])
const search = ref('')
const roleFilter = ref('all')

const profileOpen = ref(false)
const selected = ref(null)
const lightboxOpen = ref(false)

// Раздел только для просмотра: открыть карточку, написать, позвонить. Управление
// сотрудниками (создание/роли/удаление) — в разделе «Компании».
const callsOn = computed(() => true)

async function load() {
  try {
    // Супер-админ — все пользователи платформы; член компании — каталог
    // своей активной компании (компания берётся из токена на бэке).
    if (auth.isSuperAdmin) {
      users.value = await getUsers()
    } else {
      users.value = await getDirectory()
    }
  } catch (e) {
    notif.error(e?.message || 'Не удалось загрузить сотрудников')
  }
}

onMounted(() => {
  load()
  messenger.fetchPresence()
  if (auth.isSuperAdmin) companies.load()
})

// Пользователь сменил активную компанию (switchCompany меняет auth.companyId) —
// перезагружаем каталог членов новой компании.
watch(() => auth.companyId, () => {
  if (!auth.isSuperAdmin) load()
})

function statusOf(u) {
  if (messenger.isOnline(u.id)) return 'в сети'
  return formatLastSeen(messenger.lastSeenOf(u.id, u.last_seen_at))
}

function avatarOf(u) {
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

function presenceClass(u) {
  return {
    online: messenger.isOnline(u.id),
    offline: !messenger.isOnline(u.id),
  }
}

function companyOf(u) {
  if (u.company?.name) return u.company.name
  const c = companies.items.find(c => c.id === u.company_id)
  return c?.name || null
}

function fmtPhone(p) {
  if (!p || !p.startsWith('+7') || p.length !== 12) return p
  const d = p.slice(2)
  return `+7 (${d.slice(0, 3)}) ${d.slice(3, 6)}-${d.slice(6, 8)}-${d.slice(8, 10)}`
}

function pluralPeople(n) {
  const m10 = n % 10
  const m100 = n % 100
  if (m10 === 1 && m100 !== 11) return 'сотрудник'
  if ([2, 3, 4].includes(m10) && ![12, 13, 14].includes(m100)) return 'сотрудника'
  return 'сотрудников'
}

// Каталог уже отфильтрован на бэке (члены активной компании / все для супер-админа).
const scopedUsers = computed(() => users.value)

const onlineCount = computed(() =>
  scopedUsers.value.reduce((s, u) => s + (messenger.isOnline(u.id) ? 1 : 0), 0)
)

const roleFilters = computed(() => {
  const counters = new Map()
  for (const u of scopedUsers.value) {
    const lvl = u.role?.level
    if (lvl == null) continue
    counters.set(lvl, (counters.get(lvl) || 0) + 1)
  }
  const names = {
    1: 'Сотрудники',
    2: 'Менеджеры',
    3: 'Администраторы',
  }
  const items = [{ key: 'all', label: 'Все', icon: 'groups', count: scopedUsers.value.length }]
  for (const [lvl, count] of [...counters.entries()].sort((a, b) => a[0] - b[0])) {
    items.push({ key: String(lvl), label: names[lvl] || `Уровень ${lvl}`, count })
  }
  return items
})

const filtered = computed(() => {
  const q = search.value.toLowerCase()
  let arr = scopedUsers.value
  if (roleFilter.value !== 'all') {
    const lvl = Number(roleFilter.value)
    arr = arr.filter(u => u.role?.level === lvl)
  }
  if (q) {
    arr = arr.filter(u =>
      (u.fio || '').toLowerCase().includes(q) ||
      (u.login || '').toLowerCase().includes(q) ||
      (u.post || '').toLowerCase().includes(q),
    )
  }
  return arr
})

function openProfile(u) { selected.value = u; profileOpen.value = true }

async function writeTo(u) {
  profileOpen.value = false
  const cid = await messenger.openWith(u.id)
  router.push(`/messenger/${cid}`)
}

async function callTo(u, media) {
  profileOpen.value = false
  try { await callStore.startCall({ userIds: [u.id], media }) }
  catch { /* ошибка в store */ }
}

watch(profileOpen, (open) => {
  if (!open) { selected.value = null; lightboxOpen.value = false }
})

const RolePill = {
  props: ['level', 'name'],
  setup(p) {
    return () => {
      // Роль есть только у члена компании; у прочих (например в платформенном
      // списке супер-админа) роли нет — пилюлю не показываем.
      if (!p.level && !p.name) return null
      return h('span', {
        class: ['role-pill', `lvl-${p.level || 1}`],
      }, p.name || ROLE_NAMES[p.level] || 'Сотрудник')
    }
  },
}
</script>

<style scoped>
/* ============ Шапка страницы ============ */
.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}
.page-head-text { min-width: 0; }
.page-head-title {
  margin: 0 0 6px;
  font-size: 24px;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--color-text);
}
.page-head-meta {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 13px;
  color: var(--color-text-dim);
}
.page-head-meta .meta-stat {
  background: var(--color-surface-high);
  color: var(--color-text);
}
.page-head-meta .meta-stat.online {
  background: color-mix(in oklch, var(--color-success) 18%, transparent);
  color: var(--color-text);
}

/* Геометрия вкладок хаба синхронизирована с PortalView (отступ — gap
   admin-sticky): переключение «Лента ⇄ Сотрудники» не сдвигает интерфейс. */
.emp-hub-tabs { align-self: flex-start; }

.admin-body { animation: emp-fade 0.2s ease; }
@keyframes emp-fade {
  from { opacity: 0; transform: translateY(4px); }
  to { opacity: 1; transform: translateY(0); }
}
@media (prefers-reduced-motion: reduce) { .admin-body { animation: none; } }

.emp-search {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 44px;
  padding: 0 10px 0 14px;
  background: var(--color-surface-high);
  border: 1px solid transparent;
  border-radius: var(--radius-full);
  flex: 1 1 280px;
  max-width: 540px;
  min-width: 0;
  transition: border-color .12s, background .12s, box-shadow .12s;
}
.emp-search:focus-within {
  background: var(--color-surface);
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--color-primary) 18%, transparent);
}
.emp-search > .material-symbols-outlined {
  color: var(--color-text-dim);
  font-size: 20px;
}
.emp-search input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  color: var(--color-text);
  font: inherit;
  min-width: 0;
}
.search-clear {
  border: none;
  background: var(--color-surface-highest);
  width: 26px;
  height: 26px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  cursor: pointer;
  color: var(--color-text-dim);
  transition: background .12s, color .12s;
}
.search-clear:hover { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.search-clear .material-symbols-outlined { font-size: 14px; }

.emp-chips {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: nowrap;
  overflow-x: auto;
  scrollbar-width: thin;
  scroll-snap-type: x proximity;
  padding-bottom: 2px;
  max-width: 100%;
  min-width: 0;
  flex: 0 1 auto;
}
.emp-chips::-webkit-scrollbar { height: 4px; }
.emp-chips::-webkit-scrollbar-thumb { background: var(--color-outline-dim); border-radius: 999px; }

.chip {
  appearance: none;
  border: 1px solid var(--color-outline-dim);
  background: transparent;
  color: var(--color-text);
  padding: 8px 14px;
  height: 36px;
  border-radius: var(--radius-full);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  white-space: nowrap;
  scroll-snap-align: start;
  transition: background .12s, color .12s, border-color .12s, box-shadow .12s;
}
.chip:hover { background: var(--color-surface-high); }
.chip .material-symbols-outlined { font-size: 18px; opacity: 0.8; }
.chip-count {
  min-width: 18px;
  padding: 0 6px;
  height: 18px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  color: var(--color-text-dim);
  font-size: 11px;
  font-weight: 700;
  display: inline-grid;
  place-items: center;
}
.chip.active {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
  border-color: transparent;
  box-shadow: var(--shadow-sm);
}
.chip.active .chip-count {
  background: color-mix(in oklch, var(--color-on-secondary-container) 18%, transparent);
  color: var(--color-on-secondary-container);
}

/* ============ Сетка карточек ============ */
.emp-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(212px, 1fr));
  gap: 16px;
}
.emp-card {
  position: relative;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-xl);
  padding: 24px 16px 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  cursor: pointer;
  gap: 8px;
  color: var(--color-text);
  outline: none;
  transition: border-color .16s, transform .16s, box-shadow .16s, background .16s;
  overflow: hidden;
}
.emp-card:hover,
.emp-card:focus-visible {
  border-color: transparent;
  background: var(--color-surface-high);
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}
.emp-card:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}
.emp-card.is-me { border-color: var(--color-primary); }

.emp-card-avatar-wrap {
  position: relative;
  margin-bottom: 4px;
}
.avatar-lg { width: 88px; height: 88px; }

.emp-card-name {
  font-size: 15px;
  font-weight: 700;
  line-height: 1.25;
  margin: 0;
  color: var(--color-text);
  display: inline-flex;
  align-items: center;
  gap: 6px;
  justify-content: center;
  flex-wrap: wrap;
}
.emp-card-post {
  font-size: 12.5px;
  color: var(--color-text-dim);
  margin: 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}
.emp-card-status {
  font-size: 11.5px;
  color: var(--color-text-dim);
  margin: 4px 0 0;
}
.emp-card-status.on {
  color: var(--color-success);
  font-weight: 700;
}

.emp-card-actions {
  position: absolute;
  left: 8px;
  right: 8px;
  bottom: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 6px;
  border-radius: var(--radius-lg);
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  box-shadow: var(--shadow-md);
  opacity: 0;
  transform: translateY(8px);
  pointer-events: none;
  transition: opacity .18s, transform .18s;
}
.emp-card:hover .emp-card-actions,
.emp-card:focus-within .emp-card-actions {
  opacity: 1;
  transform: translateY(0);
  pointer-events: auto;
}
@media (hover: none) {
  .emp-card-actions {
    position: static;
    opacity: 1;
    transform: none;
    pointer-events: auto;
    margin-top: 6px;
    background: var(--color-surface-high);
    border: none;
    box-shadow: none;
  }
}
.card-act {
  appearance: none;
  border: none;
  background: transparent;
  width: 36px;
  height: 36px;
  border-radius: var(--radius-full);
  display: grid;
  place-items: center;
  cursor: pointer;
  color: var(--color-text);
  transition: background .12s, color .12s;
}
.card-act:hover {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}
.card-act .material-symbols-outlined { font-size: 20px; }

.root-corner {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  box-shadow: var(--shadow-sm);
  z-index: 1;
}
.root-corner .material-symbols-outlined { font-size: 16px; font-variation-settings: 'FILL' 1; }

.emp-grid-empty {
  grid-column: 1 / -1;
  background: var(--color-surface-high);
  border-radius: var(--radius-xl);
}

/* ============ Аватары с presence-ring ============ */
.avatar {
  position: relative;
  display: inline-grid;
  place-items: center;
  flex-shrink: 0;
  border-radius: 50%;
  isolation: isolate;
}
.avatar img {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  object-fit: cover;
  display: block;
}
.avatar-xl { width: 116px; height: 116px; }

.avatar::before {
  content: '';
  position: absolute;
  inset: -3px;
  border-radius: 50%;
  border: 2px solid var(--color-outline-dim);
  z-index: -1;
  transition: border-color .18s, box-shadow .18s;
}
.avatar-xl::before { inset: -5px; border-width: 4px; }
.avatar.online::before {
  border-color: var(--color-success);
  box-shadow: 0 0 0 2px color-mix(in oklch, var(--color-success) 22%, transparent);
}

.me-tag {
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  flex-shrink: 0;
}
.root-badge {
  display: inline-grid;
  place-items: center;
  width: 18px;
  height: 18px;
  color: var(--color-tertiary);
  flex-shrink: 0;
}
.root-badge .material-symbols-outlined { font-size: 18px; font-variation-settings: 'FILL' 1; }
.root-badge.inline {
  width: 22px;
  height: 22px;
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  border-radius: 50%;
  margin-left: 4px;
}
.root-badge.inline .material-symbols-outlined { font-size: 14px; }

/* ============ Role pill ============ */
.role-pill {
  display: inline-flex;
  align-items: center;
  padding: 3px 12px;
  border-radius: var(--radius-full);
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  line-height: 1.4;
}
.role-pill.lvl-1 { background: var(--color-surface-high); color: var(--color-text-dim); }
.role-pill.lvl-2 { background: var(--color-secondary-container); color: var(--color-on-secondary-container); }
.role-pill.lvl-3 { background: var(--color-tertiary-container); color: var(--color-on-tertiary-container); }

/* ============ Profile dialog ============ */
.emp-profile {
  display: flex;
  flex-direction: column;
  background: var(--color-surface);
  width: 100%;
  box-sizing: border-box;
  position: relative;
}
.profile-close {
  position: absolute;
  top: 12px;
  right: 12px;
  z-index: 2;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: none;
  background: color-mix(in oklch, var(--color-surface) 60%, transparent);
  color: var(--color-text-dim);
  display: grid;
  place-items: center;
  cursor: pointer;
  backdrop-filter: blur(8px);
  transition: background .12s, color .12s;
}
.profile-close:hover {
  background: var(--color-surface);
  color: var(--color-text);
}
.profile-close .material-symbols-outlined { font-size: 20px; }

/* Пастельная обложка, затухающая книзу — в стиле карточки в разделе аккаунта. */
.profile-cover {
  position: absolute;
  inset: 0 0 auto;
  height: 150px;
  background:
    radial-gradient(120% 140% at 85% 0%,
      color-mix(in oklch, var(--color-tertiary-container) 40%, transparent) 0%,
      transparent 60%),
    linear-gradient(120deg,
      color-mix(in oklch, var(--color-primary-container) 55%, var(--color-surface)),
      color-mix(in oklch, var(--color-secondary-container) 55%, var(--color-surface)));
  -webkit-mask-image: linear-gradient(to bottom, black 30%, transparent 100%);
  mask-image: linear-gradient(to bottom, black 30%, transparent 100%);
  pointer-events: none;
}

.profile-hero {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 36px 22px 22px;
  gap: 10px;
  color: var(--color-text);
}
.profile-avatar-btn {
  appearance: none;
  border: none;
  background: transparent;
  padding: 0;
  cursor: zoom-in;
  margin-bottom: 4px;
}
.profile-name {
  margin: 0;
  font-size: 22px;
  font-weight: 800;
  line-height: 1.2;
  letter-spacing: -0.01em;
  color: var(--color-text);
  word-break: break-word;
  overflow-wrap: anywhere;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  justify-content: center;
  flex-wrap: wrap;
}
.profile-tags {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 4px;
}
.profile-status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 600;
  background: color-mix(in oklch, var(--color-text) 8%, transparent);
  color: var(--color-text-dim);
}
.profile-status .status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-outline-dim);
}
.profile-status.on {
  background: color-mix(in oklch, var(--color-success) 22%, transparent);
}
.profile-status.on .status-dot { background: var(--color-success); }

.profile-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 16px;
  background: var(--color-surface);
}
.profile-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: var(--radius-lg);
  text-decoration: none;
  color: var(--color-text);
  background: var(--color-surface-low);
  transition: background .12s;
}
.profile-row.link { cursor: pointer; }
.profile-row.link:hover { background: var(--color-surface-high); }
.row-ico {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  display: grid;
  place-items: center;
  flex-shrink: 0;
}
.row-ico[data-tone="primary"] {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}
.row-ico[data-tone="secondary"] {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}
.row-ico[data-tone="tertiary"] {
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}
.row-ico .material-symbols-outlined { font-size: 20px; }

.row-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.row-label {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-text-dim);
}
.row-value {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.row-chev {
  font-size: 18px;
  color: var(--color-text-dim);
  flex-shrink: 0;
}

.profile-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 0 16px 16px;
}
.profile-actions > * {
  flex: 1 1 120px;
  justify-content: center;
}

/* ============ Кнопки ============ */
.btn-filled, .btn-tonal {
  appearance: none;
  border: none;
  cursor: pointer;
  border-radius: var(--radius-full);
  padding: 10px 18px;
  font: inherit;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: background .12s, color .12s, border-color .12s, box-shadow .12s, transform .12s;
}
.btn-filled {
  background: var(--color-primary);
  color: var(--color-on-primary);
  box-shadow: var(--shadow-sm);
}
.btn-filled:hover { background: var(--color-primary-hover); }
.btn-filled .material-symbols-outlined { font-size: 18px; }
.btn-tonal {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}
.btn-tonal.tertiary {
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}
.btn-tonal:hover { filter: brightness(.96); }
.btn-tonal .material-symbols-outlined { font-size: 18px; }

@media (max-width: 768px) {
  .hide-narrow { display: none; }

  .page-head-title { font-size: 20px; }
  .page-head-meta { font-size: 12px; }
  .meta-dot { display: none; }

  .emp-search { flex: 1 1 100%; max-width: 100%; }

  /* Сетка карточек 2 колонки. */
  .emp-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 10px;
  }
  .emp-card {
    padding: 16px 10px 12px;
    gap: 6px;
  }
  .avatar-lg { width: 64px; height: 64px; }
  .emp-card-name { font-size: 13.5px; }
  .emp-card-post { font-size: 11.5px; }
  .emp-card-status { font-size: 10.5px; }

  /* На тач-экранах actions всегда видны под карточкой. */
  .emp-card-actions {
    position: static;
    opacity: 1;
    transform: none;
    pointer-events: auto;
    margin-top: 4px;
    padding: 0;
    background: transparent;
    border: none;
    box-shadow: none;
  }
  .card-act { width: 34px; height: 34px; }
  .card-act .material-symbols-outlined { font-size: 18px; }

  /* Профиль-модалка адаптивная. */
  .profile-list { padding: 12px; }
  .profile-actions { padding-left: 12px; padding-right: 12px; }
}

@media (max-width: 420px) {
  /* Очень узкие — 1 колонка, чтобы ФИО не обрезалось. */
  .emp-grid { grid-template-columns: 1fr; }
}

</style>
