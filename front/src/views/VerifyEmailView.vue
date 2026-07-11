<template>
  <AuthShell
    :title="verifying ? 'Подтверждаем почту…' : 'Подтвердите почту'"
    :subtitle="verifying ? 'Секунду, проверяем ссылку.' : 'Введите код из письма или перейдите по ссылке из него.'"
  >
    <template v-if="!verifying">
      <p class="verify-email-line">
        Мы отправили код на <b>{{ email || 'указанный email' }}</b>
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

    <p v-else-if="error" class="error-msg">{{ error }}</p>
  </AuthShell>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { useThemeStore } from '@/stores/theme.js'
import { connectSocket } from '@/socket/index.js'
import AuthShell from '@/components/auth/AuthShell.vue'

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
/* Каркас страницы (фон, сплит-карточка, промо) — в AuthShell.vue;
   стили формы — те же, что на экранах входа/регистрации. */
.verify-email-line {
  margin: -8px 0 20px;
  font-size: 14px;
  line-height: 1.5;
  color: var(--color-text-dim);
  text-align: center;
  word-break: break-word;
}

.verify-email-line b { color: var(--color-text); }

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

.code-input {
  text-align: center;
  letter-spacing: 0.5em;
  font-size: 22px;
  font-weight: 700;
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
  padding: 0;
}

.switch-link.as-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
  text-decoration: none;
}
</style>
