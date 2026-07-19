<template>
  <AuthShell title="С возвращением!" subtitle="Войдите, чтобы продолжить работу.">
    <form @submit.prevent="handleLogin" class="login-form">
          <div class="form-group">
            <label>Логин</label>
            <input
              v-model="loginForm.login"
              type="text"
              class="pill-input"
              :disabled="isLoginDisabled"
              autocomplete="username"
              placeholder="Введите логин"
            />
          </div>
          <div class="form-group">
            <label>Пароль</label>
            <div class="input-wrap">
              <input
                v-model="loginForm.password"
                :type="showLoginPassword ? 'text' : 'password'"
                class="pill-input"
                :disabled="isLoginDisabled"
                autocomplete="current-password"
                placeholder="Введите пароль"
              />
              <button type="button" class="eye-btn" @click="showLoginPassword = !showLoginPassword" tabindex="-1">
                <svg v-if="!showLoginPassword" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0112 20c-7 0-11-8-11-8a18.45 18.45 0 015.06-5.94"/><path d="M9.9 4.24A9.12 9.12 0 0112 4c7 0 11 8 11 8a18.5 18.5 0 01-2.16 3.19"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
              </button>
            </div>
          </div>
          <div class="forgot-row">
            <RouterLink to="/forgot-password" class="forgot-link">Забыли пароль?</RouterLink>
          </div>
          <div v-if="cooldownSec > 0" class="cooldown-box" role="status" aria-live="polite">
            <span class="material-symbols-outlined">lock_clock</span>
            <div class="cooldown-text">
              <div class="cooldown-title">Слишком много неудачных попыток</div>
              <div class="cooldown-sub">Попробуйте снова через {{ formattedCooldown }}</div>
            </div>
          </div>
          <p v-else-if="loginError" class="error-msg">{{ loginError }}</p>
          <button type="submit" class="btn-login" :disabled="isLoginDisabled">
            {{ loginButtonLabel }}
          </button>
    </form>

    <p class="switch-line">
      Нет аккаунта?
      <RouterLink to="/register" class="switch-link">Зарегистрироваться</RouterLink>
    </p>

    <div class="alt-login">
      <button v-if="yandexAuth.enabled" type="button" class="alt-login-btn" @click="goYandex">
        <span class="ya-badge">Я</span>
        Войти с Яндексом
      </button>
      <button type="button" class="alt-login-btn" @click="showQrLogin = true">
        <span class="material-symbols-outlined">qr_code_2</span>
        Войти по QR-коду
      </button>
      <button type="button" class="alt-login-btn" @click="showTvActivate = true">
        <span class="material-symbols-outlined">tv</span>
        Активировать ТВ-режим
      </button>
    </div>

    <template #overlays>
    <!-- Вход по QR: показываем код, ждём подтверждения с телефона -->
    <AppDialog
      v-if="showQrLogin"
      model-value
      icon="qr_code_2"
      size="sm"
      title="Вход по QR-коду"
      subtitle="Отсканируйте код телефоном, где вы уже вошли."
      @update:modelValue="showQrLogin = false"
    >
      <DeviceLinkInitiator kind="login" @session="onLoginQrSession" />
    </AppDialog>

    <!-- Активация ТВ-режима: код для подтверждения из настроек аккаунта -->
    <AppDialog
      v-if="showTvActivate"
      model-value
      icon="tv"
      size="sm"
      title="ТВ-режим"
      subtitle="Подтвердите код в приложении → «Авторизовать ТВ-киоск»."
      @update:modelValue="showTvActivate = false"
    >
      <DeviceLinkInitiator kind="tv" @session="onTvSession" />
    </AppDialog>
    <!-- Неклозабельная модалка смены учётных данных -->
    <AppDialog
      v-if="showChangeModal"
      model-value
      tone="warning"
      icon="lock_reset"
      size="sm"
      title="Смена учётных данных"
      subtitle="Пожалуйста, смените логин и пароль перед началом работы."
      :closable="false"
    >
      <form @submit.prevent="handleChangeDefault" class="change-form">
        <div class="form-group">
          <label>Новый логин</label>
          <input
            v-model="changeForm.login"
            type="text"
            class="pill-input"
            autocomplete="new-username"
            placeholder="Не короче 3 символов"
          />
        </div>
        <div class="form-group">
          <label>Новый пароль</label>
          <div class="input-wrap">
            <input
              v-model="changeForm.password"
              :type="showNewPassword ? 'text' : 'password'"
              class="pill-input"
              autocomplete="new-password"
              placeholder="Не короче 8 символов"
            />
            <button type="button" class="eye-btn" @click="showNewPassword = !showNewPassword" tabindex="-1">
              <svg v-if="!showNewPassword" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
              <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0112 20c-7 0-11-8-11-8a18.45 18.45 0 015.06-5.94"/><path d="M9.9 4.24A9.12 9.12 0 0112 4c7 0 11 8 11 8a18.5 18.5 0 01-2.16 3.19"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
            </button>
          </div>
        </div>
        <div class="form-group">
          <label>Подтвердите пароль</label>
          <div class="input-wrap">
            <input
              v-model="changeForm.confirmPassword"
              :type="showConfirmPassword ? 'text' : 'password'"
              class="pill-input"
              autocomplete="new-password"
              placeholder="Повторите новый пароль"
            />
            <button type="button" class="eye-btn" @click="showConfirmPassword = !showConfirmPassword" tabindex="-1">
              <svg v-if="!showConfirmPassword" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
              <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0112 20c-7 0-11-8-11-8a18.45 18.45 0 015.06-5.94"/><path d="M9.9 4.24A9.12 9.12 0 0112 4c7 0 11 8 11 8a18.5 18.5 0 01-2.16 3.19"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
            </button>
          </div>
        </div>
        <p v-if="changeError" class="error-msg">{{ changeError }}</p>
        <button type="submit" class="btn-login" :disabled="changeLoading">
          {{ changeLoading ? 'Сохраняем...' : 'Сохранить и войти' }}
        </button>
      </form>
    </AppDialog>

    <!-- Выбор компании при логине (несколько компаний у пользователя) -->
    <AppDialog
      v-if="showCompanyPicker"
      model-value
      icon="apartment"
      size="sm"
      title="Выберите компанию"
      subtitle="Вы состоите в нескольких компаниях. В какую войти?"
      :closable="false"
    >
      <div class="company-picker">
        <button
          v-for="c in pickerCompanies"
          :key="c.company_id"
          type="button"
          class="company-option"
          :class="{ active: pickerSelected === c.company_id, disabled: !c.is_active }"
          :disabled="!c.is_active"
          @click="pickerSelected = c.company_id"
        >
          <span class="company-option-main">
            <span class="company-option-name">{{ c.company_name }}</span>
            <span class="company-option-role">{{ c.role_name }}<template v-if="!c.is_active"> · отключена</template></span>
          </span>
          <span v-if="pickerSelected === c.company_id" class="material-symbols-outlined">check_circle</span>
        </button>
      </div>
      <p v-if="loginError" class="error-msg">{{ loginError }}</p>
      <button type="button" class="btn-login" :disabled="loading || !pickerSelected" @click="confirmCompany">
        {{ loading ? 'Входим...' : 'Войти' }}
      </button>
    </AppDialog>
    </template>
  </AuthShell>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { useThemeStore } from '@/stores/theme.js'
import { connectSocket } from '@/socket/index.js'
import AppDialog from '@/components/common/AppDialog.vue'
import AuthShell from '@/components/auth/AuthShell.vue'
import DeviceLinkInitiator from '@/components/auth/DeviceLinkInitiator.vue'
import { yandexConfig, yandexAuthURL } from '@/api/auth.js'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const themeStore = useThemeStore()

const loginForm = reactive({ login: '', password: '' })
const loginError = ref('')
const loading = ref(false)
const showChangeModal = ref(false)
const showLoginPassword = ref(false)

// Брутфорс-блокировка: бэк отвечает 429 + retry_after_sec, локально
// тикаем секунды и блокируем форму до конца таймера.
const cooldownSec = ref(0)
let cooldownTimer = null

// Выбор компании при логине (если их несколько).
const showCompanyPicker = ref(false)
const pickerCompanies = ref([])
const pickerSelectToken = ref('')
const pickerSelected = ref(null)

// Вход по QR / активация ТВ-режима.
const showQrLogin = ref(false)
const showTvActivate = ref(false)

// Вход через Яндекс ID: кнопка видна, только если сервер настроен.
const yandexAuth = ref({ enabled: false, client_id: '' })
function goYandex() {
  window.location.href = yandexAuthURL(yandexAuth.value.client_id)
}

const changeForm = reactive({ login: '', password: '', confirmPassword: '' })
const changeError = ref('')
const changeLoading = ref(false)
const showNewPassword = ref(false)
const showConfirmPassword = ref(false)

const isLoginDisabled = computed(() => loading.value || cooldownSec.value > 0)

const formattedCooldown = computed(() => {
  const s = cooldownSec.value
  if (s < 60) return `${s} с`
  const m = Math.floor(s / 60)
  const rest = s % 60
  return rest > 0 ? `${m} мин ${rest} с` : `${m} мин`
})

const loginButtonLabel = computed(() => {
  if (cooldownSec.value > 0) return `Подождите ${formattedCooldown.value}`
  return loading.value ? 'Входим...' : 'Войти'
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
  themeStore.init()
  yandexConfig().then((cfg) => { yandexAuth.value = cfg }).catch(() => {})
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
      // Email не подтверждён — ведём на экран ввода кода (с возможностью переотправки).
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
    showChangeModal.value = true
  } else {
    connectSocket()
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/tasks'
    router.push(redirect)
  }
}

function openCompanyPicker(list, selectToken) {
  pickerCompanies.value = list || []
  pickerSelectToken.value = selectToken
  // Пред-выбор: последняя выбранная компания (localStorage), иначе первая.
  const last = Number(localStorage.getItem('gw_active_company_id'))
  const remembered = pickerCompanies.value.find((c) => c.company_id === last && c.is_active)
  const firstActive = pickerCompanies.value.find((c) => c.is_active)
  pickerSelected.value = (remembered || firstActive || pickerCompanies.value[0])?.company_id ?? null
  showCompanyPicker.value = true
}

async function confirmCompany() {
  if (!pickerSelected.value) return
  loading.value = true
  loginError.value = ''
  try {
    const result = await authStore.selectCompany(pickerSelectToken.value, pickerSelected.value)
    showCompanyPicker.value = false
    finishLogin(result.forceChange)
  } catch (e) {
    showCompanyPicker.value = false
    loginError.value = e?.message || 'Не удалось войти в выбранную компанию'
  } finally {
    loading.value = false
  }
}

// Вход подтверждён с телефона (QR-вход): применяем сессию как обычный login.
function onLoginQrSession(session) {
  const result = authStore.applyLinkSession(session)
  showQrLogin.value = false
  if (result.needsSelection) {
    openCompanyPicker(result.companies, result.selectToken)
    return
  }
  finishLogin(result.forceChange)
}

// ТВ-киоск авторизован (сессия уже привязана к компании) — уходим в ТВ-режим.
function onTvSession(session) {
  authStore.applyLinkSession(session)
  showTvActivate.value = false
  connectSocket()
  router.push('/tv')
}

async function handleChangeDefault() {
  changeError.value = ''
  if (changeForm.login.length < 3) {
    changeError.value = 'Логин должен содержать не менее 3 символов'
    return
  }
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
      login: changeForm.login,
      password: changeForm.password,
      confirmPassword: changeForm.confirmPassword,
    })
    showChangeModal.value = false
    connectSocket()
    router.push('/tasks')
  } catch (e) {
    changeError.value = e.message || 'Ошибка смены данных'
  } finally {
    changeLoading.value = false
  }
}
</script>

<style scoped>
/* Каркас страницы (фон, сплит-карточка, промо) — в AuthShell.vue;
   здесь только форма входа и содержимое диалогов. */
.login-form {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-group label {
  font-size: 12px;
  font-weight: 700;
  color: var(--color-primary);
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

/* Pill-инпут: полупрозрачное стекло с мягкой обводкой */
.pill-input {
  width: 100%;
  height: 48px;
  border-radius: var(--radius-full);
  border: 1.5px solid color-mix(in oklch, var(--color-outline) 55%, transparent);
  background: color-mix(in oklch, var(--color-surface) 42%, transparent);
  padding: 0 20px;
  font-size: 15px;
  color: var(--color-text);
  outline: none;
  transition: border-color 0.15s, box-shadow 0.15s, background 0.15s;
  box-sizing: border-box;
  font-family: inherit;
}

.pill-input:focus {
  border-color: var(--color-primary);
  background: color-mix(in oklch, var(--color-surface) 65%, transparent);
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--color-primary) 15%, transparent);
}

.pill-input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.input-wrap {
  position: relative;
  display: flex;
  align-items: center;
}

.input-wrap .pill-input {
  padding-right: 48px;
}

.eye-btn {
  position: absolute;
  right: 14px;
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  color: var(--color-outline);
  display: flex;
  align-items: center;
  line-height: 0;
}

.eye-btn:hover {
  color: var(--color-primary);
}

.eye-btn svg {
  width: 18px;
  height: 18px;
}

.error-msg {
  margin: 0;
  font-size: 13px;
  color: var(--color-on-error-container);
  padding: 8px 16px;
  background: var(--color-error-container);
  border-radius: 999px;
  border: 1px solid color-mix(in oklch, var(--color-error) 30%, var(--color-outline-dim));
  text-align: center;
}

.cooldown-box {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: var(--radius-lg);
  background: var(--color-error-container);
  color: var(--color-on-error-container);
  border: 1px solid color-mix(in oklch, var(--color-error) 30%, var(--color-outline-dim));
}

.cooldown-box .material-symbols-outlined {
  font-size: 26px;
  flex-shrink: 0;
}

.cooldown-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.cooldown-title {
  font-size: 13px;
  font-weight: 700;
  line-height: 1.3;
}

.cooldown-sub {
  font-size: 13px;
  font-variant-numeric: tabular-nums;
  opacity: 0.9;
}

.btn-login {
  width: 100%;
  height: 52px;
  border-radius: 999px;
  border: none;
  background: var(--grad-primary);
  color: var(--color-on-primary);
  font-size: 16px;
  font-weight: 700;
  cursor: pointer;
  transition: filter 0.15s, transform 0.1s;
  margin-top: 8px;
  letter-spacing: 0.02em;
}

.btn-login:hover:not(:disabled) {
  filter: brightness(1.06);
  transform: translateY(-1px);
}

.btn-login:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}

.switch-line {
  margin: 20px 0 0;
  text-align: center;
  font-size: 14px;
  color: var(--color-text-dim);
}

.switch-link {
  color: var(--color-primary);
  font-weight: 700;
  text-decoration: none;
  margin-left: 4px;
}

.switch-link:hover {
  text-decoration: underline;
}

.forgot-row {
  display: flex;
  justify-content: flex-end;
  margin-top: -8px;
}
.forgot-link {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-dim);
  text-decoration: none;
}
.forgot-link:hover { color: var(--color-primary); text-decoration: underline; }

/* Company picker dialog */
.company-picker {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 16px;
}

.company-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px;
  border-radius: var(--radius-lg);
  border: 1.5px solid var(--color-outline-dim);
  background: var(--color-surface);
  color: var(--color-text);
  cursor: pointer;
  text-align: left;
  transition: border-color 0.15s, background 0.15s;
}

.company-option:hover:not(.disabled) {
  border-color: var(--color-primary);
}

.company-option.active {
  border-color: var(--color-primary);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.company-option.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.company-option-main {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.company-option-name {
  font-size: 15px;
  font-weight: 600;
}

.company-option-role {
  font-size: 12px;
  color: var(--color-text-dim);
}

.company-option .material-symbols-outlined {
  color: var(--color-primary);
}

/* Change credentials dialog */
.change-hint {
  margin: 0 0 20px;
  font-size: 14px;
  color: var(--gw-text-secondary);
  line-height: 1.5;
}

.change-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* Альтернативные способы входа (QR / ТВ) */
.alt-login {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  justify-content: center;
  margin-top: 18px;
}
.alt-login-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border: none;
  border-radius: var(--radius-pill, 999px);
  background: var(--acrylic-card-bg, transparent);
  color: var(--color-text-secondary);
  font-size: 0.85rem;
  font-weight: 600;
  cursor: pointer;
  transition: color 0.15s, background 0.15s;
}
.alt-login-btn:hover { color: var(--color-primary); }
.alt-login-btn .material-symbols-outlined { font-size: 18px; }
/* Значок Яндекса: фирменная буква без внешних ресурсов. */
.ya-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  max-width: 18px;
  min-height: 18px;
  max-height: 18px;
  border-radius: 50%;
  background: var(--color-error-container, var(--color-surface-variant));
  color: var(--color-error, currentColor);
  font-size: 12px;
  font-weight: 800;
}

</style>
