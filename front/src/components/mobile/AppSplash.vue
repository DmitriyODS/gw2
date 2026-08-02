<template>
  <!-- Экран запуска раздела: значок и название по центру поверх обоев. Пока он
       виден, раздел успевает смонтироваться — открытие выглядит как запуск
       приложения, а не как мигание пустотой. -->
  <div class="msplash" role="status" :aria-label="`Открывается: ${title}`">
    <span class="msplash-icon material-symbols-outlined">{{ icon }}</span>
    <span class="msplash-title">{{ title }}</span>
  </div>
</template>

<script setup>
defineProps({
  title: { type: String, required: true },
  icon: { type: String, default: 'web_asset' },
})
</script>

<style scoped>
.msplash {
  position: absolute;
  inset: 0;
  z-index: 5;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 22px;
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
}

.msplash-icon {
  font-size: 64px;
  color: var(--color-text);
  animation: msplash-in 0.32s cubic-bezier(0.2, 0, 0, 1);
}

.msplash-title {
  font-size: 1.35rem;
  font-weight: 500;
  color: var(--color-text);
  animation: msplash-in 0.32s cubic-bezier(0.2, 0, 0, 1) 0.04s backwards;
}

@keyframes msplash-in {
  from { opacity: 0; scale: 0.86; }
  to { opacity: 1; scale: 1; }
}

@media (prefers-reduced-motion: reduce) {
  .msplash-icon,
  .msplash-title { animation: none; }
}
</style>
