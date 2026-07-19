<template>
  <div class="ya-callback">
    <div class="yc-card">
      <template v-if="state === 'loading'">
        <span class="yc-spinner"></span>
        <p class="yc-muted">Входим через Яндекс…</p>
      </template>

      <template v-else-if="state === 'select'">
        <span class="material-symbols-outlined yc-icon">apartment</span>
        <h2 class="yc-title">Выберите компанию</h2>
        <p v-if="error" class="yc-error">{{ error }}</p>
        <div class="yc-companies">
          <button
            v-for="c in pickerCompanies"
            :key="c.company_id"
            type="button"
            class="btn-glass yc-company"
            :disabled="loading || c.is_active === false"
            @click="pick(c.company_id)"
          >
            {{ c.company_name }}
          </button>
        </div>
      </template>

      <template v-else>
        <span class="material-symbols-outlined yc-icon">error</span>
        <h2 class="yc-title">Не получилось</h2>
        <p class="yc-text">{{ error }}</p>
        <button type="button" class="btn-grad" @click="router.push('/login')">Ко входу</button>
      </template>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { connectSocket } from '@/socket/index.js'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const state = ref('loading') // loading | select | error
const loading = ref(false)
const error = ref('')
const pickerCompanies = ref([])
const selectToken = ref('')

onMounted(async () => {
  const code = String(route.query.code ?? '')
  if (!code) {
    state.value = 'error'
    error.value = 'Яндекс не передал код авторизации.'
    return
  }
  try {
    const result = await authStore.yandexLogin(code)
    if (result.needsSelection) {
      pickerCompanies.value = result.companies
      selectToken.value = result.selectToken
      state.value = 'select'
      return
    }
    finish()
  } catch (e) {
    state.value = 'error'
    error.value = e?.message || 'Не удалось войти через Яндекс.'
  }
})

async function pick(companyId) {
  if (loading.value) return
  loading.value = true
  error.value = ''
  try {
    await authStore.selectCompany(selectToken.value, companyId)
    finish()
  } catch (e) {
    error.value = e?.message || 'Не удалось выбрать компанию.'
  } finally {
    loading.value = false
  }
}

function finish() {
  connectSocket()
  router.push('/')
}
</script>

<style scoped>
.ya-callback {
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}
.yc-card {
  width: 100%;
  max-width: 420px;
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
.yc-icon { font-size: 52px; color: var(--color-primary); }
.yc-title { font-size: 1.35rem; font-weight: 700; color: var(--color-text); }
.yc-text { color: var(--color-text-secondary); font-size: 0.92rem; }
.yc-muted { color: var(--color-text-secondary); }
.yc-error { color: var(--color-error); font-size: 0.85rem; }
.yc-companies { display: flex; flex-direction: column; gap: 10px; width: 100%; }
.yc-company { width: 100%; }
.yc-spinner {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 3px solid var(--color-primary);
  border-top-color: transparent;
  animation: yc-spin 0.8s linear infinite;
}
@keyframes yc-spin { to { transform: rotate(360deg); } }
</style>
