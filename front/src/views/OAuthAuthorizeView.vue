<template>
  <div class="oauth-authorize">
    <div class="oa-card">
      <span class="material-symbols-outlined oa-icon">graphic_eq</span>
      <h2 class="oa-title">Доступ для Яндекс Алисы</h2>
      <p class="oa-text">
        Навык «Groove Work» получит доступ к вашим задачам, ежедневнику и
        заметкам от имени аккаунта
        <b>{{ authStore.user?.fio || 'вашего аккаунта' }}</b>
        <template v-if="authStore.companyName">
          (компания «{{ authStore.companyName }}»)</template>.
      </p>
      <p v-if="!valid" class="oa-error">
        Некорректная ссылка авторизации: не хватает параметров запроса.
      </p>
      <p v-else-if="error" class="oa-error">{{ error }}</p>
      <div class="oa-actions">
        <button type="button" class="btn-glass" :disabled="loading" @click="deny">Отклонить</button>
        <button type="button" class="btn-grad" :disabled="loading || !valid" @click="allow">
          {{ loading ? 'Секунду…' : 'Разрешить' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { oauthAuthorize } from '@/api/auth.js'

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
  router.push('/')
}
</script>

<style scoped>
.oauth-authorize {
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}
.oa-card {
  width: 100%;
  max-width: 440px;
  padding: 32px 28px;
  border-radius: var(--radius-xl, 24px);
  background: var(--acrylic-card-bg);
  border: 1px solid var(--acrylic-border);
  box-shadow: var(--shadow-lg);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  text-align: center;
}
.oa-icon { font-size: 52px; color: var(--color-primary); }
.oa-title { font-size: 1.35rem; font-weight: 700; color: var(--color-text); }
.oa-text { color: var(--color-text-secondary); font-size: 0.92rem; max-width: 360px; }
.oa-error { color: var(--color-error); font-size: 0.85rem; }
.oa-actions { display: flex; gap: 12px; margin-top: 4px; }
</style>
