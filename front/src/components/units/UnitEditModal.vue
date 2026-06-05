<template>
  <AppDialog
    model-value
    tone="primary"
    icon="edit"
    size="sm"
    title="Редактировать юнит"
    :busy="submitting"
    :closable="!submitting"
    :actions="[
      { kind: 'cancel', label: 'Отмена', disabled: submitting },
      { kind: 'confirm', label: 'Сохранить', disabled: submitting },
    ]"
    @update:model-value="(v) => !v && $emit('close')"
    @confirm="handleSubmit"
  >
    <form class="unit-form" @submit.prevent="handleSubmit">
      <div class="form-field">
        <label class="form-label">Название юнита <span class="required">*</span></label>
        <InputText
          v-model="form.name"
          placeholder="Введите название"
          class="w-full"
          :invalid="!!errors.name"
        />
        <span v-if="errors.name" class="field-error">{{ errors.name }}</span>
      </div>

      <div class="form-field">
        <label class="form-label">Тип юнита <span class="required">*</span></label>
        <Select
          v-model="form.unit_type_id"
          :options="unitTypes"
          option-label="name"
          option-value="id"
          placeholder="Выберите тип"
          class="w-full"
          :invalid="!!errors.unit_type_id"
          filter
          filterPlaceholder="Поиск..."
        />
        <span v-if="errors.unit_type_id" class="field-error">{{ errors.unit_type_id }}</span>
      </div>

      <div class="form-field">
        <label class="form-label">Дата/время начала <span class="required">*</span></label>
        <DatePicker
          v-model="form.datetime_start"
          showTime
          hourFormat="24"
          dateFormat="dd.mm.yy"
          class="w-full"
          :invalid="!!errors.datetime_start"
        />
        <span v-if="errors.datetime_start" class="field-error">{{ errors.datetime_start }}</span>
      </div>

      <div v-if="unit.datetime_end" class="form-field">
        <label class="form-label">Дата/время окончания</label>
        <DatePicker
          v-model="form.datetime_end"
          showTime
          hourFormat="24"
          dateFormat="dd.mm.yy"
          class="w-full"
          :invalid="!!errors.datetime_end"
        />
        <span v-if="errors.datetime_end" class="field-error">{{ errors.datetime_end }}</span>
      </div>

      <div v-if="serverError" class="server-error">{{ serverError }}</div>
    </form>
  </AppDialog>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import DatePicker from 'primevue/datepicker'
import AppDialog from '@/components/common/AppDialog.vue'
import { updateUnit } from '@/api/units.js'
import { getUnitTypes } from '@/api/unitTypes.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  unit: { type: Object, required: true },
})

const emit = defineEmits(['close', 'saved'])

const notifications = useNotificationsStore()

const unitTypes = ref([])
const submitting = ref(false)
const serverError = ref('')

const form = ref({
  name: props.unit.name || '',
  unit_type_id: props.unit.unit_type_id || null,
  datetime_start: props.unit.datetime_start ? new Date(props.unit.datetime_start) : null,
  datetime_end: props.unit.datetime_end ? new Date(props.unit.datetime_end) : null,
})

const errors = ref({ name: '', unit_type_id: '', datetime_start: '', datetime_end: '' })

onMounted(async () => {
  try {
    const data = await getUnitTypes()
    unitTypes.value = Array.isArray(data) ? data : (data.unit_types ?? data.items ?? [])
  } catch {
    unitTypes.value = []
  }
})

function validate() {
  errors.value = { name: '', unit_type_id: '', datetime_start: '', datetime_end: '' }
  let valid = true
  if (!form.value.name.trim()) {
    errors.value.name = 'Введите название юнита'
    valid = false
  }
  if (!form.value.unit_type_id) {
    errors.value.unit_type_id = 'Выберите тип юнита'
    valid = false
  }
  if (!form.value.datetime_start) {
    errors.value.datetime_start = 'Укажите дату начала'
    valid = false
  }
  if (props.unit.datetime_end && form.value.datetime_end && form.value.datetime_start) {
    if (new Date(form.value.datetime_end) <= new Date(form.value.datetime_start)) {
      errors.value.datetime_end = 'Дата окончания должна быть позже даты начала'
      valid = false
    }
  }
  return valid
}

function toISOLocal(d) {
  if (!d) return null
  return d instanceof Date ? d.toISOString() : new Date(d).toISOString()
}

async function handleSubmit() {
  if (!validate()) return
  submitting.value = true
  serverError.value = ''
  try {
    const payload = {
      name: form.value.name.trim(),
      unit_type_id: form.value.unit_type_id,
      datetime_start: toISOLocal(form.value.datetime_start),
    }
    if (props.unit.datetime_end) {
      payload.datetime_end = toISOLocal(form.value.datetime_end)
    }
    await updateUnit(props.unit.id, payload)
    notifications.success('Юнит успешно обновлён')
    emit('saved')
    emit('close')
  } catch (e) {
    serverError.value = e?.message || 'Не удалось обновить юнит'
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.unit-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text);
}

.required { color: var(--color-error); }

.field-error {
  font-size: 12px;
  color: var(--color-error);
}

.server-error {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
  border-radius: var(--radius-md);
  padding: 10px 14px;
  font-size: 13px;
  font-weight: 500;
}

.w-full { width: 100%; }
</style>
