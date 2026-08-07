<template>
  <AuthShell size="md">
    <template #hero>
      <h1 class="wc-hero">
        <span class="wc-hero-1">Добро пожаловать</span>
        <span class="wc-hero-2">в рабочую волну</span>
      </h1>
    </template>

    <div class="wc-tiles">
      <button type="button" class="wc-tile" @click="go('/login')">
        <span class="material-symbols-outlined wc-tile-icon">key</span>
        <span class="wc-tile-label">у меня есть аккаунт</span>
      </button>
      <button type="button" class="wc-tile" @click="go('/register')">
        <span class="material-symbols-outlined wc-tile-icon">person_add</span>
        <span class="wc-tile-label">создать новый аккаунт</span>
      </button>
    </div>

    <template #actions>
      <RouterLink to="/" class="wc-about">о платформе</RouterLink>
    </template>
  </AuthShell>
</template>

<script setup>
import { useRouter, useRoute } from 'vue-router'
import AuthShell from '@/components/auth/AuthShell.vue'

const router = useRouter()
const route = useRoute()

// Цель, ради которой гостя завернули на вход, передаём дальше по цепочке.
// Оформление экранов входа (классическая тема, режим от системы) включает
// роутер по meta.authScreen — здесь про тему знать не нужно.
function go(path) {
  const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : ''
  router.push(redirect ? { path, query: { redirect } } : path)
}
</script>

<style scoped>
.wc-hero {
  margin: 0;
  display: flex;
  flex-direction: column;
  text-align: center;
  letter-spacing: -0.03em;
  line-height: 1.05;
}

.wc-hero-1 {
  font-size: clamp(34px, 6.2vw, 68px);
  font-weight: 300;
  color: var(--color-primary);
}

.wc-hero-2 {
  font-size: clamp(24px, 4.4vw, 48px);
  font-weight: 300;
  color: var(--color-text);
}

.wc-tiles {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}

.wc-tile {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  min-height: 168px;
  padding: 18px;
  border: 1px solid var(--acrylic-border);
  border-radius: 20px;
  background: var(--glass-bg), color-mix(in oklch, var(--color-surface) 42%, transparent);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: background 0.16s, border-color 0.16s, box-shadow 0.16s;
}

/* Выделение — блик стекла и подсветка кромки, без смещения плитки. */
.wc-tile:hover {
  border-color: color-mix(in oklch, var(--color-primary) 35%, var(--acrylic-border));
  background: var(--glass-hover-bg), color-mix(in oklch, var(--color-primary) 14%, transparent);
  box-shadow: var(--shadow-md), var(--glass-edge);
}

.wc-tile-icon {
  font-size: 34px;
  color: var(--color-primary);
  font-variation-settings: 'wght' 300;
}

.wc-tile-label {
  font-size: 16px;
  font-weight: 400;
  text-align: center;
}

.wc-about {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-dim);
  text-decoration: none;
  padding: 8px 4px;
}

.wc-about:hover { color: var(--color-primary); }

@media (max-width: 560px) {
  .wc-tiles { grid-template-columns: 1fr; }
  .wc-tile { min-height: 96px; }
}
</style>
