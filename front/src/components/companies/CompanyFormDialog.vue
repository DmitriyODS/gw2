<template>
  <AppDialog
    :model-value="modelValue"
    tone="primary"
    :icon="isEdit ? 'edit' : 'add_business'"
    size="md"
    :title="isEdit ? 'Редактирование компании' : 'Новая компания'"
    :busy="saving"
    :closable="!saving"
    :actions="[
      { kind: 'cancel', label: 'Отмена', disabled: saving },
      { kind: 'confirm', label: isEdit ? 'Сохранить' : 'Создать', disabled: !canSave || saving },
    ]"
    @update:model-value="onClose"
    @confirm="save"
  >
    <div class="form-body">
      <div class="field">
        <label class="lbl">Название <span class="req">*</span></label>
        <input
          v-model.trim="form.name"
          class="ctl"
          maxlength="255"
          placeholder="ООО «Ромашка»"
          :class="{ invalid: !!errors.name }"
        />
        <div v-if="errors.name" class="err">{{ errors.name }}</div>
      </div>

      <div class="field">
        <label class="lbl">Описание</label>
        <textarea
          v-model.trim="form.description"
          class="ctl ctl-area"
          rows="2"
          placeholder="Несколько слов о компании (необязательно)"
        />
      </div>

      <div class="field">
        <label class="lbl">Настройки</label>
        <div class="switch-list">
          <label class="switch-row">
            <span class="switch-text">
              <span class="material-symbols-outlined">view_kanban</span>
              <span>
                <strong>Этапы задач</strong>
                <small>Канбан-режим, цветные теги этапов в карточках</small>
              </span>
            </span>
            <input type="checkbox" v-model="form.settings.uses_stages" class="switch" />
          </label>
          <label class="switch-row">
            <span class="switch-text">
              <span class="material-symbols-outlined">link</span>
              <span>
                <strong>Интеграция с YouGile</strong>
                <small>Импорт/экспорт карточек, бейдж и кнопки в задачах. Если выключено — остаётся обычное поле «Ссылка на YouGile»</small>
              </span>
            </span>
            <input type="checkbox" v-model="form.settings.uses_yougile" class="switch" />
          </label>
          <label class="switch-row">
            <span class="switch-text">
              <span class="material-symbols-outlined">call</span>
              <span>
                <strong>Аудио/видео-звонки</strong>
                <small>Кнопки звонка в мессенджере и профилях</small>
              </span>
            </span>
            <input type="checkbox" v-model="form.settings.uses_calls" class="switch" />
          </label>
          <label class="switch-row">
            <span class="switch-text">
              <span class="material-symbols-outlined">celebration</span>
              <span>
                <strong>Мой Groove</strong>
                <small>Геймификация: питомцы-Грувики, лента активности, кудосы и рейды</small>
              </span>
            </span>
            <input type="checkbox" v-model="form.settings.uses_groove" class="switch" />
          </label>
        </div>
      </div>

      <div v-if="isEdit" class="field">
        <label class="lbl">Участники</label>
        <div class="members">
          <div v-if="!members.length" class="members-empty">В компании пока только создатель и добавленные сотрудники.</div>
          <div v-for="m in members" :key="m.id" class="member-row">
            <span class="member-ava">{{ initials(m.fio) }}</span>
            <span class="member-main">
              <span class="member-name">{{ m.fio }}</span>
              <span class="member-login">@{{ m.login }}</span>
            </span>
            <select
              class="ctl member-role"
              :value="m.role?.id"
              @change="changeRole(m, Number($event.target.value))"
            >
              <option v-for="r in roleOptions" :key="r.id" :value="r.id">{{ r.name }}</option>
            </select>
            <button type="button" class="member-del" title="Убрать из компании" @click="removeMember(m)">
              <span class="material-symbols-outlined">close</span>
            </button>
          </div>
        </div>

        <div class="member-add">
          <div class="member-add-search">
            <span class="material-symbols-outlined">person_search</span>
            <input
              v-model="candQuery"
              class="ctl"
              type="text"
              placeholder="Добавить существующего: имя или логин…"
              @input="onCandQuery"
            />
          </div>
          <div v-if="candidates.length" class="cand-list">
            <button
              v-for="c in candidates"
              :key="c.id"
              type="button"
              class="cand-item"
              @click="addMember(c)"
            >
              <span class="member-name">{{ c.fio }}</span>
              <span class="member-login">@{{ c.login }}</span>
              <span class="material-symbols-outlined">add</span>
            </button>
          </div>
        </div>
        <div v-if="membersError" class="err">{{ membersError }}</div>
      </div>

      <div v-if="isEdit" class="field">
        <label class="lbl">Ссылка-приглашение</label>
        <div class="invite">
          <input class="ctl invite-input" :value="inviteUrl" readonly placeholder="Ссылка ещё не создана" />
          <button type="button" class="invite-btn" :disabled="!inviteCode" title="Скопировать" @click="copyInvite">
            <span class="material-symbols-outlined">content_copy</span>
          </button>
          <button type="button" class="invite-btn" :disabled="inviteBusy" :title="inviteCode ? 'Перевыпустить' : 'Создать'" @click="regenInvite">
            <span class="material-symbols-outlined">{{ inviteCode ? 'autorenew' : 'add_link' }}</span>
          </button>
        </div>
        <div class="hint">
          Любой авторизованный пользователь, перешедший по ссылке, вступит в компанию как Сотрудник.
          Перевыпуск ссылки делает старую недействительной.
        </div>
      </div>

      <div v-if="serverError" class="form-err">{{ serverError }}</div>
    </div>
  </AppDialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import {
  listCompanyMembers, getCompanyCandidates, addCompanyMember,
  setMemberRole, removeCompanyMember, getCompanyInvite, regenerateCompanyInvite,
} from '@/api/companies.js'
import { getRoles } from '@/api/roles.js'

const props = defineProps({
  modelValue: { type: Boolean, required: true },
  company: { type: Object, default: null },
})
const emit = defineEmits(['update:modelValue', 'save'])

const isEdit = computed(() => !!props.company?.id)

const form = ref(_blank())
const errors = ref({})
const serverError = ref('')
const saving = ref(false)

// Участники + ссылка-приглашение (только в режиме редактирования).
const members = ref([])
const roleOptions = ref([])
const membersError = ref('')
const candQuery = ref('')
const candidates = ref([])
let candTimer = null
const inviteCode = ref('')
const inviteBusy = ref(false)

const inviteUrl = computed(() =>
  inviteCode.value ? `${window.location.origin}/join/${inviteCode.value}` : '')

function initials(fio) {
  return (fio || '').trim().split(/\s+/).slice(0, 2).map((p) => p[0]?.toUpperCase() || '').join('')
}

function _blank() {
  return {
    name: '',
    description: '',
    settings: { uses_stages: false, uses_yougile: false, uses_calls: true, uses_groove: true },
  }
}

watch(() => props.modelValue, (v) => {
  if (!v) return
  errors.value = {}
  serverError.value = ''
  if (props.company) {
    form.value = {
      name: props.company.name || '',
      description: props.company.description || '',
      settings: {
        uses_stages: !!props.company.settings?.uses_stages,
        uses_yougile: !!props.company.settings?.uses_yougile,
        uses_calls: props.company.settings?.uses_calls !== false,
        uses_groove: props.company.settings?.uses_groove !== false,
      },
    }
  } else {
    form.value = _blank()
  }
  members.value = []
  candidates.value = []
  candQuery.value = ''
  membersError.value = ''
  inviteCode.value = ''
  if (props.company?.id) {
    loadMembers()
    loadRoleOptions()
    loadInvite()
  }
}, { immediate: false })

async function loadMembers() {
  try {
    members.value = await listCompanyMembers(props.company.id)
  } catch (e) {
    membersError.value = e?.message || 'Не удалось загрузить участников'
  }
}

async function loadRoleOptions() {
  try {
    // Все роли — компанийные (Сотрудник/Менеджер/Администратор); платформенный
    // супер-админ задаётся флагом, а не ролью, и в списке ролей не присутствует.
    roleOptions.value = (await getRoles()) || []
  } catch {
    roleOptions.value = []
  }
}

async function loadInvite() {
  try {
    const { code } = await getCompanyInvite(props.company.id)
    inviteCode.value = code || ''
  } catch {
    inviteCode.value = ''
  }
}

function onCandQuery() {
  if (candTimer) clearTimeout(candTimer)
  candTimer = setTimeout(searchCandidates, 250)
}

async function searchCandidates() {
  const q = candQuery.value.trim()
  if (!q) { candidates.value = []; return }
  try {
    candidates.value = await getCompanyCandidates(props.company.id, q)
  } catch {
    candidates.value = []
  }
}

async function addMember(c) {
  membersError.value = ''
  const employeeRole = roleOptions.value.find((r) => r.level === 1) || roleOptions.value[0]
  try {
    await addCompanyMember(props.company.id, c.id, employeeRole.id)
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
    await setMemberRole(props.company.id, m.id, roleId)
    await loadMembers()
  } catch (e) {
    membersError.value = e?.message || 'Не удалось сменить роль'
    await loadMembers()
  }
}

async function removeMember(m) {
  membersError.value = ''
  try {
    await removeCompanyMember(props.company.id, m.id)
    await loadMembers()
  } catch (e) {
    membersError.value = e?.message || 'Не удалось убрать'
  }
}

async function regenInvite() {
  inviteBusy.value = true
  try {
    const { code } = await regenerateCompanyInvite(props.company.id)
    inviteCode.value = code || ''
  } catch (e) {
    membersError.value = e?.message || 'Не удалось создать ссылку'
  } finally {
    inviteBusy.value = false
  }
}

async function copyInvite() {
  if (!inviteUrl.value) return
  try { await navigator.clipboard.writeText(inviteUrl.value) } catch { /* ignore */ }
}

const canSave = computed(() => form.value.name.trim().length >= 1)

function validate() {
  errors.value = {}
  if (!form.value.name.trim()) errors.value.name = 'Введите название'
  return Object.keys(errors.value).length === 0
}

async function save() {
  if (!validate()) return
  serverError.value = ''
  saving.value = true
  try {
    const payload = {
      name: form.value.name.trim(),
      description: form.value.description.trim() || null,
      settings: { ...form.value.settings },
    }
    emit('save', { payload, isEdit: isEdit.value, id: props.company?.id ?? null })
  } finally {
    saving.value = false
  }
}

function onClose() {
  if (saving.value) return
  emit('update:modelValue', false)
}

defineExpose({
  showError(message) { serverError.value = message; saving.value = false },
  finish() { saving.value = false },
})
</script>

<style scoped>
.form-body { display: flex; flex-direction: column; gap: 16px; padding: 4px 0 8px; }

.field { display: flex; flex-direction: column; gap: 6px; }
.lbl { font-size: 13px; font-weight: 600; color: var(--color-on-surface-variant); }
.req { color: var(--color-error); }

.ctl {
  appearance: none;
  width: 100%;
  border: 1px solid var(--color-outline-variant);
  background: var(--acrylic-card-bg);
  color: var(--color-on-surface);
  padding: 10px 12px;
  border-radius: var(--radius-md, 12px);
  font: inherit;
  transition: border-color .15s, box-shadow .15s;
}
.ctl:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px color-mix(in oklab, var(--color-primary) 18%, transparent);
}
.ctl.invalid { border-color: var(--color-error); }
.ctl-area { resize: vertical; min-height: 56px; }

select.ctl {
  background: var(--color-surface) url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='10' height='6'><path d='M0 0l5 6 5-6z' fill='%23999'/></svg>") no-repeat right 12px center;
  padding-right: 32px;
}

.hint { font-size: 12px; color: var(--color-on-surface-variant); line-height: 1.4; }
.err { font-size: 12px; color: var(--color-error); }
.form-err {
  padding: 10px 12px;
  border-radius: var(--radius-md, 12px);
  background: var(--color-error-container);
  color: var(--color-on-error-container);
  font-size: 14px;
}

.switch-list { display: flex; flex-direction: column; gap: 6px; }
.switch-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  background: var(--color-surface-container);
  border-radius: var(--radius-md, 12px);
  cursor: pointer;
  transition: background .12s;
}
.switch-row:hover { background: var(--color-surface-high); }
.switch-text {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.switch-text .material-symbols-outlined {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md, 12px);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 20px;
  flex: none;
}
.switch-text strong { display: block; font-size: 14px; color: var(--color-on-surface); }
.switch-text small { display: block; font-size: 12px; color: var(--color-on-surface-variant); }

/* M3 Expressive switch — синхронизирован с .toggle в CompaniesView. */
.switch {
  appearance: none;
  width: 44px;
  height: 24px;
  border-radius: 999px;
  background: var(--color-surface-highest, var(--color-surface-high));
  border: 2px solid var(--color-outline, var(--color-outline-variant));
  box-sizing: border-box;
  position: relative;
  cursor: pointer;
  outline: none;
  transition: background .18s, border-color .18s;
  flex: none;
}
.switch::after {
  content: '';
  position: absolute;
  top: 50%;
  left: 4px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--color-outline, var(--color-on-surface-variant));
  transform: translateY(-50%);
  transition: transform .2s cubic-bezier(0.4, 0, 0.2, 1),
              background .2s, width .2s, height .2s, left .2s;
}
.switch:checked {
  background: var(--color-primary);
  border-color: var(--color-primary);
}
.switch:checked::after {
  width: 16px;
  height: 16px;
  left: 24px;
  background: var(--color-on-primary);
}

/* Участники */
.members { display: flex; flex-direction: column; gap: 6px; }
.members-empty { font-size: 12px; color: var(--color-on-surface-variant); padding: 4px 2px; }
.member-row {
  display: flex; align-items: center; gap: 10px;
  padding: 6px 8px; border-radius: var(--radius-md, 12px);
  background: var(--color-surface-container);
}
.member-ava {
  width: 32px; height: 32px; flex: none; border-radius: 50%;
  display: grid; place-items: center; font-size: 12px; font-weight: 700;
  background: var(--color-primary-container); color: var(--color-on-primary-container);
}
.member-main { display: flex; flex-direction: column; min-width: 0; flex: 1; }
.member-name { font-size: 14px; font-weight: 600; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.member-login { font-size: 12px; color: var(--color-on-surface-variant); }
.member-role { width: auto; min-width: 130px; padding: 6px 28px 6px 10px; }
.member-del {
  flex: none; display: grid; place-items: center; width: 30px; height: 30px; min-height: 0;
  border: none; background: transparent; color: var(--color-on-surface-variant);
  border-radius: 50%; cursor: pointer;
}
.member-del:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
.member-del .material-symbols-outlined { font-size: 18px; }

.member-add { margin-top: 8px; display: flex; flex-direction: column; gap: 6px; }
.member-add-search { position: relative; display: flex; align-items: center; }
.member-add-search > .material-symbols-outlined {
  position: absolute; left: 10px; font-size: 18px; color: var(--color-on-surface-variant); pointer-events: none;
}
.member-add-search .ctl { padding-left: 36px; }
.cand-list { display: flex; flex-direction: column; gap: 4px; max-height: 180px; overflow-y: auto; }
.cand-item {
  display: flex; align-items: center; gap: 8px; padding: 8px 10px; text-align: left;
  border: 1px solid var(--color-outline-variant); border-radius: var(--radius-md, 12px);
  background: var(--acrylic-card-bg); color: var(--color-on-surface); cursor: pointer;
}
.cand-item:hover { border-color: var(--color-primary); }
.cand-item .member-login { flex: 1; }
.cand-item .material-symbols-outlined { font-size: 18px; color: var(--color-primary); }

/* Ссылка-приглашение */
.invite { display: flex; gap: 8px; align-items: center; }
.invite-input { flex: 1; font-size: 13px; }
.invite-btn {
  flex: none; display: grid; place-items: center; width: 40px; height: 40px;
  border: 1px solid var(--color-outline-variant); border-radius: var(--radius-md, 12px);
  background: var(--acrylic-card-bg); color: var(--color-on-surface); cursor: pointer;
}
.invite-btn:hover:not(:disabled) { border-color: var(--color-primary); color: var(--color-primary); }
.invite-btn:disabled { opacity: .5; cursor: not-allowed; }
.invite-btn .material-symbols-outlined { font-size: 20px; }

</style>
