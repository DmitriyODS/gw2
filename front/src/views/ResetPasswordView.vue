<template>
  <AuthShell
    title="Новый пароль"
    subtitle="Придумайте новый пароль для входа в Groove Work."
    size="sm"
    back="/login"
  >
    <form v-if="token" class="auth-form" @submit.prevent="submit">
      <AuthField
        v-model="password"
        label="Новый пароль"
        type="password"
        placeholder="Не короче 8 символов"
        autocomplete="new-password"
        :disabled="loading"
      />
      <AuthField
        v-model="confirm"
        label="Повторите пароль"
        type="password"
        placeholder="Ещё раз"
        autocomplete="new-password"
        :disabled="loading"
      />
      <AppInfoBar v-if="error" tone="error" inline :message="error" />
      <AppButton type="submit" variant="filled" size="lg" block :loading="loading" label="Сохранить пароль" />
    </form>

    <div v-else class="auth-form">
      <AppInfoBar tone="error" inline message="Ссылка недействительна — токен не найден. Запросите сброс пароля заново." />
      <AppButton tag="router-link" to="/forgot-password" variant="filled" size="lg" block label="Запросить заново" />
    </div>
  </AuthShell>
</template>

<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import AuthShell from '@/components/auth/AuthShell.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
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
