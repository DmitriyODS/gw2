<template>
  <div class="cd-screen">
    <div class="cd-card">
      <div class="cd-icon">
        <span class="material-symbols-outlined">domain_disabled</span>
      </div>
      <h1 class="cd-title">Компания отключена</h1>
      <p class="cd-desc">
        <template v-if="companyName && typeof companyName === 'string'">
          Доступ к платформе для компании
          <strong>«{{ companyName }}»</strong>
          временно приостановлен.
        </template>
        <template v-else>
          Доступ к платформе для вашей компании временно приостановлен.
        </template>
        Обратитесь к администратору или напишите в техническую поддержку.
      </p>
      <div class="cd-actions">
        <button class="cd-btn-text" @click="onLogout">Выйти</button>
        <button class="cd-btn-filled" @click="onContactSupport">
          <span class="material-symbols-outlined">support_agent</span>
          Написать в техподдержку
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth.js'

const auth = useAuthStore()
const companyName = computed(() => auth.companyDisabled)

function onLogout() {
  auth.logout()
}

function onContactSupport() {
  // На этом этапе спец-чаты «Разработчикам» ещё не реализованы (Этап 4).
  // Пока — открываем mailto как fallback, чтобы экран не был «мёртвым».
  window.location.href = 'mailto:support@grovework.local?subject=Компания%20отключена'
}
</script>

<style scoped>
.cd-screen {
  min-height: 100vh;
  width: 100%;
  display: grid;
  place-items: center;
  background: var(--color-surface, var(--gw-bg));
  padding: 24px;
}

.cd-card {
  max-width: 480px;
  width: 100%;
  background: var(--color-surface-high, var(--gw-surface));
  border-radius: var(--radius-xl, 28px);
  padding: 40px 32px;
  text-align: center;
  box-shadow: var(--shadow-lg);
}

.cd-icon {
  width: 96px;
  height: 96px;
  margin: 0 auto 20px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}

.cd-icon .material-symbols-outlined { font-size: 48px; }

.cd-title {
  font-size: 24px;
  font-weight: 700;
  margin: 0 0 12px;
  color: var(--color-text);
}

.cd-desc {
  font-size: 15px;
  line-height: 1.5;
  color: var(--gw-text-secondary, var(--color-text));
  margin: 0 0 28px;
}

.cd-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
  flex-wrap: wrap;
}

.cd-btn-text, .cd-btn-filled {
  height: 44px;
  padding: 0 22px;
  border-radius: var(--radius-full, 999px);
  border: none;
  font: inherit;
  font-weight: 600;
  font-size: 14px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  transition: background 0.15s, transform 0.08s;
}

.cd-btn-text {
  background: transparent;
  color: var(--color-primary);
}
.cd-btn-text:hover { background: var(--color-primary-container); }

.cd-btn-filled {
  background: var(--color-primary);
  color: var(--color-on-primary);
}
.cd-btn-filled:hover { filter: brightness(0.95); }
.cd-btn-filled:active { transform: scale(0.98); }
</style>
