<template>
  <AuthShell :title="title" :subtitle="subtitle" size="sm">
    <div class="ia">
      <BrandLoader v-if="loading" :size="64" />

      <template v-else-if="invite">
        <AppInfoBar v-if="error" tone="error" inline :message="error" class="ia-wide" />
        <AppButton variant="filled" size="lg" block :loading="accepting" @click="accept">Принять приглашение</AppButton>
        <AppButton tag="router-link" to="/home" variant="text" label="Позже" />
      </template>

      <template v-else>
        <AppButton tag="router-link" to="/home" variant="filled" label="На главную" size="lg" block />
      </template>
    </div>
  </AuthShell>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { getInvitePreview } from '@/api/companies.js'
import AuthShell from '@/components/auth/AuthShell.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'

const props = defineProps({ token: { type: String, required: true } })

const router = useRouter()
const authStore = useAuthStore()
const notif = useNotificationsStore()

const invite = ref(null)
const loading = ref(true)
const accepting = ref(false)
const error = ref('')

const title = computed(() => {
  if (loading.value) return 'Проверяем приглашение'
  return invite.value ? 'Приглашение в команду' : 'Приглашение недоступно'
})

const subtitle = computed(() => {
  if (loading.value) return ''
  if (!invite.value) return error.value || 'Ссылка недействительна или срок её действия истёк.'
  return `Компания «${invite.value.company_name}» приглашает вас присоединиться на роль «${invite.value.role_name}».`
})

onMounted(async () => {
  try {
    invite.value = await getInvitePreview(props.token)
  } catch (e) {
    error.value = e?.message || ''
    invite.value = null
  } finally {
    loading.value = false
  }
})

async function accept() {
  accepting.value = true
  error.value = ''
  try {
    await authStore.acceptInvite(props.token)
    notif.success(`Вы в команде «${invite.value.company_name}»`)
    router.push('/tasks')
  } catch (e) {
    error.value = e?.message || 'Не удалось принять приглашение'
  } finally {
    accepting.value = false
  }
}
</script>

<style scoped>
.ia {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.ia-wide { width: 100%; }



</style>
