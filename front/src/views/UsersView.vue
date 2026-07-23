<template>
  <div class="admin-page">
    <header class="admin-sticky">
      <!-- Тулбар в стиле «Компаний»: заголовок, поиск, статы-чипы, действие. -->
      <div class="admin-toolbar">
        <h1 class="cmp-title">Пользователи</h1>
        <SearchField v-model="search" placeholder="Поиск по ФИО, логину, email…" hotkey />
        <span class="chip-tint chip-tint--primary cmp-chip">
          <span class="material-symbols-outlined">people</span>
          <strong>{{ users.length }}</strong>&nbsp;всего
        </span>
        <span class="chip-tint chip-tint--success cmp-chip">
          <strong>{{ activeCount }}</strong>&nbsp;активных
        </span>
        <span v-if="inactiveCount" class="chip-tint chip-tint--error cmp-chip">
          <strong>{{ inactiveCount }}</strong>&nbsp;деактивированных
        </span>
        <button class="btn-grad desktop-only" @click="openCreate">
          <span class="material-symbols-outlined">person_add</span>
          <span>Добавить</span>
        </button>
      </div>
    </header>

    <div class="admin-body">
      <!-- Мобильный: карточки -->
      <div v-if="isMobile" class="cmp-cards">
        <div v-if="loading" class="state-block"><BrandLoader /></div>
        <EmptyState
          v-else-if="!visible.length"
          class="cmp-cards-empty"
          :icon="search ? 'search_off' : 'people'"
          :title="search ? 'Ничего не нашли' : 'Пользователей пока нет'"
          :subtitle="search ? 'Попробуйте уточнить запрос.' : 'Добавьте первого пользователя платформы.'"
        />
        <template v-else>
          <article
            v-for="u in visible"
            :key="u.id"
            class="cmp-card"
            :class="{ off: !u.is_active }"
          >
            <div class="cmp-card-top">
              <img :src="avatarOf(u)" :alt="u.fio" class="user-avatar" />
              <div class="cmp-card-text">
                <div class="cmp-card-name">
                  {{ u.fio }}
                  <span v-if="u.is_super_admin" class="su-badge">Супер-админ</span>
                </div>
                <div class="cmp-card-desc">@{{ u.login }}</div>
              </div>
              <label v-if="canToggle(u)" class="toggle" @click.stop>
                <input type="checkbox" :checked="u.is_active" :disabled="actingId === u.id" @change="onToggle(u)" />
                <span class="toggle-track" />
              </label>
              <span v-else-if="!u.is_active" class="status-pill off">Деактивирован</span>
            </div>

            <div class="cmp-card-stats">
              <span v-if="u.email" class="stat"><span class="material-symbols-outlined">mail</span>{{ u.email }}</span>
              <span v-if="u.phone" class="stat"><span class="material-symbols-outlined">phone</span>{{ u.phone }}</span>
            </div>

            <div v-if="!u.is_super_admin" class="cmp-card-actions" @click.stop>
              <button class="card-act" title="Редактировать" @click="openEdit(u)">
                <span class="material-symbols-outlined">edit</span>
              </button>
              <button class="card-act" title="Сбросить пароль" @click="askReset(u)">
                <span class="material-symbols-outlined">lock_reset</span>
              </button>
              <button v-if="!u.is_active" class="card-act danger" title="Удалить окончательно" @click="askPurge(u)">
                <span class="material-symbols-outlined">delete_forever</span>
              </button>
            </div>
          </article>
        </template>
      </div>

      <!-- Десктоп: таблица -->
      <AppDataTable
        v-else
        :value="visible"
        :loading="loading"
        empty-message="Пользователи не найдены"
      >
        <Column header="Пользователь">
          <template #body="{ data }">
            <div class="cell-user">
              <img :src="avatarOf(data)" :alt="data.fio" class="user-avatar" />
              <div class="cmp-name-text">
                <div class="cmp-name-main" :class="{ off: !data.is_active }">
                  {{ data.fio }}
                  <span v-if="data.is_super_admin" class="su-badge">Супер-админ</span>
                </div>
                <div class="cmp-name-sub">@{{ data.login }}</div>
              </div>
            </div>
          </template>
        </Column>

        <Column header="Email" style="min-width: 180px">
          <template #body="{ data }">
            <span :class="{ muted: !data.email }">{{ data.email || '—' }}</span>
          </template>
        </Column>

        <Column header="Телефон" style="width: 160px">
          <template #body="{ data }">
            <span :class="{ muted: !data.phone }">{{ data.phone || '—' }}</span>
          </template>
        </Column>

        <Column header="Статус" style="width: 180px">
          <template #body="{ data }">
            <label v-if="canToggle(data)" class="toggle" :title="data.is_active ? 'Активен' : 'Деактивирован'">
              <input type="checkbox" :checked="data.is_active" :disabled="actingId === data.id" @change="onToggle(data)" />
              <span class="toggle-track" />
              <span :class="['toggle-label', { on: data.is_active }]">
                {{ data.is_active ? 'Активен' : 'Деактивирован' }}
              </span>
            </label>
            <span v-else-if="data.is_super_admin" class="muted">Платформа</span>
            <span v-else class="muted">Вы</span>
          </template>
        </Column>

        <Column header="" style="width: 150px" body-style="text-align: right">
          <template #body="{ data }">
            <div v-if="!data.is_super_admin" class="row-actions">
              <button class="icon-btn" title="Редактировать" @click="openEdit(data)">
                <span class="material-symbols-outlined">edit</span>
              </button>
              <button class="icon-btn" title="Сбросить пароль" @click="askReset(data)">
                <span class="material-symbols-outlined">lock_reset</span>
              </button>
              <button v-if="!data.is_active" class="icon-btn danger" title="Удалить окончательно" @click="askPurge(data)">
                <span class="material-symbols-outlined">delete_forever</span>
              </button>
            </div>
          </template>
        </Column>
      </AppDataTable>
    </div>

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
      size="sm"
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
      v-model="confirmPurgeOpen"
      tone="danger"
      icon="delete_forever"
      size="sm"
      :title="`Удалить окончательно ${purgeTarget?.fio || 'пользователя'}?`"
      :busy="acting"
      :closable="!acting"
      :actions="[
        { kind: 'cancel', label: 'Отмена', disabled: acting },
        { kind: 'confirm', label: 'Удалить навсегда', icon: 'delete_forever', disabled: acting },
      ]"
      @confirm="doPurge"
    >
      <p class="confirm-warn">
        Аккаунт и <strong>все его данные</strong> будут стёрты безвозвратно: задачи, юниты,
        сообщения, питомец, заметки. <strong>Восстановить будет нельзя.</strong>
      </p>
    </AppDialog>

    <AppFab icon="person_add" aria-label="Новый пользователь" @click="openCreate" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import Column from 'primevue/column'
import BrandLoader from '@/components/common/BrandLoader.vue'
import AppDataTable from '@/components/common/AppDataTable.vue'
import AppDialog from '@/components/common/AppDialog.vue'
import AppFab from '@/components/common/AppFab.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SearchField from '@/components/common/SearchField.vue'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useAuthStore } from '@/stores/auth.js'
import {
  getUsers, createPlatformUser, updatePlatformUser,
  resetPlatformUserPassword, deletePlatformUser,
  reactivatePlatformUser, purgePlatformUser,
} from '@/api/users.js'

const { isMobile } = useBreakpoint()
const notif = useNotificationsStore()
const auth = useAuthStore()

const users = ref([])
const loading = ref(false)
const search = ref('')

function avatarOf(u) {
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

const activeCount = computed(() => users.value.filter((u) => u.is_active).length)
const inactiveCount = computed(() => users.value.filter((u) => !u.is_active).length)

const visible = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return users.value
  return users.value.filter((u) =>
    u.fio?.toLowerCase().includes(q) ||
    u.login?.toLowerCase().includes(q) ||
    u.email?.toLowerCase().includes(q),
  )
})

// Тумблер статуса доступен для обычных аккаунтов, кроме своего и супер-админа.
function canToggle(u) {
  return !u.is_super_admin && u.id !== auth.userId
}

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

// ── Активация / деактивация ──
const actingId = ref(null)
async function onToggle(u) {
  actingId.value = u.id
  try {
    if (u.is_active) {
      await deletePlatformUser(u.id)
      notif.success(`${u.fio} деактивирован`)
    } else {
      await reactivatePlatformUser(u.id)
      notif.success(`${u.fio} восстановлен`)
    }
    await load()
  } catch (e) {
    notif.error(e?.message || 'Не удалось изменить статус')
  } finally {
    actingId.value = null
  }
}

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

// ── Сброс пароля / окончательное удаление ──
const acting = ref(false)
const confirmResetOpen = ref(false)
const resetTarget = ref(null)
const confirmPurgeOpen = ref(false)
const purgeTarget = ref(null)

function askReset(u) { resetTarget.value = u; confirmResetOpen.value = true }
function askPurge(u) { purgeTarget.value = u; confirmPurgeOpen.value = true }

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

async function doPurge() {
  acting.value = true
  try {
    await purgePlatformUser(purgeTarget.value.id)
    notif.success('Пользователь удалён окончательно')
    confirmPurgeOpen.value = false
    await load()
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить')
  } finally {
    acting.value = false
  }
}
</script>

<style scoped>
/* Прозрачная «плавающая» шапка как в «Компаниях»/«Задачах». */
.admin-sticky { background: transparent; -webkit-backdrop-filter: none; backdrop-filter: none; }
.admin-sticky::after { display: none; }

.cmp-title {
  margin: 0; font-size: 20px; font-weight: 800; color: var(--color-text); white-space: nowrap;
}

.cell-user { display: flex; align-items: center; gap: 12px; min-width: 0; }
.user-avatar {
  width: 38px; height: 38px; border-radius: var(--radius-full); object-fit: cover;
  background: var(--color-surface-high); flex-shrink: 0;
}
.cmp-name-text { min-width: 0; }
.cmp-name-main {
  font-size: 14px; font-weight: 700; color: var(--color-text);
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  display: flex; align-items: center; gap: 8px;
}
.cmp-name-main.off { opacity: 0.55; }
.cmp-name-sub { font-size: 12px; color: var(--color-text-dim); }

.su-badge {
  display: inline-block; padding: 2px 9px; border-radius: var(--radius-full);
  background: var(--color-tertiary-container); color: var(--color-on-tertiary-container);
  font-size: 11px; font-weight: 700;
}

.muted { color: var(--color-text-dim); font-size: 13px; }

.status-pill {
  display: inline-flex; align-items: center; padding: 3px 10px; border-radius: var(--radius-full);
  font-size: 12px; font-weight: 600;
}
.status-pill.off { background: var(--color-error-container); color: var(--color-on-error-container); }

/* ── Тумблер (как в «Компаниях») ── */
.toggle { display: inline-flex; align-items: center; gap: 10px; cursor: pointer; user-select: none; }
.toggle input { display: none; }
.toggle-track {
  position: relative; width: 44px; height: 24px; border-radius: var(--radius-full);
  background: var(--color-surface-highest); border: 2px solid var(--color-outline);
  box-sizing: border-box; transition: background .18s, border-color .18s; flex-shrink: 0;
}
.toggle-track::after {
  content: ''; position: absolute; top: 50%; left: 4px; width: 12px; height: 12px;
  border-radius: 50%; background: var(--color-outline); transform: translateY(-50%);
  transition: transform .2s cubic-bezier(0.4, 0, 0.2, 1), background .2s, width .2s, height .2s, left .2s;
}
.toggle input:checked + .toggle-track { background: var(--color-success); border-color: var(--color-success); }
.toggle input:checked + .toggle-track::after { width: 16px; height: 16px; left: 24px; background: var(--color-on-success); }
.toggle-label { font-size: 12px; font-weight: 600; color: var(--color-text-dim); }
.toggle-label.on { color: var(--color-success); }

/* ── Строчные действия ── */
.row-actions { display: inline-flex; align-items: center; justify-content: flex-end; gap: 4px; }
.icon-btn {
  appearance: none; border: none; background: transparent; width: 34px; height: 34px; min-height: 0;
  display: grid; place-items: center; border-radius: 50%; color: var(--color-text-dim);
  cursor: pointer; transition: background .14s, color .14s;
}
.icon-btn:hover { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.icon-btn.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
.icon-btn .material-symbols-outlined { font-size: 18px; }

.confirm-warn { color: var(--color-text); }
.confirm-warn strong { color: var(--color-error); }

/* ── Мобильные карточки (как в «Компаниях») ── */
.cmp-cards { display: flex; flex-direction: column; gap: 10px; }
.cmp-card {
  background: var(--acrylic-card-bg); border: 1px solid var(--acrylic-border); border-radius: var(--radius-lg);
  padding: 14px; display: flex; flex-direction: column; gap: 12px;
}
.cmp-card.off { opacity: 0.65; }
.cmp-card-top { display: flex; align-items: center; gap: 12px; }
.cmp-card-top .user-avatar { width: 44px; height: 44px; }
.cmp-card-text { flex: 1; min-width: 0; }
.cmp-card-name {
  font-size: 15px; font-weight: 700; color: var(--color-text); line-height: 1.3;
  display: flex; align-items: center; gap: 8px; overflow: hidden;
}
.cmp-card-desc { font-size: 12.5px; color: var(--color-text-dim); margin-top: 2px; }
.cmp-card-stats { display: flex; gap: 8px; flex-wrap: wrap; }
.cmp-card-stats .stat {
  display: inline-flex; align-items: center; gap: 4px; padding: 4px 10px;
  background: var(--color-surface-high); border-radius: var(--radius-full);
  font-size: 12.5px; color: var(--color-text); max-width: 100%;
  overflow: hidden; text-overflow: ellipsis;
}
.cmp-card-stats .stat .material-symbols-outlined { font-size: 15px; color: var(--color-text-dim); }
.cmp-card-actions {
  display: flex; justify-content: flex-end; gap: 4px;
  border-top: 1px solid var(--color-outline-dim); padding-top: 10px; margin-top: -2px;
}
.card-act {
  appearance: none; border: 1px solid var(--acrylic-border); background: var(--glass-bg);
  box-shadow: var(--glass-edge); width: 40px; height: 40px; border-radius: var(--radius-full);
  display: grid; place-items: center; color: var(--color-text); cursor: pointer;
  transition: background .14s, color .14s;
}
.card-act:hover { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.card-act.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
.card-act .material-symbols-outlined { font-size: 20px; }

.cmp-cards-empty { background: var(--acrylic-card-bg); border-radius: var(--radius-xl); }
.state-block { display: grid; place-items: center; padding: 48px; }

/* ── Форма (как в «Компаниях») ── */
.dlg-form { display: flex; flex-direction: column; gap: 16px; }
.field { display: flex; flex-direction: column; gap: 6px; }
.field .ctl { appearance: none; width: 100%; box-sizing: border-box; padding: 11px 13px; font: inherit; line-height: 1.3; }
.lbl { font-size: 13px; font-weight: 600; color: var(--color-text-dim); }
.req { color: var(--color-error); }
.opt { font-weight: 500; color: var(--color-text-dim); }
.hint { margin: 0; font-size: 12px; color: var(--color-text-dim); line-height: 1.5; }

@media (max-width: 768px) {
  .cmp-title { flex-basis: 100%; font-size: 18px; }
  .desktop-only { display: none; }
  .admin-toolbar :deep(.search-field) { flex: 1 1 100%; max-width: 100%; }
}
</style>
