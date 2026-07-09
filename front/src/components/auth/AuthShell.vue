<template>
  <div class="auth-page">
    <!-- Декоративный фон: крупные размытые градиентные пятна из цветов темы -->
    <div class="auth-bg" aria-hidden="true">
      <span class="blob blob-a" />
      <span class="blob blob-b" />
      <span class="blob blob-c" />
    </div>

    <!-- Сплит-карточка: слева промо-панель бренда, справа форма -->
    <div class="auth-card">
      <aside class="auth-promo">
        <div class="promo-brand">
          <Logo class="promo-logo" :size="48" />
          <span class="promo-wordmark">Groove<span class="wm-accent">Work</span></span>
        </div>

        <div class="promo-main">
          <h1 class="promo-title">
            Работайте в едином ритме —
            <b>вся команда в одном месте</b>.
          </h1>
          <div class="promo-divider" />
          <p class="promo-desc">
            Задачи и учёт времени, мессенджер и звонки, календари, заметки
            и корпоративный портал — одна платформа для всей команды.
          </p>
        </div>

        <ul class="promo-feats">
          <li v-for="f in FEATURES" :key="f.icon" class="promo-feat">
            <span class="feat-icon">
              <span class="material-symbols-outlined">{{ f.icon }}</span>
            </span>
            {{ f.text }}
          </li>
        </ul>
      </aside>

      <section class="auth-form-col">
        <header class="auth-form-head">
          <h2 class="auth-form-title">{{ title }}</h2>
          <p v-if="subtitle" class="auth-form-sub">{{ subtitle }}</p>
        </header>
        <slot />
      </section>
    </div>

    <!-- Диалоги экранов (смена учётных данных, выбор компании) -->
    <slot name="overlays" />
  </div>
</template>

<script setup>
import Logo from '@/components/common/Logo.vue'

defineProps({
  title: { type: String, required: true },
  subtitle: { type: String, default: '' },
})

const FEATURES = [
  { icon: 'timer',  text: 'Учёт времени и живая статистика' },
  { icon: 'forum',  text: 'Мессенджер, звонки и корпоративный портал' },
  { icon: 'pets',   text: 'Грувики — питомцы вашей команды' },
]
</script>

<style scoped>
.auth-page {
  position: relative;
  min-height: 100vh;
  min-height: 100dvh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-bg);
  padding: 24px;
  overflow-y: auto;
  overflow-x: hidden;
}

/* ── Фон: размытые градиентные пятна из цветов темы ──
   Лежат под акриловой карточкой — её blur «собирает» из них мягкое
   свечение. Медленный дрейф, без движения при reduced-motion. */
.auth-bg {
  position: fixed;
  inset: 0;
  pointer-events: none;
}

.blob {
  position: absolute;
  border-radius: 50%;
  filter: blur(90px);
  opacity: 0.55;
  animation: blob-drift 24s ease-in-out infinite alternate;
}

.blob-a {
  width: 55vmin;
  height: 55vmin;
  top: -12vmin;
  left: -10vmin;
  background: linear-gradient(135deg,
    color-mix(in oklch, var(--color-primary) 55%, transparent),
    color-mix(in oklch, var(--color-tertiary) 40%, transparent));
}

.blob-b {
  width: 48vmin;
  height: 48vmin;
  right: -14vmin;
  top: 18vmin;
  background: linear-gradient(200deg,
    color-mix(in oklch, var(--color-tertiary) 45%, transparent),
    color-mix(in oklch, var(--color-secondary) 35%, transparent));
  animation-delay: -8s;
}

.blob-c {
  width: 60vmin;
  height: 60vmin;
  bottom: -22vmin;
  left: 22vmin;
  background: linear-gradient(320deg,
    color-mix(in oklch, var(--color-secondary) 40%, transparent),
    color-mix(in oklch, var(--color-primary) 40%, transparent));
  animation-delay: -16s;
}

@keyframes blob-drift {
  from { transform: translate3d(0, 0, 0) scale(1); }
  to { transform: translate3d(6vmin, -4vmin, 0) scale(1.12); }
}

@media (prefers-reduced-motion: reduce) {
  .blob { animation: none; }
}

/* ── Сплит-карточка ───────────────────────────────────────────── */
.auth-card {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 1020px;
  display: grid;
  grid-template-columns: minmax(0, 0.95fr) minmax(0, 1.05fr);
  background: var(--acrylic-bg);
  backdrop-filter: var(--acrylic-blur);
  -webkit-backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  border-radius: 28px;
  overflow: hidden;
  margin: auto;
}

/* ── Промо-панель: тинтованный градиент из цветов темы ────────── */
.auth-promo {
  display: flex;
  flex-direction: column;
  gap: 32px;
  padding: 36px 40px;
  background:
    radial-gradient(60% 50% at 0% 0%, color-mix(in oklch, var(--color-primary) 16%, transparent), transparent 70%),
    radial-gradient(70% 60% at 100% 100%, color-mix(in oklch, var(--color-tertiary) 12%, transparent), transparent 70%),
    linear-gradient(160deg,
      color-mix(in oklch, var(--color-primary-container) 55%, transparent),
      color-mix(in oklch, var(--color-secondary-container) 20%, transparent) 60%,
      transparent);
  border-right: 1px solid var(--acrylic-border);
}

.promo-brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.promo-logo {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  display: block;
}

.promo-wordmark {
  font-size: 21px;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: var(--color-primary);
}

.wm-accent {
  color: color-mix(in oklch, var(--color-primary) 40%, var(--color-primary-container));
}

.promo-main {
  margin: auto 0;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.promo-title {
  margin: 0;
  font-size: clamp(24px, 2.6vw, 30px);
  font-weight: 600;
  line-height: 1.25;
  letter-spacing: -0.02em;
  color: var(--color-text);
}

.promo-title b {
  font-weight: 800;
}

.promo-divider {
  height: 1px;
  background: color-mix(in oklch, var(--color-outline-dim) 70%, transparent);
}

.promo-desc {
  margin: 0;
  font-size: 14.5px;
  line-height: 1.55;
  color: var(--color-text-dim);
  max-width: 360px;
}

.promo-feats {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.promo-feat {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13.5px;
  font-weight: 600;
  color: var(--color-text);
}

.feat-icon {
  width: 30px;
  height: 30px;
  flex-shrink: 0;
  display: grid;
  place-items: center;
  border-radius: 10px;
  background: var(--acrylic-bg-strong);
  border: 1px solid var(--acrylic-border);
  color: var(--color-primary);
}

.feat-icon .material-symbols-outlined { font-size: 18px; }

/* ── Колонка формы ────────────────────────────────────────────── */
.auth-form-col {
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 40px 48px;
  min-width: 0;
}

.auth-form-head {
  margin-bottom: 24px;
  text-align: center;
}

.auth-form-title {
  margin: 0 0 6px;
  font-size: 23px;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--color-text);
}

.auth-form-sub {
  margin: 0;
  font-size: 14px;
  line-height: 1.5;
  color: var(--color-text-dim);
}

/* ── Адаптив ──────────────────────────────────────────────────── */
@media (max-width: 900px) {
  .auth-card {
    grid-template-columns: 1fr;
    max-width: 460px;
  }

  /* Промо сжимается до бренд-шапки с заголовком; фичи — на десктопе */
  .auth-promo {
    gap: 16px;
    padding: 24px 28px 20px;
    border-right: none;
    border-bottom: 1px solid var(--acrylic-border);
  }

  .promo-main { margin: 0; gap: 0; }
  .promo-title { font-size: 18px; }
  .promo-divider,
  .promo-desc,
  .promo-feats { display: none; }

  .auth-form-col { padding: 28px; }
}

@media (max-width: 480px) {
  .auth-page { padding: 12px; }
  .auth-form-col { padding: 24px 20px; }
  .auth-promo { padding: 20px; }
  .promo-title { display: none; }
}
</style>
