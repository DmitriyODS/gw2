<template>
  <div class="admin-page">
    <header class="admin-sticky">
      <div class="page-head">
        <div class="page-head-text">
          <h1 class="page-head-title">Пользователи</h1>
          <div class="page-head-meta">
            <span class="meta-stat">
              <span class="material-symbols-outlined">people</span>
              <strong>{{ filtered.length }}</strong> из {{ users.length }}
            </span>
          </div>
        </div>
        <button class="btn-filled" @click="openCreate">
          <span class="material-symbols-outlined">person_add</span>
          <span>Добавить</span>
        </button>
      </div>

      <div class="admin-toolbar">
        <SearchField v-model="search" placeholder="Поиск по ФИО, логину, email…" hotkey />
      </div>
    </header>

    <div class="users-content">
      <!-- Мобильный: список карточек -->
      <div v-if="isMobile" class="users-cards">
        <article
          v-for="u in pageItems"
          :key="u.id"
          class="user-card"
          tabindex="0"
          @click="openProfile(u)"
          @keydown.enter.prevent="openProfile(u)"
        >
          <img :src="avatarOf(u)" :alt="u.fio" class="user-card-avatar" />
          <div class="user-card-main">
            <div class="user-card-name">
              {{ u.fio }}
              <span v-if="u.is_super_admin" class="su-badge">Супер-админ</span>
            </div>
            <div class="user-card-sub">@{{ u.login }}</div>
            <div v-if="u.email" class="user-card-sub">{{ u.email }}</div>
          </div>
          <span class="material-symbols-outlined card-chev">chevron_right</span>
        </article>
        <div v-if="!loading && !pageItems.length" class="users-empty">
          <span class="material-symbols-outlined">inbox</span>
          <span>Ничего не найдено</span>
        </div>
      </div>

      <!-- Десктоп: таблица с независимым скроллом тела -->
      <AppDataTable
        v-else
        :value="pageItems"
        :loading="loading"
        empty-message="Пользователи не найдены"
        :row-class="() => 'row-clickable'"
        @row-click="onRowClick"
      >
        <Column header="Пользователь">
          <template #body="{ data }">
            <div class="cell-user">
              <img :src="avatarOf(data)" :alt="data.fio" class="cell-avatar" />
              <div class="cell-user-text">
                <span class="cell-user-name">{{ data.fio }}</span>
                <span class="cell-user-login">@{{ data.login }}</span>
              </div>
            </div>
          </template>
        </Column>
        <Column header="Email">
          <template #body="{ data }">{{ data.email || '—' }}</template>
        </Column>
        <Column header="Телефон">
          <template #body="{ data }">{{ data.phone || '—' }}</template>
        </Column>
        <Column header="Статус">
          <template #body="{ data }">
            <span v-if="data.is_super_admin" class="su-badge">Супер-админ</span>
            <span v-else class="muted">Пользователь</span>
          </template>
        </Column>
      </AppDataTable>

      <div v-if="totalPages > 1" class="pagination">
        <button class="page-btn" :disabled="page === 1" @click="page--">
          <span class="material-symbols-outlined">chevron_left</span>
        </button>
        <button
          v-for="p in pageNumbers"
          :key="p.key"
          class="page-num"
          :class="{ active: p.value === page, gap: p.gap }"
          :disabled="p.gap"
          @click="!p.gap && (page = p.value)"
        >{{ p.gap ? '…' : p.value }}</button>
        <button class="page-btn" :disabled="page === totalPages" @click="page++">
          <span class="material-symbols-outlined">chevron_right</span>
        </button>
      </div>
    </div>

    <!-- Карточка пользователя (как в разделе «Сотрудники») + управление -->
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
          <span class="avatar avatar-xl">
            <img :src="avatarOf(selected)" :alt="selected.fio" />
          </span>
          <h2 class="profile-name">
            {{ selected.fio }}
            <span v-if="selected.is_super_admin" class="root-badge inline" title="Супер-администратор">
              <span class="material-symbols-outlined">verified</span>
            </span>
          </h2>
          <div class="profile-tags">
            <span class="profile-status">@{{ selected.login }}</span>
          </div>
        </div>

        <div class="profile-list">
          <a v-if="selected.phone" class="profile-row link" :href="`tel:${selected.phone}`">
            <span class="row-ico" data-tone="tertiary"><span class="material-symbols-outlined">phone</span></span>
            <span class="row-text">
              <span class="row-label">Телефон</span>
              <span class="row-value">{{ selected.phone }}</span>
            </span>
            <span class="material-symbols-outlined row-chev">arrow_outward</span>
          </a>
          <a v-if="selected.email" class="profile-row link" :href="`mailto:${selected.email}`">
            <span class="row-ico" data-tone="tertiary"><span class="material-symbols-outlined">mail</span></span>
            <span class="row-text">
              <span class="row-label">Email</span>
              <span class="row-value">{{ selected.email }}</span>
            </span>
            <span class="material-symbols-outlined row-chev">arrow_outward</span>
          </a>
          <div class="profile-row">
            <span class="row-ico" data-tone="secondary"><span class="material-symbols-outlined">alternate_email</span></span>
            <span class="row-text">
              <span class="row-label">Логин</span>
              <span class="row-value">@{{ selected.login }}</span>
            </span>
          </div>
        </div>

        <div class="profile-actions">
          <button class="btn-filled" @click="openEdit(selected)">
            <span class="material-symbols-outlined">edit</span> Редактировать
          </button>
          <button class="btn-tonal" @click="askReset(selected)">
            <span class="material-symbols-outlined">lock_reset</span>
            <span class="hide-narrow">Сбросить пароль</span>
            <span class="show-narrow">Пароль</span>
          </button>
          <button v-if="!selected.is_super_admin" class="btn-tonal danger" @click="askDelete(selected)">
            <span class="material-symbols-outlined">person_off</span>
            <span class="hide-narrow">Удалить</span>
          </button>
        </div>
      </div>
    </Dialog>

    <!-- Создание / редактирование -->
    <AppDialog
      v-model="formOpen"
      :title="editingId ? 'Редактирование пользователя' : 'Новый пользователь'"
      :icon="editingId ? 'edit' : 'person_add'"
      :busy="saving"
      :actions="[
        { kind: 'cancel', label: 'Отмена', disabled: saving },
        { kind: 'confirm', label: editingId ? 'Сохранить' : 'Создать', icon: 'check', disabled: saving },
      ]"
      @confirm="submitForm"
    >
      <div class="dlg-form">
        <div class="field">
          <label class="lbl">ФИО <span class="req">*</span></label>
          <input v-model.trim="form.fio" class="ctl" placeholder="Фамилия Имя Отчество" :disabled="saving" />
        </div>
        <div class="field">
          <label class="lbl">Логин <span class="req">*</span></label>
          <input v-model.trim="form.login" class="ctl" placeholder="Не короче 3 символов" :disabled="saving" />
        </div>
        <div class="field">
          <label class="lbl">Email <span class="opt">— необязательно</span></label>
          <input v-model.trim="form.email" type="email" class="ctl" placeholder="name@example.com" :disabled="saving" />
        </div>
        <div class="field">
          <label class="lbl">Телефон <span class="opt">— необязательно</span></label>
          <input v-model.trim="form.phone" class="ctl" placeholder="+7…" :disabled="saving" />
        </div>
        <div v-if="!editingId" class="field">
          <label class="lbl">Пароль <span class="opt">— необязательно</span></label>
          <input v-model="form.password" class="ctl" placeholder="Пусто → «логин123»" :disabled="saving" />
          <span class="hint">Пустой пароль → временный «{{ (form.login || 'логин') }}123» со сменой при входе.</span>
        </div>
      </div>
    </AppDialog>

    <AppDialog
      v-model="confirmResetOpen"
      tone="warning"
      icon="lock_reset"
      :title="`Сбросить пароль ${resetTarget?.fio || ''}?`"
      subtitle="Пароль станет временным «логин123» с обязательной сменой при входе."
      :busy="acting"
      :actions="[
        { kind: 'cancel', label: 'Отмена', disabled: acting },
        { kind: 'confirm', label: 'Сбросить', icon: 'lock_reset', disabled: acting },
      ]"
      @confirm="doReset"
    />

    <AppDialog
      v-model="confirmDeleteOpen"
      tone="danger"
      icon="person_off"
      :title="`Удалить ${deleteTarget?.fio || 'пользователя'}?`"
      subtitle="Аккаунт будет деактивирован: вход заблокируется, данные сохранятся."
      :busy="acting"
      :actions="[
        { kind: 'cancel', label: 'Отмена', disabled: acting },
        { kind: 'confirm', label: 'Удалить', icon: 'person_off', disabled: acting },
      ]"
      @confirm="doDelete"
    />
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import Column from 'primevue/column'
import Dialog from 'primevue/dialog'
import AppDataTable from '@/components/common/AppDataTable.vue'
import AppDialog from '@/components/common/AppDialog.vue'
import SearchField from '@/components/common/SearchField.vue'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import {
  getUsers, createPlatformUser, updatePlatformUser,
  resetPlatformUserPassword, deletePlatformUser,
} from '@/api/users.js'

const { isMobile } = useBreakpoint()
const notif = useNotificationsStore()

const users = ref([])
const loading = ref(false)
const search = ref('')
const page = ref(1)
const perPage = 15

function avatarOf(u) {
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return users.value
  return users.value.filter(u =>
    u.fio?.toLowerCase().includes(q) ||
    u.login?.toLowerCase().includes(q) ||
    u.email?.toLowerCase().includes(q),
  )
})

const totalPages = computed(() => Math.max(1, Math.ceil(filtered.value.length / perPage)))
const pageItems = computed(() => filtered.value.slice((page.value - 1) * perPage, page.value * perPage))

const pageNumbers = computed(() => {
  const total = totalPages.value
  const cur = page.value
  const out = []
  const push = (v, gap = false) => out.push({ value: v, gap, key: gap ? `gap-${v}` : `p-${v}` })
  if (total <= 7) {
    for (let i = 1; i <= total; i++) push(i)
    return out
  }
  push(1)
  if (cur > 3) push(-1, true)
  for (let i = Math.max(2, cur - 1); i <= Math.min(total - 1, cur + 1); i++) push(i)
  if (cur < total - 2) push(-2, true)
  push(total)
  return out
})

watch(search, () => { page.value = 1 })
watch(totalPages, (n) => { if (page.value > n) page.value = n })

async function load() {
  loading.value = true
  try {
    users.value = await getUsers()
  } catch (e) {
    notif.error(e?.message || 'Не удалось загрузить пользователей')
  } finally {
    loading.value = false
  }
}
onMounted(load)

// ── Карточка пользователя ──
const profileOpen = ref(false)
const selected = ref(null)
function openProfile(u) { selected.value = u; profileOpen.value = true }
function onRowClick(e) { if (e?.data) openProfile(e.data) }
watch(profileOpen, (open) => { if (!open) selected.value = null })

// ── Форма создания/редактирования ──
const formOpen = ref(false)
const editingId = ref(null)
const saving = ref(false)
const form = ref({ fio: '', login: '', email: '', phone: '', password: '' })

function openCreate() {
  editingId.value = null
  form.value = { fio: '', login: '', email: '', phone: '', password: '' }
  formOpen.value = true
}
function openEdit(u) {
  profileOpen.value = false
  editingId.value = u.id
  form.value = { fio: u.fio || '', login: u.login || '', email: u.email || '', phone: u.phone || '', password: '' }
  formOpen.value = true
}

async function submitForm() {
  if (!form.value.fio || !form.value.login) {
    notif.error('ФИО и логин обязательны')
    return
  }
  saving.value = true
  try {
    const body = {
      fio: form.value.fio,
      login: form.value.login,
      email: form.value.email || null,
      phone: form.value.phone || null,
    }
    if (editingId.value) {
      await updatePlatformUser(editingId.value, body)
      notif.success('Пользователь обновлён')
    } else {
      if (form.value.password) body.password = form.value.password
      await createPlatformUser(body)
      notif.success('Пользователь создан')
    }
    formOpen.value = false
    await load()
  } catch (e) {
    notif.error(e?.message || 'Не удалось сохранить')
  } finally {
    saving.value = false
  }
}

// ── Сброс пароля / удаление ──
const acting = ref(false)
const confirmResetOpen = ref(false)
const resetTarget = ref(null)
const confirmDeleteOpen = ref(false)
const deleteTarget = ref(null)

function askReset(u) { profileOpen.value = false; resetTarget.value = u; confirmResetOpen.value = true }
function askDelete(u) { profileOpen.value = false; deleteTarget.value = u; confirmDeleteOpen.value = true }

async function doReset() {
  acting.value = true
  try {
    await resetPlatformUserPassword(resetTarget.value.id)
    notif.success(`Пароль ${resetTarget.value.fio} сброшен`)
    confirmResetOpen.value = false
  } catch (e) {
    notif.error(e?.message || 'Не удалось сбросить пароль')
  } finally {
    acting.value = false
  }
}

async function doDelete() {
  acting.value = true
  try {
    await deletePlatformUser(deleteTarget.value.id)
    notif.success('Пользователь удалён')
    confirmDeleteOpen.value = false
    await load()
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить')
  } finally {
    acting.value = false
  }
}
</script>

<style scoped>
/* ── Шапка ── */
.page-head { display: flex; align-items: center; justify-content: space-between; gap: 16px; flex-wrap: wrap; }
.page-head-text { min-width: 0; }
.page-head-title { margin: 0 0 6px; font-size: 24px; font-weight: 800; letter-spacing: -0.01em; color: var(--color-text); }
.page-head-meta { display: inline-flex; align-items: center; gap: 10px; }

/* ── Кнопки ── */
.btn-filled, .btn-tonal {
  appearance: none; border: none; cursor: pointer; border-radius: var(--radius-full); padding: 10px 18px;
  font: inherit; font-weight: 600; display: inline-flex; align-items: center; justify-content: center; gap: 6px;
  transition: background .12s, color .12s, filter .12s, box-shadow .12s;
}
.btn-filled { background: var(--color-primary); color: var(--color-on-primary); box-shadow: var(--shadow-sm); }
.btn-filled:hover { background: var(--color-primary-hover); }
.btn-tonal { background: var(--color-secondary-container); color: var(--color-on-secondary-container); }
.btn-tonal:hover { filter: brightness(.96); }
.btn-tonal.danger { background: var(--color-error-container); color: var(--color-on-error-container); }
.btn-filled .material-symbols-outlined, .btn-tonal .material-symbols-outlined { font-size: 18px; }

/* ── Контент ── */
.users-content { flex: 1; min-height: 0; display: flex; flex-direction: column; gap: 12px; padding: 16px 0 20px; }

.cell-user { display: flex; align-items: center; gap: 12px; }
.cell-avatar, .user-card-avatar {
  width: 38px; height: 38px; border-radius: var(--radius-full); object-fit: cover; background: var(--color-surface-high); flex-shrink: 0;
}
.cell-user-text { display: flex; flex-direction: column; line-height: 1.25; }
.cell-user-name { font-weight: 600; }
.cell-user-login, .muted { color: var(--color-text-dim); font-size: 12.5px; }

.su-badge {
  display: inline-block; padding: 2px 9px; border-radius: var(--radius-full);
  background: var(--color-tertiary-container); color: var(--color-on-tertiary-container); font-size: 11px; font-weight: 700;
}

/* Мобильные карточки */
.users-cards { flex: 1; min-height: 0; overflow-y: auto; display: flex; flex-direction: column; gap: 8px; }
.user-card {
  display: flex; align-items: center; gap: 12px; padding: 12px 14px; cursor: pointer;
  background: var(--acrylic-card-bg); border: 1px solid var(--acrylic-border); border-radius: var(--radius-lg);
}
.user-card:hover { background: var(--color-surface-high); }
.user-card-main { flex: 1; min-width: 0; }
.user-card-name { font-weight: 600; display: flex; align-items: center; gap: 8px; }
.user-card-sub { color: var(--color-text-dim); font-size: 12.5px; }
.card-chev { color: var(--color-text-dim); }
.users-empty { display: flex; flex-direction: column; align-items: center; gap: 8px; padding: 48px 0; color: var(--color-text-dim); }
.users-empty .material-symbols-outlined { font-size: 36px; opacity: 0.6; }

/* Пагинация */
.pagination { display: flex; align-items: center; justify-content: center; gap: 6px; flex-wrap: wrap; }
.page-btn, .page-num {
  min-width: 38px; height: 38px; padding: 0 8px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-full);
  background: var(--acrylic-card-bg); color: var(--color-text); cursor: pointer; display: grid; place-items: center;
  font-size: 14px; transition: background .14s, border-color .14s;
}
.page-btn:disabled { opacity: 0.4; cursor: default; }
.page-num.active { background: var(--color-primary); color: var(--color-on-primary); border-color: var(--color-primary); }
.page-num.gap { border: none; background: transparent; cursor: default; }
.page-btn:not(:disabled):hover, .page-num:not(.active):not(.gap):hover { border-color: var(--color-primary); }

/* ── Профиль-модалка (стиль раздела «Сотрудники») ── */
.avatar { position: relative; display: inline-grid; place-items: center; flex-shrink: 0; border-radius: 50%; isolation: isolate; }
.avatar img { width: 100%; height: 100%; border-radius: 50%; object-fit: cover; display: block; }
.avatar-xl { width: 116px; height: 116px; }
.avatar::before {
  content: ''; position: absolute; inset: -5px; border-radius: 50%;
  border: 4px solid var(--color-outline-dim); z-index: -1;
}
.root-badge { display: inline-grid; place-items: center; flex-shrink: 0; }
.root-badge.inline {
  width: 22px; height: 22px; background: var(--color-tertiary-container); color: var(--color-on-tertiary-container);
  border-radius: 50%; margin-left: 4px;
}
.root-badge.inline .material-symbols-outlined { font-size: 14px; font-variation-settings: 'FILL' 1; }

.emp-profile { display: flex; flex-direction: column; background: var(--acrylic-card-bg); width: 100%; box-sizing: border-box; position: relative; }
.profile-close {
  position: absolute; top: 12px; right: 12px; z-index: 2; width: 36px; height: 36px; border-radius: 50%;
  border: none; background: color-mix(in oklch, var(--color-surface) 60%, transparent); color: var(--color-text-dim);
  display: grid; place-items: center; cursor: pointer; backdrop-filter: blur(8px); transition: background .12s, color .12s;
}
.profile-close:hover { background: var(--acrylic-card-bg); color: var(--color-text); }
.profile-close .material-symbols-outlined { font-size: 20px; }
.profile-cover {
  position: absolute; inset: 0 0 auto; height: 150px;
  background:
    radial-gradient(120% 140% at 85% 0%, color-mix(in oklch, var(--color-tertiary-container) 40%, transparent) 0%, transparent 60%),
    linear-gradient(120deg, color-mix(in oklch, var(--color-primary-container) 55%, var(--color-surface)), color-mix(in oklch, var(--color-secondary-container) 55%, var(--color-surface)));
  -webkit-mask-image: linear-gradient(to bottom, black 30%, transparent 100%);
  mask-image: linear-gradient(to bottom, black 30%, transparent 100%);
  pointer-events: none;
}
.profile-hero { position: relative; display: flex; flex-direction: column; align-items: center; text-align: center; padding: 36px 22px 22px; gap: 10px; color: var(--color-text); }
.profile-name {
  margin: 0; font-size: 22px; font-weight: 800; line-height: 1.2; letter-spacing: -0.01em; color: var(--color-text);
  word-break: break-word; display: inline-flex; align-items: center; gap: 6px; justify-content: center; flex-wrap: wrap;
}
.profile-tags { display: inline-flex; align-items: center; flex-wrap: wrap; gap: 8px; margin-top: 4px; }
.profile-status {
  display: inline-flex; align-items: center; gap: 6px; padding: 4px 12px; border-radius: var(--radius-full);
  font-size: 12px; font-weight: 600; background: color-mix(in oklch, var(--color-text) 8%, transparent); color: var(--color-text-dim);
}
.profile-list { display: flex; flex-direction: column; gap: 4px; padding: 16px; background: var(--acrylic-card-bg); }
.profile-row {
  display: flex; align-items: center; gap: 12px; padding: 10px 12px; border-radius: var(--radius-lg);
  text-decoration: none; color: var(--color-text); background: var(--color-surface-low); transition: background .12s;
}
.profile-row.link { cursor: pointer; }
.profile-row.link:hover { background: var(--color-surface-high); }
.row-ico { width: 40px; height: 40px; border-radius: var(--radius-md); display: grid; place-items: center; flex-shrink: 0; }
.row-ico[data-tone="secondary"] { background: var(--color-secondary-container); color: var(--color-on-secondary-container); }
.row-ico[data-tone="tertiary"] { background: var(--color-tertiary-container); color: var(--color-on-tertiary-container); }
.row-ico .material-symbols-outlined { font-size: 20px; }
.row-text { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 1px; }
.row-label { font-size: 11px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.06em; color: var(--color-text-dim); }
.row-value { font-size: 14px; font-weight: 600; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.row-chev { font-size: 18px; color: var(--color-text-dim); flex-shrink: 0; }
.profile-actions { display: flex; flex-wrap: wrap; gap: 8px; padding: 0 16px 16px; }
.profile-actions > * { flex: 1 1 120px; }
.show-narrow { display: none; }

/* ── Форма (как в разделе «Компании») ── */
.dlg-form { display: flex; flex-direction: column; gap: 16px; }
.field { display: flex; flex-direction: column; gap: 6px; }
/* Глобальный input.ctl задаёт только фон/рамку — размеры доопределяем тут
   (как в разделе «Компании»), иначе поля сжаты по вертикали. */
.field .ctl {
  appearance: none; width: 100%; box-sizing: border-box;
  padding: 11px 13px; font: inherit; line-height: 1.3;
}
.lbl { font-size: 13px; font-weight: 600; color: var(--color-text-dim); }
.req { color: var(--color-error); }
.opt { font-weight: 500; color: var(--color-text-dim); }
.hint { margin: 0; font-size: 12px; color: var(--color-text-dim); line-height: 1.5; }

@media (max-width: 768px) {
  .page-head-title { font-size: 20px; }
  .admin-toolbar :deep(.search-field) { flex: 1 1 100%; max-width: 100%; }
  .hide-narrow { display: none; }
  .show-narrow { display: inline; }
}
</style>
