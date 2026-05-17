<template>
  <div class="login-page">
    <div class="login-wrapper">
      <!-- Лого выше карточки, перекрывает её верхний край -->
      <div class="login-logo-wrap">
        <img src="/logo.svg" alt="Groove Work" class="login-logo-img" />
      </div>

      <div class="login-card">
        <form @submit.prevent="handleLogin" class="login-form">
          <div class="form-group">
            <label>Логин</label>
            <input
              v-model="loginForm.login"
              type="text"
              class="pill-input"
              :disabled="loading"
              autocomplete="username"
            />
          </div>
          <div class="form-group">
            <label>Пароль</label>
            <div class="input-wrap">
              <input
                v-model="loginForm.password"
                :type="showLoginPassword ? 'text' : 'password'"
                class="pill-input"
                :disabled="loading"
                autocomplete="current-password"
              />
              <button type="button" class="eye-btn" @click="showLoginPassword = !showLoginPassword" tabindex="-1">
                <svg v-if="!showLoginPassword" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0112 20c-7 0-11-8-11-8a18.45 18.45 0 015.06-5.94"/><path d="M9.9 4.24A9.12 9.12 0 0112 4c7 0 11 8 11 8a18.5 18.5 0 01-2.16 3.19"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
              </button>
            </div>
          </div>
          <p v-if="loginError" class="error-msg">{{ loginError }}</p>
          <button type="submit" class="btn-login" :disabled="loading">
            {{ loading ? 'Входим...' : 'Войти' }}
          </button>
        </form>
      </div>
    </div>

    <!-- Неклозабельная модалка смены учётных данных -->
    <Dialog
      v-if="showChangeModal"
      :visible="true"
      :closable="false"
      :modal="true"
      :close-on-escape="false"
      header="Смена учётных данных"
      style="width:460px"
    >
      <p class="change-hint">
        Пожалуйста, смените логин и пароль перед началом работы.
      </p>
      <form @submit.prevent="handleChangeDefault" class="change-form">
        <div class="form-group">
          <label>Новый логин</label>
          <input
            v-model="changeForm.login"
            type="text"
            class="pill-input"
            autocomplete="new-username"
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
    </Dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { useThemeStore } from '@/stores/theme.js'
import { connectSocket } from '@/socket/index.js'
import Dialog from 'primevue/dialog'

const router = useRouter()
const authStore = useAuthStore()
const themeStore = useThemeStore()

const loginForm = reactive({ login: '', password: '' })
const loginError = ref('')
const loading = ref(false)
const showChangeModal = ref(false)
const showLoginPassword = ref(false)

const changeForm = reactive({ login: '', password: '', confirmPassword: '' })
const changeError = ref('')
const changeLoading = ref(false)
const showNewPassword = ref(false)
const showConfirmPassword = ref(false)

onMounted(() => {
  themeStore.init()
})

async function handleLogin() {
  loginError.value = ''
  if (!loginForm.login || !loginForm.password) {
    loginError.value = 'Введите логин и пароль'
    return
  }
  loading.value = true
  try {
    const needsChange = await authStore.login(loginForm.login, loginForm.password)
    if (needsChange) {
      showChangeModal.value = true
    } else {
      connectSocket()
      router.push('/tasks')
    }
  } catch {
    loginError.value = 'Неверный логин или пароль'
  } finally {
    loading.value = false
  }
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
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg);
  padding: 24px;
}

/* Обёртка: лого + карточка в одной колонке */
.login-wrapper {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  max-width: 420px;
}

/* Лого перекрывает верхний край карточки */
.login-logo-wrap {
  position: relative;
  z-index: 2;
  margin-bottom: -36px;
}

.login-logo-img {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  display: block;
  filter: drop-shadow(var(--shadow-lg));
}

.login-card {
  width: 100%;
  background: var(--color-surface);
  border-radius: var(--radius-xl);
  padding: 64px 40px 40px;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-lg);
}

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

/* Pill-инпут: только border, без фона */
.pill-input {
  width: 100%;
  height: 48px;
  border-radius: var(--radius-full);
  border: 1.5px solid var(--color-outline);
  background: transparent;
  padding: 0 20px;
  font-size: 15px;
  color: var(--color-text);
  outline: none;
  transition: border-color 0.15s, box-shadow 0.15s;
  box-sizing: border-box;
  font-family: inherit;
}

.pill-input:focus {
  border-color: var(--color-primary);
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

.btn-login {
  width: 100%;
  height: 52px;
  border-radius: 999px;
  border: none;
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 16px;
  font-weight: 700;
  cursor: pointer;
  transition: background 0.15s, transform 0.1s, box-shadow 0.15s;
  margin-top: 8px;
  letter-spacing: 0.02em;
}

.btn-login:hover:not(:disabled) {
  background: var(--color-primary-hover);
  box-shadow: var(--shadow-lg);
  transform: translateY(-1px);
}

.btn-login:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
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

@media (max-width: 480px) {
  .login-page {
    padding: 16px;
  }

  .login-card {
    padding: 56px 24px 28px;
  }
}
</style>
