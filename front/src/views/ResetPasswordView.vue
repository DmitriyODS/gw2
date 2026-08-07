<template>
  <AuthShell
    title="новый пароль"
    subtitle="Придумайте новый пароль для входа в Groove Work."
    size="sm"
    back="/login"
  >
    <form v-if="token" class="auth-form" @submit.prevent="submit">
      <AuthField
        v-model="password"
        label="новый пароль"
        type="password"
        placeholder="не короче 8 символов"
        autocomplete="new-password"
        :disabled="loading"
      />
      <AuthField
        v-model="confirm"
        label="повторите пароль"
        type="password"
        placeholder="ещё раз"
        autocomplete="new-password"
        :disabled="loading"
      />
      <p v-if="error" class="auth-error">{{ error }}</p>
      <button type="submit" class="auth-submit" :disabled="loading">
        {{ loading ? 'сохраняем…' : 'сохранить пароль' }}
      </button>
    </form>

    <div v-else class="auth-form">
      <p class="auth-error">Ссылка недействительна — токен не найден. Запросите сброс пароля заново.</p>
      <RouterLink to="/forgot-password" class="auth-submit">запросить заново</RouterLink>
    </div>
  </AuthShell>
</template>

<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import AuthShell from '@/components/auth/AuthShell.vue'
import AuthField from '@/components/auth/AuthField.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const notif = useNotificationsStore()

const token = ref(route.query.token || '')
const password = ref('')
const confirm = ref('')
const error = ref('')
const loading = ref(false)

async function submit() {
  error.value = ''
  if (password.value.length < 8) {
    error.value = 'Пароль должен содержать минимум 8 символов'
    return
  }
  if (password.value !== confirm.value) {
    error.value = 'Пароли не совпадают'
    return
  }
  loading.value = true
  try {
    const { login } = await authStore.resetPassword(token.value, password.value)
    notif.success('Пароль обновлён — войдите с новым паролем')
    router.push({ path: '/login', query: login ? { login } : {} })
  } catch (e) {
    error.value = e?.message || 'Не удалось сменить пароль'
  } finally {
    loading.value = false
  }
}
</script>
