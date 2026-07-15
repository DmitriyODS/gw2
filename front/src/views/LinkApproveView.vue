<template>
  <div class="link-approve">
    <div class="la-card">
      <template v-if="state === 'loading'">
        <span class="dl-spinner"></span>
        <p class="la-muted">Проверяем код…</p>
      </template>

      <template v-else-if="state === 'confirm'">
        <span class="material-symbols-outlined la-icon">{{ isTv ? 'tv' : 'login' }}</span>
        <h2 class="la-title">{{ isTv ? 'Активировать ТВ-киоск?' : 'Подтвердить вход?' }}</h2>
        <p class="la-text">
          <template v-if="isTv">
            <template v-if="authStore.companyId != null">
              ТВ-киоск войдёт в систему под компанией «{{ authStore.companyName }}».
            </template>
            <template v-else>
              Чтобы авторизовать ТВ-киоск, сначала выберите компанию в своём аккаунте.
            </template>
          </template>
          <template v-else>
            Другое устройство войдёт под вашим аккаунтом. Подтверждайте, только если
            это ваш вход.
          </template>
        </p>
        <p v-if="error" class="la-error">{{ error }}</p>
        <div class="la-actions">
          <button type="button" class="btn-glass" @click="goHome">Отмена</button>
          <button
            type="button"
            class="btn-grad"
            :disabled="loading || (isTv && authStore.companyId == null)"
            @click="approve"
          >
            {{ loading ? 'Подтверждаем…' : 'Подтвердить' }}
          </button>
        </div>
      </template>

      <template v-else-if="state === 'done'">
        <span class="material-symbols-outlined la-icon success">check_circle</span>
        <h2 class="la-title">Готово!</h2>
        <p class="la-text">Устройство входит в систему. Можно закрыть эту страницу.</p>
        <button type="button" class="btn-grad" @click="goHome">На главную</button>
      </template>

      <template v-else>
        <span class="material-symbols-outlined la-icon">error</span>
        <h2 class="la-title">Не получилось</h2>
        <p class="la-text">{{ error }}</p>
        <button type="button" class="btn-grad" @click="goHome">На главную</button>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.js'
import { linkInfo, linkApprove } from '@/api/devicelink.js'
import { normalizeLinkCode } from '@/utils/deviceLink.js'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const state = ref('loading') // loading | confirm | done | error
const info = ref(null)
const loading = ref(false)
const error = ref('')

const code = computed(() => normalizeLinkCode(route.query.code))
const isTv = computed(() => info.value?.kind === 'tv')

onMounted(async () => {
  if (!/^[A-Z2-9]{6}$/.test(code.value)) {
    state.value = 'error'
    error.value = 'Неверная ссылка входа.'
    return
  }
  try {
    info.value = await linkInfo(code.value)
    state.value = 'confirm'
  } catch (e) {
    state.value = 'error'
    error.value = errText(e)
  }
})

async function approve() {
  if (loading.value) return
  loading.value = true
  error.value = ''
  try {
    await linkApprove(code.value)
    state.value = 'done'
  } catch (e) {
    error.value = errText(e)
  } finally {
    loading.value = false
  }
}

function errText(e) {
  switch (e?.error) {
    case 'LINK_EXPIRED':
      return 'Код устарел. Обновите его на устройстве и попробуйте снова.'
    case 'LINK_ALREADY_USED':
      return 'Этот код уже подтверждён другим аккаунтом.'
    case 'LINK_NEED_COMPANY':
      return 'Сначала выберите компанию, под которой авторизовать ТВ-киоск.'
    default:
      return e?.message || 'Не удалось подтвердить устройство.'
  }
}

function goHome() {
  router.push('/')
}
</script>

<style scoped>
.link-approve {
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}
.la-card {
  width: 100%;
  max-width: 420px;
  padding: 32px 28px;
  border-radius: var(--radius-xl, 24px);
  background: var(--acrylic-card-bg);
  border: 1px solid var(--acrylic-border);
  box-shadow: var(--shadow-lg);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  text-align: center;
}
.la-icon { font-size: 52px; color: var(--color-primary); }
.la-icon.success { color: var(--color-success, var(--color-primary)); }
.la-title { font-size: 1.35rem; font-weight: 700; color: var(--color-text); }
.la-text { color: var(--color-text-secondary); font-size: 0.92rem; max-width: 340px; }
.la-muted { color: var(--color-text-secondary); }
.la-error { color: var(--color-error); font-size: 0.85rem; }
.la-actions { display: flex; gap: 12px; margin-top: 4px; }
.dl-spinner {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 3px solid var(--color-primary);
  border-top-color: transparent;
  animation: la-spin 0.8s linear infinite;
}
@keyframes la-spin { to { transform: rotate(360deg); } }
</style>
