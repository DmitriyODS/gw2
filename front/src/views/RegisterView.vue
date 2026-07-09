<template>
  <AuthShell
    title="Создать аккаунт"
    subtitle="Заполните ФИО и почту — логин и пароль подставятся автоматически, при желании поправьте."
  >
    <form @submit.prevent="handleRegister" class="login-form">
          <div class="form-group">
            <label>ФИО</label>
            <input
              v-model.trim="form.fio"
              type="text"
              class="pill-input"
              :disabled="loading"
              autocomplete="name"
              placeholder="Фамилия Имя Отчество"
              @input="onFioInput"
            />
          </div>
          <div class="form-group">
            <label>Email</label>
            <input
              v-model.trim="form.email"
              type="email"
              class="pill-input"
              :disabled="loading"
              autocomplete="email"
              placeholder="name@example.com"
            />
          </div>
          <div class="form-group">
            <label>Логин</label>
            <input
              v-model.trim="form.login"
              type="text"
              class="pill-input"
              :disabled="loading"
              autocomplete="username"
              placeholder="Сгенерируется из ФИО"
              @input="loginTouched = true"
            />
          </div>
          <div class="form-group">
            <label>Пароль</label>
            <div class="input-wrap">
              <input
                v-model="form.password"
                :type="showPassword ? 'text' : 'password'"
                class="pill-input"
                :disabled="loading"
                autocomplete="new-password"
                placeholder="Не короче 8 символов"
              />
              <button type="button" class="field-btn" @click="regeneratePassword" title="Сгенерировать новый" tabindex="-1">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M23 4v6h-6"/><path d="M1 20v-6h6"/><path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15"/></svg>
              </button>
              <button type="button" class="field-btn" @click="copyPassword" :title="copied ? 'Скопировано' : 'Скопировать'" tabindex="-1">
                <svg v-if="!copied" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1"/></svg>
                <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
              </button>
              <button type="button" class="field-btn" @click="showPassword = !showPassword" tabindex="-1">
                <svg v-if="!showPassword" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0112 20c-7 0-11-8-11-8a18.45 18.45 0 015.06-5.94"/><path d="M9.9 4.24A9.12 9.12 0 0112 4c7 0 11 8 11 8a18.5 18.5 0 01-2.16 3.19"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
              </button>
            </div>
            <span class="field-hint">Сохраните пароль — он понадобится для входа.</span>
          </div>
          <p v-if="error" class="error-msg">{{ error }}</p>
          <button type="submit" class="btn-login" :disabled="loading">
            {{ loading ? 'Создаём…' : 'Зарегистрироваться' }}
          </button>
    </form>

    <p class="switch-line">
      Уже есть аккаунт?
      <RouterLink to="/login" class="switch-link">Войти</RouterLink>
    </p>
  </AuthShell>
</template>

<script setup>
import { reactive, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { useThemeStore } from '@/stores/theme.js'
import { suggestLogin } from '@/api/auth.js'
import AuthShell from '@/components/auth/AuthShell.vue'

const router = useRouter()
const authStore = useAuthStore()
const themeStore = useThemeStore()

const form = reactive({ fio: '', email: '', login: '', password: '' })
const error = ref('')
const loading = ref(false)
const showPassword = ref(false)
const copied = ref(false)
const loginTouched = ref(false)
let suggestTimer = null

onMounted(() => {
  themeStore.init()
  regeneratePassword()
})

// Безопасный пароль на клиенте (Web Crypto): без двусмысленных символов.
function generatePassword(len = 12) {
  const alphabet = 'abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789'
  const arr = new Uint32Array(len)
  crypto.getRandomValues(arr)
  let out = ''
  for (let i = 0; i < len; i++) out += alphabet[arr[i] % alphabet.length]
  return out
}

function regeneratePassword() {
  form.password = generatePassword()
  showPassword.value = true
}

async function copyPassword() {
  try {
    await navigator.clipboard.writeText(form.password)
    copied.value = true
    setTimeout(() => { copied.value = false }, 1500)
  } catch { /* clipboard недоступен */ }
}

// Live-подсказка логина по ФИО (debounce), пока пользователь не правил поле сам.
function onFioInput() {
  if (loginTouched.value) return
  clearTimeout(suggestTimer)
  const fio = form.fio
  suggestTimer = setTimeout(async () => {
    if (loginTouched.value || !fio.trim()) return
    try {
      const { login } = await suggestLogin(fio)
      if (!loginTouched.value && login) form.login = login
    } catch { /* подсказка необязательна */ }
  }, 400)
}

async function handleRegister() {
  error.value = ''
  if (!form.fio) { error.value = 'Укажите ФИО'; return }
  if (!form.email || !/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(form.email)) {
    error.value = 'Укажите корректный email'
    return
  }
  if (form.login && form.login.length < 3) {
    error.value = 'Логин должен содержать не менее 3 символов'
    return
  }
  if (form.password.length < 8) {
    error.value = 'Пароль должен содержать не менее 8 символов'
    return
  }
  loading.value = true
  try {
    const { email } = await authStore.register({
      fio: form.fio, email: form.email, login: form.login, password: form.password,
    })
    router.push({ path: '/verify-email', query: { email: email || form.email } })
  } catch (e) {
    error.value = e?.message || 'Не удалось зарегистрироваться'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
/* Каркас страницы (фон, сплит-карточка, промо) — в AuthShell.vue;
   здесь только форма регистрации. */
.login-form {
  display: flex;
  flex-direction: column;
  gap: 18px;
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

.field-hint {
  font-size: 12px;
  color: var(--color-text-dim);
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
  padding-right: 116px;
}

.field-btn {
  position: absolute;
  background: none;
  border: none;
  padding: 0;
  cursor: pointer;
  color: var(--color-outline);
  display: flex;
  align-items: center;
  line-height: 0;
}

.field-btn:hover {
  color: var(--color-primary);
}

.field-btn svg {
  width: 18px;
  height: 18px;
}

.input-wrap .field-btn:nth-of-type(1) { right: 78px; }
.input-wrap .field-btn:nth-of-type(2) { right: 46px; }
.input-wrap .field-btn:nth-of-type(3) { right: 14px; }

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

</style>
