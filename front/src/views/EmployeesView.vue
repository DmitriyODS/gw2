<template>
  <div class="admin-page">
    <header class="admin-sticky">
      <!-- Тулбар одной строкой (как в «Ленте»): вкладки хаба, поиск, статы, фильтры. -->
      <div class="admin-toolbar">
        <PortalHubTabs class="emp-hub-tabs" />
        <SearchField v-model="search" placeholder="Поиск по ФИО, логину, должности" hotkey />

        <span class="chip-tint chip-tint--primary emp-stat">
          <span class="material-symbols-outlined">groups</span>
          <strong>{{ scopedUsers.length }}</strong>&nbsp;{{ pluralPeople(scopedUsers.length) }}
        </span>
        <span class="chip-tint chip-tint--success emp-stat">
          <span class="presence-pulse" />
          <strong>{{ onlineCount }}</strong>&nbsp;в сети
        </span>

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

    <!-- Профиль сотрудника — общий компонент (используется и порталом). -->
    <EmployeeProfileDialog v-model="profileOpen" :user="selected" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { getDirectory, getUsers } from '@/api/users.js'
import { useAuthStore } from '@/stores/auth.js'
import { useCompaniesStore } from '@/stores/companies.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { useCallStore } from '@/stores/call.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { formatLastSeen } from '@/utils/presence.js'
import EmployeeProfileDialog from '@/components/common/EmployeeProfileDialog.vue'
import SearchField from '@/components/common/SearchField.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import RolePill from '@/components/common/RolePill.vue'
import PortalHubTabs from '@/components/portal/PortalHubTabs.vue'
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
  const q = search.value.trim().toLowerCase()
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
  if (!open) selected.value = null
})
</script>

<style scoped>
/* Тулбар без подложки — прозрачная «плавающая» шапка как в «Задачах»
   (и как во второй вкладке хаба — «Ленте»). */
.admin-sticky { background: transparent; backdrop-filter: none; -webkit-backdrop-filter: none; }
.admin-sticky::after { display: none; }

/* Вкладки хаба и статы — самостоятельные элементы одной строки тулбара. */
.emp-hub-tabs { flex-shrink: 0; }
.emp-stat { flex-shrink: 0; white-space: nowrap; }
/* Поиск не сжимается меньше комфортной ширины — при нехватке места первыми
   переносятся статы/фильтры, а не поле ввода. */
.admin-toolbar :deep(.search-field) { min-width: 240px; }

.admin-body { animation: emp-fade 0.2s ease; }
@keyframes emp-fade {
  from { opacity: 0; transform: translateY(4px); }
  to { opacity: 1; transform: translateY(0); }
}
@media (prefers-reduced-motion: reduce) { .admin-body { animation: none; } }


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
  background: var(--acrylic-card-bg);
  border: 1px solid var(--acrylic-border);
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
  background: var(--acrylic-card-bg);
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

.avatar::before {
  content: '';
  position: absolute;
  inset: -3px;
  border-radius: 50%;
  border: 2px solid var(--color-outline-dim);
  z-index: -1;
  transition: border-color .18s, box-shadow .18s;
}
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
@media (max-width: 768px) {
  /* На мобильном поиск занимает всю ширину — min-width снимаем. */
  .admin-toolbar :deep(.search-field) { flex: 1 1 100%; max-width: 100%; min-width: 0; }

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
}

@media (max-width: 420px) {
  /* Очень узкие — 1 колонка, чтобы ФИО не обрезалось. */
  .emp-grid { grid-template-columns: 1fr; }
}

</style>
