<template>
  <div class="employees-view">
    <header class="emp-header">
      <div class="emp-title-row">
        <h1 class="emp-title">Сотрудники</h1>
        <div class="emp-title-actions">
          <CompanySelect v-if="auth.isRootAdmin" />
          <button v-if="canCreate" class="btn-filled" @click="openCreate">
            <span class="material-symbols-rounded">person_add</span>
            Добавить
          </button>
        </div>
      </div>
      <div class="emp-controls">
        <div class="emp-search">
          <span class="material-symbols-rounded">search</span>
          <input v-model.trim="search" placeholder="Поиск по ФИО или логину" />
          <button v-if="search" class="search-clear" @click="search = ''" aria-label="Очистить">
            <span class="material-symbols-rounded">close</span>
          </button>
        </div>
        <div v-if="auth.isRootAdmin" class="view-toggle">
          <button :class="['vt-btn', { active: view === 'cards' }]" @click="view = 'cards'" title="Карточки">
            <span class="material-symbols-rounded">grid_view</span>
          </button>
          <button :class="['vt-btn', { active: view === 'table' }]" @click="view = 'table'" title="Таблица">
            <span class="material-symbols-rounded">list</span>
          </button>
        </div>
      </div>
    </header>

    <div v-if="loading" class="emp-loading">
      <ProgressSpinner />
    </div>

    <div v-else-if="!filtered.length" class="emp-empty">
      <div class="empty-icon">
        <span class="material-symbols-rounded">{{ search ? 'search_off' : 'person_off' }}</span>
      </div>
      <h3>{{ search ? 'Никого не нашли' : 'Сотрудников пока нет' }}</h3>
      <p v-if="!search && canCreate">Добавьте первого — кнопка справа сверху.</p>
    </div>

    <!-- Таблица (для Админа системы) -->
    <div v-else-if="view === 'table'" class="emp-table-wrap">
      <table class="emp-table">
        <thead>
          <tr>
            <th @click="setSort('fio')">ФИО <SortIcon col="fio" :sort="sort" /></th>
            <th @click="setSort('login')">Логин <SortIcon col="login" :sort="sort" /></th>
            <th>Должность</th>
            <th>Роль</th>
            <th>Компания</th>
            <th>Статус</th>
            <th class="th-act"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in filtered" :key="u.id">
            <td>
              <div class="cell-user" @click="openProfile(u)">
                <div class="avatar-wrap small">
                  <img :src="avatarOf(u)" :alt="u.fio" />
                  <span v-if="messenger.isOnline(u.id)" class="online-dot"></span>
                </div>
                <span class="user-fio">{{ u.fio }}</span>
                <span v-if="u.is_root_admin" class="root-badge" title="Корневой Администратор системы">
                  <span class="material-symbols-rounded">verified</span>
                </span>
              </div>
            </td>
            <td class="td-mono">@{{ u.login }}</td>
            <td>{{ u.post || '—' }}</td>
            <td><RolePill :level="u.role?.level" :name="u.role?.name" /></td>
            <td>
              <span v-if="companyOf(u)" class="company-chip">{{ companyOf(u) }}</span>
              <span v-else class="muted">—</span>
            </td>
            <td>
              <span :class="['status', messenger.isOnline(u.id) ? 'on' : 'off']">
                {{ statusOf(u) }}
              </span>
            </td>
            <td class="td-act">
              <button v-if="canEdit(u)" class="icon-btn" title="Редактировать" @click="openEdit(u)">
                <span class="material-symbols-rounded">edit</span>
              </button>
              <button v-if="canDelete(u)" class="icon-btn danger" title="Скрыть" @click="askDelete(u)">
                <span class="material-symbols-rounded">delete</span>
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Карточки -->
    <div v-else class="emp-grid">
      <button
        v-for="u in filtered" :key="u.id"
        class="emp-card" @click="openProfile(u)"
      >
        <div class="avatar-wrap big">
          <img :src="avatarOf(u)" :alt="u.fio" />
          <span v-if="messenger.isOnline(u.id)" class="online-dot"></span>
          <span v-if="u.is_root_admin" class="root-badge corner" title="Корневой Администратор системы">
            <span class="material-symbols-rounded">verified</span>
          </span>
        </div>
        <div class="card-name">{{ u.fio }}</div>
        <div class="card-post">{{ u.post || '—' }}</div>
        <RolePill :level="u.role?.level" :name="u.role?.name" />
        <div :class="['card-status', { on: messenger.isOnline(u.id) }]">
          {{ statusOf(u) }}
        </div>
      </button>
    </div>

    <!-- Профиль (модалка) -->
    <Dialog
      v-model:visible="profileOpen"
      modal
      :draggable="false"
      :show-header="false"
      :style="{ width: '440px', maxWidth: 'calc(100vw - 24px)' }"
      :pt="{ root: { class: 'emp-dialog' }, content: { style: 'overflow-x: hidden; padding: 0' } }"
    >
      <div v-if="selected" class="emp-profile">
        <button class="avatar-zoom" @click="lightboxOpen = true" aria-label="Открыть фото">
          <img class="profile-avatar" :src="avatarOf(selected)" :alt="selected.fio" />
          <span v-if="messenger.isOnline(selected.id)" class="online-dot profile-dot"></span>
        </button>
        <h2 class="profile-name">{{ selected.fio }}</h2>
        <div :class="['profile-status', { on: messenger.isOnline(selected.id) }]">
          {{ statusOf(selected) }}
        </div>
        <div class="profile-meta">
          <div v-if="selected.post" class="meta-line">
            <span class="material-symbols-rounded">badge</span>
            {{ selected.post }}
          </div>
          <div class="meta-line">
            <span class="material-symbols-rounded">alternate_email</span>
            @{{ selected.login }}
          </div>
          <div v-if="selected.phone" class="meta-line">
            <span class="material-symbols-rounded">phone</span>
            <a :href="`tel:${selected.phone}`">{{ fmtPhone(selected.phone) }}</a>
          </div>
          <div v-if="selected.email" class="meta-line">
            <span class="material-symbols-rounded">mail</span>
            <a :href="`mailto:${selected.email}`">{{ selected.email }}</a>
          </div>
          <div v-if="companyOf(selected)" class="meta-line">
            <span class="material-symbols-rounded">domain</span>
            {{ companyOf(selected) }}
          </div>
        </div>
        <RolePill :level="selected.role?.level" :name="selected.role?.name" />

        <div class="profile-actions">
          <button
            v-if="selected.id !== auth.user?.id"
            class="btn-filled"
            @click="writeTo(selected)"
          >
            <span class="material-symbols-rounded">chat</span> Написать
          </button>
          <button
            v-if="selected.id !== auth.user?.id && callsOn"
            class="btn-tonal"
            @click="callTo(selected, 'video')"
          >
            <span class="material-symbols-rounded">videocam</span>
            <span class="hide-narrow">Видео</span>
          </button>
          <button
            v-if="selected.id !== auth.user?.id && callsOn"
            class="btn-tonal tertiary"
            @click="callTo(selected, 'audio')"
          >
            <span class="material-symbols-rounded">call</span>
            <span class="hide-narrow">Аудио</span>
          </button>
        </div>
        <div v-if="canEdit(selected) || canDelete(selected)" class="profile-admin">
          <button v-if="canEdit(selected)" class="btn-outlined" @click="openEdit(selected)">
            <span class="material-symbols-rounded">edit</span> Редактировать
          </button>
          <button v-if="canDelete(selected)" class="btn-outlined danger" @click="askDelete(selected)">
            <span class="material-symbols-rounded">delete</span> Скрыть
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

    <EmployeeFormDialog
      ref="formDlgRef"
      v-model="formOpen"
      :user="editTarget"
      :roles="roles"
      @save="onSave"
    />

    <ConfirmDialog
      :visible="deleteDlg.open"
      header="Скрыть сотрудника"
      :message="`Скрыть сотрудника «${deleteDlg.user?.fio}»? Доступ в систему пропадёт, история работы сохранится.`"
      confirm-label="Скрыть"
      danger-confirm
      @confirm="doDelete"
      @cancel="deleteDlg.open = false"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, h } from 'vue'
import { useRouter } from 'vue-router'
import Dialog from 'primevue/dialog'
import ProgressSpinner from 'primevue/progressspinner'
import {
  getDirectory, getUsers, createUser, updateUser, deleteUser, assignRole,
} from '@/api/users.js'
import { getRoles } from '@/api/roles.js'
import { useAuthStore } from '@/stores/auth.js'
import { useCompaniesStore } from '@/stores/companies.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { useCallStore } from '@/stores/call.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { usePermission } from '@/composables/usePermission.js'
import { formatLastSeen } from '@/utils/presence.js'
import CompanySelect from '@/components/common/CompanySelect.vue'
import AvatarLightbox from '@/components/common/AvatarLightbox.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmployeeFormDialog from '@/components/employees/EmployeeFormDialog.vue'

const router = useRouter()
const auth = useAuthStore()
const companies = useCompaniesStore()
const messenger = useMessengerStore()
const callStore = useCallStore()
const notif = useNotificationsStore()
const { isAtLeast, ROLES, myLevel } = usePermission()

const users = ref([])
const roles = ref([])
const loading = ref(false)
const search = ref('')
const view = ref(auth.isRootAdmin ? 'cards' : 'cards')

const profileOpen = ref(false)
const selected = ref(null)
const lightboxOpen = ref(false)

const formOpen = ref(false)
const editTarget = ref(null)
const formDlgRef = ref(null)

const deleteDlg = ref({ open: false, user: null })
const sort = ref({ col: 'fio', dir: 'asc' })

const canCreate = computed(() => isAtLeast(ROLES.DIRECTOR))
const callsOn = computed(() => {
  if (auth.isRootAdmin) return companies.activeCompany?.settings?.uses_calls !== false
  return true
})

function canEdit(u) {
  if (!isAtLeast(ROLES.DIRECTOR)) return false
  if (!auth.isRootAdmin && u.company_id !== auth.companyId) return false
  // Не давать редактировать пользователя с ролью выше своей.
  if ((u.role?.level ?? 0) > myLevel()) return false
  return true
}

function canDelete(u) {
  if (!canEdit(u)) return false
  if (u.id === auth.user?.id) return false
  if (u.is_root_admin) return false
  return true
}

async function load() {
  loading.value = true
  try {
    // Админу системы нужны полные карточки (с phone/email/is_root_admin),
    // а обычные роли получают каталог через /directory.
    if (auth.isRootAdmin) {
      users.value = await getUsers()
    } else {
      users.value = await getDirectory()
    }
  } catch (e) {
    notif.error(e?.message || 'Не удалось загрузить сотрудников')
  } finally {
    loading.value = false
  }
}

async function loadRoles() {
  try { roles.value = await getRoles() }
  catch {/* без ролей не сможем редактировать, но просмотр работает */}
}

onMounted(() => {
  load()
  loadRoles()
  messenger.fetchPresence()
  if (auth.isRootAdmin) companies.load()
})

// Перезагружаем при смене активной компании (для Админа системы — он
// получит сотрудников только выбранной компании, т.к. /users тоже надо
// фильтровать; см. ниже filtered).
watch(() => companies.activeCompanyId, () => {
  if (auth.isRootAdmin) load()
})

function statusOf(u) {
  if (messenger.isOnline(u.id)) return 'в сети'
  return formatLastSeen(messenger.lastSeenOf(u.id, u.last_seen_at))
}

function avatarOf(u) {
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
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

const filtered = computed(() => {
  const q = search.value.toLowerCase()
  // Для Админа системы доступна фильтрация по выбранной компании.
  const cid = auth.isRootAdmin ? companies.activeCompanyId : auth.companyId
  let arr = users.value
  if (cid != null) arr = arr.filter(u => u.company_id === cid)
  if (q) {
    arr = arr.filter(u =>
      (u.fio || '').toLowerCase().includes(q) ||
      (u.login || '').toLowerCase().includes(q) ||
      (u.post || '').toLowerCase().includes(q),
    )
  }
  if (view.value === 'table') {
    const sign = sort.value.dir === 'asc' ? 1 : -1
    arr = [...arr].sort((a, b) =>
      sign * String(a[sort.value.col] || '').localeCompare(String(b[sort.value.col] || ''), 'ru'),
    )
  }
  return arr
})

function setSort(col) {
  if (sort.value.col === col) sort.value.dir = sort.value.dir === 'asc' ? 'desc' : 'asc'
  else sort.value = { col, dir: 'asc' }
}

function openProfile(u) { selected.value = u; profileOpen.value = true }

async function writeTo(u) {
  profileOpen.value = false
  const cid = await messenger.openWith(u.id)
  router.push(`/messenger/${cid}`)
}

async function callTo(u, media) {
  profileOpen.value = false
  try { await callStore.startCall({ userIds: [u.id], media }) }
  catch {/* error в store */}
}

function openCreate() {
  editTarget.value = null
  formOpen.value = true
}

function openEdit(u) {
  editTarget.value = u
  profileOpen.value = false
  formOpen.value = true
}

async function onSave({ payload, isEdit, userId, newRoleId }) {
  try {
    let saved
    if (isEdit) {
      saved = await updateUser(userId, payload)
      if (newRoleId) {
        saved = await assignRole(userId, { role_id: newRoleId })
      }
      _replace(saved)
      notif.success('Сотрудник обновлён')
    } else {
      saved = await createUser(payload)
      users.value.push(saved)
      notif.success('Сотрудник создан')
    }
    formOpen.value = false
  } catch (e) {
    const msg = typeof e?.message === 'string' ? e.message : 'Не удалось сохранить'
    formDlgRef.value?.showError(msg)
  } finally {
    formDlgRef.value?.finish()
  }
}

function _replace(u) {
  const idx = users.value.findIndex(x => x.id === u.id)
  if (idx >= 0) users.value.splice(idx, 1, u)
  if (selected.value?.id === u.id) selected.value = u
}

function askDelete(u) { deleteDlg.value = { open: true, user: u } }

async function doDelete() {
  if (!deleteDlg.value.user) return
  try {
    await deleteUser(deleteDlg.value.user.id)
    users.value = users.value.filter(x => x.id !== deleteDlg.value.user.id)
    notif.success('Сотрудник скрыт')
    deleteDlg.value.open = false
    if (profileOpen.value && selected.value?.id === deleteDlg.value.user.id) {
      profileOpen.value = false
    }
  } catch (e) {
    notif.error(e?.message || 'Не удалось скрыть')
  }
}

watch(profileOpen, (open) => { if (!open) { selected.value = null; lightboxOpen.value = false } })

const SortIcon = {
  props: ['col', 'sort'],
  setup(p) {
    return () => {
      const active = p.sort.col === p.col
      const ic = active ? (p.sort.dir === 'asc' ? 'arrow_upward' : 'arrow_downward') : 'unfold_more'
      return h('span', { class: ['sort-ic', { active }] }, [
        h('span', { class: 'material-symbols-rounded' }, ic),
      ])
    }
  },
}

const RolePill = {
  props: ['level', 'name'],
  setup(p) {
    return () => h('span', {
      class: ['role-pill', `lvl-${p.level || 1}`],
    }, p.name || 'Сотрудник')
  },
}
</script>

<style scoped>
.employees-view {
  padding: 24px;
  max-width: 1280px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.emp-header { display: flex; flex-direction: column; gap: 14px; }
.emp-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.emp-title { font-size: 28px; font-weight: 700; margin: 0; color: var(--color-on-surface); }
.emp-title-actions { display: inline-flex; gap: 10px; align-items: center; flex-wrap: wrap; }

.emp-controls {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.emp-search {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 40px;
  padding: 0 10px 0 12px;
  background: var(--color-surface-container);
  border: 1px solid transparent;
  border-radius: var(--radius-full, 999px);
  flex: 1 1 280px;
  max-width: 480px;
  transition: border-color .12s, background .12s;
}
.emp-search:focus-within {
  background: var(--color-surface);
  border-color: var(--color-primary);
}
.emp-search .material-symbols-rounded { color: var(--color-on-surface-variant); font-size: 20px; }
.emp-search input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  color: var(--color-on-surface);
  font: inherit;
  min-width: 0;
}
.search-clear {
  border: none;
  background: var(--color-surface-high);
  width: 24px;
  height: 24px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  cursor: pointer;
  color: var(--color-on-surface-variant);
}
.search-clear .material-symbols-rounded { font-size: 14px; }

.view-toggle {
  display: inline-flex;
  padding: 3px;
  background: var(--color-surface-container);
  border-radius: var(--radius-full, 999px);
  gap: 2px;
}
.vt-btn {
  appearance: none;
  border: none;
  background: transparent;
  width: 34px;
  height: 34px;
  display: grid;
  place-items: center;
  border-radius: 999px;
  cursor: pointer;
  color: var(--color-on-surface-variant);
}
.vt-btn.active { background: var(--color-primary); color: var(--color-on-primary); }
.vt-btn .material-symbols-rounded { font-size: 18px; }

.emp-loading { display: grid; place-items: center; min-height: 240px; }

.emp-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 48px 20px;
  background: var(--color-surface-container);
  border-radius: var(--radius-lg, 16px);
  text-align: center;
}
.empty-icon {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  display: grid;
  place-items: center;
}
.empty-icon .material-symbols-rounded { font-size: 32px; }
.emp-empty h3 { margin: 0; color: var(--color-on-surface); }
.emp-empty p { margin: 0; color: var(--color-on-surface-variant); font-size: 14px; }

/* Карточки */
.emp-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(212px, 1fr));
  gap: 14px;
}
.emp-card {
  appearance: none;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-variant);
  border-radius: var(--radius-lg, 16px);
  padding: 18px 14px 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  cursor: pointer;
  gap: 6px;
  color: var(--color-on-surface);
  transition: border-color .12s, transform .12s, box-shadow .12s;
}
.emp-card:hover {
  border-color: var(--color-primary);
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}
.card-name { font-size: 15px; font-weight: 600; }
.card-post { font-size: 13px; color: var(--color-on-surface-variant); }
.card-status { font-size: 11px; color: var(--color-on-surface-variant); margin-top: 2px; }
.card-status.on { color: var(--color-primary); font-weight: 600; }

.avatar-wrap { position: relative; display: inline-block; }
.avatar-wrap.big img { width: 88px; height: 88px; border: 2px solid var(--color-outline-variant); margin-bottom: 8px; }
.avatar-wrap.small img { width: 36px; height: 36px; border: 1.5px solid var(--color-outline-variant); }
.avatar-wrap img { border-radius: 50%; object-fit: cover; display: block; }

.online-dot {
  position: absolute;
  right: 2px;
  bottom: 2px;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: var(--color-success);
  border: 2.5px solid var(--color-surface);
}
.avatar-wrap.big .online-dot { right: 4px; bottom: 10px; width: 16px; height: 16px; }

.root-badge {
  display: inline-grid;
  place-items: center;
  width: 18px;
  height: 18px;
  color: var(--color-tertiary);
}
.root-badge .material-symbols-rounded { font-size: 18px; font-variation-settings: 'FILL' 1; }
.root-badge.corner {
  position: absolute;
  top: 0;
  right: 0;
  width: 24px;
  height: 24px;
  background: var(--color-tertiary-container);
  border-radius: 50%;
  border: 2px solid var(--color-surface);
}
.root-badge.corner .material-symbols-rounded { font-size: 14px; }

/* Роль-плашка */
.role-pill {
  display: inline-flex;
  align-items: center;
  padding: 2px 10px;
  border-radius: var(--radius-full, 999px);
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.role-pill.lvl-1 { background: var(--color-surface-high); color: var(--color-on-surface-variant); }
.role-pill.lvl-2 { background: var(--color-secondary-container); color: var(--color-on-secondary-container); }
.role-pill.lvl-3 { background: var(--color-tertiary-container); color: var(--color-on-tertiary-container); }
.role-pill.lvl-4 { background: var(--color-primary-container); color: var(--color-on-primary-container); }

/* Таблица */
.emp-table-wrap {
  background: var(--color-surface);
  border-radius: var(--radius-lg, 16px);
  border: 1px solid var(--color-outline-variant);
  overflow: hidden;
}
.emp-table { width: 100%; border-collapse: collapse; }
.emp-table thead { background: var(--color-surface-container); }
.emp-table th, .emp-table td {
  padding: 10px 14px;
  text-align: left;
  font-size: 13px;
  border-bottom: 1px solid var(--color-outline-variant);
  vertical-align: middle;
}
.emp-table th {
  font-weight: 600;
  color: var(--color-on-surface-variant);
  text-transform: uppercase;
  font-size: 11px;
  letter-spacing: 0.05em;
  cursor: pointer;
  user-select: none;
}
.emp-table tbody tr:last-child td { border-bottom: none; }
.emp-table tbody tr:hover { background: var(--color-surface-container); }
.cell-user { display: inline-flex; align-items: center; gap: 8px; cursor: pointer; }
.user-fio { font-weight: 600; color: var(--color-on-surface); }
.td-mono { color: var(--color-on-surface-variant); font-variant-numeric: tabular-nums; }
.company-chip {
  display: inline-block;
  padding: 2px 10px;
  background: var(--color-surface-container);
  border-radius: 999px;
  font-size: 12px;
}
.muted { color: var(--color-on-surface-variant); font-style: italic; }
.status.on { color: var(--color-primary); font-weight: 600; }
.status.off { color: var(--color-on-surface-variant); }
.th-act, .td-act { width: 96px; text-align: right; }
.td-act { display: flex; gap: 2px; justify-content: flex-end; }
.icon-btn {
  appearance: none;
  border: none;
  background: transparent;
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  color: var(--color-on-surface-variant);
  cursor: pointer;
  transition: background .12s, color .12s;
}
.icon-btn:hover { background: var(--color-surface-high); color: var(--color-on-surface); }
.icon-btn.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
.icon-btn .material-symbols-rounded { font-size: 18px; }

.sort-ic { display: inline-flex; vertical-align: middle; opacity: .4; margin-left: 2px; }
.sort-ic.active { opacity: 1; color: var(--color-primary); }
.sort-ic .material-symbols-rounded { font-size: 14px; }

/* Профиль */
.emp-profile {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  padding: 28px 22px 22px;
  gap: 8px;
  width: min(440px, calc(100vw - 24px));
  box-sizing: border-box;
}
.avatar-zoom {
  appearance: none;
  border: none;
  background: transparent;
  padding: 0;
  cursor: zoom-in;
  position: relative;
  margin-bottom: 6px;
}
.profile-avatar {
  width: 128px;
  height: 128px;
  border-radius: 50%;
  object-fit: cover;
  border: 3px solid var(--color-primary);
  display: block;
}
.profile-dot { right: 8px; bottom: 8px; width: 20px; height: 20px; }
.profile-name {
  font-size: 20px;
  font-weight: 700;
  margin: 0;
  color: var(--color-on-surface);
  word-break: break-word;
  overflow-wrap: anywhere;
}
.profile-status { font-size: 12px; color: var(--color-on-surface-variant); }
.profile-status.on { color: var(--color-primary); font-weight: 600; }

.profile-meta {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 12px 0 8px;
  width: 100%;
  align-items: flex-start;
}
.meta-line {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--color-on-surface);
}
.meta-line .material-symbols-rounded { font-size: 18px; color: var(--color-on-surface-variant); }
.meta-line a { color: var(--color-primary); text-decoration: none; }
.meta-line a:hover { text-decoration: underline; }

.profile-actions, .profile-admin {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 14px;
  width: 100%;
  justify-content: center;
}
.profile-admin {
  margin-top: 8px;
  padding-top: 12px;
  border-top: 1px solid var(--color-outline-variant);
}
.profile-actions > *, .profile-admin > * { flex: 1 1 120px; justify-content: center; }

/* Кнопки */
.btn-filled, .btn-tonal, .btn-outlined, .btn-text {
  appearance: none;
  border: none;
  cursor: pointer;
  border-radius: var(--radius-full, 999px);
  padding: 10px 18px;
  font: inherit;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.btn-filled { background: var(--color-primary); color: var(--color-on-primary); }
.btn-filled:hover { filter: brightness(.94); }
.btn-tonal { background: var(--color-secondary-container); color: var(--color-on-secondary-container); }
.btn-tonal.tertiary { background: var(--color-tertiary-container); color: var(--color-on-tertiary-container); }
.btn-tonal:hover { filter: brightness(.96); }
.btn-outlined {
  background: transparent;
  border: 1px solid var(--color-outline-variant);
  color: var(--color-on-surface);
}
.btn-outlined:hover { background: var(--color-surface-container); }
.btn-outlined.danger { color: var(--color-error); border-color: var(--color-error); }
.btn-outlined.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
.btn-filled .material-symbols-rounded,
.btn-tonal .material-symbols-rounded,
.btn-outlined .material-symbols-rounded { font-size: 18px; }

@media (max-width: 768px) {
  .employees-view { padding: 16px 12px 80px; }
  .emp-table thead { display: none; }
  .emp-table, .emp-table tbody, .emp-table tr, .emp-table td { display: block; }
  .emp-table tr {
    background: var(--color-surface);
    border: 1px solid var(--color-outline-variant);
    border-radius: var(--radius-lg, 16px);
    margin-bottom: 10px;
  }
  .emp-table td {
    border: none;
    padding: 6px 14px;
  }
  .td-act { justify-content: flex-end; }
  .hide-narrow { display: none; }
}
</style>
