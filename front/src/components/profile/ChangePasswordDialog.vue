<template>
  <AppDialog
    :model-value="modelValue"
    tone="tertiary"
    icon="lock_reset"
    size="sm"
    title="Изменить пароль"
    subtitle="Новый пароль — не короче 8 символов."
    :busy="loading"
    :closable="!loading"
    :actions="[
      { kind: 'cancel', label: 'Отмена', disabled: loading },
      { kind: 'confirm', label: 'Изменить', icon: 'check', disabled: loading },
    ]"
    @update:model-value="close"
    @confirm="submit"
  >
    <form class="dlg-form" @submit.prevent="submit">
      <div class="form-group">
        <label>Текущий пароль</label>
        <InputText
          v-model="form.current"
          type="password"
          class="w-full"
          placeholder="Введите текущий пароль"
          autocomplete="current-password"
        />
      </div>
      <div class="form-group">
        <label>Новый пароль</label>
        <InputText
          v-model="form.password"
          type="password"
          class="w-full"
          placeholder="Минимум 8 символов"
          autocomplete="new-password"
        />
      </div>
      <div class="form-group">
        <label>Подтвердите пароль</label>
        <InputText
          v-model="form.confirm"
          type="password"
          class="w-full"
          placeholder="Повторите пароль"
          autocomplete="new-password"
        />
      </div>
      <p v-if="error" class="error-msg">{{ error }}</p>
    </form>
  </AppDialog>
</template>

<script setup>
import { reactive, ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import InputText from 'primevue/inputtext'
import { updateMe } from '@/api/users.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const notif = useNotificationsStore()

const form = reactive({ current: '', password: '', confirm: '' })
const error = ref('')
const loading = ref(false)

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      form.current = ''
      form.password = ''
      form.confirm = ''
      error.value = ''
    }
  },
)

function close() {
  if (loading.value) return
  emit('update:modelValue', false)
}

async function submit() {
  error.value = ''
  if (!form.current) {
    error.value = 'Введите текущий пароль'
    return
  }
  if (form.password.length < 8) {
    error.value = 'Пароль должен содержать не менее 8 символов'
    return
  }
  if (form.password !== form.confirm) {
    error.value = 'Пароли не совпадают'
    return
  }
  loading.value = true
  try {
    await updateMe({
      current_password: form.current,
      new_password: form.password,
      confirm_password: form.confirm,
    })
    notif.success('Пароль изменён')
    emit('update:modelValue', false)
  } catch (e) {
    error.value = e.message || 'Ошибка смены пароля'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.dlg-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-group label {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-dim);
}

.w-full {
  width: 100%;
}

.error-msg {
  margin: 0;
  font-size: 13px;
  color: var(--color-on-error-container);
  padding: 8px 12px;
  background: var(--color-error-container);
  border-radius: 8px;
  border: 1px solid color-mix(in oklch, var(--color-error) 30%, var(--color-outline-dim));
}
</style>
