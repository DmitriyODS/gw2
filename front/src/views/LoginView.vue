<template>
  <!-- Экран входа ведёт три шага в ОДНОЙ карточке: учётные данные, выбор
       компании и обязательная смена пароля. -->
  <AuthShell
    :title="stepTitle"
    :subtitle="stepSubtitle"
    size="md"
    :back="step === 'credentials' ? '/welcome' : ''"
  >
    <!-- ── Шаг 1: логин и пароль ─────────────────────────────── -->
    <form v-if="step === 'credentials'" class="auth-form" @submit.prevent="handleLogin">
      <AuthField
        v-model="loginForm.login"
        label="Логин"
        placeholder="Ваш логин"
        autocomplete="username"
        :disabled="isLoginDisabled"
      />
      <AuthField
        v-model="loginForm.password"
        label="Пароль"
        type="password"
        placeholder="Пароль"
        autocomplete="current-password"
        :disabled="isLoginDisabled"
      />

      <div class="lg-forgot">
        <AppButton tag="router-link" to="/forgot-password" variant="text" size="sm" label="Забыли пароль?" />
      </div>

      <AppInfoBar
        v-if="cooldownSec > 0"
        tone="warning"
        icon="lock_clock"
        inline
        :message="`Слишком много неудачных попыток — попробуйте через ${formattedCooldown}`"
      />
      <AppInfoBar v-else-if="loginError" tone="error" inline :message="loginError" />

      <AppButton
        type="submit"
        variant="filled"
        size="lg"
        block
        :loading="loading"
        :disabled="cooldownSec > 0"
        :label="loginButtonLabel"
      />

      <div class="auth-divider"><span>или</span></div>

      <div class="lg-alts">
        <AppButton class="lg-yandex" block @click="goYandex">
          <YandexLogo :size="16" />
          Яндекс ID
        </AppButton>
        <AppButton tag="router-link" to="/qr-login" icon="qr_code_2" label="QR-код" block />
        <AppButton tag="router-link" to="/tv-activate" icon="tv" label="ТВ-режим" block />
      </div>

      <p class="auth-switch">
        Нет аккаунта?
        <AppButton tag="router-link" to="/register" variant="text" size="sm" label="Создать" />
      </p>
    </form>

    <!-- ── Шаг 2: выбор компании ─────────────────────────────── -->
    <div v-else-if="step === 'company'" class="auth-form">
      <AppCard variant="group" :gap="6">
        <AppRow
          v-for="c in pickerCompanies"
          :key="c.company_id"
          :title="c.company_name"
          :hint="c.is_active ? c.role_name : `${c.role_name} · отключена`"
          icon="apartment"
          clickable
          :selected="pickerSelected === c.company_id"
          :disabled="!c.is_active"
          :chevron-icon="pickerSelected === c.company_id ? 'check_circle' : 'radio_button_unchecked'"
          @click="pickerSelected = c.company_id"
        />
      </AppCard>
      <AppInfoBar v-if="loginError" tone="error" inline :message="loginError" />
      <AppButton
        variant="filled"
        size="lg"
        block
        :loading="loading"
        :disabled="!pickerSelected"
        label="Войти"
        @click="confirmCompany"
      />
    </div>

    <!-- ── Шаг 3: обязательная смена пароля ──────────────────── -->
    <form v-else class="auth-form" @submit.prevent="handleChangeDefault">
      <AuthField
        v-model="changeForm.password"
        label="Новый пароль"
        type="password"
        placeholder="Не короче 8 символов"
        autocomplete="new-password"
        :disabled="changeLoading"
      />
      <AuthField
        v-model="changeForm.confirmPassword"
        label="Повторите пароль"
        type="password"
        placeholder="Ещё раз"
        autocomplete="new-password"
        :disabled="changeLoading"
      />
      <AppInfoBar v-if="changeError" tone="error" inline :message="changeError" />
      <AppButton
        type="submit"
        variant="filled"
        size="lg"
        block
        :loading="changeLoading"
        label="Сохранить и войти"
      />
    </form>
  </AuthShell>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { connectSocket } from '@/socket/index.js'
import AuthShell from '@/components/auth/AuthShell.vue'
import AuthField from '@/components/auth/AuthField.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import YandexLogo from '@/components/common/YandexLogo.vue'
import { yandexConfig, yandexAuthURL } from '@/api/auth.js'
import { inAppShell } from '@/utils/appShell.js'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

// credentials → company → change-password: шаги одной карточки.
const step = ref('credentials')

const loginForm = reactive({ login: '', password: '' })
const loginError = ref('')
const loading = ref(false)

// Брутфорс-блокировка: бэк отвечает 429 + retry_after_sec, локально
// тикаем секунды и блокируем форму до конца таймера.
const cooldownSec = ref(0)
let cooldownTimer = null

// Выбор компании при логине (если их несколько).
const pickerCompanies = ref([])
const pickerSelectToken = ref('')
const pickerSelected = ref(null)

// Вход через Яндекс ID. Из обёртки state=app: авторизация идёт в системном браузере (там уже есть
// сессия Яндекса), а /yandex-callback вернёт флоу в приложение по deep link.
const yandexAuth = ref({ enabled: false, client_id: '' })
function goYandex() {
  // Кнопка на экране есть всегда; о ненастроенном входе честно сообщаем по
  // клику, а не прячем способ входа втихую.
  if (!yandexAuth.value.enabled) {
    loginError.value = 'Вход через Яндекс на этом сервере не настроен'
    return
  }
  window.location.href = yandexAuthURL(yandexAuth.value.client_id, inAppShell() ? 'app' : '')
}

const changeForm = reactive({ password: '', confirmPassword: '' })
const changeError = ref('')
const changeLoading = ref(false)

const isLoginDisabled = computed(() => loading.value || cooldownSec.value > 0)

const stepTitle = computed(() => {
  if (step.value === 'company') return 'Выбор компании'
  if (step.value === 'change-password') return 'Смена пароля'
  return 'Вход в аккаунт'
})

const stepSubtitle = computed(() => {
  if (step.value === 'company') return 'Вы состоите в нескольких компаниях — в какую войти?'
  if (step.value === 'change-password') return 'Пароль по умолчанию нужно сменить перед началом работы.'
  return 'С возвращением — продолжим с того места, где вы остановились.'
})

const formattedCooldown = computed(() => {
  const s = cooldownSec.value
  if (s < 60) return `${s} с`
  const m = Math.floor(s / 60)
  const rest = s % 60
  return rest > 0 ? `${m} мин ${rest} с` : `${m} мин`
})

const loginButtonLabel = computed(() => {
  if (cooldownSec.value > 0) return `Подождите ${formattedCooldown.value}`
  return 'Войти'
})

function startCooldown(seconds) {
  cooldownSec.value = Math.max(0, Math.floor(seconds))
  if (cooldownTimer) clearInterval(cooldownTimer)
  if (cooldownSec.value <= 0) return
  cooldownTimer = setInterval(() => {
    cooldownSec.value -= 1
    if (cooldownSec.value <= 0) {
      clearInterval(cooldownTimer)
      cooldownTimer = null
    }
  }, 1000)
}

onMounted(() => {
  yandexConfig().then((cfg) => { yandexAuth.value = cfg }).catch(() => {})
  // После логина с force_change App.vue переключает layout-ветку и ПЕРЕСОЗДАЁТ
  // router-view — экран монтируется заново и теряет локальный шаг.
  // Восстанавливаем его по флагу сессии.
  if (authStore.token && authStore.forceChange) step.value = 'change-password'
})

onBeforeUnmount(() => {
  if (cooldownTimer) clearInterval(cooldownTimer)
})

async function handleLogin() {
  loginError.value = ''
  if (cooldownSec.value > 0) return
  if (!loginForm.login || !loginForm.password) {
    loginError.value = 'Введите логин и пароль'
    return
  }
  loading.value = true
  try {
    const result = await authStore.login(loginForm.login, loginForm.password)
    if (result.needsSelection) {
      openCompanyPicker(result.companies, result.selectToken)
      return
    }
    finishLogin(result.forceChange)
  } catch (e) {
    if (e?.error === 'EMAIL_NOT_VERIFIED') {
      // Email не подтверждён — ведём на экран ввода кода (с переотправкой).
      router.push({ path: '/verify-email', query: { email: e?.email || loginForm.login } })
    } else if (e?.status === 429 && e?.retry_after_sec) {
      startCooldown(e.retry_after_sec)
    } else {
      loginError.value = e?.message || 'Неверный логин или пароль'
    }
  } finally {
    loading.value = false
  }
}

function finishLogin(forceChange) {
  if (forceChange) {
    step.value = 'change-password'
    return
  }
  connectSocket()
  const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/home'
  router.push(redirect)
}

function openCompanyPicker(list, selectToken) {
  pickerCompanies.value = list || []
  pickerSelectToken.value = selectToken
  // Пред-выбор: последняя выбранная компания (localStorage), иначе первая.
  const last = Number(localStorage.getItem('gw_active_company_id'))
  const remembered = pickerCompanies.value.find((c) => c.company_id === last && c.is_active)
  const firstActive = pickerCompanies.value.find((c) => c.is_active)
  pickerSelected.value = (remembered || firstActive || pickerCompanies.value[0])?.company_id ?? null
  step.value = 'company'
}

async function confirmCompany() {
  if (!pickerSelected.value) return
  loading.value = true
  loginError.value = ''
  try {
    const result = await authStore.selectCompany(pickerSelectToken.value, pickerSelected.value)
    finishLogin(result.forceChange)
  } catch (e) {
    step.value = 'credentials'
    loginError.value = e?.message || 'Не удалось войти в выбранную компанию'
  } finally {
    loading.value = false
  }
}

async function handleChangeDefault() {
  changeError.value = ''
  if (changeForm.password.length < 8) {
    changeError.value = 'Пароль должен содержать не менее 8 символов'
    return
  }
  if (changeForm.password !== changeForm.confirmPassword) {
    changeError.value = 'Пароли не совпадают'
    return
  }
  changeLoading.value = true
  try {
    await authStore.changeDefaultCredentials({
      password: changeForm.password,
      confirmPassword: changeForm.confirmPassword,
    })
    connectSocket()
    router.push('/home')
  } catch (e) {
    changeError.value = e.message || 'Ошибка смены данных'
  } finally {
    changeLoading.value = false
  }
}
</script>

<style scoped>
/* Форма, разделитель «или» и строка «нет аккаунта» — общие классы экранов
   входа (main.css); здесь только специфика экрана. */
.lg-forgot {
  display: flex;
  justify-content: flex-end;
  margin-top: -8px;
}

.lg-alts {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.lg-yandex :deep(.btn-label) {
  display: inline-flex;
  align-items: center;
  gap: 7px;
}

@media (max-width: 560px) {
  .lg-alts { grid-template-columns: minmax(0, 1fr); }
}
</style>
