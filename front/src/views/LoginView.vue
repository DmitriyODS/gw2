<template>
  <!-- Экран входа ведёт три шага в ОДНОЙ карточке: учётные данные, выбор
       компании и обязательная смена пароля. Отдельных модалок больше нет —
       так шаги живут в общем языке экранов входа. -->
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
        label="логин"
        placeholder="логин"
        autocomplete="username"
        :disabled="isLoginDisabled"
      />
      <AuthField
        v-model="loginForm.password"
        label="пароль"
        type="password"
        placeholder="пароль"
        autocomplete="current-password"
        :disabled="isLoginDisabled"
      />

      <div v-if="cooldownSec > 0" class="auth-error" role="status" aria-live="polite">
        <span class="material-symbols-outlined">lock_clock</span>
        <span>
          Слишком много неудачных попыток — попробуйте через {{ formattedCooldown }}
        </span>
      </div>
      <p v-else-if="loginError" class="auth-error">{{ loginError }}</p>

      <button type="submit" class="auth-submit" :disabled="isLoginDisabled">
        {{ loginButtonLabel }}
      </button>

      <div class="lg-alts">
        <button type="button" class="auth-alt" @click="goYandex">
          <YandexLogo :size="16" />
          Войти через Яндекс
        </button>
        <RouterLink to="/qr-login" class="auth-alt">
          <span class="material-symbols-outlined">qr_code_2</span>
          Войти по QR-коду
        </RouterLink>
        <RouterLink to="/tv-activate" class="auth-alt">
          <span class="material-symbols-outlined">tv</span>
          ТВ-режим
        </RouterLink>
        <RouterLink to="/forgot-password" class="auth-alt">
          <span class="material-symbols-outlined">lock_reset</span>
          Сбросить пароль
        </RouterLink>
      </div>

      <p class="lg-switch">
        Нет аккаунта?
        <RouterLink to="/register">создать</RouterLink>
      </p>
    </form>

    <!-- ── Шаг 2: выбор компании ─────────────────────────────── -->
    <div v-else-if="step === 'company'" class="auth-form">
      <div class="lg-companies">
        <button
          v-for="c in pickerCompanies"
          :key="c.company_id"
          type="button"
          class="lg-company"
          :class="{ active: pickerSelected === c.company_id }"
          :disabled="!c.is_active"
          @click="pickerSelected = c.company_id"
        >
          <span class="lg-company-main">
            <span class="lg-company-name">{{ c.company_name }}</span>
            <span class="lg-company-role">
              {{ c.role_name }}<template v-if="!c.is_active"> · отключена</template>
            </span>
          </span>
          <span v-if="pickerSelected === c.company_id" class="material-symbols-outlined">check_circle</span>
        </button>
      </div>
      <p v-if="loginError" class="auth-error">{{ loginError }}</p>
      <button type="button" class="auth-submit" :disabled="loading || !pickerSelected" @click="confirmCompany">
        {{ loading ? 'входим…' : 'войти' }}
      </button>
    </div>

    <!-- ── Шаг 3: обязательная смена пароля ──────────────────── -->
    <form v-else class="auth-form" @submit.prevent="handleChangeDefault">
      <AuthField
        v-model="changeForm.password"
        label="новый пароль"
        type="password"
        placeholder="не короче 8 символов"
        autocomplete="new-password"
        :disabled="changeLoading"
      />
      <AuthField
        v-model="changeForm.confirmPassword"
        label="повторите пароль"
        type="password"
        placeholder="ещё раз"
        autocomplete="new-password"
        :disabled="changeLoading"
      />
      <p v-if="changeError" class="auth-error">{{ changeError }}</p>
      <button type="submit" class="auth-submit" :disabled="changeLoading">
        {{ changeLoading ? 'сохраняем…' : 'сохранить и войти' }}
      </button>
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
  if (step.value === 'company') return 'выбор компании'
  if (step.value === 'change-password') return 'смена пароля'
  return 'вход в аккаунт'
})

const stepSubtitle = computed(() => {
  if (step.value === 'company') return 'Вы состоите в нескольких компаниях — в какую войти?'
  if (step.value === 'change-password') return 'Пароль по умолчанию нужно сменить перед началом работы.'
  return ''
})

const formattedCooldown = computed(() => {
  const s = cooldownSec.value
  if (s < 60) return `${s} с`
  const m = Math.floor(s / 60)
  const rest = s % 60
  return rest > 0 ? `${m} мин ${rest} с` : `${m} мин`
})

const loginButtonLabel = computed(() => {
  if (cooldownSec.value > 0) return `подождите ${formattedCooldown.value}`
  return loading.value ? 'входим…' : 'войти'
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
/* Форма, кнопки и плашка ошибки — общие классы экранов входа (main.css);
   здесь только специфика: сетка альтернативных способов и выбор компании. */
.lg-alts {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}

.lg-switch {
  margin: 0;
  text-align: center;
  font-size: 13.5px;
  color: var(--color-text-dim);
}

.lg-switch a {
  color: var(--color-primary);
  font-weight: 600;
  text-decoration: none;
  margin-left: 4px;
}

.lg-switch a:hover { text-decoration: underline; }

/* Выбор компании */
.lg-companies {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.lg-company {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 13px 16px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: color-mix(in oklch, var(--color-surface) 60%, transparent);
  color: var(--color-text);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}

.lg-company:hover:not(:disabled) {
  border-color: color-mix(in oklch, var(--color-primary) 35%, var(--acrylic-border));
}

.lg-company.active {
  border-color: var(--color-primary);
  background: color-mix(in oklch, var(--color-primary) 14%, var(--color-surface));
}

.lg-company:disabled { opacity: 0.5; cursor: not-allowed; }

.lg-company-main { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.lg-company-name { font-size: 15px; font-weight: 600; }
.lg-company-role { font-size: 12px; color: var(--color-text-dim); }
.lg-company .material-symbols-outlined { color: var(--color-primary); }

@media (max-width: 560px) {
  .lg-alts { grid-template-columns: 1fr; }
}
</style>
