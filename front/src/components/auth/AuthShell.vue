<template>
  <div class="auth-page">
    <AuthWave />

    <div v-if="$slots.hero" class="auth-hero">
      <slot name="hero" />
    </div>

    <div class="auth-card" :class="`is-${size}`">
      <header class="auth-brand">
        <Logo :size="22" class="auth-brand-logo" />
        <span class="auth-brand-name">
          <span class="wm-groove">Groove</span>
          <span class="wm-work">Work</span>
          <span v-if="majorVersion" class="wm-work">{{ majorVersion }}</span>
        </span>
        <slot name="brand-extra" />
      </header>

      <section class="auth-body">
        <h1 v-if="title" class="auth-title">{{ title }}</h1>
        <p v-if="subtitle" class="auth-sub">{{ subtitle }}</p>
        <div class="auth-content" :class="{ spaced: !!title }">
          <slot />
        </div>
      </section>

      <footer v-if="showBack || $slots.actions" class="auth-foot">
        <button v-if="showBack" type="button" class="auth-back" @click="goBack">
          <span class="material-symbols-outlined">arrow_back</span>
          {{ backLabel }}
        </button>
        <span class="auth-foot-gap" />
        <slot name="actions" />
      </footer>
    </div>

    <slot name="overlays" />
  </div>
</template>

<script setup>
import { computed, onMounted, useAttrs } from 'vue'
import { useRouter } from 'vue-router'
import Logo from '@/components/common/Logo.vue'
import AuthWave from '@/components/auth/AuthWave.vue'
import { useAppVersion } from '@/composables/useAppVersion.js'

const props = defineProps({
  title: { type: String, default: '' },
  subtitle: { type: String, default: '' },
  // Ширина карточки: sm — короткие экраны (код, QR), md — форма входа,
  // lg — регистрация (две колонки полей + выбор темы).
  size: { type: String, default: 'md' },
  // Путь кнопки «назад»; пустая строка — кнопки нет. Экран с внутренними
  // шагами вместо пути вешает @back и решает сам, куда возвращаться.
  back: { type: String, default: '' },
  backLabel: { type: String, default: 'назад' },
})

const emit = defineEmits(['back'])

const router = useRouter()
const attrs = useAttrs()
const { majorVersion, load: loadVersion } = useAppVersion()

// Марка показывает мажорную версию выпуска — сведения тянутся с сервера
// (в бандл версия не зашивается).
onMounted(loadVersion)

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
  align-items: center;
  justify-content: center;
  gap: 28px;
  padding: 24px;
  overflow-x: hidden;
  overflow-y: auto;
}

/* Приветственный заголовок над карточкой (экран выбора «вход/регистрация»). */
.auth-hero {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 940px;
  animation: auth-rise 0.5s cubic-bezier(0.2, 0.8, 0.3, 1) both;
}

/* ── Внешняя стеклянная карточка ──────────────────────────────────
   Лежит на гребне волны: полупрозрачное стекло собирает её цвет. */
.auth-card {
  position: relative;
  z-index: 1;
  width: 100%;
  margin: auto;
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 16px;
  border-radius: 32px;
  border: 1px solid var(--acrylic-border);
  background: var(--glass-bg), var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  box-shadow: var(--shadow-xl), var(--glass-edge);
  animation: auth-rise 0.42s cubic-bezier(0.2, 0.8, 0.3, 1) both;
}

.auth-card.is-sm { max-width: 460px; }
.auth-card.is-md { max-width: 620px; }
.auth-card.is-lg { max-width: 940px; }

/* Карточка выезжает следом за приветствием, а не одновременно с ним. */
.auth-hero + .auth-card { animation-delay: 0.14s; }

@keyframes auth-rise {
  from { opacity: 0; transform: translateY(18px) scale(0.985); }
  to { opacity: 1; transform: none; }
}

@media (prefers-reduced-motion: reduce) {
  .auth-card { animation: none; }
}

/* ── Марка ─────────────────────────────────────────────────────── */
.auth-brand {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 2px 8px 0;
}

.auth-brand-logo { display: block; flex-shrink: 0; }

/* Марка — то же начертание, что и в меню «Пуск»: ExtraBlack вариативного
   Roboto Flex, «Groove» фирменным цветом, «Work N» — цветом текста. */
.auth-brand-name {
  display: flex;
  align-items: baseline;
  gap: 5px;
  font-family: 'Roboto Flex', 'Roboto', sans-serif;
  font-size: 15px;
  font-weight: 1000;
  font-variation-settings: 'wght' 1000;
  letter-spacing: 0.2px;
}

.wm-groove { color: var(--color-primary); }
.wm-work { color: var(--color-text); }

/* ── Внутренняя панель с содержимым ────────────────────────────── */
.auth-body {
  background: var(--acrylic-bg-strong);
  border: 1px solid color-mix(in oklch, var(--acrylic-border) 60%, transparent);
  border-radius: 24px;
  padding: 28px 32px 32px;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.auth-title {
  margin: 0;
  font-size: clamp(24px, 3vw, 32px);
  font-weight: 500;
  line-height: 1.15;
  letter-spacing: -0.02em;
  color: var(--color-text);
}

.auth-sub {
  margin: 8px 0 0;
  font-size: 14px;
  line-height: 1.5;
  color: var(--color-text-dim);
}

.auth-content.spaced { margin-top: 22px; }

/* ── Подвал: «назад» слева, действия справа ────────────────────── */
.auth-foot {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 2px 8px 4px;
}

.auth-foot-gap { flex: 1 1 auto; }

.auth-back {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px 8px 12px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: var(--glass-bg), var(--acrylic-bg-strong);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
}

.auth-back:hover {
  background: var(--glass-bg), color-mix(in oklch, var(--color-primary) 12%, var(--acrylic-bg-strong));
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border));
}

.auth-back .material-symbols-outlined { font-size: 18px; }

@media (max-width: 560px) {
  .auth-page { padding: 12px; }
  .auth-card { padding: 12px; border-radius: 26px; gap: 10px; }
  .auth-body { padding: 22px 20px 24px; border-radius: 20px; }
}
</style>
