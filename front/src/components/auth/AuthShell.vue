<template>
  <div class="auth-page">
    <AuthBackdrop />

    <header class="auth-top">
      <RouterLink to="/" class="auth-brand" aria-label="Groove Work — о платформе">
        <BrandLogo :size="28" />
        <BrandWordmark :size="17" />
      </RouterLink>
    </header>

    <main class="auth-main">
      <div v-if="$slots.hero" class="auth-hero">
        <slot name="hero" />
      </div>

      <AppCard class="auth-card" :class="`is-${size}`" :gap="0" tag="section">
        <header v-if="title || subtitle || showBack" class="auth-head">
          <AppButton
            v-if="showBack"
            variant="icon"
            icon="arrow_back"
            :aria-label="backLabel"
            :title="backLabel"
            class="auth-back"
            @click="goBack"
          />
          <div class="auth-head-text">
            <h1 v-if="title" class="auth-title">{{ title }}</h1>
            <p v-if="subtitle" class="auth-sub">{{ subtitle }}</p>
          </div>
        </header>

        <div class="auth-content">
          <slot />
        </div>

        <footer v-if="$slots.actions" class="auth-foot">
          <slot name="actions" />
        </footer>
      </AppCard>
    </main>

    <footer class="auth-bottom">
      <span>© Groove Work</span>
      <RouterLink to="/">О платформе</RouterLink>
      <RouterLink v-if="LEGAL_CONSENT_VISIBLE" to="/legal">Правовые документы</RouterLink>
    </footer>

    <slot name="overlays" />
  </div>
</template>

<script setup>
/* Каркас экранов входа и смежных публичных страниц: шапка с маркой, по центру
   карточка ядра (AppCard) и короткий подвал. Карточка — тот же матовый лист,
   что и в разделах приложения, поэтому вход выглядит частью платформы, а не
   отдельным сайтом. */
import { computed, useAttrs } from 'vue'
import { useRouter } from 'vue-router'
import BrandLogo from '@/components/common/BrandLogo.vue'
import BrandWordmark from '@/components/common/BrandWordmark.vue'
import AuthBackdrop from '@/components/auth/AuthBackdrop.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppButton from '@/components/ui/AppButton.vue'
import { LEGAL_CONSENT_VISIBLE } from '@/utils/release.js'

const props = defineProps({
  title: { type: String, default: '' },
  subtitle: { type: String, default: '' },
  // Ширина карточки: sm — короткие экраны (код, QR), md — форма входа,
  // lg — регистрация и документы.
  size: { type: String, default: 'md' },
  // Путь кнопки «назад»; пустая строка — кнопки нет. Экран с внутренними
  // шагами вместо пути вешает @back и решает сам, куда возвращаться.
  back: { type: String, default: '' },
  backLabel: { type: String, default: 'Назад' },
})

const emit = defineEmits(['back'])

const router = useRouter()
const attrs = useAttrs()

const showBack = computed(() => !!props.back || !!attrs.onBack)

function goBack() {
  if (attrs.onBack) {
    emit('back')
    return
  }
  router.push(props.back)
}
</script>

<style scoped>
.auth-page {
  position: relative;
  min-height: 100vh;
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  padding: 0 24px;
  overflow-x: hidden;
  overflow-y: auto;
}

/* ── Шапка: марка слева, как в шапке лендинга ─────────────────── */
.auth-top {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  width: 100%;
  max-width: 1120px;
  margin: 0 auto;
  padding: 20px 0;
}

.auth-brand {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  text-decoration: none;
}

/* ── Центр: приветствие и карточка ────────────────────────────── */
.auth-main {
  position: relative;
  z-index: 1;
  flex: 1 0 auto;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 28px;
  padding: 12px 0 32px;
}

.auth-hero {
  width: 100%;
  max-width: 900px;
  animation: auth-rise 0.45s cubic-bezier(0.2, 0.8, 0.3, 1) both;
}

/* Специфичность выше scoped-класса AppCard: поля и радиус карточки входа
   крупнее, чем у карточек разделов. */
.auth-page .auth-card {
  width: 100%;
  padding: 28px 32px 32px;
  border-radius: var(--radius-xl);
  box-shadow: var(--sk-panel-shadow), var(--shadow-lg);
  animation: auth-rise 0.4s cubic-bezier(0.2, 0.8, 0.3, 1) both;
}

.auth-card.is-sm { max-width: 440px; }
.auth-card.is-md { max-width: 520px; }
.auth-card.is-lg { max-width: 900px; }

.auth-hero + .auth-card { animation-delay: 0.1s; }

@keyframes auth-rise {
  from { opacity: 0; transform: translateY(14px); }
  to { opacity: 1; transform: none; }
}

@media (prefers-reduced-motion: reduce) {
  .auth-hero,
  .auth-page .auth-card { animation: none; }
}

/* ── Заголовок карточки: «назад» слева от названия ────────────── */
.auth-head {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  margin-bottom: 24px;
}

.auth-back { margin-top: -2px; }

.auth-head-text { flex: 1; min-width: 0; }

.auth-title {
  margin: 0;
  font-size: clamp(22px, 2.6vw, 28px);
  font-weight: 600;
  line-height: 1.2;
  letter-spacing: -0.015em;
  color: var(--color-text);
  overflow-wrap: anywhere;
}

.auth-sub {
  margin: 6px 0 0;
  font-size: 14px;
  line-height: 1.5;
  color: var(--color-text-dim);
}

.auth-content { min-width: 0; }

.auth-foot {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid var(--color-outline-dim);
}

/* ── Подвал страницы ──────────────────────────────────────────── */
.auth-bottom {
  position: relative;
  z-index: 1;
  display: flex;
  justify-content: center;
  flex-wrap: wrap;
  gap: 6px 18px;
  padding: 0 0 20px;
  font-size: 12.5px;
  color: var(--color-text-dim);
}

.auth-bottom a {
  color: inherit;
  text-decoration: none;
}

.auth-bottom a:hover { color: var(--color-primary); }

@media (max-width: 560px) {
  .auth-page { padding: 0 12px; }
  .auth-top { padding: 14px 4px; }
  .auth-main { gap: 20px; padding-top: 4px; }
  .auth-page .auth-card { padding: 20px 18px 22px; border-radius: var(--radius-lg); }
  .auth-head { margin-bottom: 18px; gap: 10px; }
}
</style>
