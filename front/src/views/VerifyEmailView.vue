<template>
  <div class="login-page">
    <div class="login-wrapper">
      <div class="login-logo-wrap">
        <Logo class="login-logo-img" :size="80" />
      </div>

      <div class="login-card">
        <template v-if="verifying">
          <h1 class="register-title">Подтверждаем почту…</h1>
          <p class="register-sub">Секунду, проверяем ссылку.</p>
        </template>

        <template v-else>
          <h1 class="register-title">Подтвердите почту</h1>
          <p class="register-sub">
            Мы отправили код на<br><b>{{ email || 'указанный email' }}</b>.<br>
            Введите его ниже или перейдите по ссылке из письма.
          </p>

          <form @submit.prevent="submitCode" class="login-form">
            <div class="form-group">
              <label>Код подтверждения</label>
              <input
                v-model.trim="code"
                inputmode="numeric"
                maxlength="6"
                class="pill-input code-input"
                :disabled="loading"
                placeholder="------"
                autocomplete="one-time-code"
              />
            </div>
            <p v-if="error" class="error-msg">{{ error }}</p>
            <button type="submit" class="btn-login" :disabled="loading || code.length < 6">
              {{ loading ? 'Проверяем…' : 'Подтвердить' }}
            </button>
          </form>

          <p class="switch-line">
            Не пришло письмо?
            <button type="button" class="switch-link as-btn" :disabled="cooldown > 0 || !email" @click="resend">
              {{ cooldown > 0 ? `Отправить ещё раз (${cooldown})` : 'Отправить ещё раз' }}
            </button>
          </p>
          <p class="switch-line">
            <RouterLink to="/login" class="switch-link">Вернуться ко входу</RouterLink>
          </p>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { useThemeStore } from '@/stores/theme.js'
import { connectSocket } from '@/socket/index.js'
import Logo from '@/components/common/Logo.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const themeStore = useThemeStore()

// email нужен для подтверждения по коду. Берём из query (ссылка письма /
// переход с регистрации), иначе — из localStorage (экран мог пересоздаться без
// query): иначе код-путь уходил с пустым email и падал «email не задан».
const PENDING_EMAIL_KEY = 'gw_verify_email'
const email = ref(route.query.email || localStorage.getItem(PENDING_EMAIL_KEY) || '')
const code = ref('')
const error = ref('')
const loading = ref(false)
const verifying = ref(false)
const cooldown = ref(0)
let cooldownTimer = null

onMounted(() => {
  themeStore.init()
  if (email.value) localStorage.setItem(PENDING_EMAIL_KEY, email.value)
  if (route.query.token) {
    verifyWith({ token: route.query.token })
  }
})

onBeforeUnmount(() => clearInterval(cooldownTimer))

function startCooldown(sec = 60) {
  cooldown.value = sec
  clearInterval(cooldownTimer)
  cooldownTimer = setInterval(() => {
    cooldown.value -= 1
    if (cooldown.value <= 0) clearInterval(cooldownTimer)
  }, 1000)
}

async function verifyWith(payload) {
  error.value = ''
  if (payload.token) verifying.value = true
  loading.value = true
  try {
    await authStore.verifyEmail(payload)
    localStorage.removeItem(PENDING_EMAIL_KEY)
    connectSocket()
    router.push('/')
  } catch (e) {
    verifying.value = false
    error.value = e?.message || 'Не удалось подтвердить почту'
  } finally {
    loading.value = false
  }
}

function submitCode() {
  if (code.value.length < 6) return
  verifyWith({ email: email.value, code: code.value })
}

async function resend() {
  if (!email.value || cooldown.value > 0) return
  error.value = ''
  try {
    await authStore.resendVerification(email.value)
    startCooldown(60)
  } catch (e) {
    error.value = e?.message || 'Не удалось отправить письмо'
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

.login-wrapper {
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 100%;
  max-width: 420px;
}

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

.register-title {
  margin: 0 0 6px;
  font-size: 22px;
  font-weight: 800;
  color: var(--color-text);
  text-align: center;
}

.register-sub {
  margin: 0 0 24px;
  font-size: 14px;
  line-height: 1.5;
  color: var(--color-text-dim);
  text-align: center;
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

.code-input {
  text-align: center;
  letter-spacing: 0.5em;
  font-size: 22px;
  font-weight: 700;
}

.pill-input:focus {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--color-primary) 15%, transparent);
}

.pill-input:disabled {
  opacity: 0.5;
  cursor: not-allowed;
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

.switch-line {
  margin: 16px 0 0;
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

.switch-link.as-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-family: inherit;
  font-size: 14px;
}

.switch-link.as-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  text-decoration: none;
}

@media (max-width: 480px) {
  .login-page { padding: 16px; }
  .login-card { padding: 56px 24px 28px; }
}
</style>
