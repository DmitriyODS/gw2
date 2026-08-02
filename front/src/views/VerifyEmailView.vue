<template>
  <AuthShell
    :title="verifying ? 'подтверждаем почту' : 'подтверждение почты'"
    :subtitle="verifying ? 'Секунду, проверяем ссылку.' : `Мы отправили код на ${email || 'указанный адрес'}.`"
    size="sm"
    back="/login"
  >
    <template v-if="!verifying">
      <form class="auth-form" @submit.prevent="submitCode">
        <AuthField
          v-model="code"
          label="код из письма"
          placeholder="——————"
          inputmode="numeric"
          autocomplete="one-time-code"
          :maxlength="6"
          :disabled="loading"
          center
        />
        <p v-if="error" class="auth-error">{{ error }}</p>
        <button type="submit" class="auth-submit" :disabled="loading || code.length < 6">
          {{ loading ? 'проверяем…' : 'подтвердить' }}
        </button>
      </form>

      <p class="ve-resend">
        Не пришло письмо?
        <button type="button" :disabled="cooldown > 0 || !email" @click="resend">
          {{ cooldown > 0 ? `отправить ещё раз (${cooldown})` : 'отправить ещё раз' }}
        </button>
      </p>
    </template>

    <p v-else-if="error" class="auth-error">{{ error }}</p>
  </AuthShell>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { connectSocket } from '@/socket/index.js'
import { flushPendingAvatar } from '@/utils/pendingAvatar.js'
import AuthShell from '@/components/auth/AuthShell.vue'
import AuthField from '@/components/auth/AuthField.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

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
    // Фото, выбранное на регистрации, ждало появления сессии — отправляем.
    await flushPendingAvatar()
    connectSocket()
    router.push('/home')
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
.ve-resend {
  margin: 16px 0 0;
  text-align: center;
  font-size: 13.5px;
  color: var(--color-text-dim);
}

.ve-resend button {
  margin-left: 4px;
  padding: 0;
  border: none;
  background: none;
  color: var(--color-primary);
  font: inherit;
  font-size: 13.5px;
  font-weight: 600;
  cursor: pointer;
}

.ve-resend button:disabled { opacity: 0.55; cursor: not-allowed; }

</style>
