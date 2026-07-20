<template>
  <AppDialog
    :model-value="modelValue"
    tone="primary"
    icon="alternate_email"
    size="sm"
    title="Изменить логин"
    subtitle="Под этим логином вы входите в систему."
    :busy="loading"
    :closable="!loading"
    :actions="[
      { kind: 'cancel', label: 'Отмена', disabled: loading },
      { kind: 'confirm', label: 'Сохранить', icon: 'check', disabled: loading },
    ]"
    @update:model-value="close"
    @confirm="submit"
  >
    <form class="dlg-form" @submit.prevent="submit">
      <div class="form-group">
        <label>Логин</label>
        <InputText
          v-model="login"
          class="w-full"
          placeholder="ivanov"
          autocomplete="username"
        />
      </div>
      <p v-if="error" class="error-msg">{{ error }}</p>
    </form>
  </AppDialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import InputText from 'primevue/inputtext'
import { updateMe } from '@/api/users.js'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  currentLogin: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue'])

const authStore = useAuthStore()
const notif = useNotificationsStore()

const login = ref('')
const error = ref('')
const loading = ref(false)

watch(
  () => props.modelValue,
  (open) => {
    if (open) {
      login.value = props.currentLogin || ''
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
  const value = login.value.trim()
  if (!value) {
    error.value = 'Логин обязателен'
    return
  }
  if (value === props.currentLogin) {
    emit('update:modelValue', false)
    return
  }
  loading.value = true
  try {
    await updateMe({ login: value })
    await authStore.loadMe()
    notif.success('Логин изменён')
    emit('update:modelValue', false)
  } catch (e) {
    error.value = e.message || 'Не удалось изменить логин'
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
