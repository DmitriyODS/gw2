<template>
  <!-- Личный dev-чат с командой разработки: бэк создаёт его при первом
       обращении, поэтому кнопка сразу ведёт в мессенджер. -->
  <button class="sup-card" type="button" :disabled="opening" @click="openSupport">
    <span class="sup-icon">
      <span class="material-symbols-outlined">support_agent</span>
    </span>
    <span class="sup-text">
      <strong>Чат с техподдержкой</strong>
      <small>Опишите проблему или предложите улучшение — ответ придёт сюда же, в мессенджер.</small>
    </span>
    <span class="material-symbols-outlined sup-chev">chevron_right</span>
  </button>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useMessengerStore } from '@/stores/messenger.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const router = useRouter()
const messenger = useMessengerStore()
const notif = useNotificationsStore()

const opening = ref(false)

async function openSupport() {
  if (opening.value) return
  opening.value = true
  try {
    const convId = await messenger.openDevChat()
    router.push(`/messenger/${convId}`)
  } catch (e) {
    notif.error(e.message || 'Не удалось открыть чат техподдержки')
  } finally {
    opening.value = false
  }
}
</script>

<style scoped>
.sup-card {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  padding: 16px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
  color: var(--color-text);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.2s ease;
}

.sup-card:hover:not(:disabled) { border-color: var(--color-primary); }
.sup-card:disabled { opacity: 0.6; cursor: progress; }

.sup-icon {
  display: grid;
  place-items: center;
  width: 44px;
  min-width: 44px;
  max-width: 44px;
  height: 44px;
  min-height: 44px;
  max-height: 44px;
  border-radius: var(--radius-md);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.sup-icon .material-symbols-outlined { font-size: 24px; }

.sup-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

.sup-text strong { font-size: 0.95rem; font-weight: 600; }

.sup-text small {
  font-size: 0.82rem;
  color: var(--color-text-dim);
  line-height: 1.35;
}

.sup-chev { color: var(--color-text-dim); }
</style>
