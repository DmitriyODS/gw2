<template>
  <AuthShell title="Доступ для Яндекс Алисы" size="sm">
    <p class="oa-text">
      Навык «Groove Work» получит доступ к вашим задачам, ежедневнику и заметкам
      от имени аккаунта <b>{{ authStore.user?.fio || 'вашего аккаунта' }}</b>
      <template v-if="authStore.companyName"> (компания «{{ authStore.companyName }}»)</template>.
    </p>

    <AppInfoBar
      v-if="!valid"
      tone="error"
      inline
      class="oa-error"
      message="Некорректная ссылка авторизации: не хватает параметров запроса."
    />
    <AppInfoBar v-else-if="error" tone="error" inline class="oa-error" :message="error" />

    <template #actions>
      <AppButton label="Отклонить" :disabled="loading" @click="deny" />
      <AppButton variant="filled" :loading="loading" :disabled="!valid" @click="allow">Разрешить</AppButton>
    </template>
  </AuthShell>
</template>

<script setup>
import { computed, ref } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { oauthAuthorize } from '@/api/auth.js'
import AuthShell from '@/components/auth/AuthShell.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const loading = ref(false)
const error = ref('')

const params = computed(() => ({
  client_id: String(route.query.client_id ?? ''),
  redirect_uri: String(route.query.redirect_uri ?? ''),
  state: String(route.query.state ?? ''),
  scope: String(route.query.scope ?? ''),
}))
const valid = computed(() => params.value.client_id !== '' && params.value.redirect_uri !== '')

async function allow() {
  if (loading.value) return
  loading.value = true
  error.value = ''
  try {
    const { redirect_url: url } = await oauthAuthorize(params.value)
    window.location.href = url
  } catch (e) {
    error.value = e?.message || 'Не удалось выдать доступ. Попробуйте ещё раз.'
    loading.value = false
  }
}

function deny() {
  // Штатный отказ OAuth: возвращаем пользователя к Яндексу с access_denied.
  if (valid.value) {
    const sep = params.value.redirect_uri.includes('?') ? '&' : '?'
    const q = new URLSearchParams({ error: 'access_denied', state: params.value.state })
    window.location.href = params.value.redirect_uri + sep + q.toString()
    return
  }
  router.push('/home')
}
</script>

<style scoped>
.oa-text {
  margin: 0;
  font-size: 14px;
  line-height: 1.55;
  color: var(--color-text-dim);
}

.oa-text b { color: var(--color-text); }

.oa-error { margin-top: 14px; }
</style>
