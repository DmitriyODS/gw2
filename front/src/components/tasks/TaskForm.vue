<template>
  <AppDialog
    model-value
    tone="primary"
    :icon="task ? 'edit' : 'add_task'"
    size="md"
    mobile="full"
    :title="task ? 'Редактировать задачу' : 'Новая задача'"
    :subtitle="task ? '' : 'Заполните основные поля. Юнит можно начать прямо отсюда.'"
    :busy="submitting"
    :closable="!submitting"
    :actions="[
      { kind: 'cancel', label: 'Отмена', disabled: submitting },
      { kind: 'confirm', label: task ? 'Сохранить' : 'Создать', disabled: submitting },
    ]"
    @update:model-value="(v) => !v && $emit('close')"
    @confirm="handleSubmit"
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

      <div v-if="usesYougile" class="form-field">
        <label class="form-label">Ссылка на YouGile</label>
        <InputText
          v-model="form.link_yougile"
          placeholder="https://yougile.com/..."
          class="w-full"
        />
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

      <div v-if="usesStages" class="form-field">
        <label class="form-label">Этап</label>
        <Select
          v-model="form.stage_id"
          :options="stages"
          option-label="name"
          option-value="id"
          placeholder="Без этапа"
          class="w-full"
          show-clear
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
        <div v-if="yougileAvailable" class="form-field">
          <label class="checkbox-label">
            <input type="checkbox" v-model="alsoExportToYg" class="unit-checkbox" />
            <span>Создать также карточку в YouGile</span>
          </label>
        </div>
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
    </form>
  </AppDialog>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import DatePicker from 'primevue/datepicker'
import { createTask, updateTask } from '@/api/tasks.js'
import { getDepartments } from '@/api/departments.js'
import { getUnitTypes } from '@/api/unitTypes.js'
import { createUnit } from '@/api/units.js'
import { getStages } from '@/api/stages.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useUnitsStore } from '@/stores/units.js'
import { useAuthStore } from '@/stores/auth.js'
import { useCompanySettings } from '@/composables/useCompanySettings.js'
import { getDirectory, getDirectoryUser } from '@/api/users.js'
import { exportYougileTask } from '@/api/yougile.js'
import { useYougileStore } from '@/stores/yougile.js'

const props = defineProps({
  task: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['close', 'saved'])

const notifications = useNotificationsStore()
const unitsStore = useUnitsStore()
const auth = useAuthStore()
const { usesYougile, usesStages } = useCompanySettings()

const yougileStore = useYougileStore()
const yougileAvailable = computed(() => yougileStore.isAvailable)
// Тумблер «также создать в YouGile». Виден только при создании и только если
// YG включён и юзер подключён.
const alsoExportToYg = ref(false)

const departments = ref([])
const depsLoading = ref(false)
const submitting = ref(false)
const serverError = ref('')

const stages = ref([])

const responsibles = ref([])
const responsiblesLoading = ref(false)

const form = ref({
  name: props.task?.name || '',
  link_yougile: props.task?.link_yougile || '',
  department_id: props.task?.department?.id || props.task?.department_id || null,
  received_at: props.task?.received_at ? new Date(props.task.received_at) : new Date(),
  deadline: props.task?.deadline ? new Date(props.task.deadline) : null,
  responsible_user_id: props.task
    ? (props.task.responsible_user_id ?? props.task.responsible?.id ?? null)
    : (auth.user?.id ?? null),
  stage_id: props.task?.stage_id ?? props.task?.stage?.id ?? null,
})

// При создании задачи автоматически назначаем автора ответственным.
// Подстраховка: если auth.user успел догрузиться после монтирования формы
// (refresh-flow) — подставим, как только появится id. Если пользователь
// уже сам поменял/снял ответственного — не перетираем.
const responsibleTouched = ref(false)
if (!props.task) {
  watch(
    () => auth.user?.id,
    (uid) => {
      if (!responsibleTouched.value && uid != null && form.value.responsible_user_id == null) {
        form.value.responsible_user_id = uid
      }
    },
    { immediate: true },
  )
  watch(
    () => form.value.responsible_user_id,
    (_v, prev) => {
      // первая авто-подстановка из watch выше не считается изменением пользователя
      if (prev !== undefined) responsibleTouched.value = true
    },
  )
}

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

  if (usesStages.value) {
    try {
      const data = await getStages()
      stages.value = Array.isArray(data) ? data : (data.items ?? [])
    } catch {
      stages.value = []
    }
  }

  responsiblesLoading.value = true
  try {
    const data = await getDirectory()
    const list = Array.isArray(data) ? data : (data?.items || [])
    responsibles.value = list
    /* Если выбранного ответственного нет в директории (например, он —
       Администратор системы вне scope'нутой компанией директории),
       докидываем его одиночным запросом, чтобы Select показал имя. */
    const rid = form.value.responsible_user_id
    if (rid != null && !list.some((u) => u.id === rid)) {
      try {
        const extra = await getDirectoryUser(rid)
        if (extra) responsibles.value = [extra, ...list]
      } catch { /* ignore */ }
    }
  } catch {
    responsibles.value = []
  } finally {
    responsiblesLoading.value = false
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
      link_yougile: usesYougile.value ? (form.value.link_yougile?.trim() || null) : null,
      deadline: form.value.deadline ? toDateStr(form.value.deadline) : null,
      responsible_user_id: form.value.responsible_user_id ?? null,
      stage_id: usesStages.value ? (form.value.stage_id ?? null) : null,
    }

    let result
    if (props.task) {
      result = await updateTask(props.task.id, payload)
      notifications.success('Задача успешно обновлена')
    } else {
      result = await createTask(payload)
      // Если включён тумблер YG — заводим карточку в YouGile и подменяем
      // result обновлённой задачей (там уже yougile_task_id+link_yougile).
      if (alsoExportToYg.value && yougileAvailable.value && result?.id) {
        try {
          result = await exportYougileTask({ gw_task_id: result.id })
        } catch (e) {
          notifications.error('Задача создана, но в YouGile не получилось: '
            + (e?.data?.message || e?.message || 'ошибка'))
        }
      }
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

/* ── Мобильный full-screen: крупнее поля, более тач-френдли. ── */
@media (max-width: 600px) {
  .task-form {
    gap: 18px;
    padding: 4px 0 8px;
  }

  .form-label {
    font-size: 12.5px;
    color: var(--color-text-dim);
    text-transform: uppercase;
    letter-spacing: 0.4px;
    font-weight: 700;
  }

  /* PrimeVue input'ы — крупнее по высоте для тача. */
  .task-form :deep(.p-inputtext),
  .task-form :deep(.p-select-label),
  .task-form :deep(.p-datepicker-input) {
    min-height: 48px;
    padding: 12px 14px;
    font-size: 15px;
    border-radius: var(--radius-md);
  }

  .task-form :deep(.p-select) {
    min-height: 48px;
    border-radius: var(--radius-md);
  }

  .checkbox-label {
    padding: 6px 0;
    font-size: 15px;
    min-height: 44px;
  }

  .unit-checkbox {
    width: 20px;
    height: 20px;
  }
}
</style>
