<template>
  <!-- Фон экранов входа, лендинга, блокировки и плашки согласия: однотонная
       подложка темы с зерном soft touch — тот же матовый материал, что у панели
       задач и окон. Цвет — токен, поэтому фон следует теме и тёмному режиму.
       Движения и стекла здесь нет намеренно: под формой входа фон должен быть
       спокойным, а полноэкранная анимация впустую грузила видеопроцесс. -->
  <div class="ab" aria-hidden="true" />
</template>

<style scoped>
.ab {
  position: fixed;
  inset: 0;
  z-index: 0;
  pointer-events: none;
  background: var(--color-bg);
  background: var(--grain), var(--color-bg);
}
</style>
