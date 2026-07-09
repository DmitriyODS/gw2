<template>
  <div class="empty-state" :class="[`empty-state--${size}`, { 'empty-state--error': tone === 'error', 'empty-state--soft': tone === 'soft' }]">
    <div class="es-icon">
      <span class="material-symbols-outlined">{{ icon }}</span>
    </div>
    <h3 v-if="title" class="es-title">{{ title }}</h3>
    <p v-if="subtitle" class="es-sub">{{ subtitle }}</p>
    <slot />
  </div>
</template>

<script setup>
defineProps({
  icon: { type: String, required: true },
  title: { type: String, default: '' },
  subtitle: { type: String, default: '' },
  /* md — обычный (страницы/панели), sm — компактный (сайдбары, узкие списки) */
  size: { type: String, default: 'md' },
  /* primary — обычное пустое состояние, error — ошибка,
     soft — «ничего не выбрано» правой панели мастер-детейл разделов */
  tone: { type: String, default: 'primary' },
})
</script>

<style scoped>
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 48px 20px;
  text-align: center;
  color: var(--color-text-dim);
}

.es-icon {
  width: 84px;
  height: 84px;
  border-radius: var(--radius-full);
  display: grid;
  place-items: center;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  margin-bottom: 4px;
}

.es-icon .material-symbols-outlined { font-size: 40px; }

.empty-state--error .es-icon {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}

/* Мягкий круг: полупрозрачная surface-подложка вместо тонального контейнера. */
.empty-state--soft .es-icon {
  width: 96px;
  height: 96px;
  background: color-mix(in oklch, var(--color-surface) 70%, transparent);
  color: var(--color-primary);
  box-shadow: var(--shadow-sm);
}

.empty-state--soft .es-icon .material-symbols-outlined { font-size: 44px; }

.empty-state--soft .es-title {
  font-size: 16px;
  font-weight: 700;
}

.es-title {
  margin: 0;
  font-size: 17px;
  font-weight: 650;
  color: var(--color-text);
}

.es-sub {
  margin: 0;
  font-size: 14px;
  line-height: 1.5;
  max-width: 340px;
}

/* ── Компактный вариант ── */
.empty-state--sm { gap: 6px; padding: 28px 16px; }
.empty-state--sm .es-icon { width: 56px; height: 56px; }
.empty-state--sm .es-icon .material-symbols-outlined { font-size: 28px; }
.empty-state--sm .es-title { font-size: 15px; }
.empty-state--sm .es-sub { font-size: 13px; }
</style>
