<template>
  <AuthShell :title="title" :subtitle="subtitle" size="sm">
    <div class="ia">
      <BrandLoader v-if="loading" :size="64" />

      <template v-else-if="invite">
        <p v-if="error" class="ia-error">{{ error }}</p>
        <AppButton variant="filled" class="ia-wide" :disabled="accepting" @click="accept">{{ accepting ? 'входим в команду…' : 'принять приглашение' }}</AppButton>
        <RouterLink to="/home" class="ia-later">позже</RouterLink>
      </template>

      <template v-else>
        <AppButton tag="router-link" to="/home" variant="filled" label="на главную" class="ia-wide" />
      </template>
    </div>
  </AuthShell>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
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
  if (loading.value) return 'проверяем приглашение'
  return invite.value ? 'приглашение в команду' : 'приглашение недоступно'
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

.ia-wide {
  width: 100%;
  justify-content: center;
  height: 44px;
  text-decoration: none;
}

.ia-later {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-dim);
  text-decoration: none;
}

.ia-later:hover { color: var(--color-primary); }

.ia-error {
  margin: 0;
  font-size: 13px;
  color: var(--color-error);
  text-align: center;
}
</style>
