<template>
  <div class="login-page">
    <div class="login-wrapper">
      <div class="login-logo-wrap">
        <Logo class="login-logo-img" :size="80" />
      </div>

      <div class="login-card">
        <h1 class="register-title">Новый пароль</h1>
        <p class="register-sub">Придумайте новый пароль для входа в Groove Work.</p>

        <form v-if="token" @submit.prevent="submit" class="login-form">
          <div class="form-group">
            <label>Новый пароль</label>
            <div class="input-wrap">
              <input
                v-model="password"
                :type="show ? 'text' : 'password'"
                class="pill-input"
                :disabled="loading"
                autocomplete="new-password"
                placeholder="Не короче 8 символов"
              />
              <button type="button" class="eye-btn" @click="show = !show" tabindex="-1">
                <svg v-if="!show" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>
                <svg v-else viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17.94 17.94A10.07 10.07 0 0112 20c-7 0-11-8-11-8a18.45 18.45 0 015.06-5.94"/><path d="M9.9 4.24A9.12 9.12 0 0112 4c7 0 11 8 11 8a18.5 18.5 0 01-2.16 3.19"/><line x1="1" y1="1" x2="23" y2="23"/></svg>
              </button>
            </div>
          </div>
          <div class="form-group">
            <label>Повторите пароль</label>
            <input
              v-model="confirm"
              :type="show ? 'text' : 'password'"
              class="pill-input"
              :disabled="loading"
              autocomplete="new-password"
              placeholder="Ещё раз"
            />
          </div>
          <p v-if="error" class="error-msg">{{ error }}</p>
          <button type="submit" class="btn-login" :disabled="loading">
            {{ loading ? 'Сохраняем…' : 'Сохранить пароль' }}
          </button>
        </form>

        <template v-else>
          <p class="error-msg">Ссылка недействительна — токен не найден. Запросите сброс пароля заново.</p>
          <RouterLink to="/forgot-password" class="btn-login as-link">Запросить заново</RouterLink>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { useThemeStore } from '@/stores/theme.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import Logo from '@/components/common/Logo.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const themeStore = useThemeStore()
const notif = useNotificationsStore()

const token = ref(route.query.token || '')
const password = ref('')
const confirm = ref('')
const show = ref(false)
const error = ref('')
const loading = ref(false)

onMounted(() => themeStore.init())

async function submit() {
  error.value = ''
  if (password.value.length < 8) {
    error.value = 'Пароль должен содержать минимум 8 символов'
    return
  }
  if (password.value !== confirm.value) {
    error.value = 'Пароли не совпадают'
    return
  }
  loading.value = true
  try {
    const { login } = await authStore.resetPassword(token.value, password.value)
    notif.success('Пароль обновлён — войдите с новым паролем')
    router.push({ path: '/login', query: login ? { login } : {} })
  } catch (e) {
    error.value = e?.message || 'Не удалось сменить пароль'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg);
  padding: 24px;
}
.login-wrapper { display: flex; flex-direction: column; align-items: center; width: 100%; max-width: 420px; }
.login-logo-wrap { position: relative; z-index: 2; margin-bottom: -36px; }
.login-logo-img { width: 72px; height: 72px; border-radius: 50%; display: block; filter: drop-shadow(var(--shadow-lg)); }
.login-card {
  width: 100%;
  background: var(--acrylic-card-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  padding: 64px 40px 40px;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-lg);
}
.register-title { margin: 0 0 6px; font-size: 22px; font-weight: 800; color: var(--color-text); text-align: center; }
.register-sub { margin: 0 0 24px; font-size: 14px; line-height: 1.5; color: var(--color-text-dim); text-align: center; }
.login-form { display: flex; flex-direction: column; gap: 20px; }
.form-group { display: flex; flex-direction: column; gap: 8px; }
.form-group label { font-size: 12px; font-weight: 700; color: var(--color-primary); text-transform: uppercase; letter-spacing: 0.08em; }
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
.pill-input:focus { border-color: var(--color-primary); box-shadow: 0 0 0 3px color-mix(in oklch, var(--color-primary) 15%, transparent); }
.pill-input:disabled { opacity: 0.5; cursor: not-allowed; }
.input-wrap { position: relative; display: flex; align-items: center; }
.input-wrap .pill-input { padding-right: 48px; }
.eye-btn { position: absolute; right: 14px; background: none; border: none; padding: 0; cursor: pointer; color: var(--color-outline); display: flex; align-items: center; line-height: 0; }
.eye-btn:hover { color: var(--color-primary); }
.eye-btn svg { width: 18px; height: 18px; }
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
.btn-login:hover:not(:disabled) { background: var(--color-primary-hover); box-shadow: var(--shadow-lg); transform: translateY(-1px); }
.btn-login:disabled { opacity: 0.6; cursor: not-allowed; transform: none; }
.btn-login.as-link { display: inline-flex; align-items: center; justify-content: center; text-decoration: none; margin-top: 16px; }
@media (max-width: 480px) {
  .login-page { padding: 16px; }
  .login-card { padding: 56px 24px 28px; }
}
</style>
