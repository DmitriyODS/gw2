<template>
  <div class="dl">
    <template v-if="status === 'expired'">
      <div class="dl-expired">
        <span class="material-symbols-outlined">timer_off</span>
        <p>{{ error || 'Код устарел' }}</p>
        <button type="button" class="dl-refresh" @click="start">обновить код</button>
      </div>
    </template>

    <template v-else>
      <div class="dl-qr" :class="{ loading: status === 'starting' }">
        <QrImage v-if="qrUrl" :value="qrUrl" :size="196" />
        <div v-else class="dl-qr-skeleton" />
        <span class="dl-code">{{ prettyCode || '· · · · · ·' }}</span>
      </div>

      <p class="dl-status">
        <span class="dl-spinner" aria-hidden="true" />
        ждём подтверждения…
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
.dl {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  text-align: center;
}

/* Белая подложка обязательна: QR читается сканерами только по контрасту
   тёмного кода на светлом, поэтому она НЕ следует тёмной теме. */
.dl-qr {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 16px 16px 12px;
  border-radius: var(--radius-lg);
  background: white;
  box-shadow: var(--shadow-md);
}

.dl-qr.loading { opacity: 0.55; }

.dl-qr-skeleton {
  width: 196px;
  height: 196px;
  border-radius: var(--radius-md);
  background: color-mix(in oklch, var(--color-primary) 10%, white);
}

.dl-code {
  font-size: 17px;
  font-weight: 700;
  letter-spacing: 0.22em;
  text-indent: 0.22em;
  font-family: var(--font-mono, monospace);
  color: oklch(0.25 0 0);
}

.dl-status {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 13px;
  color: var(--color-text-dim);
}

.dl-error { margin: 0; color: var(--color-error); font-size: 13px; }

.dl-refresh {
  padding: 9px 18px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: var(--glass-bg), color-mix(in oklch, var(--color-surface) 45%, transparent);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.dl-refresh:hover { color: var(--color-primary); }

.dl-expired {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 20px 0;
  color: var(--color-text-dim);
}

.dl-expired p { margin: 0; font-size: 14px; }
.dl-expired .material-symbols-outlined { font-size: 38px; }

.dl-spinner {
  width: 13px;
  height: 13px;
  border-radius: 50%;
  border: 2px solid var(--color-primary);
  border-top-color: transparent;
  animation: dl-spin 0.8s linear infinite;
  flex-shrink: 0;
}

@keyframes dl-spin { to { transform: rotate(360deg); } }
</style>
