<template>
  <AuthShell
    title="вход по QR-коду"
    subtitle="Отсканируйте код телефоном, где вы уже вошли, или введите код вручную."
    size="sm"
    back="/login"
  >
    <DeviceLinkInitiator kind="login" @session="onSession" />

    <!-- Выбор компании, если пользователь состоит в нескольких. -->
    <div v-if="companies.length" class="ql-companies">
      <button
        v-for="c in companies"
        :key="c.company_id"
        type="button"
        class="ql-company"
        :disabled="loading || !c.is_active"
        @click="pick(c.company_id)"
      >
        {{ c.company_name }}
      </button>
    </div>
    <p v-if="error" class="auth-error ql-error">{{ error }}</p>
  </AuthShell>
</template>

<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { connectSocket } from '@/socket/index.js'
import AuthShell from '@/components/auth/AuthShell.vue'
import DeviceLinkInitiator from '@/components/auth/DeviceLinkInitiator.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const companies = ref([])
const selectToken = ref('')
const loading = ref(false)
const error = ref('')

// Вход подтверждён с телефона: применяем сессию как обычный login.
function onSession(session) {
  const result = authStore.applyLinkSession(session)
  if (result.needsSelection) {
    companies.value = result.companies || []
    selectToken.value = result.selectToken
    return
  }
  finish()
}

async function pick(companyId) {
  loading.value = true
  error.value = ''
  try {
    await authStore.selectCompany(selectToken.value, companyId)
    finish()
  } catch (e) {
    error.value = e?.message || 'Не удалось войти в выбранную компанию'
  } finally {
    loading.value = false
  }
}

function finish() {
  connectSocket()
  const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/home'
  router.push(redirect)
}
</script>

<style scoped>
.ql-error { margin-top: 12px; }

.ql-companies {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 18px;
}

.ql-company {
  padding: 12px 16px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: color-mix(in oklch, var(--color-surface) 60%, transparent);
  color: var(--color-text);
  font: inherit;
  font-size: 15px;
  font-weight: 600;
  cursor: pointer;
  transition: border-color 0.15s;
}

.ql-company:hover:not(:disabled) { border-color: var(--color-primary); }
.ql-company:disabled { opacity: 0.5; cursor: not-allowed; }

</style>
