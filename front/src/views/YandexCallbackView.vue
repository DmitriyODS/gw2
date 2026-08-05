<template>
  <AuthShell :title="title" :subtitle="subtitle" size="sm">
    <div class="yc">
      <template v-if="state === 'loading'">
        <BrandLoader :size="64" />
      </template>

      <template v-else-if="state === 'linked'">
        <AppButton
          variant="filled"
          label="в аккаунт"
          class="yc-wide"
          @click="router.push('/settings?section=account')"
        />
      </template>

      <template v-else-if="state === 'return-app'">
        <AppButton variant="filled" label="открыть приложение" class="yc-wide" @click="openInApp" />
        <AppButton label="продолжить в браузере" class="yc-wide" @click="continueInBrowser" />
      </template>

      <template v-else-if="state === 'select'">
        <p v-if="error" class="yc-error">{{ error }}</p>
        <AppButton
          class="yc-wide"
          v-for="c in pickerCompanies"
          :key="c.company_id"
          :disabled="loading || c.is_active === false"
          @click="pick(c.company_id)"
        >{{ c.company_name }}</AppButton>
      </template>

      <template v-else>
        <p class="yc-error">{{ error }}</p>
        <AppButton variant="filled" label="ко входу" class="yc-wide" @click="router.push('/login')" />
      </template>
    </div>
  </AuthShell>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { yandexLink } from '@/api/auth.js'
import { connectSocket } from '@/socket/index.js'
import { inAppShell, APP_SCHEME } from '@/utils/appShell.js'
import AuthShell from '@/components/auth/AuthShell.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const state = ref('loading') // loading | linked | return-app | select | error
const loading = ref(false)
const error = ref('')
const pickerCompanies = ref([])
const selectToken = ref('')

const TITLES = {
  loading: 'входим через Яндекс',
  linked: 'аккаунт привязан',
  'return-app': 'возврат в приложение',
  select: 'выбор компании',
  error: 'не получилось',
}

const SUBTITLES = {
  loading: 'Секунду, проверяем данные.',
  linked: 'Теперь можно входить кнопкой «Войти через Яндекс».',
  'return-app': 'Вход подтверждён. Продолжите в приложении Groove Work.',
  select: 'Вы состоите в нескольких компаниях — в какую войти?',
  error: '',
}

const title = computed(() => TITLES[state.value] || TITLES.error)
const subtitle = computed(() => SUBTITLES[state.value] ?? '')

// state OAuth-редиректа: '' — вход в браузере, 'link' — привязка из профиля,
// 'app'/'app-link' — то же, но флоу начат из обёртки (десктоп/Android): эта
// страница открыта в СИСТЕМНОМ браузере и должна вернуть код в приложение по
// deep link — код одноразовый, обменять его можно только в одном месте.
const stateParam = String(route.query.state ?? '')
const fromApp = stateParam === 'app' || stateParam === 'app-link'
const isLink = stateParam === 'link' || stateParam === 'app-link'
const oauthCode = ref('')

function appDeepLink() {
  return `${APP_SCHEME}://yandex-callback?code=${encodeURIComponent(oauthCode.value)}&state=${encodeURIComponent(stateParam)}`
}

function openInApp() {
  window.location.href = appDeepLink()
}

// Ручной фолбэк, если приложение не открылось (deep link не сработал).
function continueInBrowser() {
  state.value = 'loading'
  isLink ? runLink() : runLogin()
}

async function runLink() {
  try {
    await authStore.ensureReady()
    if (!authStore.token) throw new Error('Сначала войдите в свой аккаунт Groove Work.')
    await yandexLink(oauthCode.value)
    state.value = 'linked'
  } catch (e) {
    state.value = 'error'
    error.value = e?.message || 'Не удалось привязать Яндекс-аккаунт.'
  }
}

async function runLogin() {
  try {
    const result = await authStore.yandexLogin(oauthCode.value)
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
}

onMounted(() => {
  oauthCode.value = String(route.query.code ?? '')
  if (!oauthCode.value) {
    state.value = 'error'
    error.value = 'Яндекс не передал код авторизации.'
    return
  }
  if (fromApp && !inAppShell()) {
    state.value = 'return-app'
    openInApp()
    return
  }
  isLink ? runLink() : runLogin()
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
  router.push('/home')
}
</script>

<style scoped>
.yc {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}

.yc-wide {
  width: 100%;
  justify-content: center;
  height: 44px;
}

.yc-error {
  margin: 0;
  font-size: 13px;
  color: var(--color-error);
  text-align: center;
}
</style>
