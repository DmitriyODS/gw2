<template>
  <Dialog
    :visible="modelValue"
    @update:visible="onClose"
    :header="isEdit ? 'Редактирование сотрудника' : 'Новый сотрудник'"
    modal
    :style="{ width: '520px' }"
    :pt="{ content: { style: 'overflow: visible' } }"
  >
    <div class="form">
      <div class="grid-2">
        <div class="field">
          <label class="lbl">ФИО <span class="req">*</span></label>
          <input v-model.trim="form.fio" class="ctl" maxlength="255"
                 :class="{ invalid: errors.fio }" placeholder="Иванов Иван Иванович" />
          <div v-if="errors.fio" class="err">{{ errors.fio }}</div>
        </div>
        <div class="field">
          <label class="lbl">Логин <span class="req">*</span></label>
          <input v-model.trim="form.login" class="ctl" maxlength="100"
                 :disabled="isEdit"
                 :class="{ invalid: errors.login }" placeholder="ivan.ivanov" />
          <div v-if="errors.login" class="err">{{ errors.login }}</div>
          <div v-else-if="isEdit" class="hint">Логин нельзя изменить после создания.</div>
        </div>
      </div>

      <div class="field">
        <label class="lbl">Должность</label>
        <input v-model.trim="form.post" class="ctl" maxlength="255" placeholder="Дизайнер" />
      </div>

      <div class="grid-2">
        <div class="field">
          <label class="lbl">Телефон</label>
          <PhoneInput v-model="form.phone" :invalid="!!errors.phone" />
          <div v-if="errors.phone" class="err">{{ errors.phone }}</div>
        </div>
        <div class="field">
          <label class="lbl">Email</label>
          <input v-model.trim="form.email" type="email" class="ctl"
                 :class="{ invalid: errors.email }" placeholder="ivan@example.com" />
          <div v-if="errors.email" class="err">{{ errors.email }}</div>
        </div>
      </div>

      <div class="grid-2">
        <div v-if="canPickCompany" class="field">
          <label class="lbl">Компания</label>
          <select v-model="form.company_id" class="ctl">
            <option :value="null">— без компании —</option>
            <option v-for="c in companies.items" :key="c.id" :value="c.id">
              {{ c.name }}
            </option>
          </select>
        </div>
        <div v-if="!isEdit" class="field">
          <label class="lbl">Пароль</label>
          <input v-model="form.password" type="password" class="ctl"
                 :class="{ invalid: errors.password }" placeholder="Минимум 8 символов" />
          <div class="hint">
            Пусто — выдадим пароль по умолчанию, человек сменит его при первом входе.
          </div>
          <div v-if="errors.password" class="err">{{ errors.password }}</div>
        </div>
      </div>

      <div class="field">
        <label class="lbl">Роль <span class="req">*</span></label>
        <div class="role-chips">
          <label
            v-for="r in assignableRoles" :key="r.id"
            class="role-chip"
            :class="{ active: form.role_id === r.id, locked: r.locked }"
          >
            <input
              type="radio"
              :value="r.id"
              v-model="form.role_id"
              :disabled="r.locked"
            />
            <span class="material-symbols-rounded">{{ roleIcon(r.level) }}</span>
            <span>{{ r.name }}</span>
          </label>
        </div>
        <div v-if="errors.role_id" class="err">{{ errors.role_id }}</div>
      </div>

      <div v-if="serverError" class="form-err">{{ serverError }}</div>
    </div>

    <template #footer>
      <button class="btn-text" :disabled="saving" @click="onClose">Отмена</button>
      <button class="btn-filled" :disabled="!canSave || saving" @click="save">
        <span v-if="saving" class="material-symbols-rounded spin">progress_activity</span>
        {{ isEdit ? 'Сохранить' : 'Создать' }}
      </button>
    </template>
  </Dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import Dialog from 'primevue/dialog'
import PhoneInput from '@/components/common/PhoneInput.vue'
import { useAuthStore } from '@/stores/auth.js'
import { useCompaniesStore } from '@/stores/companies.js'
import { usePermission } from '@/composables/usePermission.js'

const props = defineProps({
  modelValue: { type: Boolean, required: true },
  user: { type: Object, default: null },
  roles: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:modelValue', 'save'])

const auth = useAuthStore()
const companies = useCompaniesStore()
const { myLevel } = usePermission()

const isEdit = computed(() => !!props.user?.id)
const canPickCompany = computed(() => auth.isRootAdmin)

const form = ref(_blank())
const errors = ref({})
const serverError = ref('')
const saving = ref(false)

function _blank() {
  return {
    fio: '', login: '', post: '', password: '',
    phone: '', email: '', role_id: null,
    company_id: auth.isRootAdmin ? null : auth.companyId,
  }
}

watch(() => props.modelValue, (v) => {
  if (!v) return
  errors.value = {}
  serverError.value = ''
  if (props.user) {
    form.value = {
      fio: props.user.fio || '',
      login: props.user.login || '',
      post: props.user.post || '',
      password: '',
      phone: props.user.phone || '',
      email: props.user.email || '',
      role_id: props.user.role?.id ?? null,
      company_id: props.user.company_id ?? null,
    }
  } else {
    form.value = _blank()
  }
  if (auth.isRootAdmin) companies.load()
})

const assignableRoles = computed(() => {
  const lvl = myLevel()
  return props.roles
    .filter(r => r.level <= lvl)
    .map(r => ({ ...r, locked: false }))
    .sort((a, b) => a.level - b.level)
})

function roleIcon(level) {
  if (level >= 4) return 'workspace_premium'
  if (level >= 3) return 'shield_person'
  if (level >= 2) return 'badge'
  return 'person'
}

const canSave = computed(() =>
  form.value.fio.trim().length >= 1 &&
  (isEdit.value || form.value.login.trim().length >= 3) &&
  form.value.role_id != null
)

function validate() {
  errors.value = {}
  if (!form.value.fio.trim()) errors.value.fio = 'Введите ФИО'
  if (!isEdit.value) {
    if (!form.value.login || form.value.login.length < 3) {
      errors.value.login = 'Минимум 3 символа'
    }
    if (form.value.password && form.value.password.length < 8) {
      errors.value.password = 'Минимум 8 символов'
    }
  }
  if (form.value.email && !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.value.email)) {
    errors.value.email = 'Некорректный email'
  }
  if (form.value.phone && form.value.phone.length < 12) {
    errors.value.phone = 'Полностью введите телефон или очистите поле'
  }
  if (form.value.role_id == null) errors.value.role_id = 'Выберите роль'
  return Object.keys(errors.value).length === 0
}

function save() {
  if (!validate()) return
  serverError.value = ''
  saving.value = true
  const payload = {
    fio: form.value.fio.trim(),
    post: form.value.post.trim() || null,
    phone: form.value.phone || null,
    email: form.value.email || null,
  }
  if (!isEdit.value) {
    payload.login = form.value.login.trim()
    payload.role_id = form.value.role_id
    if (form.value.password) payload.password = form.value.password
  }
  if (auth.isRootAdmin) payload.company_id = form.value.company_id ?? null
  emit('save', {
    payload,
    isEdit: isEdit.value,
    userId: props.user?.id ?? null,
    // Для существующего пользователя — отдельный апдейт роли (отдельный endpoint).
    newRoleId: isEdit.value && form.value.role_id !== props.user?.role?.id
      ? form.value.role_id : null,
  })
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
.form { display: flex; flex-direction: column; gap: 14px; padding: 4px 0 8px; }
.grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
@media (max-width: 520px) { .grid-2 { grid-template-columns: 1fr; } }

.field { display: flex; flex-direction: column; gap: 6px; }
.lbl { font-size: 13px; font-weight: 600; color: var(--color-on-surface-variant); }
.req { color: var(--color-error); }
.hint { font-size: 12px; color: var(--color-on-surface-variant); }
.err { font-size: 12px; color: var(--color-error); }
.form-err {
  padding: 10px 12px;
  border-radius: var(--radius-md, 12px);
  background: var(--color-error-container);
  color: var(--color-on-error-container);
  font-size: 14px;
}

.ctl {
  appearance: none;
  width: 100%;
  border: 1px solid var(--color-outline-variant);
  background: var(--color-surface);
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
.ctl:disabled { background: var(--color-surface-container); cursor: not-allowed; }
.ctl.invalid { border-color: var(--color-error); }
select.ctl {
  background: var(--color-surface) url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='10' height='6'><path d='M0 0l5 6 5-6z' fill='%23999'/></svg>") no-repeat right 12px center;
  padding-right: 32px;
}

.role-chips { display: flex; flex-wrap: wrap; gap: 6px; }
.role-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: var(--radius-full, 999px);
  background: var(--color-surface-container);
  color: var(--color-on-surface);
  cursor: pointer;
  border: 1px solid transparent;
  transition: background .12s, border-color .12s;
}
.role-chip input { display: none; }
.role-chip .material-symbols-rounded { font-size: 18px; color: var(--color-on-surface-variant); }
.role-chip.active {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  border-color: var(--color-primary);
}
.role-chip.active .material-symbols-rounded { color: var(--color-on-primary-container); }
.role-chip.locked { opacity: .5; cursor: not-allowed; }

.btn-text, .btn-filled {
  appearance: none;
  border: none;
  cursor: pointer;
  border-radius: var(--radius-full, 999px);
  padding: 8px 18px;
  font: inherit;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.btn-text { background: transparent; color: var(--color-on-surface-variant); }
.btn-text:hover { background: var(--color-surface-container); color: var(--color-on-surface); }
.btn-filled { background: var(--color-primary); color: var(--color-on-primary); }
.btn-filled:hover { filter: brightness(.94); }
.btn-filled:disabled, .btn-text:disabled { opacity: .55; cursor: not-allowed; }

.spin { animation: spin 1s linear infinite; font-size: 18px; }
@keyframes spin { to { transform: rotate(360deg); } }

:deep(.p-dialog-footer) { display: flex; justify-content: flex-end; gap: 8px; padding-top: 14px; }
</style>
