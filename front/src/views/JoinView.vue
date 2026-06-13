<template>
  <div class="join-page">
    <div class="join-card">
      <Logo :size="64" class="join-logo" />
      <template v-if="loading">
        <div class="join-spinner" />
        <p class="join-text">Подключаем вас к компании…</p>
      </template>
      <template v-else>
        <span class="material-symbols-outlined join-icon">{{ ok ? 'check_circle' : 'error' }}</span>
        <p class="join-text">{{ message }}</p>
        <router-link v-if="!ok" to="/tasks" class="join-btn">На главную</router-link>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import Logo from '@/components/common/Logo.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const loading = ref(true)
const ok = ref(false)
const message = ref('')

onMounted(async () => {
  try {
    await auth.joinCompany(route.params.code)
    ok.value = true
    // Токен/активная компания уже переключены — уходим в приложение.
    router.replace('/tasks')
  } catch (e) {
    message.value = e?.message || 'Ссылка-приглашение недействительна или истекла'
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.join-page {
  position: fixed;
  inset: 0;
  display: grid;
  place-items: center;
  background: var(--color-surface-low, var(--color-surface));
  padding: 24px;
}

.join-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 40px 32px;
  border-radius: var(--radius-xl, 20px);
  background: var(--color-surface);
  box-shadow: var(--shadow-lg);
  max-width: 360px;
  text-align: center;
}

.join-logo { opacity: 0.9; }

.join-text { margin: 0; font-size: 15px; color: var(--color-text); }

.join-icon { font-size: 48px; color: var(--color-error); }

.join-spinner {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: 3px solid var(--color-outline-dim);
  border-top-color: var(--color-primary);
  animation: join-spin 0.8s linear infinite;
}

@keyframes join-spin {
  to { transform: rotate(360deg); }
}

.join-btn {
  padding: 10px 20px;
  border-radius: 999px;
  background: var(--color-primary);
  color: var(--color-on-primary);
  text-decoration: none;
  font-weight: 600;
}
</style>
