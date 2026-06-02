<template>
  <Dialog
    :visible="true"
    @update:visible="$emit('close')"
    modal
    :header="task ? 'Редактировать задачу' : 'Создание новой задачи'"
    style="width: 500px; max-width: 95vw"
    :closable="true"
  >
    <form class="task-form" @submit.prevent="handleSubmit">
      <div class="form-field">
        <label class="form-label">Название задачи <span class="required">*</span></label>
        <InputText
          v-model="form.name"
          placeholder="Введите название задачи"
          class="w-full"
          :invalid="!!errors.name"
        />
        <span v-if="errors.name" class="field-error">{{ errors.name }}</span>
      </div>

      <div class="form-field">
        <label class="form-label">Ссылка на YouGile</label>
        <InputText
          v-model="form.link_yougile"
          placeholder="https://yougile.com/..."
          class="w-full"
        />
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
        <label class="form-label">Дата поступления <span class="required">*</span></label>
        <DatePicker
          v-model="form.received_at"
          dateFormat="dd.mm.yy"
          class="w-full"
          :invalid="!!errors.received_at"
        />
        <span v-if="errors.received_at" class="field-error">{{ errors.received_at }}</span>
      </div>

      <div class="form-field">
        <label class="form-label">Дедлайн</label>
        <DatePicker
          v-model="form.deadline"
          dateFormat="dd.mm.yy"
          class="w-full"
          showClear
        />
      </div>

      <template v-if="!task">
        <div class="form-field">
          <label class="checkbox-label">
            <input type="checkbox" v-model="createFirstUnit" class="unit-checkbox" />
            <span>Создать первый юнит</span>
          </label>
        </div>

        <template v-if="createFirstUnit">
          <div class="form-field">
            <label class="form-label">Название юнита</label>
            <InputText
              v-model="unitName"
              placeholder="Название юнита"
              class="w-full"
              @input="unitNameEdited = true"
            />
          </div>
          <div class="form-field">
            <label class="form-label">Тип юнита <span class="required">*</span></label>
            <Select
              v-model="unitTypeId"
              :options="unitTypes"
              option-label="name"
              option-value="id"
              placeholder="Выберите тип"
              class="w-full"
              :invalid="!!errors.unit_type_id"
              :loading="unitTypesLoading"
              filter
              filterPlaceholder="Поиск..."
            />
            <span v-if="errors.unit_type_id" class="field-error">{{ errors.unit_type_id }}</span>
          </div>
        </template>
      </template>

      <div v-if="serverError" class="server-error">{{ serverError }}</div>

      <div class="form-actions">
        <button type="button" class="btn-secondary" @click="$emit('close')" :disabled="submitting">
          Отмена
        </button>
        <button type="submit" class="btn-primary" :disabled="submitting">
          {{ submitting ? 'Сохранение...' : (task ? 'Сохранить' : 'Создать') }}
        </button>
      </div>
    </form>
  </Dialog>
</template>

<script setup>
import { ref, watch, onMounted } from 'vue'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import DatePicker from 'primevue/datepicker'
import { createTask, updateTask } from '@/api/tasks.js'
import { getDepartments } from '@/api/departments.js'
import { getUnitTypes } from '@/api/unitTypes.js'
import { createUnit } from '@/api/units.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useUnitsStore } from '@/stores/units.js'

const props = defineProps({
  task: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['close', 'saved'])

const notifications = useNotificationsStore()
const unitsStore = useUnitsStore()

const departments = ref([])
const depsLoading = ref(false)
const submitting = ref(false)
const serverError = ref('')

const form = ref({
  name: props.task?.name || '',
  link_yougile: props.task?.link_yougile || '',
  department_id: props.task?.department?.id || props.task?.department_id || null,
  received_at: props.task?.received_at ? new Date(props.task.received_at) : new Date(),
  deadline: props.task?.deadline ? new Date(props.task.deadline) : null
})

const errors = ref({
  name: '',
  department_id: '',
  received_at: ''
})

const createFirstUnit = ref(false)
const unitName = ref('')
const unitNameEdited = ref(false)
const unitTypeId = ref(null)
const unitTypes = ref([])
const unitTypesLoading = ref(false)

watch(() => form.value.name, (v) => {
  if (!unitNameEdited.value) unitName.value = v || ''
})

onMounted(async () => {
  depsLoading.value = true
  try {
    const data = await getDepartments()
    departments.value = Array.isArray(data) ? data : (data.departments ?? data.items ?? [])
  } catch {
    departments.value = []
  } finally {
    depsLoading.value = false
  }

  unitTypesLoading.value = true
  try {
    const data = await getUnitTypes()
    unitTypes.value = Array.isArray(data) ? data : (data.unit_types ?? data.items ?? [])
  } catch {
    unitTypes.value = []
  } finally {
    unitTypesLoading.value = false
  }
})

function validate() {
  errors.value = { name: '', department_id: '', received_at: '', unit_type_id: '' }
  let valid = true

  if (!form.value.name.trim()) {
    errors.value.name = 'Введите название задачи'
    valid = false
  }

  if (!form.value.department_id) {
    errors.value.department_id = 'Выберите отдел'
    valid = false
  }

  if (!form.value.received_at) {
    errors.value.received_at = 'Укажите дату поступления'
    valid = false
  }

  if (!props.task && createFirstUnit.value && !unitTypeId.value) {
    errors.value.unit_type_id = 'Выберите тип юнита'
    valid = false
  }

  return valid
}

function toDateStr(d) {
  if (!d) return null
  const date = d instanceof Date ? d : new Date(d)
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

async function handleSubmit() {
  if (!validate()) return

  submitting.value = true
  serverError.value = ''

  try {
    const payload = {
      name: form.value.name.trim(),
      department_id: form.value.department_id,
      received_at: toDateStr(form.value.received_at),
      link_yougile: form.value.link_yougile?.trim() || null,
      deadline: form.value.deadline ? toDateStr(form.value.deadline) : null
    }

    let result
    if (props.task) {
      result = await updateTask(props.task.id, payload)
      notifications.success('Задача успешно обновлена')
    } else {
      result = await createTask(payload)
      if (createFirstUnit.value && result?.id) {
        try {
          const unit = await createUnit(result.id, {
            name: unitName.value.trim() || result.name,
            unit_type_id: unitTypeId.value,
          })
          unitsStore.startUnit(unit)
          notifications.success('Задача создана, юнит запущен')
        } catch (e) {
          notifications.success('Задача создана')
          notifications.error('Не удалось запустить юнит: ' + (e?.message || 'ошибка'))
        }
      } else {
        notifications.success('Задача успешно создана')
      }
    }

    emit('saved', result)
  } catch (e) {
    serverError.value = e?.message || 'Не удалось сохранить задачу'
  } finally {
    submitting.value = false
  }
}
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

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding-top: 8px;
}

.btn-secondary {
  background: transparent;
  border: 1px solid var(--gw-border);
  border-radius: 8px;
  padding: 9px 20px;
  font-size: 14px;
  color: var(--gw-text);
  cursor: pointer;
  transition: background 0.12s;
}

.btn-secondary:hover:not(:disabled) {
  background: var(--gw-bg);
}

.btn-primary {
  background: var(--gw-primary);
  border: none;
  border-radius: 8px;
  padding: 9px 20px;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-on-primary);
  cursor: pointer;
  transition: opacity 0.12s;
}

.btn-primary:hover:not(:disabled) {
  opacity: 0.88;
}

.btn-primary:disabled,
.btn-secondary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
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
</style>
