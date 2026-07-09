<template>
  <div class="login-page">
    <div class="login-wrapper">
      <div class="login-logo-wrap">
        <Logo class="login-logo-img" :size="80" />
      </div>

      <div class="login-card">
        <div v-if="loading" class="state"><ProgressSpinner /></div>

        <template v-else-if="invite">
          <div class="invite-icon"><span class="material-symbols-outlined">groups</span></div>
          <h1 class="register-title">Приглашение в команду</h1>
          <p class="register-sub">
            Компания <b>{{ invite.company_name }}</b> приглашает вас присоединиться на роль <b>{{ invite.role_name }}</b>.
          </p>
          <p v-if="error" class="error-msg">{{ error }}</p>
          <button class="btn-login" :disabled="accepting" @click="accept">
            {{ accepting ? 'Входим в команду…' : 'Принять приглашение' }}
          </button>
          <RouterLink to="/" class="switch-link center">Позже</RouterLink>
        </template>

        <template v-else>
          <div class="invite-icon error"><span class="material-symbols-outlined">link_off</span></div>
          <h1 class="register-title">Приглашение недоступно</h1>
          <p class="register-sub">{{ error || 'Ссылка недействительна или срок её действия истёк.' }}</p>
          <RouterLink to="/" class="btn-login as-link">На главную</RouterLink>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import ProgressSpinner from 'primevue/progressspinner'
import { useAuthStore } from '@/stores/auth.js'
import { useThemeStore } from '@/stores/theme.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { getInvitePreview } from '@/api/companies.js'
import Logo from '@/components/common/Logo.vue'

const props = defineProps({ token: { type: String, required: true } })

const router = useRouter()
const authStore = useAuthStore()
const themeStore = useThemeStore()
const notif = useNotificationsStore()

const invite = ref(null)
const loading = ref(true)
const accepting = ref(false)
const error = ref('')

onMounted(async () => {
  themeStore.init()
  try {
    invite.value = await getInvitePreview(props.token)
  } catch (e) {
    error.value = e?.message || ''
    invite.value = null
  } finally {
    loading.value = false
  }
})

async function accept() {
  accepting.value = true
  error.value = ''
  try {
    await authStore.acceptInvite(props.token)
    notif.success(`Вы в команде «${invite.value.company_name}»`)
    router.push('/tasks')
  } catch (e) {
    error.value = e?.message || 'Не удалось принять приглашение'
  } finally {
    accepting.value = false
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
  align-items: center;
  box-shadow: var(--shadow-lg);
}
.state { padding: 32px; }
.invite-icon {
  width: 64px; height: 64px; border-radius: var(--radius-xl);
  background: var(--color-primary-container); color: var(--color-on-primary-container);
  display: grid; place-items: center; margin-bottom: 12px;
}
.invite-icon.error { background: var(--color-error-container); color: var(--color-on-error-container); }
.invite-icon .material-symbols-outlined { font-size: 32px; }
.register-title { margin: 0 0 6px; font-size: 22px; font-weight: 800; color: var(--color-text); text-align: center; }
.register-sub { margin: 0 0 24px; font-size: 14px; line-height: 1.5; color: var(--color-text-dim); text-align: center; }
.error-msg {
  margin: 0 0 12px;
  font-size: 13px;
  color: var(--color-on-error-container);
  padding: 8px 16px;
  background: var(--color-error-container);
  border-radius: 999px;
  text-align: center;
  width: 100%;
  box-sizing: border-box;
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
  letter-spacing: 0.02em;
}
.btn-login:hover:not(:disabled) { background: var(--color-primary-hover); box-shadow: var(--shadow-lg); transform: translateY(-1px); }
.btn-login:disabled { opacity: 0.6; cursor: not-allowed; transform: none; }
.btn-login.as-link { display: inline-flex; align-items: center; justify-content: center; text-decoration: none; }
.switch-link { color: var(--color-primary); font-weight: 700; text-decoration: none; }
.switch-link.center { margin-top: 16px; }
.switch-link:hover { text-decoration: underline; }
@media (max-width: 480px) {
  .login-page { padding: 16px; }
  .login-card { padding: 56px 24px 28px; }
}
</style>
