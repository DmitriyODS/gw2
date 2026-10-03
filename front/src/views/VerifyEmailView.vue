<template>
  <AuthShell
    :title="verifying ? 'Подтверждаем почту' : 'Подтверждение почты'"
    :subtitle="verifying ? 'Секунду, проверяем ссылку.' : `Мы отправили код на ${email || 'указанный адрес'}.`"
    size="sm"
    back="/login"
  >
    <template v-if="!verifying">
      <form class="auth-form" @submit.prevent="submitCode">
        <AuthField
          v-model="code"
          label="Код из письма"
          placeholder="——————"
          inputmode="numeric"
          autocomplete="one-time-code"
          :maxlength="6"
          :disabled="loading"
          center
        />
        <AppInfoBar v-if="error" tone="error" inline :message="error" />
        <AppButton
          type="submit"
          variant="filled"
          size="lg"
          block
          :loading="loading"
          :disabled="code.length < 6"
          label="Подтвердить"
        />
      </form>

      <p class="auth-switch ve-resend">
        Не пришло письмо?
        <AppButton
          variant="text"
          size="sm"
          :disabled="cooldown > 0 || !email"
          :label="cooldown > 0 ? `Отправить ещё раз (${cooldown})` : 'Отправить ещё раз'"
          @click="resend"
        />
      </p>
    </template>

    <BrandLoader v-else-if="!error" :size="64" class="ve-loader" />
    <AppInfoBar v-else tone="error" inline :message="error" />
  </AuthShell>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { connectSocket } from '@/socket/index.js'
import { flushPendingAvatar } from '@/utils/pendingAvatar.js'
import AuthShell from '@/components/auth/AuthShell.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
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
.ve-resend { margin-top: 16px; }
.ve-loader { margin: 8px auto; }
</style>
