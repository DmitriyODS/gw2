<template>
  <AuthShell
    title="Добро пожаловать"
    subtitle="Groove Work — ваш timeline, обёрнутый в удобный интерфейс."
    size="md"
  >
    <AppGrid :min="200" :gap="12">
      <AppTile icon="login" label="Войти" hint="У меня уже есть аккаунт" @click="go('/login')" />
      <AppTile icon="person_add" label="Создать аккаунт" hint="Бесплатно, за минуту" @click="go('/register')" />
    </AppGrid>
  </AuthShell>
</template>

<script setup>
import { useRouter, useRoute } from 'vue-router'
import AuthShell from '@/components/auth/AuthShell.vue'
import AppGrid from '@/components/ui/AppGrid.vue'
import AppTile from '@/components/ui/AppTile.vue'

const router = useRouter()
const route = useRoute()

// Цель, ради которой гостя завернули на вход, передаём дальше по цепочке.
// Оформление экранов входа (флагманская тема, режим от системы) включает
// роутер по meta.authScreen — здесь про тему знать не нужно.
function go(path) {
  const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : ''
  router.push(redirect ? { path, query: { redirect } } : path)
}
</script>

<style scoped>
/* Плитки выбора крупнее плиток разделов: на экране их всего две. */
:deep(.tile) { min-height: 150px; font-size: 1rem; }
:deep(.tile-icon) { font-size: 32px; color: var(--color-primary); }
</style>
