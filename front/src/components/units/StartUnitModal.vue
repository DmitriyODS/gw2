<template>
  <AppDialog
    model-value
    tone="primary"
    icon="play_circle"
    size="sm"
    title="Начать юнит"
    subtitle="Зафиксируйте время работы над задачей."
    :busy="submitting"
    :closable="!submitting"
    :actions="[
      { kind: 'cancel', label: 'Отмена', disabled: submitting },
      { kind: 'confirm', label: 'Начать', icon: 'play_arrow', disabled: submitting },
    ]"
    @update:model-value="(v) => !v && $emit('close')"
    @confirm="handleSubmit"
  >
    <form class="unit-form" @submit.prevent="handleSubmit">
      <div class="form-field">
        <label class="form-label">Название юнита <span class="required">*</span></label>
        <InputText
          v-model="form.name"
          placeholder="Введите название юнита"
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

      <div v-if="serverError" class="server-error">{{ serverError }}</div>
    </form>
  </AppDialog>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import AppDialog from '@/components/common/AppDialog.vue'
import { createUnit } from '@/api/units.js'
import { getUnitTypes } from '@/api/unitTypes.js'
import { useUnitsStore } from '@/stores/units.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  taskId: { type: Number, required: true },
})

const emit = defineEmits(['close', 'started'])

const unitsStore = useUnitsStore()
const notifications = useNotificationsStore()

const unitTypes = ref([])
const submitting = ref(false)
const serverError = ref('')

const form = ref({ name: '', unit_type_id: null })
const errors = ref({ name: '', unit_type_id: '' })

onMounted(async () => {
  try {
    const data = await getUnitTypes()
    unitTypes.value = Array.isArray(data) ? data : (data.unit_types ?? data.items ?? [])
  } catch {
    unitTypes.value = []
  }
})

function validate() {
  errors.value = { name: '', unit_type_id: '' }
  let valid = true
  if (!form.value.name.trim()) {
    errors.value.name = 'Введите название юнита'
    valid = false
  }
  if (!form.value.unit_type_id) {
    errors.value.unit_type_id = 'Выберите тип юнита'
    valid = false
  }
  return valid
}

async function handleSubmit() {
  if (!validate()) return
  submitting.value = true
  serverError.value = ''
  try {
    const unit = await createUnit(props.taskId, {
      name: form.value.name.trim(),
      unit_type_id: form.value.unit_type_id,
    })
    unitsStore.startUnit(unit)
    notifications.success('Юнит успешно запущен')
    emit('started', unit)
  } catch (e) {
    if (e?.status === 409) {
      serverError.value = 'У вас уже есть активный юнит'
    } else {
      serverError.value = e?.message || 'Не удалось запустить юнит'
    }
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
