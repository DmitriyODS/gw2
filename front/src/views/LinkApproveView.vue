<template>
  <AuthShell :title="title" :subtitle="subtitle" size="sm">
    <div class="la">
      <BrandLoader v-if="state === 'loading'" :size="64" />

      <template v-else-if="state === 'confirm'">
        <AppInfoBar v-if="error" tone="error" inline :message="error" class="la-wide" />
        <div class="la-actions">
          <AppButton label="Отмена" @click="goHome" />
          <AppButton
            variant="filled"
            :loading="loading"
            :disabled="isTv && authStore.companyId == null"
            @click="approve"
          >Подтвердить</AppButton>
        </div>
      </template>

      <template v-else-if="state === 'done'">
        <AppButton variant="filled" label="На главную" size="lg" block @click="goHome" />
      </template>

      <template v-else>
        <AppInfoBar tone="error" inline :message="error" class="la-wide" />
        <AppButton variant="filled" label="На главную" size="lg" block @click="goHome" />
      </template>
    </div>
  </AuthShell>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { linkInfo, linkApprove } from '@/api/devicelink.js'
import { normalizeLinkCode } from '@/utils/deviceLink.js'
import AuthShell from '@/components/auth/AuthShell.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const state = ref('loading') // loading | confirm | done | error
const info = ref(null)
const loading = ref(false)
const error = ref('')

const code = computed(() => normalizeLinkCode(route.query.code))
const isTv = computed(() => info.value?.kind === 'tv')

const title = computed(() => {
  if (state.value === 'loading') return 'Проверяем код'
  if (state.value === 'done') return 'Готово'
  if (state.value === 'error') return 'Не получилось'
  return isTv.value ? 'Активировать ТВ-киоск?' : 'Подтвердить вход?'
})

const subtitle = computed(() => {
  if (state.value === 'loading') return ''
  if (state.value === 'done') return 'Устройство входит в систему. Эту страницу можно закрыть.'
  if (state.value === 'error') return ''
  if (isTv.value) {
    return authStore.companyId != null
      ? `ТВ-киоск войдёт в систему под компанией «${authStore.companyName}».`
      : 'Чтобы авторизовать ТВ-киоск, сначала выберите компанию в своём аккаунте.'
  }
  return 'Другое устройство войдёт под вашим аккаунтом. Подтверждайте, только если это ваш вход.'
})

onMounted(async () => {
  if (!/^[A-Z2-9]{6}$/.test(code.value)) {
    state.value = 'error'
    error.value = 'Неверная ссылка входа.'
    return
  }
  try {
    info.value = await linkInfo(code.value)
    state.value = 'confirm'
  } catch (e) {
    state.value = 'error'
    error.value = errText(e)
  }
})

async function approve() {
  if (loading.value) return
  loading.value = true
  error.value = ''
  try {
    await linkApprove(code.value)
    state.value = 'done'
  } catch (e) {
    error.value = errText(e)
  } finally {
    loading.value = false
  }
}

function errText(e) {
  switch (e?.error) {
    case 'LINK_EXPIRED':
      return 'Код устарел. Обновите его на устройстве и попробуйте снова.'
    case 'LINK_ALREADY_USED':
      return 'Этот код уже подтверждён другим аккаунтом.'
    case 'LINK_NEED_COMPANY':
      return 'Сначала выберите компанию, под которой авторизовать ТВ-киоск.'
    default:
      return e?.message || 'Не удалось подтвердить устройство.'
  }
}

function goHome() {
  router.push('/home')
}
</script>

<style scoped>
.la {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.la-actions {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  justify-content: center;
}

.la-wide { width: 100%; }

</style>
