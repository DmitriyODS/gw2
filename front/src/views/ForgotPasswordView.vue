<template>
  <div class="login-page">
    <div class="login-wrapper">
      <div class="login-logo-wrap">
        <Logo class="login-logo-img" :size="80" />
      </div>

      <div class="login-card">
        <template v-if="sent">
          <h1 class="register-title">Проверьте почту</h1>
          <p class="register-sub">
            Если аккаунт с адресом <b>{{ email }}</b> существует, мы отправили на него письмо со ссылкой для сброса пароля.
          </p>
          <RouterLink to="/login" class="btn-login as-link">Вернуться ко входу</RouterLink>
        </template>

        <template v-else>
          <h1 class="register-title">Сброс пароля</h1>
          <p class="register-sub">Укажите email — пришлём ссылку для установки нового пароля.</p>

          <form @submit.prevent="submit" class="login-form">
            <div class="form-group">
              <label>Email</label>
              <input
                v-model.trim="email"
                type="email"
                class="pill-input"
                :disabled="loading"
                autocomplete="email"
                placeholder="name@example.com"
              />
            </div>
            <p v-if="error" class="error-msg">{{ error }}</p>
            <button type="submit" class="btn-login" :disabled="loading">
              {{ loading ? 'Отправляем…' : 'Отправить ссылку' }}
            </button>
          </form>

          <p class="switch-line">
            Вспомнили пароль?
            <RouterLink to="/login" class="switch-link">Войти</RouterLink>
          </p>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth.js'
import { useThemeStore } from '@/stores/theme.js'
import Logo from '@/components/common/Logo.vue'

const authStore = useAuthStore()
const themeStore = useThemeStore()

const email = ref('')
const error = ref('')
const loading = ref(false)
const sent = ref(false)

onMounted(() => themeStore.init())

async function submit() {
  error.value = ''
  if (!email.value || !/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(email.value)) {
    error.value = 'Укажите корректный email'
    return
  }
  loading.value = true
  try {
    await authStore.forgotPassword(email.value)
    sent.value = true
  } catch (e) {
    error.value = e?.message || 'Не удалось отправить письмо'
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
  backdrop-filter: var(--acrylic-blur);
  -webkit-backdrop-filter: var(--acrylic-blur);
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
.btn-login.as-link { display: inline-flex; align-items: center; justify-content: center; text-decoration: none; }
.switch-line { margin: 20px 0 0; text-align: center; font-size: 14px; color: var(--color-text-dim); }
.switch-link { color: var(--color-primary); font-weight: 700; text-decoration: none; margin-left: 4px; }
.switch-link:hover { text-decoration: underline; }
@media (max-width: 480px) {
  .login-page { padding: 16px; }
  .login-card { padding: 56px 24px 28px; }
}
</style>
