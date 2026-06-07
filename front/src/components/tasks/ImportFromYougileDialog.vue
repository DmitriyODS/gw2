<template>
  <AppDialog
    :model-value="visible"
    tone="primary"
    icon="download"
    size="md"
    mobile="full"
    title="Импорт из YouGile"
    subtitle="Вставьте ссылку на карточку — задача появится в нужном отделе."
    :busy="busy"
    :closable="!busy"
    :actions="[
      { kind: 'cancel', label: 'Отмена', disabled: busy },
      { kind: 'confirm', label: busy ? 'Импорт…' : 'Импортировать', disabled: busy || !canSubmit },
    ]"
    @update:model-value="(v) => !v && $emit('close')"
    @confirm="onImport"
  >
    <form class="task-form" @submit.prevent="onImport">
      <div class="form-field">
        <label class="form-label">Ссылка на карточку YouGile <span class="required">*</span></label>
        <InputText
          v-model="form.url"
          placeholder="https://yougile.com/team/.../#OIP1-2454"
          class="w-full"
          :invalid="!!errors.url"
        />
        <span v-if="errors.url" class="field-error">{{ errors.url }}</span>
        <span v-else class="form-hint">Скопируйте из адресной строки открытой карточки.</span>
      </div>

      <div class="form-field">
        <label class="form-label">Заказчик (отдел) <span class="required">*</span></label>
        <Select
          v-model="form.department_id"
          :options="departments"
          option-label="name"
          option-value="id"
          placeholder="Выберите отдел"
          class="w-full"
          :invalid="!!errors.department_id"
          :loading="depsLoading"
          filter
          filterPlaceholder="Поиск..."
        />
        <span v-if="errors.department_id" class="field-error">{{ errors.department_id }}</span>
      </div>

      <div class="form-field">
        <label class="form-label">Ответственный</label>
        <Select
          v-model="form.responsible_user_id"
          :options="responsibles"
          option-label="fio"
          option-value="id"
          placeholder="Не назначен"
          class="w-full"
          :loading="responsiblesLoading"
          filter
          filterPlaceholder="Поиск сотрудника..."
          show-clear
        />
      </div>

      <div class="form-field">
        <label class="checkbox-label">
          <input type="checkbox" v-model="form.pull_deadline" class="unit-checkbox" />
          <span>Подтянуть дедлайн из YouGile, если задан</span>
        </label>
      </div>

      <div v-if="serverError" class="server-error">{{ serverError }}</div>
    </form>
  </AppDialog>
</template>

<script setup>
import { reactive, ref, computed, onMounted, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useAuthStore } from '@/stores/auth.js'
import { importYougileTask } from '@/api/yougile.js'
import { getDepartments } from '@/api/departments.js'
import { getDirectory } from '@/api/users.js'

const props = defineProps({ visible: { type: Boolean, default: false } })
const emit = defineEmits(['close', 'imported'])

const notif = useNotificationsStore()
const auth = useAuthStore()

const form = reactive({
  url: '',
  department_id: null,
  responsible_user_id: auth.user?.id ?? null,
  pull_deadline: true,
})

const departments = ref([])
const depsLoading = ref(false)
const responsibles = ref([])
const responsiblesLoading = ref(false)
const busy = ref(false)
const serverError = ref('')

const errors = ref({ url: '', department_id: '' })

const canSubmit = computed(() => !!form.url.trim() && !!form.department_id)

function validate() {
  errors.value = { url: '', department_id: '' }
  let valid = true
  if (!form.url.trim()) {
    errors.value.url = 'Вставьте ссылку на карточку'
    valid = false
  }
  if (!form.department_id) {
    errors.value.department_id = 'Выберите отдел'
    valid = false
  }
  return valid
}

async function loadDepartments() {
  depsLoading.value = true
  try {
    const data = await getDepartments()
    departments.value = Array.isArray(data) ? data : (data.departments ?? data.items ?? [])
  } catch {
    departments.value = []
  } finally {
    depsLoading.value = false
  }
}

async function loadResponsibles() {
  responsiblesLoading.value = true
  try {
    const list = await getDirectory()
    responsibles.value = Array.isArray(list) ? list : (list.items ?? [])
  } catch {
    responsibles.value = []
  } finally {
    responsiblesLoading.value = false
  }
}

async function onImport() {
  if (!validate()) return
  busy.value = true
  serverError.value = ''
  try {
    const task = await importYougileTask({
      url: form.url.trim(),
      department_id: form.department_id,
      responsible_user_id: form.responsible_user_id || null,
      pull_deadline: form.pull_deadline,
    })
    notif.success('Карточка импортирована из YouGile')
    emit('imported', task)
  } catch (e) {
    serverError.value = e?.data?.message || e?.message || 'Не удалось импортировать'
  } finally {
    busy.value = false
  }
}

watch(() => props.visible, (v) => {
  if (v) {
    form.url = ''
    form.department_id = null
    form.responsible_user_id = auth.user?.id ?? null
    form.pull_deadline = true
    errors.value = { url: '', department_id: '' }
    serverError.value = ''
  }
})

onMounted(() => {
  loadDepartments()
  loadResponsibles()
})
</script>

<style scoped>
.task-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 8px 0;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--gw-text);
}

.required {
  color: var(--color-secondary);
}

.form-hint {
  font-size: 12px;
  color: var(--color-text-dim);
}

.field-error {
  font-size: 12px;
  color: var(--color-error);
}

.server-error {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
  border-radius: 8px;
  padding: 10px 14px;
  font-size: 13px;
  font-weight: 500;
}

.w-full {
  width: 100%;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--gw-text);
  cursor: pointer;
  user-select: none;
}

.unit-checkbox {
  width: 16px;
  height: 16px;
  accent-color: var(--gw-primary);
  cursor: pointer;
  flex-shrink: 0;
}

@media (max-width: 600px) {
  .task-form { gap: 18px; padding: 4px 0 8px; }
  .form-label {
    font-size: 12.5px;
    color: var(--color-text-dim);
    text-transform: uppercase;
    letter-spacing: 0.4px;
    font-weight: 700;
  }
  .task-form :deep(.p-inputtext),
  .task-form :deep(.p-select-label) {
    min-height: 48px;
    padding: 12px 14px;
    font-size: 15px;
    border-radius: var(--radius-md);
  }
  .task-form :deep(.p-select) {
    min-height: 48px;
    border-radius: var(--radius-md);
  }
  .checkbox-label { padding: 6px 0; font-size: 15px; min-height: 44px; }
  .unit-checkbox { width: 20px; height: 20px; }
}
</style>
