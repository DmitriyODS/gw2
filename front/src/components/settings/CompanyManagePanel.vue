<template>
  <div class="manage-page">
    <header class="manage-head">
      <AppButton
        variant="icon"
        size="sm"
        icon="arrow_back"
        title="К списку компаний"
        aria-label="К списку компаний"
        @click="goBack"
      />
      <div class="head-text">
        <h1 class="head-title">{{ company?.name || 'Компания' }}</h1>
        <AppChip v-if="company" :tone="isCreator ? 'primary' : 'neutral'">
          {{ isCreator ? 'Создатель' : (isSuper ? 'Супер-админ' : 'Администратор') }}
        </AppChip>
      </div>
    </header>

    <div v-if="loading" class="state-block"><BrandLoader /></div>
    <AppInfoBar v-else-if="loadError" tone="error" :message="loadError" />

    <template v-else-if="company">
      <AppTabs v-model="tab" :tabs="mainTabs" class="manage-tabs" />

      <div class="manage-body">
        <!-- ОБЗОР -->
        <section v-show="tab === 'overview'" class="pane pane-scroll">
          <AppStack>
            <AppGrid :min="200" :gap="12">
              <AppCard>
                <span class="ov-value">{{ company.employees_count }}</span>
                <span class="ov-label">сотрудников</span>
              </AppCard>
              <AppCard>
                <span class="ov-value">{{ company.tasks_count }}</span>
                <span class="ov-label">задач</span>
              </AppCard>
              <AppCard>
                <span class="ov-value">{{ fmtDate(company.created_at) }}</span>
                <span class="ov-label">создана</span>
              </AppCard>
            </AppGrid>

            <AppCard v-if="company.description" title="О компании">
              <p class="ov-desc">{{ company.description }}</p>
            </AppCard>
          </AppStack>
        </section>

        <!-- УЧАСТНИКИ -->
        <section v-show="tab === 'members'" class="pane pane-members">
          <AppCard
            title="Участники"
            hint="Роль определяет права в компании: сотрудник ведёт задачи, менеджер управляет юнитами и отделами, администратор — настройками."
          >
            <AppStack :gap="10" row>
              <AppChip tone="primary" icon="groups" :count="members.length" label="в компании" />
              <AppChip v-if="vacationCount" tone="warning" icon="beach_access" :count="vacationCount" label="в отпуске" />

              <template v-if="canManageMembers">
                <AppButton
                  class="members-add"
                  variant="filled"
                  size="sm"
                  icon="person_add"
                  label="Сотрудник"
                  @click="openAdd"
                />
                <AppButton
                  variant="glass"
                  size="sm"
                  icon="mail"
                  label="Пригласить"
                  @click="openInvite"
                />
                <AppButton
                  variant="glass"
                  size="sm"
                  icon="link"
                  label="Ссылка"
                  @click="openInviteLink"
                />
              </template>
            </AppStack>
          </AppCard>

          <AppInfoBar
            v-if="!canManageMembers"
            tone="info"
            message="Управлять участниками может только создатель компании. Вам доступны просмотр и настройки."
          />

          <EmptyState
            v-if="!members.length"
            size="sm"
            icon="groups"
            title="В компании пока нет участников"
            subtitle="Добавьте сотрудника или отправьте приглашение по почте."
          />

          <AppStack v-else :gap="8">
            <AppRow
              v-for="m in members"
              :key="m.id"
              :title="m.fio"
              inline
              :tone="m.on_vacation ? 'warning' : 'neutral'"
            >
              <template #lead>
                <span class="member-ava">{{ initials(m.fio) }}</span>
              </template>

              <template #hint>
                @{{ m.login }}<template v-if="m.post"> · {{ m.post }}</template>
                <template v-if="m.on_vacation"> · в отпуске</template>
              </template>

              <Select
                v-if="canManageMembers"
                :model-value="m.role?.id"
                :options="roleOptions"
                option-label="name"
                option-value="id"
                class="role-select"
                @update:model-value="(v) => changeRole(m, v)"
              />
              <span v-else class="role-pill">{{ m.role?.name }}</span>

              <AppButton
                v-if="canManageMembers"
                variant="icon"
                size="sm"
                :icon="m.on_vacation ? 'beach_access' : 'work'"
                :title="m.on_vacation ? 'Снять отпуск' : 'Отправить в отпуск'"
                :aria-label="m.on_vacation ? 'Снять отпуск' : 'Отправить в отпуск'"
                @click="changeVacation(m, !m.on_vacation)"
              />
              <AppButton
                v-if="canManageMembers"
                variant="icon"
                size="sm"
                icon="lock_reset"
                title="Сбросить пароль"
                aria-label="Сбросить пароль"
                @click="askResetPassword(m)"
              />
              <AppButton
                v-if="canManageMembers && !isOwner(m)"
                variant="icon"
                size="sm"
                tone="danger"
                icon="person_remove"
                title="Убрать из компании"
                aria-label="Убрать из компании"
                @click="askRemove(m)"
              />
            </AppRow>
          </AppStack>

          <p v-if="membersError" class="err">{{ membersError }}</p>
        </section>

        <!-- НАСТРОЙКИ -->
        <section v-show="tab === 'settings'" class="pane pane-settings">
          <AppTabs v-model="settingsTab" :tabs="settingsTabs" class="settings-subtabs" />

          <div class="settings-content pane-scroll">
            <AppCard v-show="settingsTab === 'features'" title="Возможности">
              <AppSwitchRow
                v-model="flags.uses_stages"
                title="Этапы задач"
                hint="Канбан-режим и цветные теги этапов"
                @update:model-value="saveFlags"
              />
              <AppSwitchRow
                v-model="flags.uses_yougile"
                title="Интеграция с YouGile"
                hint="Импорт и экспорт карточек"
                @update:model-value="saveFlags"
              />
              <AppSwitchRow
                v-model="flags.uses_calls"
                title="Аудио- и видеозвонки"
                hint="Кнопки звонка в мессенджере"
                @update:model-value="saveFlags"
              />
            </AppCard>

            <div v-show="settingsTab === 'lists'"><CompanyListsSettings :company-id="company.id" /></div>
            <div v-show="settingsTab === 'ai'"><AiSettings :company-id="company.id" :owner-id="company.creator?.id ?? company.created_by ?? null" /></div>
            <div v-show="settingsTab === 'schedule'"><WeekendSettings :company-id="company.id" /></div>
            <div v-show="settingsTab === 'groove'"><GrooveSettings :company-id="company.id" /></div>

            <div v-show="settingsTab === 'yougile'">
              <YougileCompanySettings v-if="company.id === auth.companyId" />
              <AppInfoBar
                v-else
                tone="info"
                message="Чтобы настроить интеграцию с YouGile, переключитесь на эту компанию — настройки привязаны к активной сессии."
              />
            </div>

            <div v-show="settingsTab === 'calendars'" class="settings-fill">
              <CalendarsManager v-if="company.id === auth.companyId" />
              <AppInfoBar
                v-else
                tone="info"
                message="Чтобы настроить календари, переключитесь на эту компанию — настройки привязаны к активной сессии."
              />
            </div>
          </div>
        </section>

        <!-- ПЕРЕНОС -->
        <section v-show="tab === 'transfer'" class="pane pane-scroll">
          <CompanyTransferCard
            :company-id="company.id"
            :company-name="company.name"
            @imported="onCompanyImported"
          />
        </section>

        <!-- ОПАСНАЯ ЗОНА -->
        <section v-show="tab === 'danger'" class="pane pane-scroll">
          <AppStack>
            <AppRow
              v-if="isSuper"
              tone="warning"
              :title="company.is_active ? 'Отключить компанию' : 'Включить компанию'"
              hint="Отключённая компания недоступна сотрудникам, но данные сохраняются."
            >
              <AppButton
                :label="company.is_active ? 'Отключить' : 'Включить'"
                :loading="toggling"
                @click="toggleActive"
              />
            </AppRow>

            <AppRow
              tone="danger"
              title="Удалить компанию"
              hint="Все данные удалятся каскадно: задачи, юниты, чаты, звонки. Действие необратимо."
            >
              <AppButton tone="danger" icon="delete" label="Удалить" @click="confirmDelete = true" />
            </AppRow>
          </AppStack>
        </section>
      </div>
    </template>

    <!-- Добавить сотрудника: существующий / новый -->
    <AppDialog
      v-model="addOpen"
      title="Добавить сотрудника"
      tone="primary"
      size="md"
      :busy="creatingUser"
      :closable="!creatingUser"
    >
      <div class="add-body">
        <AppTabs v-model="addTab" :tabs="addTabs" full-width class="add-subtabs" />

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
          <AppStack v-if="candidates.length" :gap="6">
            <AppRow
              v-for="c in candidates"
              :key="c.id"
              :title="c.fio"
              :hint="`@${c.login}`"
              dense
              clickable
              inline
              @click="addExisting(c)"
            >
              <template #lead>
                <span class="member-ava sm">{{ initials(c.fio) }}</span>
              </template>
              <span class="material-symbols-outlined add-ic">add</span>
            </AppRow>
          </AppStack>
          <p v-else-if="candQuery.trim()" class="hint">Никого не нашли — попробуйте другой запрос или вкладку «Новый».</p>
          <p v-else class="hint">Начните вводить имя или логин уже зарегистрированного пользователя.</p>
        </div>

        <!-- Новый -->
        <form v-show="addTab === 'new'" class="add-pane add-form" @submit.prevent="createUser">
          <div class="field">
            <label class="lbl">ФИО <span class="req">*</span></label>
            <input
              v-model.trim="newUser.fio"
              class="ctl"
              placeholder="Фамилия Имя Отчество"
              :disabled="creatingUser"
              @input="onNewUserFio"
            />
          </div>
          <div class="field">
            <label class="lbl">Логин <span class="req">*</span></label>
            <input
              v-model.trim="newUser.login"
              class="ctl"
              placeholder="Подставится из ФИО"
              :disabled="creatingUser"
              @input="loginTouched = true"
            />
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
          <!-- Пароль не выдумывается: он детерминирован (<логин>123), и
               человек должен видеть его ДО создания — иначе непонятно, что
               передавать сотруднику. -->
          <div class="new-pass">
            <span class="new-pass-label">Пароль для первого входа</span>
            <code class="new-pass-value">{{ newUserPassword || '—' }}</code>
            <span class="new-pass-hint">
              Сотрудник войдёт с ним и сразу задаст свой — сменить пароль потребуется при первом входе.
            </span>
          </div>

          <p v-if="createUserError" class="err">{{ createUserError }}</p>
        </form>
      </div>

      <template #footer>
        <AppButton label="Закрыть" :disabled="creatingUser" @click="addOpen = false" />
        <AppButton
          v-if="addTab === 'new'"
          class="foot-main"
          variant="filled"
          label="Создать"
          :loading="creatingUser"
          :disabled="!newUser.fio || !newUser.login"
          @click="createUser"
        />
      </template>
    </AppDialog>

    <!-- Пригласить по email -->
    <AppDialog
      v-model="inviteOpen"
      title="Пригласить по email"
      subtitle="Получатель перейдёт по ссылке из письма и вступит в компанию с выбранной ролью."
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
        <AppButton label="Отмена" :disabled="inviting" @click="inviteOpen = false" />
        <AppButton
          class="foot-main"
          variant="filled"
          icon="send"
          label="Отправить"
          :loading="inviting"
          :disabled="!invite.email"
          @click="sendEmailInvite"
        />
      </template>
    </AppDialog>

    <!-- Подтверждение удаления участника -->
    <AppDialog
      v-model="confirmRemove"
      tone="danger"
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

    <!-- Подтверждение сброса пароля -->
    <AppDialog
      v-model="confirmReset"
      tone="primary"
      size="sm"
      :title="`Сбросить пароль ${resetTarget?.fio || 'сотрудника'}?`"
      :busy="resetting"
      :closable="!resetting"
      :actions="[
        { kind: 'cancel', label: 'Отмена', disabled: resetting },
        { kind: 'confirm', label: 'Сбросить', icon: 'lock_reset', disabled: resetting },
      ]"
      @confirm="doResetPassword"
    >
      <p>Текущий пароль сотрудника перестанет действовать. Ему будет назначен временный пароль, который потребуется сменить при первом входе.</p>
      <div class="reset-temp">
        <span class="reset-temp-label">Временный пароль</span>
        <div class="reset-temp-value">
          <code>{{ tempPassword }}</code>
          <AppButton
            variant="icon"
            size="sm"
            :icon="resetCopied ? 'check' : 'content_copy'"
            :title="resetCopied ? 'Скопировано' : 'Скопировать'"
            aria-label="Скопировать временный пароль"
            @click="copyTempPassword"
          />
        </div>
        <span class="reset-temp-hint">Передайте его сотруднику — при входе он введёт этот пароль и сразу задаст свой.</span>
      </div>
    </AppDialog>

    <!-- Ссылка-приглашение -->
    <AppDialog
      v-model="inviteLinkOpen"
      title="Ссылка-приглашение"
      subtitle="Любой авторизованный пользователь, перешедший по ссылке, вступит в компанию как Сотрудник. Перевыпуск делает старую ссылку недействительной."
      tone="primary"
      size="md"
    >
      <div class="invite-link-body">
        <div class="invite-link-row">
          <input class="ctl" :value="inviteUrl" readonly placeholder="Ссылка ещё не создана" />
          <AppButton
            variant="icon"
            size="sm"
            :icon="inviteCopied ? 'check' : 'content_copy'"
            :disabled="!inviteCode"
            title="Скопировать"
            aria-label="Скопировать ссылку"
            @click="copyInviteLink"
          />
        </div>
        <AppButton
          :icon="inviteCode ? 'autorenew' : 'add_link'"
          :label="inviteCode ? 'Перевыпустить' : 'Создать ссылку'"
          :loading="inviteLinkBusy"
          @click="regenInviteLink"
        />
        <p v-if="inviteLinkError" class="err">{{ inviteLinkError }}</p>
      </div>
    </AppDialog>

    <AppDialog
      v-model="confirmDelete"
      tone="danger"
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
import { useRoute } from 'vue-router'
import BrandLoader from '@/components/common/BrandLoader.vue'
import Select from 'primevue/select'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppGrid from '@/components/ui/AppGrid.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppSwitchRow from '@/components/ui/AppSwitchRow.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import CompanyTransferCard from '@/components/settings/CompanyTransferCard.vue'
import GrooveSettings from '@/components/settings/GrooveSettings.vue'
import WeekendSettings from '@/components/settings/WeekendSettings.vue'
import AiSettings from '@/components/settings/AiSettings.vue'
import CompanyListsSettings from '@/components/settings/CompanyListsSettings.vue'
import YougileCompanySettings from '@/components/settings/YougileCompanySettings.vue'
import CalendarsManager from '@/components/calendar/CalendarsManager.vue'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { usePermission } from '@/composables/usePermission.js'
import {
  getCompany, updateCompany, deleteCompany, toggleCompanyActive,
  listCompanyMembers, getCompanyCandidates, addCompanyMember, setMemberRole, removeCompanyMember,
  createCompanyUser, updateCompanyMember, resetCompanyMemberPassword, createCompanyInvite,
  getCompanyInvite, regenerateCompanyInvite,
} from '@/api/companies.js'
import { getRoles } from '@/api/roles.js'
import { suggestLogin } from '@/api/auth.js'

const props = defineProps({ id: { type: [String, Number], required: true } })
const emit = defineEmits(['back', 'deleted', 'imported'])

// Открыть сразу на нужной вкладке по ссылке (например, пилюля YouGile в
// «Аккаунте» ведёт прямо в ?tab=settings&settingsTab=yougile, а не на общую
// страницу компании, которую потом надо разгадывать самому).
const route = useRoute()

const auth = useAuthStore()
const notif = useNotificationsStore()
const { isSuperAdmin, ROLES } = usePermission()
const isSuper = computed(() => isSuperAdmin())

const companyId = computed(() => Number(props.id))
const company = ref(null)
const loading = ref(true)
const loadError = ref('')

const vacationCount = computed(() => members.value.filter((m) => m.on_vacation).length)

const MAIN_TAB_KEYS = ['overview', 'members', 'settings', 'transfer', 'danger']
const tab = ref(MAIN_TAB_KEYS.includes(route.query.tab) ? route.query.tab : 'overview')
const mainTabs = computed(() => {
  const list = [
    { value: 'overview', label: 'Обзор', icon: 'info' },
    { value: 'members', label: 'Участники', icon: 'groups' },
    { value: 'settings', label: 'Настройки', icon: 'tune' },
  ]
  if (canManageMembers.value) {
    list.push({ value: 'transfer', label: 'Перенос', icon: 'swap_horiz' })
    list.push({ value: 'danger', label: 'Опасная зона', icon: 'warning' })
  }
  return list
})

/* Реестров здесь НЕТ намеренно: они принадлежат человеку, а не компании, и
   настраиваются в самом разделе (структуру правит владелец или тот, кому выдан
   уровень «администрирование»). */
const SETTINGS_TAB_KEYS = ['features', 'lists', 'ai', 'schedule', 'groove', 'calendars', 'yougile']
const settingsTab = ref(SETTINGS_TAB_KEYS.includes(route.query.settingsTab) ? route.query.settingsTab : 'features')
const settingsTabs = [
  { value: 'features', label: 'Возможности', icon: 'tune' },
  { value: 'lists', label: 'Списки', icon: 'format_list_bulleted' },
  { value: 'ai', label: 'ИИ', icon: 'smart_toy' },
  { value: 'schedule', label: 'Расписание', icon: 'weekend' },
  { value: 'groove', label: 'Мой Groove', icon: 'celebration' },
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

/* Логин подставляется из ФИО теми же правилами, что и при регистрации
   (транслит на сервере), пока его не начали править руками. Пароль отсюда же:
   он детерминирован — <логин>123. */
const loginTouched = ref(false)
let suggestTimer = null

const newUserPassword = computed(() => (newUser.value.login ? `${newUser.value.login}123` : ''))

function onNewUserFio() {
  if (loginTouched.value) return
  clearTimeout(suggestTimer)
  const fio = newUser.value.fio
  suggestTimer = setTimeout(async () => {
    if (loginTouched.value || !fio.trim()) return
    try {
      const { login } = await suggestLogin(fio)
      if (!loginTouched.value && login) newUser.value.login = login
    } catch { /* подсказка необязательна: логин можно ввести руками */ }
  }, 400)
}

const inviteOpen = ref(false)
const invite = ref({ email: '', roleId: ROLES.EMPLOYEE })
const inviting = ref(false)
const inviteError = ref('')

// Подтверждение удаления участника.
const confirmRemove = ref(false)
const removeTarget = ref(null)
const removing = ref(false)

// Подтверждение сброса пароля. Временный пароль детерминирован — <логин>123
// (совпадает с authsvc ResetPassword), показываем его администратору, чтобы
// было что передать сотруднику.
const confirmReset = ref(false)
const resetTarget = ref(null)
const resetting = ref(false)
const resetCopied = ref(false)
const tempPassword = computed(() => (resetTarget.value ? `${resetTarget.value.login}123` : ''))

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

function goBack() { emit('back') }

// Архив поднят новой компанией — список компаний должен её показать.
function onCompanyImported(result) { emit('imported', result) }

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

// Отпуск сотрудника явно проставляет/снимает создатель компании (или
// супер-админ) — тот же режим, что тумблер в личном профиле.
async function changeVacation(m, v) {
  if (!!m.on_vacation === v) return
  membersError.value = ''
  try {
    await updateCompanyMember(companyId.value, m.id, { on_vacation: v })
    m.on_vacation = v
  } catch (e) {
    membersError.value = e?.message || 'Не удалось изменить режим отпуска'
    await loadMembers()
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

function askResetPassword(m) {
  resetTarget.value = m
  resetCopied.value = false
  confirmReset.value = true
}

async function copyTempPassword() {
  if (!tempPassword.value) return
  try {
    await navigator.clipboard.writeText(tempPassword.value)
    resetCopied.value = true
    setTimeout(() => { resetCopied.value = false }, 1500)
  } catch { /* ignore */ }
}

async function doResetPassword() {
  const m = resetTarget.value
  if (!m) return
  resetting.value = true
  try {
    await resetCompanyMemberPassword(companyId.value, m.id)
    notif.success(`Пароль ${m.fio} сброшен на временный`)
    confirmReset.value = false
  } catch (e) {
    notif.error(e?.message || 'Не удалось сбросить пароль')
  } finally {
    resetting.value = false
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
    emit('deleted')
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить')
    deleting.value = false
  }
}
</script>

<style scoped>
/* Панель внутри настроек: внешние поля и фон даёт AppPage настроек. */
.manage-page {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.manage-head { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; flex: none; }
.head-text { display: flex; align-items: center; gap: 12px; min-width: 0; flex-wrap: wrap; }
.head-title { margin: 0; font-size: 22px; font-weight: 800; color: var(--color-text); }

.manage-tabs { flex: none; margin-bottom: 16px; }

.manage-body { flex: 1; min-height: 0; display: flex; }

/* min-width:0 — чтобы дочерние SegmentedTabs/таблицы могли сжаться до ширины
   экрана и скроллиться, а не распирать панель за край (особенно на мобильном). */
.pane { flex: 1; min-width: 0; min-height: 0; display: flex; flex-direction: column; gap: 18px; }
.pane-scroll { overflow-y: auto; }
.ov-desc { color: var(--color-text); font-size: 14px; line-height: 1.5; }

/* Временный пароль в диалоге сброса. */
.reset-temp {
  margin-top: 14px; display: flex; flex-direction: column; gap: 6px;
  padding: 12px 14px; border-radius: var(--radius-md, 12px);
  background: var(--color-primary-container);
}
.new-pass {
  display: grid;
  gap: 4px;
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  background: var(--color-surface-variant);
}

.new-pass-label { font-size: 0.82rem; color: var(--color-text-dim); }

.new-pass-value {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.95rem;
  overflow-wrap: anywhere;
}

.new-pass-hint { font-size: 0.8rem; color: var(--color-text-dim); }

.reset-temp-label { font-size: 12px; font-weight: 600; color: var(--color-on-primary-container); opacity: 0.85; }
.reset-temp-value { display: flex; align-items: center; gap: 8px; }
.reset-temp-value code {
  flex: 1; min-width: 0; font-family: var(--font-mono, monospace); font-size: 16px; font-weight: 700;
  color: var(--color-on-primary-container); letter-spacing: 0.5px; word-break: break-all;
}
.reset-temp-hint { font-size: 12px; color: var(--color-on-primary-container); opacity: 0.8; line-height: 1.4; }

/* ── Участники: таблица занимает всю высоту и скроллится отдельно ── */
.pane-members { gap: 14px; }
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

.err { margin: 0; font-size: 13px; color: var(--color-error); flex: none; }

/* ── Настройки с под-вкладками ── */
.pane-settings { gap: 16px; }
.settings-subtabs { flex: none; }
.settings-content { flex: 1; min-height: 0; display: flex; flex-direction: column; gap: 18px; }
/* Вкладка реестров сама держит высоту и внутреннюю прокрутку — растягиваем её
   обёртку на всю область настроек, чтобы конструктор не растягивал .settings-content. */
.settings-fill { flex: 1; min-height: 0; display: flex; }

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

.hint { margin: 0; font-size: 13px; color: var(--color-text-dim); line-height: 1.5; }

.invite-link-body { display: flex; flex-direction: column; gap: 14px; align-items: flex-start; }
.invite-link-row { display: flex; gap: 8px; align-items: center; width: 100%; }
.invite-link-row .ctl { flex: 1; font-size: 13px; }

.w-full { width: 100%; }

.state-block { flex: 1; display: grid; place-items: center; padding: 64px; gap: 10px; color: var(--color-text-dim); }

@media (max-width: 768px) {
  /* Панель внутри настроек: внешние поля и фон даёт AppPage настроек. */
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

.ov-value { font-size: 22px; font-weight: 800; letter-spacing: -0.01em; }
.ov-label { font-size: 13px; color: var(--color-text-dim); }
.ov-desc { margin: 0; font-size: 14px; line-height: 1.55; }

/* Главное действие подвала прижимаем вправо (слева — «Отмена»/«Закрыть»). */
.foot-main { margin-left: auto; }
</style>
