<template>
  <div class="manage-page">
    <header class="manage-head">
      <button class="back-btn" @click="goBack" title="К списку компаний">
        <span class="material-symbols-outlined">arrow_back</span>
      </button>
      <div class="head-text" v-if="company">
        <h1 class="head-title">{{ company.name }}</h1>
        <span class="role-badge" :class="{ creator: isCreator }">
          {{ isCreator ? 'Создатель' : (isSuper ? 'Супер-админ' : 'Администратор') }}
        </span>
      </div>
      <div class="head-text" v-else>
        <h1 class="head-title">Компания</h1>
      </div>
    </header>

    <div v-if="loading" class="state-block"><ProgressSpinner /></div>
    <div v-else-if="loadError" class="state-block error-block">
      <span class="material-symbols-outlined">error</span>
      <p>{{ loadError }}</p>
    </div>

    <template v-else-if="company">
      <nav class="tabs">
        <button v-for="t in tabs" :key="t.key" class="tab" :class="{ active: tab === t.key }" @click="tab = t.key">
          <span class="material-symbols-outlined">{{ t.icon }}</span>
          <span>{{ t.label }}</span>
        </button>
      </nav>

      <div class="manage-body">
        <!-- ОБЗОР -->
        <section v-show="tab === 'overview'" class="pane">
          <div class="ov-stats">
            <div class="ov-stat">
              <span class="material-symbols-outlined">groups</span>
              <div><strong>{{ company.employees_count }}</strong><small>сотрудников</small></div>
            </div>
            <div class="ov-stat">
              <span class="material-symbols-outlined">checklist</span>
              <div><strong>{{ company.tasks_count }}</strong><small>задач</small></div>
            </div>
            <div class="ov-stat">
              <span class="material-symbols-outlined">event</span>
              <div><strong>{{ fmtDate(company.created_at) }}</strong><small>создана</small></div>
            </div>
          </div>
          <div v-if="company.description" class="ov-desc">{{ company.description }}</div>
          <CompanyInviteSettings v-if="canManageMembers" :company-id="company.id" />
        </section>

        <!-- УЧАСТНИКИ -->
        <section v-show="tab === 'members'" class="pane">
          <div v-if="!canManageMembers" class="note">
            Управлять участниками может только создатель компании. Вам доступен просмотр и настройки.
          </div>
          <div class="members-head">
            <h2>Участники <span class="count">{{ members.length }}</span></h2>
            <button v-if="canManageMembers" class="btn-filled" @click="openCreateUser">
              <span class="material-symbols-outlined">person_add</span>
              <span>Создать сотрудника</span>
            </button>
          </div>

          <div class="members">
            <div v-for="m in members" :key="m.id" class="member-row">
              <span class="member-ava">{{ initials(m.fio) }}</span>
              <span class="member-main">
                <span class="member-name">{{ m.fio }}</span>
                <span class="member-login">@{{ m.login }}<template v-if="m.post"> · {{ m.post }}</template></span>
              </span>
              <select
                v-if="canManageMembers"
                class="ctl member-role"
                :value="m.role?.id"
                @change="changeRole(m, Number($event.target.value))"
              >
                <option v-for="r in roleOptions" :key="r.id" :value="r.id">{{ r.name }}</option>
              </select>
              <span v-else class="member-rolelabel">{{ m.role?.name }}</span>
              <button v-if="canManageMembers" class="member-del" title="Сбросить пароль" @click="resetPassword(m)">
                <span class="material-symbols-outlined">lock_reset</span>
              </button>
              <button v-if="canManageMembers" class="member-del" title="Убрать из компании" @click="removeMember(m)">
                <span class="material-symbols-outlined">close</span>
              </button>
            </div>
          </div>

          <div v-if="canManageMembers" class="member-add">
            <div class="member-add-search">
              <span class="material-symbols-outlined">person_search</span>
              <input v-model="candQuery" class="ctl" type="text" placeholder="Добавить существующего: имя или логин…" @input="onCandQuery" />
            </div>
            <div v-if="candidates.length" class="cand-list">
              <button v-for="c in candidates" :key="c.id" type="button" class="cand-item" @click="addExisting(c)">
                <span class="member-name">{{ c.fio }}</span>
                <span class="member-login">@{{ c.login }}</span>
                <span class="material-symbols-outlined">add</span>
              </button>
            </div>
          </div>
          <div v-if="canManageMembers" class="invite-box">
            <div class="invite-box-head">
              <span class="material-symbols-outlined">mail</span>
              <span>Пригласить по email</span>
            </div>
            <div class="invite-row">
              <input
                v-model.trim="invite.email"
                type="email"
                class="ctl"
                placeholder="name@example.com"
                :disabled="inviting"
              />
              <select v-model.number="invite.roleId" class="ctl invite-role" :disabled="inviting">
                <option v-for="r in roleOptions" :key="r.id" :value="r.id">{{ r.name }}</option>
              </select>
              <button class="btn-filled" :disabled="inviting || !invite.email" @click="sendEmailInvite">
                <span class="material-symbols-outlined">send</span>
                <span>Пригласить</span>
              </button>
            </div>
            <p v-if="inviteSent" class="invite-ok">{{ inviteSent }}</p>
            <p v-if="inviteError" class="err">{{ inviteError }}</p>
          </div>

          <div v-if="membersError" class="err">{{ membersError }}</div>
        </section>

        <!-- НАСТРОЙКИ -->
        <section v-show="tab === 'settings'" class="pane settings-pane">
          <div class="settings-card flags-card">
            <h3>Возможности</h3>
            <label class="flag-row">
              <span><strong>Этапы задач</strong><small>Канбан-режим и теги этапов</small></span>
              <input type="checkbox" class="switch" v-model="flags.uses_stages" @change="saveFlags" />
            </label>
            <label class="flag-row">
              <span><strong>Интеграция с YouGile</strong><small>Импорт/экспорт карточек</small></span>
              <input type="checkbox" class="switch" v-model="flags.uses_yougile" @change="saveFlags" />
            </label>
            <label class="flag-row">
              <span><strong>Аудио/видео-звонки</strong><small>Кнопки звонка в мессенджере</small></span>
              <input type="checkbox" class="switch" v-model="flags.uses_calls" @change="saveFlags" />
            </label>
          </div>

          <GrooveSettings :company-id="company.id" />
          <WeekendSettings :company-id="company.id" />
          <AiSettings :company-id="company.id" />

          <div v-if="company.id === auth.companyId" class="yg-wrap">
            <YougileCompanySettings />
          </div>
          <div v-else class="note">
            Чтобы настроить интеграцию с YouGile, переключитесь на эту компанию в боковой панели (она привязана к активной сессии).
          </div>
        </section>

        <!-- ОПАСНАЯ ЗОНА -->
        <section v-show="tab === 'danger'" class="pane">
          <div class="settings-card danger-card" v-if="isSuper">
            <div>
              <h3>{{ company.is_active ? 'Отключить компанию' : 'Включить компанию' }}</h3>
              <p>Отключённая компания недоступна сотрудникам, но данные сохраняются.</p>
            </div>
            <button class="btn-outlined" :disabled="toggling" @click="toggleActive">
              {{ company.is_active ? 'Отключить' : 'Включить' }}
            </button>
          </div>
          <div class="settings-card danger-card">
            <div>
              <h3>Удалить компанию</h3>
              <p>Все данные удалятся каскадно: задачи, юниты, чаты, звонки. Необратимо.</p>
            </div>
            <button class="btn-outlined danger" @click="confirmDelete = true">Удалить</button>
          </div>
        </section>
      </div>
    </template>

    <!-- Создание сотрудника -->
    <AppDialog
      v-model="createUserOpen"
      title="Новый сотрудник"
      subtitle="Аккаунт создаётся с временным паролем — сотрудник сменит его при первом входе."
      icon="person_add"
      tone="primary"
      size="sm"
      :busy="creatingUser"
      :closable="!creatingUser"
    >
      <form class="cu-form" @submit.prevent="createUser">
        <div class="cu-field">
          <label>ФИО</label>
          <input v-model.trim="newUser.fio" class="ctl" placeholder="Фамилия Имя Отчество" :disabled="creatingUser" />
        </div>
        <div class="cu-field">
          <label>Логин</label>
          <input v-model.trim="newUser.login" class="ctl" placeholder="Не короче 3 символов" :disabled="creatingUser" />
        </div>
        <div class="cu-field">
          <label>Email <span class="cu-opt">— необязательно</span></label>
          <input v-model.trim="newUser.email" type="email" class="ctl" placeholder="name@example.com" :disabled="creatingUser" />
        </div>
        <div class="cu-field">
          <label>Должность <span class="cu-opt">— необязательно</span></label>
          <input v-model.trim="newUser.post" class="ctl" placeholder="Например: Дизайнер" :disabled="creatingUser" />
        </div>
        <div class="cu-field">
          <label>Роль</label>
          <select v-model.number="newUser.roleId" class="ctl">
            <option v-for="r in roleOptions" :key="r.id" :value="r.id">{{ r.name }}</option>
          </select>
        </div>
        <p v-if="createUserError" class="err">{{ createUserError }}</p>
      </form>
      <template #footer>
        <div class="cu-footer">
          <button class="cu-cancel" type="button" :disabled="creatingUser" @click="createUserOpen = false">Отмена</button>
          <button class="cu-submit" type="button" :disabled="creatingUser || !newUser.fio || !newUser.login" @click="createUser">
            Создать
          </button>
        </div>
      </template>
    </AppDialog>

    <AppDialog
      v-model="confirmDelete"
      tone="danger"
      icon="warning"
      size="sm"
      :title="`Удалить компанию «${company?.name}»?`"
      :busy="deleting"
      :closable="!deleting"
      :actions="[
        { kind: 'cancel', label: 'Отмена', disabled: deleting },
        { kind: 'confirm', label: 'Удалить', icon: 'delete', disabled: deleting },
      ]"
      @confirm="doDelete"
    >
      <p>Данные компании будут удалены безвозвратно.</p>
    </AppDialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import ProgressSpinner from 'primevue/progressspinner'
import AppDialog from '@/components/common/AppDialog.vue'
import GrooveSettings from '@/components/settings/GrooveSettings.vue'
import WeekendSettings from '@/components/settings/WeekendSettings.vue'
import AiSettings from '@/components/settings/AiSettings.vue'
import CompanyInviteSettings from '@/components/settings/CompanyInviteSettings.vue'
import YougileCompanySettings from '@/components/settings/YougileCompanySettings.vue'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { usePermission } from '@/composables/usePermission.js'
import {
  getCompany, updateCompany, deleteCompany, toggleCompanyActive,
  listCompanyMembers, getCompanyCandidates, addCompanyMember, setMemberRole, removeCompanyMember,
  createCompanyUser, resetCompanyMemberPassword, createCompanyInvite,
} from '@/api/companies.js'
import { getRoles } from '@/api/roles.js'

const props = defineProps({ id: { type: [String, Number], required: true } })

const router = useRouter()
const auth = useAuthStore()
const notif = useNotificationsStore()
const { isSuperAdmin, ROLES } = usePermission()
const isSuper = computed(() => isSuperAdmin())

const companyId = computed(() => Number(props.id))
const company = ref(null)
const loading = ref(true)
const loadError = ref('')

const tab = ref('overview')
const tabs = computed(() => {
  const list = [
    { key: 'overview', label: 'Обзор', icon: 'info' },
    { key: 'members', label: 'Участники', icon: 'groups' },
    { key: 'settings', label: 'Настройки', icon: 'tune' },
  ]
  if (canManageMembers.value) list.push({ key: 'danger', label: 'Опасная зона', icon: 'warning' })
  return list
})

const isCreator = computed(() => company.value?.created_by != null && company.value.created_by === auth.userId)
// Управление участниками/создание сотрудников — только создатель или супер-админ.
const canManageMembers = computed(() => isSuper.value || isCreator.value)

const members = ref([])
const roleOptions = ref([])
const membersError = ref('')
const candQuery = ref('')
const candidates = ref([])
let candTimer = null

const flags = ref({ uses_stages: false, uses_yougile: false, uses_calls: true })

const invite = ref({ email: '', roleId: ROLES.EMPLOYEE })
const inviting = ref(false)
const inviteError = ref('')
const inviteSent = ref('')

const createUserOpen = ref(false)
const creatingUser = ref(false)
const createUserError = ref('')
const newUser = ref({ fio: '', login: '', email: '', post: '', roleId: ROLES.EMPLOYEE })

const confirmDelete = ref(false)
const deleting = ref(false)
const toggling = ref(false)

onMounted(load)

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    company.value = await getCompany(companyId.value)
    flags.value = {
      uses_stages: !!company.value.settings?.uses_stages,
      uses_yougile: !!company.value.settings?.uses_yougile,
      uses_calls: company.value.settings?.uses_calls !== false,
    }
    await Promise.all([loadMembers(), loadRoles()])
  } catch (e) {
    loadError.value = e?.message || 'Не удалось загрузить компанию'
  } finally {
    loading.value = false
  }
}

async function loadMembers() {
  try {
    members.value = await listCompanyMembers(companyId.value)
  } catch (e) {
    membersError.value = e?.message || 'Не удалось загрузить участников'
  }
}

async function loadRoles() {
  try { roleOptions.value = (await getRoles()) || [] } catch { roleOptions.value = [] }
}

function fmtDate(s) {
  if (!s) return '—'
  return new Date(s).toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' })
}
function initials(fio) {
  return (fio || '').trim().split(/\s+/).slice(0, 2).map((p) => p[0]?.toUpperCase() || '').join('')
}

function goBack() { router.push('/companies') }

function onCandQuery() {
  if (candTimer) clearTimeout(candTimer)
  candTimer = setTimeout(searchCandidates, 250)
}
async function searchCandidates() {
  const q = candQuery.value.trim()
  if (!q) { candidates.value = []; return }
  try { candidates.value = await getCompanyCandidates(companyId.value, q) } catch { candidates.value = [] }
}

async function addExisting(c) {
  membersError.value = ''
  const employeeRole = roleOptions.value.find((r) => r.level === ROLES.EMPLOYEE) || roleOptions.value[0]
  try {
    await addCompanyMember(companyId.value, c.id, employeeRole.id)
    candQuery.value = ''
    candidates.value = []
    await loadMembers()
  } catch (e) {
    membersError.value = e?.message || 'Не удалось добавить'
  }
}

async function changeRole(m, roleId) {
  membersError.value = ''
  try {
    await setMemberRole(companyId.value, m.id, roleId)
    await loadMembers()
  } catch (e) {
    membersError.value = e?.message || 'Не удалось сменить роль'
    await loadMembers()
  }
}

async function removeMember(m) {
  membersError.value = ''
  try { await removeCompanyMember(companyId.value, m.id); await loadMembers() }
  catch (e) { membersError.value = e?.message || 'Не удалось убрать' }
}

async function resetPassword(m) {
  try {
    await resetCompanyMemberPassword(companyId.value, m.id)
    notif.success(`Пароль ${m.fio} сброшен на временный`)
  } catch (e) {
    notif.error(e?.message || 'Не удалось сбросить пароль')
  }
}

function openCreateUser() {
  newUser.value = { fio: '', login: '', email: '', post: '', roleId: ROLES.EMPLOYEE }
  createUserError.value = ''
  createUserOpen.value = true
}

async function createUser() {
  if (!newUser.value.fio || !newUser.value.login) return
  creatingUser.value = true
  createUserError.value = ''
  try {
    const payload = {
      fio: newUser.value.fio,
      login: newUser.value.login,
      role_id: newUser.value.roleId,
    }
    if (newUser.value.email) payload.email = newUser.value.email
    if (newUser.value.post) payload.post = newUser.value.post
    await createCompanyUser(companyId.value, payload)
    createUserOpen.value = false
    notif.success('Сотрудник создан')
    await loadMembers()
  } catch (e) {
    createUserError.value = e?.message || 'Не удалось создать сотрудника'
  } finally {
    creatingUser.value = false
  }
}

async function sendEmailInvite() {
  inviteError.value = ''
  inviteSent.value = ''
  if (!invite.value.email || !/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(invite.value.email)) {
    inviteError.value = 'Укажите корректный email'
    return
  }
  inviting.value = true
  try {
    await createCompanyInvite(companyId.value, invite.value.email, invite.value.roleId)
    inviteSent.value = `Приглашение отправлено на ${invite.value.email}`
    invite.value.email = ''
  } catch (e) {
    inviteError.value = e?.message || 'Не удалось отправить приглашение'
  } finally {
    inviting.value = false
  }
}

async function saveFlags() {
  try {
    const updated = await updateCompany(companyId.value, { settings: { ...flags.value } })
    company.value = updated
  } catch (e) {
    notif.error(e?.message || 'Не удалось сохранить настройки')
    await load()
  }
}

async function toggleActive() {
  toggling.value = true
  try {
    company.value = await toggleCompanyActive(companyId.value, !company.value.is_active)
    notif.success(company.value.is_active ? 'Компания включена' : 'Компания отключена')
  } catch (e) {
    notif.error(e?.message || 'Не удалось переключить статус')
  } finally {
    toggling.value = false
  }
}

async function doDelete() {
  deleting.value = true
  try {
    await deleteCompany(companyId.value)
    notif.success('Компания удалена')
    router.push('/companies')
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить')
    deleting.value = false
  }
}
</script>

<style scoped>
.manage-page { padding: 20px; max-width: 860px; margin: 0 auto; }

.manage-head { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
.back-btn {
  border: none; background: var(--color-surface-high); width: 40px; height: 40px;
  border-radius: 50%; display: grid; place-items: center; cursor: pointer; color: var(--color-text);
  flex: none;
}
.back-btn:hover { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.head-text { display: flex; align-items: center; gap: 12px; min-width: 0; flex-wrap: wrap; }
.head-title { margin: 0; font-size: 22px; font-weight: 800; color: var(--color-text); }
.role-badge {
  display: inline-flex; align-items: center; padding: 3px 10px; border-radius: var(--radius-full);
  font-size: 12px; font-weight: 600; background: var(--color-surface-high); color: var(--color-text-dim);
}
.role-badge.creator { background: var(--color-primary-container); color: var(--color-on-primary-container); }

.tabs { display: flex; gap: 4px; border-bottom: 1px solid var(--color-outline-dim); margin-bottom: 18px; overflow-x: auto; }
.tab {
  display: inline-flex; align-items: center; gap: 6px; padding: 10px 14px; border: none; background: none;
  color: var(--color-text-dim); font: inherit; font-weight: 600; cursor: pointer;
  border-bottom: 2px solid transparent; white-space: nowrap;
}
.tab:hover { color: var(--color-text); }
.tab.active { color: var(--color-primary); border-bottom-color: var(--color-primary); }
.tab .material-symbols-outlined { font-size: 18px; }

.pane { display: flex; flex-direction: column; gap: 18px; }

.ov-stats { display: flex; gap: 12px; flex-wrap: wrap; }
.ov-stat {
  display: flex; align-items: center; gap: 12px; padding: 14px 18px; flex: 1 1 160px;
  background: var(--color-surface-high); border-radius: var(--radius-lg);
}
.ov-stat .material-symbols-outlined { font-size: 26px; color: var(--color-primary); }
.ov-stat strong { display: block; font-size: 18px; font-weight: 800; color: var(--color-text); }
.ov-stat small { font-size: 12px; color: var(--color-text-dim); }
.ov-desc { color: var(--color-text); font-size: 14px; line-height: 1.5; }

.note {
  padding: 12px 14px; border-radius: var(--radius-md, 12px);
  background: var(--color-surface-high); color: var(--color-text-dim); font-size: 13px; line-height: 1.5;
}

.members-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
.members-head h2 { margin: 0; font-size: 17px; font-weight: 700; color: var(--color-text); }
.members-head .count { color: var(--color-text-dim); font-weight: 600; margin-left: 4px; }

.btn-filled {
  appearance: none; border: none; cursor: pointer; border-radius: var(--radius-full); padding: 9px 16px;
  font: inherit; font-weight: 600; display: inline-flex; align-items: center; gap: 6px;
  background: var(--color-primary); color: var(--color-on-primary);
}
.btn-filled:hover { background: var(--color-primary-hover); }
.btn-filled .material-symbols-outlined { font-size: 18px; }

.btn-outlined {
  appearance: none; cursor: pointer; border-radius: var(--radius-full); padding: 9px 18px; font: inherit;
  font-weight: 600; background: transparent; border: 1.5px solid var(--color-outline);
  color: var(--color-text);
}
.btn-outlined:hover { border-color: var(--color-primary); color: var(--color-primary); }
.btn-outlined.danger { border-color: color-mix(in oklch, var(--color-error) 50%, var(--color-outline)); color: var(--color-error); }
.btn-outlined.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); border-color: var(--color-error); }
.btn-outlined:disabled { opacity: .5; cursor: not-allowed; }

.members { display: flex; flex-direction: column; gap: 6px; }
.member-row {
  display: flex; align-items: center; gap: 10px; padding: 8px 10px;
  border-radius: var(--radius-md, 12px); background: var(--color-surface-high);
}
.member-ava {
  width: 34px; height: 34px; flex: none; border-radius: 50%; display: grid; place-items: center;
  font-size: 12px; font-weight: 700; background: var(--color-primary-container); color: var(--color-on-primary-container);
}
.member-main { display: flex; flex-direction: column; min-width: 0; flex: 1; }
.member-name { font-size: 14px; font-weight: 600; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.member-login { font-size: 12px; color: var(--color-text-dim); }
.member-rolelabel { font-size: 13px; color: var(--color-text-dim); }
.member-del {
  flex: none; display: grid; place-items: center; width: 32px; height: 32px; border: none;
  background: transparent; color: var(--color-text-dim); border-radius: 50%; cursor: pointer;
}
.member-del:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
.member-del .material-symbols-outlined { font-size: 18px; }

.ctl {
  appearance: none; width: 100%; box-sizing: border-box; border: 1px solid var(--color-outline-variant, var(--color-outline-dim));
  background: var(--color-surface); color: var(--color-text); padding: 10px 12px;
  border-radius: var(--radius-md, 12px); font: inherit;
}
.ctl:focus { outline: none; border-color: var(--color-primary); box-shadow: 0 0 0 3px color-mix(in oklch, var(--color-primary) 18%, transparent); }
select.ctl { padding-right: 32px; }
.member-role { width: auto; min-width: 140px; padding: 6px 28px 6px 10px; }

.member-add { margin-top: 10px; display: flex; flex-direction: column; gap: 6px; }
.member-add-search { position: relative; display: flex; align-items: center; }
.member-add-search > .material-symbols-outlined { position: absolute; left: 10px; font-size: 18px; color: var(--color-text-dim); pointer-events: none; }
.member-add-search .ctl { padding-left: 36px; }
.cand-list { display: flex; flex-direction: column; gap: 4px; max-height: 220px; overflow-y: auto; }
.cand-item {
  display: flex; align-items: center; gap: 8px; padding: 8px 10px; text-align: left;
  border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md, 12px);
  background: var(--color-surface); color: var(--color-text); cursor: pointer;
}
.cand-item:hover { border-color: var(--color-primary); }
.cand-item .member-login { flex: 1; }
.cand-item .material-symbols-outlined { font-size: 18px; color: var(--color-primary); }

.err { font-size: 13px; color: var(--color-error); }

.invite-box {
  margin-top: 16px;
  padding: 14px;
  border: 1px dashed var(--color-outline-dim);
  border-radius: var(--radius-md, 12px);
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.invite-box-head {
  display: flex; align-items: center; gap: 8px;
  font-size: 13px; font-weight: 700; color: var(--color-text);
}
.invite-box-head .material-symbols-outlined { font-size: 18px; color: var(--color-primary); }
.invite-row { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.invite-row .ctl { flex: 1 1 200px; }
.invite-role { flex: 0 0 auto; width: auto; min-width: 150px; padding-right: 32px; }
.invite-ok { margin: 0; font-size: 13px; color: var(--color-success); }

.settings-pane { gap: 22px; }
.settings-card {
  background: var(--color-surface-high); border-radius: var(--radius-lg); padding: 18px;
  display: flex; flex-direction: column; gap: 12px;
}
.flags-card h3 { margin: 0 0 4px; font-size: 16px; font-weight: 700; color: var(--color-text); }
.flag-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.flag-row strong { display: block; font-size: 14px; color: var(--color-text); }
.flag-row small { display: block; font-size: 12px; color: var(--color-text-dim); }

.danger-card { flex-direction: row; align-items: center; justify-content: space-between; gap: 16px; }
.danger-card h3 { margin: 0 0 4px; font-size: 15px; font-weight: 700; color: var(--color-text); }
.danger-card p { margin: 0; font-size: 13px; color: var(--color-text-dim); }

.switch {
  appearance: none; width: 44px; height: 24px; border-radius: 999px; flex: none;
  background: var(--color-surface-highest, var(--color-surface-high)); border: 2px solid var(--color-outline);
  box-sizing: border-box; position: relative; cursor: pointer; transition: background .18s, border-color .18s;
}
.switch::after {
  content: ''; position: absolute; top: 50%; left: 4px; width: 12px; height: 12px; border-radius: 50%;
  background: var(--color-outline); transform: translateY(-50%);
  transition: transform .2s, background .2s, width .2s, height .2s, left .2s;
}
.switch:checked { background: var(--color-primary); border-color: var(--color-primary); }
.switch:checked::after { width: 16px; height: 16px; left: 24px; background: var(--color-on-primary); }

.cu-form { display: flex; flex-direction: column; gap: 14px; }
.cu-field { display: flex; flex-direction: column; gap: 6px; }
.cu-field label { font-size: 12px; font-weight: 700; color: var(--color-primary); text-transform: uppercase; letter-spacing: 0.06em; }
.cu-opt { font-weight: 600; color: var(--color-text-dim); text-transform: none; letter-spacing: normal; }
.cu-footer { display: flex; justify-content: flex-end; gap: 10px; width: 100%; }
.cu-cancel { border: none; background: none; color: var(--color-primary); font: inherit; font-weight: 600; padding: 10px 16px; border-radius: var(--radius-full); cursor: pointer; }
.cu-cancel:hover:not(:disabled) { background: var(--color-surface-high); }
.cu-submit { border: none; border-radius: var(--radius-full); background: var(--color-primary); color: var(--color-on-primary); font: inherit; font-weight: 600; padding: 10px 20px; cursor: pointer; }
.cu-submit:disabled { opacity: .45; cursor: not-allowed; }

.state-block { display: grid; place-items: center; padding: 64px; gap: 10px; color: var(--color-text-dim); }
.error-block .material-symbols-outlined { font-size: 40px; color: var(--color-error); }
</style>
