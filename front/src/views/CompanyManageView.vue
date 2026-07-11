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
      <SegmentedTabs v-model="tab" :tabs="mainTabs" class="manage-tabs" />

      <div class="manage-body">
        <!-- ОБЗОР -->
        <section v-show="tab === 'overview'" class="pane pane-scroll">
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
        </section>

        <!-- УЧАСТНИКИ -->
        <section v-show="tab === 'members'" class="pane pane-members">
          <div class="members-toolbar">
            <h2 class="toolbar-title">Участники <span class="count">{{ members.length }}</span></h2>
            <div v-if="canManageMembers" class="toolbar-actions">
              <button class="btn-outlined" @click="openInviteLink">
                <span class="material-symbols-outlined">link</span>
                <span>Ссылка-приглашение</span>
              </button>
              <button class="btn-outlined" @click="openInvite">
                <span class="material-symbols-outlined">mail</span>
                <span>Пригласить</span>
              </button>
              <button class="btn-filled" @click="openAdd">
                <span class="material-symbols-outlined">person_add</span>
                <span>Добавить сотрудника</span>
              </button>
            </div>
          </div>

          <div v-if="!canManageMembers" class="note">
            Управлять участниками может только создатель компании. Вам доступен просмотр и настройки.
          </div>

          <div class="members-table">
            <AppDataTable :value="members" empty-message="В компании пока нет участников">
              <Column header="Сотрудник">
                <template #body="{ data }">
                  <div class="cell-member">
                    <span class="member-ava">{{ initials(data.fio) }}</span>
                    <div class="member-text">
                      <span class="member-name">{{ data.fio }}</span>
                      <span class="member-login">@{{ data.login }}<template v-if="data.post"> · {{ data.post }}</template></span>
                    </div>
                  </div>
                </template>
              </Column>

              <Column header="Роль" :style="canManageMembers ? 'width: 200px' : 'width: 160px'">
                <template #body="{ data }">
                  <Select
                    v-if="canManageMembers"
                    :model-value="data.role?.id"
                    :options="roleOptions"
                    option-label="name"
                    option-value="id"
                    class="role-select"
                    @update:model-value="(v) => changeRole(data, v)"
                  />
                  <span v-else class="role-pill">{{ data.role?.name }}</span>
                </template>
              </Column>

              <Column v-if="canManageMembers" header="" style="width: 110px" body-style="text-align: right">
                <template #body="{ data }">
                  <div class="row-actions">
                    <button class="icon-btn" title="Сбросить пароль" @click="resetPassword(data)">
                      <span class="material-symbols-outlined">lock_reset</span>
                    </button>
                    <button
                      v-if="!isOwner(data)"
                      class="icon-btn danger"
                      title="Убрать из компании"
                      @click="askRemove(data)"
                    >
                      <span class="material-symbols-outlined">person_remove</span>
                    </button>
                  </div>
                </template>
              </Column>
            </AppDataTable>
          </div>

          <p v-if="membersError" class="err">{{ membersError }}</p>
        </section>

        <!-- НАСТРОЙКИ -->
        <section v-show="tab === 'settings'" class="pane pane-settings">
          <SegmentedTabs v-model="settingsTab" :tabs="settingsTabs" class="settings-subtabs" />

          <div class="settings-content pane-scroll">
            <div v-show="settingsTab === 'features'" class="settings-card">
              <h3 class="card-title">Возможности</h3>
              <div class="switch-list">
                <label class="switch-row">
                  <span class="switch-text">
                    <span class="material-symbols-outlined">view_kanban</span>
                    <span><strong>Этапы задач</strong><small>Канбан-режим и цветные теги этапов</small></span>
                  </span>
                  <input type="checkbox" class="switch" v-model="flags.uses_stages" @change="saveFlags" />
                </label>
                <label class="switch-row">
                  <span class="switch-text">
                    <span class="material-symbols-outlined">link</span>
                    <span><strong>Интеграция с YouGile</strong><small>Импорт/экспорт карточек</small></span>
                  </span>
                  <input type="checkbox" class="switch" v-model="flags.uses_yougile" @change="saveFlags" />
                </label>
                <label class="switch-row">
                  <span class="switch-text">
                    <span class="material-symbols-outlined">call</span>
                    <span><strong>Аудио/видео-звонки</strong><small>Кнопки звонка в мессенджере</small></span>
                  </span>
                  <input type="checkbox" class="switch" v-model="flags.uses_calls" @change="saveFlags" />
                </label>
              </div>
            </div>

            <div v-show="settingsTab === 'lists'"><CompanyListsSettings :company-id="company.id" /></div>
            <div v-show="settingsTab === 'ai'"><AiSettings :company-id="company.id" /></div>
            <div v-show="settingsTab === 'schedule'"><WeekendSettings :company-id="company.id" /></div>
            <div v-show="settingsTab === 'groove'"><GrooveSettings :company-id="company.id" /></div>

            <div v-show="settingsTab === 'yougile'">
              <YougileCompanySettings v-if="company.id === auth.companyId" />
              <div v-else class="note">
                Чтобы настроить интеграцию с YouGile, переключитесь на эту компанию в боковой панели
                (она привязана к активной сессии).
              </div>
            </div>

            <div v-show="settingsTab === 'registries'" class="settings-fill">
              <RegistriesManager v-if="company.id === auth.companyId" />
              <div v-else class="note">
                Чтобы настроить реестры, переключитесь на эту компанию в боковой панели
                (они привязаны к активной сессии).
              </div>
            </div>

            <div v-show="settingsTab === 'calendars'" class="settings-fill">
              <CalendarsManager v-if="company.id === auth.companyId" />
              <div v-else class="note">
                Чтобы настроить календари, переключитесь на эту компанию в боковой панели
                (они привязаны к активной сессии).
              </div>
            </div>
          </div>
        </section>

        <!-- ОПАСНАЯ ЗОНА -->
        <section v-show="tab === 'danger'" class="pane pane-scroll">
          <div class="settings-card danger-card" v-if="isSuper">
            <div>
              <h3 class="card-title">{{ company.is_active ? 'Отключить компанию' : 'Включить компанию' }}</h3>
              <p class="card-desc">Отключённая компания недоступна сотрудникам, но данные сохраняются.</p>
            </div>
            <button class="btn-outlined" :disabled="toggling" @click="toggleActive">
              {{ company.is_active ? 'Отключить' : 'Включить' }}
            </button>
          </div>
          <div class="settings-card danger-card">
            <div>
              <h3 class="card-title">Удалить компанию</h3>
              <p class="card-desc">Все данные удалятся каскадно: задачи, юниты, чаты, звонки. Необратимо.</p>
            </div>
            <button class="btn-outlined danger" @click="confirmDelete = true">Удалить</button>
          </div>
        </section>
      </div>
    </template>

    <!-- Добавить сотрудника: существующий / новый -->
    <AppDialog
      v-model="addOpen"
      title="Добавить сотрудника"
      icon="group_add"
      tone="primary"
      size="md"
      :busy="creatingUser"
      :closable="!creatingUser"
    >
      <div class="add-body">
        <SegmentedTabs v-model="addTab" :tabs="addTabs" full-width class="add-subtabs" />

        <!-- Существующий -->
        <div v-show="addTab === 'existing'" class="add-pane">
          <div class="search-field">
            <span class="material-symbols-outlined">person_search</span>
            <input
              v-model="candQuery"
              class="ctl"
              type="text"
              placeholder="Поиск по имени или логину…"
              @input="onCandQuery"
            />
          </div>
          <div v-if="candidates.length" class="cand-list">
            <button v-for="c in candidates" :key="c.id" type="button" class="cand-item" @click="addExisting(c)">
              <span class="member-ava sm">{{ initials(c.fio) }}</span>
              <span class="cand-text">
                <span class="member-name">{{ c.fio }}</span>
                <span class="member-login">@{{ c.login }}</span>
              </span>
              <span class="material-symbols-outlined add-ic">add</span>
            </button>
          </div>
          <p v-else-if="candQuery.trim()" class="hint">Никого не нашли — попробуйте другой запрос или вкладку «Новый».</p>
          <p v-else class="hint">Начните вводить имя или логин уже зарегистрированного пользователя.</p>
        </div>

        <!-- Новый -->
        <form v-show="addTab === 'new'" class="add-pane add-form" @submit.prevent="createUser">
          <div class="field">
            <label class="lbl">ФИО <span class="req">*</span></label>
            <input v-model.trim="newUser.fio" class="ctl" placeholder="Фамилия Имя Отчество" :disabled="creatingUser" />
          </div>
          <div class="field">
            <label class="lbl">Логин <span class="req">*</span></label>
            <input v-model.trim="newUser.login" class="ctl" placeholder="Не короче 3 символов" :disabled="creatingUser" />
          </div>
          <div class="field">
            <label class="lbl">Email <span class="opt">— необязательно</span></label>
            <input v-model.trim="newUser.email" type="email" class="ctl" placeholder="name@example.com" :disabled="creatingUser" />
          </div>
          <div class="field">
            <label class="lbl">Должность <span class="opt">— необязательно</span></label>
            <input v-model.trim="newUser.post" class="ctl" placeholder="Например: Дизайнер" :disabled="creatingUser" />
          </div>
          <div class="field">
            <label class="lbl">Роль</label>
            <Select
              v-model="newUser.roleId"
              :options="roleOptions"
              option-label="name"
              option-value="id"
              class="w-full"
            />
          </div>
          <p v-if="createUserError" class="err">{{ createUserError }}</p>
        </form>
      </div>

      <template #footer>
        <div class="modal-foot">
          <button class="btn-text" type="button" :disabled="creatingUser" @click="addOpen = false">Закрыть</button>
          <button
            v-if="addTab === 'new'"
            class="btn-filled"
            type="button"
            :disabled="creatingUser || !newUser.fio || !newUser.login"
            @click="createUser"
          >
            Создать
          </button>
        </div>
      </template>
    </AppDialog>

    <!-- Пригласить по email -->
    <AppDialog
      v-model="inviteOpen"
      title="Пригласить по email"
      subtitle="Получатель перейдёт по ссылке из письма и вступит в компанию с выбранной ролью."
      icon="mail"
      tone="primary"
      size="sm"
      :busy="inviting"
      :closable="!inviting"
    >
      <form class="add-form" @submit.prevent="sendEmailInvite">
        <div class="field">
          <label class="lbl">Email <span class="req">*</span></label>
          <input
            v-model.trim="invite.email"
            type="email"
            class="ctl"
            placeholder="name@example.com"
            :disabled="inviting"
          />
        </div>
        <div class="field">
          <label class="lbl">Роль</label>
          <Select
            v-model="invite.roleId"
            :options="roleOptions"
            option-label="name"
            option-value="id"
            class="w-full"
          />
        </div>
        <p v-if="inviteError" class="err">{{ inviteError }}</p>
      </form>
      <template #footer>
        <div class="modal-foot">
          <button class="btn-text" type="button" :disabled="inviting" @click="inviteOpen = false">Отмена</button>
          <button class="btn-filled" type="button" :disabled="inviting || !invite.email" @click="sendEmailInvite">
            <span class="material-symbols-outlined">send</span>
            <span>Отправить</span>
          </button>
        </div>
      </template>
    </AppDialog>

    <!-- Подтверждение удаления участника -->
    <AppDialog
      v-model="confirmRemove"
      tone="danger"
      icon="person_remove"
      size="sm"
      :title="`Убрать ${removeTarget?.fio || 'сотрудника'} из компании?`"
      :busy="removing"
      :closable="!removing"
      :actions="[
        { kind: 'cancel', label: 'Отмена', disabled: removing },
        { kind: 'confirm', label: 'Убрать', icon: 'person_remove', disabled: removing },
      ]"
      @confirm="doRemoveMember"
    >
      <p>Сотрудник потеряет доступ к компании. Его аккаунт и данные сохранятся, при необходимости его можно добавить снова.</p>
    </AppDialog>

    <!-- Ссылка-приглашение -->
    <AppDialog
      v-model="inviteLinkOpen"
      title="Ссылка-приглашение"
      subtitle="Любой авторизованный пользователь, перешедший по ссылке, вступит в компанию как Сотрудник. Перевыпуск делает старую ссылку недействительной."
      icon="link"
      tone="primary"
      size="md"
    >
      <div class="invite-link-body">
        <div class="invite-link-row">
          <input class="ctl" :value="inviteUrl" readonly placeholder="Ссылка ещё не создана" />
          <button class="icon-btn bordered" :disabled="!inviteCode" title="Скопировать" @click="copyInviteLink">
            <span class="material-symbols-outlined">{{ inviteCopied ? 'check' : 'content_copy' }}</span>
          </button>
        </div>
        <button class="btn-outlined" :disabled="inviteLinkBusy" @click="regenInviteLink">
          <span class="material-symbols-outlined">{{ inviteCode ? 'autorenew' : 'add_link' }}</span>
          <span>{{ inviteCode ? 'Перевыпустить' : 'Создать ссылку' }}</span>
        </button>
        <p v-if="inviteLinkError" class="err">{{ inviteLinkError }}</p>
      </div>
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
import Select from 'primevue/select'
import Column from 'primevue/column'
import AppDialog from '@/components/common/AppDialog.vue'
import AppDataTable from '@/components/common/AppDataTable.vue'
import SegmentedTabs from '@/components/common/SegmentedTabs.vue'
import GrooveSettings from '@/components/settings/GrooveSettings.vue'
import WeekendSettings from '@/components/settings/WeekendSettings.vue'
import AiSettings from '@/components/settings/AiSettings.vue'
import CompanyListsSettings from '@/components/settings/CompanyListsSettings.vue'
import YougileCompanySettings from '@/components/settings/YougileCompanySettings.vue'
import RegistriesManager from '@/components/registry/RegistriesManager.vue'
import CalendarsManager from '@/components/calendar/CalendarsManager.vue'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { usePermission } from '@/composables/usePermission.js'
import {
  getCompany, updateCompany, deleteCompany, toggleCompanyActive,
  listCompanyMembers, getCompanyCandidates, addCompanyMember, setMemberRole, removeCompanyMember,
  createCompanyUser, resetCompanyMemberPassword, createCompanyInvite,
  getCompanyInvite, regenerateCompanyInvite,
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
const mainTabs = computed(() => {
  const list = [
    { value: 'overview', label: 'Обзор', icon: 'info' },
    { value: 'members', label: 'Участники', icon: 'groups' },
    { value: 'settings', label: 'Настройки', icon: 'tune' },
  ]
  if (canManageMembers.value) list.push({ value: 'danger', label: 'Опасная зона', icon: 'warning' })
  return list
})

const settingsTab = ref('features')
const settingsTabs = [
  { value: 'features', label: 'Возможности', icon: 'tune' },
  { value: 'lists', label: 'Списки', icon: 'format_list_bulleted' },
  { value: 'ai', label: 'ИИ', icon: 'smart_toy' },
  { value: 'schedule', label: 'Расписание', icon: 'weekend' },
  { value: 'groove', label: 'Мой Groove', icon: 'celebration' },
  { value: 'registries', label: 'Реестры', icon: 'table' },
  { value: 'calendars', label: 'Календари', icon: 'calendar_month' },
  { value: 'yougile', label: 'YouGile', icon: 'link' },
]

const addTabs = [
  { value: 'existing', label: 'Существующий', icon: 'person_search' },
  { value: 'new', label: 'Новый', icon: 'person_add' },
]

const isCreator = computed(() => company.value?.created_by != null && company.value.created_by === auth.userId)
// Управление участниками/создание сотрудников — только создатель или супер-админ.
const canManageMembers = computed(() => isSuper.value || isCreator.value)
// Владелец компании (created_by) — его из компании убрать нельзя.
const isOwner = (m) => company.value?.created_by != null && company.value.created_by === m.id

const members = ref([])
const roleOptions = ref([])
const membersError = ref('')

const flags = ref({ uses_stages: false, uses_yougile: false, uses_calls: true })

// Добавление сотрудника (существующий/новый) и приглашение по email — модалки.
const addOpen = ref(false)
const addTab = ref('existing')
const candQuery = ref('')
const candidates = ref([])
let candTimer = null
const creatingUser = ref(false)
const createUserError = ref('')
const newUser = ref({ fio: '', login: '', email: '', post: '', roleId: ROLES.EMPLOYEE })

const inviteOpen = ref(false)
const invite = ref({ email: '', roleId: ROLES.EMPLOYEE })
const inviting = ref(false)
const inviteError = ref('')

// Подтверждение удаления участника.
const confirmRemove = ref(false)
const removeTarget = ref(null)
const removing = ref(false)

// Ссылка-приглашение (модалка).
const inviteLinkOpen = ref(false)
const inviteCode = ref('')
const inviteLinkBusy = ref(false)
const inviteLinkError = ref('')
const inviteCopied = ref(false)
const inviteUrl = computed(() => (inviteCode.value ? `${window.location.origin}/join/${inviteCode.value}` : ''))

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

// ── Добавление: существующий ──
function openAdd() {
  addTab.value = 'existing'
  candQuery.value = ''
  candidates.value = []
  createUserError.value = ''
  newUser.value = { fio: '', login: '', email: '', post: '', roleId: ROLES.EMPLOYEE }
  addOpen.value = true
}

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
    addOpen.value = false
    await loadMembers()
  } catch (e) {
    membersError.value = e?.message || 'Не удалось добавить'
  }
}

// ── Добавление: новый ──
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
    addOpen.value = false
    notif.success('Сотрудник создан')
    await loadMembers()
  } catch (e) {
    createUserError.value = e?.message || 'Не удалось создать сотрудника'
  } finally {
    creatingUser.value = false
  }
}

async function changeRole(m, roleId) {
  if (!roleId || roleId === m.role?.id) return
  membersError.value = ''
  try {
    await setMemberRole(companyId.value, m.id, roleId)
    await loadMembers()
  } catch (e) {
    membersError.value = e?.message || 'Не удалось сменить роль'
    await loadMembers()
  }
}

function askRemove(m) {
  removeTarget.value = m
  confirmRemove.value = true
}

async function doRemoveMember() {
  const m = removeTarget.value
  if (!m) return
  membersError.value = ''
  removing.value = true
  try {
    await removeCompanyMember(companyId.value, m.id)
    confirmRemove.value = false
    await loadMembers()
  } catch (e) {
    membersError.value = e?.message || 'Не удалось убрать'
  } finally {
    removing.value = false
  }
}

// ── Ссылка-приглашение ──
async function openInviteLink() {
  inviteLinkError.value = ''
  inviteCopied.value = false
  inviteLinkOpen.value = true
  try {
    const res = await getCompanyInvite(companyId.value)
    inviteCode.value = res.code || ''
  } catch (e) {
    inviteLinkError.value = e?.message || 'Не удалось загрузить ссылку'
  }
}

async function regenInviteLink() {
  inviteLinkBusy.value = true
  inviteLinkError.value = ''
  try {
    const res = await regenerateCompanyInvite(companyId.value)
    inviteCode.value = res.code || ''
  } catch (e) {
    inviteLinkError.value = e?.message || 'Не удалось создать ссылку'
  } finally {
    inviteLinkBusy.value = false
  }
}

async function copyInviteLink() {
  if (!inviteUrl.value) return
  try {
    await navigator.clipboard.writeText(inviteUrl.value)
    inviteCopied.value = true
    setTimeout(() => { inviteCopied.value = false }, 1500)
  } catch { /* ignore */ }
}

async function resetPassword(m) {
  try {
    await resetCompanyMemberPassword(companyId.value, m.id)
    notif.success(`Пароль ${m.fio} сброшен на временный`)
  } catch (e) {
    notif.error(e?.message || 'Не удалось сбросить пароль')
  }
}

// ── Приглашение по email ──
function openInvite() {
  invite.value = { email: '', roleId: ROLES.EMPLOYEE }
  inviteError.value = ''
  inviteOpen.value = true
}

async function sendEmailInvite() {
  inviteError.value = ''
  if (!invite.value.email || !/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(invite.value.email)) {
    inviteError.value = 'Укажите корректный email'
    return
  }
  inviting.value = true
  try {
    await createCompanyInvite(companyId.value, invite.value.email, invite.value.roleId)
    inviteOpen.value = false
    notif.success(`Приглашение отправлено на ${invite.value.email}`)
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
.manage-page {
  height: 100%;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 20px;
  max-width: 1040px;
  width: 100%;
  margin: 0 auto;
  box-sizing: border-box;
}

.manage-head { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; flex: none; }
.back-btn {
  border: 1px solid var(--acrylic-border);
  background: var(--color-surface-high);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  width: 40px; height: 40px;
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

.manage-tabs { flex: none; margin-bottom: 16px; }

.manage-body { flex: 1; min-height: 0; display: flex; }

/* min-width:0 — чтобы дочерние SegmentedTabs/таблицы могли сжаться до ширины
   экрана и скроллиться, а не распирать панель за край (особенно на мобильном). */
.pane { flex: 1; min-width: 0; min-height: 0; display: flex; flex-direction: column; gap: 18px; }
.pane-scroll { overflow-y: auto; }

.ov-stats { display: flex; gap: 12px; flex-wrap: wrap; }
.ov-stat {
  display: flex; align-items: center; gap: 12px; padding: 14px 18px; flex: 1 1 160px;
  background: var(--acrylic-card-bg); border-radius: var(--radius-lg);
}
.ov-stat .material-symbols-outlined { font-size: 26px; color: var(--color-primary); }
.ov-stat strong { display: block; font-size: 18px; font-weight: 800; color: var(--color-text); }
.ov-stat small { font-size: 12px; color: var(--color-text-dim); }
.ov-desc { color: var(--color-text); font-size: 14px; line-height: 1.5; }

.note {
  padding: 12px 14px; border-radius: var(--radius-md, 12px);
  background: var(--acrylic-card-bg); color: var(--color-text-dim); font-size: 13px; line-height: 1.5;
}

/* ── Участники: таблица занимает всю высоту и скроллится отдельно ── */
.pane-members { gap: 14px; }
.members-toolbar {
  flex: none; display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap;
}
.toolbar-title { margin: 0; font-size: 17px; font-weight: 700; color: var(--color-text); }
.toolbar-title .count { color: var(--color-text-dim); font-weight: 600; margin-left: 4px; }
.toolbar-actions { display: flex; gap: 10px; flex-wrap: wrap; }

.members-table { flex: 1; min-height: 0; }

.cell-member { display: flex; align-items: center; gap: 12px; min-width: 0; }
.member-ava {
  width: 36px; height: 36px; flex: none; border-radius: 50%; display: grid; place-items: center;
  font-size: 13px; font-weight: 700; background: var(--color-primary-container); color: var(--color-on-primary-container);
}
.member-ava.sm { width: 32px; height: 32px; font-size: 12px; }
.member-text { display: flex; flex-direction: column; min-width: 0; }
.member-name { font-size: 14px; font-weight: 600; color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.member-login { font-size: 12px; color: var(--color-text-dim); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.role-pill {
  display: inline-flex; align-items: center; padding: 4px 12px; border-radius: var(--radius-full);
  font-size: 13px; font-weight: 600; background: var(--color-surface-high); color: var(--color-text-dim);
}
.role-select { min-width: 160px; }

.row-actions { display: inline-flex; gap: 4px; justify-content: flex-end; }
.icon-btn {
  width: 36px; height: 36px; border: none; border-radius: 50%; background: transparent;
  color: var(--color-text-dim); cursor: pointer; display: grid; place-items: center;
}
.icon-btn:hover { background: var(--color-surface-high); color: var(--color-text); }
.icon-btn.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
.icon-btn .material-symbols-outlined { font-size: 20px; }

.err { margin: 0; font-size: 13px; color: var(--color-error); flex: none; }

/* ── Настройки с под-вкладками ── */
.pane-settings { gap: 16px; }
.settings-subtabs { flex: none; }
.settings-content { flex: 1; min-height: 0; display: flex; flex-direction: column; gap: 18px; }
/* Вкладка реестров сама держит высоту и внутреннюю прокрутку — растягиваем её
   обёртку на всю область настроек, чтобы конструктор не растягивал .settings-content. */
.settings-fill { flex: 1; min-height: 0; display: flex; }

.settings-card {
  background: var(--acrylic-card-bg); border-radius: var(--radius-lg); padding: 18px;
  display: flex; flex-direction: column; gap: 12px;
}
.card-title { margin: 0; font-size: 16px; font-weight: 700; color: var(--color-text); }
.card-desc { margin: 0; font-size: 13px; color: var(--color-text-dim); line-height: 1.5; }

.danger-card { flex-direction: row; align-items: center; justify-content: space-between; gap: 16px; }

/* ── Кнопки ── */
.btn-filled {
  appearance: none; border: none; cursor: pointer; border-radius: var(--radius-full); padding: 9px 16px;
  font: inherit; font-weight: 600; display: inline-flex; align-items: center; gap: 6px;
  background: var(--color-primary); color: var(--color-on-primary);
}
.btn-filled:hover:not(:disabled) { background: var(--color-primary-hover); }
.btn-filled:disabled { opacity: .5; cursor: not-allowed; }
.btn-filled .material-symbols-outlined { font-size: 18px; }

.btn-outlined {
  appearance: none; cursor: pointer; border-radius: var(--radius-full); padding: 9px 16px; font: inherit;
  font-weight: 600; background: transparent; border: 1.5px solid var(--color-outline);
  color: var(--color-text); display: inline-flex; align-items: center; gap: 6px;
}
.btn-outlined:hover:not(:disabled) { border-color: var(--color-primary); color: var(--color-primary); }
.btn-outlined.danger { border-color: color-mix(in oklch, var(--color-error) 50%, var(--color-outline)); color: var(--color-error); }
.btn-outlined.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); border-color: var(--color-error); }
.btn-outlined:disabled { opacity: .5; cursor: not-allowed; }
.btn-outlined .material-symbols-outlined { font-size: 18px; }

.btn-text {
  appearance: none; border: none; background: none; color: var(--color-primary); font: inherit;
  font-weight: 600; padding: 9px 16px; border-radius: var(--radius-full); cursor: pointer;
}
.btn-text:hover:not(:disabled) { background: var(--color-surface-high); }
.btn-text:disabled { opacity: .5; cursor: not-allowed; }

/* ── Переключатель (switch-row, 1:1 с CompanyFormDialog) ── */
.switch-list { display: flex; flex-direction: column; gap: 6px; }
.switch-row {
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
  padding: 10px 12px; background: var(--acrylic-card-bg); border-radius: var(--radius-md, 12px);
  cursor: pointer; transition: background .12s;
}
.switch-row:hover { background: var(--glass-hover-bg); }
.switch-text { display: flex; align-items: center; gap: 12px; min-width: 0; }
.switch-text > .material-symbols-outlined {
  display: grid; place-items: center; width: 36px; height: 36px; border-radius: var(--radius-md, 12px);
  background: var(--color-primary-container); color: var(--color-on-primary-container); font-size: 20px; flex: none;
}
.switch-text strong { display: block; font-size: 14px; color: var(--color-text); }
.switch-text small { display: block; font-size: 12px; color: var(--color-text-dim); }

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

/* ── Модалки добавления/приглашения ── */
.add-body { display: flex; flex-direction: column; gap: 16px; }
.add-subtabs { align-self: stretch; }
.add-pane { display: flex; flex-direction: column; gap: 10px; min-height: 220px; }
.add-form { display: flex; flex-direction: column; gap: 14px; }

.field { display: flex; flex-direction: column; gap: 6px; }
.lbl { font-size: 13px; font-weight: 600; color: var(--color-on-surface-variant, var(--color-text-dim)); }
.req { color: var(--color-error); }
.opt { font-weight: 500; color: var(--color-text-dim); }

.ctl {
  appearance: none; width: 100%; box-sizing: border-box;
  border: 1px solid var(--color-outline-variant, var(--color-outline-dim));
  background: var(--acrylic-card-bg); color: var(--color-text); padding: 10px 12px;
  border-radius: var(--radius-md, 12px); font: inherit; transition: border-color .15s, box-shadow .15s;
}
.ctl:focus { outline: none; border-color: var(--color-primary); box-shadow: 0 0 0 3px color-mix(in oklch, var(--color-primary) 18%, transparent); }

.search-field { position: relative; display: flex; align-items: center; }
.search-field > .material-symbols-outlined { position: absolute; left: 10px; font-size: 18px; color: var(--color-text-dim); pointer-events: none; }
.search-field .ctl { padding-left: 36px; }

.cand-list { display: flex; flex-direction: column; gap: 6px; max-height: 280px; overflow-y: auto; }
.cand-item {
  display: flex; align-items: center; gap: 10px; padding: 8px 10px; text-align: left;
  border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md, 12px);
  background: var(--acrylic-card-bg); color: var(--color-text); cursor: pointer;
}
.cand-item:hover { border-color: var(--color-primary); }
.cand-text { display: flex; flex-direction: column; min-width: 0; flex: 1; }
.cand-item .add-ic { font-size: 20px; color: var(--color-primary); }

.hint { margin: 0; font-size: 13px; color: var(--color-text-dim); line-height: 1.5; }

.modal-foot { display: flex; justify-content: flex-end; gap: 10px; width: 100%; }

.invite-link-body { display: flex; flex-direction: column; gap: 14px; align-items: flex-start; }
.invite-link-row { display: flex; gap: 8px; align-items: center; width: 100%; }
.invite-link-row .ctl { flex: 1; font-size: 13px; }
.icon-btn.bordered {
  flex: none; width: 44px; height: 44px; border-radius: var(--radius-md, 12px);
  border: 1px solid var(--color-outline-dim); color: var(--color-text);
}
.icon-btn.bordered:hover:not(:disabled) { border-color: var(--color-primary); color: var(--color-primary); background: transparent; }
.icon-btn.bordered:disabled { opacity: .5; cursor: not-allowed; }

.w-full { width: 100%; }

.state-block { flex: 1; display: grid; place-items: center; padding: 64px; gap: 10px; color: var(--color-text-dim); }
.error-block .material-symbols-outlined { font-size: 40px; color: var(--color-error); }

@media (max-width: 768px) {
  .manage-page { padding: 12px; }
  /* Резерв под нижнюю навигацию (64px) + 12px воздуха: вкладки со своим
     скроллом (.pane-scroll) уводят контент под стекло; карточка-таблица
     участников скроллится внутри себя — резерв вешаем на саму вкладку,
     чтобы таблица (и её последние строки) не пряталась за навигацией. */
  .pane-scroll { padding-bottom: calc(76px + env(safe-area-inset-bottom, 0px)); }
  .pane-members { padding-bottom: calc(76px + env(safe-area-inset-bottom, 0px)); }
  .manage-tabs, .settings-subtabs { margin-bottom: 12px; }
  .danger-card { flex-direction: column; align-items: flex-start; }

  .members-toolbar { gap: 10px; }
  .toolbar-actions { width: 100%; }
  /* Кнопки переносятся по 2 в ряд (мин. база 140px), не сминаясь в нечитаемые. */
  .toolbar-actions .btn-filled, .toolbar-actions .btn-outlined { flex: 1 1 140px; justify-content: center; }

  /* Плотнее ячейки таблицы участников — иначе на узком экране слишком широко. */
  .members-table :deep(.p-datatable-thead > tr > th),
  .members-table :deep(.p-datatable-tbody > tr > td) {
    padding: 12px 14px !important;
  }
  .members-table :deep(.p-datatable-thead > tr > th:first-child),
  .members-table :deep(.p-datatable-tbody > tr > td:first-child) { padding-left: 16px !important; }
  .members-table :deep(.p-datatable-thead > tr > th:last-child),
  .members-table :deep(.p-datatable-tbody > tr > td:last-child) { padding-right: 16px !important; }
  .role-select { min-width: 132px; }

  .modal-foot { flex-wrap: wrap; }
}
</style>
