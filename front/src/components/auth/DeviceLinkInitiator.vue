<template>
  <div class="dl-initiator">
    <template v-if="status === 'expired'">
      <div class="dl-expired">
        <span class="material-symbols-outlined">timer_off</span>
        <p>Код устарел</p>
        <button type="button" class="btn-glass" @click="start">Обновить код</button>
      </div>
    </template>

    <template v-else>
      <div class="dl-qr-wrap" :class="{ loading: status === 'starting' }">
        <QrImage v-if="qrUrl" :value="qrUrl" :size="220" />
        <div v-else class="dl-qr-skeleton"></div>
      </div>

      <div class="dl-code" v-if="code">
        <span class="dl-code-label">Или код для ручного ввода</span>
        <span class="dl-code-value">{{ prettyCode }}</span>
      </div>

      <p class="dl-hint">
        <span class="dl-spinner" aria-hidden="true"></span>
        {{ hint }}
      </p>
      <p v-if="error" class="dl-error">{{ error }}</p>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { linkStart, linkClaim } from '@/api/devicelink.js'
import QrImage from '@/components/common/QrImage.vue'

const props = defineProps({
  // 'login' — обычный вход по QR; 'tv' — авторизация ТВ-киоска.
  kind: { type: String, default: 'login' },
})
const emit = defineEmits(['session'])

const code = ref('')
const secret = ref('')
const qrUrl = ref('')
const status = ref('starting') // starting | waiting | expired
const error = ref('')

let pollTimer = null

const prettyCode = computed(() =>
  code.value ? code.value.replace(/(.{3})(.{3})/, '$1-$2') : '',
)

const hint = computed(() =>
  props.kind === 'tv'
    ? 'Отсканируйте код телефоном или введите его в настройках → «Авторизовать ТВ-киоск».'
    : 'Отсканируйте QR камерой телефона, где вы уже вошли, — и подтвердите вход.',
)

function stop() {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

async function start() {
  stop()
  status.value = 'starting'
  error.value = ''
  code.value = ''
  qrUrl.value = ''
  try {
    const res = await linkStart(props.kind)
    code.value = res.code
    secret.value = res.secret
    qrUrl.value = `${window.location.origin}/link?code=${encodeURIComponent(res.code)}`
    status.value = 'waiting'
    schedulePoll()
  } catch (e) {
    error.value = e?.message || 'Не удалось создать код. Попробуйте ещё раз.'
    status.value = 'expired'
  }
}

function schedulePoll() {
  pollTimer = setTimeout(poll, 2500)
}

async function poll() {
  if (!code.value || !secret.value) return
  try {
    const res = await linkClaim(code.value, secret.value)
    if (res.status === 'ok' && res.session) {
      stop()
      emit('session', res.session)
      return
    }
    if (res.status === 'expired') {
      stop()
      status.value = 'expired'
      return
    }
  } catch {
    /* сеть моргнула — просто попробуем на следующем тике */
  }
  schedulePoll()
}

onMounted(start)
onBeforeUnmount(stop)
</script>

<style scoped>
.dl-initiator {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  text-align: center;
}
.dl-qr-wrap {
  padding: 10px;
  border-radius: var(--radius-lg, 16px);
  background: #fff;
  box-shadow: var(--shadow-sm);
}
.dl-qr-wrap.loading { opacity: 0.5; }
.dl-qr-skeleton {
  width: 220px;
  height: 220px;
  border-radius: var(--radius-md, 12px);
  background: var(--color-surface-2, rgba(0, 0, 0, 0.05));
}
.dl-code {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.dl-code-label {
  font-size: 0.78rem;
  color: var(--color-text-secondary);
}
.dl-code-value {
  font-size: 1.5rem;
  font-weight: 700;
  letter-spacing: 0.18em;
  font-family: var(--font-mono, monospace);
  color: var(--color-text);
}
.dl-hint {
  max-width: 320px;
  font-size: 0.85rem;
  color: var(--color-text-secondary);
  display: flex;
  align-items: center;
  gap: 8px;
  justify-content: center;
}
.dl-error { color: var(--color-error); font-size: 0.85rem; }
.dl-expired {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 24px 0;
  color: var(--color-text-secondary);
}
.dl-expired .material-symbols-outlined { font-size: 40px; }
.dl-spinner {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  border: 2px solid var(--color-primary);
  border-top-color: transparent;
  animation: dl-spin 0.8s linear infinite;
  flex-shrink: 0;
}
@keyframes dl-spin { to { transform: rotate(360deg); } }
</style>
