<template>
  <AuthShell
    :title="sent ? 'проверьте почту' : 'сброс пароля'"
    :subtitle="sent ? '' : 'Укажите email — пришлём ссылку для установки нового пароля.'"
    size="sm"
    back="/login"
  >
    <template v-if="sent">
      <p class="fp-done">
        Если аккаунт с адресом <b>{{ email }}</b> существует, мы отправили на него
        письмо со ссылкой для сброса пароля.
      </p>
      <RouterLink to="/login" class="auth-submit">вернуться ко входу</RouterLink>
    </template>

    <form v-else class="auth-form" @submit.prevent="submit">
      <AuthField
        v-model="email"
        label="email"
        type="email"
        placeholder="name@example.com"
        autocomplete="email"
        :disabled="loading"
      />
      <p v-if="error" class="auth-error">{{ error }}</p>
      <button type="submit" class="auth-submit" :disabled="loading">
        {{ loading ? 'отправляем…' : 'отправить ссылку' }}
      </button>
    </form>
  </AuthShell>
</template>

<script setup>
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth.js'
import AuthShell from '@/components/auth/AuthShell.vue'
import AuthField from '@/components/auth/AuthField.vue'

const authStore = useAuthStore()

const email = ref('')
const error = ref('')
const loading = ref(false)
const sent = ref(false)

async function submit() {
  error.value = ''
  const value = email.value.trim()
  if (!value || !/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(value)) {
    error.value = 'Укажите корректный email'
    return
  }
  loading.value = true
  try {
    await authStore.forgotPassword(value)
    email.value = value
    sent.value = true
  } catch (e) {
    error.value = e?.message || 'Не удалось отправить письмо'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.fp-done {
  margin: 0 0 18px;
  font-size: 14px;
  line-height: 1.55;
  color: var(--color-text-dim);
}

.fp-done b { color: var(--color-text); }

</style>
