<template>
  <AuthShell
    title="Вход по QR-коду"
    subtitle="Отсканируйте код телефоном, где вы уже вошли, или введите код вручную."
    size="sm"
    back="/login"
  >
    <!-- Как только вход подтверждён с телефона, дальше либо сразу уходим в
         приложение (finish), либо нужен выбор компании — в обоих случаях
         код уже отработал и висеть на экране не должен: иначе выглядит так,
         будто вход всё ещё ждёт сканирования. -->
    <DeviceLinkInitiator v-if="!companies.length" kind="login" @session="onSession" />

    <!-- Выбор компании, если пользователь состоит в нескольких. -->
    <AppCard v-if="companies.length" variant="group" :gap="6">
      <AppRow
        v-for="c in companies"
        :key="c.company_id"
        :title="c.company_name"
        :hint="c.is_active ? c.role_name : `${c.role_name} · отключена`"
        icon="apartment"
        clickable
        :disabled="loading || !c.is_active"
        @click="pick(c.company_id)"
      />
    </AppCard>
    <AppInfoBar v-if="error" tone="error" inline :message="error" class="ql-error" />
  </AuthShell>
</template>

<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { connectSocket } from '@/socket/index.js'
import AuthShell from '@/components/auth/AuthShell.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
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
</style>
